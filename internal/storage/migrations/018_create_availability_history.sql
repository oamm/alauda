-- Migration: 018_create_availability_history
-- Description: Create availability history table
-- Up

CREATE TABLE IF NOT EXISTS availability_history (
    id TEXT PRIMARY KEY,
    environment_id TEXT,
    service_id TEXT,
    deployment_id TEXT,
    instance_id TEXT,
    window_hours INTEGER NOT NULL,
    window_start TEXT NOT NULL,
    window_end TEXT NOT NULL,
    availability_percent REAL NOT NULL,
    downtime_seconds INTEGER NOT NULL,
    incident_count INTEGER NOT NULL,
    calculated_at TEXT NOT NULL,
    metadata TEXT NOT NULL DEFAULT '{}',

    FOREIGN KEY (environment_id) REFERENCES environments(id),
    FOREIGN KEY (service_id) REFERENCES services(id),
    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    FOREIGN KEY (instance_id) REFERENCES service_instances(id)
);

CREATE INDEX IF NOT EXISTS idx_availability_history_scope ON availability_history(environment_id, service_id, deployment_id, instance_id);
CREATE INDEX IF NOT EXISTS idx_availability_history_window ON availability_history(window_hours, window_end DESC);

-- Down

DROP TABLE IF EXISTS availability_history;
