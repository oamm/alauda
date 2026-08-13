package integration_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
)

func TestHealthLifecycleOverConnect(t *testing.T) {
	ctx := context.Background()
	db, srv, cleanup := setupIntegrationServer(t)
	defer cleanup()
	defer db.Close()

	envClient := registryv1connect.NewEnvironmentServiceClient(http.DefaultClient, srv.URL)
	catalogClient := registryv1connect.NewCatalogServiceClient(http.DefaultClient, srv.URL)
	deploymentClient := registryv1connect.NewDeploymentServiceClient(http.DefaultClient, srv.URL)
	instanceClient := registryv1connect.NewInstanceServiceClient(http.DefaultClient, srv.URL)
	healthClient := registryv1connect.NewHealthServiceClient(http.DefaultClient, srv.URL)

	envResp, err := envClient.CreateEnvironment(ctx, connect.NewRequest(&registryv1.CreateEnvironmentRequest{
		Key:  "staging",
		Name: "Staging",
	}))
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	svcResp, err := catalogClient.CreateService(ctx, connect.NewRequest(&registryv1.CreateServiceRequest{
		Name:        "checkout",
		DisplayName: "Checkout",
	}))
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	depResp, err := deploymentClient.CreateDeployment(ctx, connect.NewRequest(&registryv1.CreateDeploymentRequest{
		ServiceId:            svcResp.Msg.GetService().GetId(),
		EnvironmentId:        envResp.Msg.GetEnvironment().GetId(),
		HealthEnabled:        true,
		AlertsEnabled:        true,
		AlertCooldownMinutes: 5,
	}))
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	instResp, err := instanceClient.CreateInstance(ctx, connect.NewRequest(&registryv1.CreateInstanceRequest{
		DeploymentId: depResp.Msg.GetDeployment().GetId(),
		Name:         "checkout-1",
		Address:      "10.10.0.1",
		Port:         8081,
		Enabled:      true,
	}))
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	createResp, err := healthClient.CreateHealthCheck(ctx, connect.NewRequest(&registryv1.CreateHealthCheckRequest{
		InstanceId:              instResp.Msg.GetInstance().GetId(),
		Name:                    "http-ready",
		Type:                    registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP,
		Enabled:                 true,
		IntervalSeconds:         10,
		TimeoutSeconds:          3,
		FailuresBeforeUnhealthy: 3,
		SuccessesBeforeHealthy:  2,
		Description:             "Readiness endpoint",
		Tags:                    map[string]string{"kind": "readiness"},
		Metadata:                map[string]string{"path": "/healthz"},
	}))
	if err != nil {
		t.Fatalf("create health check: %v", err)
	}
	hcID := createResp.Msg.GetHealthCheck().GetId()
	if hcID == "" {
		t.Fatalf("expected non-empty health check id")
	}

	getResp, err := healthClient.GetHealthCheck(ctx, connect.NewRequest(&registryv1.GetHealthCheckRequest{Id: hcID}))
	if err != nil {
		t.Fatalf("get health check: %v", err)
	}
	if getResp.Msg.GetHealthCheck().GetName() != "http-ready" {
		t.Fatalf("unexpected health check name: %s", getResp.Msg.GetHealthCheck().GetName())
	}

	listResp, err := healthClient.ListHealthChecks(ctx, connect.NewRequest(&registryv1.ListHealthChecksRequest{
		InstanceId: instResp.Msg.GetInstance().GetId(),
	}))
	if err != nil {
		t.Fatalf("list health checks: %v", err)
	}
	if len(listResp.Msg.GetHealthChecks()) == 0 {
		t.Fatalf("expected at least one health check")
	}

	if _, err := healthClient.UpdateHealthCheck(ctx, connect.NewRequest(&registryv1.UpdateHealthCheckRequest{
		Id:                      hcID,
		Enabled:                 true,
		IntervalSeconds:         15,
		TimeoutSeconds:          4,
		FailuresBeforeUnhealthy: 4,
		SuccessesBeforeHealthy:  1,
		Description:             "Updated readiness endpoint",
	})); err != nil {
		t.Fatalf("update health check: %v", err)
	}

	resultsResp, err := healthClient.ListHealthResults(ctx, connect.NewRequest(&registryv1.ListHealthResultsRequest{
		HealthCheckId: hcID,
	}))
	if err != nil {
		t.Fatalf("list health results: %v", err)
	}
	if len(resultsResp.Msg.GetResults()) != 0 {
		t.Fatalf("expected no health results initially")
	}

	if _, err := healthClient.GetInstanceHealthState(ctx, connect.NewRequest(&registryv1.GetInstanceHealthStateRequest{
		InstanceId: instResp.Msg.GetInstance().GetId(),
	})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for initial health state, got: %v", err)
	}

	if _, err := healthClient.DeleteHealthCheck(ctx, connect.NewRequest(&registryv1.DeleteHealthCheckRequest{Id: hcID})); err != nil {
		t.Fatalf("delete health check: %v", err)
	}
	if _, err := healthClient.GetHealthCheck(ctx, connect.NewRequest(&registryv1.GetHealthCheckRequest{Id: hcID})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for deleted health check, got: %v", err)
	}
}

