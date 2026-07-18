package cromwell

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
)

// loadFixture parses one of the call-cache metadata payloads captured from a
// real Cromwell 91 server (see testdata/callcache/README.md for how they were
// produced and what each one pins).
func loadFixture(t *testing.T, name string) *workflow.Workflow {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "callcache", name))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", name, err)
	}
	var resp metadataResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		t.Fatalf("parsing fixture %s: %v", name, err)
	}
	return mapMetadataResponseToWorkflow(&resp)
}

func firstCall(t *testing.T, w *workflow.Workflow, name string) workflow.Call {
	t.Helper()
	calls, ok := w.Calls[name]
	if !ok || len(calls) == 0 {
		t.Fatalf("call %q missing from fixture", name)
	}
	return calls[0]
}

// The whole feature rests on hashes surviving the mapper, so assert the exact
// shape observed on the wire rather than just "not empty".
func TestMapperFlattensCallCachingHashes(t *testing.T) {
	w := loadFixture(t, "run1_reference.json")
	call := firstCall(t, w, "VcfIndexAndStats.IndexVcf")

	if len(call.Fingerprint) == 0 {
		t.Fatal("Fingerprint is empty: the mapper dropped callCaching.hashes")
	}
	// Nested groups must be flattened, not lost.
	if got := call.Fingerprint["runtime attribute: docker"]; got != "5CF5ADB49E5191F0A1DA579CA8C9FB0E" {
		t.Errorf("runtime attribute: docker = %q, want the captured hash", got)
	}
	if got := call.Fingerprint["command template"]; got != "94A353F002A219A742B02214A920582A" {
		t.Errorf("command template = %q, want the captured hash", got)
	}
	// File hashes are lowercase content MD5s, unlike the uppercase metadata ones.
	if got := call.Fingerprint["input: File input_vcf"]; got != "41a44e64f3c014c39dfc5b7b09fbf75c" {
		t.Errorf("input: File input_vcf = %q, want the captured content hash", got)
	}
	if call.CacheMode != "ReadAndWriteCache" {
		t.Errorf("CacheMode = %q, want ReadAndWriteCache", call.CacheMode)
	}
	if !call.AllowResultReuse {
		t.Error("AllowResultReuse = false, want true")
	}
}

func TestMapperReadsCacheHitProvenance(t *testing.T) {
	w := loadFixture(t, "run2_all_hits.json")

	for _, name := range []string{"VcfIndexAndStats.IndexVcf", "VcfIndexAndStats.StatsVcf"} {
		call := firstCall(t, w, name)
		if !call.CacheHit {
			t.Errorf("%s: CacheHit = false, want true", name)
		}
		src, ok := workflow.ParseCacheResult(call.CacheResult)
		if !ok {
			t.Errorf("%s: could not parse cache result %q", name, call.CacheResult)
			continue
		}
		if src.WorkflowID != "1d875251-3d90-457c-9fd4-8464d52553ce" {
			t.Errorf("%s: source workflow = %q, want the run 1 id", name, src.WorkflowID)
		}
	}
}

// The experiment this fixture pair came from: only IndexVcf's docker changed,
// and StatsVcf missed purely because its input file did. This is the end-to-end
// proof that root cause and cascade are separable from real data.
func TestFingerprintDiffSeparatesRootCauseFromCascade(t *testing.T) {
	ref := loadFixture(t, "run1_reference.json")
	cur := loadFixture(t, "run3_docker_changed.json")

	index := workflow.CompareFingerprints(
		firstCall(t, ref, "VcfIndexAndStats.IndexVcf").Fingerprint,
		firstCall(t, cur, "VcfIndexAndStats.IndexVcf").Fingerprint,
	)
	cats := workflow.Categories(index)
	if len(cats) != 1 || cats[0] != workflow.CategoryDocker {
		t.Errorf("IndexVcf categories = %v, want exactly [docker image]", cats)
	}

	stats := workflow.CompareFingerprints(
		firstCall(t, ref, "VcfIndexAndStats.StatsVcf").Fingerprint,
		firstCall(t, cur, "VcfIndexAndStats.StatsVcf").Fingerprint,
	)
	if len(stats) != 1 {
		t.Fatalf("StatsVcf: got %d changes, want exactly 1: %+v", len(stats), stats)
	}
	if stats[0].Category != workflow.CategoryInputFile {
		t.Errorf("StatsVcf change category = %v, want input file (the cascade)", stats[0].Category)
	}
}

// The command template hash is computed before input substitution: StatsVcf
// received a different input path in each run, yet its template hash is
// identical. This is what lets a *pre-submission* prediction compare WDL text
// instead of reimplementing Cromwell's hashing.
func TestCommandTemplateHashIsIndependentOfInputValues(t *testing.T) {
	ref := firstCall(t, loadFixture(t, "run1_reference.json"), "VcfIndexAndStats.StatsVcf")
	cur := firstCall(t, loadFixture(t, "run3_docker_changed.json"), "VcfIndexAndStats.StatsVcf")

	refPath, _ := ref.Inputs["input_vcf"].(string)
	curPath, _ := cur.Inputs["input_vcf"].(string)
	if refPath == "" || refPath == curPath {
		t.Fatalf("fixtures must carry different input paths, got %q and %q", refPath, curPath)
	}
	if ref.Fingerprint["command template"] != cur.Fingerprint["command template"] {
		t.Error("command template hash changed with the input value; " +
			"pre-submission prediction from WDL text alone would be unsound")
	}
}

// A miss with an identical fingerprint cannot be blamed on a change — it means
// the cached copy was unusable. This fixture is that exact situation.
func TestIdenticalFingerprintWithMissMeansUnusableCandidate(t *testing.T) {
	ref := loadFixture(t, "run1_reference.json")
	cur := loadFixture(t, "run5_outputs_deleted.json")

	call := firstCall(t, cur, "VcfIndexAndStats.IndexVcf")
	if call.CacheHit {
		t.Fatal("fixture should be a miss")
	}
	changes := workflow.CompareFingerprints(
		firstCall(t, ref, "VcfIndexAndStats.IndexVcf").Fingerprint,
		call.Fingerprint,
	)
	if len(changes) != 0 {
		t.Errorf("expected an identical fingerprint despite the miss, got %+v", changes)
	}
}
