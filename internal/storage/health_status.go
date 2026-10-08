package storage

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

type HealthStatusSnapshot struct {
	Services  map[string]string `json:"services"`
	Instances map[string]string `json:"instances"`
	Monitored map[string]bool   `json:"monitored"`
}

type healthStatusQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type instanceHealthCounts struct {
	serviceID string
	enabled   bool
	healthOn  bool
	healthy   int
	unhealthy int
	unknown   int
	checks    int
}

func AggregateHealth(healthy, unhealthy, unknown int) string {
	switch {
	case healthy > 0 && (unhealthy > 0 || unknown > 0):
		return "Degraded"
	case unhealthy > 0:
		return "Unhealthy"
	case healthy > 0:
		return "Healthy"
	default:
		return "Unknown"
	}
}

// CurrentHealth reads the latest completed execution of each active check.
// The same snapshot drives public catalog, management, discovery and the UI.
func (r *HealthRepository) CurrentHealth(ctx context.Context, environmentID string) (HealthStatusSnapshot, error) {
	return currentHealth(ctx, r.db, environmentID, "", environmentAccess(ctx))
}

func currentInstanceHealthTx(ctx context.Context, tx transaction, instanceID string) (string, error) {
	snapshot, err := currentHealth(ctx, tx, "", instanceID, nil)
	return snapshot.Instances[instanceID], err
}

func currentHealth(ctx context.Context, queryer healthStatusQueryer, environmentID, instanceID string, allowedEnvironments []string) (HealthStatusSnapshot, error) {
	snapshot := HealthStatusSnapshot{Services: map[string]string{}, Instances: map[string]string{}, Monitored: map[string]bool{}}
	query := `SELECT d.service_id, si.id, si.enabled, d.health_enabled,
	 hc.id, hc.interval_seconds, hc.type, hc.endpoint_id,
	 ep.enabled, ep.deleted_at, ep.port, si.port,
	 hr.success, hr.timestamp
	 FROM service_instances si
	 JOIN service_deployments d ON d.id=si.deployment_id AND d.deleted_at IS NULL
	 JOIN services s ON s.id=d.service_id AND s.deleted_at IS NULL
	 JOIN environments env ON env.id=d.environment_id AND env.deleted_at IS NULL AND env.enabled=1
	 LEFT JOIN health_checks hc ON hc.instance_id=si.id AND hc.deleted_at IS NULL AND hc.enabled=1
	 LEFT JOIN endpoints ep ON ep.id=hc.endpoint_id
	 LEFT JOIN health_results hr ON hr.id=(SELECT latest.id FROM health_results latest
	     WHERE latest.health_check_id=hc.id ORDER BY julianday(latest.timestamp) DESC, latest.id DESC LIMIT 1)
	 WHERE si.deleted_at IS NULL`
	args := []any{}
	if environmentID != "" {
		query += " AND d.environment_id=?"
		args = append(args, environmentID)
	}
	if instanceID != "" {
		query += " AND si.id=?"
		args = append(args, instanceID)
	}
	if len(allowedEnvironments) > 0 {
		query += " AND d.environment_id IN (" + strings.TrimSuffix(strings.Repeat("?,", len(allowedEnvironments)), ",") + ")"
		for _, id := range allowedEnvironments {
			args = append(args, id)
		}
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return snapshot, err
	}
	defer rows.Close()
	instances := map[string]*instanceHealthCounts{}
	now := time.Now().UTC()
	for rows.Next() {
		var service, instance string
		var enabled, healthOn bool
		var checkID, checkType, endpointID, endpointDeleted, resultAt sql.NullString
		var interval, endpointPort, instancePort sql.NullInt64
		var endpointEnabled, success sql.NullBool
		if err := rows.Scan(&service, &instance, &enabled, &healthOn, &checkID, &interval, &checkType, &endpointID, &endpointEnabled, &endpointDeleted, &endpointPort, &instancePort, &success, &resultAt); err != nil {
			return snapshot, err
		}
		counts := instances[instance]
		if counts == nil {
			counts = &instanceHealthCounts{serviceID: service, enabled: enabled, healthOn: healthOn}
			instances[instance] = counts
		}
		if !enabled || !healthOn || !checkID.Valid {
			continue
		}
		port := instancePort.Int64
		if endpointID.Valid && endpointID.String != "" {
			if !endpointEnabled.Valid || !endpointEnabled.Bool || endpointDeleted.Valid {
				continue
			}
			port = endpointPort.Int64
		}
		if checkType.String != "HEALTH_CHECK_TYPE_DNS" && (port < 1 || port > 65535) {
			continue
		}
		counts.checks++
		if !success.Valid || !resultAt.Valid {
			counts.unknown++
			continue
		}
		at, err := time.Parse(time.RFC3339Nano, resultAt.String)
		if err != nil {
			counts.unknown++
			continue
		}
		freshFor := time.Duration(interval.Int64*3) * time.Second
		if freshFor < 5*time.Minute {
			freshFor = 5 * time.Minute
		}
		if at.After(now.Add(time.Minute)) || now.Sub(at) > freshFor {
			counts.unknown++
		} else if success.Bool {
			counts.healthy++
		} else {
			counts.unhealthy++
		}
	}
	if err := rows.Err(); err != nil {
		return snapshot, err
	}
	serviceCounts := map[string]*instanceHealthCounts{}
	for id, counts := range instances {
		state := "Unknown"
		if !counts.enabled {
			state = "Disabled"
		} else if counts.healthOn {
			state = AggregateHealth(counts.healthy, counts.unhealthy, counts.unknown)
		}
		snapshot.Instances[id] = state
		snapshot.Monitored[id] = counts.enabled && counts.healthOn && counts.checks > 0
		if state == "Disabled" {
			continue
		}
		service := serviceCounts[counts.serviceID]
		if service == nil {
			service = &instanceHealthCounts{}
			serviceCounts[counts.serviceID] = service
		}
		switch state {
		case "Healthy":
			service.healthy++
		case "Unhealthy":
			service.unhealthy++
		case "Degraded":
			service.healthy++
			service.unhealthy++
		default:
			service.unknown++
		}
	}
	for id, counts := range serviceCounts {
		snapshot.Services[id] = AggregateHealth(counts.healthy, counts.unhealthy, counts.unknown)
	}
	return snapshot, nil
}
