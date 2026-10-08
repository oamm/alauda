package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	v1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

func decodePublic(w http.ResponseWriter, r *http.Request, out any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		publicError(w, 400, "validation_failed", "Invalid request body.", map[string][]string{"body": {"Use the documented JSON schema."}})
		return false
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		publicError(w, 400, "validation_failed", "Only one JSON document is allowed.", nil)
		return false
	}
	return true
}

func publicPage(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	size := 50
	offset := 0
	var err error
	if value := r.URL.Query().Get("pageSize"); value != "" {
		size, err = strconv.Atoi(value)
		if err != nil || size < 1 || size > 200 {
			publicError(w, 400, "validation_failed", "Invalid pagination.", map[string][]string{"pageSize": {"Use an integer from 1 through 200."}})
			return 0, 0, false
		}
	}
	if value := r.URL.Query().Get("pageToken"); value != "" {
		offset, err = strconv.Atoi(value)
		if err != nil || offset < 0 {
			publicError(w, 400, "validation_failed", "Invalid pagination.", map[string][]string{"pageToken": {"Use the returned nextPageToken."}})
			return 0, 0, false
		}
	}
	return size, offset, true
}

type publicBootstrap struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"displayName,omitempty"`
	Description string            `json:"description,omitempty"`
	Tags        map[string]string `json:"tags,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

func (a *publicContractAPI) createService(w http.ResponseWriter, r *http.Request) {
	var input publicBootstrap
	if !decodePublic(w, r, &input) {
		return
	}
	if !validPublicKey(input.Name) {
		publicError(w, 400, "validation_failed", "Invalid Service key.", map[string][]string{"name": {"Use 1..128 letters, digits, dots, underscores or hyphens."}})
		return
	}
	if input.DisplayName == "" {
		input.DisplayName = input.Name
	}
	item, err := a.services.Create(r.Context(), &v1.CreateServiceRequest{Name: input.Name, DisplayName: input.DisplayName, Description: input.Description, Tags: input.Tags, Metadata: input.Metadata})
	if err != nil {
		publicStorageError(w, err)
		return
	}
	writeJSON(w, 201, publicServiceData(item))
}
func publicServiceData(item *v1.Service) map[string]any {
	return map[string]any{"name": item.Name, "displayName": item.DisplayName, "description": item.Description, "tags": item.Tags, "metadata": item.Metadata, "id": item.Id}
}

func (a *publicContractAPI) managementContext(w http.ResponseWriter, r *http.Request, key string) (*v1.Service, *v1.Environment, bool) {
	envKey := r.URL.Query().Get("environment")
	if !validPublicKey(envKey) {
		publicError(w, 400, "validation_failed", "Environment key is required.", map[string][]string{"environment": {"Use a valid Environment key."}})
		return nil, nil, false
	}
	env, err := a.environments.GetByKey(r.Context(), envKey)
	if err != nil {
		publicLookupError(w, err, "environment_not_found", "Environment does not exist.")
		return nil, nil, false
	}
	if !environmentAllowed(r, env.Id) {
		publicError(w, 403, "permission_denied", "Credential is not authorized for this environment.", nil)
		return nil, nil, false
	}
	svc, err := a.services.GetByName(r.Context(), key)
	if err != nil {
		publicLookupError(w, err, "service_not_found", "Service does not exist.")
		return nil, nil, false
	}
	return svc, env, true
}

