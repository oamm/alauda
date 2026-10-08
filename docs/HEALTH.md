# Health Scheduler Architecture

Current Health Check, Instance and Service state semantics are defined in [HEALTH_STATE.md](HEALTH_STATE.md). The diagrams and pseudocode below describe the original scheduler design; threshold transitions and cache examples are historical and are not the current status contract.

The health scheduler is the core subsystem responsible for continuously monitoring service availability through automated health checks. This document describes its design, concurrency model, and execution guarantees.

---

## Overview

The health scheduler is responsible for:

1. **Loading** active health checks from the database
2. **Determining** which checks are due for execution
3. **Queuing** ready checks for execution
4. **Executing** checks using bounded worker pool
5. **Evaluating** results and determining state transitions
6. **Persisting** health state changes and creating incidents
7. **Triggering** alerts when health changes
8. **Emitting** events for UI updates and audit trail

### Key Design Principle

**Network I/O never executes inside a database transaction.**

This prevents database locks during slow or hanging network operations.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Health Scheduler                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Scheduler Loop (main)                   │  │
│  │  • Runs every 100ms                                 │  │
│  │  • Loads active health checks from DB               │  │
│  │  • Determines checks due for execution              │  │
│  │  • Queues ready checks                              │  │
│  └──────┬───────────────────────────────────────────────┘  │
│         │                                                    │
│  ┌──────▼────────────────────────────────────────────────┐  │
│  │         Check Queue (buffered channel)               │  │
│  │  • Capacity: 1000 (configurable)                     │  │
│  │  • Blocks when full                                  │  │
│  │  • Drains on shutdown                                │  │
│  └──────┬────────────────────────────────────────────────┘  │
│         │                                                    │
│  ┌──────▴────────────────────────────────────────────────┐  │
│  │         Worker Pool (N workers)                      │  │
│  │  • Default N=20 (configurable)                       │  │
│  │  • Each worker runs in separate goroutine           │  │
│  │  • Bounded concurrency (no unlimited goroutines)     │  │
│  │                                                       │  │
│  │  Worker 1 ─┐                                         │  │
│  │  Worker 2 ─┼─► [Check Executor]                      │  │
│  │  Worker N ─┘                                         │  │
│  │              • Makes network request                 │  │
│  │              • Handles timeout                       │  │
│  │              • Evaluates response                    │  │
│  │              • Publishes result                      │  │
│  └──────┬────────────────────────────────────────────────┘  │
│         │                                                    │
│  ┌──────▼────────────────────────────────────────────────┐  │
│  │       Result Queue (buffered channel)                │  │
│  │  • Capacity: 1000 (configurable)                     │  │
│  │  • Non-blocking publish (drops on full)              │  │
│  └──────┬────────────────────────────────────────────────┘  │
│         │                                                    │
│  ┌──────▼────────────────────────────────────────────────┐  │
│  │     State Transition Engine                          │  │
│  │  • Evaluates result against thresholds              │  │
│  │  • Determines if state transition occurred          │  │
│  │  • No network I/O (database only)                    │  │
│  └──────┬────────────────────────────────────────────────┘  │
│         │                                                    │
│  ┌──────▼────────────────────────────────────────────────┐  │
│  │     Persistence & Event Engine                       │  │
│  │  • BEGIN TRANSACTION                                │  │
│  │  • Update health_states table                        │  │
│  │  • Update health_results table (retention)           │  │
│  │  • Create incident if state → UNHEALTHY             │  │
│  │  • Resolve incident if state → HEALTHY              │  │
│  │  • Publish domain events                             │  │
│  │  • COMMIT TRANSACTION                                │  │
│  └──────┬────────────────────────────────────────────────┘  │
│         │                                                    │
│  ┌──────▼────────────────────────────────────────────────┐  │
│  │     Alert Engine (subscription handler)              │  │
│  │  • Receives domain events                            │  │
│  │  • Finds matching alert policies                     │  │
│  │  • Checks cooldown timers                            │  │
│  │  • Sends notifications (async)                       │  │
│  └────────────────────────────────────────────────────────┘  │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Details

