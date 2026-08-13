-- Migration: 006_create_health_checks
-- Description: Create health_checks table
-- Up

CREATE TABLE IF NOT EXISTS health_checks (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    endpoint_id TEXT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    interval_seconds INTEGER NOT NULL,
    timeout_seconds INTEGER NOT NULL,
    failures_before_unhealthy INTEGER NOT NULL DEFAULT 3,
    successes_before_healthy INTEGER NOT NULL DEFAULT 2,
    description TEXT,
    tags TEXT NOT NULL DEFAULT '{}',
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,

    FOREIGN KEY (instance_id) REFERENCES service_instances(id) ON DELETE CASCADE,
    FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE SET NULL,
    UNIQUE(instance_id, name)
);

CREATE INDEX IF NOT EXISTS idx_health_checks_instance ON health_checks(instance_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_health_checks_enabled ON health_checks(enabled) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_health_checks_type ON health_checks(type) WHERE deleted_at IS NULL;

-- Down

DROP TABLE IF EXISTS health_checks;
