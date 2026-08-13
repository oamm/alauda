package health

import (
	"context"
	"log/slog"
	"sync"
	"time"

	registryv1 "github.com/company/service-registry/gen/go/api/registry/v1"
)

type Store interface {
	ListDueHealthCheckTargets(context.Context, int) ([]Target, error)
	RecordHealthResult(context.Context, *registryv1.HealthCheck, *registryv1.HealthResult) (*registryv1.HealthStateView, error)
}

type MonitorConfig struct {
	WorkerCount   int
	CheckInterval time.Duration
	QueueCapacity int
	BatchSize     int
}

type Monitor struct {
	store    Store
	executor *Executor
	cfg      MonitorConfig

	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
	started bool
}

func NewMonitor(store Store, executor *Executor, cfg MonitorConfig) *Monitor {
	if executor == nil {
		executor = NewExecutor(nil)
	}
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = 1
	}
	if cfg.CheckInterval <= 0 {
		cfg.CheckInterval = 10 * time.Second
	}
	if cfg.QueueCapacity <= 0 {
		cfg.QueueCapacity = 100
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = cfg.QueueCapacity
	}
	return &Monitor{
		store:    store,
		executor: executor,
		cfg:      cfg,
		done:     make(chan struct{}),
	}
}

func (m *Monitor) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	m.cancel = cancel
	m.started = true
	queue := make(chan Target, m.cfg.QueueCapacity)

	var workers sync.WaitGroup
	workers.Add(m.cfg.WorkerCount)
	for i := 0; i < m.cfg.WorkerCount; i++ {
		go func() {
			defer workers.Done()
			m.worker(ctx, queue)
		}()
	}

	go func() {
		defer close(m.done)
		m.schedule(ctx, queue)
		close(queue)
		workers.Wait()
	}()
}

func (m *Monitor) Stop() {
	m.once.Do(func() {
		if !m.started {
			return
		}
		if m.cancel != nil {
			m.cancel()
		}
		<-m.done
	})
}

func (m *Monitor) schedule(ctx context.Context, queue chan<- Target) {
	ticker := time.NewTicker(m.cfg.CheckInterval)
	defer ticker.Stop()

	m.enqueueDue(ctx, queue)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.enqueueDue(ctx, queue)
		}
	}
}

func (m *Monitor) enqueueDue(ctx context.Context, queue chan<- Target) {
	targets, err := m.store.ListDueHealthCheckTargets(ctx, m.cfg.BatchSize)
	if err != nil {
		slog.Warn("failed to load due health checks", slog.Any("error", err))
		return
	}

	for _, target := range targets {
		select {
		case <-ctx.Done():
			return
		case queue <- target:
		default:
			slog.Warn("health check queue full", slog.String("check_id", target.Check.GetId()))
			return
		}
	}
}

func (m *Monitor) worker(ctx context.Context, queue <-chan Target) {
	for {
		select {
		case <-ctx.Done():
			return
		case target, ok := <-queue:
			if !ok {
				return
			}
			result := m.executor.Execute(ctx, target)
			if _, err := m.store.RecordHealthResult(ctx, target.Check, result); err != nil {
				slog.Warn("failed to record health result", slog.String("check_id", target.Check.GetId()), slog.Any("error", err))
			}
		}
	}
}
