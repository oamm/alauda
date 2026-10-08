package api

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestHealthHandlerGetAvailability(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	check := createAPIHealthCheck(t, ctx, db)
	handler := &healthHandler{availabilityRepo: storage.NewAvailabilityRepository(db)}

	resp, err := handler.GetAvailability(ctx, connect.NewRequest(&registryv1.GetAvailabilityRequest{InstanceId: check.GetInstanceId()}))
	if err != nil {
		t.Fatalf("get availability: %v", err)
	}
	if resp.Msg.GetAvailability_24H().GetInstanceId() != check.GetInstanceId() || resp.Msg.GetAvailability_24H().GetAvailabilityPercent() != 100 {
		t.Fatalf("availability response = %#v", resp.Msg.GetAvailability_24H())
	}
}

func TestIncidentHandlerVerifyRecoveryReturnsResolvedIncident(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	check := createAPIHealthCheck(t, ctx, db)
	healthRepo := storage.NewHealthRepository(db)
	incidentRepo := storage.NewIncidentRepository(db)
	start := time.Now().UTC().Add(-time.Minute)
	for _, at := range []time.Time{start, start.Add(time.Second)} {
		if _, err := healthRepo.RecordHealthResult(ctx, check, &registryv1.HealthResult{Timestamp: timestamppb.New(at), Success: false, Metadata: map[string]string{}}); err != nil {
			t.Fatalf("record failure: %v", err)
		}
	}
	openIncidents, _, err := incidentRepo.List(ctx, "", "", "", check.GetInstanceId(), registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list incidents: %v", err)
	}
	if len(openIncidents) != 1 {
		t.Fatalf("open incidents = %d, want 1", len(openIncidents))
	}
	resolved, err := incidentRepo.ResolveManually(ctx, openIncidents[0].GetId(), "verified manually")
	if err != nil {
		t.Fatalf("resolve incident: %v", err)
	}

	handler := &incidentHandler{repo: incidentRepo, healthRepo: healthRepo}
	getResp, err := handler.GetIncident(ctx, connect.NewRequest(&registryv1.GetIncidentRequest{Id: resolved.GetId()}))
	if err != nil {
		t.Fatalf("get incident: %v", err)
	}
	if getResp.Msg.GetIncident().GetId() != resolved.GetId() {
		t.Fatalf("get incident response = %#v", getResp.Msg)
	}
	listResp, err := handler.ListIncidents(ctx, connect.NewRequest(&registryv1.ListIncidentsRequest{InstanceId: check.GetInstanceId()}))
	if err != nil {
		t.Fatalf("list incidents: %v", err)
	}
	if len(listResp.Msg.GetIncidents()) != 1 {
		t.Fatalf("listed incidents = %d, want 1", len(listResp.Msg.GetIncidents()))
	}
	manualResp, err := handler.ResolveIncidentManually(ctx, connect.NewRequest(&registryv1.ResolveIncidentManuallyRequest{Id: resolved.GetId(), Note: "already handled"}))
	if err != nil {
		t.Fatalf("resolve already resolved incident: %v", err)
	}
	if manualResp.Msg.GetIncident().GetId() != resolved.GetId() {
		t.Fatalf("manual resolve response = %#v", manualResp.Msg)
	}
	resp, err := handler.VerifyIncidentRecovery(ctx, connect.NewRequest(&registryv1.VerifyIncidentRecoveryRequest{Id: resolved.GetId()}))
	if err != nil {
		t.Fatalf("verify recovery: %v", err)
	}
	if !resp.Msg.GetRecovered() || resp.Msg.GetReason() != "incident already resolved" || resp.Msg.GetIncident().GetId() != resolved.GetId() {
		t.Fatalf("verify response = %#v", resp.Msg)
	}
}

func createAPIHealthCheck(t *testing.T, ctx context.Context, db *storage.Database) *registryv1.HealthCheck {
	t.Helper()
	env, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "api-test", Name: "API Test"})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	svc, err := storage.NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "api-svc", DisplayName: "API Service"})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	deployment, err := storage.NewDeploymentRepository(db).Create(ctx, &registryv1.CreateDeploymentRequest{ServiceId: svc.GetId(), EnvironmentId: env.GetId(), HealthEnabled: true})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	instance, err := storage.NewInstanceRepository(db).Create(ctx, &registryv1.CreateInstanceRequest{DeploymentId: deployment.GetId(), Name: "api-inst", Address: "127.0.0.1", Port: 8080, Enabled: true})
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
