package history

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"io/fs"
	"strings"

	"github.com/pressly/goose/v3"

	"github.com/lmtani/pumbaa/internal/application/ports"
	"github.com/lmtani/pumbaa/internal/domain/workflow"
	"github.com/lmtani/pumbaa/internal/infrastructure/sqlitedb"
)

// migrationFiles are the SQL steps of the schema, one numbered file each.
// A released step is never edited: a change is a new file, or a Go step
// below when SQL cannot express it.
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

// legacyVersion is the last PRAGMA user_version the hand-rolled runner wrote
// before goose took over: versions 1 to 3 below are the same steps it ran.
const legacyVersion = 3

// schema is how history.db is brought up to date.
func schema() sqlitedb.Schema {
	files, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		// The directory is embedded at build time; a failure here is a bug.
		panic(err)
	}
	return sqlitedb.Schema{
		Migrations: files,
		GoMigrations: []*goose.Migration{
			// v3: credential options are no longer kept. withheld names the
			// ones taken out of an options file, comma-separated, so an
			// export can say which to add back; the rows v2 kept are
			// scrubbed in the same transaction.
			goose.NewGoMigration(3, &goose.GoFunc{RunTx: func(ctx context.Context, tx *sql.Tx) error {
				if _, err := tx.ExecContext(ctx, `ALTER TABLE run_files ADD COLUMN withheld TEXT NOT NULL DEFAULT ''`); err != nil {
					return err
				}
				return withholdKeptCredentials(ctx, tx)
			}}, nil),
		},
		LegacyVersion: legacyVersion,
	}
}

// withholdKeptCredentials takes the credential options out of every options
// file kept before the history stopped keeping them, recording which were
// there, as a submission does now. Options that cannot be read as an object
// cannot be vouched for and are dropped, as a submission would not keep
// them. secure_delete makes SQLite zero the bytes it frees, so the
// credentials do not linger in the file's free pages.
func withholdKeptCredentials(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `PRAGMA secure_delete = ON`); err != nil {
		return err
	}
	type keptOptions struct {
		host, workflowID string
		data             []byte
	}
	rows, err := tx.QueryContext(ctx, `SELECT host, workflow_id, data FROM run_files WHERE kind = ?`, string(ports.RunFileOptions))
	if err != nil {
		return err
	}
	var all []keptOptions
	for rows.Next() {
		var k keptOptions
		if err := rows.Scan(&k.host, &k.workflowID, &k.data); err != nil {
			_ = rows.Close()
			return err
		}
		all = append(all, k)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}

	for _, k := range all {
		data, withheld, err := workflow.WithoutCredentials(k.data)
		if err != nil {
			if _, err := tx.ExecContext(ctx, `DELETE FROM run_files WHERE host = ? AND workflow_id = ? AND kind = ?`,
				k.host, k.workflowID, string(ports.RunFileOptions)); err != nil {
				return err
			}
			continue
		}
		if len(withheld) == 0 {
			continue
		}
		sum := sha256.Sum256(data)
		if _, err := tx.ExecContext(ctx, `UPDATE run_files SET data = ?, sha256 = ?, size = ?, withheld = ? WHERE host = ? AND workflow_id = ? AND kind = ?`,
			data, hex.EncodeToString(sum[:]), len(data), strings.Join(withheld, ","),
			k.host, k.workflowID, string(ports.RunFileOptions)); err != nil {
			return err
		}
	}
	return nil
}
