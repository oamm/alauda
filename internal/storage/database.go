package storage

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/company/service-registry/internal/config"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations
var migrations embed.FS

type Database struct {
	db        *sql.DB
	provider  string
	runtimeMu sync.Mutex
}

// Tx keeps dialect details inside storage while preserving database/sql transaction behavior.
type Tx struct {
	tx       *sql.Tx
	provider string
}

func NewDatabase(ctx context.Context, dbPath string) (*Database, error) {
	return NewStorage(ctx, "sqlite", dbPath)
}

func Open(ctx context.Context, cfg config.StorageConfig) (*Database, error) {
	return NewStorage(ctx, cfg.Provider, cfg.Connection)
}

func NewStorage(ctx context.Context, provider, connection string) (*Database, error) {
	var driver, dsn string
	switch provider {
	case "postgres":
		driver, dsn = "pgx", connection
	case "sqlite":
		driver, dsn = "sqlite3", sqliteDSN(connection)
		if connection != ":memory:" && !strings.HasPrefix(connection, "file:") {
			if dir := filepath.Dir(connection); dir != "." {
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return nil, fmt.Errorf("create SQLite data directory: %w", err)
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported storage provider %q; supported values: postgres, sqlite", provider)
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", provider, err)
	}
	if provider == "sqlite" {
		// A small pool allows callers to read while an iterator remains open; WAL and
		// busy_timeout serialize the few concurrent writes without pool starvation.
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(4)
		for _, statement := range []string{"PRAGMA foreign_keys = ON", "PRAGMA journal_mode = WAL", "PRAGMA busy_timeout = 5000"} {
			if _, err = db.ExecContext(ctx, statement); err != nil {
				db.Close()
				return nil, fmt.Errorf("configure SQLite: %w", err)
			}
		}
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping %s database: %w", provider, err)
	}
	return &Database{db: db, provider: provider}, nil
}

func sqliteDSN(path string) string {
	if path == ":memory:" {
		return "file:alauda-" + uuid.NewString() + "?mode=memory&cache=shared&_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"
	}
	if strings.Contains(path, "?") {
		return path + "&_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"
	}
	return path + "?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"
}

func (d *Database) Provider() string { return d.provider }
func (d *Database) Close() error     { return d.db.Close() }
func (d *Database) GetDB() *sql.DB   { return d.db }
func (d *Database) Ready(ctx context.Context) error {
	if err := d.db.PingContext(ctx); err != nil {
		return err
	}
	if d.provider == "sqlite" {
		var value int
		return d.db.QueryRowContext(ctx, "SELECT 1").Scan(&value)
	}
	return nil
}

func (d *Database) RunMigrations(ctx context.Context) error { return RunMigrations(ctx, d) }

func RunMigrations(ctx context.Context, d *Database) error {
	create := `CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`
	if d.provider == "postgres" {
		create = `CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`
	}
	if _, err := d.db.ExecContext(ctx, create); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		var applied bool
		if err := d.QueryRow(ctx, "SELECT COUNT(*) > 0 FROM migrations WHERE name = ?", entry.Name()).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if applied {
			continue
		}
		migrationPath := "migrations/" + entry.Name()
		if d.provider == "postgres" {
			providerPath := "migrations/postgres/" + entry.Name()
			if _, statErr := migrations.ReadFile(providerPath); statErr == nil {
				migrationPath = providerPath
			}
		}
		content, err := migrations.ReadFile(migrationPath)
		if err != nil {
			return err
		}
		parts := strings.SplitN(string(content), "-- Up", 2)
		if len(parts) < 2 {
			continue
		}
		up := strings.TrimSpace(strings.SplitN(parts[1], "-- Down", 2)[0])
		rebuild := d.provider == "sqlite" && strings.Contains(parts[0], "-- SQLite: rebuild table")
		if err := d.applyMigration(ctx, entry.Name(), up, rebuild); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		slog.Info("Migration completed", slog.String("name", entry.Name()), slog.String("provider", d.provider))
	}
	return nil
}

func (d *Database) applyMigration(ctx context.Context, name, up string, rebuild bool) (err error) {
	begin := d.db.BeginTx
	if rebuild {
		// SQLite requires changing foreign key enforcement outside the rebuild transaction.
		conn, connErr := d.db.Conn(ctx)
		if connErr != nil {
			return connErr
		}
		defer conn.Close()
		defer func() {
			if _, restoreErr := conn.ExecContext(context.WithoutCancel(ctx), "PRAGMA foreign_keys = ON"); restoreErr != nil {
				_ = conn.Raw(func(any) error { return driver.ErrBadConn })
				err = errors.Join(err, fmt.Errorf("restore SQLite foreign key enforcement: %w", restoreErr))
			}
		}()
		if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
			return err
		}
		begin = conn.BeginTx
	}

	sqlTx, err := begin(ctx, nil)
	if err != nil {
		return err
	}
	tx := &Tx{tx: sqlTx, provider: d.provider}
	defer tx.Rollback()
	if d.provider == "sqlite" && name == "025_endpoint_kind.sql" {
		if err := migrateSQLiteEndpointKind(ctx, tx); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, up); err != nil {
		return err
	}
	if rebuild {
		var invalid bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pragma_foreign_key_check)").Scan(&invalid); err != nil {
			return err
		}
		if invalid {
			return errors.New("SQLite foreign key check failed after table rebuild")
		}
	}
	if _, err := tx.ExecContext(ctx, "INSERT INTO migrations (name, applied_at) VALUES (?, ?)", name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	return tx.Commit()
}

