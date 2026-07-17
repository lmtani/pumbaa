package chat

import (
	"context"
	"testing"

	"google.golang.org/adk/session"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// fakeStore wraps the ADK in-memory session service with the pumbaa-specific
// queries of ports.ChatSessionStore, recording what was written.
type fakeStore struct {
	session.Service
	latest *ports.ChatSessionInfo
	labels map[string]string
}

func newFakeStore() *fakeStore {
	return &fakeStore{Service: session.InMemoryService(), labels: map[string]string{}}
}

func (f *fakeStore) ListWithSummaries(ctx context.Context, appName, userID string) ([]ports.ChatSessionInfo, error) {
	if f.latest == nil {
		return nil, nil
	}
	return []ports.ChatSessionInfo{*f.latest}, nil
}

func (f *fakeStore) FindLatestByContextLabel(ctx context.Context, appName, userID, label string) (*ports.ChatSessionInfo, error) {
	return f.latest, nil
}

func (f *fakeStore) SetContextLabel(ctx context.Context, sessionID, label string) error {
	f.labels[sessionID] = label
	return nil
}

func (f *fakeStore) UpdateSummary(ctx context.Context, sessionID, summary string) error { return nil }

func (f *fakeStore) UpdateTokenUsage(ctx context.Context, sessionID string, inputTokens, outputTokens int) error {
	return nil
}

func (f *fakeStore) Close() error { return nil }

func TestFindResumableCmdRequiresContext(t *testing.T) {
	store := newFakeStore()
	store.latest = &ports.ChatSessionInfo{ID: "sess-9"}

	// Without a context label there is nothing to resume by.
	m := newTestModel(t, store)
	if cmd := m.findResumableCmd(); cmd != nil {
		t.Errorf("no context label: expected nil cmd")
	}

	// A plain ADK service without the extended queries cannot resume.
	m2 := newTestModel(t, session.InMemoryService())
	m2.SetContextLabel("wf ▸ task")
	if cmd := m2.findResumableCmd(); cmd != nil {
		t.Errorf("service without ChatSessionStore: expected nil cmd")
	}
}

func TestFindResumableCmdReportsPreviousSession(t *testing.T) {
	store := newFakeStore()
	store.latest = &ports.ChatSessionInfo{ID: "sess-9", Summary: "old chat"}

	m := newTestModel(t, store)
	m.SetContextLabel("wf ▸ task")

	cmd := m.findResumableCmd()
	if cmd == nil {
		t.Fatal("expected a lookup command")
	}
	msg, ok := cmd().(resumableFoundMsg)
	if !ok {
		t.Fatalf("expected resumableFoundMsg, got %T", cmd())
	}
	if msg.info.ID != "sess-9" || msg.owner != m.msgs {
		t.Errorf("unexpected msg: %+v", msg)
	}
}

func TestCreateSessionCmdTagsContextLabel(t *testing.T) {
	store := newFakeStore()
	m := newTestModel(t, store)
	m.SetContextLabel("wf ▸ align")

	cmd := m.createSessionCmd()
	if cmd == nil {
		t.Fatal("expected a create command")
	}
	msg, ok := cmd().(sessionCreatedMsg)
	if !ok {
		t.Fatalf("expected sessionCreatedMsg, got %T", cmd())
	}
	if msg.err != nil {
		t.Fatalf("create failed: %v", msg.err)
	}
	if msg.session == nil {
		t.Fatal("no session returned")
	}
	if got := store.labels[msg.session.ID()]; got != "wf ▸ align" {
		t.Errorf("context label not tagged on the new session, got %q", got)
	}
}
