package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
)

func registerHealthChecksREST(mux *http.ServeMux, repo *storage.HealthRepository) {
	mux.HandleFunc("/api/v1/health/checks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		v := r.URL.Query()
		q := storage.HealthChecksQuery{Search: strings.TrimSpace(v.Get("search")), ServiceID: v.Get("serviceId"), EnvironmentID: v.Get("environmentId"), InstanceID: v.Get("instanceId"), EndpointID: v.Get("endpointId"), LatestStatus: v.Get("latestStatus"), Page: 1, PageSize: 25}
		if raw := v.Get("page"); raw != "" {
			page, err := strconv.Atoi(raw)
			if err != nil || page < 1 {
				writeAPIError(w, 400, "Invalid page")
				return
			}
			q.Page = page
		}
		if raw := v.Get("pageSize"); raw != "" {
			size, err := strconv.Atoi(raw)
			if err != nil || (size != 25 && size != 50 && size != 100) {
				writeAPIError(w, 400, "Invalid pageSize")
				return
			}
			q.PageSize = size
		}
		if raw, ok := v["enabled"]; ok && len(raw) > 0 {
			value, err := strconv.ParseBool(raw[0])
			if err != nil {
				writeAPIError(w, 400, "Invalid enabled")
				return
			}
			q.Enabled = &value
		}
		if principal, ok := auth.PrincipalFromContext(r.Context()); ok {
			q.AllowedEnvironmentIDs = principal.EnvironmentIDs
		}
		result, err := repo.QueryHealthChecks(r.Context(), q)
		if err != nil {
			writeAPIError(w, 400, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(result)
	})
}
