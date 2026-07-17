package cromwell

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
)

// stubFetcher returns a pre-built workflow, standing in for the expanded
// metadata fetch + parse round trip.
type stubFetcher struct {
	wf      *workflow.Workflow
	apiCost float64
	err     error
}

func (s *stubFetcher) GetRawMetadataWithOptions(ctx context.Context, workflowID string, expand bool) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return []byte("{}"), nil
}

func (s *stubFetcher) ParseMetadata(data []byte) (*workflow.Workflow, error) {
	return s.wf, nil
}

func (s *stubFetcher) GetWorkflowCost(ctx context.Context, workflowID string) (float64, string, error) {
	if s.apiCost == 0 {
		return 0, "", errors.New("cost endpoint unavailable")
	}
	return s.apiCost, "USD", nil
}

func window(h float64) (time.Time, time.Time) {
	s, _ := time.Parse(time.RFC3339, "2026-07-16T06:00:00Z")
	return s, s.Add(time.Duration(h * float64(time.Hour)))
}

func failedWorkflow() *workflow.Workflow {
	s, e := window(1)
	return &workflow.Workflow{
		Status: workflow.StatusFailed,
		Calls: map[string][]workflow.Call{
			"WF.Align": {
				{Name: "WF.Align", ShardIndex: 0, Attempt: 1, Status: workflow.StatusFailed, Start: s, End: e,
					Stderr:   "gs://bucket/align-0/stderr",
					Failures: []workflow.Failure{{Message: "exit code 137 at gs://bucket/align-0/stderr"}}},
				{Name: "WF.Align", ShardIndex: 1, Attempt: 1, Status: workflow.StatusFailed, Start: s, End: e,
					Stderr:   "gs://bucket/align-1/stderr",
					Failures: []workflow.Failure{{Message: "exit code 137 at gs://bucket/align-1/stderr"}}},
			},
			"WF.Sort": {
				{Name: "WF.Sort", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded, Start: s, End: e, VMCostPerHour: 0.5},
			},
		},
	}
}

func TestFailuresHandlerGroupsAndHints(t *testing.T) {
	h := NewFailuresHandler(&stubFetcher{wf: failedWorkflow()})

	out, err := h.Handle(context.Background(), types.Input{Action: "failures", WorkflowID: "wf-1"})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}

	data := out.Data.(map[string]any)
	if data["failed_tasks"] != 2 {
		t.Errorf("failed_tasks = %v, want 2", data["failed_tasks"])
	}
	groups := data["groups"].([]map[string]any)
	if len(groups) != 1 {
		t.Fatalf("expected 1 deduplicated group, got %+v", groups)
	}
	if groups[0]["count"] != 2 {
		t.Errorf("group count = %v, want 2 (same signature across shards)", groups[0]["count"])
	}
	tasks := groups[0]["tasks"].([]map[string]any)
	if tasks[0]["stderr"] == "" {
		t.Errorf("group tasks should carry stderr paths for read_log chaining")
	}
}

func TestFailuresHandlerRequiresWorkflowID(t *testing.T) {
	h := NewFailuresHandler(&stubFetcher{wf: failedWorkflow()})
	out, _ := h.Handle(context.Background(), types.Input{Action: "failures"})
	if out.Success {
		t.Error("missing workflow_id should fail")
	}
}

func TestCostHandlerSeparatesUnitsAndAPITotal(t *testing.T) {
	s, e := window(2)
	wf := &workflow.Workflow{
		Status: workflow.StatusSucceeded,
		Calls: map[string][]workflow.Call{
			"WF.Priced": {{Name: "WF.Priced", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				Start: s, End: e, VMCostPerHour: 0.5, Preemptible: "2"}},
			"WF.Unpriced": {{Name: "WF.Unpriced", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
				Start: s, End: e, CPU: "2", Memory: "4 GB"}},
		},
	}
	h := NewCostHandler(&stubFetcher{wf: wf, apiCost: 1.05})

	out, err := h.Handle(context.Background(), types.Input{Action: "cost", WorkflowID: "wf-1"})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}

	data := out.Data.(map[string]any)
	if data["total_cost_usd"] != 1.0 {
		t.Errorf("total_cost_usd = %v, want 1.0 (2h × $0.5)", data["total_cost_usd"])
	}
	if data["estimated_total_resource_hours"] != 16.0 {
		t.Errorf("estimated_total_resource_hours = %v, want 16 (2cpu × 4GB × 2h)", data["estimated_total_resource_hours"])
	}
	if data["api_total"] != 1.05 {
		t.Errorf("api_total = %v, want 1.05", data["api_total"])
	}
	tasks := data["tasks"].([]map[string]any)
	if tasks[0]["task"] != "Priced" {
		t.Errorf("dollar-bearing task should sort first, got %+v", tasks[0])
	}
	if _, hasUSD := tasks[1]["cost_usd"]; hasUSD {
		t.Errorf("estimate-only task must not report cost_usd: %+v", tasks[1])
	}
}

