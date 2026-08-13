-- Migration: 010_create_alert_policies
-- Description: Create alert_policies table
-- Up

CREATE TABLE IF NOT EXISTS alert_policies (
    id TEXT PRIMARY KEY,
    deployment_id TEXT,
    environment_id TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    notify_on TEXT NOT NULL DEFAULT '["unhealthy","recovered"]',
    cooldown_minutes INTEGER NOT NULL DEFAULT 10,
    send_recovery_notification BOOLEAN NOT NULL DEFAULT TRUE,
    filters TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,

    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    FOREIGN KEY (environment_id) REFERENCES environments(id)
);

CREATE INDEX IF NOT EXISTS idx_alert_policies_deployment ON alert_policies(deployment_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alert_policies_environment ON alert_policies(environment_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_alert_policies_enabled ON alert_policies(enabled) WHERE deleted_at IS NULL;

-- Down

DROP TABLE IF EXISTS alert_policies;
