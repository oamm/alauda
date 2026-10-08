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

type DeploymentRepository struct {
	db *Database
}

func NewDeploymentRepository(db *Database) *DeploymentRepository {
	return &DeploymentRepository{db: db}
}

func (r *DeploymentRepository) Create(ctx context.Context, req *registryv1.CreateDeploymentRequest) (*registryv1.ServiceDeployment, error) {
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
		INSERT INTO service_deployments (
			id, service_id, environment_id, health_enabled, alerts_enabled, alert_cooldown_minutes,
			tags, metadata, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.GetServiceId(), req.GetEnvironmentId(), req.GetHealthEnabled(), req.GetAlertsEnabled(), req.GetAlertCooldownMinutes(), string(tagsJSON), string(metadataJSON), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *DeploymentRepository) Get(ctx context.Context, id string) (*registryv1.ServiceDeployment, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, service_id, environment_id, health_enabled, alerts_enabled, alert_cooldown_minutes,
		       tags, metadata, created_at, updated_at
		FROM service_deployments
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanDeployment(row)
}

func (r *DeploymentRepository) List(ctx context.Context, serviceID, environmentID string, pageSize int, pageToken string) ([]*registryv1.ServiceDeployment, string, error) {
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
		SELECT id, service_id, environment_id, health_enabled, alerts_enabled, alert_cooldown_minutes,
		       tags, metadata, created_at, updated_at
		FROM service_deployments
		WHERE deleted_at IS NULL
	`
	args := make([]any, 0, 4)
	if serviceID != "" {
		query += ` AND service_id = ?`
		args = append(args, serviceID)
	}
	if environmentID != "" {
		query += ` AND environment_id = ?`
		args = append(args, environmentID)
	}
	query, args = appendEnvironmentAccess(ctx, query, "environment_id", args)
	query += ` ORDER BY created_at ASC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.ServiceDeployment, 0, pageSize)
	for rows.Next() {
		item, err := scanDeployment(rows)
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

func (r *DeploymentRepository) Update(ctx context.Context, req *registryv1.UpdateDeploymentRequest) (*registryv1.ServiceDeployment, error) {
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
		UPDATE service_deployments
		SET health_enabled = ?, alerts_enabled = ?, alert_cooldown_minutes = ?, tags = ?, metadata = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, req.GetHealthEnabled(), req.GetAlertsEnabled(), req.GetAlertCooldownMinutes(), string(tagsJSON), string(metadataJSON), now, req.GetId())
	if err != nil {
		return nil, err
	}

	return r.Get(ctx, req.GetId())
}

func (r *DeploymentRepository) Delete(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(ctx, `
		UPDATE service_deployments
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, now, now, id)
	return err
}

func scanDeployment(row scanner) (*registryv1.ServiceDeployment, error) {
	var (
		id                   string
		serviceID            string
		environmentID        string
		healthEnabled        bool
		alertsEnabled        bool
		alertCooldownMinutes int32
		tagsRaw              string
		metadataRaw          string
		createdRaw           string
		updatedRaw           string
	)

	if err := row.Scan(&id, &serviceID, &environmentID, &healthEnabled, &alertsEnabled, &alertCooldownMinutes, &tagsRaw, &metadataRaw, &createdRaw, &updatedRaw); err != nil {
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

	return &registryv1.ServiceDeployment{
		Id:                   id,
		ServiceId:            serviceID,
		EnvironmentId:        environmentID,
		HealthEnabled:        healthEnabled,
		AlertsEnabled:        alertsEnabled,
		AlertCooldownMinutes: alertCooldownMinutes,
		Tags:                 tags,
		Metadata:             metadata,
		CreatedAt:            createdAt,
		UpdatedAt:            updatedAt,
	}, nil
}
