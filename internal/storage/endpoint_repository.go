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
	"github.com/google/uuid"
)

type EndpointRepository struct {
	db *Database
}

func NewEndpointRepository(db *Database) *EndpointRepository {
	return &EndpointRepository{db: db}
}

func (r *EndpointRepository) Create(ctx context.Context, req *registryv1.CreateEndpointRequest) (*registryv1.Endpoint, error) {
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

	_, err = r.db.Exec(ctx, `
		INSERT INTO endpoints (id, instance_id, name, protocol, port, path, enabled, tags, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.GetInstanceId(), req.GetName(), int32(req.GetProtocol()),
		req.GetPort(), req.GetPath(), req.GetEnabled(), string(tagsJSON), string(metadataJSON), now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create endpoint: %w", err)
	}
	return r.Get(ctx, id)
}

func (r *EndpointRepository) Get(ctx context.Context, id string) (*registryv1.Endpoint, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, instance_id, name, protocol, port, path, enabled, tags, metadata
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
		SELECT id, instance_id, name, protocol, port, path, enabled, tags, metadata
		FROM endpoints
		WHERE deleted_at IS NULL
	`
	args := make([]interface{}, 0, 3)

	if instanceID != "" {
		query += ` AND instance_id = ?`
		args = append(args, instanceID)
	}

	query += ` ORDER BY created_at ASC LIMIT ? OFFSET ?`
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
		    protocol = COALESCE(?, protocol),
		    port = COALESCE(?, port),
		    path = COALESCE(?, path),
		    enabled = ?,
		    updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`
	var name *string
	if req.GetName() != "" {
		name = &req.Name
	}
	var proto *int32
	if req.GetProtocol() != 0 {
		p := int32(req.GetProtocol())
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

	_, err := r.db.Exec(ctx, query, name, proto, port, path, req.GetEnabled(), now, req.GetId())
	if err != nil {
		return nil, fmt.Errorf("failed to update endpoint: %w", err)
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
		protocol     int32
		port         int32
		path         string
		enabled      bool
		tagsJSON     string
		metadataJSON string
	)

	err := scanner.Scan(&id, &instanceID, &name, &protocol, &port, &path, &enabled, &tagsJSON, &metadataJSON)
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
		Protocol:   registryv1.Protocol(protocol),
		Port:       port,
		Path:       path,
		Enabled:    enabled,
		Tags:       make(map[string]string),
		Metadata:   make(map[string]string),
	}

	if tagsJSON != "" {
		json.Unmarshal([]byte(tagsJSON), &endpoint.Tags)
	}
	if metadataJSON != "" {
		json.Unmarshal([]byte(metadataJSON), &endpoint.Metadata)
	}

	return endpoint, nil
}
