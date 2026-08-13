# SQLite Database Schema

## Schema Design Philosophy

1. **Normalized Structure**: Use proper relationships, not JSON blobs
2. **Flexible Metadata**: JSON columns for non-queryable attributes
3. **Denormalized for Performance**: Include frequently-joined fields directly
4. **Soft Delete**: Preserve historical data via `deleted_at` timestamps
5. **Audit Trail**: Every write timestamped and immutable
6. **Indexes for Queries**: Optimize common query patterns
7. **No Application-Level Transactions**: Rely on database transactions

### SQLite Configuration
```sql
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
```

---

## Tables

### 1. environments

```sql
CREATE TABLE environments (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tier TEXT,
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE INDEX idx_environments_key ON environments(key) WHERE deleted_at IS NULL;
CREATE INDEX idx_environments_enabled ON environments(enabled) WHERE deleted_at IS NULL;
```

**Rationale**:
- `id`: UUID primary key, immutable
- `key`: Unique identifier, used in lookups
- `tags`: JSON for flexible classification
- `deleted_at`: Soft delete, queries filter `WHERE deleted_at IS NULL`
- Indexes on frequently-queried columns

---

### 2. services

```sql
CREATE TABLE services (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    description TEXT,
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE INDEX idx_services_name ON services(name) WHERE deleted_at IS NULL;
CREATE INDEX idx_services_display_name ON services(display_name) WHERE deleted_at IS NULL;
```

**Rationale**:
- Service is independent of environment (deployed to many)
- Name immutable and globally unique
- Tags and metadata flexible

---

### 3. service_deployments

```sql
CREATE TABLE service_deployments (
    id TEXT PRIMARY KEY,
    service_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    health_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    alerts_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    alert_cooldown_minutes INTEGER NOT NULL DEFAULT 10,
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,

    FOREIGN KEY (service_id) REFERENCES services(id),
    FOREIGN KEY (environment_id) REFERENCES environments(id),
    UNIQUE(service_id, environment_id)
);

CREATE INDEX idx_service_deployments_service ON service_deployments(service_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_service_deployments_environment ON service_deployments(environment_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_service_deployments_health ON service_deployments(health_enabled, deleted_at);
```

**Rationale**:
- Links service to environment
- Unique constraint prevents duplicates
- Indexes for common queries (by service, by environment)
- Denormalized health_enabled for filtering

---

### 4. service_instances

```sql
CREATE TABLE service_instances (
    id TEXT PRIMARY KEY,
    deployment_id TEXT NOT NULL,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    port INTEGER,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_seen_at TEXT,
    deleted_at TEXT,

    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    UNIQUE(deployment_id, name)
);

CREATE INDEX idx_instances_deployment ON service_instances(deployment_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_instances_enabled ON service_instances(enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_instances_address ON service_instances(address) WHERE deleted_at IS NULL;
CREATE INDEX idx_instances_last_seen ON service_instances(last_seen_at) WHERE deleted_at IS NULL;
```

**Rationale**:
- Instance name unique within deployment
- Addresses may not be unique (shared IP pools)
- Indexes for lookup patterns
- `last_seen_at` for operational insights

---

### 5. endpoints

```sql
CREATE TABLE endpoints (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL,  -- http, https, grpc, tcp, udp
    port INTEGER NOT NULL,
    path TEXT,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,

    FOREIGN KEY (instance_id) REFERENCES service_instances(id) ON DELETE CASCADE,
    UNIQUE(instance_id, name)
);

CREATE INDEX idx_endpoints_instance ON endpoints(instance_id);
CREATE INDEX idx_endpoints_protocol ON endpoints(protocol);
```

**Rationale**:
- Multiple endpoints per instance (e.g., HTTP, gRPC, metrics)
- Name unique within instance
- Protocol indexed for filtering

---

### 6. health_checks

