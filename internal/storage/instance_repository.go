package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type InstanceRepository struct {
	db *Database
}

func NewInstanceRepository(db *Database) *InstanceRepository {
	return &InstanceRepository{db: db}
}

func (r *InstanceRepository) Create(ctx context.Context, req *registryv1.CreateInstanceRequest) (*registryv1.ServiceInstance, error) {
	now := time.Now().UTC()
	id := uuid.NewString()
	tagsJSON, err := json.Marshal(req.GetTags())
	if err != nil {
		return nil, err
	}
	metadataJSON, err := json.Marshal(req.GetMetadata())
	if err != nil {
		return nil, err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO service_instances (
			id, deployment_id, name, address, port, description, enabled, tags, metadata, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.GetDeploymentId(), req.GetName(), req.GetAddress(), req.GetPort(), req.GetDescription(), req.GetEnabled(), string(tagsJSON), string(metadataJSON), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *InstanceRepository) Get(ctx context.Context, id string) (*registryv1.ServiceInstance, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, deployment_id, name, address, port, description, enabled, tags, metadata, created_at, updated_at, last_seen_at
		FROM service_instances
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanInstance(row)
}

func (r *InstanceRepository) List(ctx context.Context, deploymentID string, pageSize int, pageToken string) ([]*registryv1.ServiceInstance, string, error) {
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
		SELECT id, deployment_id, name, address, port, description, enabled, tags, metadata, created_at, updated_at, last_seen_at
		FROM service_instances
		WHERE deleted_at IS NULL
	`
	args := make([]any, 0, 3)
	if deploymentID != "" {
		query += ` AND deployment_id = ?`
		args = append(args, deploymentID)
	}
	query += ` ORDER BY created_at ASC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.ServiceInstance, 0, pageSize)
	for rows.Next() {
		item, err := scanInstance(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}

	nextToken := ""
	if len(items) == pageSize {
		nextToken = strconv.Itoa(offset + len(items))
	}
	return items, nextToken, nil
}

func (r *InstanceRepository) Update(ctx context.Context, req *registryv1.UpdateInstanceRequest) (*registryv1.ServiceInstance, error) {
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
		UPDATE service_instances
		SET address = ?, port = ?, description = ?, enabled = ?, tags = ?, metadata = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, req.GetAddress(), req.GetPort(), req.GetDescription(), req.GetEnabled(), string(tagsJSON), string(metadataJSON), now, req.GetId())
	if err != nil {
		return nil, err
	}

	return r.Get(ctx, req.GetId())
}

func (r *InstanceRepository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(ctx, `
		UPDATE service_instances
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, now, now, id)
	return err
}

func scanInstance(row scanner) (*registryv1.ServiceInstance, error) {
	var (
		id           string
		deploymentID string
		name         string
		address      string
		port         int32
		description  string
		enabled      bool
		tagsRaw      string
		metadataRaw  string
		createdRaw   string
		updatedRaw   string
		lastSeenRaw  sql.NullString
	)

	if err := row.Scan(&id, &deploymentID, &name, &address, &port, &description, &enabled, &tagsRaw, &metadataRaw, &createdRaw, &updatedRaw, &lastSeenRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	tags := map[string]string{}
	if tagsRaw != "" {
		_ = json.Unmarshal([]byte(tagsRaw), &tags)
	}
	metadata := map[string]string{}
	if metadataRaw != "" {
		_ = json.Unmarshal([]byte(metadataRaw), &metadata)
	}

	createdAt := timestamppb.Now()
	if t, err := time.Parse(time.RFC3339Nano, createdRaw); err == nil {
		createdAt = timestamppb.New(t)
	}
	updatedAt := timestamppb.Now()
	if t, err := time.Parse(time.RFC3339Nano, updatedRaw); err == nil {
		updatedAt = timestamppb.New(t)
	}
	var lastSeenAt *timestamppb.Timestamp
	if lastSeenRaw.Valid {
		if t, err := time.Parse(time.RFC3339Nano, lastSeenRaw.String); err == nil {
			lastSeenAt = timestamppb.New(t)
		}
	}

	return &registryv1.ServiceInstance{
		Id:           id,
		DeploymentId: deploymentID,
		Name:         name,
		Address:      address,
		Port:         port,
		Description:  description,
		Enabled:      enabled,
		Tags:         tags,
		Metadata:     metadata,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
		LastSeenAt:   lastSeenAt,
	}, nil
}
