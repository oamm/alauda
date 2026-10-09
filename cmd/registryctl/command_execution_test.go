package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
	"github.com/company/service-registry/internal/api"
	"github.com/company/service-registry/internal/auth"
	"github.com/company/service-registry/internal/storage"
	"github.com/spf13/cobra"
)

func TestCatalogCommandExecutionPaths(t *testing.T) {
	ctx := context.Background()
	db, err := storage.NewDatabase(ctx, ":memory:")
	if err != nil {
		t.Fatalf("new database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := storage.RunMigrations(ctx, db); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, db)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	webhook := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("webhook method = %s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(webhook.Close)

	oldConfig := cliConfig
	cliConfig = &config{ServerURL: server.URL, Timeout: time.Second}
	t.Cleanup(func() { cliConfig = oldConfig })

	runCommand(t, newAuthUserCreateCommand(), "--username", "admin", "--email", "admin@example.test", "--display-name", "Admin", "--password", "password", "--role", "Administrator")
	runCommand(t, newAuthUserListCommand())
	runCommand(t, newAuthLoginCommand(), "--username", "admin", "--password", "password")
	users, err := auth.NewRepository(db).ListUsers(ctx)
	if err != nil {
		t.Fatalf("list auth users: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("auth users = %d, want 1", len(users))
	}
	createdToken, err := auth.NewRepository(db).CreateToken(ctx, auth.CreateTokenInput{
		UserID:    users[0].ID,
		Name:      "automation",
		Scopes:    []auth.Scope{auth.ScopeRead},
		CreatedBy: users[0].ID,
	})
	if err != nil {
		t.Fatalf("create auth token: %v", err)
	}
	runCommand(t, newAuthTokenListCommand(), "--user-id", users[0].ID)
	runCommand(t, newAuthTokenRevokeCommand(), createdToken.Token.ID)

	runCommand(t, newEnvironmentCreateCommand(), "--key", "prod", "--name", "Production", "--tier", "prod")
	runCommand(t, newEnvironmentListCommand(), "--include-disabled")
	runCommand(t, newServiceCreateCommand(), "--name", "payments", "--display-name", "Payments")
	runCommand(t, newServiceListCommand())

	envs, _, err := storage.NewEnvironmentRepository(db).List(ctx, true, 10, "")
	if err != nil {
		t.Fatalf("list envs: %v", err)
	}
	services, _, err := storage.NewServiceRepository(db).List(ctx, "", 10, "")
	if err != nil {
		t.Fatalf("list services: %v", err)
	}
	if len(envs) != 1 || len(services) != 1 {
		t.Fatalf("seeded envs/services = %d/%d, want 1/1", len(envs), len(services))
	}
	envID := envs[0].GetId()
	serviceID := services[0].GetId()

	runCommand(t, newEnvironmentGetCommand(), envID)
	runCommand(t, newEnvironmentUpdateCommand(), envID, "--name", "Production", "--description", "primary", "--tier", "prod")
	runCommand(t, newServiceGetCommand(), serviceID)
	runCommand(t, newServiceUpdateCommand(), serviceID, "--display-name", "Payments API", "--description", "core")
	runCommand(t, newServiceListCommand(), "--environment-id", envID)

	runCommand(t, newDeploymentCreateCommand(), "--service-id", serviceID, "--environment-id", envID, "--alert-cooldown-minutes", "15")
	deployments, _, err := storage.NewDeploymentRepository(db).List(ctx, serviceID, envID, 10, "")
	if err != nil {
		t.Fatalf("list deployments: %v", err)
	}
	if len(deployments) != 1 {
		t.Fatalf("deployments = %d, want 1", len(deployments))
	}
	deploymentID := deployments[0].GetId()
	runCommand(t, newDeploymentListCommand(), "--service-id", serviceID, "--environment-id", envID)
	runCommand(t, newDeploymentGetCommand(), deploymentID)
	runCommand(t, newDeploymentUpdateCommand(), deploymentID, "--health-enabled=true", "--alerts-enabled=false", "--alert-cooldown-minutes", "5")

	runCommand(t, newInstanceRegisterCommand(), "--deployment-id", deploymentID, "--name", "payments-1", "--address", "127.0.0.1", "--port", "8080")
	instances, _, err := storage.NewInstanceRepository(db).List(ctx, deploymentID, 10, "")
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(instances) != 1 {
		t.Fatalf("instances = %d, want 1", len(instances))
	}
	instanceID := instances[0].GetId()
	runCommand(t, newInstanceListCommand(), "--deployment-id", deploymentID)
	runCommand(t, newInstanceGetCommand(), instanceID)
	runCommand(t, newInstanceUpdateCommand(), instanceID, "--address", "127.0.0.2", "--port", "9090", "--enabled=true")

	runCommand(t, newEndpointCreateCommand(), "--instance-id", instanceID, "--name", "ready", "--kind", "1", "--port", "9090", "--path", "/ready", "--enabled=true")
	endpoints, _, err := storage.NewEndpointRepository(db).List(ctx, instanceID, 10, "")
	if err != nil {
		t.Fatalf("list endpoints: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("endpoints = %d, want 1", len(endpoints))
	}
	endpointID := endpoints[0].GetId()
	runCommand(t, newEndpointListCommand(), "--instance-id", instanceID)
	runCommand(t, newEndpointGetCommand(), endpointID)
	runCommand(t, newEndpointUpdateCommand(), endpointID, "--name", "ready", "--kind", "1", "--port", "9091", "--path", "/healthz", "--enabled=true")

	runCommand(t, newHealthCheckCreateCommand(), "--instance-id", instanceID, "--endpoint-id", endpointID, "--name", "ready", "--type", "http", "--interval-seconds", "10", "--timeout-seconds", "1", "--failures-before-unhealthy", "1", "--successes-before-healthy", "1")
	checks, _, err := storage.NewHealthRepository(db).ListHealthChecks(ctx, instanceID, true, 10, "")
	if err != nil {
		t.Fatalf("list checks: %v", err)
	}
	if len(checks) != 1 {
		t.Fatalf("checks = %d, want 1", len(checks))
	}
	checkID := checks[0].GetId()
	runCommand(t, newHealthCheckListCommand(), "--instance-id", instanceID, "--include-disabled")
	runCommand(t, newHealthCheckRunCommand(), checkID)

	incidents, _, err := storage.NewIncidentRepository(db).List(ctx, envID, serviceID, deploymentID, instanceID, registryv1.IncidentState_INCIDENT_STATE_OPEN, 10, "")
	if err != nil {
		t.Fatalf("list incidents: %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("incidents = %d, want 1", len(incidents))
	}
	incidentID := incidents[0].GetId()
	runCommand(t, newIncidentListCommand(), "--environment-id", envID, "--service-id", serviceID, "--deployment-id", deploymentID, "--instance-id", instanceID, "--state", "open")
	runCommand(t, newIncidentGetCommand(), incidentID)
	runCommand(t, newIncidentResolveManuallyCommand(), incidentID, "--note", "verified by operator")

	runCommand(t, newEventListCommand(), "--environment-id", envID, "--service-id", serviceID, "--deployment-id", deploymentID, "--instance-id", instanceID, "--type", "incident.resolved")

	runCommand(t, newNotificationChannelCreateCommand(), "--type", "webhook", "--name", "pager", "--config", "url="+webhook.URL, "--retry-policy", "max_attempts=1", "--tag", "team=sre")
	channels, _, err := storage.NewAlertRepository(db).ListNotificationChannels(ctx, 10, "")
	if err != nil {
		t.Fatalf("list channels: %v", err)
	}
	if len(channels) != 1 {
		t.Fatalf("channels = %d, want 1", len(channels))
	}
	channelID := channels[0].GetId()
	runCommand(t, newNotificationChannelListCommand())
	runCommand(t, newNotificationChannelTestCommand(), channelID)
	runCommand(t, newAlertPolicyCreateCommand(), "--deployment-id", deploymentID, "--channel", channelID, "--notify-on", "unhealthy,recovered", "--filter", "severity=high")
	runCommand(t, newAlertPolicyListCommand(), "--deployment-id", deploymentID)

	if _, err := storage.NewAuditRepository(db).Create(ctx, storage.AuditLog{
		Actor:         "admin",
		Action:        "update",
		ResourceType:  "service",
		ResourceID:    serviceID,
		EnvironmentID: envID,
		Status:        "success",
		Timestamp:     time.Now().UTC(),
		Metadata:      map[string]string{"source": "test"},
	}); err != nil {
		t.Fatalf("create audit log: %v", err)
	}
	runCommand(t, newAuditCommand(), "--actor", "admin", "--action", "update", "--resource-type", "service", "--resource-id", serviceID, "--environment-id", envID, "--page-size", "10")

	runCommand(t, newEndpointRemoveCommand(), endpointID)
	runCommand(t, newInstanceRemoveCommand(), instanceID)
	runCommand(t, newDeploymentDeleteCommand(), deploymentID)
	runCommand(t, newServiceDeleteCommand(), serviceID)
	runCommand(t, newEnvironmentDeleteCommand(), envID)
}

func TestAuthJSONCommandExecutionPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/me":
			_ = json.NewEncoder(w).Encode(map[string]any{"user": map[string]string{"id": "user-1"}})
		case "/api/v1/auth/tokens":
			_ = json.NewEncoder(w).Encode(map[string]any{"secret": "sr_test", "token": map[string]string{"id": "token-1"}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)

	oldConfig := cliConfig
	cliConfig = &config{ServerURL: server.URL, Timeout: time.Second}
	t.Cleanup(func() { cliConfig = oldConfig })

	runCommand(t, newAuthMeCommand())
	runCommand(t, newAuthTokenCreateCommand(), "--user-id", "user-1", "--name", "ci", "--scope", "read,write", "--expires-in", "1h")

	badExpires := newAuthTokenCreateCommand()
	badExpires.SetArgs([]string{"--user-id", "user-1", "--name", "ci", "--expires-in", "soon"})
	if err := badExpires.Execute(); err == nil {
		t.Fatalf("expected invalid expires-in error")
	}
}

func runCommand(t *testing.T, cmd *cobra.Command, args ...string) {
	t.Helper()
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%s %v: %v", cmd.Use, args, err)
	}
}
