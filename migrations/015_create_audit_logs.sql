-- Migration: 015_create_audit_logs
-- Description: Create audit_logs table
-- Up

CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    timestamp TEXT NOT NULL,
    actor TEXT NOT NULL,
    actor_id TEXT,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    environment_id TEXT,
    changes TEXT,
    change_description TEXT,
    ip TEXT,
    user_agent TEXT,
    status TEXT NOT NULL,
    error_message TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',

    FOREIGN KEY (environment_id) REFERENCES environments(id)
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_actor ON audit_logs(actor);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);

-- Down

DROP TABLE IF EXISTS audit_logs;
