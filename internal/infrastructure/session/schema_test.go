package session

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"google.golang.org/adk/session"
)

// oldSessionsSchema is the sessions database as the first chat release
// created it, before the token, summary and context columns: what the
// earliest users still have on disk.
const oldSessionsSchema = `
	CREATE TABLE sessions (
		id TEXT PRIMARY KEY,
		app_name TEXT NOT NULL,
		user_id TEXT NOT NULL,
		state TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE TABLE events (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		invocation_id TEXT,
		branch TEXT,
		author TEXT,
		content TEXT NOT NULL,
		timestamp DATETIME NOT NULL
	);
	INSERT INTO sessions (id, app_name, user_id, created_at) VALUES ('old', 'pumbaa', 'user', datetime('now'));
	INSERT INTO events (id, session_id, invocation_id, branch, author, content, timestamp) VALUES ('e1', 'old', 'inv', '', 'user', '{}', datetime('now'));`

// A database from before versioned migrations must open, keep its
// conversations, and gain the columns added since.
func TestOpensADatabaseFromBeforeVersionedMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := raw.Exec(oldSessionsSchema); err != nil {
		t.Fatalf("old schema: %v", err)
	}
	_ = raw.Close()

	svc, err := NewSQLiteService(path)
	if err != nil {
		t.Fatalf("NewSQLiteService: %v", err)
	}
	defer func() { _ = svc.Close() }()
	ctx := context.Background()

	got, err := svc.Get(ctx, &session.GetRequest{AppName: "pumbaa", UserID: "user", SessionID: "old"})
	if err != nil || got.Session.ID() != "old" {
		t.Fatalf("Get old session = %v, %v; want it kept", got, err)
	}
	if err := svc.SetContextLabel(ctx, "old", "wf ▸ align"); err != nil {
		t.Errorf("context_label was not added: %v", err)
	}
	if err := svc.UpdateSummary(ctx, "old", "a summary"); err != nil {
		t.Errorf("summary was not added: %v", err)
	}
	if err := svc.UpdateTokenUsage(ctx, "old", 10, 20); err != nil {
		t.Errorf("token columns were not added: %v", err)
	}
}

// Most databases on disk already have every column — the old code ALTERed
// them in on each open. Adding them again must not be attempted.
func TestOpensADatabaseThatAlreadyHasEveryColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := raw.Exec(oldSessionsSchema); err != nil {
		t.Fatalf("old schema: %v", err)
	}
	for _, column := range columnsAddedLater {
		if _, err := raw.Exec(`ALTER TABLE sessions ADD COLUMN ` + column.name + ` ` + column.definition); err != nil {
			t.Fatalf("add %s: %v", column.name, err)
		}
	}
	_ = raw.Close()

	for i := range 2 {
		svc, err := NewSQLiteService(path)
		if err != nil {
			t.Fatalf("NewSQLiteService #%d: %v", i+1, err)
		}
		_ = svc.Close()
	}
}
