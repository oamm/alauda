package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type HealthRepository struct {
	db              *Database
	alertDispatcher AlertDispatcher
}

type HealthCheckTarget struct {
	Check   *registryv1.HealthCheck
	Address string
	Port    int32
	Path    string
}

func NewHealthRepository(db *Database) *HealthRepository {
	return &HealthRepository{db: db}
}

func NewHealthRepositoryWithAlerts(db *Database, dispatcher AlertDispatcher) *HealthRepository {
	return &HealthRepository{db: db, alertDispatcher: dispatcher}
}

func (r *HealthRepository) CreateHealthCheck(ctx context.Context, req *registryv1.CreateHealthCheckRequest) (*registryv1.HealthCheck, error) {
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
		INSERT INTO health_checks (
			id, instance_id, endpoint_id, name, type, enabled, interval_seconds, timeout_seconds,
			failures_before_unhealthy, successes_before_healthy, description, tags, metadata, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		id,
		req.GetInstanceId(),
		nullIfEmpty(req.GetEndpointId()),
		req.GetName(),
		req.GetType().String(),
		req.GetEnabled(),
		req.GetIntervalSeconds(),
		req.GetTimeoutSeconds(),
		req.GetFailuresBeforeUnhealthy(),
		req.GetSuccessesBeforeHealthy(),
		req.GetDescription(),
		string(tagsJSON),
		string(metadataJSON),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, err
	}

	return r.GetHealthCheck(ctx, id)
}

func (r *HealthRepository) GetHealthCheck(ctx context.Context, id string) (*registryv1.HealthCheck, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, instance_id, endpoint_id, name, type, enabled, interval_seconds, timeout_seconds,
		       failures_before_unhealthy, successes_before_healthy, description, tags, metadata
		FROM health_checks
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanHealthCheck(row)
}

