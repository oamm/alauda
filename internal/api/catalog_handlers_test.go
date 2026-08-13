package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/storage"
)

func TestCatalogHandlersLifecycle(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)

	envHandler := &environmentHandler{repo: storage.NewEnvironmentRepository(db)}
	serviceHandler := &catalogHandler{repo: storage.NewServiceRepository(db)}
	deploymentHandler := &deploymentHandler{repo: storage.NewDeploymentRepository(db)}
	instanceHandler := &instanceHandler{repo: storage.NewInstanceRepository(db)}
	endpointHandler := &endpointHandler{repo: storage.NewEndpointRepository(db)}

	envResp, err := envHandler.CreateEnvironment(ctx, connect.NewRequest(&registryv1.CreateEnvironmentRequest{
		Key:  "prod",
		Name: "Production",
	}))
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}
	envID := envResp.Msg.GetEnvironment().GetId()

	svcResp, err := serviceHandler.CreateService(ctx, connect.NewRequest(&registryv1.CreateServiceRequest{
		Name:        "payments",
		DisplayName: "Payments",
	}))
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	serviceID := svcResp.Msg.GetService().GetId()

	depResp, err := deploymentHandler.CreateDeployment(ctx, connect.NewRequest(&registryv1.CreateDeploymentRequest{
		ServiceId:     serviceID,
		EnvironmentId: envID,
		HealthEnabled: true,
	}))
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	deploymentID := depResp.Msg.GetDeployment().GetId()

	instResp, err := instanceHandler.CreateInstance(ctx, connect.NewRequest(&registryv1.CreateInstanceRequest{
		DeploymentId: deploymentID,
		Name:         "payments-1",
		Address:      "127.0.0.1",
		Port:         8080,
		Enabled:      true,
	}))
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	instanceID := instResp.Msg.GetInstance().GetId()

	endpointResp, err := endpointHandler.CreateEndpoint(ctx, connect.NewRequest(&registryv1.CreateEndpointRequest{
		InstanceId: instanceID,
		Name:       "http",
		Protocol:   registryv1.Protocol_PROTOCOL_HTTP,
		Port:       8080,
		Path:       "/healthz",
		Enabled:    true,
	}))
	if err != nil {
		t.Fatalf("create endpoint: %v", err)
	}
	endpointID := endpointResp.Msg.GetEndpoint().GetId()

	if _, err := envHandler.ListEnvironments(ctx, connect.NewRequest(&registryv1.ListEnvironmentsRequest{})); err != nil {
		t.Fatalf("list environments: %v", err)
	}
	if _, err := serviceHandler.ListServices(ctx, connect.NewRequest(&registryv1.ListServicesRequest{EnvironmentId: envID})); err != nil {
		t.Fatalf("list services: %v", err)
	}
	if _, err := deploymentHandler.ListDeployments(ctx, connect.NewRequest(&registryv1.ListDeploymentsRequest{ServiceId: serviceID, EnvironmentId: envID})); err != nil {
		t.Fatalf("list deployments: %v", err)
	}
	if _, err := instanceHandler.ListInstances(ctx, connect.NewRequest(&registryv1.ListInstancesRequest{DeploymentId: deploymentID})); err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if _, err := endpointHandler.ListEndpoints(ctx, connect.NewRequest(&registryv1.ListEndpointsRequest{InstanceId: instanceID})); err != nil {
		t.Fatalf("list endpoints: %v", err)
	}

	if _, err := envHandler.UpdateEnvironment(ctx, connect.NewRequest(&registryv1.UpdateEnvironmentRequest{Id: envID, Name: "Prod", Enabled: true})); err != nil {
		t.Fatalf("update environment: %v", err)
	}
	if _, err := serviceHandler.UpdateService(ctx, connect.NewRequest(&registryv1.UpdateServiceRequest{Id: serviceID, DisplayName: "Payments API"})); err != nil {
		t.Fatalf("update service: %v", err)
	}
	if _, err := deploymentHandler.UpdateDeployment(ctx, connect.NewRequest(&registryv1.UpdateDeploymentRequest{Id: deploymentID, HealthEnabled: true, AlertsEnabled: true})); err != nil {
		t.Fatalf("update deployment: %v", err)
	}
	if _, err := instanceHandler.UpdateInstance(ctx, connect.NewRequest(&registryv1.UpdateInstanceRequest{Id: instanceID, Address: "127.0.0.2", Port: 9090, Enabled: true})); err != nil {
		t.Fatalf("update instance: %v", err)
	}
	if _, err := endpointHandler.UpdateEndpoint(ctx, connect.NewRequest(&registryv1.UpdateEndpointRequest{Id: endpointID, Name: "ready", Port: 9090, Enabled: true})); err != nil {
		t.Fatalf("update endpoint: %v", err)
	}

	if _, err := endpointHandler.DeleteEndpoint(ctx, connect.NewRequest(&registryv1.DeleteEndpointRequest{Id: endpointID})); err != nil {
		t.Fatalf("delete endpoint: %v", err)
	}
	if _, err := instanceHandler.DeleteInstance(ctx, connect.NewRequest(&registryv1.DeleteInstanceRequest{Id: instanceID})); err != nil {
		t.Fatalf("delete instance: %v", err)
	}
	if _, err := deploymentHandler.DeleteDeployment(ctx, connect.NewRequest(&registryv1.DeleteDeploymentRequest{Id: deploymentID})); err != nil {
		t.Fatalf("delete deployment: %v", err)
	}
	if _, err := serviceHandler.DeleteService(ctx, connect.NewRequest(&registryv1.DeleteServiceRequest{Id: serviceID})); err != nil {
		t.Fatalf("delete service: %v", err)
	}
	if _, err := envHandler.DeleteEnvironment(ctx, connect.NewRequest(&registryv1.DeleteEnvironmentRequest{Id: envID})); err != nil {
		t.Fatalf("delete environment: %v", err)
	}
}

