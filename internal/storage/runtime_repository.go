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
	"github.com/google/uuid"
)

type RuntimeRepository struct {
	db *Database
}

var ErrInvalidRuntimeRegistration = errors.New("invalid runtime registration")

func NewRuntimeRepository(db *Database) *RuntimeRepository {
	return &RuntimeRepository{db: db}
}

func (r *RuntimeRepository) RegisterRuntime(ctx context.Context, req *registryv1.RegisterRuntimeRequest) (*registryv1.RegisterRuntimeResponse, error) {
	if err := validateRuntimeRegistration(req); err != nil {
		return nil, err
	}

	tx, err := r.db.GetDB().BeginTx(ctx, nil)
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

func (r *RuntimeRepository) resolveDeployment(ctx context.Context, tx *sql.Tx, serviceID, environmentID string) (*registryv1.ServiceDeployment, error) {
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

func (r *RuntimeRepository) createRuntimeInstance(ctx context.Context, tx *sql.Tx, deploymentID string, req *registryv1.RuntimeInstanceRegistration) (*registryv1.ServiceInstance, error) {
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

func (r *RuntimeRepository) createRuntimeEndpoint(ctx context.Context, tx *sql.Tx, instanceID string, req *registryv1.RuntimeEndpointRegistration, primary bool) (*registryv1.Endpoint, error) {
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
