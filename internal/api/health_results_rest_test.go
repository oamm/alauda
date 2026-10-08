package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/company/service-registry/internal/config"
	"github.com/company/service-registry/internal/storage"
)

func TestHealthResultsRESTValidation(t *testing.T) {
	db := newAPITestDB(t, context.Background())
	mux := http.NewServeMux()
	registerHealthResultsREST(mux, storage.NewHealthRepository(db))
	for _, query := range []string{"from=bad", "from=2026-10-08T00:00:00Z&to=2026-10-07T00:00:00Z", "port=0", "port=65536", "port=1.2", "pageSize=101", "pageToken=-1", "sort=wrong", "status=unknown", "type=unsupported"} {
		t.Run(query, func(t *testing.T) {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/health/results?"+query, nil))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status %d, body %s", w.Code, w.Body.String())
			}
		})
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/health/results?pageSize=25&sort=oldest", nil))
	if w.Code != 200 || w.Body.String() != "{\"results\":[],\"nextPageToken\":\"\"}\n" {
		t.Fatalf("empty response: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/health/results", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status %d", w.Code)
	}
}

func TestHealthResultsRESTRequiresAuthentication(t *testing.T) {
	db := newAPITestDB(t, context.Background())
	mux := http.NewServeMux()
	RegisterRoutesWithConfig(mux, db, &config.Config{})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/health/results", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated history status %d", w.Code)
	}
}
