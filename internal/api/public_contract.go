package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/company/service-registry/internal/address"
	"github.com/company/service-registry/internal/contract"
	"github.com/company/service-registry/internal/problem"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

// Public registration is deliberately expressed in Service/Environment terms.
// The persistence layer may continue to use deployments internally.
type publicRegistrationRequest = contract.Registration

func validPublicKey(value string) bool {
	if len(value) < 1 || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

type publicInstanceRequest = contract.InstancePatch
type publicEndpointRequest = contract.EndpointPatch

type publicRegistrationResponse struct {
	Service     string           `json:"service"`
	Environment string           `json:"environment"`
	Instance    publicInstance   `json:"instance"`
	Endpoints   []publicEndpoint `json:"endpoints"`
}

type publicInstance struct {
	ID          string            `json:"id,omitempty"`
	Name        string            `json:"name"`
	Address     string            `json:"address"`
	Description string            `json:"description,omitempty"`
	Enabled     bool              `json:"enabled"`
	Healthy     bool              `json:"healthy"`
	HealthState string            `json:"healthState"`
	Endpoints   []publicEndpoint  `json:"endpoints,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type publicEndpoint struct {
	ID       string            `json:"id,omitempty"`
	Name     string            `json:"name"`
	Protocol string            `json:"protocol"`
	Port     int32             `json:"port"`
	Path     string            `json:"path,omitempty"`
	Primary  bool              `json:"primary"`
	Enabled  bool              `json:"enabled"`
	Address  string            `json:"address,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type publicDiscoveryResponse struct {
	Service     string           `json:"service"`
	Environment string           `json:"environment"`
	Instances   []publicInstance `json:"instances"`
}

func RegisterPublicContractREST(mux *http.ServeMux, db *storage.Database) {
	api := &publicContractAPI{db: db, runtime: storage.NewRuntimeRepository(db), services: storage.NewServiceRepository(db), environments: storage.NewEnvironmentRepository(db)}
	mux.HandleFunc("/api/v1/services", api.servicesList)
	mux.HandleFunc("/api/v1/services/", api.serviceResource)
	mux.HandleFunc("/api/v1/discovery/", api.discovery)
	mux.HandleFunc("/api/v1/health/status", api.healthStatus)
}

type publicContractAPI struct {
	db           *storage.Database
	runtime      *storage.RuntimeRepository
	services     *storage.ServiceRepository
	environments *storage.EnvironmentRepository
}

func (a *publicContractAPI) servicesList(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		a.createService(w, r)
		return
	}
	if r.Method != http.MethodGet {
		publicError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Only GET is supported.", nil)
		return
	}
	environment := r.URL.Query().Get("environment")
	size, _, ok := publicPage(w, r)
	if !ok {
		return
	}
	environmentID := ""
	if environment != "" {
		env, err := a.environments.GetByKey(r.Context(), environment)
		if err != nil {
			publicLookupError(w, err, "environment_not_found", "Environment does not exist.")
			return
		}
		environmentID = env.GetId()
		if !environmentAllowed(r, environmentID) {
			publicError(w, http.StatusForbidden, "permission_denied", "Application key is not authorized for this environment.", nil)
			return
		}
	}
	items, next, err := a.services.List(r.Context(), environmentID, size, r.URL.Query().Get("pageToken"))
	if err != nil {
		publicError(w, http.StatusInternalServerError, "server_error", "Failed to list services.", nil)
		return
	}
	result := make([]map[string]any, 0, len(items))
	snapshot, err := storage.NewHealthRepository(a.db).CurrentHealth(r.Context(), environmentID)
	if err != nil {
		publicStorageError(w, err)
		return
	}
	for _, item := range items {
		data := publicServiceData(item)
		data["healthStatus"] = healthOrUnknown(snapshot.Services[item.Id])
		result = append(result, data)
	}
	writeJSON(w, http.StatusOK, map[string]any{"services": result, "nextPageToken": next})
}

