package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRecordHealthResultUpdatesStateWithThresholds(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	repo := NewHealthRepository(db)
	check := createTestHealthCheck(t, ctx, db)

	firstFailure, err := repo.RecordHealthResult(ctx, check, resultAt(false, time.Now().UTC()))
	if err != nil {
		t.Fatalf("record first failure: %v", err)
	}
	if firstFailure.GetCurrentState() != registryv1.HealthState_HEALTH_STATE_UNKNOWN {
		t.Fatalf("first failure state = %s, want UNKNOWN", firstFailure.GetCurrentState())
	}
	if firstFailure.GetConsecutiveFailures() != 1 {
		t.Fatalf("failures = %d, want 1", firstFailure.GetConsecutiveFailures())
	}

	secondFailure, err := repo.RecordHealthResult(ctx, check, resultAt(false, time.Now().UTC().Add(time.Second)))
	if err != nil {
		t.Fatalf("record second failure: %v", err)
	}
	if secondFailure.GetCurrentState() != registryv1.HealthState_HEALTH_STATE_UNHEALTHY {
		t.Fatalf("second failure state = %s, want UNHEALTHY", secondFailure.GetCurrentState())
	}
	if secondFailure.GetConsecutiveFailures() != 2 {
		t.Fatalf("failures = %d, want 2", secondFailure.GetConsecutiveFailures())
	}

	firstSuccess, err := repo.RecordHealthResult(ctx, check, resultAt(true, time.Now().UTC().Add(2*time.Second)))
	if err != nil {
		t.Fatalf("record first success: %v", err)
	}
	if firstSuccess.GetCurrentState() != registryv1.HealthState_HEALTH_STATE_UNHEALTHY {
		t.Fatalf("first success state = %s, want UNHEALTHY", firstSuccess.GetCurrentState())
	}
	if firstSuccess.GetConsecutiveSuccesses() != 1 || firstSuccess.GetConsecutiveFailures() != 0 {
		t.Fatalf("unexpected counters: successes=%d failures=%d", firstSuccess.GetConsecutiveSuccesses(), firstSuccess.GetConsecutiveFailures())
	}

	secondSuccess, err := repo.RecordHealthResult(ctx, check, resultAt(true, time.Now().UTC().Add(3*time.Second)))
	if err != nil {
		t.Fatalf("record second success: %v", err)
	}
	if secondSuccess.GetCurrentState() != registryv1.HealthState_HEALTH_STATE_HEALTHY {
		t.Fatalf("second success state = %s, want HEALTHY", secondSuccess.GetCurrentState())
	}

	results, _, err := repo.ListHealthResults(ctx, check.GetId(), check.GetInstanceId(), 10, "")
	if err != nil {
		t.Fatalf("list results: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("results count = %d, want 4", len(results))
	}
}

func TestRecordHealthResultDispatchesAlertsAfterTransitions(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	dispatcher := &recordingAlertDispatcher{}
	healthRepo := NewHealthRepositoryWithAlerts(db, dispatcher)
	check := createTestHealthCheck(t, ctx, db)
	start := time.Now().UTC()

	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start)); err != nil {
		t.Fatalf("record first failure: %v", err)
	}
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start.Add(time.Second))); err != nil {
		t.Fatalf("record second failure: %v", err)
	}
	if len(dispatcher.calls) != 1 || dispatcher.calls[0].notificationType != "unhealthy" {
		t.Fatalf("dispatcher calls after unhealthy = %+v, want unhealthy", dispatcher.calls)
	}
	if dispatcher.calls[0].incident.GetId() == "" {
		t.Fatalf("dispatcher received incident without id")
	}

	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(true, start.Add(2*time.Second))); err != nil {
		t.Fatalf("record first success: %v", err)
	}
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(true, start.Add(3*time.Second))); err != nil {
		t.Fatalf("record second success: %v", err)
	}
	if len(dispatcher.calls) != 2 || dispatcher.calls[1].notificationType != "recovered" {
		t.Fatalf("dispatcher calls after recovery = %+v, want recovered", dispatcher.calls)
	}
}

