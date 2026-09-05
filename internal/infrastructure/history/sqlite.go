// Package history provides the SQLite-backed local memory of submitted runs.
//
// It is a separate database from the chat sessions on purpose: that one
// belongs to the ADK session service and has its own lifecycle (including an
// automatic purge on open), and sharing a file would put history writes in
// lock contention with a streaming conversation.
package history

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"

	"github.com/lmtani/pumbaa/internal/application/ports"
)

// Compile-time check: the store satisfies the port it exists to implement.
var _ ports.RunHistory = (*Store)(nil)

// schemaVersion is bumped whenever migrate gains a step. Unlike the chat
// store's ignored ALTER TABLEs, migrations here are explicit and recorded in
// PRAGMA user_version.
const schemaVersion = 1

// timeLayout stores instants as text so what is written is what is read back,
// independent of driver date handling.
const timeLayout = time.RFC3339Nano

// Store is the local run history. It opens its database on first use, so
// commands that never touch history do not create a file.
type Store struct {
	path string

	once sync.Once
	db   *sql.DB
	err  error
}

// New returns a store backed by dbPath. Nothing is opened yet.
func New(dbPath string) *Store {
	return &Store{path: dbPath}
}

// DefaultPath is the history database location under the user's home.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".pumbaa", "history.db"), nil
}

func (s *Store) open() (*sql.DB, error) {
	s.once.Do(func() {
		path := s.path
		if path == "" {
			path, s.err = DefaultPath()
			if s.err != nil {
				return
			}
		}
		if dir := filepath.Dir(path); dir != "" {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				s.err = fmt.Errorf("failed to create directory %s: %w", dir, err)
				return
			}
		}
		db, err := sql.Open("sqlite", path)
		if err != nil {
			s.err = fmt.Errorf("failed to open history database: %w", err)
			return
		}
		if err := migrate(db); err != nil {
			_ = db.Close()
			s.err = fmt.Errorf("failed to migrate history database: %w", err)
			return
		}
		s.db = db
	})
	return s.db, s.err
}

// migrate brings the schema up to schemaVersion, stepping from whatever
// version the file is at.
func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		return err
	}
	if version >= schemaVersion {
		return nil
	}

	steps := []string{
		// v1: the initial runs table.
		`CREATE TABLE IF NOT EXISTS runs (
			host              TEXT NOT NULL,
			workflow_id       TEXT NOT NULL,
			host_alias        TEXT NOT NULL DEFAULT '',
			name              TEXT NOT NULL DEFAULT '',
			description       TEXT NOT NULL DEFAULT '',
			workflow_file     TEXT NOT NULL DEFAULT '',
			inputs_file       TEXT NOT NULL DEFAULT '',
			options_file      TEXT NOT NULL DEFAULT '',
			dependencies_file TEXT NOT NULL DEFAULT '',
			labels            TEXT NOT NULL DEFAULT '{}',
			origin            TEXT NOT NULL DEFAULT 'note',
			submitted_at      TEXT NOT NULL,
			last_status       TEXT NOT NULL DEFAULT '',
			status_synced_at  TEXT NOT NULL DEFAULT '',
			created_at        TEXT NOT NULL,
			updated_at        TEXT NOT NULL,
			PRIMARY KEY (host, workflow_id)
		);
		CREATE INDEX IF NOT EXISTS idx_runs_submitted ON runs(submitted_at DESC);`,
	}

	for i := version; i < schemaVersion; i++ {
		if _, err := db.Exec(steps[i]); err != nil {
			return fmt.Errorf("migration to v%d failed: %w", i+1, err)
		}
	}
	// PRAGMA does not take a bound parameter.
	if _, err := db.Exec(fmt.Sprintf("PRAGMA user_version = %d", schemaVersion)); err != nil {
		return err
	}
	return nil
}

