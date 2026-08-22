-- Migration: 019_add_endpoint_primary
-- Description: Add primary endpoint marker and invariant
-- Up

ALTER TABLE endpoints ADD COLUMN primary_endpoint BOOLEAN NOT NULL DEFAULT FALSE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_endpoints_one_primary_per_instance
ON endpoints(instance_id)
WHERE primary_endpoint = TRUE AND deleted_at IS NULL;

-- Down

DROP INDEX IF EXISTS idx_endpoints_one_primary_per_instance;

