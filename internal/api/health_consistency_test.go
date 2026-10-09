package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

func TestHealthStatusConsistentAcrossPublicSurfaces(t *testing.T) {
	f := newPublicFixture(t)
	first := requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"one","address":"host-one"},"endpoints":[{"name":"default","kind":"http","port":81}]}`))
	second := requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"two","address":"host-two"},"endpoints":[{"name":"default","kind":"http","port":82}]}`))
	if _, err := f.db.Exec(context.Background(), "UPDATE service_deployments SET health_enabled=1 WHERE service_id=? AND environment_id=?", f.serviceID, f.environmentID); err != nil {
		t.Fatal(err)
	}
	check, err := storage.NewHealthRepository(f.db).CreateHealthCheck(context.Background(), &registryv1.CreateHealthCheckRequest{InstanceId: first.Instance.ID, EndpointId: first.Endpoints[0].ID, Name: "ready", Type: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP, Enabled: true, IntervalSeconds: 10, TimeoutSeconds: 3})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO health_results(id,health_check_id,instance_id,timestamp,success) VALUES(?,?,?,?,1)`, "older-success", check.Id, first.Instance.ID, time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `INSERT INTO health_results(id,health_check_id,instance_id,timestamp,success) VALUES(?,?,?,?,0)`, "latest-failure", check.Id, first.Instance.ID, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	for _, route := range []string{
		"/api/v1/services?environment=stg",
		"/api/v1/services/Authentication.Grpc?environment=stg",
		"/api/v1/environments/stg/services",
		"/api/v1/health/status?environment=stg",
	} {
		response := f.request("GET", route, "")
		if response.Code != 200 || !strings.Contains(response.Body.String(), `"Unhealthy"`) {
			t.Fatalf("%s: %d %s", route, response.Code, response.Body.String())
		}
	}
	management := f.request("GET", "/api/v1/services/Authentication.Grpc/instances?environment=stg", "")
	if management.Code != 200 {
		t.Fatal(management.Body.String())
	}
	var data struct {
		Instances []publicInstance `json:"instances"`
	}
	if err := json.Unmarshal(management.Body.Bytes(), &data); err != nil {
		t.Fatal(err)
	}
	if len(data.Instances) != 2 || data.Instances[0].HealthState != "Unhealthy" || data.Instances[0].Healthy || data.Instances[1].ID != second.Instance.ID || data.Instances[1].HealthState != "Unknown" {
		t.Fatalf("management states = %+v", data.Instances)
	}
	all := f.request("GET", "/api/v1/discovery/Authentication.Grpc?environment=stg&health=all", "")
	if all.Code != 200 || !strings.Contains(all.Body.String(), `"healthState":"Unhealthy"`) || !strings.Contains(all.Body.String(), `"healthState":"Unknown"`) {
		t.Fatalf("all discovery = %d %s", all.Code, all.Body.String())
	}
	usable := f.request("GET", "/api/v1/discovery/Authentication.Grpc?environment=stg", "")
	if usable.Code != 200 || strings.Contains(usable.Body.String(), `"name":"one"`) || !strings.Contains(usable.Body.String(), `"name":"two"`) {
		t.Fatalf("usable discovery = %d %s", usable.Code, usable.Body.String())
	}
	if response := f.request("GET", "/api/v1/discovery/Authentication.Grpc?environment=stg&health=healthy", ""); response.Code != 404 {
		t.Fatalf("healthy discovery = %d %s", response.Code, response.Body.String())
	}
}