func TestCatalogHandlersValidateRequiredFields(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)

	tests := []struct {
		name string
		err  error
	}{
		{"create environment", errFrom(func() (any, error) {
			return (&environmentHandler{repo: storage.NewEnvironmentRepository(db)}).CreateEnvironment(ctx, connect.NewRequest(&registryv1.CreateEnvironmentRequest{}))
		})},
		{"create service", errFrom(func() (any, error) {
			return (&catalogHandler{repo: storage.NewServiceRepository(db)}).CreateService(ctx, connect.NewRequest(&registryv1.CreateServiceRequest{}))
		})},
		{"create deployment", errFrom(func() (any, error) {
			return (&deploymentHandler{repo: storage.NewDeploymentRepository(db)}).CreateDeployment(ctx, connect.NewRequest(&registryv1.CreateDeploymentRequest{}))
		})},
		{"create instance", errFrom(func() (any, error) {
			return (&instanceHandler{repo: storage.NewInstanceRepository(db)}).CreateInstance(ctx, connect.NewRequest(&registryv1.CreateInstanceRequest{}))
		})},
		{"create endpoint", errFrom(func() (any, error) {
			return (&endpointHandler{repo: storage.NewEndpointRepository(db)}).CreateEndpoint(ctx, connect.NewRequest(&registryv1.CreateEndpointRequest{}))
		})},
		{"update service", errFrom(func() (any, error) {
			return (&catalogHandler{repo: storage.NewServiceRepository(db)}).UpdateService(ctx, connect.NewRequest(&registryv1.UpdateServiceRequest{}))
		})},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if connect.CodeOf(tc.err) != connect.CodeInvalidArgument {
				t.Fatalf("code = %s, want invalid_argument: %v", connect.CodeOf(tc.err), tc.err)
			}
		})
	}
}

