package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

func TestPublicRegistrationAndDiscoveryUseHumanReadableKeys(t *testing.T) {
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, fixtureDatabasePath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	env, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "stg", Name: "Staging"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = storage.NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "Authentication.Grpc", DisplayName: "Authentication"})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	body := []byte(`{"environment":"stg","instance":{"name":"auth-01","address":"lynx-authentication.lynx"},"endpoints":[{"name":"default","kind":"http","port":81,"path":"/","primary":true}]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/services/Authentication.Grpc/instances", bytes.NewReader(body))
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", res.Code, res.Body.String())
	}
	var registered map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if registered["service"] != "Authentication.Grpc" || registered["environment"] != "stg" {
		t.Fatalf("registration=%v", registered)
	}

	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/services/Authentication.Grpc/instances", bytes.NewReader(body)))
	if res.Code != http.StatusOK {
		t.Fatalf("repeat registration status=%d body=%s", res.Code, res.Body.String())
	}
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/discovery/Authentication.Grpc?environment=stg", nil))
	if res.Code != http.StatusOK {
		t.Fatalf("discovery status=%d body=%s", res.Code, res.Body.String())
	}
	var discovery publicDiscoveryResponse
	if err := json.Unmarshal(res.Body.Bytes(), &discovery); err != nil {
		t.Fatal(err)
	}
	if len(discovery.Instances) != 1 || len(discovery.Instances[0].Endpoints) != 1 || discovery.Instances[0].Endpoints[0].Address != "lynx-authentication.lynx" || discovery.Instances[0].Endpoints[0].Port != 81 {
		t.Fatalf("discovery=%+v", discovery)
	}
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/discovery/Authentication.Grpc/resolve?environment=stg", nil))
	if res.Code != http.StatusOK || !bytes.Contains(res.Body.Bytes(), []byte(`"address":"lynx-authentication.lynx"`)) || !bytes.Contains(res.Body.Bytes(), []byte(`"port":81`)) {
		t.Fatalf("resolve status=%d body=%s", res.Code, res.Body.String())
	}
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodDelete, "/api/v1/services/Authentication.Grpc/instances/auth-01?environment=stg", nil))
	if res.Code != http.StatusNoContent {
		t.Fatalf("deregister status=%d body=%s", res.Code, res.Body.String())
	}
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/services/Authentication.Grpc/instances", bytes.NewReader(body)))
	if res.Code != http.StatusOK {
		t.Fatalf("restore registration status=%d body=%s", res.Code, res.Body.String())
	}
	_ = env
}

func TestPublicRegistrationRejectsSchemeInAddress(t *testing.T) {
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, fixtureDatabasePath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "stg", Name: "Staging"}); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "Kafka", DisplayName: "Kafka"}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	body := []byte(`{"environment":"stg","instance":{"name":"broker","address":"https://pkc-lgk0v.us-west1.gcp.confluent.cloud:9092/"},"endpoints":[{"name":"default","kind":"tcp","port":9092}]}`)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/services/Kafka/instances", bytes.NewReader(body)))
	if res.Code != http.StatusBadRequest || !bytes.Contains(res.Body.Bytes(), []byte(`instance.address`)) {
		t.Fatalf("register status=%d body=%s", res.Code, res.Body.String())
	}
}

func TestPublicPostgresDiscoveryIsStructured(t *testing.T) {
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, fixtureDatabasePath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "development", Name: "Development"}); err != nil {
		t.Fatal(err)
	}
	if _, err := storage.NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "postgres", DisplayName: "PostgreSQL"}); err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	body := []byte(`{"environment":"development","instance":{"name":"postgres-01","address":"192.168.0.109"},"endpoints":[{"name":"default","kind":"postgres","port":5432,"primary":true}]}`)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/services/postgres/instances", bytes.NewReader(body)))
	if res.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", res.Code, res.Body.String())
	}
	var registered publicRegistrationResponse
	if err := json.Unmarshal(res.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if registered.Endpoints[0].Kind != "POSTGRES" || registered.Endpoints[0].Address != "192.168.0.109" || registered.Endpoints[0].Path != "" {
		t.Fatalf("registered endpoint=%+v", registered.Endpoints[0])
	}
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/discovery/postgres/resolve?environment=development", nil))
	if res.Code != http.StatusOK || !bytes.Contains(res.Body.Bytes(), []byte(`"kind":"POSTGRES"`)) || !bytes.Contains(res.Body.Bytes(), []byte(`"address":"192.168.0.109"`)) || !bytes.Contains(res.Body.Bytes(), []byte(`"port":5432`)) {
		t.Fatalf("resolve status=%d body=%s", res.Code, res.Body.String())
	}
	bad := []byte(`{"environment":"development","instance":{"name":"postgres-02","address":"192.168.0.110"},"endpoints":[{"name":"default","kind":"postgres","port":5432,"path":"/health"}]}`)
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/services/postgres/instances", bytes.NewReader(bad)))
	if res.Code != http.StatusBadRequest || !bytes.Contains(res.Body.Bytes(), []byte(`endpoints[0].path`)) {
		t.Fatalf("bad path status=%d body=%s", res.Code, res.Body.String())
	}
	for _, item := range []struct {
		name string
		body []byte
		want []byte
	}{
		{"old protocol", []byte(`{"environment":"development","instance":{"name":"postgres-03","address":"192.168.0.111"},"endpoints":[{"name":"default","protocol":"tcp","port":5432}]}`), []byte("validation_failed")},
		{"missing kind", []byte(`{"environment":"development","instance":{"name":"postgres-04","address":"192.168.0.112"},"endpoints":[{"name":"default","port":5432}]}`), []byte("endpoint kind")},
		{"invalid kind", []byte(`{"environment":"development","instance":{"name":"postgres-05","address":"192.168.0.113"},"endpoints":[{"name":"default","kind":"smtp","port":25}]}`), []byte("endpoints[0].kind")},
	} {
		res = httptest.NewRecorder()
		mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/services/postgres/instances", bytes.NewReader(item.body)))
		if res.Code != http.StatusBadRequest || !bytes.Contains(res.Body.Bytes(), item.want) {
			t.Fatalf("%s status=%d body=%s", item.name, res.Code, res.Body.String())
		}
	}
}
