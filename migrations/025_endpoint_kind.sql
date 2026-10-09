-- Migration: 025_endpoint_kind
-- Description: Endpoint kind cleanup for clean-break schema.
-- Up

DROP INDEX IF EXISTS idx_endpoints_protocol;
CREATE INDEX IF NOT EXISTS idx_endpoints_kind ON endpoints(kind) WHERE deleted_at IS NULL;
UPDATE endpoints SET path = NULL WHERE kind NOT IN (1, 2);

-- Down

DROP INDEX IF EXISTS idx_endpoints_kind;