### 1. Scheduler Loop

**Responsibility**: Determine which checks are due and queue them.

```go
// Pseudocode
func (s *Scheduler) Run(ctx context.Context) {
  ticker := time.NewTicker(100 * time.Millisecond)
  defer ticker.Stop()

  for {
    select {
    case <-ctx.Done():
      return
    case <-ticker.C:
      s.tick()
    }
  }
}

func (s *Scheduler) tick() {
  // Load enabled checks (with caching)
  checks := s.repo.GetEnabledHealthChecks(ctx)
  
  now := time.Now()
  for _, check := range checks {
    // Determine if check is due
    if s.isDue(check, now) {
      // Queue for execution
      select {
      case s.checkQueue <- check:
        // Queued successfully
      default:
        // Queue full, skip (logged as dropped check)
      }
    }
  }
}

func (s *Scheduler) isDue(check HealthCheck, now time.Time) bool {
  if check.NextDueTime == nil {
    return true  // Never run before
  }
  return now.After(*check.NextDueTime)
}
```

**Key Properties**:
- Runs every 100ms (tunable)
- Non-blocking queue publish
- Skips checks if queue full (logged)
- Loads checks from database (not in-memory)
- Calculates next due time after each execution

### 2. Check Queue

**Responsibility**: Buffer scheduled checks awaiting execution.

```go
type CheckQueue struct {
  ch chan HealthCheck
}

// Non-blocking send
func (q *CheckQueue) Enqueue(check HealthCheck) error {
  select {
  case q.ch <- check:
    return nil
  default:
    return ErrQueueFull  // Logged, check skipped
  }
}

// Blocking receive
func (q *CheckQueue) Dequeue(ctx context.Context) (HealthCheck, error) {
  select {
  case check := <-q.ch:
    return check, nil
  case <-ctx.Done():
    return HealthCheck{}, ctx.Err()
  }
}
```

**Sizing**:
- Default: 1000 items
- Configurable via environment variable
- At 20 workers × 10s intervals, queue rarely fills
- Full queue indicates scheduler overload

### 3. Worker Pool

**Responsibility**: Execute individual health checks concurrently with bounded concurrency.

```go
type WorkerPool struct {
  workers int
  queue   *CheckQueue
}

func (wp *WorkerPool) Start(ctx context.Context) {
  for i := 0; i < wp.workers; i++ {
    go wp.worker(ctx)  // Each runs independently
  }
}

func (wp *WorkerPool) worker(ctx context.Context) {
  for {
    check, err := wp.queue.Dequeue(ctx)
    if err != nil {
      return  // Context canceled
    }
    
    result := wp.executeCheck(ctx, check)
    wp.publishResult(result)
  }
}

func (wp *WorkerPool) executeCheck(ctx context.Context, check HealthCheck) HealthCheckResult {
  // Enforce timeout at context level
  ctx, cancel := context.WithTimeout(ctx, check.Timeout)
  defer cancel()
  
  start := time.Now()
  
  switch check.Type {
  case HTTP, HTTPS:
    return wp.executeHTTPCheck(ctx, check)
  case TCP:
    return wp.executeTCPCheck(ctx, check)
  case GRPC:
    return wp.executeGRPCCheck(ctx, check)
  case HEARTBEAT:
    return wp.executeHeartbeatCheck(ctx, check)
  }
}

func (wp *WorkerPool) executeHTTPCheck(ctx context.Context, check HealthCheck) HealthCheckResult {
  result := HealthCheckResult{
    ID:            uuid.New().String(),
    HealthCheckID: check.ID,
    InstanceID:    check.InstanceID,
    Timestamp:     time.Now(),
  }
  
  req, _ := http.NewRequestWithContext(ctx, check.HTTPMethod, check.URL(), nil)
  
  resp, err := wp.httpClient.Do(req)
  duration := time.Since(start)
  
  if err != nil {
    // Categorize error
    if ctx.Err() == context.DeadlineExceeded {
      result.ErrorType = "TIMEOUT"
      result.ErrorMessage = "Health check timed out"
    } else if errors.Is(err, net.ErrClosed) {
      result.ErrorType = "CONNECTION_REFUSED"
    } else {
      result.ErrorType = "UNKNOWN"
    }
    result.Success = false
    result.LatencyMs = int32(duration.Milliseconds())
    return result
  }
  defer resp.Body.Close()
  
  result.Success = contains(check.ExpectedStatusCodes, resp.StatusCode)
  result.StatusCode = int32(resp.StatusCode)
  result.LatencyMs = int32(duration.Milliseconds())
  
  return result
}
```

