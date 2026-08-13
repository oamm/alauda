# Architectural Risks & Mitigations

This document identifies potential risks in the Service Registry architecture and proposes mitigation strategies.

---

## Critical Risks

### 1. Health Check Scheduler Overload

**Risk Level**: HIGH

**Description**: 
- If health checks accumulate faster than workers can execute them, the queue grows unbounded
- Under sustained load, queue depth metric should alert operators
- Dropped checks lead to stale health data

**Symptoms**:
- Check queue depth increasing over time
- Checks executing hours after scheduled
- Incidents created late or missed
- UI showing stale health information

**Mitigation**:
1. **Bounded Queue**: Maximum queue size (1000 items)
2. **Monitoring**: Alert if queue depth > 500 for 5 minutes
3. **Backpressure**: Don't queue new checks if queue full (skip and log)
4. **Worker Scaling**: Configuration allows increasing worker count
5. **Check Prioritization**: Future enhancement - prioritize critical services
6. **Graceful Degradation**: If overloaded, checks run less frequently (not skipped)

**Recovery**:
- Increase worker count: `health.workerCount: 50`
- Reduce check frequency (longer intervals)
- Disable non-critical checks temporarily
- Scale out with multiple registry instances (future)

**Monitoring**:
```
ALERT HealthCheckQueueDepthHigh
  for: 5m
  if: registry_health_check_queue_depth > 500
  severity: warning
```

---

### 2. Database Lock Contention

**Risk Level**: HIGH

**Description**:
- If health check results persist while scheduler loop runs, locks could block new checks
- Long-running transactions hold locks on health state tables
- Alert engine transactions compete with state transitions
- Could cause database busy timeouts

**Symptoms**:
- Frequent "database is locked" errors
- Health check execution latency increasing
- Intermittent connectivity timeouts
- Database busy timeout exceptions

**Mitigation**:
1. **Separate Transactions**: Health execution separate from persistence
2. **Quick Transactions**: Minimize transaction duration (< 10ms target)
3. **WAL Mode**: Write-ahead logging reduces lock conflicts
4. **Busy Timeout**: `PRAGMA busy_timeout = 5000` (5 second retry)
5. **Connection Pooling**: Single connection per thread
6. **Read Replicas**: Future enhancement - read from replica for queries

**Architecture Enforcement**:
```go
// ✗ BAD: Network I/O inside transaction
txn.Begin()
  result := executeCheck()  // Network call - can hang!
  txn.InsertResult()
txn.Commit()

// ✓ GOOD: Network I/O outside transaction
result := executeCheck()      // No transaction lock
txn.Begin()
  txn.InsertResult(result)
txn.Commit()
```

**Monitoring**:
```
ALERT DatabaseBusyErrors
  for: 1m
  if: rate(registry_db_busy_errors_total[5m]) > 0.1
  severity: critical
```

---

### 3. Unbounded Goroutine Growth

**Risk Level**: HIGH

**Description**:
- Each alert notification attempt could spawn goroutines
- If alert handlers aren't properly bounded, goroutines leak
- Event subscribers could create goroutines per event
- Memory usage grows without limit

**Symptoms**:
- Memory usage increasing over time
- `go tool pprof` shows increasing goroutine count
- Application becomes slower, eventually crashes
- OOM killer terminates process

**Mitigation**:
1. **Fixed Worker Pool**: All I/O uses bounded pool (20 workers default)
2. **Goroutine Limits**: Use semaphore for alert sends
3. **Event Subscribers**: Non-blocking publish (no goroutines)
4. **Circuit Breaker**: Stop retry if notification channel fails
5. **Graceful Shutdown**: Drain all pending work before exit
6. **Monitoring**: Track goroutine count

**Code Patterns**:
```go
// ✗ BAD: Unbounded goroutine per alert
for _, channel := range policy.Channels {
  go sendNotification(channel)  // Leak!
}

// ✓ GOOD: Bounded worker pool
alertWorkerPool.Send(alertTask)
```

**Monitoring**:
```
ALERT GoroutineCountHigh
  for: 5m
  if: go_goroutines > 1000
  severity: warning
```

---

### 4. Event Loss on Crash

**Risk Level**: MEDIUM