```sql
CREATE TABLE health_checks (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    endpoint_id TEXT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,  -- http, https, tcp, grpc, heartbeat
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    interval_seconds INTEGER NOT NULL,  -- e.g., 10
    timeout_seconds INTEGER NOT NULL,   -- e.g., 5
    failures_before_unhealthy INTEGER NOT NULL DEFAULT 3,
    successes_before_healthy INTEGER NOT NULL DEFAULT 2,
    description TEXT,
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,

    FOREIGN KEY (instance_id) REFERENCES service_instances(id) ON DELETE CASCADE,
    FOREIGN KEY (endpoint_id) REFERENCES endpoints(id) ON DELETE SET NULL,
    UNIQUE(instance_id, name)
);

CREATE INDEX idx_health_checks_instance ON health_checks(instance_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_health_checks_enabled ON health_checks(enabled) WHERE deleted_at IS NULL;
CREATE INDEX idx_health_checks_type ON health_checks(type) WHERE deleted_at IS NULL;
```

**Rationale**:
- Endpoint optional (instance-level or endpoint-level checks)
- Type-specific metadata stored as JSON
- Indexes for scheduler queries

---

### 7. health_states

```sql
CREATE TABLE health_states (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL UNIQUE,
    current_state TEXT NOT NULL,  -- UNKNOWN, HEALTHY, DEGRADED, UNHEALTHY, DISABLED
    consecutive_successes INTEGER NOT NULL DEFAULT 0,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_transition_time TEXT NOT NULL,
    last_check_time TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    updated_at TEXT NOT NULL,

    FOREIGN KEY (instance_id) REFERENCES service_instances(id) ON DELETE CASCADE
);

CREATE INDEX idx_health_states_state ON health_states(current_state);
CREATE INDEX idx_health_states_updated ON health_states(updated_at);
```

**Rationale**:
- One state per instance (1:1 relationship)
- Consecutive counters for threshold logic
- Indexed on state for dashboard queries

---

### 8. health_results

```sql
CREATE TABLE health_results (
    id TEXT PRIMARY KEY,
    health_check_id TEXT NOT NULL,
    instance_id TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    latency_ms INTEGER,
    status_code INTEGER,
    error_type TEXT,
    error_message TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    expires_at TEXT,

    FOREIGN KEY (health_check_id) REFERENCES health_checks(id),
    FOREIGN KEY (instance_id) REFERENCES service_instances(id)
);

CREATE INDEX idx_health_results_check ON health_results(health_check_id, timestamp DESC);
CREATE INDEX idx_health_results_instance ON health_results(instance_id, timestamp DESC);
CREATE INDEX idx_health_results_expires ON health_results(expires_at) WHERE expires_at IS NOT NULL;
```

**Rationale**:
- Denormalized `instance_id` for direct queries
- Timestamp desc for recent results first
- Expires_at for auto-cleanup (use trigger or scheduled job)
- NOT soft-deleted; completely removed after expiration

---

### 9. incidents

```sql
CREATE TABLE incidents (
    id TEXT PRIMARY KEY,
    instance_id TEXT NOT NULL,
    deployment_id TEXT NOT NULL,
    environment_id TEXT NOT NULL,
    service_id TEXT NOT NULL,
    state TEXT NOT NULL,  -- OPEN, RESOLVED
    opened_at TEXT NOT NULL,
    resolved_at TEXT,
    duration_seconds INTEGER,
    reason TEXT NOT NULL,
    impact_summary TEXT,
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,

    FOREIGN KEY (instance_id) REFERENCES service_instances(id),
    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    FOREIGN KEY (environment_id) REFERENCES environments(id),
    FOREIGN KEY (service_id) REFERENCES services(id)
);

CREATE INDEX idx_incidents_state ON incidents(state) WHERE state = 'OPEN';
CREATE INDEX idx_incidents_instance ON incidents(instance_id);
CREATE INDEX idx_incidents_deployment ON incidents(deployment_id);
CREATE INDEX idx_incidents_environment ON incidents(environment_id);
CREATE INDEX idx_incidents_service ON incidents(service_id);
CREATE INDEX idx_incidents_opened ON incidents(opened_at DESC);
```

