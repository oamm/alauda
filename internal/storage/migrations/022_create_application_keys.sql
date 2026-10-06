-- Up
CREATE TABLE IF NOT EXISTS application_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    secret_hash TEXT NOT NULL UNIQUE,
    scopes TEXT NOT NULL,
    environment_ids TEXT,
    expires_at TEXT,
    last_used_at TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_application_keys_enabled ON application_keys(enabled);
CREATE INDEX IF NOT EXISTS idx_application_keys_expires ON application_keys(expires_at);

-- Down
DROP TABLE IF EXISTS application_keys;
