package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
)

func TestHealthChecksRESTValidationAndEmptyResult(t *testing.T) {
	db := newAPITestDB(t, context.Background())
	mux := http.NewServeMux()
	registerHealthChecksREST(mux, storage.NewHealthRepository(db))

	for _, query := range []string{"page=0", "page=bad", "pageSize=10", "pageSize=bad", "enabled=maybe"} {
		t.Run(query, func(t *testing.T) {
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/health/checks?"+query, nil))
			if w.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusBadRequest, w.Body.String())
			}
		})
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/checks?search=ready&page=1&pageSize=25&enabled=false&latestStatus=healthy", nil)
	principal := &auth.Principal{EnvironmentIDs: []string{"env-1"}}
	mux.ServeHTTP(w, req.WithContext(auth.WithPrincipal(context.Background(), principal)))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", w.Code, http.StatusOK, w.Body.String())
	}

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/health/checks", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
