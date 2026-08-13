-- Migration: 013_create_alert_attempts
-- Description: Create alert_attempts table
-- Up

CREATE TABLE IF NOT EXISTS alert_attempts (
    id TEXT PRIMARY KEY,
    incident_id TEXT NOT NULL,
    policy_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    notification_type TEXT NOT NULL,
    attempted_at TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    status_code INTEGER,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',

    FOREIGN KEY (incident_id) REFERENCES incidents(id),
    FOREIGN KEY (policy_id) REFERENCES alert_policies(id),
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id)
);

CREATE INDEX IF NOT EXISTS idx_alert_attempts_incident ON alert_attempts(incident_id);
CREATE INDEX IF NOT EXISTS idx_alert_attempts_channel ON alert_attempts(channel_id);
CREATE INDEX IF NOT EXISTS idx_alert_attempts_success ON alert_attempts(success);
CREATE INDEX IF NOT EXISTS idx_alert_attempts_retry ON alert_attempts(next_retry_at) WHERE next_retry_at IS NOT NULL;

-- Down

DROP TABLE IF EXISTS alert_attempts;
