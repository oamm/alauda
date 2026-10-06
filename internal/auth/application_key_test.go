package auth

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestApplicationKeyLifecycle(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	admin, err := repo.CreateUser(ctx, CreateUserInput{Username: "admin", Email: "admin@example.test", DisplayName: "Admin", Password: "password", Role: RoleAdministrator})
	if err != nil {
		t.Fatalf("create admin: %v", err)
	}
	service := NewService(repo, 0)
	created, err := service.CreateApplicationKey(ctx, &Principal{UserID: admin.ID, Scopes: []Scope{ScopeAdmin}}, CreateApplicationKeyInput{Name: "external-client"})
	if err != nil {
		t.Fatalf("create application key: %v", err)
	}
	if !strings.HasPrefix(created.Secret, "ak_") {
		t.Fatalf("secret prefix = %q, want ak_", created.Secret)
	}
	encoded, err := json.Marshal(created.Key)
	if err != nil {
		t.Fatalf("marshal application key: %v", err)
	}
	if strings.Contains(string(encoded), created.Key.SecretHash) {
		t.Fatal("application key hash was exposed")
	}

	principal, err := service.AuthenticateApplicationKey(ctx, created.Secret)
	if err != nil {
		t.Fatalf("authenticate application key: %v", err)
	}
	if principal.ApplicationKeyID != created.Key.ID || !HasScope(principal.Scopes, ScopeRead) {
		t.Fatalf("principal = %+v, want application key read principal", principal)
	}

	if err := repo.RevokeApplicationKey(ctx, created.Key.ID); err != nil {
		t.Fatalf("revoke application key: %v", err)
	}
	if _, err := service.AuthenticateApplicationKey(ctx, created.Secret); err == nil {
		t.Fatal("revoked application key authenticated")
	}
}
