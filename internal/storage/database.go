package storage

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Database wraps the SQL database connection
type Database struct {
	db *sql.DB
}

// NewDatabase creates a new database connection
func NewDatabase(ctx context.Context, dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL: %w", err)
	}

	// Set busy timeout
	if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	// Test connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Database{db: db}, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// GetDB returns the underlying database connection
func (d *Database) GetDB() *sql.DB {
	return d.db
}

// RunMigrations runs all pending migrations
func RunMigrations(ctx context.Context, d *Database) error {
	// Create migrations table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS migrations (
		name TEXT PRIMARY KEY,
		applied_at TEXT NOT NULL
	)
	`

	if _, err := d.db.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Read migration files
	entries, err := migrations.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrationName := entry.Name()

			// Check if already applied
			var applied bool
			err := d.db.QueryRowContext(ctx,
				"SELECT COUNT(*) > 0 FROM migrations WHERE name = ?",
				migrationName,
			).Scan(&applied)
			if err != nil {
				return fmt.Errorf("failed to check migration status: %w", err)
			}

			if applied {
				slog.Debug("Migration already applied", slog.String("name", migrationName))
				continue
			}

			// Read and execute migration
			content, err := migrations.ReadFile("migrations/" + migrationName)
			if err != nil {
				return fmt.Errorf("failed to read migration file %s: %w", migrationName, err)
			}

			// Parse migration SQL (Up section only)
			parts := strings.Split(string(content), "-- Up")
			if len(parts) < 2 {
				slog.Warn("Migration has no Up section", slog.String("name", migrationName))
				continue
			}

			upSQL := parts[1]
			downParts := strings.Split(upSQL, "-- Down")
			if len(downParts) > 0 {
				upSQL = downParts[0]
			}
			upSQL = strings.TrimSpace(upSQL)

			// Execute migration
			slog.Info("Applying migration", slog.String("name", migrationName))
			if _, err := d.db.ExecContext(ctx, upSQL); err != nil {
				return fmt.Errorf("failed to apply migration %s: %w", migrationName, err)
			}

			// Record migration
			insertSQL := "INSERT INTO migrations (name, applied_at) VALUES (?, datetime('now'))"
			if _, err := d.db.ExecContext(ctx, insertSQL, migrationName); err != nil {
				return fmt.Errorf("failed to record migration %s: %w", migrationName, err)
			}

			slog.Info("Migration completed", slog.String("name", migrationName))
		}
	}

	return nil
}

// QueryRow executes a query that returns a single row
func (d *Database) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return d.db.QueryRowContext(ctx, query, args...)
}

// Query executes a query that returns multiple rows
func (d *Database) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return d.db.QueryContext(ctx, query, args...)
}

// Exec executes a query without returning rows
func (d *Database) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return d.db.ExecContext(ctx, query, args...)
}

// BeginTx begins a transaction
func (d *Database) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return d.db.BeginTx(ctx, nil)
}
