package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/company/service-registry/internal/storage"
)

type memoryAuditLogger struct {
	entries []storage.AuditLog
}

func (l *memoryAuditLogger) Create(ctx context.Context, entry storage.AuditLog) (*storage.AuditLog, error) {
	l.entries = append(l.entries, entry)
	return &entry, nil
}

func TestAuditMiddlewareLogsMutationWithRedactedBody(t *testing.T) {
	logger := &memoryAuditLogger{}
	handler := AuditMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/users", strings.NewReader(`{"username":"admin","password":"secret"}`))
	req = req.WithContext(WithPrincipal(req.Context(), &Principal{UserID: "user-1", Username: "admin", Scopes: []Scope{ScopeAdmin}}))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if len(logger.entries) != 1 {
		t.Fatalf("logged entries = %d, want 1", len(logger.entries))
	}
	entry := logger.entries[0]
	if entry.Actor != "admin" || entry.Status != "success" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if strings.Contains(entry.Changes, "secret") || !strings.Contains(entry.Changes, "[REDACTED]") {
		t.Fatalf("body was not redacted: %s", entry.Changes)
	}
}

func TestAuditMiddlewareSkipsReads(t *testing.T) {
	logger := &memoryAuditLogger{}
	handler := AuditMiddleware(logger, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest(http.MethodGet, "/registry.v1.CatalogService/ListServices", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if len(logger.entries) != 0 {
		t.Fatalf("logged entries = %d, want 0", len(logger.entries))
	}
}
