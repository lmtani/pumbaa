package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools"
	"github.com/lmtani/pumbaa/internal/infrastructure/agents/tools/types"
	"github.com/lmtani/pumbaa/internal/infrastructure/cromwell"
	wdlindexer "github.com/lmtani/pumbaa/internal/infrastructure/wdl"
)

// TestToolsE2E exercises every built-in agent action against a real Cromwell
// server and WDL directory. It is the "are the tools actually functional"
// check, opt-in because it needs live infrastructure:
//
//	PUMBAA_TOOLS_E2E=1 CROMWELL_HOST=http://localhost:8000 \
//	PUMBAA_WDL_DIR=/path/to/workflows \
//	go test ./internal/infrastructure/agents/tools/ -run TestToolsE2E -v
func TestToolsE2E(t *testing.T) {
	if os.Getenv("PUMBAA_TOOLS_E2E") != "1" {
		t.Skip("set PUMBAA_TOOLS_E2E=1 to run the live tools check")
	}

	host := os.Getenv("CROMWELL_HOST")
	if host == "" {
		host = "http://localhost:8000"
	}
	reader := cromwell.NewClient(cromwell.Config{Host: host})

	var wdlRepo *wdlindexer.Indexer
	if dir := os.Getenv("PUMBAA_WDL_DIR"); dir != "" {
		var err error
		wdlRepo, err = wdlindexer.NewIndexer(dir, filepath.Join(t.TempDir(), "index.json"), true)
		if err != nil {
			t.Fatalf("failed to build WDL index for %s: %v", dir, err)
		}
	}

	registry := tools.NewDefaultRegistry(reader, wdlRepo)
	ctx := context.Background()

	handle := func(t *testing.T, input types.Input) types.Output {
		t.Helper()
		out, err := registry.Handle(ctx, input)
		if err != nil {
			t.Fatalf("action %s returned transport error: %v", input.Action, err)
		}
		if !out.Success {
			t.Fatalf("action %s failed: %s", input.Action, out.Error)
		}
		return out
	}

	var workflowID string

	t.Run("query", func(t *testing.T) {
		out := handle(t, types.Input{Action: "query", PageSize: 3})
		data, ok := out.Data.(map[string]any)
		if !ok {
			t.Fatalf("unexpected data shape: %T", out.Data)
		}
		wfs, _ := data["workflows"].([]map[string]any)
		if len(wfs) == 0 {
			t.Skipf("no workflows on %s; skipping id-dependent actions", host)
		}
		workflowID, _ = wfs[0]["id"].(string)
		t.Logf("query ok: total=%v, using workflow %s", data["total"], workflowID)
	})

	for _, action := range []string{"status", "metadata", "outputs", "logs"} {
		t.Run(action, func(t *testing.T) {
			if workflowID == "" {
				t.Skip("no workflow id available")
			}
			out := handle(t, types.Input{Action: action, WorkflowID: workflowID})
			t.Logf("%s ok (data type %T)", action, out.Data)
		})
	}

	t.Run("gcs_download_bad_path", func(t *testing.T) {
		// Only the error path: must fail gracefully, never panic or succeed.
		out, err := registry.Handle(ctx, types.Input{Action: "gcs_download", Path: "gs://pumbaa-e2e-nonexistent-bucket-xyz/nope.txt"})
		if err != nil {
			t.Fatalf("gcs_download returned transport error: %v", err)
		}
		if out.Success {
			t.Fatalf("gcs_download of nonexistent object should not succeed")
		}
		t.Logf("gcs_download error path ok: %s", out.Error)
	})

	if wdlRepo == nil {
		t.Log("PUMBAA_WDL_DIR not set; skipping WDL actions")
		return
	}

	var taskName string

	t.Run("wdl_list", func(t *testing.T) {
		out := handle(t, types.Input{Action: "wdl_list"})
		data, ok := out.Data.(map[string]any)
		if !ok {
			t.Fatalf("unexpected data shape: %T", out.Data)
		}
		tasks, _ := data["tasks"].([]string)
		if len(tasks) == 0 {
			t.Fatalf("wdl_list indexed no tasks from %s", os.Getenv("PUMBAA_WDL_DIR"))
		}
		taskName = tasks[0]
		t.Logf("wdl_list ok: %v tasks, %v workflows", data["task_count"], data["workflow_count"])
	})

	t.Run("wdl_search", func(t *testing.T) {
		if taskName == "" {
			t.Skip("no task name available")
		}
		out := handle(t, types.Input{Action: "wdl_search", Query: taskName})
		t.Logf("wdl_search ok (data type %T)", out.Data)
	})

	t.Run("wdl_info", func(t *testing.T) {
		if taskName == "" {
			t.Skip("no task name available")
		}
		out := handle(t, types.Input{Action: "wdl_info", Name: taskName, Type: "task"})
		t.Logf("wdl_info ok (data type %T)", out.Data)
	})
}
