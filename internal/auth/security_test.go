package auth

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/company/service-registry/internal/storage"
)

func TestBootstrapCreatesRootOnceWithRestrictedCredentialFile(t *testing.T) {
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, filepath.Join(t.TempDir(), "registry.db"))
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
	databasePath := filepath.Join(t.TempDir(), "registry.db")
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
