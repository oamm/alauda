-- Migration: 009_create_incidents
-- Description: Create incidents table
-- Up

CREATE TABLE IF NOT EXISTS incidents (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    deployment_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    service_id TEXT NOT NULL,
    state TEXT NOT NULL,
    opened_at TEXT NOT NULL,
    resolved_at TEXT,
    duration_seconds INTEGER,
    reason TEXT NOT NULL,
    impact_summary TEXT,
    tags TEXT NOT NULL DEFAULT '{}',
    metadata TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,

    FOREIGN KEY (instance_id) REFERENCES service_instances(id),
    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    FOREIGN KEY (environment_id) REFERENCES environments(id),
    FOREIGN KEY (service_id) REFERENCES services(id)
);

CREATE INDEX IF NOT EXISTS idx_incidents_state ON incidents(state) WHERE state = 'OPEN';
CREATE INDEX IF NOT EXISTS idx_incidents_instance ON incidents(instance_id);
CREATE INDEX IF NOT EXISTS idx_incidents_deployment ON incidents(deployment_id);
CREATE INDEX IF NOT EXISTS idx_incidents_environment ON incidents(environment_id);
CREATE INDEX IF NOT EXISTS idx_incidents_service ON incidents(service_id);
CREATE INDEX IF NOT EXISTS idx_incidents_opened ON incidents(opened_at DESC);

-- Down

DROP TABLE IF EXISTS incidents;
