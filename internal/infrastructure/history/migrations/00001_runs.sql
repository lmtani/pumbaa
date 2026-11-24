-- +goose Up
-- The runs submitted from here, or annotated, keyed by (host, workflow id):
-- an ID is only unique within a server.
CREATE TABLE IF NOT EXISTS runs (
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
CREATE INDEX IF NOT EXISTS idx_runs_submitted ON runs(submitted_at DESC);
