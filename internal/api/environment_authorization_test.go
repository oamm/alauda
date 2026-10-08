package api

import (
	"bytes"
	"context"
	"google.golang.org/protobuf/proto"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	v1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/config"
	"github.com/company/service-registry/internal/storage"
)

func TestScopedApplicationKeyCannotBypassEnvironmentBoundary(t *testing.T) {
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, fixtureDatabasePath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	er := storage.NewEnvironmentRepository(db)
	allowed, _ := er.Create(ctx, &v1.CreateEnvironmentRequest{Key: "stg", Name: "Staging"})
	denied, _ := er.Create(ctx, &v1.CreateEnvironmentRequest{Key: "prod", Name: "Production"})
	sr := storage.NewServiceRepository(db)
	service, _ := sr.Create(ctx, &v1.CreateServiceRequest{Name: "private", DisplayName: "Private"})
	runtime, err := storage.NewRuntimeRepository(db).RegisterRuntime(ctx, &v1.RegisterRuntimeRequest{ServiceId: service.Id, EnvironmentId: denied.Id, Instance: &v1.RuntimeInstanceRegistration{Name: "private-01", Address: "private-host"}, Endpoints: []*v1.RuntimeEndpointRegistration{{Name: "default", Protocol: v1.Protocol_PROTOCOL_HTTP, Port: 81, Enabled: true}}})
	if err != nil {
		t.Fatal(err)
	}
	key, err := auth.NewRepository(db).CreateApplicationKey(ctx, auth.CreateApplicationKeyInput{Name: "scoped", Scopes: []auth.Scope{auth.ScopeRead, auth.ScopeWrite}, EnvironmentIDs: []string{allowed.Id}, CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterRoutesWithConfig(mux, db, &config.Config{Auth: config.AuthConfig{Enabled: true, TokenTTL: time.Hour}})
	requests := []struct{ method, path, body string }{
		{"POST", "/api/v1/services/private/instances", `{"environment":"prod","instance":{"name":"private-01","address":"private-host"},"endpoints":[]}`},
		{"DELETE", "/api/v1/services/private/instances/private-01?environment=prod", ""},
		{"GET", "/api/v1/environments/prod", ""},
		{"GET", "/api/v1/services/private/instances?environment=prod", ""},
		{"GET", "/api/v1/health/checks?environmentId=" + denied.Id, ""},
		{"GET", "/api/v1/health/results?environmentId=" + denied.Id, ""},
		{"GET", "/api/v1/health/status?environment=prod", ""},
		{"POST", "/registry.v1.EnvironmentService/GetEnvironment", `{"id":"` + denied.Id + `"}`},
		{"POST", "/registry.v1.CatalogService/GetService", `{"id":"` + service.Id + `"}`},
		{"POST", "/registry.v1.DeploymentService/GetDeployment", `{"id":"` + runtime.Deployment.Id + `"}`},
		{"POST", "/registry.v1.EndpointService/GetEndpoint", `{"id":"` + runtime.Endpoints[0].Id + `"}`},
		{"POST", "/registry.v1.HealthService/GetInstanceHealthState", `{"instanceId":"` + runtime.Instance.Id + `"}`},
		{"POST", "/registry.v1.EventService/ListEvents", `{"environmentId":"` + denied.Id + `"}`},
		{"POST", "/registry.v1.EventService/WatchEvents", `{"environmentId":"` + denied.Id + `"}`},
		{"POST", "/registry.v1.IncidentService/ListIncidents", `{"environmentId":"` + denied.Id + `"}`},
		{"GET", "/api/v1/discovery/private/resolve?environment=prod", ""},
		{"GET", "/api/v1/events/watch?environmentId=" + denied.Id, ""},
		{"POST", "/registry.v1.InstanceService/GetInstance", `{"id":"` + runtime.Instance.Id + `"}`},
		{"POST", "/registry.v1.InstanceService/DeleteInstance", `{"id":"` + runtime.Instance.Id + `"}`},
		{"POST", "/registry.v1.InstanceService/ListInstances", "{}"},
	}
	// Protobuf is a separate reachable transport, not just Connect JSON.
	for _, item := range []struct {
		path    string
		message proto.Message
	}{{"/registry.v1.InstanceService/GetInstance", &v1.GetInstanceRequest{Id: runtime.Instance.Id}}, {"/registry.v1.DeploymentService/ListDeployments", &v1.ListDeploymentsRequest{EnvironmentId: denied.Id}}} {
		body, err := proto.Marshal(item.message)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("POST", item.path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/proto")
		req.Header.Set("Authorization", "Bearer "+key.Secret)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != 403 {
			t.Fatalf("protobuf bypass %s: %d %s", item.path, res.Code, res.Body.String())
		}
	}
	for _, item := range requests {
		t.Run(item.path, func(t *testing.T) {
			req := httptest.NewRequest(item.method, item.path, bytes.NewBufferString(item.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+key.Secret)
			res := httptest.NewRecorder()
			mux.ServeHTTP(res, req)
			if res.Code != 403 {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
	req := httptest.NewRequest("GET", "/api/v1/services", nil)
	req.Header.Set("Authorization", "Bearer "+key.Secret)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != 200 || bytes.Contains(res.Body.Bytes(), []byte(`"private"`)) {
		t.Fatalf("catalog=%d %s", res.Code, res.Body.String())
	}
	getPrivate := httptest.NewRequest("GET", "/api/v1/services/private?environment=stg", nil)
	getPrivate.Header.Set("Authorization", "Bearer "+key.Secret)
	privateResponse := httptest.NewRecorder()
	mux.ServeHTTP(privateResponse, getPrivate)
	if privateResponse.Code != 404 || strings.Contains(privateResponse.Body.String(), service.Id) {
		t.Fatal("private Service metadata escaped through an unrelated allowed Environment")
	}
	// An unrelated allowed filter must not bypass a list's scope predicate.
	for _, path := range []string{"/api/v1/services/private/instances?environment=stg", "/api/v1/services?environment=stg", "/api/v1/health/status?environment=stg", "/api/v1/health/status"} {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+key.Secret)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != 200 || strings.Contains(res.Body.String(), runtime.Instance.Id) {
			t.Fatalf("filter bypass: %s", res.Body.String())
		}
	}
	allowedRuntime, err := storage.NewRuntimeRepository(db).RegisterRuntime(ctx, &v1.RegisterRuntimeRequest{ServiceId: service.Id, EnvironmentId: allowed.Id, Instance: &v1.RuntimeInstanceRegistration{Name: "allowed-01", Address: "allowed-host"}, Endpoints: []*v1.RuntimeEndpointRegistration{{Name: "default", Protocol: v1.Protocol_PROTOCOL_HTTP, Port: 81, Enabled: true}}})
	if err != nil {
		t.Fatal(err)
	}
	if allowedRuntime.Instance.Id == runtime.Instance.Id {
		t.Fatal("scope identities aliased")
	}
	discoveryKey, err := auth.NewRepository(db).CreateApplicationKey(ctx, auth.CreateApplicationKeyInput{Name: "discovery", Scopes: []auth.Scope{auth.ScopeDiscoveryRead}, EnvironmentIDs: []string{allowed.Id}, CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		method, path, body string
		want               int
	}{{"GET", "/api/v1/discovery/private/resolve?environment=stg", "", 200}, {"GET", "/api/v1/discovery/private/resolve?environment=prod", "", 403}, {"POST", "/api/v1/services/private/instances", `{"environment":"stg","instance":{"name":"allowed-01"},"endpoints":[]}`, 403}, {"POST", "/api/v1/services/private/instances/allowed-01/health-checks/test/run?environment=stg", "", 403}} {
		req := httptest.NewRequest(item.method, item.path, strings.NewReader(item.body))
		req.Header.Set("Authorization", "Bearer "+discoveryKey.Secret)
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != item.want {
			t.Fatalf("least privilege %s: %d %s", item.path, res.Code, res.Body.String())
		}
	}
	for _, path := range []string{"/api/v1/services", "/api/v1/audit-logs", "/api/v1/alerts/test/anything", "/api/v1/health/checks"} {
		req := httptest.NewRequest("GET", path, nil)
		req.Header.Set("Authorization", "Bearer "+discoveryKey.Secret)
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != 403 {
			t.Fatalf("discovery-only %s=%d", path, res.Code)
		}
	}
}
