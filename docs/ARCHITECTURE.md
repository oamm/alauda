# Service Registry Architecture

## 1. Overview

The Service Registry is a lightweight, self-hosted service registry and health-management platform for cataloging service endpoints, continuously validating their availability, tracking health history, and notifying operators when services fail or recover.

### Core Responsibilities

```
Service Registry
│
├── Service Catalog (CRUD for services, deployments, instances, endpoints)
├── Health Monitoring (scheduled health checks with worker pool)
├── Health State Management (deterministic state transitions)
├── Incident Tracking (lifecycle from open to resolved)
├── Alert Engine (webhook, email notifications with cooldown)
├── Administration UI (modern dashboard and management tools)
└── Registry API (service lookup, audit trail)
```

### Key Non-Responsibilities

This system is **NOT**:

- A service mesh
- A reverse proxy or API gateway
- A load balancer
- A traffic router or interceptor
- A DNS or network manager
- A distributed configuration system
- A secret manager
- A Kubernetes networking component
- A general monitoring or metrics platform

**Core principle**: The registry provides information about where services run. It never proxies, routes, or redirects traffic to those endpoints.

---

## 2. High-Level Architecture

```
                    ┌───────────────────────┐
                    │     React UI          │
                    │   (static files)      │
                    └───────────┬───────────┘
                                │
                                │ API (ConnectRPC)
                                ▼
┌────────────────┐      ┌───────────────────────┐
│  registryctl   │ ───► │     Registry API      │
│  (CLI tool)    │      │   (ConnectRPC)        │
└────────────────┘      └───────────┬───────────┘
                                    │
                       ┌────────────┼────────────┬──────────────┐
                       │            │            │              │
                       ▼            ▼            ▼              ▼
                    Catalog      Health      Incidents        Alerts
                   Service        Check        &            Notification
                  Management     Scheduler    Events         Channels
                       │            │            │              │
                       └────────────┼────────────┼──────────────┘
                                    │
                       ┌────────────▼────────────┐
                       │    Persistence Layer    │
                       │     (Repositories)      │
                       └────────────┬────────────┘
                                    │
                                    ▼
                              SQLite Database
```

### Module Boundaries

#### Domain Layer (`internal/domain`)

- Pure business logic entities
- Domain services (interfaces, no implementation)
- Event definitions
- Error types
- No external dependencies

#### Application Layer (`internal/application`)

- Use case orchestration
- Command/query handling
- Transaction management
- Domain event publishing

#### Health Subsystem (`internal/health`)

- Health check types
- Health scheduler
- Worker pool
- State transition engine
- Health result evaluation

#### Storage Layer (`internal/storage`)

- Repository interfaces
- SQLite implementation
- Migrations
- Query builders

#### API Layer (`internal/api`)

- ConnectRPC handlers
- gRPC/HTTP server
- Request/response mapping
- Authentication/authorization

#### Server (`internal/server`)

- HTTP server setup
- Middleware
- Graceful shutdown
- Health endpoints

---

## 3. Data Flow Patterns

### Service Registration

```
CLI/API Request
    ↓
Validation (command handler)
    ↓
Domain Service (apply business rules)
    ↓
Repository (persist)
    ↓
Event Published
    ↓
Audit Logged
    ↓
Response
```

### Health Check Execution

```
Scheduler (time-based)
    ↓
Load enabled checks
    ↓
Queue ready checks
    ↓
Worker Pool (bounded concurrency)
    ↓
Execute Network Request
    ↓
Evaluate Result
    ↓
Determine Health State Transition
    ↓
Update State Repository
    ↓
Create Incident if needed
    ↓
Trigger Alert if needed
    ↓
Persist Health Result & Events
```

### Live UI Updates

```
Health Event Created
    ↓
Event Persisted
    ↓
Event Stream Emitted (SSE or ConnectRPC streaming)
    ↓
React Component Updates
    ↓
Dashboard reflects new status
```

---

## 4. Key Architectural Decisions

### 1. Environment as First-Class Concept

