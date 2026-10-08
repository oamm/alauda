package auth

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/company/service-registry/internal/storage"
	"github.com/company/service-registry/internal/testutil"
)

func TestBootstrapCreatesRootOnceWithRestrictedCredentialFile(t *testing.T) {
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, testutil.DatabasePath(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(db)
	credentialPath := filepath.Join(t.TempDir(), "bootstrap", "credential")
	result, err := repo.Bootstrap(ctx, "root", "root@example.test", credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || result.Username != "root" {
		t.Fatalf("unexpected bootstrap result: %#v", result)
	}
	user, err := repo.GetUserByUsername(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if user.Role != RoleAdministrator || !user.MustChangePassword || strings.Contains(user.PasswordHash, "sr_") {
		t.Fatalf("unsafe root state: %#v", user)
	}
	credential, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(credential), "username: root\n") {
		t.Fatalf("unexpected credential file: %q", credential)
	}
	mode := func() os.FileMode { info, _ := os.Stat(credentialPath); return info.Mode().Perm() }()
	if runtime.GOOS != "windows" && mode != 0o600 {
		t.Fatalf("credential mode = %o, want 600", mode)
	}
	second, err := repo.Bootstrap(ctx, "root", "root@example.test", credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if second.Created {
		t.Fatal("bootstrap recreated root")
	}
	users, err := repo.ListUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("users = %d, want 1", len(users))
	}
}

func TestBootstrapConcurrentStartsCreateOneRoot(t *testing.T) {
	ctx := context.Background()
	databasePath := testutil.DatabasePath(t)
	first, err := storage.NewDatabase(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.RunMigrations(ctx, first); err != nil {
		t.Fatal(err)
	}
	first.Close()
	credentialPath := filepath.Join(t.TempDir(), "credential")
	results := make([]*BootstrapResult, 2)
	errors := make([]error, 2)
	var wg sync.WaitGroup
	for index := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			db, openErr := storage.NewDatabase(ctx, databasePath)
			if openErr != nil {
				errors[index] = openErr
				return
			}
			defer db.Close()
			results[index], errors[index] = NewRepository(db).Bootstrap(ctx, "root", "root@example.test", credentialPath)
		}(index)
	}
	wg.Wait()
	for _, err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	created := 0
	for _, result := range results {
		if result.Created {
			created++
		}
	}
	if created != 1 {
		t.Fatalf("bootstrap creators = %d, want 1", created)
	}
}

func TestBootstrapWithExistingUserMarksInitializedWithoutCreatingRoot(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	if _, err := repo.CreateUser(ctx, CreateUserInput{Username: "existing", Email: "existing@example.test", DisplayName: "Existing", Password: "password", Role: RoleViewer}); err != nil {
		t.Fatal(err)
	}

	result, err := repo.Bootstrap(ctx, "root", "root@example.test", filepath.Join(t.TempDir(), "credential"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Created {
		t.Fatal("bootstrap created root for existing installation")
	}
	if _, err := repo.GetUserByUsername(ctx, "root"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("root lookup error = %v, want sql.ErrNoRows", err)
	}
}

func TestResetDevelopmentBootstrapRotatesCredentialAndRevokesSessions(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	user, err := repo.CreateUser(ctx, CreateUserInput{Username: "root", Email: "root@example.test", DisplayName: "Root", Password: "old-password", Role: RoleAdministrator})
	if err != nil {
		t.Fatal(err)
	}
	session, err := repo.CreateSession(ctx, user.ID, time.Hour, "unit-test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	credentialPath := filepath.Join(t.TempDir(), "bootstrap", "credential")
	if err := os.WriteFile(credentialPath, []byte("stale"), 0o600); err == nil {
		t.Fatal("expected write to missing credential directory to fail")
	}

	result, err := repo.ResetDevelopmentBootstrap(ctx, "root", credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created || result.Username != "root" || result.CredentialPath != credentialPath {
		t.Fatalf("reset result = %#v", result)
	}
	credential, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(credential), "username: root\n") {
		t.Fatalf("credential = %q", credential)
	}
	updated, err := repo.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.MustChangePassword || CheckPassword(updated.PasswordHash, "old-password") {
		t.Fatalf("password was not rotated: %#v", updated)
	}
	if _, _, err := repo.FindSessionBySecret(ctx, session.Secret); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("session lookup error = %v, want sql.ErrNoRows", err)
	}
}

func TestBootstrapCredentialFilesAreValidatedAndReused(t *testing.T) {
	dir := t.TempDir()
	validPath := filepath.Join(dir, "valid")
	if err := os.WriteFile(validPath, []byte("username: root\nsecret-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret, err := bootstrapCredential(validPath, "root")
	if err != nil {
		t.Fatal(err)
	}
	if secret != "secret-value" {
		t.Fatalf("secret = %q, want secret-value", secret)
	}
	if _, err := bootstrapCredential(validPath, "other"); err == nil {
		t.Fatal("bootstrapCredential accepted mismatched username")
	}

	concurrentPath := filepath.Join(dir, "concurrent")
	if err := os.WriteFile(concurrentPath, []byte("username: root\nconcurrent-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secret, err = newBootstrapCredential(concurrentPath, "root")
	if err != nil {
		t.Fatal(err)
	}
	if secret != "concurrent-secret" {
		t.Fatalf("concurrent secret = %q, want concurrent-secret", secret)
	}
}

func TestSessionRepositoryLifecycle(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	user, err := repo.CreateUser(ctx, CreateUserInput{Username: "admin", Email: "admin@example.test", DisplayName: "Admin", Password: "password", Role: RoleAdministrator})
	if err != nil {
		t.Fatal(err)
	}
	created, err := repo.CreateSession(ctx, user.ID, time.Hour, "unit-test", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	session, sessionUser, err := repo.FindSessionBySecret(ctx, created.Secret)
	if err != nil {
		t.Fatal(err)
	}
	if session.ID != created.Session.ID || sessionUser.ID != user.ID {
		t.Fatalf("session/user = %#v/%#v", session, sessionUser)
	}
	sessions, err := repo.ListSessions(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != created.Session.ID {
		t.Fatalf("sessions = %#v", sessions)
	}
	if err := repo.RevokeSession(ctx, created.Session.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.RevokeSession(ctx, created.Session.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("second revoke error = %v, want sql.ErrNoRows", err)
	}
}

func TestPasswordChangeClearsTemporaryState(t *testing.T) {
	ctx := context.Background()
	db := newAuthTestDB(t, ctx)
	repo := NewRepository(db)
	user, err := repo.CreateUser(ctx, CreateUserInput{Username: "root", Email: "root@example.test", DisplayName: "Root", Password: "temporary-password", Role: RoleAdministrator, MustChangePassword: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.ChangePassword(ctx, user.ID, "new secure password"); err != nil {
		t.Fatal(err)
	}
	updated, err := repo.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.MustChangePassword || !CheckPassword(updated.PasswordHash, "new secure password") {
		t.Fatalf("password state not updated: %#v", updated)
	}
}
