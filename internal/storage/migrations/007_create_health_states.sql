-- Migration: 007_create_health_states
-- Description: Create health_states table
-- Up

CREATE TABLE IF NOT EXISTS health_states (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL UNIQUE,
    current_state TEXT NOT NULL,
    consecutive_successes INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_transition_time TEXT NOT NULL,
    last_check_time TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',
    updated_at TEXT NOT NULL,

    FOREIGN KEY (instance_id) REFERENCES service_instances(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_health_states_state ON health_states(current_state);
CREATE INDEX IF NOT EXISTS idx_health_states_updated ON health_states(updated_at);

-- Down

DROP TABLE IF EXISTS health_states;
