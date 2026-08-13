package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/company/service-registry/internal/storage"
)

func TestRegistryResolveServiceReturnsRegisteredEndpoints(t *testing.T) {
	ctx := context.Background()
	db, server := setupRegistryTestServer(t, ctx)
	defer db.Close()
	defer server.Close()

	serviceID, deploymentID := seedRegistryEndpoint(t, ctx, db)
	client := registryv1connect.NewRegistryServiceClient(http.DefaultClient, server.URL)

	resp, err := client.ResolveService(ctx, connect.NewRequest(&registryv1.ResolveServiceRequest{
		ServiceName:    "payments",
		EnvironmentKey: "prod",
	}))
	if err != nil {
		t.Fatalf("resolve service: %v", err)
	}
	if resp.Msg.GetServiceId() != serviceID || resp.Msg.GetDeploymentId() != deploymentID {
		t.Fatalf("resolved ids = %s/%s, want %s/%s", resp.Msg.GetServiceId(), resp.Msg.GetDeploymentId(), serviceID, deploymentID)
	}
	endpoints := resp.Msg.GetEndpoints()
	if len(endpoints) != 1 {
		t.Fatalf("endpoints = %d, want 1", len(endpoints))
	}
	got := endpoints[0]
	if got.GetHost() != "10.1.0.12" || got.GetPort() != 8443 || got.GetPath() != "/readyz" || got.GetProtocol() != registryv1.Protocol_PROTOCOL_HTTPS {
		t.Fatalf("endpoint = %+v, want 10.1.0.12:8443/readyz HTTPS", got)
	}
}

func TestRegistryResolveEndpointFiltersByEndpointName(t *testing.T) {
	ctx := context.Background()
	db, server := setupRegistryTestServer(t, ctx)
	defer db.Close()
	defer server.Close()

	seedRegistryEndpoint(t, ctx, db)
	client := registryv1connect.NewRegistryServiceClient(http.DefaultClient, server.URL)

	resp, err := client.ResolveEndpoint(ctx, connect.NewRequest(&registryv1.ResolveEndpointRequest{
		ServiceName:    "payments",
		EndpointName:   "public",
		EnvironmentKey: "prod",
	}))
	if err != nil {
		t.Fatalf("resolve endpoint: %v", err)
	}
	if len(resp.Msg.GetEndpoints()) != 1 {
		t.Fatalf("endpoints = %d, want 1", len(resp.Msg.GetEndpoints()))
	}
	if resp.Msg.GetEndpoints()[0].GetPort() != 8443 {
		t.Fatalf("port = %d, want 8443", resp.Msg.GetEndpoints()[0].GetPort())
	}
}

func setupRegistryTestServer(t *testing.T, ctx context.Context) (*storage.Database, *httptest.Server) {
	t.Helper()
	db, err := storage.NewDatabase(ctx, filepath.Join(t.TempDir(), "registry-test.db"))
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	return db, httptest.NewServer(mux)
}

func seedRegistryEndpoint(t *testing.T, ctx context.Context, db *storage.Database) (string, string) {
	t.Helper()
	env, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{
		Key:  "prod",
		Name: "Production",
		Tier: "prod",
	})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	svc, err := storage.NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{
		Name:        "payments",
		DisplayName: "Payments",
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	deployment, err := storage.NewDeploymentRepository(db).Create(ctx, &registryv1.CreateDeploymentRequest{
		ServiceId:            svc.GetId(),
		EnvironmentId:        env.GetId(),
		HealthEnabled:        true,
		AlertsEnabled:        true,
		AlertCooldownMinutes: 10,
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	instance, err := storage.NewInstanceRepository(db).Create(ctx, &registryv1.CreateInstanceRequest{
		DeploymentId: deployment.GetId(),
		Name:         "payments-1",
		Address:      "10.1.0.12",
		Port:         8080,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if _, err := storage.NewEndpointRepository(db).Create(ctx, &registryv1.CreateEndpointRequest{
		InstanceId: instance.GetId(),
		Name:       "public",
		Protocol:   registryv1.Protocol_PROTOCOL_HTTPS,
		Port:       8443,
		Path:       "/readyz",
		Enabled:    true,
	}); err != nil {
		t.Fatalf("create endpoint: %v", err)
	}
	return svc.GetId(), deployment.GetId()
}
