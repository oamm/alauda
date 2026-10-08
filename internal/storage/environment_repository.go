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

type EnvironmentRepository struct {
	db *Database
}

func NewEnvironmentRepository(db *Database) *EnvironmentRepository {
	return &EnvironmentRepository{db: db}
}

func (r *EnvironmentRepository) Create(ctx context.Context, req *registryv1.CreateEnvironmentRequest) (*registryv1.Environment, error) {
	now := time.Now().UTC()
	id := uuid.NewString()
	tagsJSON, err := json.Marshal(req.GetTags())
	if err != nil {
		return nil, err
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO environments (id, key, name, description, enabled, tier, tags, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.GetKey(), req.GetName(), req.GetDescription(), true, req.GetTier(), string(tagsJSON), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}

	return r.Get(ctx, id)
}

func (r *EnvironmentRepository) Get(ctx context.Context, id string) (*registryv1.Environment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, key, name, description, enabled, tier, tags, created_at, updated_at
		FROM environments
		WHERE id = ? AND deleted_at IS NULL
	`, id)

	return scanEnvironment(row)
}

func (r *EnvironmentRepository) GetByKey(ctx context.Context, key string) (*registryv1.Environment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, key, name, description, enabled, tier, tags, created_at, updated_at
		FROM environments
		WHERE key = ? AND deleted_at IS NULL
	`, key)
	return scanEnvironment(row)
}

func (r *EnvironmentRepository) List(ctx context.Context, includeDisabled bool, pageSize int, pageToken string) ([]*registryv1.Environment, string, error) {
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
		SELECT id, key, name, description, enabled, tier, tags, created_at, updated_at
		FROM environments
		WHERE deleted_at IS NULL
	`
	args := make([]any, 0, 2)
	if !includeDisabled {
		query += ` AND enabled = 1`
	}
	query, args = appendEnvironmentAccess(ctx, query, "id", args)
	query += ` ORDER BY created_at ASC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.Environment, 0, pageSize)
	for rows.Next() {
		env, err := scanEnvironment(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, env)
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

func (r *EnvironmentRepository) Update(ctx context.Context, req *registryv1.UpdateEnvironmentRequest) (*registryv1.Environment, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tagsJSON, err := json.Marshal(req.GetTags())
	if err != nil {
		return nil, err
	}

	_, err = r.db.Exec(ctx, `
		UPDATE environments
		SET name = ?, description = ?, enabled = ?, tier = ?, tags = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, req.GetName(), req.GetDescription(), req.GetEnabled(), req.GetTier(), string(tagsJSON), now, req.GetId())
	if err != nil {
		return nil, err
	}

	return r.Get(ctx, req.GetId())
}

func (r *EnvironmentRepository) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE environments
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, time.Now().UTC().Format(time.RFC3339Nano), time.Now().UTC().Format(time.RFC3339Nano), id)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEnvironment(row scanner) (*registryv1.Environment, error) {
	var (
		id          string
		key         string
		name        string
		description string
		enabled     bool
		tier        string
		tagsRaw     string
		createdRaw  string
		updatedRaw  string
	)

	if err := row.Scan(&id, &key, &name, &description, &enabled, &tier, &tagsRaw, &createdRaw, &updatedRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	tags := map[string]string{}
	if tagsRaw != "" {
		_ = json.Unmarshal([]byte(tagsRaw), &tags)
	}

	createdAt := timestamppb.Now()
	if t, err := time.Parse(time.RFC3339Nano, createdRaw); err == nil {
		createdAt = timestamppb.New(t)
	}
	updatedAt := timestamppb.Now()
	if t, err := time.Parse(time.RFC3339Nano, updatedRaw); err == nil {
		updatedAt = timestamppb.New(t)
	}

	return &registryv1.Environment{
		Id:          id,
		Key:         key,
		Name:        name,
		Description: description,
		Enabled:     enabled,
		Tier:        tier,
		Tags:        tags,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}, nil
}
