-- Migration: 012_create_alert_policy_channels
-- Description: Create alert_policy_channels junction table
-- Up

CREATE TABLE IF NOT EXISTS alert_policy_channels (
    policy_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,

    FOREIGN KEY (policy_id) REFERENCES alert_policies(id) ON DELETE CASCADE,
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id),
    PRIMARY KEY (policy_id, channel_id)
);

-- Down

DROP TABLE IF EXISTS alert_policy_channels;
