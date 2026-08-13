package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/alerts"
	"github.com/company/service-registry/internal/api"
	"github.com/company/service-registry/internal/config"
	"github.com/company/service-registry/internal/health"
	"github.com/company/service-registry/internal/storage"
)

// Server represents the registry HTTP server
type Server struct {
	db      *storage.Database
	cfg     *config.Config
	webFS   fs.FS
	server  *http.Server
	listen  net.Listener
	monitor *health.Monitor
}

// New creates a new registry server
func New(db *storage.Database, cfg *config.Config, webFS fs.FS) *Server {
	return &Server{
		db:    db,
		cfg:   cfg,
		webFS: webFS,
	}
}

// Start starts the server
func (s *Server) Start() error {
	// Create HTTP mux
	mux := http.NewServeMux()

	// Register health endpoints
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/readyz", s.readyz)
	mux.HandleFunc("/version", s.version)
	mux.HandleFunc("/metrics", s.metrics)

	if s.webFS != nil {
		fileServer := http.FileServer(http.FS(s.webFS))
		mux.Handle("/", fileServer)
	}

	// API routes.
	api.RegisterRoutesWithConfig(mux, s.db, s.cfg)

	// Create HTTP server
	s.server = &http.Server{
		Addr:           fmt.Sprintf("%s:%d", s.cfg.Server.Address, s.cfg.Server.Port),
		Handler:        mux,
		ReadTimeout:    s.cfg.Server.ReadTimeout,
		WriteTimeout:   s.cfg.Server.WriteTimeout,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Listen
	listener, err := net.Listen("tcp", s.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.server.Addr, err)
	}
	s.listen = listener

	alertEngine := alerts.NewEngine(storage.NewAlertRepository(s.db), nil)
	healthRepo := storage.NewHealthRepositoryWithAlerts(s.db, alertEngine)
	s.monitor = health.NewMonitor(healthStore{repo: healthRepo}, health.NewExecutor(nil), health.MonitorConfig{
		WorkerCount:   s.cfg.Health.WorkerCount,
		CheckInterval: s.cfg.Health.CheckInterval,
		QueueCapacity: s.cfg.Health.CheckQueueCapacity,
		BatchSize:     s.cfg.Health.CheckQueueCapacity,
	})
	s.monitor.Start(context.Background())
	defer s.monitor.Stop()
	if s.cfg.Health.ResultRetentionHours > 0 {
		pruned, err := healthRepo.PruneHealthResults(context.Background(), time.Now().UTC().Add(-time.Duration(s.cfg.Health.ResultRetentionHours)*time.Hour))
		if err != nil {
			slog.Warn("Failed to prune health results", slog.Any("error", err))
		} else if pruned > 0 {
			slog.Info("Pruned old health results", slog.Int64("count", pruned))
		}
	}

	slog.Info("Server listening", slog.String("address", s.server.Addr))

	// Start server (blocks until shutdown)
	if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.monitor != nil {
		s.monitor.Stop()
	}
	if s.server == nil {
		return nil
	}
	slog.Info("Shutting down server...")
	return s.server.Shutdown(ctx)
}

// healthz returns 200 if the server is alive
func (s *Server) healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ok","timestamp":"%s"}`, time.Now().UTC().Format(time.RFC3339))
}

// readyz returns 200 if the server is ready to serve traffic
func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	err := s.db.GetDB().PingContext(ctx)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not_ready","reason":"database_unavailable"}`)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"ready"}`)
}

// version returns version information
func (s *Server) version(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"version":"0.1.0","commit":"dev"}`)
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	dbUp := 1
	if err := s.db.GetDB().PingContext(ctx); err != nil {
		dbUp = 0
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP registry_up Registry process liveness.\n")
	fmt.Fprintf(w, "# TYPE registry_up gauge\n")
	fmt.Fprintf(w, "registry_up 1\n")
	fmt.Fprintf(w, "# HELP registry_database_up Registry database readiness.\n")
	fmt.Fprintf(w, "# TYPE registry_database_up gauge\n")
	fmt.Fprintf(w, "registry_database_up %d\n", dbUp)
	fmt.Fprintf(w, "# HELP registry_health_worker_count Configured health check worker count.\n")
	fmt.Fprintf(w, "# TYPE registry_health_worker_count gauge\n")
	fmt.Fprintf(w, "registry_health_worker_count %d\n", s.cfg.Health.WorkerCount)
}

type healthStore struct {
	repo *storage.HealthRepository
}

func (s healthStore) ListDueHealthCheckTargets(ctx context.Context, limit int) ([]health.Target, error) {
	targets, err := s.repo.ListDueHealthCheckTargets(ctx, limit)
	if err != nil {
		return nil, err
	}
	mapped := make([]health.Target, 0, len(targets))
	for _, target := range targets {
		mapped = append(mapped, health.Target{
			Check:   target.Check,
			Address: target.Address,
			Port:    target.Port,
			Path:    target.Path,
		})
	}
	return mapped, nil
}

func (s healthStore) RecordHealthResult(ctx context.Context, check *registryv1.HealthCheck, result *registryv1.HealthResult) (*registryv1.HealthStateView, error) {
	return s.repo.RecordHealthResult(ctx, check, result)
}