- **Decision**: Environment is NOT a tag; it's a primary domain entity
- **Reasoning**: Services behave differently per environment; this is fundamental
- **Implementation**: Every service deployment is explicitly linked to an environment
- **UI Impact**: Global environment selector affects all views

### 2. Service → Deployment → Instance → Endpoint Hierarchy

- **Service**: Logical application (e.g., `lynx-authentication`)
- **Deployment**: Service in a specific environment (e.g., `lynx-authentication/prod`)
- **Instance**: Individual running copy (e.g., `auth-prod-01` at `10.20.1.15`)
- **Endpoint**: Network location exposed by instance (e.g., `https://10.20.1.15:8080/health`)

### 3. Health Checks Execute Outside Transactions

- **Decision**: Network I/O never happens within database transactions
- **Reasoning**: Prevent database locks during slow network operations
- **Implementation**:
  - Check scheduled independently
  - Result evaluated
  - State transition persisted in separate transaction

### 4. Bounded Worker Pool for Health Checks

- **Decision**: Use fixed-size worker pool, not one goroutine per check
- **Reasoning**: Predictable resource usage; prevents unlimited goroutine explosion
- **Configuration**: Default 20 workers (configurable)
- **Scalability**: Supports 20,000+ checks efficiently

### 5. Health State Machine

- **States**: `UNKNOWN`, `HEALTHY`, `DEGRADED`, `UNHEALTHY`, `DISABLED`
- **Transitions**: Deterministic based on explicit rules
- **Failure Thresholds**: Configurable `failuresBeforeUnhealthy`, `successesBeforeHealthy`
- **Incident Creation**: Only when transitioning into unhealthy state

### 6. Incidents vs. Events

- **Incidents**: Discrete operational events (opened/resolved)
- **Events**: Broader operational timeline (created, updated, deleted, etc.)
- **Relationship**: Multiple events may occur within one incident

### 7. Simple Alert Mechanism

- **Initial Channels**: Webhook + Email
- **No Rules Engine**: Webhook enables unlimited integrations (Slack, Teams, PagerDuty)
- **Cooldown**: Prevent alert fatigue
- **Policy Model**: Per-environment, per-service configuration

### 8. Repositories as Domain-Specific Interfaces

- **Pattern**: `ServiceRepository`, `HealthRepository`, `IncidentRepository`
- **Reasoning**: Clear intent, type-safe, testable
- **No Generic**: Avoid `Repository[T]` unless concrete benefit
- **Implementation**: PostgreSQL and SQLite adapters behind the storage layer

### 9. OpenTelemetry for Observability

- **Scope**: Instrument registry itself only
- **Metrics**: Service/instance counts, health check latency, incident counts
- **Logs**: Structured logs via `slog`
- **Out of Scope**: Full APM or metrics storage (bring-your-own collector)

### 10. Monorepo Structure

- **Backend**: Pure Go server
- **Frontend**: React + TypeScript, compiled to static files
- **Embedded**: Frontend embedded in Go binary via `go:embed`
- **Deployment**: Single binary; no Node.js required in production

---

## 5. Concurrency Model

### Health Scheduler

```
                 ┌──────────────┐
                 │  Scheduler   │
                 │  (main loop) │
                 └──────┬───────┘
                        │ (every 100ms)
                        ▼
                 ┌──────────────┐
                 │ Load enabled │
                 │   checks     │
                 └──────┬───────┘
                        │
                        ▼
                 ┌──────────────┐
                 │ Determine    │
                 │  due checks  │
                 └──────┬───────┘
                        │
                        ▼
                 ┌──────────────────┐
                 │  Queue checks    │
                 │ (buffered chan)  │
                 └──────┬───────────┘
                        │
         ┌──────────────┼──────────────┐
         │              │              │
         ▼              ▼              ▼
      Worker 1      Worker 2  ...  Worker N
         │              │              │
         └──────────────┼──────────────┘
                        │
         ┌──────────────┴──────────────┐
         │                             │
         ▼                             ▼
    Result Ch               State Transition
    (buffered)              Event Published
         │                         │
         └─────────────┬───────────┘
                       │
                  Persist Transaction
                  ├─ Update health state
                  ├─ Create/resolve incident
                  └─ Publish events
```

