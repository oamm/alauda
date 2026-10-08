-- Migration: 024_incident_resolution_semantics
-- Description: Preserve explicit incident resolution method, note and evidence
-- Up

ALTER TABLE incidents ADD COLUMN resolution_method TEXT NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN resolution_note TEXT NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN resolution_evidence_health_result_id TEXT NOT NULL DEFAULT '';
ALTER TABLE incidents ADD COLUMN resolved_by TEXT NOT NULL DEFAULT '';

-- Down

-- SQLite does not support dropping columns on all supported versions.
