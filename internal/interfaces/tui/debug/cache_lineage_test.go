package debug

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	workflowapp "github.com/lmtani/pumbaa/internal/application/workflow"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/interfaces/tui/common"
)

// fakeReader serves workflows by ID for the lineage resolver.
type fakeReader map[string]*workflow.Workflow

func (f fakeReader) GetMetadata(_ context.Context, id string) (*workflow.Workflow, error) {
	if wf, ok := f[id]; ok {
		return wf, nil
	}
	return nil, workflow.ErrWorkflowNotFound
}

// hit is a cache-hit call copied from the same call and shard in pointsTo.
func hit(name string, shard int, pointsTo string) workflow.Call {
	return hitFrom(name, shard, pointsTo, shard)
}

func hitFrom(name string, shard int, pointsTo string, fromShard int) workflow.Call {
	return workflow.Call{Name: name, ShardIndex: shard, Attempt: 1, Status: workflow.StatusSucceeded,
		CacheHit: true, CacheResult: fmt.Sprintf("Cache Hit: %s:%s:%d", pointsTo, name, fromShard)}
}

func ran(name string, shard int) workflow.Call {
	return workflow.Call{Name: name, ShardIndex: shard, Attempt: 1, Status: workflow.StatusSucceeded}
}

// The current run's wf.Amber is a hit on "mid", itself a hit on "orig",
// where it ran. wf.Align shard 1 is a hit within the same run (shard 0).
func lineageFixture() (*WorkflowMetadata, fakeReader) {
	current := &WorkflowMetadata{ID: "current", Name: "wf", Status: workflow.StatusSucceeded,
		Calls: map[string][]workflow.Call{
			"wf.Amber": {hit("wf.Amber", -1, "mid")},
			"wf.Align": {ran("wf.Align", 0), hitFrom("wf.Align", 1, "current", 0)},
			"wf.Lost":  {hit("wf.Lost", -1, "gone")},
		}}
	reader := fakeReader{
		"current": current,
		"mid":     {ID: "mid", Name: "wf", Calls: map[string][]workflow.Call{"wf.Amber": {hit("wf.Amber", -1, "orig")}}},
		"orig":    {ID: "orig", Name: "wf", Calls: map[string][]workflow.Call{"wf.Amber": {ran("wf.Amber", -1)}}},
	}
	return current, reader
}

func lineageModel(t *testing.T) Model {
	t.Helper()
	wf, reader := lineageFixture()
	m := NewModel(wf, nil, nil, nil, nil)
	m.SetCacheLineage(workflowapp.NewCacheLineageUseCase(reader))
	m.width, m.height = 120, 40
	m.recalcPanelWidths()
	return m
}

// run executes cmd and feeds the resulting messages back into m, returning
// any navigation request found along the way. Timers (status-bar clears)
// are dropped instead of waited for.
func run(m Model, cmd tea.Cmd) (Model, *common.NavigateToDebugMsg) {
	var nav *common.NavigateToDebugMsg
	queue := []tea.Cmd{cmd}
	for len(queue) > 0 {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := runQuick(c).(type) {
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case common.NavigateToDebugMsg:
			nav = &msg
		case lineageResolvedMsg:
			model, next := m.Update(msg)
			m = model.(Model)
			queue = append(queue, next)
		}
	}
	return m, nav
}

func focus(t *testing.T, m *Model, name string, shard int) {
	t.Helper()
	if !m.focusCall(name, shard) {
		t.Fatalf("FocusCall(%s, %d) found nothing", name, shard)
	}
}

func TestFocusCallExpandsAndSelectsShard(t *testing.T) {
	m := lineageModel(t)
	focus(t, &m, "wf.Align", 1)

	if cd := m.selectedNode().CallData; cd.Name != "wf.Align" || cd.ShardIndex != 1 {
		t.Fatalf("selected %s/%d, want wf.Align/1", cd.Name, cd.ShardIndex)
	}
	if !m.findNodeByID(m.tree, "wf.Align").Expanded {
		t.Error("the scatter holding the shard should be expanded")
	}
	if m.focusCall("wf.Missing", -1) {
		t.Error("FocusCall should report a missing call")
	}
}

