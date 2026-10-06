package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// RegisterExternalCatalogREST mounts the read-only, client-facing catalog API.
// The caller is responsible for applying authentication and authorization.
func RegisterExternalCatalogREST(mux *http.ServeMux, db *storage.Database) {
	api := &externalCatalogAPI{
		environments: storage.NewEnvironmentRepository(db),
		services:     storage.NewServiceRepository(db),
		deployments:  storage.NewDeploymentRepository(db),
		instances:    storage.NewInstanceRepository(db),
		endpoints:    storage.NewEndpointRepository(db),
		registry:     storage.NewRegistryRepository(db),
	}

	mux.HandleFunc("/api/v1/catalog/environments", api.listEnvironments)
	mux.HandleFunc("/api/v1/catalog/services", api.listServices)
	mux.HandleFunc("/api/v1/catalog/deployments", api.listDeployments)
	mux.HandleFunc("/api/v1/catalog/instances", api.listInstances)
	mux.HandleFunc("/api/v1/catalog/endpoints", api.listEndpoints)
	mux.HandleFunc("/api/v1/discovery/services/", api.resolveService)
}

type externalCatalogAPI struct {
	environments *storage.EnvironmentRepository
	services     *storage.ServiceRepository
	deployments  *storage.DeploymentRepository
	instances    *storage.InstanceRepository
	endpoints    *storage.EndpointRepository
	registry     *storage.RegistryRepository
}

func (a *externalCatalogAPI) listEnvironments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	items, next, err := a.environments.List(r.Context(), r.URL.Query().Get("includeDisabled") == "true", pageSize(r), r.URL.Query().Get("pageToken"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list environments")
		return
	}
	writeProtoJSON(w, &registryv1.ListEnvironmentsResponse{
		Environments: items,
		Pagination:   &registryv1.PaginationResponse{NextPageToken: next, TotalSize: int32(len(items))},
	})
}

func (a *externalCatalogAPI) listServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	environmentID := r.URL.Query().Get("environmentId")
	if environmentKey := catalogQueryValue(r, "environment", "environmentKey"); environmentKey != "" {
		environment, err := a.environments.GetByKey(r.Context(), environmentKey)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeAPIError(w, http.StatusNotFound, "environment not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "failed to find environment")
			return
		}
		environmentID = environment.GetId()
	}
	if !environmentAllowed(r, environmentID) {
		writeAPIError(w, http.StatusForbidden, "application key is not authorized for this environment")
		return
	}
	if environmentID == "" && applicationKeyIsEnvironmentScoped(r) {
		writeAPIError(w, http.StatusBadRequest, "environment is required for a scoped application key")
		return
	}
	items, next, err := a.services.List(r.Context(), environmentID, pageSize(r), r.URL.Query().Get("pageToken"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list services")
		return
	}
	writeProtoJSON(w, &registryv1.ListServicesResponse{
		Services:   items,
		Pagination: &registryv1.PaginationResponse{NextPageToken: next, TotalSize: int32(len(items))},
	})
}

func catalogQueryValue(r *http.Request, names ...string) string {
	for _, name := range names {
		if value := r.URL.Query().Get(name); value != "" {
			return value
		}
	}
	return ""
}

func applicationKeyIsEnvironmentScoped(r *http.Request) bool {
	principal, ok := auth.PrincipalFromContext(r.Context())
	return ok && principal.ApplicationKeyID != "" && len(principal.EnvironmentIDs) > 0
}

func (a *externalCatalogAPI) listDeployments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	environmentID := r.URL.Query().Get("environmentId")
	if environmentKey := catalogQueryValue(r, "environment", "environmentKey"); environmentKey != "" {
		environment, err := a.environments.GetByKey(r.Context(), environmentKey)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeAPIError(w, http.StatusNotFound, "environment not found")
				return
			}
			writeAPIError(w, http.StatusInternalServerError, "failed to find environment")
			return
		}
		environmentID = environment.GetId()
	}
	if !environmentAllowed(r, environmentID) {
		writeAPIError(w, http.StatusForbidden, "application key is not authorized for this environment")
		return
	}
	if environmentID == "" && applicationKeyIsEnvironmentScoped(r) {
		writeAPIError(w, http.StatusBadRequest, "environment is required for a scoped application key")
		return
	}
	items, next, err := a.deployments.List(r.Context(), r.URL.Query().Get("serviceId"), environmentID, pageSize(r), r.URL.Query().Get("pageToken"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list deployments")
		return
	}
	writeProtoJSON(w, &registryv1.ListDeploymentsResponse{
		Deployments: items,
		Pagination:  &registryv1.PaginationResponse{NextPageToken: next, TotalSize: int32(len(items))},
	})
}

func (a *externalCatalogAPI) listInstances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	items, next, err := a.instances.List(r.Context(), r.URL.Query().Get("deploymentId"), pageSize(r), r.URL.Query().Get("pageToken"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list instances")
		return
	}
	writeProtoJSON(w, &registryv1.ListInstancesResponse{
		Instances:  items,
		Pagination: &registryv1.PaginationResponse{NextPageToken: next, TotalSize: int32(len(items))},
	})
}

func (a *externalCatalogAPI) listEndpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	items, next, err := a.endpoints.List(r.Context(), r.URL.Query().Get("instanceId"), pageSize(r), r.URL.Query().Get("pageToken"))
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to list endpoints")
		return
	}
	writeProtoJSON(w, &registryv1.ListEndpointsResponse{
		Endpoints:  items,
		Pagination: &registryv1.PaginationResponse{NextPageToken: next, TotalSize: int32(len(items))},
	})
}

func (a *externalCatalogAPI) resolveService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	serviceName := strings.TrimPrefix(r.URL.Path, "/api/v1/discovery/services/")
	if serviceName == "" || strings.Contains(serviceName, "/") {
		writeAPIError(w, http.StatusBadRequest, "service name is required")
		return
	}
	environment := r.URL.Query().Get("environment")
	if environment == "" {
		writeAPIError(w, http.StatusBadRequest, "environment is required")
		return
	}
	healthyOnly := r.URL.Query().Get("healthyOnly") == "true"
	serviceID, deploymentID, endpoints, err := a.registry.ResolveService(r.Context(), serviceName, environment, healthyOnly)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, sql.ErrNoRows) {
			status = http.StatusNotFound
		}
		writeAPIError(w, status, "failed to resolve service")
		return
	}
	writeProtoJSON(w, &registryv1.ResolveServiceResponse{
		ServiceId: serviceID, DeploymentId: deploymentID, Endpoints: endpoints,
	})
}

func pageSize(r *http.Request) int {
	value, err := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if err != nil || value <= 0 {
		return 50
	}
	if value > 200 {
		return 200
	}
	return value
}

func writeProtoJSON(w http.ResponseWriter, message proto.Message) {
	payload, err := (protojson.MarshalOptions{UseProtoNames: false}).Marshal(message)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, "failed to encode response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", http.MethodGet)
	writeAPIError(w, http.StatusMethodNotAllowed, "method not allowed")
}
