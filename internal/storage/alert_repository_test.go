package storage

import (
	"context"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestAlertRepositoryNotificationChannels(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	repo := NewAlertRepository(db)
	channel, err := repo.CreateNotificationChannel(ctx, &registryv1.CreateNotificationChannelRequest{
		Type:        "webhook",
		Name:        "ops",
		Enabled:     true,
		Description: "Ops webhook",
		Configuration: map[string]string{
			"url": "https://example.test/hook",
		},
		RetryPolicy: map[string]string{"max_attempts": "3"},
		Tags:        map[string]string{"team": "platform"},
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if channel.GetId() == "" || !channel.GetEnabled() || channel.GetConfiguration()["url"] == "" {
		t.Fatalf("unexpected channel: %+v", channel)
	}

	channels, _, err := repo.ListNotificationChannels(ctx, 10, "")
	if err != nil {
		t.Fatalf("list channels: %v", err)
	}
	if len(channels) != 1 {
		t.Fatalf("channels = %d, want 1", len(channels))
	}

	updated, err := repo.UpdateNotificationChannel(ctx, &registryv1.UpdateNotificationChannelRequest{
		Id:          channel.GetId(),
		Name:        "ops-primary",
		Enabled:     false,
		Description: "Primary ops webhook",
		Configuration: map[string]string{
			"url": "https://example.test/primary",
		},
	})
	if err != nil {
		t.Fatalf("update channel: %v", err)
	}
	if updated.GetName() != "ops-primary" || updated.GetEnabled() {
		t.Fatalf("unexpected updated channel: %+v", updated)
	}

	if err := repo.DeleteNotificationChannel(ctx, channel.GetId()); err != nil {
		t.Fatalf("delete channel: %v", err)
	}
	channels, _, err = repo.ListNotificationChannels(ctx, 10, "")
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(channels) != 0 {
		t.Fatalf("channels after delete = %d, want 0", len(channels))
	}
}

func TestAlertRepositoryMatchPoliciesHonorsCooldown(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	check := createTestHealthCheck(t, ctx, db)
	scope := loadTestScope(t, ctx, db, check.GetInstanceId())
	repo := NewAlertRepository(db)
	channel, err := repo.CreateNotificationChannel(ctx, &registryv1.CreateNotificationChannelRequest{
		Type:    "webhook",
		Name:    "ops",
		Enabled: true,
		Configuration: map[string]string{
			"url": "https://example.test/hook",
		},
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	policy, err := repo.CreateAlertPolicy(ctx, &registryv1.CreateAlertPolicyRequest{
		DeploymentId:             scope.deploymentID,
		EnvironmentId:            scope.environmentID,
		Enabled:                  true,
		NotifyOn:                 []string{"unhealthy", "recovered"},
		CooldownMinutes:          10,
		SendRecoveryNotification: true,
		ChannelIds:               []string{channel.GetId()},
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	start := time.Now().UTC()
	healthRepo := NewHealthRepository(db)
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start)); err != nil {
		t.Fatalf("record first failure: %v", err)
	}
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start.Add(time.Second))); err != nil {
		t.Fatalf("record second failure: %v", err)
	}
	incidents, _, err := NewIncidentRepository(db).List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list incidents: %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("incidents = %d, want 1", len(incidents))
	}
	incident := incidents[0]

	matches, err := repo.MatchPolicies(ctx, incident, "unhealthy", time.Now().UTC())
	if err != nil {
		t.Fatalf("match policies: %v", err)
	}
	if len(matches) != 1 || matches[0].GetId() != policy.GetId() {
		t.Fatalf("matches = %+v, want policy %s", matches, policy.GetId())
	}

	if err := repo.LogAttempt(ctx, AlertAttempt{
		IncidentID:       incident.GetId(),
		PolicyID:         policy.GetId(),
		ChannelID:        channel.GetId(),
		NotificationType: "unhealthy",
		AttemptedAt:      time.Now().UTC(),
		Success:          true,
	}); err != nil {
		t.Fatalf("log attempt: %v", err)
	}
	matches, err = repo.MatchPolicies(ctx, incident, "unhealthy", time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatalf("match policies after attempt: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("matches during cooldown = %d, want 0", len(matches))
	}
}

type testScope struct {
	deploymentID  string
	environmentID string
}

func loadTestScope(t *testing.T, ctx context.Context, db *Database, instanceID string) testScope {
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

func TestAlertRepositoryPolicies(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	check := createTestHealthCheck(t, ctx, db)
	var deploymentID, environmentID string
	if err := db.QueryRow(ctx, `
		SELECT si.deployment_id, d.environment_id
		FROM service_instances si
		JOIN service_deployments d ON d.id = si.deployment_id
		WHERE si.id = ?
	`, check.GetInstanceId()).Scan(&deploymentID, &environmentID); err != nil {
		t.Fatalf("load scope: %v", err)
	}

	repo := NewAlertRepository(db)
	channel, err := repo.CreateNotificationChannel(ctx, &registryv1.CreateNotificationChannelRequest{
		Type:    "webhook",
		Name:    "ops",
		Enabled: true,
		Configuration: map[string]string{
			"url": "https://example.test/hook",
		},
	})
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}

	policy, err := repo.CreateAlertPolicy(ctx, &registryv1.CreateAlertPolicyRequest{
		DeploymentId:             deploymentID,
		EnvironmentId:            environmentID,
		Enabled:                  true,
		NotifyOn:                 []string{"unhealthy"},
		CooldownMinutes:          5,
		SendRecoveryNotification: false,
		Filters:                  map[string]string{"severity": "critical"},
		ChannelIds:               []string{channel.GetId()},
	})
	if err != nil {
		t.Fatalf("create policy: %v", err)
	}
	if len(policy.GetChannelIds()) != 1 || policy.GetChannelIds()[0] != channel.GetId() {
		t.Fatalf("policy channels = %v, want %s", policy.GetChannelIds(), channel.GetId())
	}

	policies, _, err := repo.ListAlertPolicies(ctx, deploymentID, "", 10, "")
	if err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("policies = %d, want 1", len(policies))
	}

	updated, err := repo.UpdateAlertPolicy(ctx, &registryv1.UpdateAlertPolicyRequest{
		Id:                       policy.GetId(),
		Enabled:                  false,
		NotifyOn:                 []string{"unhealthy", "recovered"},
		CooldownMinutes:          15,
		SendRecoveryNotification: true,
		ChannelIds:               []string{},
	})
	if err != nil {
		t.Fatalf("update policy: %v", err)
	}
	if updated.GetEnabled() || updated.GetCooldownMinutes() != 15 || len(updated.GetChannelIds()) != 0 {
		t.Fatalf("unexpected updated policy: %+v", updated)
	}

	if err := repo.DeleteAlertPolicy(ctx, policy.GetId()); err != nil {
		t.Fatalf("delete policy: %v", err)
	}
	policies, _, err = repo.ListAlertPolicies(ctx, deploymentID, "", 10, "")
	if err != nil {
		t.Fatalf("list policies after delete: %v", err)
	}
	if len(policies) != 0 {
		t.Fatalf("policies after delete = %d, want 0", len(policies))
	}
}
