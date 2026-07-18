package workflow

import "testing"

func fateOf(t *testing.T, preds []CallPrediction, call string) CallPrediction {
	t.Helper()
	for _, p := range preds {
		if p.Call == call {
			return p
		}
	}
	t.Fatalf("call %q missing from predictions %+v", call, preds)
	return CallPrediction{}
}

// The showcase experiment in miniature: IndexVcf → StatsVcf, with docker
// changed on IndexVcf only. IndexVcf is the root cause; StatsVcf inherits.
func TestPredictCacheReuseCascadesFromRootCause(t *testing.T) {
	graph := map[string][]string{
		"IndexVcf": nil,
		"StatsVcf": {"IndexVcf"},
	}
	changed := map[string][]string{"IndexVcf": {"docker image changed"}}

	preds := PredictCacheReuse(graph, changed, nil)

	if p := fateOf(t, preds, "IndexVcf"); p.Fate != FateRerun {
		t.Errorf("IndexVcf: got %v, want rerun", p.Fate)
	}
	stats := fateOf(t, preds, "StatsVcf")
	if stats.Fate != FateRerunDownstream {
		t.Errorf("StatsVcf: got %v, want rerun (downstream)", stats.Fate)
	}
	if stats.Cause != "IndexVcf" {
		t.Errorf("StatsVcf cause: got %q, want IndexVcf", stats.Cause)
	}
}

// An unchanged workflow must predict full reuse — the case that tells a user
// their run is free.
func TestPredictCacheReusePredictsFullReuseWhenNothingChanged(t *testing.T) {
	graph := map[string][]string{"A": nil, "B": {"A"}, "C": {"B"}}

	preds := PredictCacheReuse(graph, nil, nil)

	if len(preds) != 3 {
		t.Fatalf("expected 3 predictions, got %d", len(preds))
	}
	for _, p := range preds {
		if p.Fate != FateReuse {
			t.Errorf("%s: got %v, want reuse", p.Call, p.Fate)
		}
	}
}

// A chain A→B→C with A changed must blame A for C, not the immediate parent B.
func TestPredictCacheReuseBlamesRootNotImmediateParent(t *testing.T) {
	graph := map[string][]string{"A": nil, "B": {"A"}, "C": {"B"}}
	changed := map[string][]string{"A": {"command changed"}}

	preds := PredictCacheReuse(graph, changed, nil)

	c := fateOf(t, preds, "C")
	if c.Fate != FateRerunDownstream {
		t.Errorf("C: got %v, want rerun (downstream)", c.Fate)
	}
	if c.Cause != "A" {
		t.Errorf("C cause: got %q, want A (the root, not B)", c.Cause)
	}
}

// A call reached by both a changed and an unchanged upstream still reruns.
func TestPredictCacheReuseFanInRerunsWhenAnyUpstreamChanges(t *testing.T) {
	graph := map[string][]string{
		"Clean":  nil,
		"Dirty":  nil,
		"Merged": {"Clean", "Dirty"},
	}
	changed := map[string][]string{"Dirty": {"input file changed"}}

	preds := PredictCacheReuse(graph, changed, nil)

	if p := fateOf(t, preds, "Clean"); p.Fate != FateReuse {
		t.Errorf("Clean: got %v, want reuse", p.Fate)
	}
	merged := fateOf(t, preds, "Merged")
	if merged.Fate != FateRerunDownstream {
		t.Errorf("Merged: got %v, want rerun (downstream)", merged.Fate)
	}
	if merged.Cause != "Dirty" {
		t.Errorf("Merged cause: got %q, want Dirty", merged.Cause)
	}
}

// An unknowable input poisons the subtree: we must not claim reuse downstream
// of something we could not verify.
func TestPredictCacheReuseUnknownPoisonsDownstream(t *testing.T) {
	graph := map[string][]string{"A": nil, "B": {"A"}}
	unknown := map[string]string{"A": "input not readable"}

	preds := PredictCacheReuse(graph, nil, unknown)

	a := fateOf(t, preds, "A")
	if a.Fate != FateUnknown {
		t.Errorf("A: got %v, want unknown", a.Fate)
	}
	if len(a.Reasons) != 1 || a.Reasons[0] != "input not readable" {
		t.Errorf("A reasons: got %v, want [input not readable]", a.Reasons)
	}
	if b := fateOf(t, preds, "B"); b.Fate != FateUnknown {
		t.Errorf("B: got %v, want unknown (downstream of unknown)", b.Fate)
	}
}

// Unknown must win over rerun: if one upstream is unknowable, "will rerun" is
// not a claim we can make.
func TestPredictCacheReuseUnknownWinsOverRerun(t *testing.T) {
	graph := map[string][]string{
		"Unknowable": nil,
		"Changed":    nil,
		"Sink":       {"Changed", "Unknowable"},
	}
	changed := map[string][]string{"Changed": {"docker image changed"}}
	unknown := map[string]string{"Unknowable": "backend not supported"}

	preds := PredictCacheReuse(graph, changed, unknown)

	if s := fateOf(t, preds, "Sink"); s.Fate != FateUnknown {
		t.Errorf("Sink: got %v, want unknown", s.Fate)
	}
}

// A cyclic graph cannot come from valid WDL, but a partial parse might produce
// one and it must not hang.
func TestPredictCacheReuseSurvivesCycle(t *testing.T) {
	graph := map[string][]string{"A": {"B"}, "B": {"A"}}

	preds := PredictCacheReuse(graph, nil, nil)

	if len(preds) != 2 {
		t.Fatalf("expected 2 predictions, got %d", len(preds))
	}
	for _, p := range preds {
		if p.Fate == FateReuse {
			t.Errorf("%s: cycle must not be reported as reuse", p.Call)
		}
	}
}

func TestCacheForecastCountsAndRootCauses(t *testing.T) {
	f := CacheForecast{Calls: []CallPrediction{
		{Call: "A", Fate: FateRerun, Reasons: []string{"docker image changed"}},
		{Call: "B", Fate: FateRerunDownstream, Cause: "A"},
		{Call: "C", Fate: FateReuse},
		{Call: "D", Fate: FateReuse},
	}}

	counts := f.Counts()
	if counts[FateReuse] != 2 || counts[FateRerun] != 1 || counts[FateRerunDownstream] != 1 {
		t.Errorf("Counts() = %v", counts)
	}
	roots := f.RootCauses()
	if len(roots) != 1 || roots[0].Call != "A" {
		t.Errorf("RootCauses() = %+v, want only A", roots)
	}
}

func TestClassifyBackend(t *testing.T) {
	tests := []struct {
		name string
		want BackendKind
	}{
		{"Local", BackendLocal},
		{"local", BackendLocal},
		{"LocalWithDocker", BackendLocal},
		{"PAPIv2", BackendGCP},
		{"PAPIv2-beta", BackendGCP},
		{"GCPBATCH", BackendGCP},
		{"JES", BackendGCP},
		{"SLURM", BackendUnsupported},
		{"AWSBatch", BackendUnsupported},
		{"", BackendUnsupported},
	}
	for _, tt := range tests {
		if got := ClassifyBackend(tt.name); got != tt.want {
			t.Errorf("ClassifyBackend(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
	if !BackendGCP.Supported() || !BackendLocal.Supported() || BackendUnsupported.Supported() {
		t.Error("Supported() disagrees with the two backends we claim to handle")
	}
}