// Close releases the database if it was ever opened.
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Record stores a run. An incoming empty description leaves whatever note the
// row already had, so recording never destroys writing.
func (s *Store) Record(ctx context.Context, rec ports.RunRecord) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	if rec.Host == "" || rec.WorkflowID == "" {
		return fmt.Errorf("history: host and workflow id are required")
	}

	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	if rec.SubmittedAt.IsZero() {
		rec.SubmittedAt = now
	}
	if rec.Origin == "" {
		rec.Origin = ports.OriginNote
	}

	labels, err := json.Marshal(rec.Labels)
	if err != nil {
		return fmt.Errorf("history: failed to encode labels: %w", err)
	}

	_, err = db.ExecContext(ctx, `
		INSERT INTO runs (
			host, workflow_id, host_alias, name, description,
			workflow_file, inputs_file, options_file, dependencies_file,
			labels, origin, submitted_at, last_status, status_synced_at,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(host, workflow_id) DO UPDATE SET
			host_alias        = excluded.host_alias,
			name              = CASE WHEN excluded.name != '' THEN excluded.name ELSE runs.name END,
			description       = CASE WHEN excluded.description != '' THEN excluded.description ELSE runs.description END,
			workflow_file     = excluded.workflow_file,
			inputs_file       = excluded.inputs_file,
			options_file      = excluded.options_file,
			dependencies_file = excluded.dependencies_file,
			labels            = excluded.labels,
			origin            = excluded.origin,
			submitted_at      = excluded.submitted_at,
			last_status       = excluded.last_status,
			status_synced_at  = excluded.status_synced_at,
			updated_at        = excluded.updated_at`,
		rec.Host, rec.WorkflowID, rec.HostAlias, rec.Name, rec.Description,
		rec.WorkflowFile, rec.InputsFile, rec.OptionsFile, rec.DependenciesFile,
		string(labels), string(rec.Origin), formatTime(rec.SubmittedAt), rec.LastStatus,
		formatTime(rec.StatusSyncedAt), formatTime(rec.CreatedAt), formatTime(now),
	)
	if err != nil {
		return fmt.Errorf("history: failed to record run: %w", err)
	}
	return nil
}

// SetDescription edits the note of a run already remembered.
func (s *Store) SetDescription(ctx context.Context, ref ports.RunRef, description string) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	res, err := db.ExecContext(ctx,
		`UPDATE runs SET description = ?, updated_at = ? WHERE host = ? AND workflow_id = ?`,
		description, formatTime(time.Now().UTC()), ref.Host, ref.WorkflowID)
	if err != nil {
		return fmt.Errorf("history: failed to update description: %w", err)
	}
	return requireAffected(res, ref)
}

// SetStatus refreshes the last known status of a remembered run.
func (s *Store) SetStatus(ctx context.Context, ref ports.RunRef, status string, at time.Time) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	res, err := db.ExecContext(ctx,
		`UPDATE runs SET last_status = ?, status_synced_at = ?, updated_at = ? WHERE host = ? AND workflow_id = ?`,
		status, formatTime(at.UTC()), formatTime(time.Now().UTC()), ref.Host, ref.WorkflowID)
	if err != nil {
		return fmt.Errorf("history: failed to update status: %w", err)
	}
	return requireAffected(res, ref)
}

// Forget drops a single record.
func (s *Store) Forget(ctx context.Context, ref ports.RunRef) error {
	db, err := s.open()
	if err != nil {
		return err
	}
	res, err := db.ExecContext(ctx, `DELETE FROM runs WHERE host = ? AND workflow_id = ?`, ref.Host, ref.WorkflowID)
	if err != nil {
		return fmt.Errorf("history: failed to forget run: %w", err)
	}
	return requireAffected(res, ref)
}

// Prune drops records submitted before the cutoff.
func (s *Store) Prune(ctx context.Context, before time.Time) (int, error) {
	db, err := s.open()
	if err != nil {
		return 0, err
	}
	res, err := db.ExecContext(ctx, `DELETE FROM runs WHERE submitted_at < ?`, formatTime(before.UTC()))
	if err != nil {
		return 0, fmt.Errorf("history: failed to prune: %w", err)
	}
	affected, err := res.RowsAffected()
	return int(affected), err
}

// Get returns a single record, or ErrRunNotRemembered.
func (s *Store) Get(ctx context.Context, ref ports.RunRef) (ports.RunRecord, error) {
	db, err := s.open()
	if err != nil {
		return ports.RunRecord{}, err
	}
	row := db.QueryRowContext(ctx, selectColumns+` FROM runs WHERE host = ? AND workflow_id = ?`, ref.Host, ref.WorkflowID)
	rec, err := scanRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return ports.RunRecord{}, ports.ErrRunNotRemembered
	}
	if err != nil {
		return ports.RunRecord{}, fmt.Errorf("history: failed to read run: %w", err)
	}
	return rec, nil
}

