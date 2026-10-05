-- Migration: 020_authentication_boundary
-- Description: Track first-boot security initialization and temporary credentials
-- Up

ALTER TABLE users ADD COLUMN must_change_password BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS security_bootstrap (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    initialized_at TEXT,
    root_user_id TEXT,
    FOREIGN KEY (root_user_id) REFERENCES users(id)
);

-- Down

DROP TABLE IF EXISTS security_bootstrap;
-- SQLite does not support dropping a column on all supported versions.
