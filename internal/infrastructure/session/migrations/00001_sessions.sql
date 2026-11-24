-- +goose Up
-- The chat sessions and their events. IF NOT EXISTS because databases from
-- before versioned migrations already have both tables, possibly without the
-- columns added since; version 2 adds those.
CREATE TABLE IF NOT EXISTS sessions (
    id            TEXT PRIMARY KEY,
    app_name      TEXT NOT NULL,
    user_id       TEXT NOT NULL,
    state         TEXT DEFAULT '{}',
    input_tokens  INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    summary       TEXT DEFAULT '',
    -- Which workflow ▸ task the chat was opened for, for resume-by-task.
    context_label TEXT DEFAULT '',
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS events (
    id            TEXT PRIMARY KEY,
    session_id    TEXT NOT NULL,
    invocation_id TEXT,
    branch        TEXT,
    author        TEXT,
    content       TEXT NOT NULL,
    timestamp     DATETIME NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_events_session ON events(session_id);
CREATE INDEX IF NOT EXISTS idx_sessions_app_user ON sessions(app_name, user_id);
