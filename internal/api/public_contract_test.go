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
	body := []byte(`{"environment":"stg","instance":{"name":"auth-01","address":"lynx-authentication.lynx"},"endpoints":[{"name":"default","protocol":"http","port":81,"path":"/","primary":true}]}`)
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
	if len(discovery.Instances) != 1 || len(discovery.Instances[0].Endpoints) != 1 || discovery.Instances[0].Endpoints[0].Address != "http://lynx-authentication.lynx:81/" {
		t.Fatalf("discovery=%+v", discovery)
	}
	res = httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/api/v1/discovery/Authentication.Grpc/resolve?environment=stg", nil))
	if res.Code != http.StatusOK || !bytes.Contains(res.Body.Bytes(), []byte(`"address":"http://lynx-authentication.lynx:81/"`)) {
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

func TestPublicRegistrationNormalizesConfluentCloudAddress(t *testing.T) {
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
	body := []byte(`{"environment":"stg","instance":{"name":"broker","address":"https://pkc-lgk0v.us-west1.gcp.confluent.cloud:9092/"},"endpoints":[{"name":"default","protocol":"tcp","port":9092}]}`)
	res := httptest.NewRecorder()
	mux.ServeHTTP(res, httptest.NewRequest(http.MethodPost, "/api/v1/services/Kafka/instances", bytes.NewReader(body)))
	if res.Code != http.StatusOK {
		t.Fatalf("register status=%d body=%s", res.Code, res.Body.String())
	}
	var registered publicRegistrationResponse
	if err := json.Unmarshal(res.Body.Bytes(), &registered); err != nil {
		t.Fatal(err)
	}
	if registered.Instance.Address != "pkc-lgk0v.us-west1.gcp.confluent.cloud" {
		t.Fatalf("address = %q, want normalized Confluent hostname", registered.Instance.Address)
	}
}