func TestFocusCallClearsSearchHidingTarget(t *testing.T) {
	m := lineageModel(t)
	m.searchQuery = "Amber"
	m.updateSearchFilter()

	focus(t, &m, "wf.Align", 0)
	if m.searchQuery != "" {
		t.Errorf("searchQuery = %q, want cleared", m.searchQuery)
	}
}

func TestSelectingCacheHitResolvesAndShowsLineage(t *testing.T) {
	m := lineageModel(t)
	focus(t, &m, "wf.Amber", -1)

	m, _ = run(m, m.ensureSelectedLineage())

	entry := m.lineage.get(workflow.CacheSource{WorkflowID: "mid", CallName: "wf.Amber", ShardIndex: -1})
	if entry == nil || entry.pending || len(entry.lineage.Hops) != 2 {
		t.Fatalf("lineage entry = %+v, want 2 resolved hops", entry)
	}
	details := m.renderCacheSection(m.selectedNode())
	for _, want := range []string{"2 runs back", "mid", "orig", "ran", "producing run"} {
		if !strings.Contains(details, want) {
			t.Errorf("cache section lacks %q:\n%s", want, details)
		}
	}
}

func TestOpenOriginalWaitsForChainThenJumpsToProducingRun(t *testing.T) {
	m := lineageModel(t)
	focus(t, &m, "wf.Amber", -1)

	model, cmd := m.openOriginal(m.selectedNode())
	m = model.(Model)
	if !m.isLoading {
		t.Error("o before the chain is known should show progress")
	}

	m, nav := run(m, cmd)
	if nav == nil {
		t.Fatal("expected a navigation request once the chain resolved")
	}
	if nav.Workflow.ID != "orig" || nav.Focus == nil || nav.Focus.CallName != "wf.Amber" {
		t.Errorf("navigated to %+v, want wf.Amber in orig", nav)
	}
	if !strings.Contains(nav.Notice, "2 runs back") || nav.Origin == "" {
		t.Errorf("notice %q / origin %q should say how far back", nav.Notice, nav.Origin)
	}
	if m.isLoading {
		t.Error("loading should stop after the jump")
	}
}

func TestLineageModalOpensIntermediateRun(t *testing.T) {
	m := lineageModel(t)
	focus(t, &m, "wf.Amber", -1)
	m, _ = run(m, m.ensureSelectedLineage())

	model, _ := m.openLineageModal(m.selectedNode())
	m = model.(Model)
	if m.activeModal != ModalCacheLineage || m.lineageModal.cursor != 2 {
		t.Fatalf("modal %v cursor %d, want lineage modal on the producing run", m.activeModal, m.lineageModal.cursor)
	}
	if view := m.renderLineageModal(); !strings.Contains(view, "original") {
		t.Errorf("modal should mark the original:\n%s", view)
	}

	model, _ = m.handleLineageModalKeys(tea.KeyMsg{Type: tea.KeyUp})
	m = model.(Model)
	_, cmd := m.handleLineageModalKeys(tea.KeyMsg{Type: tea.KeyEnter})
	_, nav := run(m, cmd)
	if nav == nil || nav.Workflow.ID != "mid" {
		t.Fatalf("enter on the middle row should open mid, got %+v", nav)
	}
}

func TestOpenOriginalWithinSameWorkflowMovesCursor(t *testing.T) {
	m := lineageModel(t)
	focus(t, &m, "wf.Align", 1)

	model, cmd := m.openOriginal(m.selectedNode())
	m, nav := run(model.(Model), cmd)

	if nav != nil {
		t.Fatal("a source in this run needs no new screen")
	}
	if cd := m.selectedNode().CallData; cd.Name != "wf.Align" || cd.ShardIndex != 0 {
		t.Fatalf("selected %s/%d, want wf.Align/0", cd.Name, cd.ShardIndex)
	}
}