**Concurrency Model**:
- Fixed number of workers (default 20)
- Each worker runs in separate goroutine
- Workers compete for checks from queue
- Total goroutines: workers + scheduler + other components
- No unbounded goroutine creation

**Timeout Enforcement**:
- Context deadline enforced per check
- Network timeouts abort immediately
- Hung requests cleaned up by Go runtime

### 4. Result Queue & State Transition Engine

**Responsibility**: Evaluate results and determine state transitions without network I/O.

```go
type StateTransitionEngine struct {
  resultQueue chan HealthCheckResult
  repo        HealthRepository
}

func (ste *StateTransitionEngine) Run(ctx context.Context) {
  for {
    select {
    case <-ctx.Done():
      return
    case result := <-ste.resultQueue:
      ste.handleResult(ctx, result)
    }
  }
}

func (ste *StateTransitionEngine) handleResult(ctx context.Context, result HealthCheckResult) {
  // Non-blocking; load current state
  check, _ := ste.repo.GetHealthCheck(ctx, result.HealthCheckID)
  state, _ := ste.repo.GetHealthState(ctx, result.InstanceID)
  
  // Update consecutive counters
  if result.Success {
    state.ConsecutiveFailures = 0
    state.ConsecutiveSuccesses++
  } else {
    state.ConsecutiveSuccesses = 0
    state.ConsecutiveFailures++
  }
  
  oldState := state.CurrentState
  
  // Evaluate transition rules
  newState := ste.evaluateState(state, check)
  
  // Determine if transition occurred
  if newState != oldState {
    ste.transitionState(ctx, state, oldState, newState, result)
  } else {
    // Persist result only (no state change)
    ste.persistResult(ctx, result)
  }
}

func (ste *StateTransitionEngine) evaluateState(state *HealthState, check *HealthCheck) HealthStateEnum {
  // Rules:
  // 1. All checks healthy → HEALTHY
  // 2. Consecutive failures >= threshold → UNHEALTHY
  // 3. Consecutive successes >= threshold (from UNHEALTHY) → HEALTHY
  // 4. Some checks degraded → DEGRADED (future)
  // 5. Instance disabled → DISABLED
  
  if !check.Enabled || !state.Enabled {
    return DISABLED
  }
  
  switch state.CurrentState {
  case HEALTHY:
    if state.ConsecutiveFailures >= check.FailuresBeforeUnhealthy {
      return UNHEALTHY
    }
    return HEALTHY
    
  case UNHEALTHY:
    if state.ConsecutiveSuccesses >= check.SuccessesBeforeHealthy {
      return HEALTHY
    }
    return UNHEALTHY
    
  case UNKNOWN:
    if state.ConsecutiveFailures >= check.FailuresBeforeUnhealthy {
      return UNHEALTHY
    }
    if state.ConsecutiveSuccesses > 0 {
      return HEALTHY
    }
    return UNKNOWN
  }
}

func (ste *StateTransitionEngine) transitionState(ctx context.Context, 
  state *HealthState, oldState HealthStateEnum, newState HealthStateEnum, 
  result HealthCheckResult) {
  
  // Single transaction for all changes
  txn, _ := ste.repo.BeginTransaction(ctx)
  
  defer func() {
    if r := recover(); r != nil {
      txn.Rollback()
      panic(r)
    }
  }()
  
  // Update health state
  state.CurrentState = newState
  state.LastTransitionTime = time.Now()
  state.ConsecutiveFailures = 0
  state.ConsecutiveSuccesses = 0
  ste.repo.UpdateHealthState(txn, state)
  
  // Save result
  result.CreatedAt = time.Now()
  result.ExpiresAt = time.Now().Add(24 * time.Hour)
  ste.repo.InsertHealthResult(txn, result)
  
  // Create incident if transitioning to UNHEALTHY
  if newState == UNHEALTHY && oldState != UNHEALTHY {
    incident := &Incident{
      ID:            uuid.New().String(),
      InstanceID:    state.InstanceID,
      State:         OPEN,
      OpenedAt:      time.Now(),
      Reason:        fmt.Sprintf("Failed health check: %s", result.ErrorMessage),
      ImpactSummary: fmt.Sprintf("Instance %s became unhealthy", state.InstanceID),
    }
    ste.repo.InsertIncident(txn, incident)
    ste.publishEvent(txn, &IncidentOpenedEvent{IncidentID: incident.ID})
  }
  
  // Resolve incident if transitioning to HEALTHY
  if newState == HEALTHY && oldState == UNHEALTHY {
    incident, _ := ste.repo.GetOpenIncident(txn, state.InstanceID)
    if incident != nil {
      incident.State = RESOLVED
      incident.ResolvedAt = time.Now()
      incident.DurationSeconds = int32(incident.ResolvedAt.Sub(incident.OpenedAt).Seconds())
      ste.repo.UpdateIncident(txn, incident)
      ste.publishEvent(txn, &IncidentResolvedEvent{IncidentID: incident.ID})
    }
  }
  
  // Publish state change event
  ste.publishEvent(txn, &HealthChangedEvent{
    InstanceID: state.InstanceID,
    OldState:   oldState,
    NewState:   newState,
  })
  
  txn.Commit()
}

func (ste *StateTransitionEngine) persistResult(ctx context.Context, result HealthCheckResult) {
  // Result-only persistence (no state change)
  txn, _ := ste.repo.BeginTransaction(ctx)
  result.ExpiresAt = time.Now().Add(24 * time.Hour)
  ste.repo.InsertHealthResult(txn, result)
  txn.Commit()
}
```

