package cromwell

import (
	"fmt"
	"os"
	"testing"
)

// TestCostBreakdownMatchesAPI validates the per-task cost reconstruction
// (domain CalculateCostBreakdown, fed through the real metadata mapper)
// against Cromwell's authoritative /cost total. Opt-in: point it at a saved
// expanded-metadata JSON and the API cost.
//
//	PUMBAA_COST_VALIDATE=/path/to/expanded_metadata.json \
//	PUMBAA_COST_EXPECTED=5.6765 \
//	go test ./internal/infrastructure/cromwell/ -run TestCostBreakdownMatchesAPI -v
func TestCostBreakdownMatchesAPI(t *testing.T) {
	path := os.Getenv("PUMBAA_COST_VALIDATE")
	if path == "" {
		t.Skip("set PUMBAA_COST_VALIDATE to an expanded-metadata JSON to run")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}

	client := NewClient(Config{Host: "http://unused"})
	wf, err := client.ParseMetadata(data)
	if err != nil {
		t.Fatalf("parse metadata: %v", err)
	}

	b := wf.CalculateCostBreakdown()
	t.Logf("reconstructed actual total: $%.4f (+ ~%.1f resource-hours estimated) across %d tasks (pending subs: %d)",
		b.ActualTotal, b.EstimatedTotal, len(b.Tasks), b.SubworkflowsPending)
	for i, tc := range b.Tasks {
		if i >= 5 {
			break
		}
		t.Logf("  %-40s $%.2f (~%.1frh, %.1f%%, %.1fh, preempt=%v)", tc.Name, tc.ActualCost, tc.EstimatedCost, tc.Percent, tc.VMHours, tc.Preemptible)
	}

	expected := os.Getenv("PUMBAA_COST_EXPECTED")
	if expected == "" {
		return
	}
	var want float64
	if _, err := fmt.Sscanf(expected, "%g", &want); err != nil {
		t.Fatalf("bad PUMBAA_COST_EXPECTED: %v", err)
	}
	// Only real dollars are comparable to the API total; the resource-hours
	// estimate is a different unit and stays out of the check.
	// Allow 2% drift: our estimate uses call Start/End while Cromwell's cost
	// may use slightly different VM lifetime boundaries.
	diff := b.ActualTotal - want
	if diff < 0 {
		diff = -diff
	}
	if want > 0 && diff/want > 0.02 {
		t.Errorf("reconstructed $%.4f differs from API $%.4f by %.1f%% (>2%%)", b.ActualTotal, want, 100*diff/want)
	}
}