### Critical Requirements

1. **Bounded Concurrency**: Fixed worker pool size (default 20)
2. **No Nested Transactions**: Health check result processing in separate transaction
3. **Graceful Shutdown**: Drain queue, wait for in-flight checks
4. **Context Propagation**: Timeout enforcement per check
5. **Resource Limits**: No unbounded channel buffers

---

## 6. Database Design Philosophy

### Principles

1. **Relational Structure**: Use tables/relationships, not JSON blobs
2. **Flexible Metadata**: JSON for non-queryable attributes
3. **Normalized for Performance**: Appropriate indexes on query paths
4. **Audit Trail**: Every mutation timestamped
5. **Soft Delete**: Maintain historical records
6. **Foreign Keys Enabled**: Database enforces referential integrity

### Query Performance Targets

- List services by environment: **< 100ms** (10K+ services)
- List instances by deployment: **< 50ms** (10K+ instances)
- List unhealthy instances: **< 100ms**
- List open incidents: **< 50ms**
- Recent events: **< 100ms**

---

## 7. API Design Principles

### Service Boundaries

- `EnvironmentService`: Manage environments
- `CatalogService`: Manage services (CRUD)
- `DeploymentService`: Manage deployments (service + environment)
- `InstanceService`: Register/manage instances
- `HealthService`: Configure health checks, query health state
- `IncidentService`: Query incidents
- `AlertService`: Configure alerts and notification channels
- `EventService`: Stream events and query history

### Core Lookup API

Public discovery is GET /api/v1/discovery/{serviceKey}?environment={key}; resolve-one adds /resolve. Responses provide named Instances/Endpoints and canonical addresses, with no Deployment relationship to reconstruct. RegistryService discovery RPCs were removed. See [SERVICE_DISCOVERY.md](SERVICE_DISCOVERY.md) for health and selection semantics.

### Versioning

- Package: `registry.v1`
- API versioning: URL-based when evolving

---

## 8. Security Model

### Authentication (MVP)

- Local administrator account (username/password)
- API tokens for automation
- Token scopes (read, write, admin)

### Authorization (MVP)

- Simple RBAC roles:
  - `Administrator`: Full access
  - `Operator`: Service operations, no configuration
  - `Viewer`: Read-only access
  - `Automation`: Token-based access for CI/CD

### Audit Trail

- Every write operation logged
- Actor, action, resource, timestamp, IP
- RBAC rules changes tracked
- Token access logged

### Future Enhancements

- OIDC integration
- Environment-level permissions
- Service-level permissions
- Webhook signature verification

---

## 9. Health Check Behavior

### Check Types

```
HTTP/HTTPS
├── GET/POST/etc
├── Expected status codes
├── Response timeout
└── Optional response body validation

TCP
├── Connection success/failure
└── Timeout

gRPC
├── Service name
├── Health method
└── Timeout

Heartbeat/TTL
├── Last seen timestamp
└── TTL threshold
```

### Health State Computation

```
Rule 1: All checks configured for instance are healthy
        → Instance is HEALTHY

Rule 2: Some checks failing, some passing (for degraded checks)
        → Instance is DEGRADED

Rule 3: Required check repeatedly failing (threshold met)
        → Instance is UNHEALTHY

Rule 4: No checks configured
        → Instance is UNKNOWN

Rule 5: Explicitly disabled
        → Instance is DISABLED
```

### Threshold Logic

```
State: HEALTHY
  consecutive_failures = 0

Check fails:
  consecutive_failures++

  if consecutive_failures >= failuresBeforeUnhealthy:
    State → UNHEALTHY
    consecutive_failures = 0
    consecutive_successes = 0
    Incident created


State: UNHEALTHY
  consecutive_successes = 0

Check succeeds:
  consecutive_successes++

  if consecutive_successes >= successesBeforeHealthy:
    State → HEALTHY
    consecutive_failures = 0
    consecutive_successes = 0
    Incident resolved
```

