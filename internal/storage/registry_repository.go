package storage

import (
	"context"
	"database/sql"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

type RegistryRepository struct {
	db *Database
}

func NewRegistryRepository(db *Database) *RegistryRepository {
	return &RegistryRepository{db: db}
}

func (r *RegistryRepository) ResolveService(ctx context.Context, serviceName, environmentKey string, healthyOnly bool) (string, string, []*registryv1.ResolvedEndpoint, error) {
	return r.resolve(ctx, serviceName, "", environmentKey, healthyOnly)
}

func (r *RegistryRepository) ResolveEndpoint(ctx context.Context, serviceName, endpointName, environmentKey string, healthyOnly bool) (string, string, []*registryv1.ResolvedEndpoint, error) {
	return r.resolve(ctx, serviceName, endpointName, environmentKey, healthyOnly)
}

func (r *RegistryRepository) resolve(ctx context.Context, serviceName, endpointName, environmentKey string, healthyOnly bool) (string, string, []*registryv1.ResolvedEndpoint, error) {
	query := `
		SELECT s.id, d.id, si.id, COALESCE(e.id, ''), si.address, COALESCE(e.port, si.port, 0),
		       COALESCE(e.path, ''), COALESCE(e.protocol, 0), COALESCE(hs.current_state, 'HEALTH_STATE_UNKNOWN')
		FROM services s
		JOIN service_deployments d ON d.service_id = s.id AND d.deleted_at IS NULL
		JOIN environments env ON env.id = d.environment_id AND env.deleted_at IS NULL
		JOIN service_instances si ON si.deployment_id = d.id AND si.deleted_at IS NULL AND si.enabled = 1
		LEFT JOIN endpoints e ON e.instance_id = si.id AND e.deleted_at IS NULL AND e.enabled = 1
		LEFT JOIN health_states hs ON hs.instance_id = si.id
		WHERE s.deleted_at IS NULL AND s.name = ? AND env.key = ?
	`
	args := []any{serviceName, environmentKey}
	if endpointName != "" {
		query += ` AND e.name = ?`
		args = append(args, endpointName)
	}
	if healthyOnly {
		query += ` AND hs.current_state = ?`
		args = append(args, registryv1.HealthState_HEALTH_STATE_HEALTHY.String())
	}
	query += ` ORDER BY si.name ASC, e.name ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return "", "", nil, err
	}
	defer rows.Close()

	var serviceID, deploymentID string
	endpoints := []*registryv1.ResolvedEndpoint{}
	for rows.Next() {
		var (
			rowServiceID    string
			rowDeploymentID string
			instanceID      string
			endpointID      string
			host            string
			port            int32
			path            string
			protocol        int32
			healthStateRaw  string
		)
		if err := rows.Scan(&rowServiceID, &rowDeploymentID, &instanceID, &endpointID, &host, &port, &path, &protocol, &healthStateRaw); err != nil {
			return "", "", nil, err
		}
		if serviceID == "" {
			serviceID = rowServiceID
			deploymentID = rowDeploymentID
		}
		endpoints = append(endpoints, &registryv1.ResolvedEndpoint{
			InstanceId:  instanceID,
			EndpointId:  endpointID,
			Host:        host,
			Port:        port,
			Path:        path,
			Protocol:    registryv1.Protocol(protocol),
			HealthState: parseHealthState(healthStateRaw),
		})
	}
	if err := rows.Err(); err != nil {
		return "", "", nil, err
	}
	if serviceID == "" {
		return "", "", nil, sql.ErrNoRows
	}
	if endpointName != "" && len(endpoints) == 0 {
		return "", "", nil, sql.ErrNoRows
	}
	return serviceID, deploymentID, endpoints, nil
}
