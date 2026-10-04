package debug

import (
	"testing"

	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

func cacheTestWorkflow(id string) *WorkflowMetadata {
	return &WorkflowMetadata{
		ID:     id,
		Name:   "wf",
		Status: workflow.StatusSucceeded,
		Calls: map[string][]workflow.Call{
			"wf.Align": {
				{Name: "wf.Align", ShardIndex: 0, Attempt: 1, Status: workflow.StatusSucceeded},
				{Name: "wf.Align", ShardIndex: 1, Attempt: 1, Status: workflow.StatusSucceeded,
					CacheHit: true, CacheResult: "Cache Hit: src-0000-1111:wf.Align:1"},
			},
			"wf.Merge": {
				{Name: "wf.Merge", ShardIndex: -1, Attempt: 1, Status: workflow.StatusSucceeded,
					CacheHit: true, CacheResult: "Cache Hit: " + id + ":wf.Align:0"},
			},
		},
	}
}

func cacheTestModel(id string) Model {
	m := NewModel(cacheTestWorkflow(id), nil, nil, nil, nil)
	m.width, m.height = 120, 40
	m.recalcPanelWidths()
	return m
}

func selectedCall(m Model) *workflow.Call {
	return m.nodes[m.cursor].CallData
}

func TestFocusCallExpandsAndSelectsShard(t *testing.T) {
	m := cacheTestModel("wf-id")

	if !m.FocusCall("wf.Align", 1) {
		t.Fatal("FocusCall should find wf.Align shard 1")
	}
	if cd := selectedCall(m); cd == nil || cd.Name != "wf.Align" || cd.ShardIndex != 1 {
		t.Fatalf("selected %+v, want wf.Align shard 1", cd)
	}
	if !m.findNodeByID(m.tree, "wf.Align").Expanded {
		t.Error("the scatter holding the shard should be expanded")
	}
	if m.FocusCall("wf.Missing", -1) {
		t.Error("FocusCall should report a missing call")
	}
}

func TestFocusCallClearsSearchHidingTarget(t *testing.T) {
	m := cacheTestModel("wf-id")
	m.searchQuery = "Merge"
	m.updateSearchFilter()

	if !m.FocusCall("wf.Align", 0) {
		t.Fatal("FocusCall should clear the filter hiding the target")
	}
	if m.searchQuery != "" {
		t.Errorf("searchQuery = %q, want cleared", m.searchQuery)
	}
}

func TestCacheSourceOf(t *testing.T) {
	m := cacheTestModel("wf-id")
	m.FocusCall("wf.Align", 1)
	src, ok := cacheSourceOf(m.nodes[m.cursor])
	if !ok || src.WorkflowID != "src-0000-1111" || src.CallName != "wf.Align" || src.ShardIndex != 1 {
		t.Fatalf("cacheSourceOf = %+v, %v", src, ok)
	}
	if got := cacheSourceCallLabel(src); got != "Align · shard 1" {
		t.Errorf("label = %q", got)
	}

	m.FocusCall("wf.Align", 0)
	if _, ok := cacheSourceOf(m.nodes[m.cursor]); ok {
		t.Error("a call that ran here has no cache source")
	}
}

func TestOpenCacheSourceWithinSameWorkflowMovesCursor(t *testing.T) {
	m := cacheTestModel("wf-id")
	m.FocusCall("wf.Merge", -1)

	model, _ := m.openCacheSource(m.nodes[m.cursor])
	m = model.(Model)

	if cd := selectedCall(m); cd == nil || cd.Name != "wf.Align" || cd.ShardIndex != 0 {
		t.Fatalf("selected %+v, want wf.Align shard 0", cd)
	}
}

func TestOpenCacheSourceWithoutServerExplains(t *testing.T) {
	m := cacheTestModel("wf-id")
	m.FocusCall("wf.Align", 1)

	model, _ := m.openCacheSource(m.nodes[m.cursor])
	m = model.(Model)

	if m.isLoading {
		t.Error("nothing to load without a server connection")
	}
	if m.statusMessage == "" {
		t.Error("expected a status message explaining why it cannot open")
	}
}

func TestCacheSourceLoadedNavigatesWithFocus(t *testing.T) {
	m := cacheTestModel("wf-id")
	m.isLoading = true
	src := workflow.CacheSource{WorkflowID: "src", CallName: "wf.Align", ShardIndex: 1}

	model, cmd := m.Update(cacheSourceLoadedMsg{workflow: cacheTestWorkflow("src"), source: src, from: "Align"})
	if model.(Model).isLoading {
		t.Error("loading should stop")
	}
	if cmd == nil {
		t.Fatal("expected a navigation command")
	}
	nav, ok := cmd().(common.NavigateToDebugMsg)
	if !ok {
		t.Fatalf("got %T, want NavigateToDebugMsg", cmd())
	}
	if nav.Workflow.ID != "src" || nav.Focus == nil || nav.Focus.CallName != "wf.Align" || nav.Focus.Shard != 1 || nav.Origin == "" {
		t.Errorf("unexpected navigation %+v", nav)
	}
}

func TestQuickActionHintOnlyOnCacheHits(t *testing.T) {
	m := cacheTestModel("wf-id")
	hasHint := func() bool {
		node := m.nodes[m.cursor]
		for _, a := range m.quickActionsFor(node) {
			if a.key == "o" {
				return a.visible(m, node)
			}
		}
		return false
	}

	m.FocusCall("wf.Align", 1)
	if !hasHint() {
		t.Error("cache hit should advertise o")
	}
	m.FocusCall("wf.Align", 0)
	if hasHint() {
		t.Error("a call that ran here should not advertise o")
	}
}

func TestWatchIgnoresAnotherWorkflowsTick(t *testing.T) {
	m := cacheTestModel("wf-id")
	m.watchActive = true

	_, cmd := m.Update(watchTickMsg{workflowID: "other"})
	if cmd != nil {
		t.Error("a tick for another workflow must not start a refresh")
	}
}

func TestSuspendAndReactivateWatch(t *testing.T) {
	m := cacheTestModel("wf-id")
	m.watchActive = true

	m.Suspend()
	if m.watchActive || !m.watchSuspended {
		t.Fatal("Suspend should pause watch")
	}
	if cmd := m.Reactivate(); cmd == nil {
		t.Error("Reactivate should refresh right away")
	}
	if !m.watchActive || m.watchSuspended {
		t.Error("Reactivate should resume watch")
	}
}