**Rationale**:
- Denormalized IDs for direct lookups
- OPEN index for dashboards
- Retained indefinitely
- No soft delete

---

### 10. alert_policies

```sql
CREATE TABLE alert_policies (
    id TEXT PRIMARY KEY,
    deployment_id TEXT,
    environment_id TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    notify_on TEXT NOT NULL DEFAULT '["unhealthy","recovered"]',  -- JSON array
    cooldown_minutes INTEGER NOT NULL DEFAULT 10,
    send_recovery_notification BOOLEAN NOT NULL DEFAULT TRUE,
    filters TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT,

    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    FOREIGN KEY (environment_id) REFERENCES environments(id)
);

CREATE INDEX idx_alert_policies_deployment ON alert_policies(deployment_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_alert_policies_environment ON alert_policies(environment_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_alert_policies_enabled ON alert_policies(enabled) WHERE deleted_at IS NULL;
```

**Rationale**:
- Optional deployment or environment scope
- notify_on stored as JSON array for flexibility
- Separate table for policy-channel relationships

---

### 11. alert_policy_channels

```sql
CREATE TABLE alert_policy_channels (
    policy_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,

    FOREIGN KEY (policy_id) REFERENCES alert_policies(id) ON DELETE CASCADE,
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id),
    PRIMARY KEY (policy_id, channel_id)
);
```

**Rationale**:
- Many-to-many relationship
- Cascade delete on policy

---

### 12. notification_channels

```sql
CREATE TABLE notification_channels (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,  -- webhook, email
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    description TEXT,
    configuration TEXT NOT NULL,  -- JSON
    retry_policy TEXT NOT NULL DEFAULT '{}',  -- JSON
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    deleted_at TEXT
);

CREATE INDEX idx_notification_channels_type ON notification_channels(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_notification_channels_enabled ON notification_channels(enabled) WHERE deleted_at IS NULL;
```

**Rationale**:
- Configuration is JSON (type-specific fields)
- Retry policy stored with channel

---

### 13. alert_attempts

```sql
CREATE TABLE alert_attempts (
    id TEXT PRIMARY KEY,
    incident_id TEXT NOT NULL,
    policy_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    notification_type TEXT NOT NULL,  -- unhealthy, recovered, etc
    attempted_at TEXT NOT NULL,
    success BOOLEAN NOT NULL,
    status_code INTEGER,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    next_retry_at TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON

    FOREIGN KEY (incident_id) REFERENCES incidents(id),
    FOREIGN KEY (policy_id) REFERENCES alert_policies(id),
    FOREIGN KEY (channel_id) REFERENCES notification_channels(id)
);

CREATE INDEX idx_alert_attempts_incident ON alert_attempts(incident_id);
CREATE INDEX idx_alert_attempts_channel ON alert_attempts(channel_id);
CREATE INDEX idx_alert_attempts_success ON alert_attempts(success);
CREATE INDEX idx_alert_attempts_retry ON alert_attempts(next_retry_at) WHERE next_retry_at IS NOT NULL;
```

**Rationale**:
- Per-channel alert history
- Retry tracking
- Not soft-deleted; kept for audit

---

### 14. events

