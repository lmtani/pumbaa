-- +goose Up
-- The submitted files, kept so a run can be exported and submitted again.
-- One row per kind; the bytes live here rather than on disk so forgetting a
-- run forgets its files in the same statement and nothing is ever orphaned.
CREATE TABLE IF NOT EXISTS run_files (
    host        TEXT NOT NULL,
    workflow_id TEXT NOT NULL,
    kind        TEXT NOT NULL,
    name        TEXT NOT NULL DEFAULT '',
    sha256      TEXT NOT NULL,
    size        INTEGER NOT NULL,
    data        BLOB NOT NULL,
    PRIMARY KEY (host, workflow_id, kind)
);
