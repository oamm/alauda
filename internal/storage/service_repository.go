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

type ServiceRepository struct {
	db *Database
}

func NewServiceRepository(db *Database) *ServiceRepository {
	return &ServiceRepository{db: db}
}

func (r *ServiceRepository) Create(ctx context.Context, req *registryv1.CreateServiceRequest) (*registryv1.Service, error) {
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
		INSERT INTO services (id, name, display_name, description, tags, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.GetName(), req.GetDisplayName(), req.GetDescription(), string(tagsJSON), string(metadataJSON), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *ServiceRepository) Get(ctx context.Context, id string) (*registryv1.Service, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, display_name, description, tags, metadata, created_at, updated_at
		FROM services
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanService(row)
}

func (r *ServiceRepository) GetByName(ctx context.Context, name string) (*registryv1.Service, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, display_name, description, tags, metadata, created_at, updated_at
		FROM services
		WHERE name = ? AND deleted_at IS NULL
	`, name)
	return scanService(row)
}

func (r *ServiceRepository) List(ctx context.Context, environmentID string, pageSize int, pageToken string) ([]*registryv1.Service, string, error) {
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
		SELECT DISTINCT s.id, s.name, s.display_name, s.description, s.tags, s.metadata, s.created_at, s.updated_at
		FROM services s
	`
	args := make([]any, 0, 3)
	if environmentID != "" {
		query += `
			JOIN service_deployments d ON d.service_id = s.id AND d.deleted_at IS NULL
			WHERE s.deleted_at IS NULL AND d.environment_id = ?
		`
		args = append(args, environmentID)
	} else {
		query += ` WHERE s.deleted_at IS NULL`
	}
	query += ` ORDER BY s.created_at ASC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.Service, 0, pageSize)
	for rows.Next() {
		svc, err := scanService(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, svc)
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

func (r *ServiceRepository) Update(ctx context.Context, req *registryv1.UpdateServiceRequest) (*registryv1.Service, error) {
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
		UPDATE services
		SET display_name = ?, description = ?, tags = ?, metadata = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, req.GetDisplayName(), req.GetDescription(), string(tagsJSON), string(metadataJSON), now, req.GetId())
	if err != nil {
		return nil, err
	}

	return r.Get(ctx, req.GetId())
}

func (r *ServiceRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE services
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func scanService(row scanner) (*registryv1.Service, error) {
	var (
		id          string
		name        string
		displayName string
		description string
		tagsRaw     string
		metadataRaw string
		createdRaw  string
		updatedRaw  string
	)

	if err := row.Scan(&id, &name, &displayName, &description, &tagsRaw, &metadataRaw, &createdRaw, &updatedRaw); err != nil {
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

	return &registryv1.Service{
		Id:          id,
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Tags:        tags,
		Metadata:    metadata,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