**Key Properties**:
- Never executes network calls
- Deterministic state transitions
- Single transaction for consistency
- Publishes events via event bus
- Non-blocking consumption of results

### 5. Event Publishing

```go
type EventBus struct {
  subscribers map[string][]EventHandler
  mu          sync.RWMutex
}

func (eb *EventBus) Publish(event DomainEvent) {
  eb.mu.RLock()
  handlers := eb.subscribers[event.Type()]
  eb.mu.RUnlock()
  
  for _, handler := range handlers {
    go handler.Handle(event)  // Async, non-blocking
  }
}

func (eb *EventBus) Subscribe(eventType string, handler EventHandler) {
  eb.mu.Lock()
  defer eb.mu.Unlock()
  eb.subscribers[eventType] = append(eb.subscribers[eventType], handler)
}
```

**Event Subscribers**:
1. **Audit Logger**: Logs all events to audit_logs table
2. **Alert Engine**: Triggers notifications
3. **Event Stream**: Sends events to UI (WebSocket or SSE)
4. **Metrics Collector**: Updates gauges and counters

---

## Health Check Execution Patterns

### HTTP/HTTPS Check

```go
func (wp *WorkerPool) executeHTTPCheck(ctx context.Context, check HealthCheck) {
  // Construct URL
  scheme := "http"
  if check.Protocol == HTTPS {
    scheme = "https"
  }
  url := fmt.Sprintf("%s://%s:%d%s", scheme, check.Address, check.Port, check.Path)
  
  // Create request with timeout
  ctx, cancel := context.WithTimeout(ctx, check.Timeout)
  defer cancel()
  
  req, _ := http.NewRequestWithContext(ctx, check.HTTPMethod, url, nil)
  
  // Add custom headers
  for k, v := range check.RequestHeaders {
    req.Header.Set(k, v)
  }
  
  // Execute
  resp, err := wp.httpClient.Do(req)
  
  // Handle response
  if err != nil {
    return errorResult(err)
  }
  
  // Check status code
  success := contains(check.ExpectedStatusCodes, resp.StatusCode)
  
  return successResult(success, resp.StatusCode, latency)
}
```

