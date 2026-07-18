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
	preds := PredictCacheReuse(graph, map[string]CallAssessment{
		"IndexVcf": {Reasons: []string{"docker image changed"}},
	})

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

	preds := PredictCacheReuse(graph, nil)

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
	preds := PredictCacheReuse(graph, map[string]CallAssessment{
		"A": {Reasons: []string{"command changed"}},
	})

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
	preds := PredictCacheReuse(graph, map[string]CallAssessment{
		"Dirty": {Reasons: []string{"input file changed"}},
	})

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
	preds := PredictCacheReuse(graph, map[string]CallAssessment{
		"A": {Unknown: "input not readable"},
	})

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
	preds := PredictCacheReuse(graph, map[string]CallAssessment{
		"Changed":    {Reasons: []string{"docker image changed"}},
		"Unknowable": {Unknown: "backend not supported"},
	})

	if s := fateOf(t, preds, "Sink"); s.Fate != FateUnknown {
		t.Errorf("Sink: got %v, want unknown", s.Fate)
	}
}

// A cyclic graph cannot come from valid WDL, but a partial parse might produce
// one and it must not hang.
func TestPredictCacheReuseSurvivesCycle(t *testing.T) {
	graph := map[string][]string{"A": {"B"}, "B": {"A"}}

	preds := PredictCacheReuse(graph, nil)

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

// A call whose inputs could not be fully verified must not be reported as
// reusable, but it is still worth more than "unknown": when an ancestor
// reruns, the more informative verdict survives.
func TestPredictCacheReuseBlockedReuseFallsBackToUnknown(t *testing.T) {
	graph := map[string][]string{"A": nil, "B": {"A"}}

	preds := PredictCacheReuse(graph, map[string]CallAssessment{
		"B": {ReuseBlocked: "an input could not be checked"},
	})

	if p := fateOf(t, preds, "A"); p.Fate != FateReuse {
		t.Errorf("A: got %v, want reuse", p.Fate)
	}
	b := fateOf(t, preds, "B")
	if b.Fate != FateUnknown {
		t.Errorf("B: got %v, want unknown when reuse is blocked and nothing upstream reruns", b.Fate)
	}
	if len(b.Reasons) != 1 || b.Reasons[0] != "an input could not be checked" {
		t.Errorf("B reasons = %v, want the blocking reason", b.Reasons)
	}
}

func TestPredictCacheReuseBlockedReuseStillInheritsRerun(t *testing.T) {
	graph := map[string][]string{"A": nil, "B": {"A"}}

	preds := PredictCacheReuse(graph, map[string]CallAssessment{
		"A": {Reasons: []string{"docker image changed"}},
		"B": {ReuseBlocked: "an input could not be checked"},
	})

	b := fateOf(t, preds, "B")
	if b.Fate != FateRerunDownstream {
		t.Errorf("B: got %v, want rerun (downstream) — more informative than unknown", b.Fate)
	}
	if b.Cause != "A" {
		t.Errorf("B cause = %q, want A", b.Cause)
	}
}
