package integration_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
)

func TestRegisterRuntimeCreatesDeploymentAndEndpoints(t *testing.T) {
	ctx := context.Background()
	db, srv, cleanup := setupIntegrationServer(t)
	defer cleanup()
	defer db.Close()

	envClient := registryv1connect.NewEnvironmentServiceClient(http.DefaultClient, srv.URL)
	catalogClient := registryv1connect.NewCatalogServiceClient(http.DefaultClient, srv.URL)
	deploymentClient := registryv1connect.NewDeploymentServiceClient(http.DefaultClient, srv.URL)
	instanceClient := registryv1connect.NewInstanceServiceClient(http.DefaultClient, srv.URL)
	endpointClient := registryv1connect.NewEndpointServiceClient(http.DefaultClient, srv.URL)

	envID := createRuntimeTestEnvironment(t, ctx, envClient, "prod")
	serviceID := createRuntimeTestService(t, ctx, catalogClient, "checkout")

	resp, err := instanceClient.RegisterRuntime(ctx, connect.NewRequest(&registryv1.RegisterRuntimeRequest{
		ServiceId:     serviceID,
		EnvironmentId: envID,
		Instance: &registryv1.RuntimeInstanceRegistration{
			Name:        "checkout-prod-01",
			Address:     "10.0.0.10",
			Description: "primary checkout instance",
		},
		Endpoints: []*registryv1.RuntimeEndpointRegistration{
			{Name: "http", Kind: registryv1.EndpointKind_ENDPOINT_KIND_HTTPS, Port: 8080, Path: "/", Primary: true},
			{Name: "grpc", Kind: registryv1.EndpointKind_ENDPOINT_KIND_GRPC, Port: 5001},
		},
	}))
	if err != nil {
		t.Fatalf("register runtime: %v", err)
	}
	if resp.Msg.GetDeployment().GetServiceId() != serviceID || resp.Msg.GetDeployment().GetEnvironmentId() != envID {
		t.Fatalf("deployment = %+v, want service/environment", resp.Msg.GetDeployment())
	}
	if resp.Msg.GetInstance().GetDeploymentId() != resp.Msg.GetDeployment().GetId() {
		t.Fatalf("instance deployment = %q, want %q", resp.Msg.GetInstance().GetDeploymentId(), resp.Msg.GetDeployment().GetId())
	}
	if got := resp.Msg.GetEndpoints(); len(got) != 2 || !got[0].GetPrimary() || got[1].GetPrimary() {
		t.Fatalf("endpoints = %+v, want first primary only", got)
	}

	deployments, err := deploymentClient.ListDeployments(ctx, connect.NewRequest(&registryv1.ListDeploymentsRequest{
		ServiceId:     serviceID,
		EnvironmentId: envID,
	}))
	if err != nil {
		t.Fatalf("list deployments: %v", err)
	}
	if len(deployments.Msg.GetDeployments()) != 1 {
		t.Fatalf("deployment count = %d, want 1", len(deployments.Msg.GetDeployments()))
	}

	instances, err := instanceClient.ListInstances(ctx, connect.NewRequest(&registryv1.ListInstancesRequest{
		DeploymentId: resp.Msg.GetDeployment().GetId(),
	}))
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(instances.Msg.GetInstances()) != 1 {
		t.Fatalf("instance count = %d, want 1", len(instances.Msg.GetInstances()))
	}
	endpoints, err := endpointClient.ListEndpoints(ctx, connect.NewRequest(&registryv1.ListEndpointsRequest{
		InstanceId: resp.Msg.GetInstance().GetId(),
	}))
	if err != nil {
		t.Fatalf("list endpoints: %v", err)
	}
	if len(endpoints.Msg.GetEndpoints()) != 2 {
		t.Fatalf("endpoint count = %d, want 2", len(endpoints.Msg.GetEndpoints()))
	}
}

