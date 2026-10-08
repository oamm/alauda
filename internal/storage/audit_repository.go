package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID                string            `json:"id"`
	Timestamp         time.Time         `json:"timestamp"`
	Actor             string            `json:"actor"`
	ActorID           string            `json:"actorId,omitempty"`
	Action            string            `json:"action"`
	ResourceType      string            `json:"resourceType"`
	ResourceID        string            `json:"resourceId"`
	EnvironmentID     string            `json:"environmentId,omitempty"`
	Changes           string            `json:"changes,omitempty"`
	ChangeDescription string            `json:"changeDescription,omitempty"`
	IP                string            `json:"ip,omitempty"`
	UserAgent         string            `json:"userAgent,omitempty"`
	Status            string            `json:"status"`
	ErrorMessage      string            `json:"errorMessage,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

type AuditFilters struct {
	Actor         string
	Action        string
	ResourceType  string
	ResourceID    string
	EnvironmentID string
}

type AuditRepository struct {
	db *Database
}

func NewAuditRepository(db *Database) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Create(ctx context.Context, entry AuditLog) (*AuditLog, error) {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	if entry.Actor == "" {
		entry.Actor = "anonymous"
	}
	if entry.ResourceID == "" {
		entry.ResourceID = "-"
	}
	if entry.Status == "" {
		entry.Status = "success"
	}
	metadataJSON, err := json.Marshal(emptyAuditMetadata(entry.Metadata))
	if err != nil {
		return nil, err
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO audit_logs (
			id, timestamp, actor, actor_id, action, resource_type, resource_id, environment_id,
			changes, change_description, ip, user_agent, status, error_message, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		entry.ID,
		entry.Timestamp.UTC().Format(time.RFC3339Nano),
		entry.Actor,
		nullIfEmptyAudit(entry.ActorID),
		entry.Action,
		entry.ResourceType,
		entry.ResourceID,
		nullIfEmptyAudit(entry.EnvironmentID),
		nullIfEmptyAudit(entry.Changes),
		nullIfEmptyAudit(entry.ChangeDescription),
		nullIfEmptyAudit(entry.IP),
		nullIfEmptyAudit(entry.UserAgent),
		entry.Status,
		nullIfEmptyAudit(entry.ErrorMessage),
		string(metadataJSON),
	)
	if err != nil {
		return nil, err
	}
	return r.Get(ctx, entry.ID)
}

func (r *AuditRepository) Get(ctx context.Context, id string) (*AuditLog, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, timestamp, actor, actor_id, action, resource_type, resource_id, environment_id,
		       changes, change_description, ip, user_agent, status, error_message, metadata
		FROM audit_logs
		WHERE id = ?
	`, id)
	return scanAuditLog(row)
}

func (r *AuditRepository) List(ctx context.Context, filters AuditFilters, pageSize int, pageToken string) ([]*AuditLog, string, error) {
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
		SELECT id, timestamp, actor, actor_id, action, resource_type, resource_id, environment_id,
		       changes, change_description, ip, user_agent, status, error_message, metadata
		FROM audit_logs
		WHERE 1 = 1
	`
	args := make([]any, 0, 7)
	if filters.Actor != "" {
		query += ` AND actor = ?`
		args = append(args, filters.Actor)
	}
	if filters.Action != "" {
		query += ` AND action = ?`
		args = append(args, filters.Action)
	}
	if filters.ResourceType != "" {
		query += ` AND resource_type = ?`
		args = append(args, filters.ResourceType)
	}
	if filters.ResourceID != "" {
		query += ` AND resource_id = ?`
		args = append(args, filters.ResourceID)
	}
	if filters.EnvironmentID != "" {
		query += ` AND environment_id = ?`
		args = append(args, filters.EnvironmentID)
	}
	query += ` ORDER BY julianday(timestamp) DESC, id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	items := make([]*AuditLog, 0, pageSize)
	for rows.Next() {
		item, err := scanAuditLog(rows)
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

func scanAuditLog(row interface{ Scan(...any) error }) (*AuditLog, error) {
	var item AuditLog
	var timestampRaw string
	var actorID, environmentID, changes, changeDescription, ip, userAgent, errorMessage sql.NullString
	var metadataRaw string
	if err := row.Scan(
		&item.ID,
		&timestampRaw,
		&item.Actor,
		&actorID,
		&item.Action,
		&item.ResourceType,
		&item.ResourceID,
		&environmentID,
		&changes,
		&changeDescription,
		&ip,
		&userAgent,
		&item.Status,
		&errorMessage,
		&metadataRaw,
	); err != nil {
		return nil, err
	}
	if t, err := time.Parse(time.RFC3339Nano, timestampRaw); err == nil {
		item.Timestamp = t
	}
	if actorID.Valid {
		item.ActorID = actorID.String
	}
	if environmentID.Valid {
		item.EnvironmentID = environmentID.String
	}
	if changes.Valid {
		item.Changes = changes.String
	}
	if changeDescription.Valid {
		item.ChangeDescription = changeDescription.String
	}
	if ip.Valid {
		item.IP = ip.String
	}
	if userAgent.Valid {
		item.UserAgent = userAgent.String
	}
	if errorMessage.Valid {
		item.ErrorMessage = errorMessage.String
	}
	item.Metadata = map[string]string{}
	_ = json.Unmarshal([]byte(metadataRaw), &item.Metadata)
	return &item, nil
}

func emptyAuditMetadata(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func nullIfEmptyAudit(value string) any {
	if value == "" {
		return nil
	}
	return value
}