func TestRunHealthCheckOverConnect(t *testing.T) {
	ctx := context.Background()
	db, srv, cleanup := setupIntegrationServer(t)
	defer cleanup()
	defer db.Close()

	probe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/readyz" {
			t.Fatalf("unexpected probe path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer probe.Close()
	probeHost, probePort := splitServerURL(t, probe.URL)

	envClient := registryv1connect.NewEnvironmentServiceClient(http.DefaultClient, srv.URL)
	catalogClient := registryv1connect.NewCatalogServiceClient(http.DefaultClient, srv.URL)
	deploymentClient := registryv1connect.NewDeploymentServiceClient(http.DefaultClient, srv.URL)
	instanceClient := registryv1connect.NewInstanceServiceClient(http.DefaultClient, srv.URL)
	healthClient := registryv1connect.NewHealthServiceClient(http.DefaultClient, srv.URL)

	envResp, err := envClient.CreateEnvironment(ctx, connect.NewRequest(&registryv1.CreateEnvironmentRequest{
		Key:  "run",
		Name: "Run",
	}))
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	svcResp, err := catalogClient.CreateService(ctx, connect.NewRequest(&registryv1.CreateServiceRequest{
		Name:        "runner",
		DisplayName: "Runner",
	}))
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	depResp, err := deploymentClient.CreateDeployment(ctx, connect.NewRequest(&registryv1.CreateDeploymentRequest{
		ServiceId:     svcResp.Msg.GetService().GetId(),
		EnvironmentId: envResp.Msg.GetEnvironment().GetId(),
	}))
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	instResp, err := instanceClient.CreateInstance(ctx, connect.NewRequest(&registryv1.CreateInstanceRequest{
		DeploymentId: depResp.Msg.GetDeployment().GetId(),
		Name:         "runner-1",
		Address:      probeHost,
		Port:         probePort,
		Enabled:      true,
	}))
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	createResp, err := healthClient.CreateHealthCheck(ctx, connect.NewRequest(&registryv1.CreateHealthCheckRequest{
		InstanceId:              instResp.Msg.GetInstance().GetId(),
		Name:                    "manual-ready",
		Type:                    registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP,
		Enabled:                 true,
		IntervalSeconds:         60,
		TimeoutSeconds:          2,
		FailuresBeforeUnhealthy: 1,
		SuccessesBeforeHealthy:  1,
		Metadata: map[string]string{
			"path":           "/readyz",
			"expectedStatus": "204",
		},
	}))
	if err != nil {
		t.Fatalf("create health check: %v", err)
	}

	runResp, err := healthClient.RunHealthCheck(ctx, connect.NewRequest(&registryv1.RunHealthCheckRequest{
		Id: createResp.Msg.GetHealthCheck().GetId(),
	}))
	if err != nil {
		t.Fatalf("run health check: %v", err)
	}
	if !runResp.Msg.GetResult().GetSuccess() {
		t.Fatalf("expected successful result, got %#v", runResp.Msg.GetResult())
	}
	if runResp.Msg.GetState().GetCurrentState() != registryv1.HealthState_HEALTH_STATE_HEALTHY {
		t.Fatalf("state = %s, want HEALTHY", runResp.Msg.GetState().GetCurrentState())
	}

	resultsResp, err := healthClient.ListHealthResults(ctx, connect.NewRequest(&registryv1.ListHealthResultsRequest{
		HealthCheckId: createResp.Msg.GetHealthCheck().GetId(),
	}))
	if err != nil {
		t.Fatalf("list health results: %v", err)
	}
	if len(resultsResp.Msg.GetResults()) != 1 {
		t.Fatalf("health results = %d, want 1", len(resultsResp.Msg.GetResults()))
	}
}

func splitServerURL(t *testing.T, rawURL string) (string, int32) {
	t.Helper()

	hostPort := rawURL
	if len(hostPort) > len("http://") && hostPort[:len("http://")] == "http://" {
		hostPort = hostPort[len("http://"):]
	}
	host, portRaw, err := net.SplitHostPort(hostPort)
	if err != nil {
		t.Fatalf("split server URL %q: %v", rawURL, err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse server port %q: %v", portRaw, err)
	}
	return host, int32(port)
}