func TestRegisterRuntimeReusesDeploymentAndRollsBackDuplicateInstance(t *testing.T) {
	ctx := context.Background()
	db, srv, cleanup := setupIntegrationServer(t)
	defer cleanup()
	defer db.Close()

	envClient := registryv1connect.NewEnvironmentServiceClient(http.DefaultClient, srv.URL)
	catalogClient := registryv1connect.NewCatalogServiceClient(http.DefaultClient, srv.URL)
	deploymentClient := registryv1connect.NewDeploymentServiceClient(http.DefaultClient, srv.URL)
	instanceClient := registryv1connect.NewInstanceServiceClient(http.DefaultClient, srv.URL)

	envID := createRuntimeTestEnvironment(t, ctx, envClient, "qa")
	serviceID := createRuntimeTestService(t, ctx, catalogClient, "billing")
	deploymentResp, err := deploymentClient.CreateDeployment(ctx, connect.NewRequest(&registryv1.CreateDeploymentRequest{
		ServiceId:            serviceID,
		EnvironmentId:        envID,
		HealthEnabled:        true,
		AlertsEnabled:        true,
		AlertCooldownMinutes: 10,
	}))
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	deploymentID := deploymentResp.Msg.GetDeployment().GetId()

	registerReq := &registryv1.RegisterRuntimeRequest{
		ServiceId:     serviceID,
		EnvironmentId: envID,
		Instance: &registryv1.RuntimeInstanceRegistration{
			Name:    "billing-qa-01",
			Address: "10.0.1.10",
		},
		Endpoints: []*registryv1.RuntimeEndpointRegistration{
			{Name: "http", Kind: registryv1.EndpointKind_ENDPOINT_KIND_HTTP, Port: 8080},
		},
	}
	resp, err := instanceClient.RegisterRuntime(ctx, connect.NewRequest(registerReq))
	if err != nil {
		t.Fatalf("register runtime: %v", err)
	}
	if resp.Msg.GetDeployment().GetId() != deploymentID {
		t.Fatalf("deployment id = %q, want reused %q", resp.Msg.GetDeployment().GetId(), deploymentID)
	}
	if got := resp.Msg.GetEndpoints(); len(got) != 1 || !got[0].GetPrimary() {
		t.Fatalf("single endpoint = %+v, want auto primary", got)
	}

	_, err = instanceClient.RegisterRuntime(ctx, connect.NewRequest(registerReq))
	if connect.CodeOf(err) != connect.CodeAlreadyExists {
		t.Fatalf("duplicate code = %s, want already_exists: %v", connect.CodeOf(err), err)
	}

	deployments, err := deploymentClient.ListDeployments(ctx, connect.NewRequest(&registryv1.ListDeploymentsRequest{
		ServiceId:     serviceID,
		EnvironmentId: envID,
	}))
	if err != nil {
		t.Fatalf("list deployments: %v", err)
	}
	if len(deployments.Msg.GetDeployments()) != 1 {
		t.Fatalf("deployment count = %d, want 1", len(deployments.Msg.GetDeployments()))
	}
	instances, err := instanceClient.ListInstances(ctx, connect.NewRequest(&registryv1.ListInstancesRequest{DeploymentId: deploymentID}))
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(instances.Msg.GetInstances()) != 1 {
		t.Fatalf("instance count = %d, want rollback left 1", len(instances.Msg.GetInstances()))
	}
}

func TestRegisterRuntimeEnvironmentIsolation(t *testing.T) {
	ctx := context.Background()
	db, srv, cleanup := setupIntegrationServer(t)
	defer cleanup()
	defer db.Close()

	envClient := registryv1connect.NewEnvironmentServiceClient(http.DefaultClient, srv.URL)
	catalogClient := registryv1connect.NewCatalogServiceClient(http.DefaultClient, srv.URL)
	deploymentClient := registryv1connect.NewDeploymentServiceClient(http.DefaultClient, srv.URL)
	instanceClient := registryv1connect.NewInstanceServiceClient(http.DefaultClient, srv.URL)

	devID := createRuntimeTestEnvironment(t, ctx, envClient, "dev")
	prodID := createRuntimeTestEnvironment(t, ctx, envClient, "prod")
	serviceID := createRuntimeTestService(t, ctx, catalogClient, "orders")

	for _, item := range []struct {
		envID string
		name  string
	}{
		{devID, "orders-dev-01"},
		{prodID, "orders-prod-01"},
	} {
		if _, err := instanceClient.RegisterRuntime(ctx, connect.NewRequest(&registryv1.RegisterRuntimeRequest{
			ServiceId:     serviceID,
			EnvironmentId: item.envID,
			Instance:      &registryv1.RuntimeInstanceRegistration{Name: item.name, Address: "orders.internal"},
			Endpoints: []*registryv1.RuntimeEndpointRegistration{
				{Name: "http", Kind: registryv1.EndpointKind_ENDPOINT_KIND_HTTP, Port: 8080},
			},
		})); err != nil {
			t.Fatalf("register %s: %v", item.name, err)
		}
	}

	for _, envID := range []string{devID, prodID} {
		resp, err := deploymentClient.ListDeployments(ctx, connect.NewRequest(&registryv1.ListDeploymentsRequest{
			ServiceId:     serviceID,
			EnvironmentId: envID,
		}))
		if err != nil {
			t.Fatalf("list deployments: %v", err)
		}
		if len(resp.Msg.GetDeployments()) != 1 {
			t.Fatalf("env %s deployment count = %d, want 1", envID, len(resp.Msg.GetDeployments()))
		}
	}
}

func createRuntimeTestEnvironment(t *testing.T, ctx context.Context, client registryv1connect.EnvironmentServiceClient, key string) string {
	t.Helper()
	resp, err := client.CreateEnvironment(ctx, connect.NewRequest(&registryv1.CreateEnvironmentRequest{
		Key:  key,
		Name: key,
		Tier: key,
	}))
	if err != nil {
		t.Fatalf("create environment %s: %v", key, err)
	}
	return resp.Msg.GetEnvironment().GetId()
}

func createRuntimeTestService(t *testing.T, ctx context.Context, client registryv1connect.CatalogServiceClient, name string) string {
	t.Helper()
	resp, err := client.CreateService(ctx, connect.NewRequest(&registryv1.CreateServiceRequest{
		Name:        name,
		DisplayName: name,
	}))
	if err != nil {
		t.Fatalf("create service %s: %v", name, err)
	}
	return resp.Msg.GetService().GetId()
}
