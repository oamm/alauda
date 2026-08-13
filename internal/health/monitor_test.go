package health

import (
	"context"
	"sync"
	"testing"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

func TestMonitorExecutesDueChecksWithWorkers(t *testing.T) {
	store := &fakeStore{
		targets: []Target{{
			Check: &registryv1.HealthCheck{
				Id:         "unsupported",
				InstanceId: "inst-1",
				Type:       registryv1.HealthCheckType_HEALTH_CHECK_TYPE_GRPC,
			},
			Address: "127.0.0.1",
			Port:    1,
		}},
	}

	monitor := NewMonitor(store, NewExecutor(nil), MonitorConfig{
		WorkerCount:   1,
		CheckInterval: 10 * time.Millisecond,
		QueueCapacity: 1,
		BatchSize:     1,
	})
	monitor.Start(context.Background())
	defer monitor.Stop()

	deadline := time.After(time.Second)
	for {
		store.mu.Lock()
		records := len(store.records)
		store.mu.Unlock()
		if records > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for monitor to record a result")
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestMonitorProcessesLargeBatchWithBoundedWorkers(t *testing.T) {
	targets := make([]Target, 1000)
	for i := range targets {
		targets[i] = Target{
			Check: &registryv1.HealthCheck{
				Id:         "check",
				InstanceId: "instance",
				Type:       registryv1.HealthCheckType_HEALTH_CHECK_TYPE_GRPC,
			},
			Address: "127.0.0.1",
			Port:    1,
		}
	}
	store := &fakeStore{targets: targets}

	monitor := NewMonitor(store, NewExecutor(nil), MonitorConfig{
		WorkerCount:   20,
		CheckInterval: time.Hour,
		QueueCapacity: 1000,
		BatchSize:     1000,
	})
	monitor.Start(context.Background())
	defer monitor.Stop()

	deadline := time.After(2 * time.Second)
	for {
		store.mu.Lock()
		records := len(store.records)
		store.mu.Unlock()
		if records >= 1000 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for 1000 records, got %d", records)
		default:
			time.Sleep(time.Millisecond)
		}
	}
}

func TestMonitorStopIsGraceful(t *testing.T) {
	store := &fakeStore{}
	monitor := NewMonitor(store, NewExecutor(nil), MonitorConfig{
		WorkerCount:   2,
		CheckInterval: time.Hour,
		QueueCapacity: 2,
		BatchSize:     2,
	})
	monitor.Start(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		monitor.Stop()
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("monitor did not stop gracefully")
	}
}

func TestMonitorStopBeforeStartDoesNotBlock(t *testing.T) {
	monitor := NewMonitor(&fakeStore{}, NewExecutor(nil), MonitorConfig{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		monitor.Stop()
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("monitor stop before start blocked")
	}
}

type fakeStore struct {
	mu      sync.Mutex
	targets []Target
	records []*registryv1.HealthResult
}

func (s *fakeStore) ListDueHealthCheckTargets(context.Context, int) ([]Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Target(nil), s.targets...), nil
}

func (s *fakeStore) RecordHealthResult(_ context.Context, _ *registryv1.HealthCheck, result *registryv1.HealthResult) (*registryv1.HealthStateView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, result)
	return &registryv1.HealthStateView{}, nil
}
