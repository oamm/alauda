package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/company/service-registry/internal/storage"
)

func registerAuditREST(mux *http.ServeMux, repo *storage.AuditRepository) {
	mux.HandleFunc("/api/v1/audit-logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		pageSize := 50
		if raw := r.URL.Query().Get("pageSize"); raw != "" {
			if parsed, err := strconv.Atoi(raw); err == nil {
				pageSize = parsed
			}
		}
		items, nextToken, err := repo.List(r.Context(), storage.AuditFilters{
			Actor:         r.URL.Query().Get("actor"),
			Action:        r.URL.Query().Get("action"),
			ResourceType:  r.URL.Query().Get("resourceType"),
			ResourceID:    r.URL.Query().Get("resourceId"),
			EnvironmentID: r.URL.Query().Get("environmentId"),
		}, pageSize, r.URL.Query().Get("pageToken"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"auditLogs":     items,
			"nextPageToken": nextToken,
		})
	})
}