func TestCreateHealthCheckRestoresSoftDeletedName(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	repo := NewHealthRepository(db)
	check := createTestHealthCheck(t, ctx, db)

	if err := repo.DeleteHealthCheck(ctx, check.GetId()); err != nil {
		t.Fatalf("delete health check: %v", err)
	}

	restored, err := repo.CreateHealthCheck(ctx, &registryv1.CreateHealthCheckRequest{
		InstanceId:              check.GetInstanceId(),
		EndpointId:              check.GetEndpointId(),
		Name:                    check.GetName(),
		Type:                    registryv1.HealthCheckType_HEALTH_CHECK_TYPE_TCP,
		Enabled:                 true,
		IntervalSeconds:         15,
		TimeoutSeconds:          4,
		FailuresBeforeUnhealthy: 3,
		SuccessesBeforeHealthy:  2,
		Description:             "restored",
		Metadata:                map[string]string{"path": "/healthz"},
	})
	if err != nil {
		t.Fatalf("restore health check: %v", err)
	}
	if restored.GetId() != check.GetId() {
		t.Fatalf("restored id = %q, want original %q", restored.GetId(), check.GetId())
	}
	if restored.GetType() != registryv1.HealthCheckType_HEALTH_CHECK_TYPE_TCP {
		t.Fatalf("restored type = %s, want TCP", restored.GetType())
	}
	if restored.GetIntervalSeconds() != 15 || restored.GetTimeoutSeconds() != 4 {
		t.Fatalf("restored timing = %ds/%ds, want 15s/4s", restored.GetIntervalSeconds(), restored.GetTimeoutSeconds())
	}
}

type recordingAlertDispatcher struct {
	calls []recordedAlertCall
}

type recordedAlertCall struct {
	incident         *registryv1.Incident
	notificationType string
	at               time.Time
}

func (d *recordingAlertDispatcher) ProcessIncident(_ context.Context, incident *registryv1.Incident, notificationType string, at time.Time) error {
	d.calls = append(d.calls, recordedAlertCall{
		incident:         incident,
		notificationType: notificationType,
		at:               at,
	})
	return nil
}

