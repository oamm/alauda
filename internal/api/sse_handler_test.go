package api

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

func TestEventSSEStreamsEvents(t *testing.T) {
	testEventSSEStreamsEvents(t, func(handler http.Handler) http.Handler {
		return handler
	})
}

func TestEventSSEStreamsEventsThroughHardeningMiddleware(t *testing.T) {
	testEventSSEStreamsEvents(t, hardeningMiddleware)
}

func testEventSSEStreamsEvents(t *testing.T, wrap func(http.Handler) http.Handler) {
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, fixtureDatabasePath(t))
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	defer db.Close()
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	eventRepo := storage.NewEventRepository(db)
	if _, err := eventRepo.Create(ctx, &registryv1.Event{
		Type:         "test.event",
		ResourceType: "test",
		ResourceId:   "test-1",
		Actor:        "test",
		Message:      "test event emitted",
		Metadata:     map[string]string{},
	}); err != nil {
		t.Fatalf("create event: %v", err)
	}

	mux := http.NewServeMux()
	registerEventSSE(mux, eventRepo)
	finished := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { defer close(finished); wrap(mux).ServeHTTP(w, r) }))
	defer server.Close()
	defer func() {
		server.CloseClientConnections()
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			t.Error("stream handler did not finish")
		}
	}()

	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, server.URL+"/api/v1/events/watch?replay=true", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("watch events: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content type = %q, want text/event-stream", got)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") && strings.Contains(line, "test.event") {
			return
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan stream: %v", err)
	}
	t.Fatalf("stream ended before test event was emitted")
}