**Description**:
- Domain events published before persistence could be lost
- If registry crashes after publishing event to subscribers but before persisting
- Audit trail gaps
- Alerts sent but not logged
- UI updates but not persisted

**Symptoms**:
- Events missing from audit log
- Alert attempts not recorded
- Inconsistent state between system components

**Mitigation**:
1. **Persist First**: Always persist event before publishing
2. **Transaction**: Event persisted in same transaction as state change
3. **Idempotent Subscribers**: Alert engine de-duplicates by incident
4. **Event Sourcing**: Future enhancement - store all events

**Implementation**:
```go
// ✓ GOOD: Persist event in transaction
txn.Begin()
  txn.InsertEvent(event)        // Persisted
  incident := txn.GetIncident() // Data written
txn.Commit()
// → Now safe to publish event
eventBus.Publish(event)
```

---

### 5. Alert Notification Failures

**Risk Level**: MEDIUM

**Description**:
- Webhook endpoints could be down or unreachable
- Email SMTP server could be misconfigured
- Network issues prevent notifications
- Operator misses critical incidents

**Symptoms**:
- Service goes unhealthy
- No webhook/email received
- Alert attempt marked as failed in database
- Operator unaware of incident

**Mitigation**:
1. **Retry Logic**: Exponential backoff (5s, 10s, 20s, 60s, 300s)
2. **Test Notification**: UI feature to test channels before enabling
3. **Fallback Channels**: Configure multiple channels per policy
4. **Webhook Validation**: Verify URL is reachable on creation
5. **Email Validation**: Test SMTP connection on configuration
6. **Dead Letter Queue**: Retry failed notifications up to 24 hours
7. **Monitoring**: Alert on failed notifications

**Configuration**:
```yaml
alerts:
  channels:
    - type: webhook
      url: https://hooks.slack.com/...
      retry:
        maxAttempts: 5
        initialBackoffSeconds: 5
        maxBackoffSeconds: 300
```

**Monitoring**:
```
ALERT AlertSendFailureRate
  for: 10m
  if: rate(registry_alerts_failed_total[5m]) > 0.01
  severity: warning
```

---

### 6. SQLite Scale Limits

**Risk Level**: MEDIUM (for MVP scale)

**Description**:
- SQLite designed for ~1GB single database files
- Not optimized for 1000s of concurrent health writes
- Contention on write locks at scale
- May need PostgreSQL sooner than expected

**Symptoms**:
- Database file > 500MB
- "Database is busy" errors under load
- Slow query performance
- Migration/vacuum taking > 1 minute

**Mitigation**:
1. **Data Retention**: Aggressive cleanup of old health results (24h default)
2. **Partitioning**: Health results by date (future)
3. **Archive Strategy**: Move old data to separate database
4. **PostgreSQL Ready**: Repository pattern allows backend swap
5. **Scale Monitoring**: Track database size and alert at 500MB

**Growth Plan**:
- **MVP** (0-1 month): SQLite fine
- **100 services** (1-3 months): Still SQLite
- **1000 services** (3-6 months): Consider PostgreSQL migration
- **10000 services** (6+ months): PostgreSQL required

**Monitoring**:
```
ALERT SQLiteSizeHigh
  if: file_size(registry.db) > 500MB
  severity: warning
  action: Migrate to PostgreSQL
```

---

## Operational Risks

### 7. Cascading Failure During Deployment

**Risk Level**: MEDIUM

**Description**:
- New version deployed with health check bugs
- All checks start failing
- Incidents created for all services
- Alert flood to operators
- Registry becomes noisy, alerts ignored

**Symptoms**:
- Sudden spike in incident creation
- All services showing unhealthy
- Operator inbox flooded with alerts
- False positives make alerts untrustworthy

**Mitigation**:
1. **Gradual Rollout**: Deploy to staging first
2. **Canary Health**: Test with dummy health checks
3. **Circuit Breaker**: Pause health checks if failure rate > 50%
4. **Metrics Alert**: Alert on incident creation rate spike
5. **Rollback Plan**: Easy rollback to previous version
6. **Check Disable**: Manual option to disable all checks temporarily

**Monitoring**:
```
ALERT IncidentCreationRateSpike
  for: 2m
  if: rate(registry_incidents_created_total[5m]) > 1000
  severity: critical
  action: Disable health checks, investigate
```

---

### 8. Stale Health Data

