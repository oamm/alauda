-- Migration: 026_reuse_service_names
-- Description: Enforce unique names only for active services.
-- Up

ALTER TABLE services DROP CONSTRAINT services_name_key;
DROP INDEX idx_services_name;
CREATE UNIQUE INDEX idx_services_name ON services(name) WHERE deleted_at IS NULL;

-- Down

-- Requires names to be unique across all services, including deleted ones.
ALTER TABLE services ADD CONSTRAINT services_name_key UNIQUE (name);
DROP INDEX idx_services_name;
CREATE INDEX idx_services_name ON services(name) WHERE deleted_at IS NULL;
