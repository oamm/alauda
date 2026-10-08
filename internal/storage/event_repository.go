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

type EventRepository struct {
	db *Database
}

type EventFilters struct {
	EnvironmentID string
	ServiceID     string
	DeploymentID  string
	InstanceID    string
	Type          string
}

func NewEventRepository(db *Database) *EventRepository {
	return &EventRepository{db: db}
}

func (r *EventRepository) Create(ctx context.Context, event *registryv1.Event) (*registryv1.Event, error) {
	if event == nil {
		return nil, errors.New("event is required")
	}
	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := createEventTx(ctx, tx, event); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, event.GetId())
}

func (r *EventRepository) Get(ctx context.Context, id string) (*registryv1.Event, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, type, timestamp, resource_type, resource_id, environment_id, service_id,
		       deployment_id, instance_id, actor, actor_id, message, changes, metadata
		FROM events
		WHERE id = ?
	`, id)
	return scanEvent(row)
}

func (r *EventRepository) List(ctx context.Context, filters EventFilters, pageSize int, pageToken string) ([]*registryv1.Event, string, error) {
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

	query, args := eventListQuery(filters)
	query, args = appendEnvironmentAccess(ctx, query, "environment_id", args)
	query += ` ORDER BY julianday(timestamp) DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.Event, 0, pageSize)
	for rows.Next() {
		item, err := scanEvent(rows)
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

func (r *EventRepository) ListSince(ctx context.Context, filters EventFilters, after time.Time, limit int) ([]*registryv1.Event, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query, args := eventListQuery(filters)
	query, args = appendEnvironmentAccess(ctx, query, "environment_id", args)
	query += ` AND julianday(timestamp) > julianday(?) ORDER BY julianday(timestamp) ASC, id ASC LIMIT ?`
	args = append(args, after.Format(time.RFC3339Nano), limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]*registryv1.Event, 0, limit)
	for rows.Next() {
		item, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func createEventTx(ctx context.Context, tx transaction, event *registryv1.Event) error {
	if event.Id == "" {
		event.Id = uuid.NewString()
	}
	timestamp := time.Now().UTC()
	if event.GetTimestamp() != nil {
		timestamp = event.GetTimestamp().AsTime().UTC()
	}
	if event.Actor == "" {
		event.Actor = "system"
	}
	metadataJSON, err := json.Marshal(event.GetMetadata())
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO events (
			id, type, timestamp, resource_type, resource_id, environment_id, service_id,
			deployment_id, instance_id, actor, actor_id, message, changes, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		event.GetId(),
		event.GetType(),
		timestamp.Format(time.RFC3339Nano),
		event.GetResourceType(),
		event.GetResourceId(),
		nullIfEmpty(event.GetEnvironmentId()),
		nullIfEmpty(event.GetServiceId()),
		nullIfEmpty(event.GetDeploymentId()),
		nullIfEmpty(event.GetInstanceId()),
		event.GetActor(),
		nullIfEmpty(event.GetActorId()),
		event.GetMessage(),
		nullIfEmpty(event.GetChanges()),
		string(metadataJSON),
	)
	return err
}

func eventListQuery(filters EventFilters) (string, []any) {
	query := `
		SELECT id, type, timestamp, resource_type, resource_id, environment_id, service_id,
		       deployment_id, instance_id, actor, actor_id, message, changes, metadata
		FROM events
		WHERE 1 = 1
	`
	args := make([]any, 0, 6)
	if filters.EnvironmentID != "" {
		query += ` AND environment_id = ?`
		args = append(args, filters.EnvironmentID)
	}
	if filters.ServiceID != "" {
		query += ` AND service_id = ?`
		args = append(args, filters.ServiceID)
	}
	if filters.DeploymentID != "" {
		query += ` AND deployment_id = ?`
		args = append(args, filters.DeploymentID)
	}
	if filters.InstanceID != "" {
		query += ` AND instance_id = ?`
		args = append(args, filters.InstanceID)
	}
	if filters.Type != "" {
		query += ` AND type = ?`
		args = append(args, filters.Type)
	}
	return query, args
}

func scanEvent(row scanner) (*registryv1.Event, error) {
	var (
		id            string
		eventType     string
		timestampRaw  string
		resourceType  string
		resourceID    string
		environmentID sql.NullString
		serviceID     sql.NullString
		deploymentID  sql.NullString
		instanceID    sql.NullString
		actor         string
		actorID       sql.NullString
		message       string
		changes       sql.NullString
		metadataRaw   string
	)
	if err := row.Scan(&id, &eventType, &timestampRaw, &resourceType, &resourceID, &environmentID, &serviceID, &deploymentID, &instanceID, &actor, &actorID, &message, &changes, &metadataRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	timestamp := timestamppb.Now()
	if t, err := time.Parse(time.RFC3339Nano, timestampRaw); err == nil {
		timestamp = timestamppb.New(t)
	}
	metadata := map[string]string{}
	if metadataRaw != "" {
		_ = json.Unmarshal([]byte(metadataRaw), &metadata)
	}

	event := &registryv1.Event{
		Id:           id,
		Type:         eventType,
		Timestamp:    timestamp,
		ResourceType: resourceType,
		ResourceId:   resourceID,
		Actor:        actor,
		Message:      message,
		Metadata:     metadata,
	}
	if environmentID.Valid {
		event.EnvironmentId = environmentID.String
	}
	if serviceID.Valid {
		event.ServiceId = serviceID.String
	}
	if deploymentID.Valid {
		event.DeploymentId = deploymentID.String
	}
	if instanceID.Valid {
		event.InstanceId = instanceID.String
	}
	if actorID.Valid {
		event.ActorId = actorID.String
	}
	if changes.Valid {
		event.Changes = changes.String
	}
	return event, nil
}
