-- Migration: 004_create_service_instances
-- Description: Create service_instances table
-- Up

CREATE TABLE IF NOT EXISTS service_instances (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    port INTEGER,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tags TEXT NOT NULL DEFAULT '{}',
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_seen_at TEXT,
    deleted_at TEXT,

    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    UNIQUE(deployment_id, name)
);

CREATE INDEX IF NOT EXISTS idx_instances_deployment ON service_instances(deployment_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_instances_enabled ON service_instances(enabled) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_instances_address ON service_instances(address) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_instances_last_seen ON service_instances(last_seen_at) WHERE deleted_at IS NULL;

-- Down

DROP TABLE IF EXISTS service_instances;
