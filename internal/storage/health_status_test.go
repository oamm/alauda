package storage

import (
	"context"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestAggregateHealthMatrix(t *testing.T) {
	for _, test := range []struct {
		healthy, unhealthy, unknown int
		want                        string
	}{
		{0, 0, 0, "Unknown"}, {1, 0, 0, "Healthy"}, {0, 1, 0, "Unhealthy"},
		{0, 0, 1, "Unknown"}, {2, 0, 0, "Healthy"}, {1, 1, 0, "Degraded"},
		{1, 0, 1, "Degraded"}, {0, 1, 1, "Unhealthy"}, {0, 0, 2, "Unknown"},
	} {
		if got := AggregateHealth(test.healthy, test.unhealthy, test.unknown); got != test.want {
			t.Errorf("AggregateHealth(%d,%d,%d)=%s, want %s", test.healthy, test.unhealthy, test.unknown, got, test.want)
		}
	}
}

func TestCurrentHealthLatestResultAndPartialCoverage(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()
	repo := NewHealthRepository(db)
	check := createTestHealthCheck(t, ctx, db)
	var serviceID, deploymentID string
	if err := db.QueryRow(ctx, `SELECT d.service_id,d.id FROM service_instances si JOIN service_deployments d ON d.id=si.deployment_id WHERE si.id=?`, check.InstanceId).Scan(&serviceID, &deploymentID); err != nil {
		t.Fatal(err)
	}
	assert := func(instance, service string) {
		t.Helper()
		snapshot, err := repo.CurrentHealth(ctx, "")
		if err != nil {
			t.Fatal(err)
		}
		if snapshot.Instances[check.InstanceId] != instance || snapshot.Services[serviceID] != service {
			t.Fatalf("instance=%s service=%s, want %s/%s", snapshot.Instances[check.InstanceId], snapshot.Services[serviceID], instance, service)
		}
	}
	assert("Unknown", "Unknown")
	now := time.Now().UTC()
	if _, err := repo.RecordHealthResult(ctx, check, resultAt(true, now)); err != nil {
		t.Fatal(err)
	}
	assert("Healthy", "Healthy")
	if _, err := repo.RecordHealthResult(ctx, check, resultAt(false, now.Add(time.Second))); err != nil {
		t.Fatal(err)
	}
	assert("Unhealthy", "Unhealthy")
	if _, err := repo.RecordHealthResult(ctx, check, resultAt(true, now.Add(2*time.Second))); err != nil {
		t.Fatal(err)
	}
	assert("Healthy", "Healthy")
	second, err := NewInstanceRepository(db).Create(ctx, &registryv1.CreateInstanceRequest{DeploymentId: deploymentID, Name: "unmonitored", Address: "127.0.0.2", Port: 8080, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	assert("Healthy", "Degraded")
	if _, err := repo.RecordHealthResult(ctx, check, resultAt(false, now.Add(3*time.Second))); err != nil {
		t.Fatal(err)
	}
	assert("Unhealthy", "Unhealthy")
	if _, err := db.Exec(ctx, "UPDATE health_results SET timestamp='2000-01-01T00:00:00Z' WHERE health_check_id=?", check.Id); err != nil {
		t.Fatal(err)
	}
	assert("Unknown", "Unknown")
	if _, err := db.Exec(ctx, "UPDATE service_instances SET enabled=0 WHERE id=?", second.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(ctx, "UPDATE service_deployments SET health_enabled=0 WHERE id=?", deploymentID); err != nil {
		t.Fatal(err)
	}
	assert("Unknown", "Unknown")
}
