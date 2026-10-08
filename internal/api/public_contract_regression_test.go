package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	v1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/config"
	"github.com/company/service-registry/internal/storage"
)

type publicFixture struct {
	db                              *storage.Database
	mux                             *http.ServeMux
	token, serviceID, environmentID string
}

func TestPublicKeyManagementAndBootstrap(t *testing.T) {
	f := newPublicFixture(t)
	for _, item := range []struct{ path, body string }{{"/api/v1/environments", `{"key":"dev","name":"Development"}`}, {"/api/v1/services", `{"name":"Redis"}`}} {
		w := f.request("POST", item.path, item.body)
		if w.Code != 201 {
			t.Fatalf("%s: %d %s", item.path, w.Code, w.Body.String())
		}
	}
	out := requireRegistration(t, f.request("POST", registerRoute, initialRegistration))
	requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01","enabled":false},"endpoints":[{"name":"default","enabled":false}]}`))
	for _, path := range []string{registerRoute + "?environment=stg", registerRoute + "/auth-01?environment=stg", registerRoute + "/auth-01/endpoints?environment=stg"} {
		w := f.request("GET", path, "")
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"enabled":false`) {
			t.Fatalf("%s: %s", path, w.Body.String())
		}
	}
	w := f.request("GET", registerRoute+"/missing/endpoints?environment=stg", "")
	if w.Code != 404 || !strings.Contains(w.Body.String(), "instance_not_found") {
		t.Fatal(w.Body.String())
	}
	w = f.request("POST", registerRoute+"/auth-01/health-checks?environment=stg", `{"name":"alive","endpoint":"default","type":"http"}`)
	if w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	w = f.request("GET", registerRoute+"/auth-01/health-checks?environment=stg", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "deployment") || !strings.Contains(w.Body.String(), "alive") {
		t.Fatal(w.Body.String())
	}
	w = f.request("POST", registerRoute+"/auth-01/health-checks/alive/run?environment=stg", "")
	if w.Code != 404 {
		t.Fatalf("inactive target executed: %s", w.Body.String())
	}
	if _, err := storage.NewHealthRepository(f.db).GetHealthCheckTarget(context.Background(), out.Instance.ID); err == nil {
		t.Fatal("invalid target")
	}
	for _, query := range []string{"pageSize=201", "pageSize=0", "pageToken=-1", "pageToken=bad"} {
		if f.request("GET", "/api/v1/services?"+query, "").Code != 400 {
			t.Fatal(query)
		}
	}
}

func TestPublicHealthExecutionAndRetainedLifecycle(t *testing.T) {
	f := newPublicFixture(t)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer target.Close()
	u, err := url.Parse(target.URL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"environment":"stg","instance":{"name":"probe","address":"%s"},"endpoints":[{"name":"default","protocol":"http","port":%d}]}`, u.Hostname(), port)
	out := requireRegistration(t, f.request("POST", registerRoute, body))
	path := registerRoute + "/probe/health-checks"
	w := f.request("POST", path+"?environment=stg", `{"name":"alive","endpoint":"default","type":"http","successesBeforeHealthy":1}`)
	if w.Code != 201 {
		t.Fatal(w.Body.String())
	}
	run := func(want int) {
		t.Helper()
		w := f.request("POST", path+"/alive/run?environment=stg", "")
		if w.Code != want {
			t.Fatalf("run: %d %s", w.Code, w.Body.String())
		}
		if want == 200 && !strings.Contains(w.Body.String(), `"healthState":"Healthy"`) {
			t.Fatal(w.Body.String())
		}
	}
	run(200)
	requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"probe"},"endpoints":[{"name":"default","enabled":false}]}`))
	run(404)
	w = f.request("DELETE", registerRoute+"/probe?environment=stg", "")
	if w.Code != 204 {
		t.Fatal(w.Body.String())
	}
	w = f.request("GET", "/api/v1/services/Authentication.Grpc/health-results?environment=stg", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"check":"alive"`) || strings.Contains(w.Body.String(), "deployment") {
		t.Fatal(w.Body.String())
	}
	out2 := requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"probe"},"endpoints":[{"name":"default","enabled":true}]}`))
	if out2.Instance.ID != out.Instance.ID {
		t.Fatal("restore identity changed")
	}
	run(200)
}

func TestAuthenticationStorageFailureIsUnavailable(t *testing.T) {
	f := newPublicFixture(t)
	if err := f.db.Close(); err != nil {
		t.Fatal(err)
	}
	w := f.request("GET", "/api/v1/services", "")
	if w.Code != 503 || w.Header().Get("Content-Type") != "application/problem+json" || strings.Contains(w.Body.String(), "sql:") {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
}

func TestPublicMembershipBeyondFirstPageAndScopedPagination(t *testing.T) {
	f := newPublicFixture(t)
	ctx := context.Background()
	for i := 0; i < 201; i++ {
		name := fmt.Sprintf("paging-%03d", i)
		svc, err := storage.NewServiceRepository(f.db).Create(ctx, &v1.CreateServiceRequest{Name: name, DisplayName: name})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = storage.NewDeploymentRepository(f.db).Create(ctx, &v1.CreateDeploymentRequest{ServiceId: svc.Id, EnvironmentId: f.environmentID}); err != nil {
			t.Fatal(err)
		}
	}
	w := f.request("GET", "/api/v1/environments/stg/services/paging-200", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "paging-200") {
		t.Fatal("membership lookup depends on pagination")
	}
	allowed, err := storage.NewEnvironmentRepository(f.db).Create(ctx, &v1.CreateEnvironmentRequest{Key: "last-env", Name: "Last"})
	if err != nil {
		t.Fatal(err)
	}
	key, err := auth.NewRepository(f.db).CreateApplicationKey(ctx, auth.CreateApplicationKeyInput{Name: "page", Scopes: []auth.Scope{auth.ScopeRegistryRead}, EnvironmentIDs: []string{allowed.Id}, CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	f.token = key.Secret
	w = f.request("GET", "/api/v1/environments?pageSize=1", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "last-env") || strings.Contains(w.Body.String(), `"key":"stg"`) {
		t.Fatalf("authorization not before pagination: %s", w.Body.String())
	}
}

func TestDiscoveryIterationFailureDoesNotReturnPartialSuccess(t *testing.T) {
	f := newPublicFixture(t)
	for _, name := range []string{"a-good", "z-broken"} {
		body := fmt.Sprintf(`{"environment":"stg","instance":{"name":"%s","address":"host"},"endpoints":[{"name":"default","protocol":"http","port":81}]}`, name)
		requireRegistration(t, f.request("POST", registerRoute, body))
	}
	if _, err := f.db.Exec(context.Background(), "ALTER TABLE service_instances RENAME TO instance_fixture"); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(context.Background(), `CREATE VIEW service_instances AS SELECT id,deployment_id,name,CASE WHEN name='z-broken' THEN abs(-9223372036854775808) ELSE address END AS address,port,description,enabled,tags,metadata,created_at,updated_at,last_seen_at,deleted_at FROM instance_fixture`); err != nil {
		t.Fatal(err)
	}
	w := f.request("GET", "/api/v1/discovery/Authentication.Grpc?environment=stg", "")
	if w.Code != 500 || !strings.Contains(w.Body.String(), "server_error") || strings.Contains(w.Body.String(), "overflow") || strings.Contains(w.Body.String(), "a-good") {
		t.Fatalf("iteration failure: %d %s", w.Code, w.Body.String())
	}
}

func TestPublicDiscoveryHealthPoliciesAndOrdering(t *testing.T) {
	f := newPublicFixture(t)
	for _, item := range []struct{ name, state string }{{"a-unknown", "Unknown"}, {"z-healthy", "Healthy"}, {"b-unhealthy", "Unhealthy"}, {"c-degraded", "Degraded"}, {"d-disabled", "Disabled"}} {
		body := `{"environment":"stg","instance":{"name":"` + item.name + `","address":"auth-host"},"endpoints":[{"name":"default","protocol":"http","port":81},{"name":"grpc","protocol":"grpc","port":82}]}`
		out := requireRegistration(t, f.request("POST", registerRoute, body))
		if out.Instance.Healthy || out.Instance.HealthState != "Unknown" {
			t.Fatal("registration invented healthy state")
		}
		if item.state == "Disabled" {
			if _, err := f.db.Exec(context.Background(), "UPDATE service_instances SET enabled=0 WHERE id=?", out.Instance.ID); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if item.state != "Unknown" {
			check, err := storage.NewHealthRepository(f.db).CreateHealthCheck(context.Background(), &v1.CreateHealthCheckRequest{InstanceId: out.Instance.ID, EndpointId: out.Endpoints[0].ID, Name: "check", Type: v1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP, Enabled: true, IntervalSeconds: 10, TimeoutSeconds: 3, FailuresBeforeUnhealthy: 3, SuccessesBeforeHealthy: 2})
			if err != nil {
				t.Fatal(err)
			}
			now := time.Now().UTC().Format(time.RFC3339Nano)
			_, err = f.db.Exec(context.Background(), `INSERT INTO health_results(id,health_check_id,instance_id,timestamp,success) VALUES(?,?,?,?,?)`, item.name, check.Id, out.Instance.ID, now, item.state == "Healthy" || item.state == "Degraded")
			if err != nil {
				t.Fatal(err)
			}
			if item.state == "Degraded" {
				second, err := storage.NewHealthRepository(f.db).CreateHealthCheck(context.Background(), &v1.CreateHealthCheckRequest{InstanceId: out.Instance.ID, EndpointId: out.Endpoints[0].ID, Name: "second", Type: v1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP, Enabled: true, IntervalSeconds: 10, TimeoutSeconds: 3})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := f.db.Exec(context.Background(), `INSERT INTO health_results(id,health_check_id,instance_id,timestamp,success) VALUES(?,?,?,?,0)`, "second-"+item.name, second.Id, out.Instance.ID, now); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	for _, policy := range []struct {
		value string
		count int
	}{{"usable", 3}, {"healthy", 1}, {"all", 4}} {
		w := f.request("GET", "/api/v1/discovery/Authentication.Grpc?environment=stg&health="+policy.value, "")
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		var data publicDiscoveryResponse
		json.Unmarshal(w.Body.Bytes(), &data)
		if len(data.Instances) != policy.count || data.Instances[0].Name != "z-healthy" {
			t.Fatalf("%s=%+v", policy.value, data)
		}
	}
	w := f.request("GET", "/api/v1/discovery/Authentication.Grpc/resolve?environment=stg&endpoint=grpc", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"address":"grpc://auth-host:82"`) || !strings.Contains(w.Body.String(), `"instance":"z-healthy"`) {
		t.Fatal(w.Body.String())
	}
	for _, path := range []string{"/api/v1/discovery/Authentication.Grpc/resolve/extra?environment=stg", "/api/v1/discovery/Authentication.Grpc/wrong?environment=stg"} {
		if f.request("GET", path, "").Code != 404 {
			t.Fatal("loose route")
		}
	}
	if f.request("GET", "/api/v1/discovery/Authentication.Grpc?environment=stg&health=invalid", "").Code != 400 {
		t.Fatal("invalid policy accepted")
	}
	_, err := f.db.Exec(context.Background(), `UPDATE health_results SET timestamp='2000-01-01T00:00:00Z'`)
	if err != nil {
		t.Fatal(err)
	}
	if f.request("GET", "/api/v1/discovery/Authentication.Grpc?environment=stg&health=healthy", "").Code != 404 {
		t.Fatal("stale state routed as healthy")
	}
	_, err = f.db.Exec(context.Background(), `UPDATE environments SET enabled=FALSE WHERE id=?`, f.environmentID)
	if err != nil {
		t.Fatal(err)
	}
	if f.request("GET", "/api/v1/discovery/Authentication.Grpc?environment=stg", "").Code != 409 {
		t.Fatal("disabled environment discovered")
	}
	if f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"a-unknown"},"endpoints":[]}`).Code != 409 {
		t.Fatal("disabled environment registered")
	}
}

func TestPublicDiscoveryEndpointEligibilityAndFallback(t *testing.T) {
	f := newPublicFixture(t)
	requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"one","address":"::1"},"endpoints":[{"name":"z-primary","protocol":"http","port":81,"primary":true},{"name":"a-secondary","protocol":"https","port":82}]}`))
	resolve := func(query string, code int, fragment string) {
		t.Helper()
		w := f.request("GET", "/api/v1/discovery/Authentication.Grpc/resolve?environment=stg"+query, "")
		if w.Code != code || !strings.Contains(w.Body.String(), fragment) {
			t.Fatalf("%d %s", w.Code, w.Body.String())
		}
	}
	resolve("", 200, `"address":"http://[::1]:81/"`)
	requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"one"},"endpoints":[{"name":"z-primary","primary":false}]}`))
	resolve("", 200, `"endpoint":"a-secondary"`)
	requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"one"},"endpoints":[{"name":"a-secondary","enabled":false}]}`))
	resolve("", 200, `"endpoint":"z-primary"`)
	resolve("&endpoint=a-secondary", 404, "endpoint_not_found")
	resolve("&endpoint=missing", 404, "endpoint_not_found")
	requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"one","enabled":false},"endpoints":[]}`))
	resolve("", 404, "no_healthy_instance")
	requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"empty","address":"host"},"endpoints":[]}`))
	resolve("", 404, "no_healthy_instance")
	w := f.request("GET", registerRoute+"?environment=stg", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "empty") {
		t.Fatal("endpointless resources hidden from management")
	}
}

