-- Migration: 017_create_api_tokens
-- Description: Create api_tokens table
-- Up

CREATE TABLE IF NOT EXISTS api_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    scopes TEXT NOT NULL,
    environment_ids TEXT,
    expires_at TEXT,
    last_used_at TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_api_tokens_user ON api_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_api_tokens_enabled ON api_tokens(enabled);
CREATE INDEX IF NOT EXISTS idx_api_tokens_expires ON api_tokens(expires_at) WHERE expires_at IS NOT NULL;

-- Down

DROP TABLE IF EXISTS api_tokens;
