-- Migration: 026_reuse_service_names
-- Description: Enforce unique names only for active services.
-- SQLite: rebuild table
-- Up

CREATE TABLE services_new (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    display_name TEXT NOT NULL,
    description TEXT,
    tags TEXT NOT NULL DEFAULT '{}',
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

INSERT INTO services_new (id, name, display_name, description, tags, metadata, created_at, updated_at, deleted_at)
    SELECT id, name, display_name, description, tags, metadata, created_at, updated_at, deleted_at FROM services;

DROP TABLE services;
ALTER TABLE services_new RENAME TO services;

CREATE UNIQUE INDEX idx_services_name ON services(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_services_display_name ON services(display_name) WHERE deleted_at IS NULL;

-- Down

-- Requires names to be unique across all services, including deleted ones.
CREATE UNIQUE INDEX services_name_key ON services(name);
DROP INDEX idx_services_name;
CREATE INDEX idx_services_name ON services(name) WHERE deleted_at IS NULL;
