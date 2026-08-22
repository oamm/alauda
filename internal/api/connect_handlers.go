package api

import (
	"net/http"

	registryv1connect "github.com/company/service-registry/gen/go/api/registry/v1/registryv1connect"
	"github.com/company/service-registry/internal/alerts"
	healthsvc "github.com/company/service-registry/internal/health"
	"github.com/company/service-registry/internal/storage"
)

// RegisterConnectHandlers mounts generated ConnectRPC handlers.
func RegisterConnectHandlers(mux *http.ServeMux, db *storage.Database) {
	RegisterConnectHandlersWithMiddleware(mux, db, func(handler http.Handler) http.Handler {
		return handler
	})
}

// RegisterConnectHandlersWithMiddleware mounts generated ConnectRPC handlers through a wrapper.
func RegisterConnectHandlersWithMiddleware(mux *http.ServeMux, db *storage.Database, wrap func(http.Handler) http.Handler) {
	alertRepo := storage.NewAlertRepository(db)
	alertEngine := alerts.NewEngine(alertRepo, nil)

	path, handler := registryv1connect.NewEnvironmentServiceHandler(&environmentHandler{repo: storage.NewEnvironmentRepository(db)})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewCatalogServiceHandler(&catalogHandler{repo: storage.NewServiceRepository(db)})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewDeploymentServiceHandler(&deploymentHandler{repo: storage.NewDeploymentRepository(db)})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewInstanceServiceHandler(&instanceHandler{
		repo:        storage.NewInstanceRepository(db),
		runtimeRepo: storage.NewRuntimeRepository(db),
	})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewEndpointServiceHandler(&endpointHandler{repo: storage.NewEndpointRepository(db)})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewHealthServiceHandler(&healthHandler{
		repo:             storage.NewHealthRepositoryWithAlerts(db, alertEngine),
		availabilityRepo: storage.NewAvailabilityRepository(db),
		executor:         healthsvc.NewExecutor(nil),
	})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewIncidentServiceHandler(&incidentHandler{repo: storage.NewIncidentRepositoryWithAlerts(db, alertEngine)})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewAlertServiceHandler(&alertHandler{repo: alertRepo, engine: alertEngine})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewEventServiceHandler(&eventHandler{repo: storage.NewEventRepository(db)})
	mux.Handle(path, wrap(handler))

	path, handler = registryv1connect.NewRegistryServiceHandler(&registryHandler{repo: storage.NewRegistryRepository(db)})
	mux.Handle(path, wrap(handler))
}

type environmentHandler struct {
	registryv1connect.UnimplementedEnvironmentServiceHandler
	repo *storage.EnvironmentRepository
}

type catalogHandler struct {
	registryv1connect.UnimplementedCatalogServiceHandler
	repo *storage.ServiceRepository
}

type deploymentHandler struct {
	registryv1connect.UnimplementedDeploymentServiceHandler
	repo *storage.DeploymentRepository
}

type instanceHandler struct {
	registryv1connect.UnimplementedInstanceServiceHandler
	repo        *storage.InstanceRepository
	runtimeRepo *storage.RuntimeRepository
}

type endpointHandler struct {
	registryv1connect.UnimplementedEndpointServiceHandler
	repo *storage.EndpointRepository
}

type healthHandler struct {
	registryv1connect.UnimplementedHealthServiceHandler
	repo             *storage.HealthRepository
	availabilityRepo *storage.AvailabilityRepository
	executor         *healthsvc.Executor
}

type incidentHandler struct {
	registryv1connect.UnimplementedIncidentServiceHandler
	repo *storage.IncidentRepository
}

type alertHandler struct {
	registryv1connect.UnimplementedAlertServiceHandler
	repo   *storage.AlertRepository
	engine *alerts.Engine
}

type eventHandler struct {
	registryv1connect.UnimplementedEventServiceHandler
	repo *storage.EventRepository
}

type registryHandler struct {
	registryv1connect.UnimplementedRegistryServiceHandler
	repo *storage.RegistryRepository
}
