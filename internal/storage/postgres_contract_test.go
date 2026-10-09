package storage

import (
	"context"
	"database/sql"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/contract"
	"github.com/google/uuid"
)

func TestPostgresStorageContract(t *testing.T) {
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
	schema := "contract_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	db, err := NewStorage(ctx, "postgres", parsedURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	runSharedStorageContract(t, ctx, db)
}

func TestSQLiteStorageContract(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := NewDatabase(ctx, filepath.Join(t.TempDir(), "contract.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	runSharedStorageContract(t, ctx, db)
}

func runSharedStorageContract(t *testing.T, ctx context.Context, db *Database) {
	t.Helper()
	service, err := NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "contract-" + uuid.NewString(), DisplayName: "Contract", Tags: map[string]string{"provider": db.Provider()}})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := NewServiceRepository(db).Get(ctx, service.GetId())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GetName() != service.GetName() || loaded.GetTags()["provider"] != db.Provider() {
		t.Fatalf("service round trip mismatch: %#v", loaded)
	}
	services, _, err := NewServiceRepository(db).List(ctx, "", 50, "")
	if err != nil || len(services) != 1 || services[0].GetId() != service.GetId() {
		t.Fatalf("service list parity failed: %v, %v", services, err)
	}
	environment, err := NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "contract-" + uuid.NewString(), Name: "Contract"})
	if err != nil {
		t.Fatal(err)
	}
	environments, _, err := NewEnvironmentRepository(db).List(ctx, false, 50, "")
	if err != nil || len(environments) != 1 || environments[0].GetId() != environment.GetId() {
		t.Fatalf("environment list parity failed: %v, %v", environments, err)
	}
	if db.Provider() == "postgres" {
		var kindType string
		if err := db.QueryRow(ctx, `SELECT data_type FROM information_schema.columns WHERE table_schema = current_schema() AND table_name = 'endpoints' AND column_name = 'kind'`).Scan(&kindType); err != nil {
			t.Fatal(err)
		}
		if kindType != "integer" {
			t.Fatalf("endpoint kind column type = %s, want integer", kindType)
		}
	}
	registration := contract.Registration{Instance: contract.InstancePatch{Name: "node", Address: contract.Pointer("127.0.0.1")}, Endpoints: []contract.EndpointPatch{
		{Name: "http", Kind: contract.Pointer("http"), Port: contract.Pointer(int32(8080)), Path: contract.Pointer("/"), Primary: contract.Pointer(true)},
		{Name: "metrics", Kind: contract.Pointer("http"), Port: contract.Pointer(int32(9090)), Path: contract.Pointer("/metrics"), Primary: contract.Pointer(false)},
	}}
	registered, err := NewRuntimeRepository(db).RegisterPublicRuntime(ctx, service.GetId(), environment.GetId(), registration)
	if err != nil {
		t.Fatal(err)
	}
	if len(registered.GetEndpoints()) != 2 || !registered.GetEndpoints()[0].GetPrimary() {
		t.Fatalf("runtime registration did not preserve primary endpoint: %+v", registered.GetEndpoints())
	}
	registration.Endpoints[0].Primary = contract.Pointer(false)
	registration.Endpoints[1].Primary = contract.Pointer(true)
	transitioned, err := NewRuntimeRepository(db).RegisterPublicRuntime(ctx, service.GetId(), environment.GetId(), registration)
	if err != nil {
		t.Fatal(err)
	}
	if transitioned.GetInstance().GetId() != registered.GetInstance().GetId() || transitioned.GetEndpoints()[0].GetPrimary() || !transitioned.GetEndpoints()[1].GetPrimary() {
		t.Fatalf("primary endpoint transition changed identity or was not applied: %+v", transitioned)
	}
	endpoints, _, err := NewEndpointRepository(db).List(ctx, transitioned.GetInstance().GetId(), 50, "")
	if err != nil || len(endpoints) != 2 {
		t.Fatalf("endpoint list parity failed: %v, %v", endpoints, err)
	}
	again, err := NewRuntimeRepository(db).RegisterPublicRuntime(ctx, service.GetId(), environment.GetId(), registration)
	if err != nil {
		t.Fatal(err)
	}
	if again.GetInstance().GetId() != registered.GetInstance().GetId() {
		t.Fatalf("idempotent registration changed instance identity: %s => %s", registered.GetInstance().GetId(), again.GetInstance().GetId())
	}
}
