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

type AlertRepository struct {
	db *Database
}

type AlertAttempt struct {
	IncidentID       string
	PolicyID         string
	ChannelID        string
	NotificationType string
	AttemptedAt      time.Time
	Success          bool
	StatusCode       int
	ErrorMessage     string
	RetryCount       int
	NextRetryAt      *time.Time
	Metadata         map[string]string
}

func NewAlertRepository(db *Database) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) CreateNotificationChannel(ctx context.Context, req *registryv1.CreateNotificationChannelRequest) (*registryv1.NotificationChannel, error) {
	now := time.Now().UTC()
	id := uuid.NewString()
	configurationJSON, err := json.Marshal(req.GetConfiguration())
	if err != nil {
		return nil, err
	}
	retryPolicyJSON, err := json.Marshal(req.GetRetryPolicy())
	if err != nil {
		return nil, err
	}
	tagsJSON, err := json.Marshal(req.GetTags())
	if err != nil {
		return nil, err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO notification_channels (
			id, type, name, enabled, description, configuration, retry_policy, tags, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, req.GetType(), req.GetName(), req.GetEnabled(), req.GetDescription(), string(configurationJSON), string(retryPolicyJSON), string(tagsJSON), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	return r.GetNotificationChannel(ctx, id)
}

func (r *AlertRepository) GetNotificationChannel(ctx context.Context, id string) (*registryv1.NotificationChannel, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, type, name, enabled, description, configuration, retry_policy, tags, created_at, updated_at
		FROM notification_channels
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	return scanNotificationChannel(row)
}

func (r *AlertRepository) ListNotificationChannels(ctx context.Context, pageSize int, pageToken string) ([]*registryv1.NotificationChannel, string, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := offsetFromToken(pageToken)
	rows, err := r.db.Query(ctx, `
		SELECT id, type, name, enabled, description, configuration, retry_policy, tags, created_at, updated_at
		FROM notification_channels
		WHERE deleted_at IS NULL
		ORDER BY name ASC, id ASC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.NotificationChannel, 0, pageSize)
	for rows.Next() {
		item, err := scanNotificationChannel(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	return items, nextPageToken(len(items), pageSize, offset), nil
}

func (r *AlertRepository) UpdateNotificationChannel(ctx context.Context, req *registryv1.UpdateNotificationChannelRequest) (*registryv1.NotificationChannel, error) {
	configurationJSON, err := json.Marshal(req.GetConfiguration())
	if err != nil {
		return nil, err
	}
	retryPolicyJSON, err := json.Marshal(req.GetRetryPolicy())
	if err != nil {
		return nil, err
	}
	tagsJSON, err := json.Marshal(req.GetTags())
	if err != nil {
		return nil, err
	}
	_, err = r.db.Exec(ctx, `
		UPDATE notification_channels
		SET name = ?, enabled = ?, description = ?, configuration = ?, retry_policy = ?, tags = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, req.GetName(), req.GetEnabled(), req.GetDescription(), string(configurationJSON), string(retryPolicyJSON), string(tagsJSON), time.Now().UTC().Format(time.RFC3339Nano), req.GetId())
	if err != nil {
		return nil, err
	}
	return r.GetNotificationChannel(ctx, req.GetId())
}

func (r *AlertRepository) DeleteNotificationChannel(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(ctx, `
		UPDATE notification_channels
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, now, now, id)
	return err
}

func (r *AlertRepository) CreateAlertPolicy(ctx context.Context, req *registryv1.CreateAlertPolicyRequest) (*registryv1.AlertPolicy, error) {
	now := time.Now().UTC()
	id := uuid.NewString()
	notifyOn := req.GetNotifyOn()
	if len(notifyOn) == 0 {
		notifyOn = []string{"unhealthy", "recovered"}
	}
	cooldownMinutes := req.GetCooldownMinutes()
	if cooldownMinutes <= 0 {
		cooldownMinutes = 10
	}
	notifyOnJSON, err := json.Marshal(notifyOn)
	if err != nil {
		return nil, err
	}
	filtersJSON, err := json.Marshal(req.GetFilters())
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO alert_policies (
			id, deployment_id, environment_id, enabled, notify_on, cooldown_minutes,
			send_recovery_notification, filters, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, nullIfEmpty(req.GetDeploymentId()), nullIfEmpty(req.GetEnvironmentId()), req.GetEnabled(), string(notifyOnJSON), cooldownMinutes, req.GetSendRecoveryNotification(), string(filtersJSON), now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	if err := replacePolicyChannelsTx(ctx, tx, id, req.GetChannelIds()); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAlertPolicy(ctx, id)
}

func (r *AlertRepository) GetAlertPolicy(ctx context.Context, id string) (*registryv1.AlertPolicy, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, deployment_id, environment_id, enabled, notify_on, cooldown_minutes,
		       send_recovery_notification, filters, created_at, updated_at
		FROM alert_policies
		WHERE id = ? AND deleted_at IS NULL
	`, id)
	item, err := scanAlertPolicy(row)
	if err != nil {
		return nil, err
	}
	channels, err := r.listPolicyChannelIDs(ctx, id)
	if err != nil {
		return nil, err
	}
	item.ChannelIds = channels
	return item, nil
}

func (r *AlertRepository) ListAlertPolicies(ctx context.Context, deploymentID, environmentID string, pageSize int, pageToken string) ([]*registryv1.AlertPolicy, string, error) {
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	offset := offsetFromToken(pageToken)
	query := `
		SELECT id, deployment_id, environment_id, enabled, notify_on, cooldown_minutes,
		       send_recovery_notification, filters, created_at, updated_at
		FROM alert_policies
		WHERE deleted_at IS NULL
	`
	args := make([]any, 0, 4)
	if deploymentID != "" {
		query += ` AND deployment_id = ?`
		args = append(args, deploymentID)
	}
	if environmentID != "" {
		query += ` AND environment_id = ?`
		args = append(args, environmentID)
	}
	query += ` ORDER BY julianday(created_at) DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*registryv1.AlertPolicy, 0, pageSize)
	for rows.Next() {
		item, err := scanAlertPolicy(rows)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	for _, item := range items {
		channels, err := r.listPolicyChannelIDs(ctx, item.GetId())
		if err != nil {
			return nil, "", err
		}
		item.ChannelIds = channels
	}
	return items, nextPageToken(len(items), pageSize, offset), nil
}

func (r *AlertRepository) UpdateAlertPolicy(ctx context.Context, req *registryv1.UpdateAlertPolicyRequest) (*registryv1.AlertPolicy, error) {
	notifyOn := req.GetNotifyOn()
	if len(notifyOn) == 0 {
		notifyOn = []string{"unhealthy", "recovered"}
	}
	cooldownMinutes := req.GetCooldownMinutes()
	if cooldownMinutes <= 0 {
		cooldownMinutes = 10
	}
	notifyOnJSON, err := json.Marshal(notifyOn)
	if err != nil {
		return nil, err
	}
	filtersJSON, err := json.Marshal(req.GetFilters())
	if err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		UPDATE alert_policies
		SET enabled = ?, notify_on = ?, cooldown_minutes = ?, send_recovery_notification = ?,
		    filters = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, req.GetEnabled(), string(notifyOnJSON), cooldownMinutes, req.GetSendRecoveryNotification(), string(filtersJSON), time.Now().UTC().Format(time.RFC3339Nano), req.GetId())
	if err != nil {
		return nil, err
	}
	if err := replacePolicyChannelsTx(ctx, tx, req.GetId(), req.GetChannelIds()); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetAlertPolicy(ctx, req.GetId())
}

func (r *AlertRepository) DeleteAlertPolicy(ctx context.Context, id string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := r.db.Exec(ctx, `
		UPDATE alert_policies
		SET deleted_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`, now, now, id)
	return err
}

func (r *AlertRepository) MatchPolicies(ctx context.Context, incident *registryv1.Incident, notificationType string, at time.Time) ([]*registryv1.AlertPolicy, error) {
	if incident == nil {
		return nil, errors.New("incident is required")
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, deployment_id, environment_id, enabled, notify_on, cooldown_minutes,
		       send_recovery_notification, filters, created_at, updated_at
		FROM alert_policies
		WHERE deleted_at IS NULL
		  AND enabled = 1
		  AND (deployment_id IS NULL OR deployment_id = ?)
		  AND (environment_id IS NULL OR environment_id = ?)
	`, incident.GetDeploymentId(), incident.GetEnvironmentId())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policies := make([]*registryv1.AlertPolicy, 0)
	for rows.Next() {
		policy, err := scanAlertPolicy(rows)
		if err != nil {
			return nil, err
		}
		if !policyMatchesNotification(policy, notificationType) {
			continue
		}
		if notificationType == "recovered" && !policy.GetSendRecoveryNotification() {
			continue
		}
		if onCooldown, err := r.policyOnCooldown(ctx, policy.GetId(), notificationType, at, policy.GetCooldownMinutes()); err != nil {
			return nil, err
		} else if onCooldown {
			continue
		}
		channelIDs, err := r.listPolicyChannelIDs(ctx, policy.GetId())
		if err != nil {
			return nil, err
		}
		if len(channelIDs) == 0 {
			continue
		}
		policy.ChannelIds = channelIDs
		policies = append(policies, policy)
	}
	return policies, rows.Err()
}

func (r *AlertRepository) ListPolicyChannels(ctx context.Context, policy *registryv1.AlertPolicy) ([]*registryv1.NotificationChannel, error) {
	if policy == nil {
		return nil, errors.New("policy is required")
	}
	channels := make([]*registryv1.NotificationChannel, 0, len(policy.GetChannelIds()))
	for _, channelID := range policy.GetChannelIds() {
		channel, err := r.GetNotificationChannel(ctx, channelID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			return nil, err
		}
		if channel.GetEnabled() {
			channels = append(channels, channel)
		}
	}
	return channels, nil
}

func (r *AlertRepository) LogAttempt(ctx context.Context, attempt AlertAttempt) error {
	if attempt.AttemptedAt.IsZero() {
		attempt.AttemptedAt = time.Now().UTC()
	}
	if attempt.Metadata == nil {
		attempt.Metadata = map[string]string{}
	}
	metadataJSON, err := json.Marshal(attempt.Metadata)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO alert_attempts (
			id, incident_id, policy_id, channel_id, notification_type, attempted_at,
			success, status_code, error_message, retry_count, next_retry_at, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		uuid.NewString(),
		attempt.IncidentID,
		attempt.PolicyID,
		attempt.ChannelID,
		attempt.NotificationType,
		attempt.AttemptedAt.UTC().Format(time.RFC3339Nano),
		attempt.Success,
		nullInt(attempt.StatusCode),
		nullIfEmpty(attempt.ErrorMessage),
		attempt.RetryCount,
		nullTime(attempt.NextRetryAt),
		string(metadataJSON),
	)
	return err
}

func (r *AlertRepository) policyOnCooldown(ctx context.Context, policyID, notificationType string, at time.Time, cooldownMinutes int32) (bool, error) {
	if cooldownMinutes <= 0 {
		return false, nil
	}
	var lastAttemptRaw string
	err := r.db.QueryRow(ctx, `
		SELECT attempted_at
		FROM alert_attempts
		WHERE policy_id = ? AND notification_type = ? AND success = 1
		ORDER BY julianday(attempted_at) DESC
		LIMIT 1
	`, policyID, notificationType).Scan(&lastAttemptRaw)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	lastAttempt, err := time.Parse(time.RFC3339Nano, lastAttemptRaw)
	if err != nil {
		return false, nil
	}
	return at.UTC().Before(lastAttempt.Add(time.Duration(cooldownMinutes) * time.Minute)), nil
}

func replacePolicyChannelsTx(ctx context.Context, tx transaction, policyID string, channelIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM alert_policy_channels WHERE policy_id = ?`, policyID); err != nil {
		return err
	}
	for _, channelID := range channelIDs {
		if channelID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO alert_policy_channels (policy_id, channel_id)
			VALUES (?, ?)
		`, policyID, channelID); err != nil {
			return err
		}
	}
	return nil
}

