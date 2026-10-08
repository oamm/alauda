package auth

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/company/service-registry/internal/storage"
	"github.com/google/uuid"
)

func TestPostgresSecurityBootstrapContract(t *testing.T) {
	databaseURL := os.Getenv("ALAUDA_TEST_POSTGRES_URL")
	if databaseURL == "" {
		t.Skip("ALAUDA_TEST_POSTGRES_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := sql.Open("pgx", databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := "auth_contract_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(ctx, `CREATE SCHEMA "`+schema+`"`); err != nil {
		t.Fatal(err)
	}
	defer admin.ExecContext(context.Background(), `DROP SCHEMA "`+schema+`" CASCADE`)
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	params := parsedURL.Query()
	params.Set("search_path", schema)
	parsedURL.RawQuery = params.Encode()
	db, err := storage.NewStorage(ctx, "postgres", parsedURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	credential := filepath.Join(t.TempDir(), "bootstrap")
	repo := NewRepository(db)
	first, err := repo.Bootstrap(ctx, "root", "root@example.test", credential)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Created {
		t.Fatalf("first bootstrap did not create administrator: %+v", first)
	}
	second, err := repo.Bootstrap(ctx, "root", "root@example.test", credential)
	if err != nil {
		t.Fatal(err)
	}
	if second.Created {
		t.Fatalf("second bootstrap unexpectedly created administrator: %+v", second)
	}
}
