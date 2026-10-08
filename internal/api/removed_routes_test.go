package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRemovedCompatibilityRoutesAreUnavailable(t *testing.T) {
	f := newPublicFixture(t)
	for _, path := range []string{
		"/api/v1/catalog/environments", "/api/v1/catalog/services", "/api/v1/catalog/deployments",
		"/api/v1/catalog/instances", "/api/v1/catalog/endpoints", "/api/v1/discovery/services/Authentication.Grpc",
		"/registry.v1.RegistryService/ResolveService", "/registry.v1.RegistryService/ResolveEndpoint",
	} {
		t.Run(path, func(t *testing.T) {
			method := http.MethodGet
			if strings.HasPrefix(path, "/registry.") {
				method = http.MethodPost
			}
			request := httptest.NewRequest(method, path+"?environment=stg", strings.NewReader(`{}`))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+f.token)
			response := httptest.NewRecorder()
			f.mux.ServeHTTP(response, request)
			if response.Code != http.StatusNotFound {
				t.Fatalf("removed route still reachable: %d %s", response.Code, response.Body.String())
			}
		})
	}
	for path := range buildPublicOpenAPI()["paths"].(schemaMap) {
		if strings.HasPrefix(path, "/api/v1/catalog/") || strings.HasPrefix(path, "/api/v1/discovery/services/") {
			t.Fatalf("removed route advertised: %s", path)
		}
	}
}
