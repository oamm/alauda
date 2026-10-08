package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type IncidentRepository struct {
	db              *Database
	alertDispatcher AlertDispatcher
}

func NewIncidentRepository(db *Database) *IncidentRepository {
	return &IncidentRepository{db: db}
}

func NewIncidentRepositoryWithAlerts(db *Database, dispatcher AlertDispatcher) *IncidentRepository {
	return &IncidentRepository{db: db, alertDispatcher: dispatcher}
}

func (r *IncidentRepository) Get(ctx context.Context, id string) (*registryv1.Incident, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, instance_id, deployment_id, environment_id, service_id, state, opened_at,
		       resolved_at, duration_seconds, reason, impact_summary, tags, metadata,
		       resolution_method, resolution_note, resolution_evidence_health_result_id, resolved_by
		FROM incidents
		WHERE id = ?
	`, id)
	return scanIncident(row)
}

func (r *IncidentRepository) List(ctx context.Context, environmentID, serviceID, deploymentID, instanceID string, state registryv1.IncidentState, pageSize int, pageToken string) ([]*registryv1.Incident, string, error) {
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
		SELECT id, instance_id, deployment_id, environment_id, service_id, state, opened_at,
		       resolved_at, duration_seconds, reason, impact_summary, tags, metadata,
		       resolution_method, resolution_note, resolution_evidence_health_result_id, resolved_by
		FROM incidents
		WHERE 1 = 1
	`
	args := make([]any, 0, 8)
	if environmentID != "" {
		query += ` AND environment_id = ?`
		args = append(args, environmentID)
	}
	if serviceID != "" {
		query += ` AND service_id = ?`
		args = append(args, serviceID)
	}
	if deploymentID != "" {
		query += ` AND deployment_id = ?`
		args = append(args, deploymentID)
	}
	if instanceID != "" {
		query += ` AND instance_id = ?`
		args = append(args, instanceID)
	}
	if state != registryv1.IncidentState_INCIDENT_STATE_UNSPECIFIED {
		query += ` AND state = ?`
		args = append(args, state.String())
	}
	query += ` ORDER BY opened_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.Incident, 0, pageSize)
	for rows.Next() {
		item, err := scanIncident(rows)
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

func (r *IncidentRepository) ResolveManually(ctx context.Context, id, note string) (*registryv1.Incident, error) {
	now := time.Now().UTC()
	if note == "" {
		note = "resolved manually"
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = resolveIncidentTx(ctx, tx, id, note, "ManualOverride", "", "operator", "Incident resolved manually", now)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	incident, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	r.dispatchAlert(ctx, incident, "recovered", now)
	return incident, nil
}

func resolveIncidentTx(ctx context.Context, tx *sql.Tx, id, note, method, evidenceID, actor, message string, resolvedAt time.Time) (bool, error) {
	var instanceID, deploymentID, environmentID, serviceID string
	err := tx.QueryRowContext(ctx, `
		SELECT instance_id, deployment_id, environment_id, service_id
		FROM incidents
		WHERE id = ? AND state = ?
	`, id, registryv1.IncidentState_INCIDENT_STATE_OPEN.String()).Scan(&instanceID, &deploymentID, &environmentID, &serviceID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	now := resolvedAt.UTC().Format(time.RFC3339Nano)
	_, err = tx.ExecContext(ctx, `
		UPDATE incidents
		SET state = ?, resolved_at = ?, duration_seconds = CAST((julianday(?) - julianday(opened_at)) * 86400 AS INTEGER),
		    resolution_method = ?, resolution_note = ?, resolution_evidence_health_result_id = ?,
		    resolved_by = ?,
		    metadata = json_set(metadata, '$.resolution_reason', ?)
		WHERE id = ? AND state = ?
	`, registryv1.IncidentState_INCIDENT_STATE_RESOLVED.String(), now, now, method, note, evidenceID, actor, note, id, registryv1.IncidentState_INCIDENT_STATE_OPEN.String())
	if err != nil {
		return false, err
	}
	if err := createEventTx(ctx, tx, &registryv1.Event{
		Type: "incident.resolved", Timestamp: timestamppb.New(resolvedAt), ResourceType: "incident", ResourceId: id,
		EnvironmentId: environmentID, ServiceId: serviceID, DeploymentId: deploymentID, InstanceId: instanceID,
		Actor: actor, Message: message,
		Metadata: map[string]string{"reason": note, "resolution_method": method, "health_result_id": evidenceID},
	}); err != nil {
		return false, err
	}
	return true, nil
}

func openIncidentForInstanceTx(ctx context.Context, tx *sql.Tx, check *registryv1.HealthCheck, result *registryv1.HealthResult, openedAt time.Time) (string, error) {
	var existingID string
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM incidents
		WHERE instance_id = ? AND state = ?
		LIMIT 1
	`, check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN.String()).Scan(&existingID)
	if err == nil {
		return "", nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	var deploymentID, environmentID, serviceID string
	err = tx.QueryRowContext(ctx, `
		SELECT si.deployment_id, d.environment_id, d.service_id
		FROM service_instances si
		JOIN service_deployments d ON d.id = si.deployment_id
		WHERE si.id = ? AND si.deleted_at IS NULL AND d.deleted_at IS NULL
	`, check.GetInstanceId()).Scan(&deploymentID, &environmentID, &serviceID)
	if err != nil {
		return "", err
	}

	reason := result.GetErrorMessage()
	if reason == "" {
		reason = result.GetErrorType()
	}
	if reason == "" {
		reason = "health check failed"
	}
	tagsJSON, err := json.Marshal(map[string]string{})
	if err != nil {
		return "", err
	}
	metadataJSON, err := json.Marshal(map[string]string{
		"health_check_id": check.GetId(),
		"health_state":    registryv1.HealthState_HEALTH_STATE_UNHEALTHY.String(),
	})
	if err != nil {
		return "", err
	}

	incidentID := uuid.NewString()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO incidents (
			id, instance_id, deployment_id, environment_id, service_id, state, opened_at,
			reason, impact_summary, tags, metadata, created_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		incidentID,
		check.GetInstanceId(),
		deploymentID,
		environmentID,
		serviceID,
		registryv1.IncidentState_INCIDENT_STATE_OPEN.String(),
		openedAt.Format(time.RFC3339Nano),
		reason,
		"Instance became unhealthy",
		string(tagsJSON),
		string(metadataJSON),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return "", err
	}
	if err := createEventTx(ctx, tx, &registryv1.Event{
		Type:          "incident.opened",
		Timestamp:     timestamppb.New(openedAt),
		ResourceType:  "incident",
		ResourceId:    incidentID,
		EnvironmentId: environmentID,
		ServiceId:     serviceID,
		DeploymentId:  deploymentID,
		InstanceId:    check.GetInstanceId(),
		Actor:         "health-monitor",
		Message:       "Incident opened after instance became unhealthy",
		Metadata: map[string]string{
			"health_check_id": check.GetId(),
			"reason":          reason,
		},
	}); err != nil {
		return "", err
	}
	return incidentID, nil
}

