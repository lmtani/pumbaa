// Package sqlitedb opens the SQLite databases pumbaa keeps under ~/.pumbaa,
// all the same way: readable by their owner only, waiting on another
// process's lock instead of failing, and brought up to date by goose
// migrations before anything reads them.
//
// Each store owns its schema — the migrations live next to the store, as
// numbered SQL files plus Go steps for what SQL cannot express — and this
// package owns how they are applied.
package sqlitedb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"

	_ "modernc.org/sqlite"
)

// busyTimeoutMillis is how long a statement waits for another process's
// lock before giving up. The chat writes while a reply streams and a submit
// in another terminal writes the history; neither should fail because the
// other was mid-write. WAL would let readers through too, but it needs
// shared memory, which network filesystems — the home directory of many
// HPC clusters — do not provide.
const busyTimeoutMillis = 5000

// versionTable is where goose records the applied migrations.
const versionTable = "goose_db_version"

// Schema is the set of migrations that brings a database up to date.
type Schema struct {
	// Migrations holds the SQL migrations: goose-annotated files named
	// <version>_<name>.sql at its root. May be nil.
	Migrations fs.FS
	// GoMigrations are the steps SQL cannot express, versioned in the same
	// sequence as the files.
	GoMigrations []*goose.Migration
	// LegacyVersion is the last PRAGMA user_version the hand-rolled runner
	// this package replaced wrote for this database, or 0 when it never had
	// one. A database found at user_version N with no goose table has
	// versions 1..N recorded as applied instead of run again, and after
	// migrating, user_version is raised to the applied version, capped at
	// LegacyVersion, so a binary from before goose still finds the database
	// up to date rather than replaying its steps over it.
	LegacyVersion int64
}

// Open opens the database at path, creating it owner-only when missing and
// tightening it when not, and applies the schema's pending migrations, each
// in its own transaction.
func Open(ctx context.Context, path string, schema Schema) (*sql.DB, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	if err := restrictToOwner(path); err != nil {
		return nil, fmt.Errorf("failed to restrict access to %s: %w", path, err)
	}

	// The pragma goes in the DSN, not in an Exec, so every connection the
	// pool opens gets it.
	db, err := sql.Open("sqlite", fmt.Sprintf("%s?_pragma=busy_timeout(%d)", path, busyTimeoutMillis))
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	if err := migrate(ctx, db, schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to migrate %s: %w", filepath.Base(path), err)
	}
	return db, nil
}

// restrictToOwner makes the database readable by its owner only, creating it
// that way when missing (an empty file is an empty SQLite database). The
// history keeps submitted files and the chat keeps conversations about runs;
// on a shared machine the default 0644 would hand both to every other user.
// SQLite gives its journal files the database's mode, so they follow.
func restrictToOwner(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}
	_ = f.Close()
	for _, p := range []string{path, path + "-journal", path + "-wal", path + "-shm"} {
		if err := os.Chmod(p, 0o600); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

// migrate adopts what the hand-rolled runner applied, runs what is pending,
// and keeps user_version in step for older binaries.
func migrate(ctx context.Context, db *sql.DB, schema Schema) error {
	store, err := database.NewStore(goose.DialectSQLite3, versionTable)
	if err != nil {
		return err
	}
	if err := adoptLegacyVersions(ctx, db, store, schema.LegacyVersion); err != nil {
		return fmt.Errorf("adopting the versions recorded before goose: %w", err)
	}

	// The custom dialect is how goose takes a store built here, the one
	// adoptLegacyVersions wrote to.
	provider, err := goose.NewProvider(goose.DialectCustom, db, schema.Migrations,
		goose.WithStore(store),
		goose.WithGoMigrations(schema.GoMigrations...),
		goose.WithDisableGlobalRegistry(true),
	)
	if err != nil {
		return err
	}
	if _, err := provider.Up(ctx); err != nil {
		return err
	}

	if schema.LegacyVersion == 0 {
		return nil
	}
	current, err := provider.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	return raiseLegacyVersion(ctx, db, min(current, schema.LegacyVersion))
}

// adoptLegacyVersions records versions 1..user_version as applied in a
// database the hand-rolled runner migrated, so goose does not run them again
// (an ALTER TABLE ADD COLUMN cannot run twice). It does nothing once the
// goose table exists, which is then the only record that counts.
func adoptLegacyVersions(ctx context.Context, db *sql.DB, store database.Store, legacy int64) error {
	if legacy == 0 {
		return nil
	}
	var tables int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, store.Tablename()).Scan(&tables); err != nil {
		return err
	}
	if tables > 0 {
		return nil
	}
	var version int64
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version == 0 {
		return nil
	}
	version = min(version, legacy)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := store.CreateVersionTable(ctx, tx); err != nil {
		return err
	}
	// Version 0 is the row goose writes when it creates the table itself.
	for v := int64(0); v <= version; v++ {
		if err := store.Insert(ctx, tx, database.InsertRequest{Version: v}); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// raiseLegacyVersion sets user_version to version unless it is already at or
// past it.
func raiseLegacyVersion(ctx context.Context, db *sql.DB, version int64) error {
	var current int64
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&current); err != nil {
		return err
	}
	if current >= version {
		return nil
	}
	// PRAGMA does not take a bound parameter.
	_, err := db.ExecContext(ctx, fmt.Sprintf("PRAGMA user_version = %d", version))
	return err
}