**Risk Level**: MEDIUM

**Description**:
- Scheduler fails silently (logs not monitored)
- Checks stop executing
- Health state becomes outdated
- Operator makes decisions based on stale data
- Real incident undetected

**Symptoms**:
- Health state unchanged for hours
- Last check time old
- Scheduler goroutine crashed (not detected)
- No errors in logs

**Mitigation**:
1. **Scheduler Heartbeat**: Regular log message ("processed N checks")
2. **Max Age Threshold**: Alert if any check not run in 2 × interval
3. **Liveness Check**: ReadyProbe queries last check times
4. **Health State Staleness**: Mark state as UNKNOWN if checks not run > 1 hour
5. **Graceful Degradation**: Show "data may be stale" UI warning

**Monitoring**:
```
registry_health_check_max_age_seconds
ALERT HealthChecksStaledForTooLong
  for: 5m
  if: max(registry_health_check_max_age_seconds) > 3600
  severity: critical
```

---

## Configuration Risks

### 9. Misconfigured Health Thresholds

**Risk Level**: MEDIUM

**Description**:
- Operator sets `failuresBeforeUnhealthy: 1`
- Temporary network glitch causes false incident
- Alert fatigue, operators ignore real issues
- Or `failuresBeforeUnhealthy: 100` - misses real failures

**Symptoms**:
- Too many incidents for transient failures
- Too few incidents, missing real issues
- Operators tweaking thresholds constantly

**Mitigation**:
1. **Sensible Defaults**: `failuresBeforeUnhealthy: 3`, `successes: 2`
2. **Validation**: Reject invalid thresholds (must be >= 1)
3. **UI Guidance**: Show recommendation based on check type
4. **Configuration Examples**: Provide best-practice templates
5. **Dry Run**: Show estimated frequency before saving

**UI Validation**:
```typescript
// Validate before submit
if (failuresBeforeUnhealthy < 1) {
  error("Must be at least 1")
}
if (successes < 1) {
  error("Must be at least 1")
}
// Warn if unusual
if (failuresBeforeUnhealthy > 10) {
  warn("This is very high - transient issues may not be detected")
}
```

---

### 10. Security: Weak Credentials

**Risk Level**: MEDIUM

**Description**:
- Admin password is weak (123456)
- No password expiration
- Default credentials left in examples
- API tokens exposed in logs

**Symptoms**:
- Unauthorized access to registry
- Malicious service registrations
- Incidents artificially created/resolved
- Audit trail shows unauthorized changes

**Mitigation**:
1. **Password Requirements**: Minimum 12 characters, complexity
2. **No Default Credentials**: Must set on first run
3. **Secure Examples**: Use placeholders, not real values
4. **Token Masking**: Mask tokens in UI and logs (show last 4 chars)
5. **Token Rotation**: Prompt users to rotate tokens periodically
6. **Expiration**: Default 90-day token expiration

**Implementation**:
```go
// Validate password on creation
if len(password) < 12 {
  return err("Password must be at least 12 characters")
}
if !hasComplexity(password) {
  return err("Password must contain uppercase, lowercase, digit, symbol")
}

// Hash securely
hash := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
```

---

## Integration Risks

### 11. Registry Dependency on External Systems

**Risk Level**: MEDIUM

**Description**:
- If webhook server (Slack, PagerDuty) is down, alerts can't send
- If SMTP server is down, email notifications fail
- Registry continues working but operators don't know
- Incidents accumulate without notification

**Symptoms**:
- Webhook requests timing out
- Email delivery failing
- Alert attempts marked as failed
- No incidents get resolved because nobody knows

**Mitigation**:
1. **Graceful Degradation**: Registry works even if channels down
2. **Fallback Channels**: Multiple notification methods
3. **Alert on Alert Failure**: Monitor notification channel health
4. **Dead Letter Queue**: Retain failed notifications for retry
5. **Manual Override**: UI to send notifications manually
6. **Monitoring Integration**: Registry health separate from external systems

**Architecture**:
```
Incident Created
  ↓
Alert Engine Processes
  ├─ Try Webhook (fails? → queue for retry)
  └─ Try Email (fails? → queue for retry)
  
→ Incident still created and visible in UI
→ Retry job processes failed attempts every 5 minutes
```

---

## Future Expansion Risks

