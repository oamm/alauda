-- Migration: 014_create_events
-- Description: Create events table
-- Up

CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    environment_id TEXT,
    service_id TEXT,
    deployment_id TEXT,
    instance_id TEXT,
    actor TEXT NOT NULL,
    actor_id TEXT,
    message TEXT NOT NULL,
    changes TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    expires_at TEXT,

    FOREIGN KEY (environment_id) REFERENCES environments(id),
    FOREIGN KEY (service_id) REFERENCES services(id),
    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    FOREIGN KEY (instance_id) REFERENCES service_instances(id)
);

CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
CREATE INDEX IF NOT EXISTS idx_events_resource ON events(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_events_environment ON events(environment_id);
CREATE INDEX IF NOT EXISTS idx_events_service ON events(service_id);
CREATE INDEX IF NOT EXISTS idx_events_expires ON events(expires_at) WHERE expires_at IS NOT NULL;

-- Down

DROP TABLE IF EXISTS events;