```sql
CREATE TABLE events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,  -- ServiceCreated, IncidentOpened, etc
    timestamp TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    environment_id TEXT,
    service_id TEXT,
    deployment_id TEXT,
    instance_id TEXT,
    actor TEXT NOT NULL,  -- username or "system"
    actor_id TEXT,
    message TEXT NOT NULL,
    changes TEXT,  -- JSON
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON
    expires_at TEXT,

    FOREIGN KEY (environment_id) REFERENCES environments(id),
    FOREIGN KEY (service_id) REFERENCES services(id),
    FOREIGN KEY (deployment_id) REFERENCES service_deployments(id),
    FOREIGN KEY (instance_id) REFERENCES service_instances(id)
);

CREATE INDEX idx_events_timestamp ON events(timestamp DESC);
CREATE INDEX idx_events_type ON events(type);
CREATE INDEX idx_events_resource ON events(resource_type, resource_id);
CREATE INDEX idx_events_environment ON events(environment_id);
CREATE INDEX idx_events_service ON events(service_id);
CREATE INDEX idx_events_expires ON events(expires_at) WHERE expires_at IS NOT NULL;
```

**Rationale**:
- Denormalized for direct filtering
- Timestamp indexed descending (recent first)
- Expires for auto-cleanup
- Immutable records

---

### 15. audit_logs

```sql
CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    timestamp TEXT NOT NULL,
    actor TEXT NOT NULL,
    actor_id TEXT,
    action TEXT NOT NULL,  -- Create, Update, Delete, etc
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    environment_id TEXT,
    changes TEXT,  -- JSON
    change_description TEXT,
    ip TEXT,
    user_agent TEXT,
    status TEXT NOT NULL,  -- success, failure
    error_message TEXT,
    metadata TEXT NOT NULL DEFAULT '{}',  -- JSON

    FOREIGN KEY (environment_id) REFERENCES environments(id)
);

CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
```

**Rationale**:
- Retained indefinitely (compliance)
- Timestamp indexed for timeline queries
- No expires_at field
- Detailed change tracking

---

### 16. users

```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,  -- Administrator, Operator, Viewer, Automation
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tags TEXT NOT NULL DEFAULT '{}',  -- JSON
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    last_login_at TEXT,
    deleted_at TEXT
);

CREATE INDEX idx_users_username ON users(username) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_role ON users(role) WHERE deleted_at IS NULL;
```

**Rationale**:
- Username and email unique
- Password hashed (never stored plain)
- Soft delete preserves history

---

### 17. api_tokens

```sql
CREATE TABLE api_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    scopes TEXT NOT NULL,  -- JSON array: ["read", "write", "admin"]
    environment_ids TEXT,  -- JSON array (optional scoping)
    expires_at TEXT,
    last_used_at TEXT,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TEXT NOT NULL,
    created_by TEXT NOT NULL,

    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_api_tokens_user ON api_tokens(user_id);
CREATE INDEX idx_api_tokens_enabled ON api_tokens(enabled);
CREATE INDEX idx_api_tokens_expires ON api_tokens(expires_at) WHERE expires_at IS NOT NULL;
```

**Rationale**:
- Token hashed (not stored plain)
- Scopes flexible via JSON
- Optional environment scoping
- Last used tracking

---

## Indexes Summary

### Query Performance Optimization

| Query Pattern | Index |
|---|---|
| Services by environment | idx_service_deployments_environment |
| Instances by deployment | idx_instances_deployment |
| Unhealthy instances | idx_health_states_state + idx_health_states_updated |
| Open incidents | idx_incidents_state (WHERE state = 'OPEN') |
| Recent events | idx_events_timestamp DESC |
| Enabled health checks | idx_health_checks_enabled |
| Health results for check | idx_health_results_check + timestamp DESC |
| Alert attempts to retry | idx_alert_attempts_retry |
| Audit trail | idx_audit_logs_timestamp DESC |

---

## Performance Targets

Using these indexes:
- List services by environment: **< 100ms** (1000s)
- List instances by deployment: **< 50ms** (100s)
- Query unhealthy instances: **< 100ms** (1000s)
- List open incidents: **< 50ms** (100s)
- Query recent events: **< 100ms** (10000s)

---

## Migrations Strategy

