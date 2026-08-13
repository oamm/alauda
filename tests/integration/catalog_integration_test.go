package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/company/service-registry/internal/api"
	"github.com/company/service-registry/internal/storage"
)

func TestCatalogLifecycleOverConnect(t *testing.T) {
	ctx := context.Background()
	db, srv, cleanup := setupIntegrationServer(t)
	defer cleanup()
	defer db.Close()

	envClient := registryv1connect.NewEnvironmentServiceClient(http.DefaultClient, srv.URL)
	catalogClient := registryv1connect.NewCatalogServiceClient(http.DefaultClient, srv.URL)
	deploymentClient := registryv1connect.NewDeploymentServiceClient(http.DefaultClient, srv.URL)
	instanceClient := registryv1connect.NewInstanceServiceClient(http.DefaultClient, srv.URL)
	endpointClient := registryv1connect.NewEndpointServiceClient(http.DefaultClient, srv.URL)

	createEnvResp, err := envClient.CreateEnvironment(ctx, connect.NewRequest(&registryv1.CreateEnvironmentRequest{
		Key:         "prod",
		Name:        "Production",
		Description: "Primary production environment",
		Tier:        "prod",
		Tags:        map[string]string{"region": "us-east"},
	}))
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	envID := createEnvResp.Msg.GetEnvironment().GetId()
	if envID == "" {
		t.Fatalf("expected non-empty environment id")
	}

	createSvcResp, err := catalogClient.CreateService(ctx, connect.NewRequest(&registryv1.CreateServiceRequest{
		Name:        "payments",
		DisplayName: "Payments",
		Description: "Payment processing",
		Tags:        map[string]string{"team": "finops"},
		Metadata:    map[string]string{"owner": "platform"},
	}))
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	svcID := createSvcResp.Msg.GetService().GetId()
	if svcID == "" {
		t.Fatalf("expected non-empty service id")
	}

	createDepResp, err := deploymentClient.CreateDeployment(ctx, connect.NewRequest(&registryv1.CreateDeploymentRequest{
		ServiceId:            svcID,
		EnvironmentId:        envID,
		HealthEnabled:        true,
		AlertsEnabled:        true,
		AlertCooldownMinutes: 10,
		Tags:                 map[string]string{"rollout": "blue"},
		Metadata:             map[string]string{"cluster": "east-1"},
	}))
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	depID := createDepResp.Msg.GetDeployment().GetId()
	if depID == "" {
		t.Fatalf("expected non-empty deployment id")
	}

	createInstResp, err := instanceClient.CreateInstance(ctx, connect.NewRequest(&registryv1.CreateInstanceRequest{
		DeploymentId: depID,
		Name:         "payments-1",
		Address:      "10.1.0.12",
		Port:         8080,
		Description:  "Primary instance",
		Enabled:      true,
		Tags:         map[string]string{"zone": "a"},
		Metadata:     map[string]string{"node": "node-1"},
	}))
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	instID := createInstResp.Msg.GetInstance().GetId()
	if instID == "" {
		t.Fatalf("expected non-empty instance id")
	}

	if _, err := envClient.UpdateEnvironment(ctx, connect.NewRequest(&registryv1.UpdateEnvironmentRequest{
		Id:          envID,
		Name:        "Production-Updated",
		Description: "Updated",
		Enabled:     true,
		Tier:        "prod",
		Tags:        map[string]string{"region": "us-west"},
	})); err != nil {
		t.Fatalf("update environment: %v", err)
	}

	if _, err := catalogClient.UpdateService(ctx, connect.NewRequest(&registryv1.UpdateServiceRequest{
		Id:          svcID,
		DisplayName: "Payments-Updated",
		Description: "Updated service",
		Tags:        map[string]string{"team": "core"},
		Metadata:    map[string]string{"owner": "api"},
	})); err != nil {
		t.Fatalf("update service: %v", err)
	}

	if _, err := deploymentClient.UpdateDeployment(ctx, connect.NewRequest(&registryv1.UpdateDeploymentRequest{
		Id:                   depID,
		HealthEnabled:        true,
		AlertsEnabled:        false,
		AlertCooldownMinutes: 3,
		Tags:                 map[string]string{"rollout": "green"},
		Metadata:             map[string]string{"cluster": "east-2"},
	})); err != nil {
		t.Fatalf("update deployment: %v", err)
	}

	if _, err := instanceClient.UpdateInstance(ctx, connect.NewRequest(&registryv1.UpdateInstanceRequest{
		Id:          instID,
		Address:     "10.1.0.13",
		Port:        9090,
		Description: "Updated instance",
		Enabled:     true,
		Tags:        map[string]string{"zone": "b"},
		Metadata:    map[string]string{"node": "node-2"},
	})); err != nil {
		t.Fatalf("update instance: %v", err)
	}

	createEndpointResp, err := endpointClient.CreateEndpoint(ctx, connect.NewRequest(&registryv1.CreateEndpointRequest{
		InstanceId: instID,
		Name:       "public-http",
		Protocol:   registryv1.Protocol_PROTOCOL_HTTP,
		Port:       9090,
		Path:       "/healthz",
		Enabled:    true,
		Tags:       map[string]string{"visibility": "public"},
		Metadata:   map[string]string{"probe": "primary"},
	}))
	if err != nil {
		t.Fatalf("create endpoint: %v", err)
	}
	endpointID := createEndpointResp.Msg.GetEndpoint().GetId()
	if endpointID == "" {
		t.Fatalf("expected non-empty endpoint id")
	}

	if _, err := endpointClient.UpdateEndpoint(ctx, connect.NewRequest(&registryv1.UpdateEndpointRequest{
		Id:       endpointID,
		Name:     "internal-http",
		Protocol: registryv1.Protocol_PROTOCOL_HTTP,
		Port:     9091,
		Path:     "/readyz",
		Enabled:  true,
		Tags:     map[string]string{"visibility": "internal"},
		Metadata: map[string]string{"probe": "secondary"},
	})); err != nil {
		t.Fatalf("update endpoint: %v", err)
	}

	listSvcResp, err := catalogClient.ListServices(ctx, connect.NewRequest(&registryv1.ListServicesRequest{
		EnvironmentId: envID,
	}))
	if err != nil {
		t.Fatalf("list services: %v", err)
	}
	if len(listSvcResp.Msg.GetServices()) == 0 {
		t.Fatalf("expected at least one listed service")
	}

	listDepResp, err := deploymentClient.ListDeployments(ctx, connect.NewRequest(&registryv1.ListDeploymentsRequest{
		ServiceId:     svcID,
		EnvironmentId: envID,
	}))
	if err != nil {
		t.Fatalf("list deployments: %v", err)
	}
	if len(listDepResp.Msg.GetDeployments()) == 0 {
		t.Fatalf("expected at least one listed deployment")
	}

	listInstResp, err := instanceClient.ListInstances(ctx, connect.NewRequest(&registryv1.ListInstancesRequest{
		DeploymentId: depID,
	}))
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(listInstResp.Msg.GetInstances()) == 0 {
		t.Fatalf("expected at least one listed instance")
	}

	listEndpointResp, err := endpointClient.ListEndpoints(ctx, connect.NewRequest(&registryv1.ListEndpointsRequest{
		InstanceId: instID,
	}))
	if err != nil {
		t.Fatalf("list endpoints: %v", err)
	}
	if len(listEndpointResp.Msg.GetEndpoints()) == 0 {
		t.Fatalf("expected at least one listed endpoint")
	}

	if _, err := endpointClient.DeleteEndpoint(ctx, connect.NewRequest(&registryv1.DeleteEndpointRequest{Id: endpointID})); err != nil {
		t.Fatalf("delete endpoint: %v", err)
	}

	if _, err := instanceClient.DeleteInstance(ctx, connect.NewRequest(&registryv1.DeleteInstanceRequest{Id: instID})); err != nil {
		t.Fatalf("delete instance: %v", err)
	}
	if _, err := deploymentClient.DeleteDeployment(ctx, connect.NewRequest(&registryv1.DeleteDeploymentRequest{Id: depID})); err != nil {
		t.Fatalf("delete deployment: %v", err)
	}
	if _, err := catalogClient.DeleteService(ctx, connect.NewRequest(&registryv1.DeleteServiceRequest{Id: svcID})); err != nil {
		t.Fatalf("delete service: %v", err)
	}
	if _, err := envClient.DeleteEnvironment(ctx, connect.NewRequest(&registryv1.DeleteEnvironmentRequest{Id: envID})); err != nil {
		t.Fatalf("delete environment: %v", err)
	}

	if _, err := instanceClient.GetInstance(ctx, connect.NewRequest(&registryv1.GetInstanceRequest{Id: instID})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for deleted instance, got: %v", err)
	}
	if _, err := deploymentClient.GetDeployment(ctx, connect.NewRequest(&registryv1.GetDeploymentRequest{Id: depID})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for deleted deployment, got: %v", err)
	}
	if _, err := catalogClient.GetService(ctx, connect.NewRequest(&registryv1.GetServiceRequest{Id: svcID})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for deleted service, got: %v", err)
	}
	if _, err := envClient.GetEnvironment(ctx, connect.NewRequest(&registryv1.GetEnvironmentRequest{Id: envID})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for deleted environment, got: %v", err)
	}
	if _, err := endpointClient.GetEndpoint(ctx, connect.NewRequest(&registryv1.GetEndpointRequest{Id: endpointID})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for deleted endpoint, got: %v", err)
	}
}

func setupIntegrationServer(t *testing.T) (*storage.Database, *httptest.Server, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "registry-test.db")
	db, err := storage.NewDatabase(context.Background(), dbPath)
	if err != nil {
		if isSQLiteCGODisabled(err) {
			t.Skipf("skipping integration test: sqlite3 cgo driver unavailable: %v", err)
		}
		t.Fatalf("create test db: %v", err)
	}
	if err := storage.RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, db)
	srv := httptest.NewServer(mux)

	cleanup := func() {
		srv.Close()
	}

	return db, srv, cleanup
}

func isSQLiteCGODisabled(err error) bool {
	msg := strings.ToLower(fmt.Sprintf("%v", err))
	return strings.Contains(msg, "cgo_enabled=0") || strings.Contains(msg, "requires cgo")
}