func (r *AlertRepository) listPolicyChannelIDs(ctx context.Context, policyID string) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		SELECT channel_id
		FROM alert_policy_channels
		WHERE policy_id = ?
		ORDER BY channel_id ASC
	`, policyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func scanNotificationChannel(row scanner) (*registryv1.NotificationChannel, error) {
	var (
		id               string
		channelType      string
		name             string
		enabled          bool
		description      sql.NullString
		configurationRaw string
		retryPolicyRaw   string
		tagsRaw          string
		createdRaw       string
		updatedRaw       string
	)
	if err := row.Scan(&id, &channelType, &name, &enabled, &description, &configurationRaw, &retryPolicyRaw, &tagsRaw, &createdRaw, &updatedRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	item := &registryv1.NotificationChannel{
		Id:            id,
		Type:          channelType,
		Name:          name,
		Enabled:       enabled,
		Configuration: map[string]string{},
		RetryPolicy:   map[string]string{},
		Tags:          map[string]string{},
		CreatedAt:     parseTimestamp(createdRaw),
		UpdatedAt:     parseTimestamp(updatedRaw),
	}
	if description.Valid {
		item.Description = description.String
	}
	_ = json.Unmarshal([]byte(configurationRaw), &item.Configuration)
	_ = json.Unmarshal([]byte(retryPolicyRaw), &item.RetryPolicy)
	_ = json.Unmarshal([]byte(tagsRaw), &item.Tags)
	return item, nil
}

func scanAlertPolicy(row scanner) (*registryv1.AlertPolicy, error) {
	var (
		id                       string
		deploymentID             sql.NullString
		environmentID            sql.NullString
		enabled                  bool
		notifyOnRaw              string
		cooldownMinutes          int32
		sendRecoveryNotification bool
		filtersRaw               string
		createdRaw               string
		updatedRaw               string
	)
	if err := row.Scan(&id, &deploymentID, &environmentID, &enabled, &notifyOnRaw, &cooldownMinutes, &sendRecoveryNotification, &filtersRaw, &createdRaw, &updatedRaw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, err
	}
	item := &registryv1.AlertPolicy{
		Id:                       id,
		Enabled:                  enabled,
		NotifyOn:                 []string{},
		CooldownMinutes:          cooldownMinutes,
		SendRecoveryNotification: sendRecoveryNotification,
		Filters:                  map[string]string{},
		ChannelIds:               []string{},
		CreatedAt:                parseTimestamp(createdRaw),
		UpdatedAt:                parseTimestamp(updatedRaw),
	}
	if deploymentID.Valid {
		item.DeploymentId = deploymentID.String
	}
	if environmentID.Valid {
		item.EnvironmentId = environmentID.String
	}
	_ = json.Unmarshal([]byte(notifyOnRaw), &item.NotifyOn)
	_ = json.Unmarshal([]byte(filtersRaw), &item.Filters)
	return item, nil
}

func parseTimestamp(raw string) *timestamppb.Timestamp {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return timestamppb.New(t)
	}
	return timestamppb.Now()
}

func offsetFromToken(pageToken string) int {
	if pageToken == "" {
		return 0
	}
	offset, err := strconv.Atoi(pageToken)
	if err != nil || offset < 0 {
		return 0
	}
	return offset
}

func nextPageToken(count, pageSize, offset int) string {
	if count == pageSize {
		return strconv.Itoa(offset + count)
	}
	return ""
}

func policyMatchesNotification(policy *registryv1.AlertPolicy, notificationType string) bool {
	for _, value := range policy.GetNotifyOn() {
		if value == notificationType {
			return true
		}
	}
	return false
}

func nullInt(value int) any {
	if value == 0 {
		return nil
	}
	return value
}

func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
