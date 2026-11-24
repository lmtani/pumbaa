package sqlitedb

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"testing/fstest"
	"time"

	"github.com/pressly/goose/v3"
)

// testSchema is three steps: two SQL files and a Go step whose ALTER TABLE
// fails if it ever runs twice, which is what an adopted version must never
// do.
func testSchema() Schema {
	return Schema{
		Migrations: fstest.MapFS{
			"00001_items.sql": {Data: []byte("-- +goose Up\nCREATE TABLE IF NOT EXISTS items (id TEXT PRIMARY KEY);\n")},
			"00002_tags.sql":  {Data: []byte("-- +goose Up\nCREATE TABLE IF NOT EXISTS tags (name TEXT PRIMARY KEY);\n")},
		},
		GoMigrations: []*goose.Migration{
			goose.NewGoMigration(3, &goose.GoFunc{RunTx: func(ctx context.Context, tx *sql.Tx) error {
				_, err := tx.ExecContext(ctx, `ALTER TABLE items ADD COLUMN note TEXT NOT NULL DEFAULT ''`)
				return err
			}}, nil),
		},
		LegacyVersion: 3,
	}
}

func appliedVersions(t *testing.T, db *sql.DB) []int64 {
	t.Helper()
	rows, err := db.Query(`SELECT version_id FROM ` + versionTable + ` WHERE is_applied ORDER BY version_id`)
	if err != nil {
		t.Fatalf("versions: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var versions []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan: %v", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("versions: %v", err)
	}
	return versions
}

func userVersion(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	var v int64
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	return v
}

func TestOpenMigratesAFreshDatabase(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "x.db"), testSchema())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if got := appliedVersions(t, db); len(got) != 4 || got[3] != 3 {
		t.Errorf("applied versions = %v, want 0..3", got)
	}
	if _, err := db.Exec(`INSERT INTO items (id, note) VALUES ('a', 'b')`); err != nil {
		t.Errorf("the migrated schema is not there: %v", err)
	}
	// A binary from before goose reads user_version: it must find the
	// database up to date rather than replay its own steps over it.
	if got := userVersion(t, db); got != 3 {
		t.Errorf("user_version = %d, want 3", got)
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	for i := range 2 {
		db, err := Open(context.Background(), path, testSchema())
		if err != nil {
			t.Fatalf("Open #%d: %v", i+1, err)
		}
		_ = db.Close()
	}
}

// A database the hand-rolled runner migrated to user_version 2 has no goose
// table. Versions 1 and 2 must be adopted, not run again, and 3 must run.
func TestOpenAdoptsTheVersionsRecordedBeforeGoose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	// Version 1 as the old runner left it, with a row. Version 2's table is
	// left out on purpose: user_version says 2, so finding the table created
	// afterwards would prove goose ran a step it should have adopted.
	for _, stmt := range []string{
		`CREATE TABLE items (id TEXT PRIMARY KEY)`,
		`INSERT INTO items (id) VALUES ('kept')`,
		`PRAGMA user_version = 2`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	_ = raw.Close()

	db, err := Open(context.Background(), path, testSchema())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if got := appliedVersions(t, db); len(got) != 4 {
		t.Errorf("applied versions = %v, want 0..3", got)
	}
	var tags int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name = 'tags'`).Scan(&tags); err != nil || tags != 0 {
		t.Errorf("tags table count = %d (%v): version 2 was run although user_version said it was applied", tags, err)
	}
	var note string
	if err := db.QueryRow(`SELECT note FROM items WHERE id = 'kept'`).Scan(&note); err != nil {
		t.Errorf("version 3 did not run on the adopted database: %v", err)
	}
	if got := userVersion(t, db); got != 3 {
		t.Errorf("user_version = %d, want 3", got)
	}
}

// A failing step must leave nothing behind: not its schema change, not its
// version, so the next open runs it again whole.
func TestAFailedMigrationIsRetriedWhole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	schema := testSchema()
	fail := true
	schema.GoMigrations = []*goose.Migration{
		goose.NewGoMigration(3, &goose.GoFunc{RunTx: func(ctx context.Context, tx *sql.Tx) error {
			if _, err := tx.ExecContext(ctx, `ALTER TABLE items ADD COLUMN note TEXT NOT NULL DEFAULT ''`); err != nil {
				return err
			}
			if fail {
				return errors.New("data step failed")
			}
			return nil
		}}, nil),
	}

	if _, err := Open(context.Background(), path, schema); err == nil {
		t.Fatal("Open succeeded although a migration failed")
	}
	fail = false
	db, err := Open(context.Background(), path, schema)
	if err != nil {
		t.Fatalf("Open after the failure: %v", err)
	}
	defer func() { _ = db.Close() }()
	if got := appliedVersions(t, db); len(got) != 4 {
		t.Errorf("applied versions = %v, want 0..3", got)
	}
}

func TestOpenKeepsTheDatabaseToItsOwner(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX permissions")
	}
	dir := filepath.Join(t.TempDir(), "pumbaa")
	path := filepath.Join(dir, "x.db")

	db, err := Open(context.Background(), path, testSchema())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	_ = db.Close()
	for p, want := range map[string]os.FileMode{dir: 0o700, path: 0o600} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %o, want %o", filepath.Base(p), got, want)
		}
	}

	// A database created world-readable is tightened on open.
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	db, err = Open(context.Background(), path, testSchema())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	_ = db.Close()
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Errorf("mode after reopening = %o, want 600", info.Mode().Perm())
	}
}

// Another process holding the write lock must make a write wait, not fail:
// a submit writing the history while the chat streams into its database.
func TestAWriteWaitsForAnotherConnectionsLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.db")
	db, err := Open(context.Background(), path, testSchema())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// A separate handle stands in for the other process.
	other, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("Open other: %v", err)
	}
	defer func() { _ = other.Close() }()
	conn, err := other.Conn(context.Background())
	if err != nil {
		t.Fatalf("Conn: %v", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.ExecContext(context.Background(), `BEGIN IMMEDIATE`); err != nil {
		t.Fatalf("BEGIN IMMEDIATE: %v", err)
	}
	release := time.AfterFunc(200*time.Millisecond, func() {
		_, _ = conn.ExecContext(context.Background(), `COMMIT`)
	})
	defer release.Stop()

	if _, err := db.Exec(`INSERT INTO items (id) VALUES ('waited')`); err != nil {
		t.Errorf("the write failed instead of waiting for the lock: %v", err)
	}
}
