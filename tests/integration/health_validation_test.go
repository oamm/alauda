package integration_test

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
)

func TestHealthValidationAndNotFound(t *testing.T) {
	ctx := context.Background()
	db, srv, cleanup := setupIntegrationServer(t)
	defer cleanup()
	defer db.Close()

	healthClient := registryv1connect.NewHealthServiceClient(http.DefaultClient, srv.URL)

	if _, err := healthClient.CreateHealthCheck(ctx, connect.NewRequest(&registryv1.CreateHealthCheckRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty health check create, got: %v", err)
	}
	if _, err := healthClient.UpdateHealthCheck(ctx, connect.NewRequest(&registryv1.UpdateHealthCheckRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty health check update, got: %v", err)
	}
	if _, err := healthClient.DeleteHealthCheck(ctx, connect.NewRequest(&registryv1.DeleteHealthCheckRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty health check delete, got: %v", err)
	}
	if _, err := healthClient.RunHealthCheck(ctx, connect.NewRequest(&registryv1.RunHealthCheckRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty health check run, got: %v", err)
	}
	if _, err := healthClient.GetInstanceHealthState(ctx, connect.NewRequest(&registryv1.GetInstanceHealthStateRequest{})); connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("expected invalid argument for empty instance health state request, got: %v", err)
	}

	if _, err := healthClient.GetHealthCheck(ctx, connect.NewRequest(&registryv1.GetHealthCheckRequest{Id: "missing-hc"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for missing health check, got: %v", err)
	}
	if _, err := healthClient.GetInstanceHealthState(ctx, connect.NewRequest(&registryv1.GetInstanceHealthStateRequest{InstanceId: "missing-instance"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for missing health state, got: %v", err)
	}
	if _, err := healthClient.RunHealthCheck(ctx, connect.NewRequest(&registryv1.RunHealthCheckRequest{Id: "missing-hc"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found for missing health check run, got: %v", err)
	}
}