func newPublicFixture(t *testing.T) *publicFixture {
	t.Helper()
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, fixtureDatabasePath(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	env, err := storage.NewEnvironmentRepository(db).Create(ctx, &v1.CreateEnvironmentRequest{Key: "stg", Name: "Staging"})
	if err != nil {
		t.Fatal(err)
	}
	service, err := storage.NewServiceRepository(db).Create(ctx, &v1.CreateServiceRequest{Name: "Authentication.Grpc", DisplayName: "Authentication"})
	if err != nil {
		t.Fatal(err)
	}
	key, err := auth.NewRepository(db).CreateApplicationKey(ctx, auth.CreateApplicationKeyInput{Name: "contract", Scopes: []auth.Scope{auth.ScopeAdmin}, CreatedBy: "test"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterRoutesWithConfig(mux, db, &config.Config{Auth: config.AuthConfig{Enabled: true, TokenTTL: time.Hour}})
	return &publicFixture{db: db, mux: mux, token: key.Secret, serviceID: service.Id, environmentID: env.Id}
}
func (f *publicFixture) request(method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+f.token)
	w := httptest.NewRecorder()
	f.mux.ServeHTTP(w, r)
	return w
}

const registerRoute = "/api/v1/services/Authentication.Grpc/instances"
const initialRegistration = `{"environment":"stg","instance":{"name":"auth-01","address":"auth-host","description":"retain","enabled":false,"tags":{"team":"auth"}},"endpoints":[{"name":"default","protocol":"http","port":81,"path":"/original","primary":true,"enabled":false},{"name":"metrics","protocol":"http","port":9090}]}`

func requireRegistration(t *testing.T, w *httptest.ResponseRecorder) publicRegistrationResponse {
	t.Helper()
	if w.Code != 200 {
		t.Fatalf("registration=%d %s", w.Code, w.Body.String())
	}
	var out publicRegistrationResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func TestPublicUpsertPreservesFieldsAndPromotesPrimaryAtomically(t *testing.T) {
	f := newPublicFixture(t)
	first := requireRegistration(t, f.request("POST", registerRoute, initialRegistration))
	update := `{"environment":"stg","instance":{"name":"auth-01"},"endpoints":[{"name":"metrics","primary":true}]}`
	out := requireRegistration(t, f.request("POST", registerRoute, update))
	if out.Instance.ID != first.Instance.ID || out.Instance.Enabled || out.Instance.Description != "retain" || out.Instance.Address != "auth-host" || out.Instance.Tags["team"] != "auth" {
		t.Fatalf("instance=%+v", out.Instance)
	}
	if len(out.Endpoints) != 2 || out.Endpoints[0].Primary || out.Endpoints[0].Enabled || out.Endpoints[0].Path != "/original" || !out.Endpoints[1].Primary {
		t.Fatalf("endpoints=%+v", out.Endpoints)
	}
	repeated := requireRegistration(t, f.request("POST", registerRoute, update))
	if repeated.Endpoints[1].ID != out.Endpoints[1].ID {
		t.Fatal("endpoint identity changed")
	}
	out = requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01"},"endpoints":[{"name":"admin","protocol":"http","port":9000,"primary":true}]}`))
	if len(out.Endpoints) != 3 || !out.Endpoints[0].Primary || out.Endpoints[2].Primary {
		t.Fatalf("add Primary=%+v", out.Endpoints)
	}
	out = requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01"},"endpoints":[{"name":"metrics","primary":false},{"name":"admin","primary":true}]}`))
	if !out.Endpoints[0].Primary {
		t.Fatal("request order affected Primary")
	}
}
func TestPublicSingletonReplacementRestoreAndRollback(t *testing.T) {
	f := newPublicFixture(t)
	out := requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01","address":"auth-host"},"endpoints":[{"name":"default","protocol":"http","port":81,"primary":false}]}`))
	if out.Endpoints[0].Primary {
		t.Fatal("explicit false ignored")
	}
	out = requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01"},"endpoints":[{"name":"metrics","protocol":"http","port":9090}]}`))
	if out.Endpoints[1].Primary {
		t.Fatal("incremental singleton promoted")
	}
	firstID := out.Instance.ID
	if _, err := f.db.Exec(context.Background(), `CREATE TRIGGER fail_endpoint BEFORE INSERT ON endpoints WHEN NEW.name='fail' BEGIN SELECT RAISE(ABORT,'test failure'); END`); err != nil {
		t.Fatal(err)
	}
	failed := f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01","address":"changed"},"endpoints":[{"name":"default","port":82},{"name":"fail","protocol":"http","port":9000,"primary":true}]}`)
	if failed.Code != 409 {
		t.Fatalf("rollback=%d %s", failed.Code, failed.Body.String())
	}
	out = requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01"},"endpoints":[]}`))
	if out.Instance.Address != "auth-host" || out.Endpoints[0].Port != 81 {
		t.Fatal("failed transaction mutated state")
	}
	out = requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","replaceEndpoints":true,"instance":{"name":"auth-01"},"endpoints":[{"name":"metrics","primary":true}]}`))
	if len(out.Endpoints) != 1 {
		t.Fatal("replacement preserved omitted endpoint")
	}
	for i := 0; i < 2; i++ {
		w := f.request("DELETE", registerRoute+"/auth-01?environment=stg", "")
		if w.Code != 204 {
			t.Fatalf("delete=%d", w.Code)
		}
	}
	if _, err := f.db.Exec(context.Background(), `UPDATE service_deployments SET deleted_at='2026-01-01T00:00:00Z' WHERE service_id=?`, f.serviceID); err != nil {
		t.Fatal(err)
	}
	out = requireRegistration(t, f.request("POST", registerRoute, `{"environment":"stg","instance":{"name":"auth-01"},"endpoints":[]}`))
	if out.Instance.ID != firstID {
		t.Fatal("restore changed identity")
	}
}
func TestPublicConcurrentUpsertAndStrictBody(t *testing.T) {
	f := newPublicFixture(t)
	body := `{"environment":"stg","instance":{"name":"auth-01","address":"auth-host"},"endpoints":[{"name":"default","protocol":"http","port":81}]}`
	var wg sync.WaitGroup
	results := make(chan *httptest.ResponseRecorder, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- f.request("POST", registerRoute, body) }()
	}
	wg.Wait()
	close(results)
	var id string
	for w := range results {
		out := requireRegistration(t, w)
		if id != "" && id != out.Instance.ID {
			t.Fatal("concurrent duplicate")
		}
		id = out.Instance.ID
	}
	for _, invalid := range []string{body + ` {}`, `{"environment":"stg","mode":"create","instance":{"name":"auth-01"}}`, `{"environment":"stg","instance":{"name":"auth-01"},"unexpected":true}`} {
		w := f.request("POST", registerRoute, invalid)
		if w.Code != 400 {
			t.Fatalf("invalid=%d %s", w.Code, w.Body.String())
		}
	}
}

func TestPublicProblemDetailsAndUntruncatedRequest(t *testing.T) {
	f := newPublicFixture(t)
	body := `{"environment":"stg","instance":{"name":"auth-01","address":"auth-host","description":"` + strings.Repeat("x", 80*1024) + `"},"endpoints":[{"name":"default","protocol":"http","port":81}]}`
	out := requireRegistration(t, f.request("POST", registerRoute, body))
	if len(out.Instance.Description) != 80*1024 {
		t.Fatal("audit truncated application request")
	}
	for _, body := range []string{`{"environment":"stg","instance":{"name":"auth-01","address":"http://bad"},"endpoints":[{"name":"default","port":0}]}`, `{"environment":"missing","instance":{"name":"auth-01"}}`} {
		w := f.request("POST", registerRoute, body)
		if w.Header().Get("Content-Type") != "application/problem+json" {
			t.Fatal("wrong problem media type")
		}
		var details map[string]any
		if json.Unmarshal(w.Body.Bytes(), &details) != nil || details["code"] == nil || details["status"] == nil {
			t.Fatal("invalid problem")
		}
	}
	request := httptest.NewRequest("GET", "/api/v1/services", nil)
	w := httptest.NewRecorder()
	f.mux.ServeHTTP(w, request)
	if w.Code != 401 || w.Header().Get("Content-Type") != "application/problem+json" {
		t.Fatalf("auth error=%d %s", w.Code, w.Body.String())
	}
	if _, err := f.db.Exec(context.Background(), `ALTER TABLE services RENAME TO unavailable_services`); err != nil {
		t.Fatal(err)
	}
	w = f.request("GET", "/api/v1/services", "")
	if w.Code != 500 || strings.Contains(strings.ToLower(w.Body.String()), "sql") || strings.Contains(w.Body.String(), "no such table") {
		t.Fatalf("storage fault=%d %s", w.Code, w.Body.String())
	}
}
