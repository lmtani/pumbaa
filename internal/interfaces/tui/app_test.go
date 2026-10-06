package tui

import (
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

func stackTestWorkflow(id string) *workflow.Workflow {
	return &workflow.Workflow{
		ID:     id,
		Name:   "wf-" + id,
		Status: workflow.StatusSucceeded,
		Calls: map[string][]workflow.Call{
			"wf.Task": {{Name: "wf.Task", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded}},
		},
	}
}

func TestCacheSourceStacksDebugScreensAndEscReturns(t *testing.T) {
	first := stackTestWorkflow("first")
	m := NewAppModelWithWorkflow(&Dependencies{}, first)

	model, _ := m.Update(common.NavigateToDebugMsg{
		Workflow: stackTestWorkflow("source"),
		Focus:    &common.CallFocus{CallName: "wf.Task", Shard: -1},
		Origin:   "Task (wf-first)",
	})
	m = model.(AppModel)

	if m.debugWorkflow.ID != "source" {
		t.Fatalf("debug screen shows %q, want source", m.debugWorkflow.ID)
	}
	if len(m.debugStack) != 1 {
		t.Fatalf("debugStack has %d frames, want 1", len(m.debugStack))
	}

	model, _ = m.Update(common.NavigateBackMsg{})
	m = model.(AppModel)

	if m.currentScreen != ScreenDebug || m.debugWorkflow != first {
		t.Fatalf("back should return to the first debug screen, got screen %d wf %q", m.currentScreen, m.debugWorkflow.ID)
	}
	if len(m.debugStack) != 0 {
		t.Errorf("debugStack should be empty, has %d", len(m.debugStack))
	}

	// Back again from the root screen starts the quit flow, as before
	model, _ = m.Update(common.NavigateBackMsg{})
	if !model.(AppModel).showQuitConfirm {
		t.Error("back from the first screen should ask to quit")
	}
}

func TestCacheSourceIgnoredWhenUserLeftDebug(t *testing.T) {
	m := NewAppModelWithWorkflow(&Dependencies{}, stackTestWorkflow("first"))
	m.currentScreen = ScreenChat

	model, _ := m.Update(common.NavigateToDebugMsg{Workflow: stackTestWorkflow("source"), Origin: "Task"})
	m = model.(AppModel)

	if m.currentScreen != ScreenChat || m.debugWorkflow.ID != "first" {
		t.Error("a late cache-source answer must not pull the user out of another screen")
	}
}