func migrateSQLiteEndpointKind(ctx context.Context, tx *Tx) error {
	var legacy bool
	if err := tx.QueryRowContext(ctx, "SELECT EXISTS (SELECT 1 FROM pragma_table_info('endpoints') WHERE name = 'protocol')").Scan(&legacy); err != nil || !legacy {
		return err
	}
	// Older SQLite versions of Alauda stored protocol numbers in a TEXT column.
	_, err := tx.ExecContext(ctx, `
		DROP INDEX IF EXISTS idx_endpoints_protocol;
		ALTER TABLE endpoints ADD COLUMN kind INTEGER NOT NULL DEFAULT 0;
		UPDATE endpoints SET kind = CAST(protocol AS INTEGER);
		ALTER TABLE endpoints DROP COLUMN protocol;
	`)
	return err
}

func (d *Database) QueryRow(ctx context.Context, q string, args ...any) *sql.Row {
	return d.db.QueryRowContext(ctx, d.query(q), args...)
}
func (d *Database) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, d.query(q), args...)
}
func (d *Database) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, d.query(q), args...)
}
func (d *Database) Exec(ctx context.Context, q string, args ...any) (sql.Result, error) {
	return d.db.ExecContext(ctx, d.query(q), args...)
}
func (d *Database) BeginTx(ctx context.Context) (*Tx, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx, provider: d.provider}, nil
}

// LockRegistrationScope serializes read-then-write registrations for the same deployment.
func (d *Database) LockRegistrationScope(ctx context.Context, tx *Tx, deploymentID string) error {
	if d.provider != "postgres" {
		return nil
	}
	var id string
	return tx.QueryRowContext(ctx, "SELECT id FROM service_deployments WHERE id = ? FOR UPDATE", deploymentID).Scan(&id)
}

func (d *Database) AcquireRegistrationLock() func() {
	if d.provider != "sqlite" {
		return func() {}
	}
	d.runtimeMu.Lock()
	return d.runtimeMu.Unlock
}

func (d *Database) query(q string) string { return dialectQuery(q, d.provider) }
func (t *Tx) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	return t.tx.QueryRowContext(ctx, dialectQuery(q, t.provider), args...)
}
func (t *Tx) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return t.tx.QueryContext(ctx, dialectQuery(q, t.provider), args...)
}
func (t *Tx) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	return t.tx.ExecContext(ctx, dialectQuery(q, t.provider), args...)
}
func (t *Tx) Commit() error   { return t.tx.Commit() }
func (t *Tx) Rollback() error { return t.tx.Rollback() }