func TestPreemptionHandlerReportsChains(t *testing.T) {
	s, e := window(1)
	s2, e2 := window(2)
	wf := &workflow.Workflow{
		Calls: map[string][]workflow.Call{
			"WF.Flaky": {
				{Name: "WF.Flaky", ShardIndex: -1, Attempt: 1, Status: "RetryableFailure", Start: s, End: e,
					VMCostPerHour: 0.5, Preemptible: "3"},
				{Name: "WF.Flaky", ShardIndex: -1, Attempt: 2, Status: workflow.StatusSucceeded, Start: s2, End: e2,
					VMCostPerHour: 0.5, Preemptible: "3"},
			},
		},
	}
	h := NewPreemptionHandler(&stubFetcher{wf: wf})

	out, err := h.Handle(context.Background(), types.Input{Action: "preemption", WorkflowID: "wf-1"})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}

	data := out.Data.(map[string]any)
	if data["total_preemptions"] != 1 {
		t.Errorf("total_preemptions = %v, want 1", data["total_preemptions"])
	}
	if data["total_attempts"] != 2 {
		t.Errorf("total_attempts = %v, want 2", data["total_attempts"])
	}
}

// stubLogsRepo implements the subset of ports.WorkflowReader used by read_log.
type stubLogsRepo struct {
	logs map[string][]workflow.CallLog
}

func (s *stubLogsRepo) Query(ctx context.Context, f workflow.QueryFilter) (*workflow.QueryResult, error) {
	return nil, errors.New("not implemented")
}
func (s *stubLogsRepo) GetStatus(ctx context.Context, id string) (workflow.Status, error) {
	return "", errors.New("not implemented")
}
func (s *stubLogsRepo) GetMetadata(ctx context.Context, id string) (*workflow.Workflow, error) {
	return nil, errors.New("not implemented")
}
func (s *stubLogsRepo) GetOutputs(ctx context.Context, id string) (map[string]any, error) {
	return nil, errors.New("not implemented")
}
func (s *stubLogsRepo) GetLogs(ctx context.Context, id string) (map[string][]workflow.CallLog, error) {
	return s.logs, nil
}

func TestReadLogResolvesTaskAndTails(t *testing.T) {
	dir := t.TempDir()
	stderrPath := filepath.Join(dir, "stderr")
	var content strings.Builder
	for i := 1; i <= 150; i++ {
		content.WriteString("line\n")
	}
	content.WriteString("FATAL: out of memory\n")
	if err := os.WriteFile(stderrPath, []byte(content.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	repo := &stubLogsRepo{logs: map[string][]workflow.CallLog{
		"WF.Align": {
			{Stderr: filepath.Join(dir, "attempt1-stderr"), Stdout: "x", Attempt: 1, ShardIndex: -1},
			{Stderr: stderrPath, Stdout: "x", Attempt: 2, ShardIndex: -1},
		},
	}}
	h := NewReadLogHandler(repo)

	// Short task name resolves the call; the latest attempt wins.
	out, err := h.Handle(context.Background(), types.Input{Action: "read_log", WorkflowID: "wf-1", Task: "Align", Lines: 10})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}
	data := out.Data.(map[string]any)
	if data["path"] != stderrPath {
		t.Errorf("path = %v, want the latest attempt's stderr", data["path"])
	}
	if data["total_lines"] != 151 || data["truncated"] != true {
		t.Errorf("tail accounting wrong: %+v", data)
	}
	if !strings.HasSuffix(data["content"].(string), "FATAL: out of memory") {
		t.Errorf("tail should end with the last line, got %q", data["content"])
	}
}

func TestReadLogUnknownTaskListsAvailable(t *testing.T) {
	repo := &stubLogsRepo{logs: map[string][]workflow.CallLog{
		"WF.Align": {{Stderr: "s", Attempt: 1, ShardIndex: -1}},
	}}
	h := NewReadLogHandler(repo)

	out, _ := h.Handle(context.Background(), types.Input{Action: "read_log", WorkflowID: "wf-1", Task: "Nope"})
	if out.Success {
		t.Fatal("unknown task should fail")
	}
	if !strings.Contains(out.Error, "WF.Align") {
		t.Errorf("error should list available tasks, got %q", out.Error)
	}
}

func TestReadLogDirectPathSkipsResolution(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "stderr")
	if err := os.WriteFile(p, []byte("a\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	h := NewReadLogHandler(nil) // repo unused when path is explicit

	out, err := h.Handle(context.Background(), types.Input{Action: "read_log", Path: p})
	if err != nil || !out.Success {
		t.Fatalf("Handle failed: err=%v out=%+v", err, out)
	}
	if out.Data.(map[string]any)["content"] != "a\nb" {
		t.Errorf("content = %q", out.Data.(map[string]any)["content"])
	}
}

func TestTailLines(t *testing.T) {
	tail, total := tailLines("a\nb\nc\n", 2)
	if tail != "b\nc" || total != 3 {
		t.Errorf("tailLines = %q/%d, want b\\nc / 3", tail, total)
	}
	tail, total = tailLines("", 5)
	if tail != "" || total != 0 {
		t.Errorf("empty content should be 0 lines, got %q/%d", tail, total)
	}
}
