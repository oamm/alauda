package storage

import (
	"context"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestQueryHealthResultsFiltersOrderingPaginationAndContext(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()
	repo := NewHealthRepository(db)
	check := createTestHealthCheck(t, ctx, db)
	endpoint, err := NewEndpointRepository(db).Create(ctx, &registryv1.CreateEndpointRequest{InstanceId: check.GetInstanceId(), Name: "default", Kind: registryv1.EndpointKind_ENDPOINT_KIND_HTTP, Port: 9090, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	check, err = repo.CreateHealthCheck(ctx, &registryv1.CreateHealthCheckRequest{InstanceId: check.GetInstanceId(), EndpointId: endpoint.GetId(), Name: "endpoint-ready", Type: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP, Enabled: true, IntervalSeconds: 10, TimeoutSeconds: 3, FailuresBeforeUnhealthy: 3, SuccessesBeforeHealthy: 2, Metadata: map[string]string{"path": "/ready", "expectedStatus": "200"}})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		if _, err := repo.RecordHealthResult(ctx, check, resultAt(i%2 == 0, start.Add(time.Duration(i)*time.Hour))); err != nil {
			t.Fatal(err)
		}
	}
	items, next, err := repo.QueryHealthResults(ctx, HealthResultsQuery{PageSize: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || next != "2" || items[0].Timestamp != start.Add(3*time.Hour).Format(time.RFC3339Nano) {
		t.Fatalf("unexpected first page: %+v, next=%s", items, next)
	}
	context := items[0]
	if context.Service != "Service" || context.Environment != "Test" || context.Instance != "inst-1" || context.Endpoint != "default" || context.Port != 9090 || context.Path != "/ready" {
		t.Fatalf("missing readable context: %+v", context)
	}
	older, next, err := repo.QueryHealthResults(ctx, HealthResultsQuery{PageSize: 2, Offset: 2})
	if err != nil || len(older) != 2 || next != "" || older[0].ID == items[0].ID {
		t.Fatalf("second page: %+v, %s, %v", older, next, err)
	}
	oldest, _, err := repo.QueryHealthResults(ctx, HealthResultsQuery{Oldest: true})
	if err != nil || oldest[0].Timestamp != start.Format(time.RFC3339Nano) {
		t.Fatalf("oldest ordering: %+v %v", oldest, err)
	}
	filters := []HealthResultsQuery{
		{ServiceID: context.ServiceID}, {EnvironmentID: context.EnvironmentID}, {InstanceID: context.InstanceID}, {EndpointID: context.EndpointID}, {CheckID: context.CheckID}, {Type: context.Type}, {Port: 9090},
	}
	for _, q := range filters {
		found, _, err := repo.QueryHealthResults(ctx, q)
		if err != nil || len(found) != 4 {
			t.Fatalf("filter %+v: %d %v", q, len(found), err)
		}
	}
	for _, q := range []HealthResultsQuery{{Search: "endpoint-ready"}, {Search: "ready"}, {Search: "9090"}} {
		found, _, err := repo.QueryHealthResults(ctx, q)
		if err != nil || len(found) != 4 {
			t.Fatalf("search %+v: %d %v", q, len(found), err)
		}
	}
	for _, state := range []string{"healthy", "unhealthy"} {
		found, _, err := repo.QueryHealthResults(ctx, HealthResultsQuery{Status: state})
		if err != nil || len(found) != 2 {
			t.Fatalf("status %s: %d %v", state, len(found), err)
		}
	}
	combined := HealthResultsQuery{From: "2026-10-07T06:00:00-07:00", To: "2026-10-07T14:00:00Z", ServiceID: context.ServiceID, EnvironmentID: context.EnvironmentID, InstanceID: context.InstanceID, EndpointID: context.EndpointID, CheckID: context.CheckID, Type: context.Type, Port: 9090, Status: "healthy", AllowedEnvironmentIDs: []string{context.EnvironmentID}}
	found, _, err := repo.QueryHealthResults(ctx, combined)
	if err != nil || len(found) != 1 || found[0].Timestamp != "2026-10-07T14:00:00Z" {
		t.Fatalf("combined inclusive timezone filter: %+v %v", found, err)
	}
	for _, q := range []HealthResultsQuery{{Port: 8080}, {ServiceID: "missing"}, {AllowedEnvironmentIDs: []string{"not-authorized"}}, {EnvironmentID: context.EnvironmentID, AllowedEnvironmentIDs: []string{"not-authorized"}}} {
		found, _, err := repo.QueryHealthResults(ctx, q)
		if err != nil || len(found) != 0 {
			t.Fatalf("unmatched/restricted query %+v: %+v %v", q, found, err)
		}
	}
}