func resolveOpenIncidentsForInstanceTx(ctx context.Context, tx *sql.Tx, instanceID, reason, evidenceID string, resolvedAt time.Time) ([]string, error) {
	if reason == "" {
		reason = "health check recovered"
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id, deployment_id, environment_id, service_id
		FROM incidents
		WHERE instance_id = ? AND state = ?
	`, instanceID, registryv1.IncidentState_INCIDENT_STATE_OPEN.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type incidentScope struct {
		id            string
		deploymentID  string
		environmentID string
		serviceID     string
	}
	scopes := make([]incidentScope, 0)
	for rows.Next() {
		var scope incidentScope
		if err := rows.Scan(&scope.id, &scope.deploymentID, &scope.environmentID, &scope.serviceID); err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	incidentIDs := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		resolved, err := resolveIncidentTx(ctx, tx, scope.id, reason, "AutoRecovered", evidenceID, "health-monitor", "Incident resolved after instance recovered", resolvedAt)
		if err != nil {
			return nil, err
		}
		if resolved {
			incidentIDs = append(incidentIDs, scope.id)
		}
	}
	return incidentIDs, nil
}

func (r *IncidentRepository) dispatchAlert(ctx context.Context, incident *registryv1.Incident, notificationType string, at time.Time) {
	if r.alertDispatcher == nil || incident == nil {
		return
	}
	if err := r.alertDispatcher.ProcessIncident(ctx, incident, notificationType, at); err != nil {
		slog.Warn("failed to process alert notification", slog.String("incident_id", incident.GetId()), slog.String("notification_type", notificationType), slog.Any("error", err))
	}
}

func scanIncident(row scanner) (*registryv1.Incident, error) {
	var (
		id                   string
		instanceID           string
		deploymentID         string
		environmentID        string
		serviceID            string
		stateRaw             string
		openedRaw            string
		resolvedRaw          sql.NullString
		durationSeconds      sql.NullInt64
		reason               string
		impactSummary        sql.NullString
		tagsRaw              string
		metadataRaw          string
		resolutionMethod     string
		resolutionNote       string
		resolutionEvidenceID string
		resolvedBy           string
	)
	if err := row.Scan(&id, &instanceID, &deploymentID, &environmentID, &serviceID, &stateRaw, &openedRaw, &resolvedRaw, &durationSeconds, &reason, &impactSummary, &tagsRaw, &metadataRaw, &resolutionMethod, &resolutionNote, &resolutionEvidenceID, &resolvedBy); err != nil {
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

	openedAt := timestamppb.Now()
	if t, err := time.Parse(time.RFC3339Nano, openedRaw); err == nil {
		openedAt = timestamppb.New(t)
	}
	var resolvedAt *timestamppb.Timestamp
	if resolvedRaw.Valid {
		if t, err := time.Parse(time.RFC3339Nano, resolvedRaw.String); err == nil {
			resolvedAt = timestamppb.New(t)
		}
	}

	item := &registryv1.Incident{
		Id:                               id,
		InstanceId:                       instanceID,
		DeploymentId:                     deploymentID,
		EnvironmentId:                    environmentID,
		ServiceId:                        serviceID,
		State:                            parseIncidentState(stateRaw),
		OpenedAt:                         openedAt,
		ResolvedAt:                       resolvedAt,
		Reason:                           reason,
		Tags:                             tags,
		Metadata:                         metadata,
		ResolutionMethod:                 resolutionMethod,
		ResolutionNote:                   resolutionNote,
		ResolutionEvidenceHealthResultId: resolutionEvidenceID,
		ResolvedBy:                       resolvedBy,
	}
	if durationSeconds.Valid {
		item.DurationSeconds = durationSeconds.Int64
	}
	if impactSummary.Valid {
		item.ImpactSummary = impactSummary.String
	}
	return item, nil
}

func parseIncidentState(v string) registryv1.IncidentState {
	if enumValue, ok := registryv1.IncidentState_value[v]; ok {
		return registryv1.IncidentState(enumValue)
	}
	return registryv1.IncidentState_INCIDENT_STATE_UNSPECIFIED
}
