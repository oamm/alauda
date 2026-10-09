package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type HealthChecksQuery struct {
	Search                string
	ServiceID             string
	EnvironmentID         string
	InstanceID            string
	EndpointID            string
	Enabled               *bool
	LatestStatus          string
	Page                  int
	PageSize              int
	AllowedEnvironmentIDs []string
}

type HealthCheckRecord struct {
	ID                      string `json:"id"`
	Name                    string `json:"name"`
	Enabled                 bool   `json:"enabled"`
	Type                    string `json:"type"`
	IntervalSeconds         int32  `json:"intervalSeconds"`
	TimeoutSeconds          int32  `json:"timeoutSeconds"`
	FailuresBeforeUnhealthy int32  `json:"failuresBeforeUnhealthy"`
	SuccessesBeforeHealthy  int32  `json:"successesBeforeHealthy"`
	Description             string `json:"description"`
	Path                    string `json:"path"`
	ExpectedStatus          string `json:"expectedStatus"`
	ServiceID               string `json:"serviceId"`
	Service                 string `json:"service"`
	EnvironmentID           string `json:"environmentId"`
	Environment             string `json:"environment"`
	InstanceID              string `json:"instanceId"`
	Instance                string `json:"instance"`
	Address                 string `json:"address"`
	EndpointID              string `json:"endpointId"`
	Endpoint                string `json:"endpoint"`
	Kind                    string `json:"kind"`
	Port                    int32  `json:"port"`
	LatestStatus            string `json:"latestStatus"`
	LatestResultAt          string `json:"latestResultAt"`
	LatestDurationMs        *int32 `json:"latestDurationMs"`
	LatestStatusCode        *int32 `json:"latestStatusCode"`
	LatestError             string `json:"latestError"`
}