// List returns matching records, newest submission first.
func (s *Store) List(ctx context.Context, filter ports.RunHistoryFilter) ([]ports.RunRecord, error) {
	db, err := s.open()
	if err != nil {
		return nil, err
	}

	query := selectColumns + ` FROM runs`
	var (
		where []string
		args  []any
	)
	if filter.Host != "" {
		where = append(where, `host = ?`)
		args = append(args, filter.Host)
	}
	for _, pattern := range searchPatterns(filter.Search) {
		where = append(where, searchClause)
		args = append(args, pattern)
	}
	if !filter.Since.IsZero() {
		where = append(where, `submitted_at >= ?`)
		args = append(args, formatTime(filter.Since.UTC()))
	}
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	query += ` ORDER BY submitted_at DESC`
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("history: failed to list runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var records []ports.RunRecord
	for rows.Next() {
		rec, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("history: failed to read run: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// Lookup answers which of the given runs are remembered, in one query.
func (s *Store) Lookup(ctx context.Context, host string, workflowIDs []string) (map[string]ports.RunRecord, error) {
	found := make(map[string]ports.RunRecord, len(workflowIDs))
	if len(workflowIDs) == 0 {
		return found, nil
	}
	db, err := s.open()
	if err != nil {
		return nil, err
	}

	args := make([]any, 0, len(workflowIDs)+1)
	args = append(args, host)
	placeholders := make([]string, len(workflowIDs))
	for i, id := range workflowIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}

	rows, err := db.QueryContext(ctx,
		selectColumns+` FROM runs WHERE host = ? AND workflow_id IN (`+strings.Join(placeholders, ",")+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("history: failed to look up runs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		rec, err := scanRun(rows)
		if err != nil {
			return nil, fmt.Errorf("history: failed to read run: %w", err)
		}
		found[rec.WorkflowID] = rec
	}
	return found, rows.Err()
}

// searchable is every field a keyword could plausibly appear in, joined so a
// single LIKE per term covers them all: the note, the workflow name, the ID,
// the labels, and the paths of the files the run was assembled from — asking
// "what did I run with hg38?" is as natural as asking by description.
const searchable = `LOWER(description || ' ' || name || ' ' || workflow_id || ' ' || labels || ' ' ||
	workflow_file || ' ' || inputs_file || ' ' || options_file || ' ' || dependencies_file)`

// searchClause matches one term. Terms never contain whitespace and the parts
// above are space-joined, so a term can never match across a field boundary.
const searchClause = searchable + ` LIKE ? ESCAPE '\'`

// searchPatterns splits a search into terms, every one of which must match
// somewhere in the record. Anything else surprises: a person typing three
// words means a run described by all three, not one containing that exact
// phrase, and an agent phrases its questions the same way.
func searchPatterns(search string) []string {
	terms := strings.Fields(strings.ToLower(search))
	patterns := make([]string, 0, len(terms))
	for _, term := range terms {
		patterns = append(patterns, "%"+escapeLike(term)+"%")
	}
	return patterns
}

// escapeLike neutralises the LIKE wildcards, so a search for "hg38_v2" looks
// for that text rather than treating _ as "any character".
func escapeLike(term string) string {
	replacer := strings.NewReplacer(`\`, `\\`, "%", `\%`, "_", `\_`)
	return replacer.Replace(term)
}

const selectColumns = `SELECT host, workflow_id, host_alias, name, description,
	workflow_file, inputs_file, options_file, dependencies_file, labels, origin,
	submitted_at, last_status, status_synced_at, created_at, updated_at`

// scanner is satisfied by both *sql.Row and *sql.Rows.
type scanner interface {
	Scan(dest ...any) error
}

func scanRun(row scanner) (ports.RunRecord, error) {
	var (
		rec                                               ports.RunRecord
		labels, origin                                    string
		submittedAt, statusSyncedAt, createdAt, updatedAt string
	)
	err := row.Scan(
		&rec.Host, &rec.WorkflowID, &rec.HostAlias, &rec.Name, &rec.Description,
		&rec.WorkflowFile, &rec.InputsFile, &rec.OptionsFile, &rec.DependenciesFile,
		&labels, &origin, &submittedAt, &rec.LastStatus, &statusSyncedAt,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return ports.RunRecord{}, err
	}

	rec.Origin = ports.RunOrigin(origin)
	if labels != "" {
		// A record whose labels cannot be decoded is still worth returning:
		// the description is the point, the labels are decoration.
		_ = json.Unmarshal([]byte(labels), &rec.Labels)
	}
	rec.SubmittedAt = parseTime(submittedAt)
	rec.StatusSyncedAt = parseTime(statusSyncedAt)
	rec.CreatedAt = parseTime(createdAt)
	rec.UpdatedAt = parseTime(updatedAt)
	return rec, nil
}

func requireAffected(res sql.Result, ref ports.RunRef) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", ports.ErrRunNotRemembered, ref.WorkflowID)
	}
	return nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(timeLayout)
}

func parseTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	t, err := time.Parse(timeLayout, value)
	if err != nil {
		return time.Time{}
	}
	return t
}
