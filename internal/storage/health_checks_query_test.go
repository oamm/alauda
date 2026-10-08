package storage

import (
	"context"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestQueryHealthChecksCatalogFiltersAndPaginates(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()
	seed := createTestHealthCheck(t, ctx, db)
	repo := NewHealthRepository(db)
	for i := 0; i < 26; i++ {
		_, err := repo.CreateHealthCheck(ctx, &registryv1.CreateHealthCheckRequest{
			InstanceId: seed.GetInstanceId(), Name: "check-" + string(rune('a'+i)),
			Type: registryv1.HealthCheckType_HEALTH_CHECK_TYPE_HTTP, Enabled: i%2 == 0,
			IntervalSeconds: 10, TimeoutSeconds: 3,
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	first, err := repo.QueryHealthChecks(ctx, HealthChecksQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if first.Total != 27 || len(first.Items) != 25 || first.PageSize != 25 {
		t.Fatalf("default catalog page = total %d, items %d, size %d; want 27, 25, 25", first.Total, len(first.Items), first.PageSize)
	}
	second, err := repo.QueryHealthChecks(ctx, HealthChecksQuery{Page: 2, PageSize: 25})
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Items) != 2 || second.Items[0].Name == first.Items[len(first.Items)-1].Name {
		t.Fatalf("second catalog page = %+v; want remaining deterministic rows", second.Items)
	}

	search, err := repo.QueryHealthChecks(ctx, HealthChecksQuery{Search: "inst-1", PageSize: 25})
	if err != nil || search.Total != 27 {
		t.Fatalf("human-readable instance search = total %d, error %v; want 27", search.Total, err)
	}
	enabled := false
	disabled, err := repo.QueryHealthChecks(ctx, HealthChecksQuery{Enabled: &enabled, PageSize: 25})
	if err != nil || disabled.Total != 13 {
		t.Fatalf("disabled filter = total %d, error %v; want 13", disabled.Total, err)
	}

	if _, err := repo.RecordHealthResult(ctx, seed, &registryv1.HealthResult{
		Timestamp: timestamppb.New(time.Now().UTC()), Success: true,
	}); err != nil {
		t.Fatal(err)
	}
	healthy, err := repo.QueryHealthChecks(ctx, HealthChecksQuery{LatestStatus: "healthy", PageSize: 25})
	if err != nil || healthy.Total != 1 || healthy.Items[0].LatestStatus != "healthy" {
		t.Fatalf("healthy latest-result filter = %+v, error %v; want seed only", healthy, err)
	}
}