func (r *HealthRepository) ListHealthChecks(ctx context.Context, instanceID string, includeDisabled bool, pageSize int, pageToken string) ([]*registryv1.HealthCheck, string, error) {
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
		SELECT id, instance_id, endpoint_id, name, type, enabled, interval_seconds, timeout_seconds,
		       failures_before_unhealthy, successes_before_healthy, description, tags, metadata
		FROM health_checks
		WHERE deleted_at IS NULL
	`
	args := make([]any, 0, 4)
	if instanceID != "" {
		query += ` AND instance_id = ?`
		args = append(args, instanceID)
	}
	if !includeDisabled {
		query += ` AND enabled = 1`
	}
	query += ` ORDER BY name ASC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.HealthCheck, 0, pageSize)
	for rows.Next() {
		item, err := scanHealthCheck(rows)
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

func (r *HealthRepository) UpdateHealthCheck(ctx context.Context, req *registryv1.UpdateHealthCheckRequest) (*registryv1.HealthCheck, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(ctx, `
		UPDATE health_checks
		SET enabled = ?, interval_seconds = ?, timeout_seconds = ?, failures_before_unhealthy = ?,
		    successes_before_healthy = ?, description = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`,
		req.GetEnabled(),
		req.GetIntervalSeconds(),
		req.GetTimeoutSeconds(),
		req.GetFailuresBeforeUnhealthy(),
		req.GetSuccessesBeforeHealthy(),
		req.GetDescription(),
		now,
		req.GetId(),
	)
	if err != nil {
		return nil, err
	}
	return r.GetHealthCheck(ctx, req.GetId())
}

func (r *HealthRepository) DeleteHealthCheck(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(ctx, `
		UPDATE health_checks
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, now, now, id)
	return err
}

func (r *HealthRepository) PruneHealthResults(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.db.Exec(ctx, `
		DELETE FROM health_results
		WHERE timestamp < ?
	`, olderThan.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *HealthRepository) GetInstanceHealthState(ctx context.Context, instanceID string) (*registryv1.HealthStateView, error) {
	row := r.db.QueryRow(ctx, `
		SELECT instance_id, current_state, consecutive_successes, consecutive_failures, last_transition_time, last_check_time
		FROM health_states
		WHERE instance_id = ?
	`, instanceID)

	var (
		id              string
		currentStateRaw string
		successes       int32
		failures        int32
		transitionRaw   string
		checkRaw        sql.NullString
	)
	if err := row.Scan(&id, &currentStateRaw, &successes, &failures, &transitionRaw, &checkRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	transitionAt := timestamppb.Now()
	if t, err := time.Parse(time.RFC3339Nano, transitionRaw); err == nil {
		transitionAt = timestamppb.New(t)
	}
	var checkAt *timestamppb.Timestamp
	if checkRaw.Valid {
		if t, err := time.Parse(time.RFC3339Nano, checkRaw.String); err == nil {
			checkAt = timestamppb.New(t)
		}
	}

	return &registryv1.HealthStateView{
		InstanceId:           id,
		CurrentState:         parseHealthState(currentStateRaw),
		ConsecutiveSuccesses: successes,
		ConsecutiveFailures:  failures,
		LastTransitionTime:   transitionAt,
		LastCheckTime:        checkAt,
	}, nil
}

func (r *HealthRepository) ListHealthResults(ctx context.Context, healthCheckID, instanceID string, pageSize int, pageToken string) ([]*registryv1.HealthResult, string, error) {
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
		SELECT id, health_check_id, instance_id, timestamp, success, latency_ms, status_code, error_type, error_message, metadata
		FROM health_results
		WHERE 1 = 1
	`
	args := make([]any, 0, 4)
	if healthCheckID != "" {
		query += ` AND health_check_id = ?`
		args = append(args, healthCheckID)
	}
	if instanceID != "" {
		query += ` AND instance_id = ?`
		args = append(args, instanceID)
	}
	query += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.HealthResult, 0, pageSize)
	for rows.Next() {
		var (
			id           string
			hcID         string
			instID       string
			timeRaw      string
			success      bool
			latencyMS    sql.NullInt32
			statusCode   sql.NullInt32
			errorType    sql.NullString
			errorMessage sql.NullString
			metadataRaw  string
		)
		if err := rows.Scan(&id, &hcID, &instID, &timeRaw, &success, &latencyMS, &statusCode, &errorType, &errorMessage, &metadataRaw); err != nil {
			return nil, "", err
		}

		metadata := map[string]string{}
		if metadataRaw != "" {
			_ = json.Unmarshal([]byte(metadataRaw), &metadata)
		}
		timestamp := timestamppb.Now()
		if t, err := time.Parse(time.RFC3339Nano, timeRaw); err == nil {
			timestamp = timestamppb.New(t)
		}

		item := &registryv1.HealthResult{
			Id:            id,
			HealthCheckId: hcID,
			InstanceId:    instID,
			Timestamp:     timestamp,
			Success:       success,
			Metadata:      metadata,
		}
		if latencyMS.Valid {
			item.LatencyMs = latencyMS.Int32
		}
		if statusCode.Valid {
			item.StatusCode = statusCode.Int32
		}
		if errorType.Valid {
			item.ErrorType = errorType.String
		}
		if errorMessage.Valid {
			item.ErrorMessage = errorMessage.String
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

func (r *HealthRepository) ListDueHealthCheckTargets(ctx context.Context, limit int) ([]HealthCheckTarget, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	rows, err := r.db.Query(ctx, `
		SELECT hc.id, hc.instance_id, hc.endpoint_id, hc.name, hc.type, hc.enabled,
		       hc.interval_seconds, hc.timeout_seconds, hc.failures_before_unhealthy,
		       hc.successes_before_healthy, hc.description, hc.tags, hc.metadata,
		       si.address, COALESCE(e.port, si.port), COALESCE(e.path, '')
		FROM health_checks hc
		JOIN service_instances si ON si.id = hc.instance_id AND si.deleted_at IS NULL AND si.enabled = 1
		LEFT JOIN endpoints e ON e.id = hc.endpoint_id AND e.deleted_at IS NULL AND e.enabled = 1
		LEFT JOIN (
			SELECT health_check_id, MAX(timestamp) AS last_timestamp
			FROM health_results
			GROUP BY health_check_id
		) latest ON latest.health_check_id = hc.id
		WHERE hc.deleted_at IS NULL
		  AND hc.enabled = 1
		  AND (
			latest.last_timestamp IS NULL
			OR julianday(latest.last_timestamp) <= julianday('now') - (CAST(hc.interval_seconds AS REAL) / 86400.0)
		  )
		ORDER BY COALESCE(latest.last_timestamp, hc.created_at) ASC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := make([]HealthCheckTarget, 0, limit)
	for rows.Next() {
		target, err := scanHealthCheckTarget(rows)
		if err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

func (r *HealthRepository) GetHealthCheckTarget(ctx context.Context, id string) (HealthCheckTarget, error) {
	row := r.db.QueryRow(ctx, `
		SELECT hc.id, hc.instance_id, hc.endpoint_id, hc.name, hc.type, hc.enabled,
		       hc.interval_seconds, hc.timeout_seconds, hc.failures_before_unhealthy,
		       hc.successes_before_healthy, hc.description, hc.tags, hc.metadata,
		       si.address, COALESCE(e.port, si.port), COALESCE(e.path, '')
		FROM health_checks hc
		JOIN service_instances si ON si.id = hc.instance_id AND si.deleted_at IS NULL AND si.enabled = 1
		LEFT JOIN endpoints e ON e.id = hc.endpoint_id AND e.deleted_at IS NULL AND e.enabled = 1
		WHERE hc.id = ? AND hc.deleted_at IS NULL AND hc.enabled = 1
	`, id)

	target, err := scanHealthCheckTarget(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return HealthCheckTarget{}, sql.ErrNoRows
		}
		return HealthCheckTarget{}, err
	}
	return target, nil
}

func (r *HealthRepository) RecordHealthResult(ctx context.Context, check *registryv1.HealthCheck, result *registryv1.HealthResult) (*registryv1.HealthStateView, error) {
	if check == nil {
		return nil, errors.New("health check is required")
	}
	if result == nil {
		return nil, errors.New("health result is required")
	}

	now := time.Now().UTC()
	resultID := result.GetId()
	if resultID == "" {
		resultID = uuid.NewString()
		result.Id = resultID
	}
	timestamp := now
	if result.GetTimestamp() != nil {
		timestamp = result.GetTimestamp().AsTime().UTC()
	}
	result.HealthCheckId = check.GetId()
	result.InstanceId = check.GetInstanceId()

	metadataJSON, err := json.Marshal(result.GetMetadata())
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO health_results (
			id, health_check_id, instance_id, timestamp, success, latency_ms, status_code,
			error_type, error_message, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		resultID,
		result.GetHealthCheckId(),
		result.GetInstanceId(),
		timestamp.Format(time.RFC3339Nano),
		result.GetSuccess(),
		nullInt32(result.GetLatencyMs()),
		nullInt32(result.GetStatusCode()),
		nullIfEmpty(result.GetErrorType()),
		nullIfEmpty(result.GetErrorMessage()),
		string(metadataJSON),
	)
	if err != nil {
		return nil, err
	}

	current, err := getHealthStateForUpdate(ctx, tx, check.GetInstanceId())
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	next := transitionState(current, check, result, timestamp)
	previousState := registryv1.HealthState_HEALTH_STATE_UNKNOWN
	if current != nil {
		previousState = current.GetCurrentState()
	}
	if current == nil {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO health_states (
				id, instance_id, current_state, consecutive_successes, consecutive_failures,
				last_transition_time, last_check_time, metadata, updated_at
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, '{}', ?)
		`,
			uuid.NewString(),
			next.GetInstanceId(),
			next.GetCurrentState().String(),
			next.GetConsecutiveSuccesses(),
			next.GetConsecutiveFailures(),
			next.GetLastTransitionTime().AsTime().UTC().Format(time.RFC3339Nano),
			timestamp.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
		)
	} else {
		_, err = tx.ExecContext(ctx, `
			UPDATE health_states
			SET current_state = ?, consecutive_successes = ?, consecutive_failures = ?,
			    last_transition_time = ?, last_check_time = ?, updated_at = ?
			WHERE instance_id = ?
		`,
			next.GetCurrentState().String(),
			next.GetConsecutiveSuccesses(),
			next.GetConsecutiveFailures(),
			next.GetLastTransitionTime().AsTime().UTC().Format(time.RFC3339Nano),
			timestamp.Format(time.RFC3339Nano),
			now.Format(time.RFC3339Nano),
			next.GetInstanceId(),
		)
	}
	if err != nil {
		return nil, err
	}

	type alertNotification struct {
		incidentID       string
		notificationType string
		notificationTime time.Time
	}
	notifications := make([]alertNotification, 0)
	if next.GetCurrentState() != previousState {
		switch next.GetCurrentState() {
		case registryv1.HealthState_HEALTH_STATE_UNHEALTHY:
			incidentID, err := openIncidentForInstanceTx(ctx, tx, check, result, timestamp)
			if err != nil {
				return nil, err
			}
			if incidentID != "" {
				notifications = append(notifications, alertNotification{incidentID: incidentID, notificationType: "unhealthy", notificationTime: timestamp})
			}
		case registryv1.HealthState_HEALTH_STATE_HEALTHY:
			if previousState == registryv1.HealthState_HEALTH_STATE_UNHEALTHY {
				incidentIDs, err := resolveOpenIncidentsForInstanceTx(ctx, tx, next.GetInstanceId(), "health check recovered", timestamp)
				if err != nil {
					return nil, err
				}
				for _, incidentID := range incidentIDs {
					notifications = append(notifications, alertNotification{incidentID: incidentID, notificationType: "recovered", notificationTime: timestamp})
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	for _, notification := range notifications {
		r.dispatchAlert(ctx, notification.incidentID, notification.notificationType, notification.notificationTime)
	}
	return next, nil
}

func (r *HealthRepository) dispatchAlert(ctx context.Context, incidentID, notificationType string, at time.Time) {
	if r.alertDispatcher == nil || incidentID == "" {
		return
	}
	incident, err := NewIncidentRepository(r.db).Get(ctx, incidentID)
	if err != nil {
		slog.Warn("failed to load incident for alert notification", slog.String("incident_id", incidentID), slog.Any("error", err))
		return
	}
	if err := r.alertDispatcher.ProcessIncident(ctx, incident, notificationType, at); err != nil {
		slog.Warn("failed to process alert notification", slog.String("incident_id", incidentID), slog.String("notification_type", notificationType), slog.Any("error", err))
	}
}

func scanHealthCheck(row scanner) (*registryv1.HealthCheck, error) {
	var (
		id                      string
		instanceID              string
		endpointID              sql.NullString
		name                    string
		typeRaw                 string
		enabled                 bool
		intervalSeconds         int32
		timeoutSeconds          int32
		failuresBeforeUnhealthy int32
		successesBeforeHealthy  int32
		description             string
		tagsRaw                 string
		metadataRaw             string
	)

	if err := row.Scan(
		&id,
		&instanceID,
		&endpointID,
		&name,
		&typeRaw,
		&enabled,
		&intervalSeconds,
		&timeoutSeconds,
		&failuresBeforeUnhealthy,
		&successesBeforeHealthy,
		&description,
		&tagsRaw,
		&metadataRaw,
	); err != nil {
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

	item := &registryv1.HealthCheck{
		Id:                      id,
		InstanceId:              instanceID,
		Name:                    name,
		Type:                    parseHealthCheckType(typeRaw),
		Enabled:                 enabled,
		IntervalSeconds:         intervalSeconds,
		TimeoutSeconds:          timeoutSeconds,
		FailuresBeforeUnhealthy: failuresBeforeUnhealthy,
		SuccessesBeforeHealthy:  successesBeforeHealthy,
		Description:             description,
		Tags:                    tags,
		Metadata:                metadata,
	}
	if endpointID.Valid {
		item.EndpointId = endpointID.String
	}
	return item, nil
}

func scanHealthCheckTarget(row scanner) (HealthCheckTarget, error) {
	var (
		id                      string
		instanceID              string
		endpointID              sql.NullString
		name                    string
		typeRaw                 string
		enabled                 bool
		intervalSeconds         int32
		timeoutSeconds          int32
		failuresBeforeUnhealthy int32
		successesBeforeHealthy  int32
		description             string
		tagsRaw                 string
		metadataRaw             string
		address                 string
		port                    int32
		path                    string
	)

	if err := row.Scan(
		&id,
		&instanceID,
		&endpointID,
		&name,
		&typeRaw,
		&enabled,
		&intervalSeconds,
		&timeoutSeconds,
		&failuresBeforeUnhealthy,
		&successesBeforeHealthy,
		&description,
		&tagsRaw,
		&metadataRaw,
		&address,
		&port,
		&path,
	); err != nil {
		return HealthCheckTarget{}, err
	}

	check := &registryv1.HealthCheck{
		Id:                      id,
		InstanceId:              instanceID,
		Name:                    name,
		Type:                    parseHealthCheckType(typeRaw),
		Enabled:                 enabled,
		IntervalSeconds:         intervalSeconds,
		TimeoutSeconds:          timeoutSeconds,
		FailuresBeforeUnhealthy: failuresBeforeUnhealthy,
		SuccessesBeforeHealthy:  successesBeforeHealthy,
		Description:             description,
		Tags:                    map[string]string{},
		Metadata:                map[string]string{},
	}
	if endpointID.Valid {
		check.EndpointId = endpointID.String
	}
	if tagsRaw != "" {
		_ = json.Unmarshal([]byte(tagsRaw), &check.Tags)
	}
	if metadataRaw != "" {
		_ = json.Unmarshal([]byte(metadataRaw), &check.Metadata)
	}

	return HealthCheckTarget{
		Check:   check,
		Address: address,
		Port:    port,
		Path:    path,
	}, nil
}

func parseHealthCheckType(v string) registryv1.HealthCheckType {
	if enumValue, ok := registryv1.HealthCheckType_value[strings.ToUpper(v)]; ok {
		return registryv1.HealthCheckType(enumValue)
	}
	return registryv1.HealthCheckType_HEALTH_CHECK_TYPE_UNSPECIFIED
}

func parseHealthState(v string) registryv1.HealthState {
	if enumValue, ok := registryv1.HealthState_value[strings.ToUpper(v)]; ok {
		return registryv1.HealthState(enumValue)
	}
	return registryv1.HealthState_HEALTH_STATE_UNSPECIFIED
}

func nullIfEmpty(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func nullInt32(v int32) any {
	if v == 0 {
		return nil
	}
	return v
}

func getHealthStateForUpdate(ctx context.Context, tx *sql.Tx, instanceID string) (*registryv1.HealthStateView, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT instance_id, current_state, consecutive_successes, consecutive_failures, last_transition_time, last_check_time
		FROM health_states
		WHERE instance_id = ?
	`, instanceID)

	var (
		id              string
		currentStateRaw string
		successes       int32
		failures        int32
		transitionRaw   string
		checkRaw        sql.NullString
	)
	if err := row.Scan(&id, &currentStateRaw, &successes, &failures, &transitionRaw, &checkRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}

	transitionAt := timestamppb.Now()
	if t, err := time.Parse(time.RFC3339Nano, transitionRaw); err == nil {
		transitionAt = timestamppb.New(t)
	}
	var checkAt *timestamppb.Timestamp
	if checkRaw.Valid {
		if t, err := time.Parse(time.RFC3339Nano, checkRaw.String); err == nil {
			checkAt = timestamppb.New(t)
		}
	}

	return &registryv1.HealthStateView{
		InstanceId:           id,
		CurrentState:         parseHealthState(currentStateRaw),
		ConsecutiveSuccesses: successes,
		ConsecutiveFailures:  failures,
		LastTransitionTime:   transitionAt,
		LastCheckTime:        checkAt,
	}, nil
}

func transitionState(current *registryv1.HealthStateView, check *registryv1.HealthCheck, result *registryv1.HealthResult, timestamp time.Time) *registryv1.HealthStateView {
	previousState := registryv1.HealthState_HEALTH_STATE_UNKNOWN
	successes := int32(0)
	failures := int32(0)
	transitionTime := timestamp
	if current != nil {
		previousState = current.GetCurrentState()
		successes = current.GetConsecutiveSuccesses()
		failures = current.GetConsecutiveFailures()
		if current.GetLastTransitionTime() != nil {
			transitionTime = current.GetLastTransitionTime().AsTime().UTC()
		}
	}

	nextState := previousState
	if result.GetSuccess() {
		successes++
		failures = 0
		if successes >= positiveOrDefault(check.GetSuccessesBeforeHealthy(), 1) {
			nextState = registryv1.HealthState_HEALTH_STATE_HEALTHY
		}
	} else {
		failures++
		successes = 0
		if failures >= positiveOrDefault(check.GetFailuresBeforeUnhealthy(), 1) {
			nextState = registryv1.HealthState_HEALTH_STATE_UNHEALTHY
		}
	}

	if nextState != previousState {
		transitionTime = timestamp
	}

	return &registryv1.HealthStateView{
		InstanceId:           check.GetInstanceId(),
		CurrentState:         nextState,
		ConsecutiveSuccesses: successes,
		ConsecutiveFailures:  failures,
		LastTransitionTime:   timestamppb.New(transitionTime),
		LastCheckTime:        timestamppb.New(timestamp),
	}
}

func positiveOrDefault(value, fallback int32) int32 {
	if value > 0 {
		return value
	}
	return fallback
}
