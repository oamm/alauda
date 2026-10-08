-- Migration: 023_health_results_chronology
-- PostgreSQL stores the shared timestamp representation as RFC3339 text.
-- The textual index supports the provider-neutral chronology query.
-- Up

CREATE INDEX IF NOT EXISTS idx_health_results_chronology ON health_results(timestamp, id);

-- Down
DROP INDEX IF EXISTS idx_health_results_chronology;