func (a *publicContractAPI) serviceResource(w http.ResponseWriter, r *http.Request) {
	parts := resourcePathParts(r.URL.Path, "/api/v1/services/")
	if len(parts) == 0 {
		publicError(w, http.StatusBadRequest, "validation_failed", "Service key is required.", map[string][]string{"service": {"Service key is required."}})
		return
	}
	serviceKey := parts[0]
	if !validPublicKey(serviceKey) {
		publicError(w, http.StatusBadRequest, "validation_failed", "Service key is invalid.", nil)
		return
	}
	if len(parts) >= 4 && parts[1] == "instances" && parts[3] == "health-checks" {
		a.healthResource(w, r, parts)
		return
	}
	if len(parts) == 2 && parts[1] == "health-results" {
		a.healthResults(w, r, parts[0])
		return
	}
	if len(parts) == 1 && r.Method == http.MethodGet {
		a.management(w, r, parts)
		return
	}
	if len(parts) < 2 || parts[1] != "instances" {
		publicError(w, http.StatusNotFound, "resource_not_found", "Resource not found.", nil)
		return
	}
	if len(parts) == 2 && r.Method == http.MethodPost {
		a.register(w, r, serviceKey)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodDelete {
		a.deregister(w, r, serviceKey, parts[2])
		return
	}
	if r.Method == http.MethodGet {
		a.management(w, r, parts)
		return
	}
	publicError(w, http.StatusMethodNotAllowed, "method_not_allowed", "The requested method is not supported for this resource.", nil)
}

func (a *publicContractAPI) register(w http.ResponseWriter, r *http.Request, serviceKey string) {
	var input publicRegistrationRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		publicError(w, 400, "validation_failed", "Request must contain one valid registration JSON object with known fields.", nil)
		return
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		publicError(w, 400, "validation_failed", "Trailing JSON is not allowed.", nil)
		return
	}
	fields := map[string][]string{}
	if strings.TrimSpace(input.Environment) == "" {
		fields["environment"] = []string{"Environment key is required."}
	}
	if !validPublicKey(input.Environment) {
		fields["environment"] = []string{"Use 1..128 letters, digits, dot, underscore or hyphen."}
	}
	if !validPublicKey(input.Instance.Name) {
		fields["instance.name"] = []string{"Use 1..128 letters, digits, dot, underscore or hyphen."}
	}
	if strings.TrimSpace(input.Instance.Name) == "" {
		fields["instance.name"] = []string{"Instance name is required."}
	}
	if input.Instance.Address != nil && strings.TrimSpace(*input.Instance.Address) == "" {
		fields["instance.address"] = []string{"Address must not be empty."}
	}
	if input.Instance.Address != nil {
		if err := address.ValidateHost(*input.Instance.Address); err != nil {
			fields["instance.address"] = []string{err.Error()}
		}
	}
	if input.Mode != "" && input.Mode != "upsert" {
		fields["mode"] = []string{"Only upsert registration is supported."}
	}
	seen := map[string]bool{}
	primaries := 0
	for i, ep := range input.Endpoints {
		key := fmt.Sprintf("endpoints[%d]", i)
		if strings.TrimSpace(ep.Name) == "" {
			fields[key+".name"] = []string{"Endpoint name is required."}
		}
		if !validPublicKey(ep.Name) {
			fields[key+".name"] = []string{"Use 1..128 letters, digits, dot, underscore or hyphen."}
		}
		if seen[ep.Name] {
			fields[key+".name"] = []string{"Endpoint name is duplicated."}
		}
		seen[ep.Name] = true
		if ep.Protocol != nil {
			if _, ok := publicProtocol(*ep.Protocol); !ok {
				fields[key+".protocol"] = []string{"Protocol must be http, https, grpc, tcp or udp."}
			}
		}
		if ep.Port != nil && (*ep.Port < 1 || *ep.Port > 65535) {
			fields[key+".port"] = []string{"Port must be between 1 and 65535."}
		}
		if ep.Path != nil {
			if err := address.ValidatePath(*ep.Path); err != nil {
				fields[key+".path"] = []string{err.Error()}
			}
		}
		if ep.Primary != nil && *ep.Primary {
			primaries++
		}
	}
	if primaries > 1 {
		fields["endpoints"] = []string{"At most one endpoint may be explicitly Primary."}
	}
	if len(fields) > 0 {
		publicError(w, 400, "validation_failed", "Registration validation failed.", fields)
		return
	}
	service, err := a.services.GetByName(r.Context(), serviceKey)
	if err != nil {
		publicLookupError(w, err, "service_not_found", "Service does not exist.")
		return
	}
	env, err := a.environments.GetByKey(r.Context(), input.Environment)
	if err != nil {
		publicLookupError(w, err, "environment_not_found", "Environment does not exist.")
		return
	}
	if !environmentAllowed(r, env.Id) {
		publicError(w, 403, "permission_denied", "Credential is not authorized for this environment.", nil)
		return
	}
	if !env.Enabled {
		publicError(w, 409, "environment_disabled", "Environment is disabled.", nil)
		return
	}
	result, err := a.runtime.RegisterPublicRuntime(r.Context(), service.Id, env.Id, input)
	if err != nil {
		var fields *storage.PublicRegistrationValidationError
		if errors.As(err, &fields) {
			publicError(w, 400, "validation_failed", "Registration validation failed.", fields.Fields)
		} else if errors.Is(err, storage.ErrInvalidRuntimeRegistration) {
			publicError(w, 400, "validation_failed", "New resources require an address, endpoint protocol and valid port.", nil)
		} else {
			publicStorageError(w, err)
		}
		return
	}
	response := publicRegistrationResponse{Service: serviceKey, Environment: input.Environment, Instance: publicInstance{ID: result.Instance.Id, Name: result.Instance.Name, Address: result.Instance.Address, Description: result.Instance.Description, Enabled: result.Instance.Enabled, Tags: result.Instance.Tags, Metadata: result.Instance.Metadata}}
	response.Instance.HealthState, err = a.effectiveHealth(r, result.Instance.Id, result.Instance.Enabled)
	if err != nil {
		publicStorageError(w, err)
		return
	}
	response.Instance.Healthy = response.Instance.HealthState == "Healthy"
	for _, item := range result.Endpoints {
		response.Endpoints = append(response.Endpoints, publicEndpoint{ID: item.Id, Name: item.Name, Protocol: strings.ToLower(strings.TrimPrefix(item.Protocol.String(), "PROTOCOL_")), Port: item.Port, Path: item.Path, Primary: item.Primary, Enabled: item.Enabled, Address: endpointAddress(item.Protocol, item.Path, result.Instance.Address, item.Port), Tags: item.Tags, Metadata: item.Metadata})
	}
	writeJSON(w, 200, response)
}

