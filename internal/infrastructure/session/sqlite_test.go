package session

import (
	"context"
	"path/filepath"
	"testing"

	"google.golang.org/adk/session"
	"google.golang.org/genai"
)

func newTestService(t *testing.T) *SQLiteService {
	t.Helper()
	svc, err := NewSQLiteService(filepath.Join(t.TempDir(), "sessions.db"))
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}
	t.Cleanup(func() { svc.Close() })
	return svc
}

func TestFindLatestByContextLabel(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Create(ctx, &session.CreateRequest{AppName: DefaultAppName, UserID: DefaultUserID})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := svc.SetContextLabel(ctx, resp.Session.ID(), "wf ▸ align"); err != nil {
		t.Fatalf("set context label failed: %v", err)
	}

	// Sessions without events must not be offered for resume
	info, err := svc.FindLatestByContextLabel(ctx, DefaultAppName, DefaultUserID, "wf ▸ align")
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if info != nil {
		t.Fatalf("expected no resumable session before any event, got %s", info.ID)
	}

	ev := session.NewEvent("")
	ev.Author = "user"
	ev.Content = &genai.Content{Role: "user", Parts: []*genai.Part{genai.NewPartFromText("why did it fail?")}}
	if err := svc.AppendEvent(ctx, resp.Session, ev); err != nil {
		t.Fatalf("append event failed: %v", err)
	}

	info, err = svc.FindLatestByContextLabel(ctx, DefaultAppName, DefaultUserID, "wf ▸ align")
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if info == nil {
		t.Fatalf("expected resumable session after event")
	}
	if info.ID != resp.Session.ID() || info.ContextLabel != "wf ▸ align" || info.EventCount != 1 {
		t.Errorf("unexpected info: %+v", info)
	}

	// Different label finds nothing
	other, err := svc.FindLatestByContextLabel(ctx, DefaultAppName, DefaultUserID, "wf ▸ other_task")
	if err != nil {
		t.Fatalf("find failed: %v", err)
	}
	if other != nil {
		t.Errorf("expected no session for a different label, got %s", other.ID)
	}
}

func TestListWithSummariesIncludesContextLabel(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	resp, err := svc.Create(ctx, &session.CreateRequest{AppName: DefaultAppName, UserID: DefaultUserID})
	if err != nil {
		t.Fatalf("create failed: %v", err)
	}
	if err := svc.SetContextLabel(ctx, resp.Session.ID(), "wf ▸ merge"); err != nil {
		t.Fatalf("set context label failed: %v", err)
	}

	sessions, err := svc.ListWithSummaries(ctx, DefaultAppName, DefaultUserID)
	if err != nil {
		t.Fatalf("list failed: %v", err)
	}
	if len(sessions) != 1 || sessions[0].ContextLabel != "wf ▸ merge" {
		t.Errorf("context label not returned in list: %+v", sessions)
	}
}