type transaction interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func dialectQuery(query, provider string) string {
	if provider != "postgres" {
		return query
	}
	query = strings.ReplaceAll(query, "julianday(latest.last_timestamp) <= julianday('now') - (CAST(hc.interval_seconds AS REAL) / 86400.0)", "latest.last_timestamp::timestamptz <= now() - (hc.interval_seconds * INTERVAL '1 second')")
	query = strings.ReplaceAll(query, "CAST((julianday(?) - julianday(opened_at)) * 86400 AS INTEGER)", "CAST(EXTRACT(EPOCH FROM (?::timestamptz - opened_at::timestamptz)) AS INTEGER)")
	query = strings.ReplaceAll(query, "julianday(COALESCE(resolved_at, ?))", "COALESCE(resolved_at::timestamptz, ?::timestamptz)")
	query = strings.ReplaceAll(query, "julianday(COALESCE(latest.last_timestamp, hc.created_at))", "COALESCE(latest.last_timestamp::timestamptz, hc.created_at::timestamptz)")
	ignoreConflicts := strings.Contains(query, "INSERT OR IGNORE INTO")
	query = strings.ReplaceAll(query, "INSERT OR IGNORE INTO", "INSERT INTO")
	if ignoreConflicts {
		query += " ON CONFLICT DO NOTHING"
	}
	query = strings.ReplaceAll(query, " COLLATE NOCASE", "")
	query = strings.ReplaceAll(query, "julianday(hr.timestamp)", "hr.timestamp::timestamptz")
	query = strings.ReplaceAll(query, "julianday(latest.timestamp)", "latest.timestamp::timestamptz")
	query = strings.ReplaceAll(query, "julianday(?)", "?::timestamptz")
	query = strings.ReplaceAll(query, "julianday('now')", "now()")
	query = regexp.MustCompile(`julianday\(([a-zA-Z_][a-zA-Z0-9_.]*)\)`).ReplaceAllString(query, "$1::timestamptz")
	query = regexp.MustCompile(`ORDER BY ((?:[a-zA-Z_][a-zA-Z0-9_]*\.)?(?:timestamp|[a-zA-Z_]+_at))(\s+(?:ASC|DESC))`).ReplaceAllString(query, "ORDER BY $1::timestamptz$2")
	// PostgreSQL requires ORDER BY expressions of a SELECT DISTINCT query to
	// appear in its projection. created_at is already selected, so sort by the
	// stored timestamp text as SQLite does for this service-list query.
	if strings.Contains(query, "SELECT DISTINCT s.id, s.name") {
		query = strings.ReplaceAll(query, "ORDER BY s.created_at::timestamptz", "ORDER BY s.created_at")
	}
	query = strings.ReplaceAll(query, "COALESCE(json_extract(hc.metadata, '$.path'), '')", "COALESCE(hc.metadata::jsonb ->> 'path', '')")
	query = strings.ReplaceAll(query, "COALESCE(json_extract(hc.metadata, '$.expectedStatus'), '')", "COALESCE(hc.metadata::jsonb ->> 'expectedStatus', '')")
	query = strings.ReplaceAll(query, "metadata = json_set(metadata, '$.resolution_reason', ?)", "metadata = jsonb_set(metadata::jsonb, '{resolution_reason}', to_jsonb(?::text))::text")
	query = regexp.MustCompile(`\b(enabled|health_enabled|alerts_enabled|success|primary_endpoint|must_change_password)\s*=\s*1\b`).ReplaceAllString(query, "$1 = TRUE")
	query = regexp.MustCompile(`\b(enabled|health_enabled|alerts_enabled|success|primary_endpoint|must_change_password)\s*=\s*0\b`).ReplaceAllString(query, "$1 = FALSE")
	var out strings.Builder
	quoted := byte(0)
	n := 0
	for i := 0; i < len(query); i++ {
		c := query[i]
		if quoted != 0 {
			out.WriteByte(c)
			if c == quoted {
				quoted = 0
			}
			continue
		}
		if c == '\'' || c == '"' || c == '`' {
			quoted = c
			out.WriteByte(c)
			continue
		}
		if c == '?' {
			n++
			fmt.Fprintf(&out, "$%d", n)
		} else {
			out.WriteByte(c)
		}
	}
	return out.String()
}
