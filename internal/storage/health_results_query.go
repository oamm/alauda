package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type HealthResultsQuery struct {
	From, To, Search, ServiceID, EnvironmentID, InstanceID, EndpointID, CheckID, Type, Status string
	Port, PageSize, Offset                                                                    int
	Oldest                                                                                    bool
	AllowedEnvironmentIDs                                                                     []string
}

type HealthResultRecord struct {
	ID             string `json:"id"`
	Timestamp      string `json:"timestamp"`
	ServiceID      string `json:"serviceId"`
	Service        string `json:"service"`
	EnvironmentID  string `json:"environmentId"`
	Environment    string `json:"environment"`
	InstanceID     string `json:"instanceId"`
	Instance       string `json:"instance"`
	EndpointID     string `json:"endpointId"`
	Endpoint       string `json:"endpoint"`
	Protocol       int32  `json:"protocol"`
	Port           int32  `json:"port"`
	Address        string `json:"address"`
	CheckID        string `json:"checkId"`
	Check          string `json:"check"`
	Type           string `json:"type"`
	Path           string `json:"path"`
	ExpectedStatus string `json:"expectedStatus"`
	Success        bool   `json:"success"`
	LatencyMS      *int32 `json:"latencyMs"`
	StatusCode     *int32 `json:"statusCode"`
	ErrorType      string `json:"errorType"`
	ErrorMessage   string `json:"errorMessage"`
}

func (r *HealthRepository) QueryHealthResults(ctx context.Context, q HealthResultsQuery) ([]HealthResultRecord, string, error) {
	where := []string{"1 = 1"}
	args := []any{}
	for _, filter := range []struct{ column, value string }{
		{"d.service_id", q.ServiceID}, {"d.environment_id", q.EnvironmentID}, {"hr.instance_id", q.InstanceID},
		{"hc.endpoint_id", q.EndpointID}, {"hr.health_check_id", q.CheckID}, {"hc.type", q.Type},
	} {
		if filter.value != "" {
			where = append(where, filter.column+" = ?")
			args = append(args, filter.value)
		}
	}
	if q.Search != "" {
		term := "%" + strings.ToLower(q.Search) + "%"
		where = append(where, `(LOWER(s.name) LIKE ? OR LOWER(s.display_name) LIKE ? OR LOWER(env.name) LIKE ? OR LOWER(si.name) LIKE ? OR LOWER(si.address) LIKE ? OR LOWER(e.name) LIKE ? OR LOWER(hc.name) LIKE ? OR LOWER(COALESCE(hr.error_message, '')) LIKE ? OR CAST(COALESCE(e.port, si.port) AS TEXT) LIKE ?)`)
		for range 9 {
			args = append(args, term)
		}
	}
	if q.From != "" {
		where = append(where, "julianday(hr.timestamp) >= julianday(?)")
		args = append(args, q.From)
	}
	if q.To != "" {
		where = append(where, "julianday(hr.timestamp) <= julianday(?)")
		args = append(args, q.To)
	}
	if q.Port > 0 {
		where = append(where, "COALESCE(e.port, si.port) = ?")
		args = append(args, q.Port)
	}
	if q.Status != "" {
		where = append(where, "hr.success = ?")
		args = append(args, q.Status == "healthy")
	}
	if len(q.AllowedEnvironmentIDs) > 0 {
		marks := make([]string, len(q.AllowedEnvironmentIDs))
		for i, id := range q.AllowedEnvironmentIDs {
			marks[i] = "?"
			args = append(args, id)
		}
		where = append(where, "d.environment_id IN ("+strings.Join(marks, ",")+")")
	}
	order := "DESC"
	if q.Oldest {
		order = "ASC"
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 25
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	query := `SELECT hr.id, hr.timestamp, s.id, COALESCE(NULLIF(s.display_name,''),s.name), env.id, env.name,
	 si.id, si.name, COALESCE(e.id,''), COALESCE(e.name,''), COALESCE(e.protocol,0), COALESCE(e.port,si.port), si.address,
	 hc.id, hc.name, hc.type, hc.metadata, COALESCE(e.path,''), hr.success, hr.latency_ms, hr.status_code, COALESCE(hr.error_type,''), COALESCE(hr.error_message,'')
	 FROM health_results hr JOIN health_checks hc ON hc.id=hr.health_check_id
	 JOIN service_instances si ON si.id=hr.instance_id JOIN service_deployments d ON d.id=si.deployment_id
	 JOIN services s ON s.id=d.service_id JOIN environments env ON env.id=d.environment_id
	 LEFT JOIN endpoints e ON e.id=hc.endpoint_id WHERE ` + strings.Join(where, " AND ") + " ORDER BY julianday(hr.timestamp) " + order + ", hr.id " + order + " LIMIT ? OFFSET ?"
	args = append(args, q.PageSize+1, q.Offset)
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	items := []HealthResultRecord{}
	for rows.Next() {
		var item HealthResultRecord
		var metadata string
		var latency, status sql.NullInt32
		if err := rows.Scan(&item.ID, &item.Timestamp, &item.ServiceID, &item.Service, &item.EnvironmentID, &item.Environment, &item.InstanceID, &item.Instance, &item.EndpointID, &item.Endpoint, &item.Protocol, &item.Port, &item.Address, &item.CheckID, &item.Check, &item.Type, &metadata, &item.Path, &item.Success, &latency, &status, &item.ErrorType, &item.ErrorMessage); err != nil {
			return nil, "", err
		}
		if latency.Valid {
			item.LatencyMS = &latency.Int32
		}
		if status.Valid {
			item.StatusCode = &status.Int32
		}
		var fields map[string]string
		_ = json.Unmarshal([]byte(metadata), &fields)
		if fields["path"] != "" {
			item.Path = fields["path"]
		}
		item.ExpectedStatus = fields["expectedStatus"]
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(items) > q.PageSize {
		items = items[:q.PageSize]
		next = fmt.Sprint(q.Offset + q.PageSize)
	}
	return items, next, nil
}