func publicLookupError(w http.ResponseWriter, err error, code, detail string) {
	if errors.Is(err, sql.ErrNoRows) {
		publicError(w, 404, code, detail, nil)
	} else {
		publicError(w, 500, "server_error", "The operation could not be completed.", nil)
	}
}
func publicStorageError(w http.ResponseWriter, err error) {
	switch storage.ClassifyError(err) {
	case storage.ErrorInvalid:
		publicError(w, 400, "validation_failed", "Resource fields failed validation.", nil)
	case storage.ErrorConflict, storage.ErrorPrecondition:
		publicError(w, 409, "resource_conflict", "The requested resource conflicts with existing state.", nil)
	case storage.ErrorUnavailable:
		publicError(w, 503, "temporarily_unavailable", "Retry the operation.", nil)
	default:
		publicError(w, 500, "server_error", "The operation could not be completed.", nil)
	}
}

func (a *publicContractAPI) deregister(w http.ResponseWriter, r *http.Request, serviceKey, instanceName string) {
	environment := r.URL.Query().Get("environment")
	if environment == "" {
		publicError(w, http.StatusBadRequest, "validation_failed", "environment is required.", nil)
		return
	}
	service, err := a.services.GetByName(r.Context(), serviceKey)
	if err != nil {
		publicLookupError(w, err, "service_not_found", "Service does not exist.")
		return
	}
	env, err := a.environments.GetByKey(r.Context(), environment)
	if err != nil {
		publicLookupError(w, err, "environment_not_found", "Environment does not exist.")
		return
	}
	if !environmentAllowed(r, env.GetId()) {
		publicError(w, http.StatusForbidden, "permission_denied", "Application key is not authorized for this environment.", nil)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err = a.db.Exec(r.Context(), `UPDATE service_instances SET deleted_at = ?, updated_at = ? WHERE deployment_id = (SELECT id FROM service_deployments WHERE service_id = ? AND environment_id = ? AND deleted_at IS NULL) AND name = ? AND deleted_at IS NULL`, now, now, service.GetId(), env.GetId(), instanceName)
	if err != nil {
		publicError(w, http.StatusInternalServerError, "server_error", "Failed to deregister instance.", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *publicContractAPI) effectiveHealth(r *http.Request, id string, enabled bool) (string, error) {
	if !enabled {
		return "Disabled", nil
	}
	snapshot, err := storage.NewHealthRepository(a.db).CurrentHealth(r.Context(), "")
	return healthOrUnknown(snapshot.Instances[id]), err
}

func healthOrUnknown(state string) string {
	if state == "" {
		return "Unknown"
	}
	return state
}

func (a *publicContractAPI) healthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		publicError(w, 405, "method_not_allowed", "Only GET is supported.", nil)
		return
	}
	environmentID := ""
	if key := r.URL.Query().Get("environment"); key != "" {
		env, err := a.environments.GetByKey(r.Context(), key)
		if err != nil {
			publicLookupError(w, err, "environment_not_found", "Environment does not exist.")
			return
		}
		if !environmentAllowed(r, env.Id) {
			publicError(w, 403, "permission_denied", "Credential is not authorized for this environment.", nil)
			return
		}
		environmentID = env.Id
	}
	snapshot, err := storage.NewHealthRepository(a.db).CurrentHealth(r.Context(), environmentID)
	if err != nil {
		publicStorageError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, snapshot)
}

func (a *publicContractAPI) discovery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	parts := resourcePathParts(r.URL.Path, "/api/v1/discovery/")
	if len(parts) < 1 || len(parts) > 2 || len(parts) == 2 && parts[1] != "resolve" {
		publicError(w, 404, "resource_not_found", "Resource does not exist.", nil)
		return
	}
	if r.Method != http.MethodGet {
		publicError(w, 405, "method_not_allowed", "Only GET is supported.", nil)
		return
	}
	serviceKey := parts[0]
	environment := r.URL.Query().Get("environment")
	if environment == "" {
		publicError(w, 400, "validation_failed", "Environment key is required.", map[string][]string{"environment": {"Environment key is required."}})
		return
	}
	policy := r.URL.Query().Get("health")
	if policy == "" {
		policy = "usable"
	}
	if policy != "usable" && policy != "healthy" && policy != "all" {
		publicError(w, 400, "validation_failed", "Unsupported health policy.", map[string][]string{"health": {"Use usable, healthy or all."}})
		return
	}
	endpointName := r.URL.Query().Get("endpoint")
	env, err := a.environments.GetByKey(r.Context(), environment)
	if err != nil {
		publicLookupError(w, err, "environment_not_found", "Environment does not exist.")
		return
	}
	if !environmentAllowed(r, env.Id) {
		publicError(w, 403, "permission_denied", "Credential is not authorized for this environment.", nil)
		return
	}
	if !env.Enabled {
		publicError(w, 409, "environment_disabled", "Environment is disabled.", nil)
		return
	}
	if _, err = a.services.GetByName(r.Context(), serviceKey); err != nil {
		publicLookupError(w, err, "service_not_found", "Service does not exist.")
		return
	}
	query := "SELECT si.id,si.name,si.address,COALESCE(si.description,''),e.id,e.name,e.protocol,e.port,COALESCE(e.path,''),e.primary_endpoint FROM services s JOIN service_deployments d ON d.service_id=s.id AND d.environment_id=? AND d.deleted_at IS NULL JOIN service_instances si ON si.deployment_id=d.id AND si.deleted_at IS NULL AND si.enabled=1 JOIN endpoints e ON e.instance_id=si.id AND e.deleted_at IS NULL AND e.enabled=1 WHERE s.name=? AND s.deleted_at IS NULL ORDER BY si.name,e.name"
	snapshot, err := storage.NewHealthRepository(a.db).CurrentHealth(r.Context(), env.Id)
	if err != nil {
		publicStorageError(w, err)
		return
	}
	rows, err := a.db.Query(r.Context(), query, env.Id, serviceKey)
	if err != nil {
		publicStorageError(w, err)
		return
	}
	defer rows.Close()
	response := publicDiscoveryResponse{Service: serviceKey, Environment: environment, Instances: []publicInstance{}}
	indices := map[string]int{}
	namedExists := false
	for rows.Next() {
		var id, name, host, description, eid, ename, protocol, path string
		var port int32
		var primary bool
		if err := rows.Scan(&id, &name, &host, &description, &eid, &ename, &protocol, &port, &path, &primary); err != nil {
			publicStorageError(w, err)
			return
		}
		if ename == endpointName {
			namedExists = true
		}
		if endpointName != "" && ename != endpointName {
			continue
		}
		state := healthOrUnknown(snapshot.Instances[id])
		if policy == "healthy" && state != "Healthy" || policy == "usable" && (state == "Unhealthy" || state == "Disabled") {
			continue
		}
		p := protocolFromDB(protocol)
		fullAddress, err := address.Build(p, host, port, path)
		if err != nil {
			continue
		}
		idx, ok := indices[id]
		if !ok {
			idx = len(response.Instances)
			indices[id] = idx
			response.Instances = append(response.Instances, publicInstance{ID: id, Name: name, Address: host, Description: description, Enabled: true, Healthy: state == "Healthy", HealthState: state, Endpoints: []publicEndpoint{}})
		}
		response.Instances[idx].Endpoints = append(response.Instances[idx].Endpoints, publicEndpoint{ID: eid, Name: ename, Protocol: p, Port: port, Path: path, Primary: primary, Enabled: true, Address: fullAddress})
	}
	if err := rows.Err(); err != nil {
		publicStorageError(w, err)
		return
	}
	sort.SliceStable(response.Instances, func(i, j int) bool {
		a, b := response.Instances[i], response.Instances[j]
		if a.Healthy != b.Healthy {
			return a.Healthy
		}
		return a.Name < b.Name
	})
	if len(response.Instances) == 0 {
		if endpointName != "" && !namedExists {
			publicError(w, 404, "endpoint_not_found", "Named endpoint does not exist or is disabled.", nil)
		} else {
			publicError(w, 404, "no_healthy_instance", "No eligible discovery candidate was found.", nil)
		}
		return
	}
	if len(parts) == 2 {
		a.resolveDiscovery(w, r, response, endpointName)
		return
	}
	writeJSON(w, 200, response)
}

