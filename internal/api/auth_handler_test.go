package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
)

func TestAuthLoginRouteReturnsToken(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	repo := auth.NewRepository(db)
	_, err := repo.CreateUser(ctx, auth.CreateUserInput{
		Username:    "admin",
		Email:       "admin@example.test",
		DisplayName: "Admin",
		Password:    "password",
		Role:        auth.RoleAdministrator,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	mux := http.NewServeMux()
	registerAuthREST(mux, authHandler{service: auth.NewService(repo, time.Hour), repo: repo})

	body := bytes.NewBufferString(`{"username":"admin","password":"password"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Token == "" {
		t.Fatalf("expected token in response")
	}
}

func newAPITestDB(t *testing.T, ctx context.Context) *storage.Database {
	t.Helper()
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
