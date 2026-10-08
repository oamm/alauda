package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
)

// RegisterRESTResources mounts the canonical noun-based, read-only resource API.
func RegisterRESTResources(mux *http.ServeMux, db *storage.Database) {
	api := &restResources{
		db:              db,
		environmentRepo: storage.NewEnvironmentRepository(db),
		services:        storage.NewServiceRepository(db),
	}
	mux.HandleFunc("/api/v1/environments", api.listEnvironments)
	mux.HandleFunc("/api/v1/environments/", api.environmentResource)
}

type restResources struct {
	db              *storage.Database
	environmentRepo *storage.EnvironmentRepository
	services        *storage.ServiceRepository
}

func (a *restResources) listEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		createPublicEnvironment(w, r, a.environmentRepo)
		return
	}
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	size, _, ok := publicPage(w, r)
	if !ok {
		return
	}
	items, next, err := a.environmentRepo.List(r.Context(), true, size, r.URL.Query().Get("pageToken"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list environments")
		return
	}
	items = filterEnvironments(r, items)
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, publicEnvironmentData(item))
	}
	writeJSON(w, 200, map[string]any{"environments": result, "nextPageToken": next, "pagination": map[string]any{"nextPageToken": next, "totalSize": len(items)}})
}

func (a *restResources) environmentResource(w http.ResponseWriter, r *http.Request) {
	parts := resourcePathParts(r.URL.Path, "/api/v1/environments/")
	if len(parts) == 0 || len(parts) > 3 {
		writeAPIError(w, http.StatusNotFound, "resource not found")
		return
	}

	environmentKey := parts[0]
	if !validPublicKey(environmentKey) {
		writeAPIError(w, http.StatusBadRequest, "invalid environment key")
		return
	}
	environment, err := a.environmentRepo.GetByKey(r.Context(), environmentKey)
	if err != nil {
		writeResourceError(w, err, "environment not found")
		return
	}
	if !environmentAllowed(r, environment.GetId()) {
		writeAPIError(w, http.StatusForbidden, "application key is not authorized for this environment")
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		writeJSON(w, 200, publicEnvironmentData(environment))
		return
	}
	if parts[1] != "services" {
		writeAPIError(w, http.StatusNotFound, "resource not found")
		return
	}

	if len(parts) == 2 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		size, _, ok := publicPage(w, r)
		if !ok {
			return
		}
		items, next, err := a.services.List(r.Context(), environment.GetId(), size, r.URL.Query().Get("pageToken"))
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to list services")
			return
		}
		result := make([]map[string]any, 0, len(items))
		snapshot, err := storage.NewHealthRepository(a.db).CurrentHealth(r.Context(), environment.GetId())
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to load health status")
			return
		}
		for _, item := range items {
			data := publicServiceData(item)
			data["healthStatus"] = healthOrUnknown(snapshot.Services[item.Id])
			result = append(result, data)
		}
		writeJSON(w, 200, map[string]any{"services": result, "nextPageToken": next, "pagination": map[string]any{"nextPageToken": next, "totalSize": len(items)}})
		return
	}

	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	serviceName := parts[2]
	if !validPublicKey(serviceName) {
		writeAPIError(w, http.StatusBadRequest, "invalid service name")
		return
	}
	service, err := a.services.GetByName(r.Context(), serviceName)
	if err != nil {
		writeResourceError(w, err, "service not found")
		return
	}
	member, err := a.services.BelongsToEnvironment(r.Context(), service.GetId(), environment.GetId())
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to find service in environment")
		return
	}
	if member {
		snapshot, err := storage.NewHealthRepository(a.db).CurrentHealth(r.Context(), environment.GetId())
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "failed to load health status")
			return
		}
		data := publicServiceData(service)
		data["healthStatus"] = healthOrUnknown(snapshot.Services[service.Id])
		writeJSON(w, 200, data)
		return
	}
	writeAPIError(w, http.StatusNotFound, "service not found in environment")
}

func resourcePathParts(path, prefix string) []string {
	value := strings.TrimPrefix(path, prefix)
	value = strings.Trim(value, "/")
	if value == "" {
		return nil
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" {
			return nil
		}
	}
	return parts
}

func writeResourceError(w http.ResponseWriter, err error, notFound string) {
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(w, http.StatusNotFound, notFound)
		return
	}
	writeAPIError(w, http.StatusInternalServerError, "failed to load resource")
}

func environmentAllowed(r *http.Request, environmentID string) bool {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || len(principal.EnvironmentIDs) == 0 {
		return true
	}
	for _, allowed := range principal.EnvironmentIDs {
		if allowed == environmentID {
			return true
		}
	}
	return false
}

func filterEnvironments(r *http.Request, environments []*registryv1.Environment) []*registryv1.Environment {
	principal, ok := auth.PrincipalFromContext(r.Context())
	if !ok || len(principal.EnvironmentIDs) == 0 {
		return environments
	}
	filtered := make([]*registryv1.Environment, 0, len(environments))
	for _, environment := range environments {
		if environmentAllowed(r, environment.GetId()) {
			filtered = append(filtered, environment)
		}
	}
	return filtered
}