### Critical Execution Rules

- Never execute during database transaction
- Always have timeout (default 5s)
- Always have retry interval (default 10s)
- Track consecutive successes/failures separately per check
- Publish state transition event before response

---

## 10. Alert Engine

### Lifecycle

```
Health transitions to UNHEALTHY
    ↓
Alert policy matches (environment, service, etc)
    ↓
Check cooldown (avoid duplicate alerts)
    ↓
Send notification (webhook, email)
    ↓
Log alert attempt
    ↓
Retry on failure (with backoff)
    │
    └─ After recovery:
        └─ Send recovery notification
```

### Notification Channels

```
Webhook
├── HTTP POST
├── Configurable URL
├── Signature (future)
└── Retry policy

Email
├── SMTP configuration
├── Template-based
└── HTML body
```

### Cooldown

- Default: 10 minutes
- Per alert policy
- Prevents notification spam
- Does not suppress recovery notifications

---

## 11. Event Stream Architecture

### Event Types

```
Administrative Events
├── ServiceCreated
├── ServiceUpdated
├── ServiceDeleted
├── DeploymentCreated
├── InstanceRegistered
├── InstanceRemoved
├── EndpointCreated
└── ...

Operational Events
├── HealthChanged
├── IncidentOpened
├── IncidentResolved
├── AlertSent
└── AlertFailed
```

### Event Propagation

```
Domain Event Created
    ↓
Repository saves to database
    ↓
Event bus publishes in-memory
    ↓
Subscribers notified:
  ├── Audit logger
  ├── Alert engine
  ├── Event stream (for UI)
  └── Metrics collector
```

### UI Live Updates

- SSE (Server-Sent Events) or ConnectRPC streaming
- Event subscriptions per filter (environment, service, etc)
- Automatic reconnect with backoff
- Recent event backlog on connect

---

## 12. Retention Strategy

### Detailed Health Results

- Storage: Last 24 hours
- Purge: Older results deleted
- Reasoning: Recent data useful for diagnosis; old data not

### Health Transitions

- Storage: 90 days
- Purge: Archive or delete after 90 days
- Reasoning: Useful for incident pattern analysis

### Incidents

- Storage: Retained indefinitely
- Reasoning: Historical records for audit and SLA calculations

### Events

- Storage: 30 days by default
- Purge: Configurable retention
- Reasoning: Operational timeline; old events less useful

### Audit Logs

- Storage: Retained indefinitely
- Reasoning: Compliance and audit trail

---

## 13. Observability

### Metrics (OpenTelemetry)

```
registry_services_total (counter)
registry_instances_total (counter)
registry_endpoints_total (counter)
registry_health_checks_total (counter)
registry_health_checks_passed_total (counter)
registry_health_checks_failed_total (counter)
registry_health_check_duration_seconds (histogram)
registry_healthy_instances (gauge)
registry_unhealthy_instances (gauge)
registry_degraded_instances (gauge)
registry_incidents_open (gauge)
registry_incidents_total (counter)
registry_incidents_resolved_total (counter)
registry_alerts_sent_total (counter)
registry_alerts_failed_total (counter)
```

### Logging

- Structured logs via Go `slog`
- Log levels: DEBUG, INFO, WARN, ERROR
- All public API requests logged
- Health check failures logged at WARN level
- State transitions logged at INFO level

### Registry Health Endpoints

- `/healthz`: Liveness (always true if responding)
- `/readyz`: Readiness (database connected, workers running)
- `/version`: Version information
- Separate from service health (never mark registry unhealthy based on monitored services)

---

## 14. Future Extension Points

### Storage Backend

- PostgreSQL support (repositories accept `database/sql` interface)
- Migration framework independent of SQLite

### Authentication

- OIDC provider integration
- SSO support

### Alert Providers

- Slack integration
- Microsoft Teams integration
- PagerDuty integration
- Custom webhook templates

### Service Features

- Automatic registration (SDKs, libraries)
- Service dependency visualization
- Maintenance windows
- Health check templates (library of common checks)
- SLA/SLO reporting

