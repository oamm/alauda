-- Migration: 005_create_endpoints
-- PostgreSQL stores the generated protocol enum number as INTEGER.
-- Up

CREATE TABLE IF NOT EXISTS endpoints (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    name TEXT NOT NULL,
    protocol INTEGER NOT NULL,
    port INTEGER NOT NULL,
    path TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tags TEXT NOT NULL DEFAULT '{}',
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,
    UNIQUE(instance_id, name),
    FOREIGN KEY (instance_id) REFERENCES service_instances(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_endpoints_instance ON endpoints(instance_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_endpoints_protocol ON endpoints(protocol) WHERE deleted_at IS NULL;

-- Down
DROP TABLE IF EXISTS endpoints;