func TestListDueHealthCheckTargets(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	repo := NewHealthRepository(db)
	check := createTestHealthCheck(t, ctx, db)

	targets, err := repo.ListDueHealthCheckTargets(ctx, 10)
	if err != nil {
		t.Fatalf("list due targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("due targets = %d, want 1", len(targets))
	}
	if targets[0].Check.GetId() != check.GetId() {
		t.Fatalf("due target check id = %q, want %q", targets[0].Check.GetId(), check.GetId())
	}
	if targets[0].Address != "127.0.0.1" || targets[0].Port != 8080 {
		t.Fatalf("unexpected target address: %s:%d", targets[0].Address, targets[0].Port)
	}

	if _, err := repo.RecordHealthResult(ctx, check, resultAt(true, time.Now().UTC())); err != nil {
		t.Fatalf("record result: %v", err)
	}

	targets, err = repo.ListDueHealthCheckTargets(ctx, 10)
	if err != nil {
		t.Fatalf("list due targets after result: %v", err)
	}
	if len(targets) != 0 {
		t.Fatalf("due targets after fresh result = %d, want 0", len(targets))
	}
}

func TestPruneHealthResults(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	repo := NewHealthRepository(db)
	check := createTestHealthCheck(t, ctx, db)
	oldTime := time.Now().UTC().Add(-48 * time.Hour)
	newTime := time.Now().UTC()
	if _, err := repo.RecordHealthResult(ctx, check, resultAt(true, oldTime)); err != nil {
		t.Fatalf("record old result: %v", err)
	}
	if _, err := repo.RecordHealthResult(ctx, check, resultAt(true, newTime)); err != nil {
		t.Fatalf("record new result: %v", err)
	}

	pruned, err := repo.PruneHealthResults(ctx, time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("prune health results: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("pruned = %d, want 1", pruned)
	}
	results, _, err := repo.ListHealthResults(ctx, check.GetId(), check.GetInstanceId(), 10, "")
	if err != nil {
		t.Fatalf("list health results: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("remaining results = %d, want 1", len(results))
	}
}

func TestRecordHealthResultOpensAndResolvesIncidentOnStateTransitions(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	healthRepo := NewHealthRepository(db)
	incidentRepo := NewIncidentRepository(db)
	check := createTestHealthCheck(t, ctx, db)
	start := time.Now().UTC()

	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start)); err != nil {
		t.Fatalf("record first failure: %v", err)
	}
	openIncidents, _, err := incidentRepo.List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list open incidents after first failure: %v", err)
	}
	if len(openIncidents) != 0 {
		t.Fatalf("open incidents after first failure = %d, want 0", len(openIncidents))
	}

	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start.Add(time.Second))); err != nil {
		t.Fatalf("record second failure: %v", err)
	}
	openIncidents, _, err = incidentRepo.List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list open incidents after unhealthy: %v", err)
	}
	if len(openIncidents) != 1 {
		t.Fatalf("open incidents after unhealthy = %d, want 1", len(openIncidents))
	}
	if openIncidents[0].GetReason() != "health check failed" {
		t.Fatalf("incident reason = %q, want default health failure reason", openIncidents[0].GetReason())
	}

	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start.Add(2*time.Second))); err != nil {
		t.Fatalf("record additional failure: %v", err)
	}
	openIncidents, _, err = incidentRepo.List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list open incidents after additional failure: %v", err)
	}
	if len(openIncidents) != 1 {
		t.Fatalf("open incidents after additional failure = %d, want 1", len(openIncidents))
	}

	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(true, start.Add(3*time.Second))); err != nil {
		t.Fatalf("record first success: %v", err)
	}
	openIncidents, _, err = incidentRepo.List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list open incidents before recovery threshold: %v", err)
	}
	if len(openIncidents) != 1 {
		t.Fatalf("open incidents before recovery threshold = %d, want 1", len(openIncidents))
	}

	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(true, start.Add(4*time.Second))); err != nil {
		t.Fatalf("record second success: %v", err)
	}
	resolvedIncidents, _, err := incidentRepo.List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_RESOLVED, 10, "")
	if err != nil {
		t.Fatalf("list resolved incidents: %v", err)
	}
	if len(resolvedIncidents) != 1 {
		t.Fatalf("resolved incidents = %d, want 1", len(resolvedIncidents))
	}
	if resolvedIncidents[0].GetResolvedAt() == nil {
		t.Fatalf("resolved incident has nil resolved_at")
	}
	if resolvedIncidents[0].GetDurationSeconds() <= 0 {
		t.Fatalf("duration_seconds = %d, want positive", resolvedIncidents[0].GetDurationSeconds())
	}

	events, _, err := NewEventRepository(db).List(ctx, EventFilters{InstanceID: check.GetInstanceId()}, 10, "")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[0].GetType() != "incident.resolved" || events[1].GetType() != "incident.opened" {
		t.Fatalf("unexpected event order/types: %q, %q", events[0].GetType(), events[1].GetType())
	}

	availability, err := NewAvailabilityRepository(db).Calculate(ctx, &registryv1.GetAvailabilityRequest{
		InstanceId: check.GetInstanceId(),
	}, 1, start.Add(10*time.Second))
	if err != nil {
		t.Fatalf("calculate availability: %v", err)
	}
	if availability.GetIncidentCount() != 1 {
		t.Fatalf("incident count = %d, want 1", availability.GetIncidentCount())
	}
	if availability.GetDowntimeSeconds() != 3 {
		t.Fatalf("downtime seconds = %d, want 3", availability.GetDowntimeSeconds())
	}
	if availability.GetAvailabilityPercent() <= 99.9 || availability.GetAvailabilityPercent() >= 100 {
		t.Fatalf("availability percent = %f, want between 99.9 and 100", availability.GetAvailabilityPercent())
	}

	var historyCount int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM availability_history WHERE instance_id = ?`, check.GetInstanceId()).Scan(&historyCount); err != nil {
		t.Fatalf("count availability history: %v", err)
	}
	if historyCount != 1 {
		t.Fatalf("availability history rows = %d, want 1", historyCount)
	}
}

func newTestDatabase(t *testing.T) *Database {
	t.Helper()

	db, err := NewDatabase(context.Background(), filepath.Join(t.TempDir(), "registry-test.db"))
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	if err := RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}

func resultAt(success bool, at time.Time) *registryv1.HealthResult {
	return &registryv1.HealthResult{
		Timestamp: timestamppb.New(at),
		Success:   success,
		Metadata:  map[string]string{},
	}
}

func createTestHealthCheck(t *testing.T, ctx context.Context, db *Database) *registryv1.HealthCheck {
	t.Helper()

	env, err := NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{
		Key:  "test",
		Name: "Test",
	})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	svc, err := NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{
		Name:        "svc",
		DisplayName: "Service",
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	deployment, err := NewDeploymentRepository(db).Create(ctx, &registryv1.CreateDeploymentRequest{
		ServiceId:     svc.GetId(),
		EnvironmentId: env.GetId(),
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	instance, err := NewInstanceRepository(db).Create(ctx, &registryv1.CreateInstanceRequest{
		DeploymentId: deployment.GetId(),
		Name:         "inst-1",
		Address:      "127.0.0.1",
		Port:         8080,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	check, err := NewHealthRepository(db).CreateHealthCheck(ctx, &registryv1.CreateHealthCheckRequest{
		InstanceId:              instance.GetId(),
		Name:                    "ready",
		Type:                    registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP,
		Enabled:                 true,
		IntervalSeconds:         10,
		TimeoutSeconds:          3,
		FailuresBeforeUnhealthy: 2,
		SuccessesBeforeHealthy:  2,
	})
	if err != nil {
		t.Fatalf("create health check: %v", err)
	}
	return check
}