func (a *publicContractAPI) resolveDiscovery(w http.ResponseWriter, r *http.Request, response publicDiscoveryResponse, endpointName string) {
	_ = r
	if len(response.Instances) == 0 {
		publicError(w, http.StatusNotFound, "no_healthy_instance", "No usable instance was found.", nil)
		return
	}
	instance := response.Instances[0]
	var selected *publicEndpoint
	for i := range instance.Endpoints {
		if endpointName != "" && instance.Endpoints[i].Name == endpointName {
			selected = &instance.Endpoints[i]
			break
		}
		if endpointName == "" && instance.Endpoints[i].Primary {
			selected = &instance.Endpoints[i]
		}
	}
	if selected == nil && endpointName == "" && len(instance.Endpoints) > 0 {
		selected = &instance.Endpoints[0]
	}
	if selected == nil {
		publicError(w, http.StatusNotFound, "no_primary_endpoint", "No matching endpoint was found.", nil)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"service": response.Service, "environment": response.Environment, "instance": instance.Name, "endpoint": selected.Name, "address": selected.Address})
}

func publicProtocol(value string) (registryv1.Protocol, bool) {
	switch strings.ToLower(value) {
	case "http":
		return registryv1.Protocol_PROTOCOL_HTTP, true
	case "https":
		return registryv1.Protocol_PROTOCOL_HTTPS, true
	case "grpc":
		return registryv1.Protocol_PROTOCOL_GRPC, true
	case "tcp":
		return registryv1.Protocol_PROTOCOL_TCP, true
	case "udp":
		return registryv1.Protocol_PROTOCOL_UDP, true
	}
	return registryv1.Protocol_PROTOCOL_UNSPECIFIED, false
}
func protocolFromDB(value string) string {
	switch value {
	case "1", "PROTOCOL_HTTP":
		return "http"
	case "2", "PROTOCOL_HTTPS":
		return "https"
	case "3", "PROTOCOL_GRPC":
		return "grpc"
	case "4", "PROTOCOL_TCP":
		return "tcp"
	case "5", "PROTOCOL_UDP":
		return "udp"
	}
	return strings.ToLower(strings.TrimPrefix(value, "PROTOCOL_"))
}
func publicAddress(protocol, path, host string, port int32) string {
	value, _ := address.Build(protocol, host, port, path)
	return value
}
func endpointAddress(protocol registryv1.Protocol, path, host string, port int32) string {
	return publicAddress(protocolFromDB(protocol.String()), path, host, port)
}
func publicError(w http.ResponseWriter, status int, code, detail string, fields map[string][]string) {
	problem.Write(w, status, code, detail, fields)
}
