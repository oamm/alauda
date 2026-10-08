package storage

import (
	"context"
	"strings"
	"testing"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestCreateEndpointRestoresSoftDeletedName(t *testing.T) {
	for _, primary := range []bool{false, true} {
		name := "non-primary"
		if primary {
			name = "primary"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			db := newTestDatabase(t)
			defer db.Close()
			instance := createEndpointTestInstance(t, ctx, db)
			repo := NewEndpointRepository(db)
			original, err := repo.Create(ctx, &registryv1.CreateEndpointRequest{
				InstanceId: instance.GetId(), Name: "http", Protocol: registryv1.Protocol_PROTOCOL_HTTP,
				Port: 8080, Path: "/old", Enabled: true, Primary: true,
				Tags: map[string]string{"old": "tag"}, Metadata: map[string]string{"old": "value"},
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := repo.Delete(ctx, original.GetId()); err != nil {
				t.Fatal(err)
			}
			visible, _, err := repo.List(ctx, instance.GetId(), 10, "")
			if err != nil || len(visible) != 0 {
				t.Fatalf("deleted endpoint list = %v, error %v; want empty", visible, err)
			}
			restoredEmpty, err := repo.Create(ctx, &registryv1.CreateEndpointRequest{
				InstanceId: instance.GetId(), Name: "http", Protocol: registryv1.Protocol_PROTOCOL_HTTP,
				Port: 8081, Enabled: true,
			})
			if err != nil || restoredEmpty.GetId() != original.GetId() {
				t.Fatalf("restore in empty instance = %v, error %v", restoredEmpty, err)
			}
			if err := repo.Delete(ctx, restoredEmpty.GetId()); err != nil {
				t.Fatal(err)
			}
			other, err := repo.Create(ctx, &registryv1.CreateEndpointRequest{
				InstanceId: instance.GetId(), Name: "tcp", Protocol: registryv1.Protocol_PROTOCOL_TCP,
				Port: 9000, Enabled: true, Primary: true,
			})
			if err != nil {
				t.Fatal(err)
			}
			for attempt := 0; attempt < 2; attempt++ {
				restored, err := repo.Create(ctx, &registryv1.CreateEndpointRequest{
					InstanceId: instance.GetId(), Name: "http", Protocol: registryv1.Protocol_PROTOCOL_TCP,
					Port: 9090, Path: "", Enabled: false, Primary: primary,
					Tags: map[string]string{"new": "tag"}, Metadata: map[string]string{},
				})
				if err != nil {
					t.Fatalf("restore attempt %d: %v", attempt, err)
				}
				if restored.GetId() != original.GetId() || restored.GetProtocol() != registryv1.Protocol_PROTOCOL_TCP ||
					restored.GetPort() != 9090 || restored.GetPath() != "" || restored.GetEnabled() || restored.GetPrimary() != primary {
					t.Fatalf("restored endpoint does not match submitted settings: %+v", restored)
				}
				if len(restored.GetTags()) != 1 || restored.GetTags()["new"] != "tag" || len(restored.GetMetadata()) != 0 {
					t.Fatalf("old tags/metadata retained: %+v", restored)
				}
				otherNow, err := repo.Get(ctx, other.GetId())
				if err != nil || otherNow.GetPrimary() == primary {
					t.Fatalf("other endpoint primary = %v, error %v; want %v", otherNow.GetPrimary(), err, !primary)
				}
				visible, _, err = repo.List(ctx, instance.GetId(), 10, "")
				if err != nil || len(visible) != 2 {
					t.Fatalf("restored endpoint list = %v, error %v; want two", visible, err)
				}
				if err := repo.Delete(ctx, restored.GetId()); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestCreateEndpointRejectsActiveDuplicateWithoutChangingPrimary(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()
	instance := createEndpointTestInstance(t, ctx, db)
	repo := NewEndpointRepository(db)
	for _, name := range []string{"http", "tcp"} {
		_, err := repo.Create(ctx, &registryv1.CreateEndpointRequest{
			InstanceId: instance.GetId(), Name: name, Protocol: registryv1.Protocol_PROTOCOL_HTTP,
			Port: 8080, Enabled: true, Primary: name == "tcp",
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := repo.Create(ctx, &registryv1.CreateEndpointRequest{
		InstanceId: instance.GetId(), Name: "http", Protocol: registryv1.Protocol_PROTOCOL_TCP,
		Port: 9090, Enabled: false, Primary: true,
	})
	if err == nil || !strings.Contains(err.Error(), "UNIQUE constraint failed: endpoints.instance_id, endpoints.name") {
		t.Fatalf("duplicate error = %v; want existing uniqueness error", err)
	}
	items, _, err := repo.List(ctx, instance.GetId(), 10, "")
	if err != nil || len(items) != 2 {
		t.Fatalf("list = %v, error %v", items, err)
	}
	for _, endpoint := range items {
		if endpoint.GetPort() != 8080 || !endpoint.GetEnabled() || endpoint.GetPrimary() != (endpoint.GetName() == "tcp") {
			t.Fatalf("failed duplicate creation changed endpoint: %+v", endpoint)
		}
	}
}

func TestCreateEndpointDeletedNameIsScopedToInstance(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()
	first := createEndpointTestInstance(t, ctx, db)
	second, err := NewInstanceRepository(db).Create(ctx, &registryv1.CreateInstanceRequest{
		DeploymentId: first.GetDeploymentId(), Name: "second", Address: "127.0.0.2", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	repo := NewEndpointRepository(db)
	original, err := repo.Create(ctx, &registryv1.CreateEndpointRequest{
		InstanceId: first.GetId(), Name: "http", Protocol: registryv1.Protocol_PROTOCOL_HTTP, Port: 80, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, original.GetId()); err != nil {
		t.Fatal(err)
	}
	created, err := repo.Create(ctx, &registryv1.CreateEndpointRequest{
		InstanceId: second.GetId(), Name: "http", Protocol: registryv1.Protocol_PROTOCOL_HTTP, Port: 81, Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.GetId() == original.GetId() || created.GetInstanceId() != second.GetId() {
		t.Fatalf("wrong endpoint restored across instances: %+v", created)
	}
	items, _, err := repo.List(ctx, first.GetId(), 10, "")
	if err != nil || len(items) != 0 {
		t.Fatalf("first instance should remain empty: %v, %v", items, err)
	}
}

func createEndpointTestInstance(t *testing.T, ctx context.Context, db *Database) *registryv1.ServiceInstance {
	t.Helper()
	env, err := NewEnvironmentRepository(db).Create(ctx, &registryv1.CreateEnvironmentRequest{Key: "test", Name: "Test"})
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewServiceRepository(db).Create(ctx, &registryv1.CreateServiceRequest{Name: "test"})
	if err != nil {
		t.Fatal(err)
	}
	deployment, err := NewDeploymentRepository(db).Create(ctx, &registryv1.CreateDeploymentRequest{ServiceId: service.GetId(), EnvironmentId: env.GetId()})
	if err != nil {
		t.Fatal(err)
	}
	instance, err := NewInstanceRepository(db).Create(ctx, &registryv1.CreateInstanceRequest{
		DeploymentId: deployment.GetId(), Name: "first", Address: "127.0.0.1", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return instance
}

func TestRegisterRuntimeEndpointEnabledAndPrimarySemantics(t *testing.T) {
	ctx := context.Background()
	db := newTestDatabase(t)
	defer db.Close()
	seed := createEndpointTestInstance(t, ctx, db)
	deployment, err := NewDeploymentRepository(db).Get(ctx, seed.GetDeploymentId())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewRuntimeRepository(db)
	for _, name := range []string{"second", "third"} {
		resp, err := repo.RegisterRuntime(ctx, &registryv1.RegisterRuntimeRequest{
			ServiceId: deployment.GetServiceId(), EnvironmentId: deployment.GetEnvironmentId(),
			Instance: &registryv1.RuntimeInstanceRegistration{Name: name, Address: "127.0.0.2", Enabled: true},
			Endpoints: []*registryv1.RuntimeEndpointRegistration{
				{Name: "default", Protocol: registryv1.Protocol_PROTOCOL_HTTP, Port: 80, Enabled: false},
				{Name: "metrics", Protocol: registryv1.Protocol_PROTOCOL_HTTP, Port: 80, Enabled: true, Primary: true},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		items, _, err := NewEndpointRepository(db).List(ctx, resp.GetInstance().GetId(), 10, "")
		if err != nil || len(items) != 2 {
			t.Fatalf("registered endpoints = %v, error %v", items, err)
		}
		for _, endpoint := range items {
			if endpoint.GetEnabled() != (endpoint.GetName() == "metrics") || endpoint.GetPrimary() != (endpoint.GetName() == "metrics") {
				t.Fatalf("boolean state not preserved: %+v", endpoint)
			}
		}
	}
}
