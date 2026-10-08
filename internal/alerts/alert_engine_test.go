package alerts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
	"github.com/company/service-registry/internal/testutil"
)

func TestEngineRetriesWebhookAndLogsAttempts(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	check := createTestHealthCheck(t, ctx, db)
	scope := loadTestScope(t, ctx, db, check.GetInstanceId())
	alertRepo := storage.NewAlertRepository(db)

	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if requests == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	channel, err := alertRepo.CreateNotificationChannel(ctx, &registryv1.CreateNotificationChannelRequest{
		Type:    "webhook",
		Name:    "ops",
		Enabled: true,
		Configuration: map[string]string{
			"url": server.URL,
		},
		RetryPolicy: map[string]string{
			"max_attempts": "2",
			"backoff_ms":   "1",
		},
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	policy, err := alertRepo.CreateAlertPolicy(ctx, &registryv1.CreateAlertPolicyRequest{
		DeploymentId:             scope.deploymentID,
		EnvironmentId:            scope.environmentID,
		Enabled:                  true,
		NotifyOn:                 []string{"unhealthy"},
		CooldownMinutes:          10,
		SendRecoveryNotification: true,
		ChannelIds:               []string{channel.GetId()},
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}

	start := time.Now().UTC()
	healthRepo := storage.NewHealthRepository(db)
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start)); err != nil {
		t.Fatalf("record first failure: %v", err)
	}
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start.Add(time.Second))); err != nil {
		t.Fatalf("record second failure: %v", err)
	}
	incidents, _, err := storage.NewIncidentRepository(db).List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list incidents: %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("incidents = %d, want 1", len(incidents))
	}

	engine := NewEngine(alertRepo, NewSender(server.Client()))
	engine.sleep = func(context.Context, time.Duration) error { return nil }
	if err := engine.ProcessIncident(ctx, incidents[0], "unhealthy", start.Add(2*time.Second)); err != nil {
		t.Fatalf("process incident: %v", err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}

	var totalAttempts, successfulAttempts int
	if err := db.QueryRow(ctx, `SELECT COUNT(*), SUM(CASE WHEN success THEN 1 ELSE 0 END) FROM alert_attempts WHERE policy_id = ?`, policy.GetId()).Scan(&totalAttempts, &successfulAttempts); err != nil {
		t.Fatalf("count attempts: %v", err)
	}
	if totalAttempts != 2 || successfulAttempts != 1 {
		t.Fatalf("attempts total=%d successful=%d, want 2/1", totalAttempts, successfulAttempts)
	}
}

type testScope struct {
	deploymentID  string
	environmentID string
}

func newTestDatabase(t *testing.T) *storage.Database {
	t.Helper()
	db, err := storage.NewDatabase(context.Background(), testutil.DatabasePath(t))
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	if err := storage.RunMigrations(context.Background(), db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return db
}

func createTestHealthCheck(t *testing.T, ctx context.Context, db *storage.Database) *registryv1.HealthCheck {
	t.Helper()
	env, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "test", Name: "Test"})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	svc, err := storage.NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "svc", DisplayName: "Service"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	deployment, err := storage.NewDeploymentRepository(db).Create(ctx, &registryv1.CreateDeploymentRequest{
		ServiceId:     svc.GetId(),
		EnvironmentId: env.GetId(),
		HealthEnabled: true,
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	instance, err := storage.NewInstanceRepository(db).Create(ctx, &registryv1.CreateInstanceRequest{
		DeploymentId: deployment.GetId(),
		Name:         "inst-1",
		Address:      "127.0.0.1",
		Port:         8080,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	check, err := storage.NewHealthRepository(db).CreateHealthCheck(ctx, &registryv1.CreateHealthCheckRequest{
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

func loadTestScope(t *testing.T, ctx context.Context, db *storage.Database, instanceID string) testScope {
	t.Helper()
	var scope testScope
	if err := db.QueryRow(ctx, `
		SELECT si.deployment_id, d.environment_id
		FROM service_instances si
		JOIN service_deployments d ON d.id = si.deployment_id
		WHERE si.id = ?
	`, instanceID).Scan(&scope.deploymentID, &scope.environmentID); err != nil {
		t.Fatalf("load scope: %v", err)
	}
	return scope
}

func resultAt(success bool, at time.Time) *registryv1.HealthResult {
	return &registryv1.HealthResult{
		Timestamp: nil,
		Success:   success,
		Metadata:  map[string]string{},
	}
}
