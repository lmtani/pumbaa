package session

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"

	"github.com/pressly/goose/v3"

	"github.com/lmtani/pumbaa/internal/infrastructure/sqlitedb"
)

// migrationFiles are the SQL steps of the schema, one numbered file each.
// A released step is never edited: a change is a new file, or a Go step
// below when SQL cannot express it.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// columnsAddedLater are the sessions columns added to databases that
// predate them, before migrations were versioned: they were ALTERed in on
// every open, with the error of the ones already there discarded.
var columnsAddedLater = []struct{ name, definition string }{
	{"input_tokens", "INTEGER DEFAULT 0"},
	{"output_tokens", "INTEGER DEFAULT 0"},
	{"summary", "TEXT DEFAULT ''"},
	{"context_label", "TEXT DEFAULT ''"},
}

// schema is how sessions.db is brought up to date. It never had a recorded
// version, so there is nothing to adopt: version 1 is written to find the
// tables already there, and version 2 fills in what they lack.
func schema() sqlitedb.Schema {
	files, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		// The directory is embedded at build time; a failure here is a bug.
		panic(err)
	}
	return sqlitedb.Schema{
		Migrations: files,
		GoMigrations: []*goose.Migration{
			goose.NewGoMigration(2, &goose.GoFunc{RunTx: addMissingColumns}, nil),
		},
	}
}

// addMissingColumns adds the columns a sessions table created before them
// lacks. SQLite has no ADD COLUMN IF NOT EXISTS, so it asks the table first,
// and a failing ALTER is now an error rather than a discarded one.
func addMissingColumns(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT name FROM pragma_table_info('sessions')`)
	if err != nil {
		return err
	}
	present := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			_ = rows.Close()
			return err
		}
		present[name] = true
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}

	for _, column := range columnsAddedLater {
		if present[column.name] {
			continue
		}
		if _, err := tx.ExecContext(ctx, `ALTER TABLE sessions ADD COLUMN `+column.name+` `+column.definition); err != nil {
			return err
		}
	}
	return nil
}
