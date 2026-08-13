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

func TestRepositoryListUsersAndRejectsInvalidRole(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)

	if _, err := repo.CreateUser(ctx, CreateUserInput{
		Username:    "bad",
		Email:       "bad@example.test",
		DisplayName: "Bad",
		Password:    "password",
		Role:        Role("Root"),
	}); err == nil {
		t.Fatalf("expected invalid role error")
	}
	for _, input := range []CreateUserInput{
		{Username: "viewer", Email: "viewer@example.test", DisplayName: "Viewer", Password: "password", Role: RoleViewer, Tags: map[string]string{"team": "docs"}},
		{Username: "ops", Email: "ops@example.test", DisplayName: "Ops", Password: "password", Role: RoleOperator},
	} {
		if _, err := repo.CreateUser(ctx, input); err != nil {
			t.Fatalf("create user %s: %v", input.Username, err)
		}
	}
	users, err := repo.ListUsers(ctx)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("users = %d, want 2", len(users))
	}
	if users[0].Username != "ops" || users[1].Username != "viewer" {
		t.Fatalf("users order = %s, %s, want display-name order", users[0].Username, users[1].Username)
	}
	if users[1].Tags["team"] != "docs" {
		t.Fatalf("user tags = %#v, want team docs", users[1].Tags)
	}
}

func TestServiceLoginRejectsDisabledUserAndBadPassword(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	service := NewService(repo, time.Hour)
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
	if _, _, err := service.Login(ctx, user.Username, "wrong"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("bad password error = %v, want ErrInvalidCredentials", err)
	}
	if _, err := db.Exec(ctx, `UPDATE users SET enabled = 0 WHERE id = ?`, user.ID); err != nil {
		t.Fatalf("disable user: %v", err)
	}
	if _, _, err := service.Login(ctx, user.Username, "password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("disabled user error = %v, want ErrInvalidCredentials", err)
	}
}

func TestRepositoryTokenInputValidationAndAuthenticationErrors(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	service := NewService(repo, time.Hour)
	if _, err := repo.CreateToken(ctx, CreateTokenInput{UserID: "missing", Name: "bad", Scopes: []Scope{ScopeRead}}); err == nil {
		t.Fatalf("expected foreign key error for missing user")
	}
	if _, err := service.AuthenticateToken(ctx, ""); err == nil {
		t.Fatalf("expected empty token authentication error")
	}
	if _, err := service.AuthenticateToken(ctx, "sr_missing"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("missing token error = %v, want sql.ErrNoRows", err)
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