### Kubernetes Integration

- Sync services from Kubernetes resources
- Watch cluster changes
- Export back to Kubernetes

### High Availability

- Multiple registry instances
- Shared SQLite (via network filesystem) or PostgreSQL
- Distributed event bus (for multi-instance alerting)

---

## 15. Configuration

### Deployment Modes

```
Development:
  registry server --dev
  Hot reload UI
  Verbose logging

Production:
  registry server
  Single binary
  Optimized logging
```

### Configuration Sources (precedence)

1. Defaults (code)
2. Configuration file (`registry.yaml`)
3. Environment variables
4. CLI flags

### Key Configuration

```yaml
server:
  address: 0.0.0.0
  port: 9700
  tlsCert: null
  tlsKey: null

storage:
  path: ./data/registry.db
  busyTimeout: 5s

health:
  workerCount: 20
  checkInterval: 10s
  defaultTimeout: 5s
  defaultFailureThreshold: 3
  defaultSuccessThreshold: 2

telemetry:
  enabled: true
  otlpEndpoint: http://localhost:4317

alerts:
  enabled: true
  cooldown: 10m
```

---

## 16. Deployment

### Docker Image

```dockerfile
# Stage 1: Frontend build
FROM node:20 AS frontend-builder

# Stage 2: Backend build
FROM golang:1.21 AS backend-builder

# Stage 3: Runtime
FROM debian:bookworm-slim
COPY --from=backend-builder /app/registry /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/registry"]
```

### Single Binary Composition

```
Binary
├── Embedded React SPA (static files)
├── ConnectRPC API
├── CLI subcommands (in binary)
└── SQLite database (file-based)
```

### Development Setup

```bash
make dev
# Opens http://localhost:9700
# Watches for changes
# Reloads frontend
```

---

## 17. Architectural Risks & Mitigations

| Risk                             | Impact | Mitigation                                                |
| -------------------------------- | ------ | --------------------------------------------------------- |
| Health checks block UI updates   | High   | Separate scheduler goroutine, non-blocking persistence    |
| Unbounded goroutine growth       | High   | Fixed worker pool, bounded channels                       |
| Database lock contention         | High   | Separate health check transaction, read replicas (future) |
| Alert spam on flapping           | Medium | Failure threshold, cooldown timer                         |
| Lost events on crash             | Low    | Event persistence before publication                      |
| SQLite scale limits              | Low    | PostgreSQL migration path designed upfront                |
| Missed health checks on overload | Low    | Persistent queue, status page for dropped checks          |

---

## 18. Success Criteria for MVP

### Phase 0 Complete ✓

- [ ] Go server running
- [ ] React UI running
- [ ] SQLite schema initialized
- [ ] ConnectRPC API working
- [ ] Basic CRUD working

### Phase 1 Complete ✓

- [ ] Services, deployments, instances manageable
- [ ] Environments fully functional
- [ ] CLI works for basic operations
- [ ] UI shows service catalog

### Phase 2 Complete ✓

- [ ] Health checks configured and executing
- [ ] HTTP, TCP, gRPC checks working
- [ ] State transitions deterministic
- [ ] Scheduler stable under load

### Phase 3 Complete ✓

- [ ] Incidents created and resolved
- [ ] Availability calculated
- [ ] Incident history available

### Phase 4 Complete ✓

- [ ] Webhook alerts working
- [ ] Email alerts working
- [ ] Cooldown preventing spam
- [ ] Recovery notifications sent

### Phase 5 Complete ✓

- [ ] Dashboard displays metrics
- [ ] Health view shows all checks
- [ ] Live updates working
- [ ] Service detail complete

### Phase 6 Complete ✓

- [ ] Authentication working
- [ ] RBAC enforced
- [ ] Audit log complete
- [ ] Production-ready deployment

---

## 19. Document References

- See `DOMAIN.md` for entity definitions
- See `DATABASE.md` for SQLite schema
- See `HEALTH.md` for health check details
- See `ALERTS.md` for alert system design
- See `API.md` for Protobuf contracts
- See `MVP.md` for implementation phases