func (a *publicContractAPI) management(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Method != http.MethodGet {
		publicError(w, 405, "method_not_allowed", "Only GET is supported.", nil)
		return
	}
	if len(parts) > 4 || len(parts) == 4 && parts[3] != "endpoints" {
		publicError(w, 404, "resource_not_found", "Resource does not exist.", nil)
		return
	}
	svc, env, ok := a.managementContext(w, r, parts[0])
	if !ok {
		return
	}
	size, offset, ok := publicPage(w, r)
	if !ok {
		return
	}
	if len(parts) == 1 {
		member, err := a.services.BelongsToEnvironment(r.Context(), svc.Id, env.Id)
		if err != nil {
			publicStorageError(w, err)
			return
		}
		if !member {
			publicError(w, 404, "service_not_found", "Service is not registered in this Environment.", nil)
			return
		}
		writeJSON(w, 200, publicServiceData(svc))
		return
	}
	query := "SELECT si.id,si.name,si.address,COALESCE(si.description,''),si.enabled," + effectiveHealthSQL + ",si.tags,si.metadata FROM service_instances si JOIN service_deployments d ON d.id=si.deployment_id LEFT JOIN health_states hs ON hs.instance_id=si.id WHERE d.service_id=? AND d.environment_id=? AND d.deleted_at IS NULL AND si.deleted_at IS NULL"
	args := []any{svc.Id, env.Id}
	if len(parts) >= 3 {
		query += " AND si.name=?"
		args = append(args, parts[2])
	}
	query += " ORDER BY si.name LIMIT ? OFFSET ?"
	limit := size + 1
	if len(parts) >= 3 {
		limit = 1
		offset = 0
	}
	args = append(args, limit, offset)
	rows, err := a.db.Query(r.Context(), query, args...)
	if err != nil {
		publicStorageError(w, err)
		return
	}
	items := []publicInstance{}
	for rows.Next() {
		var item publicInstance
		var tags, metadata string
		if err = rows.Scan(&item.ID, &item.Name, &item.Address, &item.Description, &item.Enabled, &item.HealthState, &tags, &metadata); err != nil {
			rows.Close()
			publicStorageError(w, err)
			return
		}
		if json.Unmarshal([]byte(tags), &item.Tags) != nil || json.Unmarshal([]byte(metadata), &item.Metadata) != nil {
			rows.Close()
			publicError(w, 500, "server_error", "Unable to read resource attributes.", nil)
			return
		}
		item.Healthy = item.HealthState == "Healthy"
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		publicStorageError(w, err)
		return
	}
	if len(parts) >= 3 && len(items) == 0 {
		publicError(w, 404, "instance_not_found", "Instance does not exist.", nil)
		return
	}
	next := ""
	if len(items) > size {
		items = items[:size]
		next = strconv.Itoa(offset + size)
	}
	if len(parts) == 2 {
		writeJSON(w, 200, map[string]any{"service": svc.Name, "environment": env.Key, "instances": items, "nextPageToken": next})
		return
	}
	if len(parts) == 3 {
		writeJSON(w, 200, items[0])
		return
	}
	endpoints, token, err := storage.NewEndpointRepository(a.db).List(r.Context(), items[0].ID, size, r.URL.Query().Get("pageToken"))
	if err != nil {
		publicStorageError(w, err)
		return
	}
	result := []publicEndpoint{}
	for _, ep := range endpoints {
		result = append(result, publicEndpoint{ID: ep.Id, Name: ep.Name, Protocol: strings.ToLower(strings.TrimPrefix(ep.Protocol.String(), "PROTOCOL_")), Port: ep.Port, Path: ep.Path, Primary: ep.Primary, Enabled: ep.Enabled, Address: endpointAddress(ep.Protocol, ep.Path, items[0].Address, ep.Port), Tags: ep.Tags, Metadata: ep.Metadata})
	}
	writeJSON(w, 200, map[string]any{"service": svc.Name, "environment": env.Key, "instance": items[0].Name, "endpoints": result, "nextPageToken": token})
}

func createPublicEnvironment(w http.ResponseWriter, r *http.Request, repo *storage.EnvironmentRepository) {
	var input struct {
		Key         string            `json:"key"`
		Name        string            `json:"name"`
		Description string            `json:"description"`
		Tags        map[string]string `json:"tags"`
	}
	if !decodePublic(w, r, &input) {
		return
	}
	if !validPublicKey(input.Key) {
		publicError(w, 400, "validation_failed", "Invalid Environment key.", map[string][]string{"key": {"Use 1..128 letters, digits, dots, underscores or hyphens."}})
		return
	}
	if input.Name == "" {
		input.Name = input.Key
	}
	item, err := repo.Create(r.Context(), &v1.CreateEnvironmentRequest{Key: input.Key, Name: input.Name, Description: input.Description, Tags: input.Tags})
	if err != nil {
		publicStorageError(w, err)
		return
	}
	writeJSON(w, 201, publicEnvironmentData(item))
}

func publicEnvironmentData(item *v1.Environment) map[string]any {
	return map[string]any{"id": item.Id, "key": item.Key, "name": item.Name, "description": item.Description, "enabled": item.Enabled, "tier": item.Tier, "tags": item.Tags, "createdAt": item.CreatedAt.AsTime().UTC().Format(time.RFC3339Nano), "updatedAt": item.UpdatedAt.AsTime().UTC().Format(time.RFC3339Nano)}
}
