package integration_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
)

func TestValidationAndNotFoundErrors(t *testing.T) {
	ctx := context.Background()
	db, srv, cleanup := setupIntegrationServer(t)
	defer cleanup()
	defer db.Close()

	envClient := registryv1connect.NewEnvironmentServiceClient(http.DefaultClient, srv.URL)
	catalogClient := registryv1connect.NewCatalogServiceClient(http.DefaultClient, srv.URL)
	deploymentClient := registryv1connect.NewDeploymentServiceClient(http.DefaultClient, srv.URL)
	instanceClient := registryv1connect.NewInstanceServiceClient(http.DefaultClient, srv.URL)
	endpointClient := registryv1connect.NewEndpointServiceClient(http.DefaultClient, srv.URL)

	if _, err := envClient.CreateEnvironment(ctx, connect.NewRequest(&registryv1.CreateEnvironmentRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty environment create, got: %v", err)
	}

	if _, err := catalogClient.CreateService(ctx, connect.NewRequest(&registryv1.CreateServiceRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty service create, got: %v", err)
	}

	if _, err := deploymentClient.CreateDeployment(ctx, connect.NewRequest(&registryv1.CreateDeploymentRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty deployment create, got: %v", err)
	}

	if _, err := instanceClient.CreateInstance(ctx, connect.NewRequest(&registryv1.CreateInstanceRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty instance create, got: %v", err)
	}

	if _, err := endpointClient.CreateEndpoint(ctx, connect.NewRequest(&registryv1.CreateEndpointRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty endpoint create, got: %v", err)
	}

	if _, err := envClient.GetEnvironment(ctx, connect.NewRequest(&registryv1.GetEnvironmentRequest{Id: "missing-env"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for missing environment, got: %v", err)
	}

	if _, err := catalogClient.GetService(ctx, connect.NewRequest(&registryv1.GetServiceRequest{Id: "missing-svc"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for missing service, got: %v", err)
	}

	if _, err := deploymentClient.GetDeployment(ctx, connect.NewRequest(&registryv1.GetDeploymentRequest{Id: "missing-dep"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for missing deployment, got: %v", err)
	}

	if _, err := instanceClient.GetInstance(ctx, connect.NewRequest(&registryv1.GetInstanceRequest{Id: "missing-inst"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for missing instance, got: %v", err)
	}

	if _, err := endpointClient.GetEndpoint(ctx, connect.NewRequest(&registryv1.GetEndpointRequest{Id: "missing-endpoint"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for missing endpoint, got: %v", err)
	}
}
