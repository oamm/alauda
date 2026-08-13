package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/company/service-registry/internal/storage"
)

func TestMiddlewareRequiresBearerTokenWhenEnabled(t *testing.T) {
	handler := Middleware(true, NewService(NewRepository(newMiddlewareTestDB(t)), time.Hour), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/registry.v1.CatalogService/CreateService", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestMiddlewareEnforcesScopes(t *testing.T) {
	ctx := context.Background()
	db := newMiddlewareTestDB(t)
	repo := NewRepository(db)
	user, err := repo.CreateUser(ctx, CreateUserInput{
		Username:    "viewer",
		Email:       "viewer@example.test",
		DisplayName: "Viewer",
		Password:    "password",
		Role:        RoleViewer,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	token, err := repo.CreateToken(ctx, CreateTokenInput{
		UserID:    user.ID,
		Name:      "viewer",
		Scopes:    []Scope{ScopeRead},
		CreatedBy: user.ID,
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	handler := Middleware(true, NewService(repo, time.Hour), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodPost, "/registry.v1.CatalogService/CreateService", nil)
	req.Header.Set("Authorization", "Bearer "+token.Secret)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func newMiddlewareTestDB(t *testing.T) *storage.Database {
	t.Helper()
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, ":memory:")
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}
