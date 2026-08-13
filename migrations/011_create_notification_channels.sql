-- Migration: 011_create_notification_channels
-- Description: Create notification_channels table
-- Up

CREATE TABLE IF NOT EXISTS notification_channels (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    configuration TEXT NOT NULL,
    retry_policy TEXT NOT NULL DEFAULT '{}',
    tags TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_notification_channels_type ON notification_channels(type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_notification_channels_enabled ON notification_channels(enabled) WHERE deleted_at IS NULL;

-- Down

DROP TABLE IF EXISTS notification_channels;
