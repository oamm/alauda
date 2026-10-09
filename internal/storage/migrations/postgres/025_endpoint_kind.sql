-- Migration: 025_endpoint_kind
-- Description: Endpoint kind cleanup for clean-break schema.
-- Up

DROP INDEX IF EXISTS idx_endpoints_protocol;
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'endpoints'
          AND column_name = 'protocol'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = current_schema()
          AND table_name = 'endpoints'
          AND column_name = 'kind'
    ) THEN
        ALTER TABLE endpoints RENAME COLUMN protocol TO kind;
    END IF;
END $$;
CREATE INDEX IF NOT EXISTS idx_endpoints_kind ON endpoints(kind) WHERE deleted_at IS NULL;
UPDATE endpoints SET path = NULL WHERE kind NOT IN (1, 2);

-- Down

DROP INDEX IF EXISTS idx_endpoints_kind;