func TestAlertHandlersLifecycle(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	handler := &alertHandler{repo: storage.NewAlertRepository(db)}
	env, err := storage.NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{
		Key:  "prod",
		Name: "Production",
	})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}

	channelResp, err := handler.CreateNotificationChannel(ctx, connect.NewRequest(&registryv1.CreateNotificationChannelRequest{
		Type:          "webhook",
		Name:          "Primary",
		Enabled:       true,
		Configuration: map[string]string{"url": "https://example.test/hook"},
	}))
	if err != nil {
		t.Fatalf("create notification channel: %v", err)
	}
	channelID := channelResp.Msg.GetChannel().GetId()

	if _, err := handler.GetNotificationChannel(ctx, connect.NewRequest(&registryv1.GetNotificationChannelRequest{Id: channelID})); err != nil {
		t.Fatalf("get notification channel: %v", err)
	}
	if _, err := handler.ListNotificationChannels(ctx, connect.NewRequest(&registryv1.ListNotificationChannelsRequest{})); err != nil {
		t.Fatalf("list notification channels: %v", err)
	}
	if _, err := handler.UpdateNotificationChannel(ctx, connect.NewRequest(&registryv1.UpdateNotificationChannelRequest{
		Id:      channelID,
		Name:    "Primary Updated",
		Enabled: true,
	})); err != nil {
		t.Fatalf("update notification channel: %v", err)
	}

	policyResp, err := handler.CreateAlertPolicy(ctx, connect.NewRequest(&registryv1.CreateAlertPolicyRequest{
		EnvironmentId:            env.GetId(),
		Enabled:                  true,
		NotifyOn:                 []string{"unhealthy", "recovered"},
		CooldownMinutes:          5,
		SendRecoveryNotification: true,
		ChannelIds:               []string{channelID},
		Filters:                  map[string]string{"team": "platform"},
	}))
	if err != nil {
		t.Fatalf("create alert policy: %v", err)
	}
	policyID := policyResp.Msg.GetPolicy().GetId()

	if _, err := handler.GetAlertPolicy(ctx, connect.NewRequest(&registryv1.GetAlertPolicyRequest{Id: policyID})); err != nil {
		t.Fatalf("get alert policy: %v", err)
	}
	if _, err := handler.ListAlertPolicies(ctx, connect.NewRequest(&registryv1.ListAlertPoliciesRequest{EnvironmentId: env.GetId()})); err != nil {
		t.Fatalf("list alert policies: %v", err)
	}
	if _, err := handler.UpdateAlertPolicy(ctx, connect.NewRequest(&registryv1.UpdateAlertPolicyRequest{
		Id:                       policyID,
		Enabled:                  true,
		NotifyOn:                 []string{"unhealthy"},
		CooldownMinutes:          10,
		SendRecoveryNotification: false,
		ChannelIds:               []string{channelID},
	})); err != nil {
		t.Fatalf("update alert policy: %v", err)
	}
	if _, err := handler.DeleteAlertPolicy(ctx, connect.NewRequest(&registryv1.DeleteAlertPolicyRequest{Id: policyID})); err != nil {
		t.Fatalf("delete alert policy: %v", err)
	}
	if _, err := handler.DeleteNotificationChannel(ctx, connect.NewRequest(&registryv1.DeleteNotificationChannelRequest{Id: channelID})); err != nil {
		t.Fatalf("delete notification channel: %v", err)
	}
}

func TestAlertHandlerTestNotificationChannel(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	handler := &alertHandler{repo: storage.NewAlertRepository(db)}
	var payload struct {
		Type      string `json:"type"`
		Test      bool   `json:"test"`
		ChannelID string `json:"channel_id"`
	}
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode webhook payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(webhook.Close)

	channelResp, err := handler.CreateNotificationChannel(ctx, connect.NewRequest(&registryv1.CreateNotificationChannelRequest{
		Type:          "webhook",
		Name:          "Primary",
		Enabled:       true,
		Configuration: map[string]string{"url": webhook.URL},
	}))
	if err != nil {
		t.Fatalf("create notification channel: %v", err)
	}

	resp, err := handler.TestNotificationChannel(ctx, connect.NewRequest(&registryv1.TestNotificationChannelRequest{
		ChannelId: channelResp.Msg.GetChannel().GetId(),
	}))
	if err != nil {
		t.Fatalf("test notification channel: %v", err)
	}
	if !resp.Msg.GetSuccess() || resp.Msg.GetMessage() != "sent" {
		t.Fatalf("response = %+v, want sent success", resp.Msg)
	}
	if payload.Type != "test" || !payload.Test || payload.ChannelID != channelResp.Msg.GetChannel().GetId() {
		t.Fatalf("payload = %+v, want test payload for channel", payload)
	}
}

func TestAlertHandlerTestNotificationChannelErrors(t *testing.T) {
	ctx := context.Background()
	db := newAPITestDB(t, ctx)
	handler := &alertHandler{repo: storage.NewAlertRepository(db)}
	failingWebhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(failingWebhook.Close)
	channelResp, err := handler.CreateNotificationChannel(ctx, connect.NewRequest(&registryv1.CreateNotificationChannelRequest{
		Type:          "webhook",
		Name:          "Primary",
		Enabled:       true,
		Configuration: map[string]string{"url": failingWebhook.URL},
	}))
	if err != nil {
		t.Fatalf("create notification channel: %v", err)
	}

	tests := []struct {
		name string
		req  *registryv1.TestNotificationChannelRequest
		code connect.Code
	}{
		{name: "missing id", req: &registryv1.TestNotificationChannelRequest{}, code: connect.CodeInvalidArgument},
		{name: "not found", req: &registryv1.TestNotificationChannelRequest{ChannelId: "missing"}, code: connect.CodeNotFound},
		{name: "send failed", req: &registryv1.TestNotificationChannelRequest{ChannelId: channelResp.Msg.GetChannel().GetId()}, code: connect.CodeUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := handler.TestNotificationChannel(ctx, connect.NewRequest(tt.req))
			if connect.CodeOf(err) != tt.code {
				t.Fatalf("code = %s, want %s: %v", connect.CodeOf(err), tt.code, err)
			}
		})
	}
}

func errFrom(fn func() (any, error)) error {
	_, err := fn()
	return err
}