type HealthChecksQueryResult struct {
	Items    []HealthCheckRecord `json:"items"`
	Total    int                 `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}

func (r *HealthRepository) QueryHealthChecks(ctx context.Context, q HealthChecksQuery) (HealthChecksQueryResult, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize != 25 && q.PageSize != 50 && q.PageSize != 100 {
		q.PageSize = 25
	}
	base, args, err := healthChecksQuerySQL(q)
	if err != nil {
		return HealthChecksQueryResult{}, err
	}
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM ("+base+")", args...).Scan(&total); err != nil {
		return HealthChecksQueryResult{}, err
	}
	args = append(args, q.PageSize, (q.Page-1)*q.PageSize)
	rows, err := r.db.QueryContext(ctx, base+" ORDER BY hc.name COLLATE NOCASE ASC, hc.id ASC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return HealthChecksQueryResult{}, err
	}
	defer rows.Close()
	items := make([]HealthCheckRecord, 0, q.PageSize)
	for rows.Next() {
		var item HealthCheckRecord
		var enabled bool
		var latestStatus, latestAt, latestError sql.NullString
		var latestDuration, latestCode sql.NullInt64
		if err := rows.Scan(&item.ID, &item.Name, &enabled, &item.Type, &item.IntervalSeconds, &item.TimeoutSeconds,
			&item.FailuresBeforeUnhealthy, &item.SuccessesBeforeHealthy, &item.Description, &item.Path, &item.ExpectedStatus,
			&item.ServiceID, &item.Service, &item.EnvironmentID, &item.Environment, &item.InstanceID, &item.Instance, &item.Address,
			&item.EndpointID, &item.Endpoint, &item.Kind, &item.Port, &latestStatus, &latestAt, &latestDuration, &latestCode, &latestError); err != nil {
			return HealthChecksQueryResult{}, err
		}
		item.Enabled = enabled
		item.LatestStatus = "unknown"
		if latestStatus.Valid {
			item.LatestStatus = latestStatus.String
		}
		if latestAt.Valid {
			item.LatestResultAt = latestAt.String
		}
		if latestDuration.Valid {
			v := int32(latestDuration.Int64)
			item.LatestDurationMs = &v
		}
		if latestCode.Valid {
			v := int32(latestCode.Int64)
			item.LatestStatusCode = &v
		}
		if latestError.Valid {
			item.LatestError = latestError.String
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return HealthChecksQueryResult{}, err
	}
	return HealthChecksQueryResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func healthChecksQuerySQL(q HealthChecksQuery) (string, []any, error) {
	latest := `(
		SELECT CASE WHEN hr.success = 1 THEN 'healthy' ELSE 'unhealthy' END
		FROM health_results hr WHERE hr.health_check_id = hc.id ORDER BY julianday(hr.timestamp) DESC, hr.id DESC LIMIT 1
	)`
	latestAt := `(SELECT hr.timestamp FROM health_results hr WHERE hr.health_check_id = hc.id ORDER BY julianday(hr.timestamp) DESC, hr.id DESC LIMIT 1)`
	latestDuration := `(SELECT hr.latency_ms FROM health_results hr WHERE hr.health_check_id = hc.id ORDER BY julianday(hr.timestamp) DESC, hr.id DESC LIMIT 1)`
	latestCode := `(SELECT hr.status_code FROM health_results hr WHERE hr.health_check_id = hc.id ORDER BY julianday(hr.timestamp) DESC, hr.id DESC LIMIT 1)`
	latestError := `(SELECT COALESCE(NULLIF(hr.error_message, ''), hr.error_type) FROM health_results hr WHERE hr.health_check_id = hc.id ORDER BY julianday(hr.timestamp) DESC, hr.id DESC LIMIT 1)`
	query := `SELECT hc.id, hc.name, hc.enabled, hc.type, hc.interval_seconds, hc.timeout_seconds,
		hc.failures_before_unhealthy, hc.successes_before_healthy, COALESCE(hc.description, ''),
		COALESCE(json_extract(hc.metadata, '$.path'), ''), COALESCE(json_extract(hc.metadata, '$.expectedStatus'), ''),
		s.id, COALESCE(NULLIF(s.display_name, ''), s.name), e.id, e.name, i.id, i.name, i.address,
		COALESCE(ep.id, ''), COALESCE(ep.name, 'Instance address'), COALESCE(ep.kind, ''), COALESCE(ep.port, i.port, 0),
		` + latest + `, ` + latestAt + `, ` + latestDuration + `, ` + latestCode + `, ` + latestError + `
		FROM health_checks hc
		JOIN service_instances i ON i.id = hc.instance_id AND i.deleted_at IS NULL
		JOIN service_deployments d ON d.id = i.deployment_id AND d.deleted_at IS NULL
		JOIN services s ON s.id = d.service_id AND s.deleted_at IS NULL
		JOIN environments e ON e.id = d.environment_id AND e.deleted_at IS NULL
		LEFT JOIN endpoints ep ON ep.id = hc.endpoint_id AND ep.deleted_at IS NULL
		WHERE hc.deleted_at IS NULL`
	args := make([]any, 0, 8)
	if q.Search != "" {
		term := "%" + strings.ToLower(q.Search) + "%"
		query += ` AND (LOWER(hc.name) LIKE ? OR LOWER(s.name) LIKE ? OR LOWER(s.display_name) LIKE ? OR LOWER(e.name) LIKE ? OR LOWER(i.name) LIKE ? OR LOWER(i.address) LIKE ? OR LOWER(ep.name) LIKE ? OR CAST(COALESCE(ep.port, i.port, 0) AS TEXT) LIKE ?)`
		for range 8 {
			args = append(args, term)
		}
	}
	if q.ServiceID != "" {
		query += " AND s.id = ?"
		args = append(args, q.ServiceID)
	}
	if q.EnvironmentID != "" {
		query += " AND e.id = ?"
		args = append(args, q.EnvironmentID)
	}
	if q.InstanceID != "" {
		query += " AND i.id = ?"
		args = append(args, q.InstanceID)
	}
	if q.EndpointID != "" {
		query += " AND hc.endpoint_id = ?"
		args = append(args, q.EndpointID)
	}
	if q.Enabled != nil {
		query += " AND hc.enabled = ?"
		args = append(args, *q.Enabled)
	}
	if q.LatestStatus != "" {
		if q.LatestStatus != "healthy" && q.LatestStatus != "unhealthy" && q.LatestStatus != "unknown" {
			return "", nil, fmt.Errorf("invalid latest status")
		}
		if q.LatestStatus == "unknown" {
			query += " AND " + latest + " IS NULL"
		} else {
			query += " AND " + latest + " = ?"
			args = append(args, q.LatestStatus)
		}
	}
	if len(q.AllowedEnvironmentIDs) > 0 {
		placeholders := make([]string, len(q.AllowedEnvironmentIDs))
		for i := range placeholders {
			placeholders[i] = "?"
			args = append(args, q.AllowedEnvironmentIDs[i])
		}
		query += " AND e.id IN (" + strings.Join(placeholders, ",") + ")"
	}
	return query, args, nil
}
