package storage

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestCatalogRepositoriesLifecycleAndEnvironmentIsolation(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()

	envRepo := NewEnvironmentRepository(db)
	serviceRepo := NewServiceRepository(db)
	deploymentRepo := NewDeploymentRepository(db)
	instanceRepo := NewInstanceRepository(db)
	endpointRepo := NewEndpointRepository(db)

	prod, err := envRepo.Create(ctx, &registryv1.CreateEnvironmentRequest{
		Key:  "prod",
		Name: "Production",
		Tier: "prod",
		Tags: map[string]string{"region": "us-east"},
	})
	if err != nil {
		t.Fatalf("create prod environment: %v", err)
	}
	staging, err := envRepo.Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "staging", Name: "Staging"})
	if err != nil {
		t.Fatalf("create staging environment: %v", err)
	}
	if _, err := envRepo.Update(ctx, &registryv1.UpdateEnvironmentRequest{
		Id:          staging.GetId(),
		Name:        "Staging",
		Description: "disabled for filter test",
		Enabled:     false,
		Tier:        "nonprod",
	}); err != nil {
		t.Fatalf("disable staging environment: %v", err)
	}

	enabledEnvs, _, err := envRepo.List(ctx, false, 10, "")
	if err != nil {
		t.Fatalf("list enabled environments: %v", err)
	}
	if len(enabledEnvs) != 1 || enabledEnvs[0].GetId() != prod.GetId() {
		t.Fatalf("enabled environments = %v, want only prod", envIDs(enabledEnvs))
	}
	allEnvs, _, err := envRepo.List(ctx, true, 10, "")
	if err != nil {
		t.Fatalf("list all environments: %v", err)
	}
	if len(allEnvs) != 2 {
		t.Fatalf("all environments = %d, want 2", len(allEnvs))
	}

	service, err := serviceRepo.Create(ctx, &registryv1.CreateServiceRequest{
		Name:        "payments",
		DisplayName: "Payments",
		Tags:        map[string]string{"team": "platform"},
		Metadata:    map[string]string{"owner": "api"},
	})
	if err != nil {
		t.Fatalf("create service: %v", err)
	}
	if _, err := serviceRepo.Update(ctx, &registryv1.UpdateServiceRequest{
		Id:          service.GetId(),
		DisplayName: "Payments API",
		Description: "updated",
		Tags:        map[string]string{"team": "core"},
		Metadata:    map[string]string{"owner": "runtime"},
	}); err != nil {
		t.Fatalf("update service: %v", err)
	}

	deployment, err := deploymentRepo.Create(ctx, &registryv1.CreateDeploymentRequest{
		ServiceId:            service.GetId(),
		EnvironmentId:        prod.GetId(),
		HealthEnabled:        true,
		AlertsEnabled:        true,
		AlertCooldownMinutes: 15,
	})
	if err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	if _, err := deploymentRepo.Update(ctx, &registryv1.UpdateDeploymentRequest{
		Id:                   deployment.GetId(),
		HealthEnabled:        true,
		AlertsEnabled:        false,
		AlertCooldownMinutes: 5,
		Tags:                 map[string]string{"rollout": "blue"},
	}); err != nil {
		t.Fatalf("update deployment: %v", err)
	}

	prodServices, _, err := serviceRepo.List(ctx, prod.GetId(), 10, "")
	if err != nil {
		t.Fatalf("list prod services: %v", err)
	}
	if len(prodServices) != 1 || prodServices[0].GetId() != service.GetId() {
		t.Fatalf("prod services = %v, want payments", serviceIDs(prodServices))
	}
	stagingServices, _, err := serviceRepo.List(ctx, staging.GetId(), 10, "")
	if err != nil {
		t.Fatalf("list staging services: %v", err)
	}
	if len(stagingServices) != 0 {
		t.Fatalf("staging services = %v, want none", serviceIDs(stagingServices))
	}

	deployments, _, err := deploymentRepo.List(ctx, service.GetId(), prod.GetId(), 10, "")
	if err != nil {
		t.Fatalf("list deployments: %v", err)
	}
	if len(deployments) != 1 || deployments[0].GetId() != deployment.GetId() {
		t.Fatalf("deployments = %v, want deployment", deploymentIDs(deployments))
	}

	instance, err := instanceRepo.Create(ctx, &registryv1.CreateInstanceRequest{
		DeploymentId: deployment.GetId(),
		Name:         "payments-1",
		Address:      "127.0.0.1",
		Port:         8080,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}
	if _, err := instanceRepo.Update(ctx, &registryv1.UpdateInstanceRequest{
		Id:          instance.GetId(),
		Address:     "127.0.0.2",
		Port:        9090,
		Description: "updated",
		Enabled:     true,
	}); err != nil {
		t.Fatalf("update instance: %v", err)
	}
	instances, _, err := instanceRepo.List(ctx, deployment.GetId(), 10, "")
	if err != nil {
		t.Fatalf("list instances: %v", err)
	}
	if len(instances) != 1 || instances[0].GetAddress() != "127.0.0.2" {
		t.Fatalf("instances = %+v, want updated instance", instances)
	}

	endpoint, err := endpointRepo.Create(ctx, &registryv1.CreateEndpointRequest{
		InstanceId: instance.GetId(),
		Name:       "http",
		Kind:   registryv1.EndpointKind_ENDPOINT_KIND_HTTP,
		Port:       9090,
		Path:       "/healthz",
		Enabled:    true,
	})
	if err != nil {
		t.Fatalf("create endpoint: %v", err)
	}
	if _, err := endpointRepo.Update(ctx, &registryv1.UpdateEndpointRequest{
		Id:       endpoint.GetId(),
		Name:     "ready",
		Kind: registryv1.EndpointKind_ENDPOINT_KIND_HTTP,
		Port:     9091,
		Path:     "/readyz",
		Enabled:  true,
	}); err != nil {
		t.Fatalf("update endpoint: %v", err)
	}
	endpoints, _, err := endpointRepo.List(ctx, instance.GetId(), 10, "")
	if err != nil {
		t.Fatalf("list endpoints: %v", err)
	}
	if len(endpoints) != 1 || endpoints[0].GetPath() != "/readyz" {
		t.Fatalf("endpoints = %+v, want updated endpoint", endpoints)
	}

	if err := endpointRepo.Delete(ctx, endpoint.GetId()); err != nil {
		t.Fatalf("delete endpoint: %v", err)
	}
	if _, err := endpointRepo.Get(ctx, endpoint.GetId()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("get deleted endpoint error = %v, want sql.ErrNoRows", err)
	}
	if err := instanceRepo.Delete(ctx, instance.GetId()); err != nil {
		t.Fatalf("delete instance: %v", err)
	}
	if err := deploymentRepo.Delete(ctx, deployment.GetId()); err != nil {
		t.Fatalf("delete deployment: %v", err)
	}
	if err := serviceRepo.Delete(ctx, service.GetId()); err != nil {
		t.Fatalf("delete service: %v", err)
	}
	if _, err := serviceRepo.Get(ctx, service.GetId()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("get deleted service error = %v, want sql.ErrNoRows", err)
	}
}

func envIDs(items []*registryv1.Environment) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.GetId())
	}
	return ids
}

func serviceIDs(items []*registryv1.Service) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.GetId())
	}
	return ids
}

func deploymentIDs(items []*registryv1.ServiceDeployment) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.GetId())
	}
	return ids
}
