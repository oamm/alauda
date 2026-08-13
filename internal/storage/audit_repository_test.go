package storage

import (
	"context"
	"testing"
	"time"
)

func TestAuditRepositoryCreateAndList(t *testing.T) {
	ctx := context.Background()
	db := newAuditTestDB(t, ctx)
	repo := NewAuditRepository(db)

	created, err := repo.Create(ctx, AuditLog{
		Timestamp:    time.Now().UTC(),
		Actor:        "admin",
		ActorID:      "user-1",
		Action:       "CreateService",
		ResourceType: "Catalog",
		ResourceID:   "svc-1",
		Changes:      `{"name":"api"}`,
		IP:           "127.0.0.1",
		UserAgent:    "test",
		Status:       "success",
		Metadata:     map[string]string{"path": "/registry.v1.CatalogService/CreateService"},
	})
	if err != nil {
		t.Fatalf("create audit log: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated id")
	}

	items, nextToken, err := repo.List(ctx, AuditFilters{Actor: "admin", ResourceType: "Catalog"}, 10, "")
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if nextToken != "" {
		t.Fatalf("unexpected next token: %q", nextToken)
	}
	if len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("unexpected audit logs: %#v", items)
	}
}

func newAuditTestDB(t *testing.T, ctx context.Context) *Database {
	t.Helper()
	db, err := NewDatabase(ctx, ":memory:")
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}
