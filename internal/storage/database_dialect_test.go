package storage

import (
	"strings"
	"testing"
)

func TestPostgresDialectAdaptsSQLiteQueries(t *testing.T) {
	query := dialectQuery(`SELECT id FROM incidents WHERE julianday(opened_at) >= julianday(?) AND enabled = 1 ORDER BY opened_at DESC`, "postgres")
	for _, expected := range []string{"opened_at::timestamptz >= $1::timestamptz", "enabled = TRUE", "opened_at::timestamptz DESC"} {
		if !strings.Contains(query, expected) {
			t.Errorf("translated query missing %q: %s", expected, query)
		}
	}
	query = dialectQuery(`UPDATE incidents SET duration_seconds = CAST((julianday(?) - julianday(opened_at)) * 86400 AS INTEGER), metadata = json_set(metadata, '$.resolution_reason', ?) WHERE id = ?`, "postgres")
	for _, expected := range []string{"EXTRACT(EPOCH FROM ($1::timestamptz - opened_at::timestamptz))", "jsonb_set(metadata::jsonb", "$3"} {
		if !strings.Contains(query, expected) {
			t.Errorf("translated query missing %q: %s", expected, query)
		}
	}
	query = dialectQuery(`INSERT OR IGNORE INTO alert_policy_channels (policy_id, channel_id) VALUES (?, ?)`, "postgres")
	if !strings.Contains(query, "ON CONFLICT DO NOTHING") || !strings.Contains(query, "$2") {
		t.Errorf("conflict translation failed: %s", query)
	}
}
