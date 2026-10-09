package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/address"
	"github.com/google/uuid"
)

type EndpointRepository struct {
	db *Database
}

var ErrInvalidEndpoint = errors.New("invalid endpoint fields")

func NewEndpointRepository(db *Database) *EndpointRepository {
	return &EndpointRepository{db: db}
}

func (r *EndpointRepository) Create(ctx context.Context, req *registryv1.CreateEndpointRequest) (*registryv1.Endpoint, error) {
	if err := address.ValidateEndpoint(req.GetKind().String(), req.GetPort(), req.GetPath()); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEndpoint, err)
	}
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tagsJSON, err := json.Marshal(req.GetTags())
	if err != nil {
		return nil, err
	}
	metadataJSON, err := json.Marshal(req.GetMetadata())
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE endpoints
		SET primary_endpoint = FALSE, updated_at = ?
		WHERE instance_id = ? AND deleted_at IS NULL AND ? = TRUE
	`, now, req.GetInstanceId(), req.GetPrimary())
	if err != nil {
		return nil, fmt.Errorf("failed to clear existing primary endpoint: %w", err)
	}

	// Deleted rows retain their unique name; restore them without changing identity.
	var deletedID string
	err = tx.QueryRowContext(ctx, `
		SELECT id FROM endpoints
		WHERE instance_id = ? AND name = ? AND deleted_at IS NOT NULL
	`, req.GetInstanceId(), req.GetName()).Scan(&deletedID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("failed to find deleted endpoint: %w", err)
	}
	if err == nil {
		id = deletedID
		_, err = tx.ExecContext(ctx, `
			UPDATE endpoints
			SET kind = ?, port = ?, path = ?, enabled = ?, tags = ?, metadata = ?,
			    primary_endpoint = ?, deleted_at = NULL, updated_at = ?
			WHERE id = ? AND deleted_at IS NOT NULL
		`, int32(req.GetKind()), req.GetPort(), req.GetPath(), req.GetEnabled(),
			string(tagsJSON), string(metadataJSON), req.GetPrimary(), now, id)
	} else {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO endpoints (id, instance_id, name, kind, port, path, enabled, tags, metadata, primary_endpoint, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, req.GetInstanceId(), req.GetName(), int32(req.GetKind()),
			req.GetPort(), req.GetPath(), req.GetEnabled(), string(tagsJSON), string(metadataJSON), req.GetPrimary(), now, now)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *EndpointRepository) Get(ctx context.Context, id string) (*registryv1.Endpoint, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, instance_id, name, kind, port, path, enabled, tags, metadata, primary_endpoint
		FROM endpoints
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanEndpoint(row)
}

func (r *EndpointRepository) List(ctx context.Context, instanceID string, pageSize int, pageToken string) ([]*registryv1.Endpoint, string, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := 0
	if pageToken != "" {
		parsed, err := strconv.Atoi(pageToken)
		if err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	query := `
		SELECT id, instance_id, name, kind, port, path, enabled, tags, metadata, primary_endpoint
		FROM endpoints
		WHERE deleted_at IS NULL
	`
	args := make([]interface{}, 0, 3)

	if instanceID != "" {
		query += ` AND instance_id = ?`
		args = append(args, instanceID)
	}

	query, args = appendEnvironmentAccess(ctx, query, "(SELECT d.environment_id FROM service_instances i JOIN service_deployments d ON d.id=i.deployment_id WHERE i.id=endpoints.instance_id)", args)
	query += ` ORDER BY julianday(created_at) ASC LIMIT ? OFFSET ?`
	args = append(args, pageSize+1, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to list endpoints: %w", err)
	}
	defer rows.Close()

	endpoints := []*registryv1.Endpoint{}
	for rows.Next() {
		endpoint, err := scanEndpoint(rows)
		if err != nil {
			return nil, "", fmt.Errorf("failed to scan endpoint: %w", err)
		}
		endpoints = append(endpoints, endpoint)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextToken := ""
	if len(endpoints) > pageSize {
		endpoints = endpoints[:pageSize]
		nextToken = fmt.Sprintf("%d", offset+pageSize)
	}

	return endpoints, nextToken, nil
}

func (r *EndpointRepository) Update(ctx context.Context, req *registryv1.UpdateEndpointRequest) (*registryv1.Endpoint, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	query := `
		UPDATE endpoints
		SET name = COALESCE(?, name),
		    kind = COALESCE(?, kind),
		    port = COALESCE(?, port),
		    path = COALESCE(?, path),
		    enabled = ?,
		    primary_endpoint = COALESCE(?, primary_endpoint),
		    updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`
	var name *string
	if req.GetName() != "" {
		name = &req.Name
	}
	var proto *int32
	if req.GetKind() != 0 {
		p := int32(req.GetKind())
		proto = &p
	}
	var port *int32
	if req.GetPort() != 0 {
		port = &req.Port
	}
	var path *string
	if req.GetPath() != "" {
		path = &req.Path
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var instanceID string
	current, err := scanEndpoint(tx.QueryRowContext(ctx, `SELECT id, instance_id, name, kind, port, path, enabled, tags, metadata, primary_endpoint FROM endpoints WHERE id=? AND deleted_at IS NULL`, req.GetId()))
	if err != nil {
		return nil, err
	}
	kind, endpointPort, endpointPath := current.Kind, current.Port, address.PublicPath(current.Kind.String(), current.Path)
	if req.GetKind() != 0 {
		kind = req.GetKind()
	}
	if req.GetPort() != 0 {
		endpointPort = req.GetPort()
	}
	if req.GetPath() != "" {
		endpointPath = req.GetPath()
	}
	capabilities, capabilityError := address.Capabilities(kind.String())
	if capabilityError == nil && !capabilities.SupportsPath && req.GetPath() == "" {
		endpointPath = ""
	}
	if err := address.ValidateEndpoint(kind.String(), endpointPort, endpointPath); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidEndpoint, err)
	}
	path = &endpointPath
	var primary *bool
	if req.Primary != nil {
		primary = req.Primary
	}
	if primary != nil && *primary {
		if err := tx.QueryRowContext(ctx, `
			SELECT instance_id FROM endpoints WHERE id = ? AND deleted_at IS NULL
		`, req.GetId()).Scan(&instanceID); err != nil {
			return nil, fmt.Errorf("failed to load endpoint instance: %w", err)
		}
		_, err := tx.ExecContext(ctx, `
			UPDATE endpoints
			SET primary_endpoint = FALSE, updated_at = ?
			WHERE instance_id = ? AND id != ? AND deleted_at IS NULL
		`, now, instanceID, req.GetId())
		if err != nil {
			return nil, fmt.Errorf("failed to clear existing primary endpoint: %w", err)
		}
	}

	_, err = tx.ExecContext(ctx, query, name, proto, port, path, req.GetEnabled(), primary, now, req.GetId())
	if err != nil {
		return nil, fmt.Errorf("failed to update endpoint: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, req.GetId())
}

func (r *EndpointRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.Exec(ctx, `
		UPDATE endpoints
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("failed to delete endpoint: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanEndpoint(scanner interface{ Scan(...interface{}) error }) (*registryv1.Endpoint, error) {
	var (
		id           string
		instanceID   string
		name         string
		kind         int32
		port         int32
		path         string
		enabled      bool
		tagsJSON     string
		metadataJSON string
		primary      bool
	)

	err := scanner.Scan(&id, &instanceID, &name, &kind, &port, &path, &enabled, &tagsJSON, &metadataJSON, &primary)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to scan endpoint: %w", err)
	}

	endpoint := &registryv1.Endpoint{
		Id:         id,
		InstanceId: instanceID,
		Name:       name,
		Kind:       registryv1.EndpointKind(kind),
		Port:       port,
		Path:       path,
		Enabled:    enabled,
		Tags:       make(map[string]string),
		Metadata:   make(map[string]string),
		Primary:    primary,
	}

	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &endpoint.Tags)
	}
	if metadataJSON != "" {
		json.Unmarshal([]byte(metadataJSON), &endpoint.Metadata)
	}

	return endpoint, nil
}
