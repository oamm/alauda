package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/company/service-registry/internal/alerts"
	"github.com/company/service-registry/internal/storage"
)

func registerAlertREST(mux *http.ServeMux, repo *storage.AlertRepository, engine *alerts.Engine) {
	mux.HandleFunc("/api/v1/alerts/test/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost && r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		channelID := strings.TrimPrefix(r.URL.Path, "/api/v1/alerts/test/")
		if channelID == "" {
			http.Error(w, "channel id is required", http.StatusBadRequest)
			return
		}
		channel, err := repo.GetNotificationChannel(r.Context(), channelID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "notification channel not found", http.StatusNotFound)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := engine.TestChannel(r.Context(), channel); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "failed",
				"error":  err.Error(),
			})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	})
}
