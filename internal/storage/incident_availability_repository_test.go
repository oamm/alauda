package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestIncidentResolveClosesOpenIncidentAndDispatchesAlert(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	dispatcher := &recordingAlertDispatcher{}
	healthRepo := NewHealthRepository(db)
	incidentRepo := NewIncidentRepositoryWithAlerts(db, dispatcher)
	check := createTestHealthCheck(t, ctx, db)
	start := time.Now().UTC().Add(-time.Minute)

	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start)); err != nil {
		t.Fatalf("record first failure: %v", err)
	}
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start.Add(time.Second))); err != nil {
		t.Fatalf("record second failure: %v", err)
	}
	openIncidents, _, err := incidentRepo.List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list open incidents: %v", err)
	}
	if len(openIncidents) != 1 {
		t.Fatalf("open incidents = %d, want 1", len(openIncidents))
	}

	resolved, err := incidentRepo.ResolveManually(ctx, openIncidents[0].GetId(), "operator confirmed maintenance")
	if err != nil {
		t.Fatalf("resolve incident: %v", err)
	}
	if resolved.GetState() != registryv1.IncidentState_INCIDENT_STATE_RESOLVED {
		t.Fatalf("state = %s, want RESOLVED", resolved.GetState())
	}
	if resolved.GetResolvedAt() == nil {
		t.Fatalf("resolved_at is nil")
	}
	if resolved.GetResolutionMethod() != "ManualOverride" || resolved.GetResolutionNote() != "operator confirmed maintenance" {
		t.Fatalf("manual resolution metadata = %q/%q", resolved.GetResolutionMethod(), resolved.GetResolutionNote())
	}
	if len(dispatcher.calls) != 1 || dispatcher.calls[0].notificationType != "recovered" {
		t.Fatalf("dispatcher calls = %+v, want recovered", dispatcher.calls)
	}

	events, _, err := NewEventRepository(db).List(ctx, EventFilters{Type: "incident.resolved"}, 10, "")
	if err != nil {
		t.Fatalf("list resolved events: %v", err)
	}
	if len(events) != 1 || events[0].GetResourceId() != resolved.GetId() {
		t.Fatalf("resolved events = %+v, want incident resolved event", events)
	}
}

func TestIncidentResolveReturnsExistingResolvedIncident(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	healthRepo := NewHealthRepository(db)
	incidentRepo := NewIncidentRepository(db)
	check := createTestHealthCheck(t, ctx, db)
	start := time.Now().UTC().Add(-time.Minute)
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start)); err != nil {
		t.Fatalf("record first failure: %v", err)
	}
	if _, err := healthRepo.RecordHealthResult(ctx, check, resultAt(false, start.Add(time.Second))); err != nil {
		t.Fatalf("record second failure: %v", err)
	}
	openIncidents, _, err := incidentRepo.List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list open incidents: %v", err)
	}
	resolved, err := incidentRepo.ResolveManually(ctx, openIncidents[0].GetId(), "maintenance complete")
	if err != nil {
		t.Fatalf("resolve incident: %v", err)
	}

	again, err := incidentRepo.ResolveManually(ctx, resolved.GetId(), "ignored")
	if err != nil {
		t.Fatalf("resolve already resolved incident: %v", err)
	}
	if again.GetId() != resolved.GetId() || again.GetState() != registryv1.IncidentState_INCIDENT_STATE_RESOLVED {
		t.Fatalf("again = %#v, want existing resolved incident", again)
	}
	if _, err := incidentRepo.ResolveManually(ctx, "missing", "ignored"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("resolve missing error = %v, want sql.ErrNoRows", err)
	}
}

func TestGetAvailabilityCalculatesDefaultWindowsAndStoresSnapshots(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	check := createTestHealthCheck(t, ctx, db)
	resp, err := NewAvailabilityRepository(db).GetAvailability(ctx, &registryv1.GetAvailabilityRequest{
		InstanceId: check.GetInstanceId(),
	})
	if err != nil {
		t.Fatalf("get availability: %v", err)
	}
	summaries := []*registryv1.AvailabilitySummary{
		resp.GetAvailability_24H(),
		resp.GetAvailability_7D(),
		resp.GetAvailability_30D(),
	}
	for _, summary := range summaries {
		if summary.GetInstanceId() != check.GetInstanceId() {
			t.Fatalf("summary instance id = %q, want %q", summary.GetInstanceId(), check.GetInstanceId())
		}
		if summary.GetAvailabilityPercent() != 100 {
			t.Fatalf("availability = %f, want 100", summary.GetAvailabilityPercent())
		}
		if summary.GetDowntimeSeconds() != 0 || summary.GetIncidentCount() != 0 {
			t.Fatalf("unexpected downtime summary: %#v", summary)
		}
	}
	if summaries[0].GetWindowHours() != 24 || summaries[1].GetWindowHours() != 24*7 || summaries[2].GetWindowHours() != 24*30 {
		t.Fatalf("window hours = %d/%d/%d, want 24/168/720", summaries[0].GetWindowHours(), summaries[1].GetWindowHours(), summaries[2].GetWindowHours())
	}

	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM availability_history WHERE instance_id = ?`, check.GetInstanceId()).Scan(&count); err != nil {
		t.Fatalf("count availability history: %v", err)
	}
	if count != 3 {
		t.Fatalf("availability history rows = %d, want 3", count)
	}
}

func TestMergedDowntimeSecondsMergesOverlappingIntervals(t *testing.T) {
	start := time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC)
	got := mergedDowntimeSeconds([]availabilityInterval{
		{start: start, end: start.Add(10 * time.Second)},
		{start: start.Add(5 * time.Second), end: start.Add(20 * time.Second)},
		{start: start.Add(30 * time.Second), end: start.Add(40 * time.Second)},
	})
	if got != 30 {
		t.Fatalf("downtime = %d, want 30", got)
	}
}
