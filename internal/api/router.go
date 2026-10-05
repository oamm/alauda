package api

import (
	"encoding/json"
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
	// A configured server is always protected; only the legacy nil-config test helper bypasses auth.
	authEnabled := cfg != nil && (cfg.Auth.Enabled || !cfg.Server.DevMode)
	authRepo := auth.NewRepository(db)
	auditRepo := storage.NewAuditRepository(db)
	tokenTTL := 24 * time.Hour
	if cfg != nil && cfg.Auth.TokenTTL > 0 {
		tokenTTL = cfg.Auth.TokenTTL
	}
	authService := auth.NewService(authRepo, tokenTTL)
	apiMux := http.NewServeMux()
	rateLimit := func(handler http.Handler) http.Handler { return handler }
	if cfg != nil {
		rateLimit = newRateLimitMiddleware(cfg.RateLimit)
	}
	cookieName := "alauda_session"
	if cfg != nil && cfg.Auth.SessionCookieName != "" {
		cookieName = cfg.Auth.SessionCookieName
	}
	secure := func(handler http.Handler) http.Handler {
		return hardeningMiddleware(rateLimit(auth.MiddlewareWithCookieName(authEnabled, authService, cookieName, auth.AuditMiddleware(auditRepo, handler))))
	}
	RegisterConnectHandlersWithMiddleware(mux, db, secure)
	registerEventSSE(apiMux, storage.NewEventRepository(db))
	registerAlertREST(apiMux, storage.NewAlertRepository(db), alerts.NewEngine(storage.NewAlertRepository(db), nil))
	registerAuthREST(apiMux, authHandler{service: authService, repo: authRepo, cookieName: cookieName, audit: auditRepo})
	registerAuditREST(apiMux, auditRepo)

	apiMux.HandleFunc("/api/v1/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.Handle("/api/v1/", secure(apiMux))
}