Use numbered migration files:
```
migrations/
├── 001_create_environments.sql
├── 002_create_services.sql
├── 003_create_service_deployments.sql
├── 004_create_service_instances.sql
├── 005_create_endpoints.sql
├── 006_create_health_checks.sql
├── 007_create_health_states.sql
├── 008_create_health_results.sql
├── 009_create_incidents.sql
├── 010_create_alert_policies.sql
├── 011_create_notification_channels.sql
├── 012_create_alert_attempts.sql
├── 013_create_events.sql
├── 014_create_audit_logs.sql
├── 015_create_users.sql
└── 016_create_api_tokens.sql
```

**Migration Tool**: Use `golang-migrate` or similar for idempotent migrations.

---

## Data Cleanup

### Health Results Expiration
```sql
DELETE FROM health_results WHERE expires_at < datetime('now')
-- Run every hour via scheduler or trigger
```

### Events Expiration
```sql
DELETE FROM events WHERE expires_at < datetime('now')
-- Run daily
```

### Soft Delete Cleanup (optional)
```sql
DELETE FROM services WHERE deleted_at < datetime('now', '-90 days')
-- Run weekly, after verifying no foreign key dependencies
```

---

## Backup Strategy

### SQLite Backup Best Practices
1. Enable WAL mode (`PRAGMA journal_mode = WAL`)
2. Backup both `.db` and `.db-wal` files
3. Use `.backup` command for hot backups
4. Implement daily automated backups to cloud storage

### Restore Procedure
1. Stop registry service
2. Restore `.db` and `.db-wal` files
3. Verify integrity: `PRAGMA integrity_check`
4. Restart registry service

---

## Growth Planning

### Estimated Capacity (SQLite)
- 1,000 services: ~50 KB
- 10,000 instances: ~500 KB
- 20,000 endpoints: ~200 KB
- 20,000 health checks: ~400 KB
- 1 month of health results (100K/day): ~200 MB
- 1,000 incidents: ~50 KB
- 1 month of events (50K/day): ~100 MB
- **Total for MVP: ~1 GB**

### When to Migrate to PostgreSQL
- Monthly data volume > 10 GB
- Concurrent connections > 20
- Complex analytical queries needed
- Multiple replicas required
- The schema supports this via repository pattern

---

## Transaction Model

### Read Transactions
```sql
BEGIN TRANSACTION;
  SELECT ... FROM health_checks WHERE enabled = true;
COMMIT;
```

### Write Transactions
```sql
BEGIN TRANSACTION;
  UPDATE health_states SET current_state = 'UNHEALTHY' WHERE instance_id = ?;
  INSERT INTO incidents (...) VALUES (...);
  INSERT INTO events (...) VALUES (...);
COMMIT;
```

### Health Check Execution
```
DO NOT execute inside database transaction:

Scheduler
  └─ Load checks (within transaction)
  └─ Queue check
    └─ Execute network request (no transaction)
    └─ Evaluate result (no transaction)
    └─ BEGIN TRANSACTION
      └─ Update health state
      └─ Create incident if needed
      └─ Insert event
    └─ COMMIT
```

---

## WAL and Performance

With `PRAGMA journal_mode = WAL`:
- Readers don't block writers
- Writers don't block readers
- Better concurrency for health scheduler
- Slightly larger disk footprint
- Production-appropriate for SQLite

---

## Soft Delete Strategy

Columns with `deleted_at`:
- environments
- services
- service_deployments
- service_instances
- health_checks
- alert_policies
- notification_channels
- users

Tables WITHOUT soft delete:
- endpoints (cascaded via foreign key)
- health_results (expired and deleted)
- incidents (retained indefinitely)
- events (expired and deleted)
- audit_logs (retained indefinitely)
- alert_attempts (retained indefinitely)
- api_tokens (cascade on user delete)

**Rationale**: Audit/operational data kept forever; configuration data kept for 90 days.

This schema provides a solid foundation for the Service Registry MVP with room to grow.
