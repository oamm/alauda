package api

import (
	"net/http"
	"strings"
	"time"

	v1 "github.com/company/service-registry/gen/go/api/registry/v1"
	healthsvc "github.com/company/service-registry/internal/health"
	"github.com/company/service-registry/internal/storage"
)

type publicHealthCheck struct {
	Name                    string            `json:"name"`
	Endpoint                string            `json:"endpoint"`
	Type                    string            `json:"type"`
	Enabled                 *bool             `json:"enabled,omitempty"`
	IntervalSeconds         int32             `json:"intervalSeconds"`
	TimeoutSeconds          int32             `json:"timeoutSeconds"`
	FailuresBeforeUnhealthy int32             `json:"failuresBeforeUnhealthy"`
	SuccessesBeforeHealthy  int32             `json:"successesBeforeHealthy"`
	Description             string            `json:"description,omitempty"`
	Tags                    map[string]string `json:"tags,omitempty"`
	Metadata                map[string]string `json:"metadata,omitempty"`
}

func (a *publicContractAPI) healthResource(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) != 4 && !(len(parts) == 6 && parts[5] == "run") {
		publicError(w, 404, "resource_not_found", "Resource does not exist.", nil)
		return
	}
	svc, env, ok := a.managementContext(w, r, parts[0])
	if !ok {
		return
	}
	var instanceID string
	err := a.db.QueryRow(r.Context(), `SELECT i.id FROM service_instances i JOIN service_deployments d ON d.id=i.deployment_id WHERE d.service_id=? AND d.environment_id=? AND i.name=? AND d.deleted_at IS NULL AND i.deleted_at IS NULL`, svc.Id, env.Id, parts[2]).Scan(&instanceID)
	if err != nil {
		publicLookupError(w, err, "instance_not_found", "Instance does not exist.")
		return
	}
	repo := storage.NewHealthRepository(a.db)
	if len(parts) == 6 {
		if r.Method != http.MethodPost {
			publicError(w, 405, "method_not_allowed", "Execution requires POST.", nil)
			return
		}
		if !env.Enabled {
			publicError(w, 409, "environment_disabled", "Environment is disabled.", nil)
			return
		}
		var id string
		err = a.db.QueryRow(r.Context(), `SELECT id FROM health_checks WHERE instance_id=? AND name=? AND deleted_at IS NULL`, instanceID, parts[4]).Scan(&id)
		if err != nil {
			publicLookupError(w, err, "health_check_not_found", "Health Check does not exist.")
			return
		}
		target, err := repo.GetHealthCheckTarget(r.Context(), id)
		if err != nil {
			publicLookupError(w, err, "health_check_inactive", "Health Check or its target is inactive.")
			return
		}
		result := healthsvc.NewExecutor(nil).Execute(r.Context(), healthsvc.Target{Check: target.Check, Address: target.Address, Port: target.Port, Path: target.Path})
		_, err = repo.RecordHealthResult(r.Context(), target.Check, result)
		if err != nil {
			publicStorageError(w, err)
			return
		}
		state, err := a.effectiveHealth(r, instanceID, true)
		if err != nil {
			publicStorageError(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"check": parts[4], "instance": parts[2], "success": result.Success, "timestamp": result.Timestamp.AsTime().UTC().Format(time.RFC3339Nano), "healthState": state})
		return
	}
	if r.Method == http.MethodPost {
		input := publicHealthCheck{IntervalSeconds: 30, TimeoutSeconds: 5, FailuresBeforeUnhealthy: 3, SuccessesBeforeHealthy: 2}
		if !decodePublic(w, r, &input) {
			return
		}
		typ := v1.HealthCheckType(v1.HealthCheckType_value["HEALTH_CHECK_TYPE_"+strings.ToUpper(input.Type)])
		fields := map[string][]string{}
		if !validPublicKey(input.Name) {
			fields["name"] = []string{"Use a valid Check name."}
		}
		if input.Type != "http" && input.Type != "tcp" && input.Type != "dns" {
			fields["type"] = []string{"Use http, tcp or dns."}
		}
		if !validPublicKey(input.Endpoint) {
			fields["endpoint"] = []string{"An active Endpoint name is required."}
		}
		if input.IntervalSeconds < 1 || input.IntervalSeconds > 86400 {
			fields["intervalSeconds"] = []string{"Use 1..86400."}
		}
		if input.TimeoutSeconds < 1 || input.TimeoutSeconds > 300 {
			fields["timeoutSeconds"] = []string{"Use 1..300."}
		}
		if input.FailuresBeforeUnhealthy < 1 {
			fields["failuresBeforeUnhealthy"] = []string{"Must be positive."}
		}
		if input.SuccessesBeforeHealthy < 1 {
			fields["successesBeforeHealthy"] = []string{"Must be positive."}
		}
		if len(fields) > 0 {
			publicError(w, 400, "validation_failed", "Invalid Health Check.", fields)
			return
		}
		var endpointID string
		var protocol int32
		err = a.db.QueryRow(r.Context(), `SELECT id,protocol FROM endpoints WHERE instance_id=? AND name=? AND deleted_at IS NULL`, instanceID, input.Endpoint).Scan(&endpointID, &protocol)
		if err != nil {
			publicLookupError(w, err, "endpoint_not_found", "Endpoint does not exist.")
			return
		}
		if input.Type == "http" {
			if protocol != int32(v1.Protocol_PROTOCOL_HTTP) && protocol != int32(v1.Protocol_PROTOCOL_HTTPS) {
				publicError(w, 400, "validation_failed", "HTTP checks require an HTTP or HTTPS Endpoint.", map[string][]string{"endpoint": {"Choose an HTTP or HTTPS Endpoint."}})
				return
			}
			if input.Metadata == nil {
				input.Metadata = map[string]string{}
			}
			input.Metadata["scheme"] = strings.ToLower(strings.TrimPrefix(v1.Protocol(protocol).String(), "PROTOCOL_"))
		}
		enabled := true
		if input.Enabled != nil {
			enabled = *input.Enabled
		}
		_, err = repo.CreateHealthCheck(r.Context(), &v1.CreateHealthCheckRequest{InstanceId: instanceID, EndpointId: endpointID, Name: input.Name, Type: typ, Enabled: enabled, IntervalSeconds: input.IntervalSeconds, TimeoutSeconds: input.TimeoutSeconds, FailuresBeforeUnhealthy: input.FailuresBeforeUnhealthy, SuccessesBeforeHealthy: input.SuccessesBeforeHealthy, Description: input.Description, Tags: input.Tags, Metadata: input.Metadata})
		if err != nil {
			publicStorageError(w, err)
			return
		}
		input.Enabled = &enabled
		writeJSON(w, 201, input)
		return
	}
	if r.Method != http.MethodGet {
		publicError(w, 405, "method_not_allowed", "Use GET or POST.", nil)
		return
	}
	size, _, ok := publicPage(w, r)
	if !ok {
		return
	}
	checks, next, err := repo.ListHealthChecks(r.Context(), instanceID, true, size, r.URL.Query().Get("pageToken"))
	if err != nil {
		publicStorageError(w, err)
		return
	}
	items := []publicHealthCheck{}
	for _, check := range checks {
		var endpoint string
		if check.EndpointId != "" {
			if err = a.db.QueryRow(r.Context(), "SELECT name FROM endpoints WHERE id=?", check.EndpointId).Scan(&endpoint); err != nil {
				publicStorageError(w, err)
				return
			}
		}
		enabled := check.Enabled
		items = append(items, publicHealthCheck{Name: check.Name, Endpoint: endpoint, Type: strings.ToLower(strings.TrimPrefix(check.Type.String(), "HEALTH_CHECK_TYPE_")), Enabled: &enabled, IntervalSeconds: check.IntervalSeconds, TimeoutSeconds: check.TimeoutSeconds, FailuresBeforeUnhealthy: check.FailuresBeforeUnhealthy, SuccessesBeforeHealthy: check.SuccessesBeforeHealthy, Description: check.Description, Tags: check.Tags, Metadata: check.Metadata})
	}
	writeJSON(w, 200, map[string]any{"checks": items, "nextPageToken": next})
}
