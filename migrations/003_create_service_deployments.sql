-- Migration: 003_create_service_deployments
-- Description: Create service_deployments table
-- Up

CREATE TABLE IF NOT EXISTS service_deployments (
    id TEXT PRIMARY KEY,
    service_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    health_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    alerts_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    alert_cooldown_minutes INTEGER NOT NULL DEFAULT 10,
    tags TEXT NOT NULL DEFAULT '{}',
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,

    FOREIGN KEY (service_id) REFERENCES services(id),
    FOREIGN KEY (environment_id) REFERENCES environments(id),
    UNIQUE(service_id, environment_id)
);

CREATE INDEX IF NOT EXISTS idx_service_deployments_service ON service_deployments(service_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_deployments_environment ON service_deployments(environment_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_service_deployments_health ON service_deployments(health_enabled, deleted_at);

-- Down

DROP TABLE IF EXISTS service_deployments;
