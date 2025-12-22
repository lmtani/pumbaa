// Package session provides a SQLite-based implementation of the ADK session.Service interface.
package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sync"
	"time"

	"google.golang.org/adk/model"
	"google.golang.org/adk/session"
	"google.golang.org/genai"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteService implements session.Service using SQLite for persistence.
type SQLiteService struct {
	db *sql.DB
	mu sync.RWMutex
}

// sqliteSession implements session.Session interface.
type sqliteSession struct {
	id             string
	appName        string
	userID         string
	state          *sqliteState
	events         *sqliteEvents
	lastUpdateTime time.Time
}

func (s *sqliteSession) ID() string                { return s.id }
func (s *sqliteSession) AppName() string           { return s.appName }
func (s *sqliteSession) UserID() string            { return s.userID }
func (s *sqliteSession) State() session.State      { return s.state }
func (s *sqliteSession) Events() session.Events    { return s.events }
func (s *sqliteSession) LastUpdateTime() time.Time { return s.lastUpdateTime }

// sqliteState implements session.State interface.
type sqliteState struct {
	data map[string]any
	mu   sync.RWMutex
}

func (s *sqliteState) Get(key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if val, ok := s.data[key]; ok {
		return val, nil
	}
	return nil, session.ErrStateKeyNotExist
}

func (s *sqliteState) Set(key string, val any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = val
	return nil
}

func (s *sqliteState) All() iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for k, v := range s.data {
			if !yield(k, v) {
				return
			}
		}
	}
}

// sqliteEvents implements session.Events interface.
type sqliteEvents struct {
	events []*session.Event
}

func (e *sqliteEvents) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, ev := range e.events {
			if !yield(ev) {
				return
			}
		}
	}
}

func (e *sqliteEvents) Len() int {
	return len(e.events)
}

func (e *sqliteEvents) At(i int) *session.Event {
	if i < 0 || i >= len(e.events) {
		return nil
	}
	return e.events[i]
}

// NewSQLiteService creates a new SQLite-based session service.
// If dbPath is empty, it defaults to ~/.pumbaa/sessions.db
func NewSQLiteService(dbPath string) (*SQLiteService, error) {
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, ".pumbaa", "sessions.db")
	}

	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	svc := &SQLiteService{db: db}
	if err := svc.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return svc, nil
}

func (s *SQLiteService) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		app_name TEXT NOT NULL,
		user_id TEXT NOT NULL,
		state TEXT DEFAULT '{}',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		invocation_id TEXT,
		branch TEXT,
		author TEXT,
		content TEXT NOT NULL,
		timestamp DATETIME NOT NULL,
		FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
	);
	
	CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_app_user ON sessions(app_name, user_id);
	`
	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection.
func (s *SQLiteService) Close() error {
	return s.db.Close()
}

// Create creates a new session.
func (s *SQLiteService) Create(ctx context.Context, req *session.CreateRequest) (*session.CreateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	stateJSON, err := json.Marshal(req.State)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal state: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sessions (id, app_name, user_id, state) VALUES (?, ?, ?, ?)`,
		sessionID, req.AppName, req.UserID, string(stateJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	state := make(map[string]any)
	if req.State != nil {
		state = req.State
	}

	sess := &sqliteSession{
		id:             sessionID,
		appName:        req.AppName,
		userID:         req.UserID,
		state:          &sqliteState{data: state},
		events:         &sqliteEvents{events: []*session.Event{}},
		lastUpdateTime: time.Now(),
	}

	return &session.CreateResponse{Session: sess}, nil
}

