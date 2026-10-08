-- Up
CREATE INDEX IF NOT EXISTS idx_health_results_chronology ON health_results(julianday(timestamp), id);

-- Down
DROP INDEX IF EXISTS idx_health_results_chronology;
