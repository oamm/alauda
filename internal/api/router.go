package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/company/service-registry/internal/alerts"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/config"
	"github.com/company/service-registry/internal/storage"
)

// RegisterRoutes installs API endpoints onto the given mux.
func RegisterRoutes(mux *http.ServeMux, db *storage.Database) {
	RegisterRoutesWithConfig(mux, db, nil)
}

// RegisterRoutesWithConfig installs API endpoints with optional auth enforcement.
func RegisterRoutesWithConfig(mux *http.ServeMux, db *storage.Database, cfg *config.Config) {
	authEnabled := cfg != nil && cfg.Auth.Enabled && !cfg.Server.DevMode
	authRepo := auth.NewRepository(db)
	auditRepo := storage.NewAuditRepository(db)
	tokenTTL := 24 * time.Hour
	if cfg != nil && cfg.Auth.TokenTTL > 0 {
		tokenTTL = cfg.Auth.TokenTTL
	}
	authService := auth.NewService(authRepo, tokenTTL)
	if cfg != nil && cfg.Auth.Enabled && cfg.Auth.BootstrapAdminPassword != "" {
		created, err := authRepo.BootstrapAdmin(
			context.Background(),
			auth.CreateUserInput{
				Username:    cfg.Auth.BootstrapAdminUsername,
				Email:       cfg.Auth.BootstrapAdminEmail,
				DisplayName: "Administrator",
				Password:    cfg.Auth.BootstrapAdminPassword,
				Role:        auth.RoleAdministrator,
			},
			cfg.Auth.BootstrapAdminTokenName,
			cfg.Auth.BootstrapAdminToken,
		)
		if err != nil {
			slog.Error("Failed to bootstrap admin user", slog.String("error", err.Error()))
		} else if created != nil {
			slog.Info("Bootstrapped admin API token", slog.String("tokenName", cfg.Auth.BootstrapAdminTokenName))
		}
	}

	apiMux := http.NewServeMux()
	rateLimit := func(handler http.Handler) http.Handler { return handler }
	if cfg != nil {
		rateLimit = newRateLimitMiddleware(cfg.RateLimit)
	}
	secure := func(handler http.Handler) http.Handler {
		return hardeningMiddleware(rateLimit(auth.Middleware(authEnabled, authService, auth.AuditMiddleware(auditRepo, handler))))
	}
	RegisterConnectHandlersWithMiddleware(mux, db, secure)
	registerEventSSE(apiMux, storage.NewEventRepository(db))
	registerAlertREST(apiMux, storage.NewAlertRepository(db), alerts.NewEngine(storage.NewAlertRepository(db), nil))
	registerAuthREST(apiMux, authHandler{service: authService, repo: authRepo})
	registerAuditREST(apiMux, auditRepo)

	apiMux.HandleFunc("/api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.Handle("/api/v1/", secure(apiMux))
}
