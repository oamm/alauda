package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/company/service-registry/internal/storage"
)

func TestRepositoryUserLoginAndToken(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	service := NewService(repo, time.Hour)

	user, err := repo.CreateUser(ctx, CreateUserInput{
		Username:    "admin",
		Email:       "admin@example.test",
		DisplayName: "Admin",
		Password:    "correct horse battery staple",
		Role:        RoleAdministrator,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if user.PasswordHash != "" && user.PasswordHash == "correct horse battery staple" {
		t.Fatalf("password was not hashed")
	}

	created, loggedIn, err := service.Login(ctx, "admin", "correct horse battery staple")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if loggedIn.ID != user.ID {
		t.Fatalf("logged in user mismatch")
	}
	if created.Secret == "" {
		t.Fatalf("expected token secret")
	}
	encoded, err := json.Marshal(created.Token)
	if err != nil {
		t.Fatalf("marshal token: %v", err)
	}
	if string(encoded) == "" || strings.Contains(string(encoded), created.Token.TokenHash) {
		t.Fatalf("token hash was exposed in json: %s", encoded)
	}

	principal, err := service.AuthenticateToken(ctx, created.Secret)
	if err != nil {
		t.Fatalf("authenticate token: %v", err)
	}
	if principal.UserID != user.ID || !HasScope(principal.Scopes, ScopeAdmin) {
		t.Fatalf("unexpected principal: %#v", principal)
	}
}

func TestRepositoryRejectsExpiredToken(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
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
	expiresAt := time.Now().UTC().Add(-time.Minute)
	created, err := repo.CreateToken(ctx, CreateTokenInput{
		UserID:    user.ID,
		Name:      "expired",
		Scopes:    []Scope{ScopeRead},
		ExpiresAt: &expiresAt,
		CreatedBy: user.ID,
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	_, _, err = repo.FindTokenBySecret(ctx, created.Secret)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected sql.ErrNoRows for expired token, got %v", err)
	}
}

func TestRepositoryListsAndRevokesTokens(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)

	user, err := repo.CreateUser(ctx, CreateUserInput{
		Username:    "operator",
		Email:       "operator@example.test",
		DisplayName: "Operator",
		Password:    "password",
		Role:        RoleOperator,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	created, err := repo.CreateToken(ctx, CreateTokenInput{
		UserID:         user.ID,
		Name:           "ci",
		Scopes:         []Scope{ScopeRead, ScopeWrite},
		EnvironmentIDs: []string{"prod"},
		CreatedBy:      user.ID,
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}

	tokens, err := repo.ListTokens(ctx, user.ID)
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(tokens) != 1 || tokens[0].ID != created.Token.ID || tokens[0].EnvironmentIDs[0] != "prod" {
		t.Fatalf("unexpected tokens: %#v", tokens)
	}

	if err := repo.RevokeToken(ctx, created.Token.ID); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	_, _, err = repo.FindTokenBySecret(ctx, created.Secret)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected revoked token to be rejected, got %v", err)
	}
}

func TestRepositoryBootstrapAdminCreatesTokenOnce(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	input := CreateUserInput{
		Username:    "admin",
		Email:       "admin@example.test",
		DisplayName: "Admin",
		Password:    "password",
		Role:        RoleAdministrator,
	}

	created, err := repo.BootstrapAdmin(ctx, input, "bootstrap", "sr_fixed")
	if err != nil {
		t.Fatalf("bootstrap admin: %v", err)
	}
	if created == nil || created.Secret != "sr_fixed" {
		t.Fatalf("expected fixed bootstrap token, got %#v", created)
	}
	created, err = repo.BootstrapAdmin(ctx, input, "bootstrap", "sr_fixed")
	if err != nil {
		t.Fatalf("bootstrap admin second run: %v", err)
	}
	if created != nil {
		t.Fatalf("expected second bootstrap to be idempotent")
	}
}

func TestServiceCreateTokenRequiresAdminScopeAndDefaultsRead(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	service := NewService(repo, time.Hour)
	user, err := repo.CreateUser(ctx, CreateUserInput{
		Username:    "admin",
		Email:       "admin@example.test",
		DisplayName: "Admin",
		Password:    "password",
		Role:        RoleAdministrator,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	_, err = service.CreateToken(ctx, &Principal{UserID: user.ID, Scopes: []Scope{ScopeRead}}, CreateTokenInput{
		UserID: user.ID,
		Name:   "denied",
	})
	if err == nil {
		t.Fatalf("expected non-admin token creation to fail")
	}

	created, err := service.CreateToken(ctx, &Principal{UserID: user.ID, Scopes: []Scope{ScopeAdmin}}, CreateTokenInput{
		UserID: user.ID,
		Name:   "default-read",
	})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if len(created.Token.Scopes) != 1 || created.Token.Scopes[0] != ScopeRead || created.Token.CreatedBy != user.ID {
		t.Fatalf("unexpected created token: %#v", created.Token)
	}
}

func newAuthTestDB(t *testing.T, ctx context.Context) *storage.Database {
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