func TestBrokenChainShowsReasonAndOpensNothing(t *testing.T) {
	m := lineageModel(t)
	focus(t, &m, "wf.Lost", -1)

	model, cmd := m.openOriginal(m.selectedNode())
	m, nav := run(model.(Model), cmd)

	if nav != nil {
		t.Error("nothing reachable to open")
	}
	if !strings.Contains(m.statusMessage, "gone") {
		t.Errorf("status %q should name the missing workflow", m.statusMessage)
	}
}

func TestWithoutResolverCacheHitsAreOnlyDescribed(t *testing.T) {
	wf, _ := lineageFixture()
	m := NewModel(wf, nil, nil, nil, nil)
	m.focusCall("wf.Amber", -1)

	if cmd := m.ensureSelectedLineage(); cmd != nil {
		t.Error("no resolver, nothing to resolve")
	}
	if details := m.renderCacheSection(m.selectedNode()); !strings.Contains(details, "mid") {
		t.Errorf("immediate source should still be named:\n%s", details)
	}
}

func TestQuickActionHintsOnlyOnCacheHits(t *testing.T) {
	m := lineageModel(t)
	hints := func() string {
		return strings.Join(m.quickActionHints(), " ")
	}

	focus(t, &m, "wf.Amber", -1)
	if h := hints(); !strings.Contains(h, "producing run") || !strings.Contains(h, "lineage") {
		t.Errorf("cache hit should advertise o and O: %s", h)
	}
	focus(t, &m, "wf.Align", 0)
	if strings.Contains(hints(), "producing run") {
		t.Error("a call that ran here should not advertise o")
	}
}

func TestReactivateRetriesLineageAnsweredElsewhere(t *testing.T) {
	m := lineageModel(t)
	focus(t, &m, "wf.Amber", -1)
	_ = m.ensureSelectedLineage() // the answer goes to the screen on top

	cmd := m.Reactivate()
	m, _ = run(m, cmd)

	src := workflow.CacheSource{WorkflowID: "mid", CallName: "wf.Amber", ShardIndex: -1}
	if entry := m.lineage.get(src); entry == nil || entry.pending {
		t.Error("Reactivate should re-request a lineage left pending")
	}
}

func TestWatchIgnoresAnotherWorkflowsTick(t *testing.T) {
	m := lineageModel(t)
	m.watchActive = true

	if _, cmd := m.Update(watchTickMsg{workflowID: "other"}); cmd != nil {
		t.Error("a tick for another workflow must not start a refresh")
	}
}

func TestSuspendAndReactivateWatch(t *testing.T) {
	m := lineageModel(t)
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

// runQuick runs a command, giving up on the ones that sleep (tea.Tick).
func runQuick(c tea.Cmd) tea.Msg {
	out := make(chan tea.Msg, 1)
	go func() { out <- c() }()
	select {
	case msg := <-out:
		return msg
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func TestMovingOntoCacheHitRequestsLineage(t *testing.T) {
	m := lineageModel(t)
	src := workflow.CacheSource{WorkflowID: "mid", CallName: "wf.Amber", ShardIndex: -1}

	for i := 0; i < len(m.nodes) && m.lineage.get(src) == nil; i++ {
		model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = model.(Model)
	}

	if m.lineage.get(src) == nil {
		t.Fatal("moving the cursor onto a cache hit should request its lineage, and the request must survive in the returned model")
	}
}

func TestEscLeavesRightAfterSelectingANode(t *testing.T) {
	m := lineageModel(t)
	focus(t, &m, "wf.Amber", -1)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !emits[common.NavigateBackMsg](cmd) {
		t.Error("ESC on a freshly selected node should go back, not switch to an identical view")
	}
}

// emits reports whether cmd, or any command batched in it, produces a T.
func emits[T any](cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	switch msg := runQuick(cmd).(type) {
	case T:
		return true
	case tea.BatchMsg:
		for _, c := range msg {
			if emits[T](c) {
				return true
			}
		}
	}
	return false
}
