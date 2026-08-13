package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/company/service-registry/internal/config"
	"github.com/company/service-registry/internal/storage"
)

func TestHealthzVersionAndMetricsHandlers(t *testing.T) {
	s := newTestServer(t)
	defer s.db.Close()

	tests := []struct {
		name        string
		handler     http.HandlerFunc
		contentType string
		body        string
	}{
		{name: "healthz", handler: s.healthz, contentType: "application/json", body: `"status":"ok"`},
		{name: "version", handler: s.version, contentType: "application/json", body: `"version":"0.1.0"`},
		{name: "metrics", handler: s.metrics, contentType: "text/plain; version=0.0.4", body: "registry_database_up 1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.handler(rec, httptest.NewRequest(http.MethodGet, "/"+tc.name, nil))
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != tc.contentType {
				t.Fatalf("content-type = %q, want %q", got, tc.contentType)
			}
			if !strings.Contains(rec.Body.String(), tc.body) {
				t.Fatalf("body = %q, want substring %q", rec.Body.String(), tc.body)
			}
		})
	}
}

func TestReadyzReportsDatabaseReadiness(t *testing.T) {
	s := newTestServer(t)
	defer s.db.Close()

	rec := httptest.NewRecorder()
	s.readyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ready"`) {
		t.Fatalf("body = %q, want ready status", rec.Body.String())
	}
}

func TestReadyzReportsUnavailableDatabase(t *testing.T) {
	s := newTestServer(t)
	if err := s.db.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	rec := httptest.NewRecorder()
	s.readyz(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
	if !strings.Contains(rec.Body.String(), `"database_unavailable"`) {
		t.Fatalf("body = %q, want database_unavailable", rec.Body.String())
	}
}

func TestShutdownWithoutStartedHTTPServerStopsMonitorOnly(t *testing.T) {
	s := &Server{}
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v", err)
	}
}

func TestStartServesHealthzAndShutsDown(t *testing.T) {
	port := freeTCPPort(t)
	s := newTestServer(t)
	defer s.db.Close()
	s.cfg.Server.Address = "127.0.0.1"
	s.cfg.Server.Port = port
	s.cfg.Server.ReadTimeout = time.Second
	s.cfg.Server.WriteTimeout = time.Second

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.Start()
	}()

	url := "http://127.0.0.1:" + strconv.Itoa(port) + "/healthz"
	var resp *http.Response
	var err error
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := s.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start() returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatalf("server did not stop after shutdown")
	}
}

func newTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := storage.NewDatabase(context.Background(), ":memory:")
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	if err := storage.RunMigrations(context.Background(), db); err != nil {
		db.Close()
		t.Fatalf("run migrations: %v", err)
	}
	cfg := &config.Config{
		Health: config.HealthConfig{
			WorkerCount:          2,
			CheckInterval:        time.Second,
			CheckQueueCapacity:   10,
			ResultRetentionHours: 24,
		},
	}
	return New(db, cfg, nil)
}

func freeTCPPort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on ephemeral port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}
