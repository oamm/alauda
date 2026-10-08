package api

import (
	"net/http"
	"strconv"
	"time"
)

type publicHealthResult struct {
	Check      string `json:"check"`
	Instance   string `json:"instance"`
	Endpoint   string `json:"endpoint"`
	Timestamp  string `json:"timestamp"`
	Success    bool   `json:"success"`
	LatencyMS  int32  `json:"latencyMs"`
	StatusCode int32  `json:"statusCode"`
	ErrorType  string `json:"errorType"`
}

func (a *publicContractAPI) healthResults(w http.ResponseWriter, r *http.Request, key string) {
	if r.Method != http.MethodGet {
		publicError(w, 405, "method_not_allowed", "Use GET.", nil)
		return
	}
	svc, env, ok := a.managementContext(w, r, key)
	if !ok {
		return
	}
	size, offset, ok := publicPage(w, r)
	if !ok {
		return
	}
	q := `SELECT hc.name,i.name,COALESCE(e.name,''),h.timestamp,h.success,COALESCE(h.latency_ms,0),COALESCE(h.status_code,0),COALESCE(h.error_type,'') FROM health_results h JOIN health_checks hc ON hc.id=h.health_check_id JOIN service_instances i ON i.id=h.instance_id JOIN service_deployments d ON d.id=i.deployment_id LEFT JOIN endpoints e ON e.id=hc.endpoint_id WHERE d.service_id=? AND d.environment_id=?`
	args := []any{svc.Id, env.Id}
	for _, filter := range []struct{ name, clause string }{{"instance", " AND i.name=?"}, {"endpoint", " AND e.name=?"}, {"check", " AND hc.name=?"}} {
		if value := r.URL.Query().Get(filter.name); value != "" {
			q += filter.clause
			args = append(args, value)
		}
	}
	for _, filter := range []struct{ name, clause string }{{"from", " AND julianday(h.timestamp)>=julianday(?)"}, {"to", " AND julianday(h.timestamp)<=julianday(?)"}} {
		if value := r.URL.Query().Get(filter.name); value != "" {
			parsed, err := time.Parse(time.RFC3339Nano, value)
			if err != nil {
				publicError(w, 400, "validation_failed", "Invalid timestamp.", map[string][]string{filter.name: {"Use UTC RFC3339."}})
				return
			}
			q += filter.clause
			args = append(args, parsed.UTC().Format(time.RFC3339Nano))
		}
	}
	q += " ORDER BY julianday(h.timestamp) DESC,h.id DESC LIMIT ? OFFSET ?"
	args = append(args, size+1, offset)
	rows, err := a.db.Query(r.Context(), q, args...)
	if err != nil {
		publicStorageError(w, err)
		return
	}
	defer rows.Close()
	items := []publicHealthResult{}
	for rows.Next() {
		var item publicHealthResult
		if err = rows.Scan(&item.Check, &item.Instance, &item.Endpoint, &item.Timestamp, &item.Success, &item.LatencyMS, &item.StatusCode, &item.ErrorType); err != nil {
			publicStorageError(w, err)
			return
		}
		parsed, err := time.Parse(time.RFC3339Nano, item.Timestamp)
		if err != nil {
			publicStorageError(w, err)
			return
		}
		item.Timestamp = parsed.UTC().Format(time.RFC3339Nano)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		publicStorageError(w, err)
		return
	}
	next := ""
	if len(items) > size {
		items = items[:size]
		next = strconv.Itoa(offset + size)
	}
	writeJSON(w, 200, map[string]any{"results": items, "nextPageToken": next})
}