### 12. API Versioning Lock-in

**Risk Level**: LOW

**Description**:
- If API design is inflexible, upgrading becomes painful
- Protobuf schema changes break clients
- No deprecation strategy
- Difficult to add features without breaking compatibility

**Mitigation**:
1. **Semantic Versioning**: Follow v1, v2 naming
2. **Proto Backward Compatibility**: New fields optional, never delete
3. **Deprecation Warnings**: Plan for v2 early
4. **Client SDKs**: Provide typed clients for multiple languages
5. **Gateway Layer**: Future API Gateway can translate between versions
6. **Documentation**: Clear API stability guarantees

**Proto Best Practice**:
```protobuf
message Service {
  string id = 1;
  string name = 2;
  string display_name = 3;
  
  // Good: Adding optional field is safe
  int32 version_count = 4;
  
  // Bad: Deleting field breaks old clients
  // reserved 5;  // was: string old_field = 5;
}
```

---

### 13. Kubernetes Integration Complexity

**Risk Level**: MEDIUM (for future)

**Description**:
- If later adding Kubernetes sync, discovery complexity grows
- Multiple sources of truth (Kubernetes + Registry)
- Sync conflicts and race conditions
- Hard to debug which system is source of truth

**Mitigation** (Design Now, Implement Later):
1. **Clear Boundaries**: Registry is source of truth initially
2. **One-Way Sync**: Read from Kubernetes, write to Registry (not bidirectional)
3. **Versioning**: Track which Kubernetes version registered each instance
4. **Dry Run**: Support "show what would sync" before committing
5. **Reconciliation**: Clear rules for conflict resolution

---

## Summary Risk Matrix

| Risk | Level | Impact | Mitigation Priority |
|------|-------|--------|-------------------|
| Scheduler Overload | HIGH | Health data stale | 1 |
| Database Locks | HIGH | Timeouts, data loss | 1 |
| Unbounded Goroutines | HIGH | Crash | 1 |
| Event Loss | MEDIUM | Audit gaps | 2 |
| Alert Failures | MEDIUM | Operators unaware | 2 |
| SQLite Scale | MEDIUM | Performance | 2 |
| Cascading Failures | MEDIUM | Alert fatigue | 2 |
| Stale Data | MEDIUM | Wrong decisions | 2 |
| Misconfigured Thresholds | MEDIUM | False positives | 3 |
| Weak Credentials | MEDIUM | Unauthorized access | 2 |
| External Dependencies | MEDIUM | Notification failure | 3 |
| API Versioning | LOW | Future migration pain | 4 |
| Kubernetes Sync | MEDIUM | Race conditions (future) | 4 |

---

## Monitoring Strategy

### Metrics to Watch
```
# Health Scheduler
registry_health_check_queue_depth         (gauge)
registry_health_checks_total              (counter)
registry_health_checks_failed_total       (counter)
registry_health_check_duration_seconds    (histogram)
registry_health_worker_busy               (gauge)

# Database
registry_db_query_duration_seconds        (histogram)
registry_db_busy_errors_total             (counter)
registry_db_transaction_duration_seconds  (histogram)

# Incidents & Alerts
registry_incidents_open                   (gauge)
registry_incidents_created_total          (counter)
registry_alerts_sent_total                (counter)
registry_alerts_failed_total              (counter)

# System
go_goroutines                             (gauge)
process_resident_memory_bytes             (gauge)
```

### Alert Rules
```yaml
groups:
  - name: service-registry
    rules:
      - alert: HealthCheckQueueDepthHigh
        expr: registry_health_check_queue_depth > 500
        for: 5m
        severity: warning

      - alert: DatabaseBusy
        expr: rate(registry_db_busy_errors_total[5m]) > 0.1
        for: 1m
        severity: critical

      - alert: GoroutineLeakSuspected
        expr: rate(go_goroutines[10m]) > 50
        for: 10m
        severity: warning

      - alert: AlertNotificationFailure
        expr: rate(registry_alerts_failed_total[5m]) > 0.01
        for: 10m
        severity: warning

      - alert: HealthChecksStaledForTooLong
        expr: max(registry_health_check_max_age_seconds) > 3600
        for: 5m
        severity: critical
```

This risk analysis provides a foundation for building a resilient, production-grade service registry.
