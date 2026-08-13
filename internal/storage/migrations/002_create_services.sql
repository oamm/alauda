-- Migration: 002_create_services
-- Description: Create services table
-- Up

CREATE TABLE IF NOT EXISTS services (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT,
    tags TEXT NOT NULL DEFAULT '{}',
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_services_name ON services(name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_services_display_name ON services(display_name) WHERE deleted_at IS NULL;

-- Down

DROP TABLE IF EXISTS services;