### TCP Check

```go
func (wp *WorkerPool) executeTCPCheck(ctx context.Context, check HealthCheck) {
  ctx, cancel := context.WithTimeout(ctx, check.Timeout)
  defer cancel()
  
  addr := fmt.Sprintf("%s:%d", check.Address, check.Port)
  
  dialer := net.Dialer{}
  conn, err := dialer.DialContext(ctx, "tcp", addr)
  
  if err != nil {
    return errorResult(err)
  }
  conn.Close()
  
  return successResult(true, 0, latency)
}
```

### gRPC Check

```go
func (wp *WorkerPool) executeGRPCCheck(ctx context.Context, check HealthCheck) {
  ctx, cancel := context.WithTimeout(ctx, check.Timeout)
  defer cancel()
  
  addr := fmt.Sprintf("%s:%d", check.Address, check.Port)
  
  conn, err := grpc.DialContext(ctx, addr, grpc.WithInsecure())
  if err != nil {
    return errorResult(err)
  }
  defer conn.Close()
  
  // Check health using gRPC Health Checking Protocol
  client := healthpb.NewHealthClient(conn)
  res, err := client.Check(ctx, &healthpb.HealthCheckRequest{
    Service: check.GRPCService,
  })
  
  if err != nil {
    return errorResult(err)
  }
  
  success := res.Status == healthpb.HealthCheckResponse_SERVING
  return successResult(success, 0, latency)
}
```

### Heartbeat Check

```go
func (wp *WorkerPool) executeHeartbeatCheck(ctx context.Context, check HealthCheck) {
  // Get instance
  instance, _ := wp.repo.GetInstance(ctx, check.InstanceID)
  
  // Check if last seen is within TTL
  if instance.LastSeenAt == nil {
    return HealthCheckResult{
      Success:     false,
      ErrorType:   "NEVER_SEEN",
      ErrorMessage: "Instance has never reported",
    }
  }
  
  age := time.Since(*instance.LastSeenAt)
  success := age < check.HeartbeatTTL
  
  return HealthCheckResult{
    Success:   success,
    LatencyMs: int32(age.Milliseconds()),
  }
}
```

---

## Lifecycle Management

### Startup

```go
func (s *Scheduler) Start(ctx context.Context) error {
  // Verify database connectivity
  err := s.repo.Ping(ctx)
  if err != nil {
    return fmt.Errorf("database not ready: %w", err)
  }
  
  // Start worker pool
  s.pool.Start(ctx)
  
  // Start scheduler loop
  go s.Run(ctx)
  
  // Start state transition engine
  go s.stateEngine.Run(ctx)
  
  // Start alert engine
  go s.alertEngine.Run(ctx)
  
  return nil
}
```

### Graceful Shutdown

```go
func (s *Scheduler) Stop(ctx context.Context) error {
  // Stop accepting new checks
  s.running = false
  
  // Wait for queue to drain (up to 30 seconds)
  deadline, _ := ctx.Deadline()
  timeout := time.Until(deadline)
  
  ticker := time.NewTicker(100 * time.Millisecond)
  defer ticker.Stop()
  
  for {
    if len(s.checkQueue) == 0 {
      break
    }
    select {
    case <-ctx.Done():
      break
    case <-ticker.C:
      continue
    }
  }
  
  // Stop workers
  s.pool.Stop()
  
  // Close channels
  close(s.checkQueue)
  
  return nil
}
```

---

## Metrics & Observability

### Key Metrics

