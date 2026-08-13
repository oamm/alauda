-- Migration: 008_create_health_results
-- Description: Create health_results table
-- Up

CREATE TABLE IF NOT EXISTS health_results (
    id TEXT PRIMARY KEY,
    health_check_id TEXT NOT NULL,
    instance_id TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    latency_ms INTEGER,
    status_code INTEGER,
    error_type TEXT,
    error_message TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    expires_at TEXT,

    FOREIGN KEY (health_check_id) REFERENCES health_checks(id),
    FOREIGN KEY (instance_id) REFERENCES service_instances(id)
);

CREATE INDEX IF NOT EXISTS idx_health_results_check ON health_results(health_check_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_health_results_instance ON health_results(instance_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_health_results_expires ON health_results(expires_at) WHERE expires_at IS NOT NULL;

-- Down

DROP TABLE IF EXISTS health_results;