// Get retrieves a session by ID.
func (s *SQLiteService) Get(ctx context.Context, req *session.GetRequest) (*session.GetResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var stateJSON string
	var updatedAt time.Time

	err := s.db.QueryRowContext(ctx,
		`SELECT state, updated_at FROM sessions WHERE id = ? AND app_name = ? AND user_id = ?`,
		req.SessionID, req.AppName, req.UserID,
	).Scan(&stateJSON, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found: %s", req.SessionID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	state := make(map[string]any)
	if err := json.Unmarshal([]byte(stateJSON), &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Load events
	events, err := s.loadEvents(ctx, req.SessionID, req.NumRecentEvents, req.After)
	if err != nil {
		return nil, fmt.Errorf("failed to load events: %w", err)
	}

	sess := &sqliteSession{
		id:             req.SessionID,
		appName:        req.AppName,
		userID:         req.UserID,
		state:          &sqliteState{data: state},
		events:         &sqliteEvents{events: events},
		lastUpdateTime: updatedAt,
	}

	return &session.GetResponse{Session: sess}, nil
}

func (s *SQLiteService) loadEvents(ctx context.Context, sessionID string, limit int, after time.Time) ([]*session.Event, error) {
	query := `SELECT id, invocation_id, branch, author, content, timestamp 
	          FROM events WHERE session_id = ?`
	args := []any{sessionID}

	if !after.IsZero() {
		query += ` AND timestamp >= ?`
		args = append(args, after)
	}

	query += ` ORDER BY timestamp ASC`

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*session.Event
	for rows.Next() {
		var id, invocationID, branch, author, contentJSON string
		var timestamp time.Time

		if err := rows.Scan(&id, &invocationID, &branch, &author, &contentJSON, &timestamp); err != nil {
			return nil, err
		}

		// Deserialize content
		var content genai.Content
		if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
			// Skip malformed events
			continue
		}

		ev := &session.Event{
			LLMResponse: model.LLMResponse{
				Content: &content,
			},
			ID:           id,
			Timestamp:    timestamp,
			InvocationID: invocationID,
			Branch:       branch,
			Author:       author,
		}
		events = append(events, ev)
	}

	return events, rows.Err()
}

// List lists all sessions for a user.
func (s *SQLiteService) List(ctx context.Context, req *session.ListRequest) (*session.ListResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, state, updated_at FROM sessions WHERE app_name = ? AND user_id = ? ORDER BY updated_at DESC`,
		req.AppName, req.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []session.Session
	for rows.Next() {
		var id, stateJSON string
		var updatedAt time.Time

		if err := rows.Scan(&id, &stateJSON, &updatedAt); err != nil {
			return nil, err
		}

		state := make(map[string]any)
		json.Unmarshal([]byte(stateJSON), &state)

		sess := &sqliteSession{
			id:             id,
			appName:        req.AppName,
			userID:         req.UserID,
			state:          &sqliteState{data: state},
			events:         &sqliteEvents{events: []*session.Event{}}, // Don't load all events for list
			lastUpdateTime: updatedAt,
		}
		sessions = append(sessions, sess)
	}

	return &session.ListResponse{Sessions: sessions}, nil
}

// Delete deletes a session.
func (s *SQLiteService) Delete(ctx context.Context, req *session.DeleteRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Delete events first
	_, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE session_id = ?`, req.SessionID)
	if err != nil {
		return fmt.Errorf("failed to delete events: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE id = ? AND app_name = ? AND user_id = ?`,
		req.SessionID, req.AppName, req.UserID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// AppendEvent appends an event to a session.
func (s *SQLiteService) AppendEvent(ctx context.Context, sess session.Session, ev *session.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize content
	contentJSON, err := json.Marshal(ev.Content)
	if err != nil {
		return fmt.Errorf("failed to marshal event content: %w", err)
	}

	eventID := ev.ID
	if eventID == "" {
		eventID = fmt.Sprintf("event_%d", time.Now().UnixNano())
	}

	timestamp := ev.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO events (id, session_id, invocation_id, branch, author, content, timestamp)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		eventID, sess.ID(), ev.InvocationID, ev.Branch, ev.Author, string(contentJSON), timestamp,
	)
	if err != nil {
		return fmt.Errorf("failed to append event: %w", err)
	}

	// Update session timestamp
	_, err = s.db.ExecContext(ctx,
		`UPDATE sessions SET updated_at = ? WHERE id = ?`,
		time.Now(), sess.ID(),
	)
	if err != nil {
		return fmt.Errorf("failed to update session timestamp: %w", err)
	}

	return nil
}