```
registry_health_checks_total (counter)
  └─ labels: check_type, status

registry_health_checks_passed_total (counter)

registry_health_checks_failed_total (counter)

registry_health_check_duration_seconds (histogram)
  └─ buckets: 0.01, 0.05, 0.1, 0.5, 1, 5, 10

registry_health_check_queue_depth (gauge)
  └─ current queue length

registry_health_worker_busy (gauge)
  └─ number of workers currently executing

registry_health_state_changes_total (counter)
  └─ labels: old_state, new_state

registry_incidents_open (gauge)

registry_incidents_created_total (counter)

registry_incidents_resolved_total (counter)
```

### Logging

```go
// Health check execution
slog.Info("health_check_completed",
  slog.String("check_id", result.HealthCheckID),
  slog.String("instance_id", result.InstanceID),
  slog.Bool("success", result.Success),
  slog.Int("latency_ms", result.LatencyMs),
)

// State transitions
slog.Info("health_state_changed",
  slog.String("instance_id", state.InstanceID),
  slog.String("old_state", oldState.String()),
  slog.String("new_state", newState.String()),
)

// Incidents
slog.Warn("incident_opened",
  slog.String("incident_id", incident.ID),
  slog.String("instance_id", incident.InstanceID),
  slog.String("reason", incident.Reason),
)
```

---

## Failure Scenarios & Recovery

### Queue Overflow
- **Symptom**: Check queue reaches capacity
- **Detection**: Enqueue returns error
- **Handling**: Log dropped check, increment metric
- **Recovery**: Automatic; check retried at next interval
- **User Impact**: Minimal (checks still run, just delayed)

### Worker Panic
- **Symptom**: Worker goroutine panics
- **Detection**: Not detected (worker exits)
- **Handling**: Implement worker supervisor to restart
- **Recovery**: Restart worker pool
- **User Impact**: Reduced concurrency temporarily

### Network Timeouts
- **Symptom**: Check hangs indefinitely
- **Detection**: Context deadline exceeded
- **Handling**: Mark as failed, clean up goroutine
- **Recovery**: Automatic at next interval
- **User Impact**: Failed check, state may transition

### Database Failure
- **Symptom**: Persisting results fails
- **Detection**: Write error in transaction
- **Handling**: Log error, exponential backoff retry
- **Recovery**: Results cached briefly, retry on reconnect
- **User Impact**: Health state may lag; alerts delayed

---

## Configuration

```yaml
health:
  # Worker pool configuration
  workerCount: 20                    # Number of concurrent workers
  
  # Scheduling
  schedulerInterval: 100ms           # How often to check for due checks
  
  # Defaults for new health checks
  defaultInterval: 10s               # Default check interval
  defaultTimeout: 5s                 # Default check timeout
  defaultFailureThreshold: 3         # Failures before unhealthy
  defaultSuccessThreshold: 2         # Successes before healthy
  
  # Queue sizing
  checkQueueCapacity: 1000           # Max pending checks
  resultQueueCapacity: 1000          # Max pending results
  
  # Timeouts
  httpCheckTimeout: 5s               # HTTP-specific timeout
  tcpCheckTimeout: 5s                # TCP-specific timeout
  grpcCheckTimeout: 5s               # gRPC-specific timeout
  
  # Retention
  resultRetentionHours: 24           # Keep detailed results 24h
  
  # Caching (optional)
  enableCheckCache: true             # Cache loaded checks
  checkCacheTTL: 30s                 # Refresh checks every 30s
```

---

## Testing Strategy

### Unit Tests
- Health state transition logic
- Timeout handling
- Result evaluation

### Integration Tests
- End-to-end check execution
- Database persistence
- Event publishing
- Incident creation/resolution

### Concurrency Tests
- Worker pool stress
- Queue overflow handling
- Graceful shutdown
- Race detection enabled

### Load Tests
- 1000 health checks
- 20 concurrent workers
- Target: < 50ms p99 latency
- Target: < 5% failures on healthy instances

This health scheduler design provides the foundation for reliable, scalable health monitoring.
