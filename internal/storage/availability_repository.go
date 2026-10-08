package storage

import (
	"context"
	"database/sql"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AvailabilityRepository struct {
	db *Database
}

type availabilityInterval struct {
	start time.Time
	end   time.Time
}

func NewAvailabilityRepository(db *Database) *AvailabilityRepository {
	return &AvailabilityRepository{db: db}
}

func (r *AvailabilityRepository) GetAvailability(ctx context.Context, req *registryv1.GetAvailabilityRequest) (*registryv1.GetAvailabilityResponse, error) {
	end := time.Now().UTC()
	availability24h, err := r.Calculate(ctx, req, 24, end)
	if err != nil {
		return nil, err
	}
	availability7d, err := r.Calculate(ctx, req, 24*7, end)
	if err != nil {
		return nil, err
	}
	availability30d, err := r.Calculate(ctx, req, 24*30, end)
	if err != nil {
		return nil, err
	}
	return &registryv1.GetAvailabilityResponse{
		Availability_24H: availability24h,
		Availability_7D:  availability7d,
		Availability_30D: availability30d,
	}, nil
}

func (r *AvailabilityRepository) Calculate(ctx context.Context, req *registryv1.GetAvailabilityRequest, windowHours int32, end time.Time) (*registryv1.AvailabilitySummary, error) {
	if windowHours <= 0 {
		windowHours = 24
	}
	end = end.UTC()
	start := end.Add(-time.Duration(windowHours) * time.Hour)

	query := `
		SELECT opened_at, resolved_at
		FROM incidents
		WHERE opened_at < ?
		  AND COALESCE(resolved_at, ?) > ?
	`
	args := []any{
		end.Format(time.RFC3339Nano),
		end.Format(time.RFC3339Nano),
		start.Format(time.RFC3339Nano),
	}
	if req.GetEnvironmentId() != "" {
		query += ` AND environment_id = ?`
		args = append(args, req.GetEnvironmentId())
	}
	if req.GetServiceId() != "" {
		query += ` AND service_id = ?`
		args = append(args, req.GetServiceId())
	}
	if req.GetDeploymentId() != "" {
		query += ` AND deployment_id = ?`
		args = append(args, req.GetDeploymentId())
	}
	if req.GetInstanceId() != "" {
		query += ` AND instance_id = ?`
		args = append(args, req.GetInstanceId())
	}
	query, args = appendEnvironmentAccess(ctx, query, "environment_id", args)
	query += ` ORDER BY opened_at ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	intervals := make([]availabilityInterval, 0)
	for rows.Next() {
		var openedRaw string
		var resolvedRaw sql.NullString
		if err := rows.Scan(&openedRaw, &resolvedRaw); err != nil {
			return nil, err
		}
		openedAt, err := time.Parse(time.RFC3339Nano, openedRaw)
		if err != nil {
			continue
		}
		resolvedAt := end
		if resolvedRaw.Valid {
			if parsed, err := time.Parse(time.RFC3339Nano, resolvedRaw.String); err == nil {
				resolvedAt = parsed.UTC()
			}
		}
		if openedAt.Before(start) {
			openedAt = start
		}
		if resolvedAt.After(end) {
			resolvedAt = end
		}
		if resolvedAt.After(openedAt) {
			intervals = append(intervals, availabilityInterval{start: openedAt.UTC(), end: resolvedAt.UTC()})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	downtimeSeconds := mergedDowntimeSeconds(intervals)
	windowSeconds := int64(end.Sub(start).Seconds())
	availabilityPercent := 100.0
	if windowSeconds > 0 {
		availabilityPercent = (float64(windowSeconds-downtimeSeconds) / float64(windowSeconds)) * 100
		if availabilityPercent < 0 {
			availabilityPercent = 0
		}
	}

	summary := &registryv1.AvailabilitySummary{
		EnvironmentId:       req.GetEnvironmentId(),
		ServiceId:           req.GetServiceId(),
		DeploymentId:        req.GetDeploymentId(),
		InstanceId:          req.GetInstanceId(),
		WindowHours:         windowHours,
		WindowStart:         timestamppb.New(start),
		WindowEnd:           timestamppb.New(end),
		AvailabilityPercent: availabilityPercent,
		DowntimeSeconds:     downtimeSeconds,
		IncidentCount:       int32(len(intervals)),
	}
	if err := r.storeSnapshot(ctx, summary); err != nil {
		return nil, err
	}
	return summary, nil
}

func (r *AvailabilityRepository) storeSnapshot(ctx context.Context, summary *registryv1.AvailabilitySummary) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO availability_history (
			id, environment_id, service_id, deployment_id, instance_id, window_hours,
			window_start, window_end, availability_percent, downtime_seconds, incident_count,
			calculated_at, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}')
	`,
		uuid.NewString(),
		nullIfEmpty(summary.GetEnvironmentId()),
		nullIfEmpty(summary.GetServiceId()),
		nullIfEmpty(summary.GetDeploymentId()),
		nullIfEmpty(summary.GetInstanceId()),
		summary.GetWindowHours(),
		summary.GetWindowStart().AsTime().UTC().Format(time.RFC3339Nano),
		summary.GetWindowEnd().AsTime().UTC().Format(time.RFC3339Nano),
		summary.GetAvailabilityPercent(),
		summary.GetDowntimeSeconds(),
		summary.GetIncidentCount(),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func mergedDowntimeSeconds(intervals []availabilityInterval) int64 {
	if len(intervals) == 0 {
		return 0
	}
	total := int64(0)
	current := intervals[0]
	for _, interval := range intervals[1:] {
		if !interval.start.After(current.end) {
			if interval.end.After(current.end) {
				current.end = interval.end
			}
			continue
		}
		total += int64(current.end.Sub(current.start).Seconds())
		current = interval
	}
	total += int64(current.end.Sub(current.start).Seconds())
	return total
}
