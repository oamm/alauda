-- Migration: 001_create_environments
-- Description: Create environments table
-- Up

CREATE TABLE IF NOT EXISTS environments (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tier TEXT,
    tags TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_environments_key ON environments(key) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_environments_enabled ON environments(enabled) WHERE deleted_at IS NULL;

-- Down

DROP TABLE IF EXISTS environments;
