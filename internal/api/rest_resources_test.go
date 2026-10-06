package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

func TestRESTResourcesListsServicesByEnvironmentKey(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	environment, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "staging", Name: "Staging"})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	service, err := storage.NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "payments", DisplayName: "Payments"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	if _, err := storage.NewDeploymentRepository(db).Create(ctx, &registryv1.CreateDeploymentRequest{ServiceId: service.GetId(), EnvironmentId: environment.GetId()}); err != nil {
		t.Fatalf("create deployment: %v", err)
	}

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/environments/staging/services", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	var body struct {
		Services []struct {
			Name string `json:"name"`
		} `json:"services"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Services) != 1 || body.Services[0].Name != "payments" {
		t.Fatalf("services = %+v, want payments", body.Services)
	}
}
