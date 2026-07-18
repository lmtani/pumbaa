package workflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmtani/pumbaa/internal/application/ports"
	domain "github.com/lmtani/pumbaa/internal/domain/workflow"
)

// The forecast makes exactly one claim that carries weight: that a call will be
// served from cache. Every other verdict costs the user a pessimistic estimate;
// a wrong REUSE costs them trust in the whole tool.
//
// The tests in this file are not about coverage. Each one constructs a program
// where a plausible shortcut would conclude REUSE and the engine would in fact
// recompute, and asserts that the forecast declines. They are expected to keep
// passing as resolution grows more capable — that is the point of having them.

// soundnessEnv builds a forecast over a program written across several files,
// so wiring can live inside an imported unit the reference never recorded.
func soundnessEnv(t *testing.T, files map[string]string, params map[string]any, reference *domain.Workflow) forecastEnv {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshalling parameters: %v", err)
	}
	paramsPath := filepath.Join(dir, "params.json")
	if err := os.WriteFile(paramsPath, paramsJSON, 0o600); err != nil {
		t.Fatalf("writing parameters: %v", err)
	}

	fp := &mockFileProvider{
		readBytesFunc: func(_ context.Context, path string) ([]byte, error) { return os.ReadFile(path) },
		getDigestsFunc: func(_ context.Context, path string) (ports.FileDigests, error) {
			// Every file is reported with a stable digest: these tests are about
			// which value reaches an input, never about the bytes behind it.
			return ports.FileDigests{MD5: "00000000000000000000000000000000"}, nil
		},
	}
	return forecastEnv{
		uc:        NewCacheForecastUseCase(&stubMetadataReader{workflow: reference}, nil, nil, fp, nil),
		wdlPath:   filepath.Join(dir, "main.wdl"),
		inputPath: paramsPath,
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	return string(b)
}

// --- Scenario 1: a dependency substituted inside an imported unit ----------
//
// The reference wired Consumer's input to ProducerA. The pending program rewires
// it to ProducerB, and the change lives in an imported file — which the engine
// never records, so no comparison against the reference can see it.
//
// ProducerA is unchanged and would be reused; ProducerB reruns. Reading the
// wiring from the *reference* would find Consumer's recorded value among
// ProducerA's recorded outputs, attach a stale dependency, and conclude REUSE
// while the engine feeds Consumer from a producer that reran.

const substitutionMain = `version 1.0

import "unit.wdl" as unit

workflow Top {
  input {
    File seed
  }
  call unit.Unit {
    input: seed = seed
  }
}
`

// The pending unit reads ProducerB; the reference ran a version that read
// ProducerA. Only the pending text exists on disk, exactly as in reality.
const substitutionUnit = `version 1.0

workflow Unit {
  input {
    File seed
  }
  call ProducerA { input: seed = seed }
  call ProducerB { input: seed = seed }
  call Consumer  { input: incoming = ProducerB.out }
}

task ProducerA {
  input { File seed }
  command <<< echo a >>>
  output { File out = "a.txt" }
}

task ProducerB {
  input { File seed }
  command <<< echo b >>>
  output { File out = "b.txt" }
}

task Consumer {
  input { File incoming }
  command <<< echo ~{incoming} >>>
  output { File out = "c.txt" }
}
`

func substitutionReference(t *testing.T) *domain.Workflow {
	t.Helper()
	// Consumer's recorded input is ProducerA's recorded output: the wiring the
	// reference ran, and the trap for any analysis that trusts it.
	const producerAOut = "/exec/Unit/call-ProducerA/a.txt"

	fingerprint := func(entries map[string]string) domain.CallFingerprint {
		fp := domain.CallFingerprint{"command template": "IRRELEVANT"}
		for k, v := range entries {
			fp[k] = v
		}
		return fp
	}
	leaf := func(name string, inputs map[string]any, outputs map[string]any, fp domain.CallFingerprint) domain.Call {
		return domain.Call{
			Name: name, Backend: "Local", Attempt: 1,
			Inputs: inputs, Outputs: outputs, Fingerprint: fp,
		}
	}

	return &domain.Workflow{
		ID:                "ref-substitution",
		Name:              "Top",
		SubmittedWorkflow: substitutionMain,
		SubmittedInputs:   mustJSON(t, map[string]any{"Top.seed": "/data/seed.txt"}),
		Calls: map[string][]domain.Call{
			"Top.Unit": {{
				Name: "Top.Unit", Backend: "Local", Attempt: 1,
				SubWorkflowMetadata: &domain.Workflow{
					Name: "Unit",
					Calls: map[string][]domain.Call{
						"Unit.ProducerA": {leaf("Unit.ProducerA",
							map[string]any{"seed": "/data/seed.txt"},
							map[string]any{"out": producerAOut},
							fingerprint(map[string]string{"input: File seed": "00000000000000000000000000000000"}))},
						"Unit.ProducerB": {leaf("Unit.ProducerB",
							map[string]any{"seed": "/data/seed.txt"},
							map[string]any{"out": "/exec/Unit/call-ProducerB/b.txt"},
							fingerprint(map[string]string{"input: File seed": "00000000000000000000000000000000"}))},
						"Unit.Consumer": {leaf("Unit.Consumer",
							map[string]any{"incoming": producerAOut},
							map[string]any{"out": "/exec/Unit/call-Consumer/c.txt"},
							fingerprint(map[string]string{"input: File incoming": "REFERENCE_VALUE"}))},
					},
				},
			}},
		},
	}
}

