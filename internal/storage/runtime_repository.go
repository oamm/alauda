package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/address"
	"github.com/company/service-registry/internal/contract"
	"github.com/google/uuid"
)

type RuntimeRepository struct {
	db *Database
}

var ErrInvalidRuntimeRegistration = errors.New("invalid runtime registration")

type PublicRegistrationValidationError struct{ Fields map[string][]string }

func (e *PublicRegistrationValidationError) Error() string { return "invalid registration fields" }
func (e *PublicRegistrationValidationError) Unwrap() error { return ErrInvalidRuntimeRegistration }

func NewRuntimeRepository(db *Database) *RuntimeRepository {
	return &RuntimeRepository{db: db}
}

func (r *RuntimeRepository) RegisterRuntime(ctx context.Context, req *registryv1.RegisterRuntimeRequest) (*registryv1.RegisterRuntimeResponse, error) {
	if err := validateRuntimeRegistration(req); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	deployment, err := r.resolveDeployment(ctx, tx, req.GetServiceId(), req.GetEnvironmentId())
	if err != nil {
		return nil, err
	}

	instance, err := r.createRuntimeInstance(ctx, tx, deployment.GetId(), req.GetInstance())
	if err != nil {
		return nil, err
	}

	endpoints := make([]*registryv1.Endpoint, 0, len(req.GetEndpoints()))
	autoPrimary := len(req.GetEndpoints()) == 1 && !req.GetEndpoints()[0].GetPrimary()
	for i, endpointReq := range req.GetEndpoints() {
		primary := endpointReq.GetPrimary() || (autoPrimary && i == 0)
		endpoint, err := r.createRuntimeEndpoint(ctx, tx, instance.GetId(), endpointReq, primary)
		if err != nil {
			return nil, err
		}
		endpoints = append(endpoints, endpoint)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &registryv1.RegisterRuntimeResponse{
		Deployment: deployment,
		Instance:   instance,
		Endpoints:  endpoints,
	}, nil
}

// RegisterRuntimeUpsert is the public registration primitive. Its identity is
// service + environment + instance name; endpoint omission preserves existing
// endpoints unless replaceEndpoints is explicitly requested.
// RegisterRuntimeUpsert preserves compatibility for internal callers supplying full fields.
func (r *RuntimeRepository) RegisterRuntimeUpsert(ctx context.Context, req *registryv1.RegisterRuntimeRequest, replace bool) (*registryv1.RegisterRuntimeResponse, error) {
	p := contract.Registration{Replace: replace, Instance: contract.InstancePatch{Name: req.GetInstance().GetName(), Address: contract.Pointer(req.GetInstance().GetAddress()), Description: contract.Pointer(req.GetInstance().GetDescription()), Enabled: contract.Pointer(req.GetInstance().GetEnabled()), Tags: req.GetInstance().GetTags(), Metadata: req.GetInstance().GetMetadata()}}
	for _, e := range req.GetEndpoints() {
		p.Endpoints = append(p.Endpoints, contract.EndpointPatch{Name: e.Name, Protocol: contract.Pointer(strings.ToLower(strings.TrimPrefix(e.Protocol.String(), "PROTOCOL_"))), Port: contract.Pointer(e.Port), Path: contract.Pointer(e.Path), Primary: contract.Pointer(e.Primary), Enabled: contract.Pointer(e.Enabled), Tags: e.Tags, Metadata: e.Metadata})
	}
	return r.RegisterPublicRuntime(ctx, req.ServiceId, req.EnvironmentId, p)
}

// RegisterPublicRuntime merges presence-aware fields inside the existing transaction.
// Serialization prevents competing read-then-write snapshots within this database.
func (r *RuntimeRepository) RegisterPublicRuntime(ctx context.Context, serviceID, environmentID string, patch contract.Registration) (*registryv1.RegisterRuntimeResponse, error) {
	unlock := r.db.AcquireRegistrationLock()
	defer unlock()
	for attempt := 0; ; attempt++ {
		result, err := r.registerPublicRuntimeOnce(ctx, serviceID, environmentID, patch)
		if err == nil || attempt >= 9 || ClassifyError(err) != ErrorUnavailable {
			return result, err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * 10 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func (r *RuntimeRepository) registerPublicRuntimeOnce(ctx context.Context, serviceID, environmentID string, patch contract.Registration) (*registryv1.RegisterRuntimeResponse, error) {
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	deployment, err := r.resolveDeployment(ctx, tx, serviceID, environmentID)
	if err != nil {
		return nil, err
	}
	if err := r.db.LockRegistrationScope(ctx, tx, deployment.GetId()); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	selectInstance := `SELECT id, deployment_id, name, address, port, description, enabled, tags, metadata, created_at, updated_at, last_seen_at FROM service_instances WHERE deployment_id=? AND name=?`
	existing, err := scanInstance(tx.QueryRowContext(ctx, selectInstance, deployment.Id, patch.Instance.Name))
	newInstance := errors.Is(err, sql.ErrNoRows)
	if err != nil && !newInstance {
		return nil, err
	}
	if newInstance {
		existing = &registryv1.ServiceInstance{Name: patch.Instance.Name, Enabled: true}
	}
	instance := &registryv1.RuntimeInstanceRegistration{Name: patch.Instance.Name, Address: existing.Address, Description: existing.Description, Enabled: existing.Enabled, Tags: existing.Tags, Metadata: existing.Metadata}
	if patch.Instance.Address != nil {
		instance.Address = *patch.Instance.Address
	}
	if patch.Instance.Description != nil {
		instance.Description = *patch.Instance.Description
	}
	if patch.Instance.Enabled != nil {
		instance.Enabled = *patch.Instance.Enabled
	}
	if patch.Instance.Tags != nil {
		instance.Tags = patch.Instance.Tags
	}
	if patch.Instance.Metadata != nil {
		instance.Metadata = patch.Instance.Metadata
	}
	merged := &registryv1.RegisterRuntimeRequest{ServiceId: serviceID, EnvironmentId: environmentID, Instance: instance}
	primaries := 0
	for _, ep := range patch.Endpoints {
		old, err := scanEndpoint(tx.QueryRowContext(ctx, `SELECT id, instance_id, name, protocol, port, path, enabled, tags, metadata, primary_endpoint FROM endpoints WHERE instance_id=? AND name=?`, existing.Id, ep.Name))
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if old == nil {
			old = &registryv1.Endpoint{Name: ep.Name, Enabled: true}
		}
		item := &registryv1.RuntimeEndpointRegistration{Name: ep.Name, Protocol: old.Protocol, Port: old.Port, Path: address.PublicPath(old.Protocol.String(), old.Path), Enabled: old.Enabled, Primary: old.Primary, Tags: old.Tags, Metadata: old.Metadata}
		if ep.Protocol != nil {
			value, ok := registryv1.Protocol_value["PROTOCOL_"+strings.ToUpper(*ep.Protocol)]
			if !ok {
				return nil, ErrInvalidRuntimeRegistration
			}
			item.Protocol = registryv1.Protocol(value)
		}
		if ep.Port != nil {
			item.Port = *ep.Port
		}
		if ep.Path != nil {
			item.Path = *ep.Path
		} else {
			item.Path = address.PublicPath(item.Protocol.String(), item.Path)
		}
		if ep.Enabled != nil {
			item.Enabled = *ep.Enabled
		}
		if ep.Tags != nil {
			item.Tags = ep.Tags
		}
		if ep.Metadata != nil {
			item.Metadata = ep.Metadata
		}
		if ep.Primary != nil {
			item.Primary = *ep.Primary
		} else if newInstance && len(patch.Endpoints) == 1 {
			item.Primary = true
		}
		// An explicit promotion demotes any preserved Primary regardless of request order.
		if ep.Primary != nil && *ep.Primary {
			primaries++
			if primaries > 1 {
				return nil, ErrInvalidRuntimeRegistration
			}
		}
		merged.Endpoints = append(merged.Endpoints, item)
	}
	if primaries == 1 {
		for i, ep := range patch.Endpoints {
			if ep.Primary == nil || !*ep.Primary {
				merged.Endpoints[i].Primary = false
			}
		}
	}
	fields := map[string][]string{}
	normalizedAddress, err := address.NormalizeHost(instance.Address)
	if err != nil {
		fields["instance.address"] = []string{err.Error()}
	} else {
		instance.Address = normalizedAddress
	}
	for i, ep := range merged.Endpoints {
		prefix := fmt.Sprintf("endpoints[%d]", i)
		if err := address.ValidateEndpointPath(ep.Protocol.String(), ep.Path); err != nil {
			fields[prefix+".path"] = []string{err.Error()}
		}
		if ep.Protocol == registryv1.Protocol_PROTOCOL_UNSPECIFIED {
			fields[prefix+".protocol"] = []string{"Protocol is required for a new Endpoint."}
		}
		if ep.Port < 1 || ep.Port > 65535 {
			fields[prefix+".port"] = []string{"Port must be between 1 and 65535."}
		}
	}
	if len(fields) > 0 {
		return nil, &PublicRegistrationValidationError{Fields: fields}
	}
	if err := validateRuntimeRegistration(merged); err != nil {
		return nil, err
	}
	for _, ep := range merged.Endpoints {
		if _, err := address.Build(strings.ToLower(strings.TrimPrefix(ep.Protocol.String(), "PROTOCOL_")), instance.Address, ep.Port, ep.Path); err != nil {
			return nil, ErrInvalidRuntimeRegistration
		}
	}
	if newInstance {
		existing, err = r.createRuntimeInstance(ctx, tx, deployment.Id, instance)
		if err != nil {
			return nil, err
		}
	}
	tags, _ := json.Marshal(instance.Tags)
	metadata, _ := json.Marshal(instance.Metadata)
	_, err = tx.ExecContext(ctx, `UPDATE service_instances SET address=?, description=?, enabled=?, tags=?, metadata=?, deleted_at=NULL, updated_at=? WHERE id=?`, instance.Address, instance.Description, instance.Enabled, string(tags), string(metadata), now, existing.Id)
	if err != nil {
		return nil, err
	}
	if patch.Replace {
		_, err = tx.ExecContext(ctx, `UPDATE endpoints SET deleted_at=?,updated_at=? WHERE instance_id=? AND deleted_at IS NULL`, now, now, existing.Id)
		if err != nil {
			return nil, err
		}
	}
	// Clear before writes to satisfy the immediate partial unique index.
	promote := false
	for _, ep := range merged.Endpoints {
		promote = promote || ep.Primary
	}
	if promote {
		_, err = tx.ExecContext(ctx, `UPDATE endpoints SET primary_endpoint=FALSE,updated_at=? WHERE instance_id=? AND deleted_at IS NULL`, now, existing.Id)
		if err != nil {
			return nil, err
		}
	}
	for _, ep := range merged.Endpoints {
		if _, err = r.upsertRuntimeEndpoint(ctx, tx, existing.Id, ep, ep.Primary); err != nil {
			return nil, err
		}
	}
	items := []*registryv1.Endpoint{}
	rows, err := tx.QueryContext(ctx, `SELECT id, instance_id, name, protocol, port, path, enabled, tags, metadata, primary_endpoint FROM endpoints WHERE instance_id=? AND deleted_at IS NULL ORDER BY name`, existing.Id)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		item, e := scanEndpoint(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		items = append(items, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	existing, err = scanInstance(tx.QueryRowContext(ctx, selectInstance, deployment.Id, patch.Instance.Name))
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &registryv1.RegisterRuntimeResponse{Deployment: deployment, Instance: existing, Endpoints: items}, nil
}

func (r *RuntimeRepository) upsertRuntimeEndpoint(ctx context.Context, tx transaction, instanceID string, req *registryv1.RuntimeEndpointRegistration, primary bool) (*registryv1.Endpoint, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tagsJSON, _ := json.Marshal(req.GetTags())
	metadataJSON, _ := json.Marshal(req.GetMetadata())
	var id string
	err := tx.QueryRowContext(ctx, `SELECT id FROM endpoints WHERE instance_id=? AND name=? AND deleted_at IS NULL`, instanceID, req.GetName()).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `SELECT id FROM endpoints WHERE instance_id=? AND name=? AND deleted_at IS NOT NULL`, instanceID, req.GetName()).Scan(&id)
		if errors.Is(err, sql.ErrNoRows) {
			id = uuid.NewString()
			_, err = tx.ExecContext(ctx, `INSERT INTO endpoints (id, instance_id, name, protocol, port, path, enabled, tags, metadata, primary_endpoint, created_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`, id, instanceID, req.GetName(), int32(req.GetProtocol()), req.GetPort(), req.GetPath(), req.GetEnabled(), string(tagsJSON), string(metadataJSON), primary, now, now)
		} else if err == nil {
			_, err = tx.ExecContext(ctx, `UPDATE endpoints SET protocol=?, port=?, path=?, enabled=?, tags=?, metadata=?, primary_endpoint=?, deleted_at=NULL, updated_at=? WHERE id=?`, int32(req.GetProtocol()), req.GetPort(), req.GetPath(), req.GetEnabled(), string(tagsJSON), string(metadataJSON), primary, now, id)
		}
	} else if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE endpoints SET protocol=?, port=?, path=?, enabled=?, tags=?, metadata=?, primary_endpoint=?, updated_at=? WHERE id=?`, int32(req.GetProtocol()), req.GetPort(), req.GetPath(), req.GetEnabled(), string(tagsJSON), string(metadataJSON), primary, now, id)
	}
	if err != nil {
		return nil, err
	}
	if primary {
		if _, err = tx.ExecContext(ctx, `UPDATE endpoints SET primary_endpoint=FALSE, updated_at=? WHERE instance_id=? AND id != ? AND deleted_at IS NULL`, now, instanceID, id); err != nil {
			return nil, err
		}
	}
	return scanEndpoint(tx.QueryRowContext(ctx, `SELECT id, instance_id, name, protocol, port, path, enabled, tags, metadata, primary_endpoint FROM endpoints WHERE id=? AND deleted_at IS NULL`, id))
}

func validateRuntimeRegistration(req *registryv1.RegisterRuntimeRequest) error {
	if req.GetServiceId() == "" || req.GetEnvironmentId() == "" {
		return fmt.Errorf("%w: service_id and environment_id are required", ErrInvalidRuntimeRegistration)
	}
	instance := req.GetInstance()
	if instance == nil || strings.TrimSpace(instance.GetName()) == "" || strings.TrimSpace(instance.GetAddress()) == "" {
		return fmt.Errorf("%w: instance name and address are required", ErrInvalidRuntimeRegistration)
	}

	primaryCount := 0
	for _, endpoint := range req.GetEndpoints() {
		if err := address.ValidateEndpoint(endpoint.GetProtocol().String(), endpoint.GetPort(), endpoint.GetPath()); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidRuntimeRegistration, err)
		}
		if strings.TrimSpace(endpoint.GetName()) == "" {
			return fmt.Errorf("%w: endpoint name is required", ErrInvalidRuntimeRegistration)
		}
		if endpoint.GetProtocol() == registryv1.Protocol_PROTOCOL_UNSPECIFIED {
			return fmt.Errorf("%w: endpoint protocol is required", ErrInvalidRuntimeRegistration)
		}
		if endpoint.GetPort() < 1 || endpoint.GetPort() > 65535 {
			return fmt.Errorf("%w: endpoint port must be between 1 and 65535", ErrInvalidRuntimeRegistration)
		}
		if endpoint.GetPrimary() {
			primaryCount++
		}
	}
	if primaryCount > 1 {
		return fmt.Errorf("%w: at most one endpoint can be primary", ErrInvalidRuntimeRegistration)
	}
	return nil
}

func (r *RuntimeRepository) resolveDeployment(ctx context.Context, tx transaction, serviceID, environmentID string) (*registryv1.ServiceDeployment, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, service_id, environment_id, health_enabled, alerts_enabled, alert_cooldown_minutes,
		       tags, metadata, created_at, updated_at
		FROM service_deployments
		WHERE service_id = ? AND environment_id = ? AND deleted_at IS NULL
	`, serviceID, environmentID)
	deployment, err := scanDeployment(row)
	if err == nil {
		return deployment, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	// A retired association still owns its unique identity; restore it atomically.
	result, err := tx.ExecContext(ctx, `UPDATE service_deployments SET deleted_at=NULL, updated_at=? WHERE service_id=? AND environment_id=? AND deleted_at IS NOT NULL`, time.Now().UTC().Format(time.RFC3339Nano), serviceID, environmentID)
	if err != nil {
		return nil, err
	}
	if count, _ := result.RowsAffected(); count > 0 {
		return r.resolveDeployment(ctx, tx, serviceID, environmentID)
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	emptyJSON := "{}"
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO service_deployments (
			id, service_id, environment_id, health_enabled, alerts_enabled, alert_cooldown_minutes,
			tags, metadata, created_at, updated_at
		)
		VALUES (?, ?, ?, TRUE, TRUE, 10, ?, ?, ?, ?)
	`, id, serviceID, environmentID, emptyJSON, emptyJSON, now, now); err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	row = tx.QueryRowContext(ctx, `
		SELECT id, service_id, environment_id, health_enabled, alerts_enabled, alert_cooldown_minutes,
		       tags, metadata, created_at, updated_at
		FROM service_deployments
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanDeployment(row)
}

func (r *RuntimeRepository) createRuntimeInstance(ctx context.Context, tx transaction, deploymentID string, req *registryv1.RuntimeInstanceRegistration) (*registryv1.ServiceInstance, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := uuid.NewString()
	tagsJSON, err := json.Marshal(req.GetTags())
	if err != nil {
		return nil, err
	}
	metadataJSON, err := json.Marshal(req.GetMetadata())
	if err != nil {
		return nil, err
	}
	enabled := req.GetEnabled()
	if !enabled {
		enabled = true
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO service_instances (
			id, deployment_id, name, address, port, description, enabled, tags, metadata, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, 0, ?, ?, ?, ?, ?, ?)
	`, id, deploymentID, req.GetName(), req.GetAddress(), req.GetDescription(), enabled, string(tagsJSON), string(metadataJSON), now, now); err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	row := tx.QueryRowContext(ctx, `
		SELECT id, deployment_id, name, address, port, description, enabled, tags, metadata, created_at, updated_at, last_seen_at
		FROM service_instances
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanInstance(row)
}

func (r *RuntimeRepository) createRuntimeEndpoint(ctx context.Context, tx transaction, instanceID string, req *registryv1.RuntimeEndpointRegistration, primary bool) (*registryv1.Endpoint, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	id := uuid.NewString()
	tagsJSON, err := json.Marshal(req.GetTags())
	if err != nil {
		return nil, err
	}
	metadataJSON, err := json.Marshal(req.GetMetadata())
	if err != nil {
		return nil, err
	}
	enabled := req.GetEnabled()

	if primary {
		if _, err := tx.ExecContext(ctx, `
			UPDATE endpoints
			SET primary_endpoint = FALSE, updated_at = ?
			WHERE instance_id = ? AND deleted_at IS NULL
		`, now, instanceID); err != nil {
			return nil, fmt.Errorf("failed to clear existing primary endpoint: %w", err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO endpoints (
			id, instance_id, name, protocol, port, path, enabled, tags, metadata, primary_endpoint, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, instanceID, req.GetName(), int32(req.GetProtocol()), req.GetPort(), req.GetPath(), enabled, string(tagsJSON), string(metadataJSON), primary, now, now); err != nil {
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}

	row := tx.QueryRowContext(ctx, `
		SELECT id, instance_id, name, protocol, port, path, enabled, tags, metadata, primary_endpoint
		FROM endpoints
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanEndpoint(row)
}
