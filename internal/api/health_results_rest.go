package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
)

func registerHealthResultsREST(mux *http.ServeMux, repo *storage.HealthRepository) {
	mux.HandleFunc("/api/v1/health/results", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		v := r.URL.Query()
		q := storage.HealthResultsQuery{From: v.Get("from"), To: v.Get("to"), Search: v.Get("search"), ServiceID: v.Get("serviceId"), EnvironmentID: v.Get("environmentId"), InstanceID: v.Get("instanceId"), EndpointID: v.Get("endpointId"), CheckID: v.Get("checkId"), Type: v.Get("type"), Status: v.Get("status"), PageSize: 25}
		var from, to time.Time
		for _, bound := range []struct {
			raw    string
			target *time.Time
		}{{q.From, &from}, {q.To, &to}} {
			if bound.raw != "" {
				value, err := time.Parse(time.RFC3339Nano, bound.raw)
				if err != nil {
					writeAPIError(w, 400, "Dates must use RFC3339 format")
					return
				}
				*bound.target = value
			}
		}
		if !from.IsZero() && !to.IsZero() && from.After(to) {
			writeAPIError(w, 400, "From must not be after To")
			return
		}
		for _, field := range []struct {
			name     string
			target   *int
			min, max int
		}{{"port", &q.Port, 1, 65535}, {"pageSize", &q.PageSize, 1, 100}, {"pageToken", &q.Offset, 0, 2147483647}} {
			if raw := v.Get(field.name); raw != "" {
				value, err := strconv.Atoi(raw)
				if err != nil || value < field.min || value > field.max {
					writeAPIError(w, 400, "Invalid "+field.name)
					return
				}
				*field.target = value
			}
		}
		if q.Status != "" && q.Status != "healthy" && q.Status != "unhealthy" {
			writeAPIError(w, 400, "Invalid result status")
			return
		}
		if value, exists := registryv1.HealthCheckType_value[q.Type]; q.Type != "" && (!exists || value == 0) {
			writeAPIError(w, 400, "Invalid check type")
			return
		}
		if sort := v.Get("sort"); sort != "" && sort != "newest" && sort != "oldest" {
			writeAPIError(w, 400, "Invalid sort order")
			return
		}
		q.Oldest = v.Get("sort") == "oldest"
		if principal, ok := auth.PrincipalFromContext(r.Context()); ok {
			q.AllowedEnvironmentIDs = principal.EnvironmentIDs
		}
		items, next, err := repo.QueryHealthResults(r.Context(), q)
		if err != nil {
			writeAPIError(w, 500, "Failed to query health results")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Results       []storage.HealthResultRecord `json:"results"`
			NextPageToken string                       `json:"nextPageToken"`
		}{items, next})
	})
}