func TestSoundnessRewiredDependencyInsideImportedUnitIsNotReuse(t *testing.T) {
	env := soundnessEnv(t,
		map[string]string{"main.wdl": substitutionMain, "unit.wdl": substitutionUnit},
		map[string]any{"Top.seed": "/data/seed.txt"},
		substitutionReference(t))

	got := run(t, env)

	consumer := predictionFor(t, got, "Unit.Consumer")
	if consumer.Fate == domain.FateReuse {
		t.Fatalf("Unit.Consumer was declared reusable, but its input now comes from "+
			"a different producer than the reference recorded: %+v", consumer)
	}
}

// --- Scenario 2: a branch that flips while every source stays unchanged ----
//
// The input reads one of two sources through a conditional. Both sources are
// individually unchanged — the producer is reusable and the fallback parameter
// is untouched — but the value steering the conditional changed, so the engine
// selects the other branch and the input receives a value the reference never
// recorded.
//
// "Every source unchanged" therefore does not imply "the value is unchanged".
// The predicate's own leaves have to count as leaves of the expression.

const branchFlipMain = `version 1.0

workflow Top {
  input {
    File? optional_source
    File fallback
    File seed
  }

  call Producer { input: seed = seed }

  File chosen = if defined(optional_source) then Producer.out else fallback

  call Consumer { input: incoming = chosen }
}

task Producer {
  input { File seed }
  command <<< echo p >>>
  output { File out = "p.txt" }
}

task Consumer {
  input { File incoming }
  command <<< echo ~{incoming} >>>
  output { File out = "c.txt" }
}
`

func branchFlipReference(t *testing.T) *domain.Workflow {
	t.Helper()
	// The reference ran with optional_source defined, so the conditional chose
	// the producer and recorded its output as Consumer's input.
	const producerOut = "/exec/Top/call-Producer/p.txt"

	fp := func(entries map[string]string) domain.CallFingerprint {
		out := domain.CallFingerprint{"command template": "IRRELEVANT"}
		for k, v := range entries {
			out[k] = v
		}
		return out
	}

	return &domain.Workflow{
		ID:                "ref-branch-flip",
		Name:              "Top",
		SubmittedWorkflow: branchFlipMain,
		SubmittedInputs: mustJSON(t, map[string]any{
			"Top.optional_source": "/data/optional.txt",
			"Top.fallback":        "/data/fallback.txt",
			"Top.seed":            "/data/seed.txt",
		}),
		Calls: map[string][]domain.Call{
			"Top.Producer": {{
				Name: "Top.Producer", Backend: "Local", Attempt: 1,
				Inputs:      map[string]any{"seed": "/data/seed.txt"},
				Outputs:     map[string]any{"out": producerOut},
				Fingerprint: fp(map[string]string{"input: File seed": "00000000000000000000000000000000"}),
			}},
			"Top.Consumer": {{
				Name: "Top.Consumer", Backend: "Local", Attempt: 1,
				Inputs:      map[string]any{"incoming": producerOut},
				Outputs:     map[string]any{"out": "/exec/Top/call-Consumer/c.txt"},
				Fingerprint: fp(map[string]string{"input: File incoming": "REFERENCE_VALUE"}),
			}},
		},
	}
}

func TestSoundnessFlippedBranchWithStableSourcesIsNotReuse(t *testing.T) {
	// optional_source is gone, so the conditional now selects the fallback.
	// The producer is untouched and the fallback is untouched; only the value
	// steering the choice changed.
	env := soundnessEnv(t,
		map[string]string{"main.wdl": branchFlipMain},
		map[string]any{
			"Top.fallback": "/data/fallback.txt",
			"Top.seed":     "/data/seed.txt",
		},
		branchFlipReference(t))

	got := run(t, env)

	consumer := predictionFor(t, got, "Consumer")
	if consumer.Fate == domain.FateReuse {
		t.Fatalf("Consumer was declared reusable, but the conditional steering its "+
			"input flipped, so it receives a value the reference never recorded: %+v",
			consumer)
	}
}

// A companion to the above: with the steering value untouched, the branch is
// stable, and a resolution that follows the conditional may legitimately reach
// a verdict. This exists so the guard above cannot be satisfied by refusing to
// resolve conditionals at all.
func TestSoundnessStableBranchIsClassifiable(t *testing.T) {
	env := soundnessEnv(t,
		map[string]string{"main.wdl": branchFlipMain},
		map[string]any{
			"Top.optional_source": "/data/optional.txt",
			"Top.fallback":        "/data/fallback.txt",
			"Top.seed":            "/data/seed.txt",
		},
		branchFlipReference(t))

	got := run(t, env)

	// The producer feeds the consumer on the selected branch, so whatever the
	// producer's verdict, the consumer must not be left unclassified.
	producer := predictionFor(t, got, "Producer")
	consumer := predictionFor(t, got, "Consumer")
	if producer.Fate == domain.FateUnknown {
		t.Fatalf("Producer should be classifiable: %+v", producer)
	}
	if consumer.Fate == domain.FateUnknown {
		t.Errorf("Consumer should be classifiable when the branch is stable: %+v", consumer)
	}
}
