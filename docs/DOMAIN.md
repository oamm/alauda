# Domain Model

## Core Entities Overview

```
Environment
├── Services (many)
    └── ServiceDeployment (per environment)
        ├── ServiceInstances (many)
        │   ├── Endpoints (many)
        │   └── HealthChecks (many)
        │       └── HealthResults (many - historical)
        │
        ├── HealthState (aggregate of instance states)
        └── IncidentPolicies (alert configuration)

Incidents (aggregate)
├── Triggered by health state transitions
├── Resolved when health recovers
└── Generates alerts

AlertPolicies (configuration)
├── Per environment
├── Per service
└── Per deployment

NotificationChannels (webhook, email)
└── Persisted attempts

Events (operational timeline)
├── ServiceLifecycle
├── DeploymentLifecycle
├── InstanceLifecycle
├── HealthTransitions
├── IncidentLifecycle
└── AlertEvents

AuditLog (administrative changes)
└── Per entity write
```

---

## 1. Environment

**Purpose**: First-class domain concept representing deployment boundaries.

### Properties
```
id: UUID                    # Immutable identifier
key: string                 # Unique string identifier (e.g., "prod")
name: string                # Display name (e.g., "Production")
description: string         # Free-form description
enabled: bool               # Can register new services in this env
tier: string                # Optional: production, staging, development
tags: map<string, string>  # Flexible classification
createdAt: timestamp
updatedAt: timestamp
deletedAt: timestamp?       # Soft delete
```

### Constraints
- `key` must be unique across all environments
- `key` immutable after creation
- `key` alphanumeric + underscores only (validate)
- Cannot delete if services exist (or soft-delete only)
- At least one environment must exist

### Example
```
Environment {
  id: "env-prod-001"
  key: "production"
  name: "Production"
  description: "Production workloads"
  enabled: true
  tier: "production"
  tags: {
    region: "us-east-1",
    criticality: "high"
  }
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-01T08:00:00Z
}
```

---

## 2. Service

**Purpose**: Logical application definition, independent of environment.

### Properties
```
id: UUID                    # Immutable identifier
name: string                # Unique service name (e.g., "lynx-authentication")
displayName: string         # Human-readable name
description: string         # Service purpose and scope
tags: map<string, string>  # Classification (team, type, runtime, etc)
metadata: JSON             # Flexible attributes
createdAt: timestamp
updatedAt: timestamp
deletedAt: timestamp?       # Soft delete
```

### Constraints
- `name` unique across all services
- `name` immutable after creation
- `name` must match pattern: `^[a-z][a-z0-9-]*[a-z0-9]$` (alphanumeric + hyphens)
- Must have at least a `displayName` or `description`

### Tag Examples
```
team=platform              # Team ownership
type=backend              # Service category
runtime=dotnet            # Runtime/technology
criticality=high          # Operational importance
owner=platform-team       # Team contact
```

### Example
```
Service {
  id: "svc-auth-001"
  name: "lynx-authentication"
  displayName: "Lynx Authentication"
  description: "Central authentication and authorization service"
  tags: {
    team: "platform",
    type: "backend",
    runtime: "dotnet",
    criticality: "high"
  }
  metadata: {
    repository: "git@github.com:company/lynx-authentication",
    documentation: "https://internal-docs/services/authentication",
    contact: "platform-team@company.com"
  }
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-12T14:30:00Z
}
```

---

## 3. ServiceDeployment

**Purpose**: Links a Service to an Environment with environment-specific configuration.

### Properties
```
id: UUID                          # Immutable
serviceId: UUID                   # Foreign key
environmentId: UUID               # Foreign key
healthEnabled: bool              # Enable health checks for this deployment
alertsEnabled: bool              # Enable alerts for this deployment
alertCooldownMinutes: int        # How long to wait before resending alert
tags: map<string, string>        # Deployment-specific tags (e.g., version)
metadata: JSON                   # Deployment-specific metadata
createdAt: timestamp
updatedAt: timestamp
deletedAt: timestamp?
```

### Constraints
- Combination of (serviceId, environmentId) must be unique
- Cannot have duplicate deployments
- At least one environment must have the deployment
- If service deleted, cascade-delete deployments

### Tag Examples
```
version=2.8.1            # Service version in this env
region=us-east-1         # Regional deployment
cluster=prod-cluster-01  # Cluster assignment
```

### Example
```
ServiceDeployment {
  id: "deploy-auth-prod-001"
  serviceId: "svc-auth-001"
  environmentId: "env-prod-001"
  healthEnabled: true
  alertsEnabled: true
  alertCooldownMinutes: 10
  tags: {
    version: "2.8.1",
    region: "us-east-1",
    cluster: "prod-cluster-01"
  }
  metadata: {
    scaleFactor: 3,
    loadBalancer: "nlb-prod-auth"
  }
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-12T14:30:00Z
}
```

---

## 4. ServiceInstance

**Purpose**: Individual running copy of a service in an environment.

### Properties
```
id: UUID                    # Immutable
deploymentId: UUID         # Foreign key (service + environment)
name: string               # e.g., "auth-prod-01"
address: string            # IPv4, IPv6, hostname, FQDN
port: int32?               # Optional default port
description: string        # Purpose of this instance
enabled: bool              # Can execute health checks
tags: map<string, string> # Instance-specific tags
metadata: JSON            # Flexible attributes
createdAt: timestamp
updatedAt: timestamp
lastSeenAt: timestamp?     # Last successful health check
deletedAt: timestamp?
```

### Constraints
- `name` unique within a deployment (multiple envs can have same name)
- `address` can be IPv4, IPv6, hostname (no protocol)
- Cannot modify immutable fields
- Soft delete only (keeps history)

### Address Examples
```
10.20.1.15                    # IPv4
2001:db8::1                   # IPv6
api-node-01.internal          # DNS name
auth-prod-01.example.com      # FQDN
```

### Example
```
ServiceInstance {
  id: "inst-auth-prod-01"
  deploymentId: "deploy-auth-prod-001"
  name: "auth-prod-01"
  address: "10.20.1.15"
  port: null
  description: "Primary authentication instance in us-east-1a"
  enabled: true
  tags: {
    zone: "us-east-1a",
    version: "2.8.1",
    capacity: "high"
  }
  metadata: {
    nodeId: "node-prod-001",
    containerRuntime: "containerd"
  }
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-12T14:30:00Z
  lastSeenAt: 2026-08-12T14:35:42Z
}
```

---

## 5. Endpoint

**Purpose**: Network location exposed by a service instance.

### Properties
```
id: UUID                    # Immutable
instanceId: UUID           # Foreign key
name: string               # e.g., "http", "grpc", "metrics"
protocol: string           # http, https, grpc, tcp, udp
port: int32               # Network port (0-65535)
path: string?             # Optional path (e.g., /health)
description: string       # Purpose of this endpoint
enabled: bool             # Can be checked by health checks
tags: map<string, string> # Endpoint classification
metadata: JSON            # Flexible attributes
createdAt: timestamp
updatedAt: timestamp
```

### Constraints
- Combination of (instanceId, name) must be unique
- `protocol` must be known type or configurable enum
- `port` must be valid (0-65535), typically > 1024
- `path` only valid for HTTP/HTTPS

### Protocol Values
```
http     - TCP/IP HTTP
https    - TCP/IP HTTPS
grpc     - gRPC over HTTP/2
tcp      - Raw TCP connection
udp      - UDP protocol
```

### Example
```
Endpoint {
  id: "ep-auth-prod-01-http"
  instanceId: "inst-auth-prod-01"
  name: "http"
  protocol: "https"
  port: 8080
  path: "/health"
  description: "Main authentication HTTP API"
  enabled: true
  tags: {
    public: "true",
    healthCheck: "true"
  }
  metadata: {
    scheme: "https",
    timeout: "5s"
  }
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-12T14:30:00Z
}
```

---

## 6. HealthCheck

**Purpose**: Configuration for checking an instance or endpoint's availability.

### Properties
```
id: UUID                          # Immutable
instanceId: UUID                 # Foreign key
endpointId: UUID?                # Foreign key (optional, if endpoint-specific)
name: string                     # e.g., "readiness", "liveness"
type: string                     # http, https, tcp, grpc, heartbeat
enabled: bool                    # Run this check
interval: duration               # How often to check (e.g., 10s)
timeout: duration                # Maximum check time (e.g., 5s)
failuresBeforeUnhealthy: int    # Consecutive failures needed (default 3)
successesBeforeHealthy: int     # Consecutive successes needed (default 2)
description: string              # Purpose
tags: map<string, string>       # Classification
metadata: JSON                  # Check-specific configuration

# HTTP-specific
httpMethod: string?              # GET, POST, etc
httpPath: string?                # /health, /ready
expectedStatusCodes: int[]?      # [200, 201]
requestHeaders: map?             # Custom headers
requestBody: string?             # Body for POST

# TCP-specific
tcpConnectTimeout: duration?    # Connection timeout

# gRPC-specific
grpcService: string?             # Service name
grpcMethod: string?              # Method name

# Heartbeat-specific
heartbeatTtl: duration?          # TTL for heartbeat

createdAt: timestamp
updatedAt: timestamp
```

### Constraints
- Must have instanceId
- Either endpointId or instance-level
- `interval` >= 5s (minimum)
- `timeout` < `interval`
- `timeout` default 5s
- Thresholds >= 1
- `type` must be valid

### Example: HTTP Health Check
```
HealthCheck {
  id: "hc-auth-prod-01-http"
  instanceId: "inst-auth-prod-01"
  endpointId: "ep-auth-prod-01-http"
  name: "readiness"
  type: "https"
  enabled: true
  interval: 10s
  timeout: 3s
  failuresBeforeUnhealthy: 3
  successesBeforeHealthy: 2
  description: "Readiness probe via HTTPS"
  tags: {
    critical: "true"
  }
  metadata: {
    acceptedStatusCodes: [200, 201],
    followRedirects: false
  }
  httpMethod: "GET"
  httpPath: "/health/ready"
  expectedStatusCodes: [200]
  requestHeaders: {
    "X-Request-ID": "health-check"
  }
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-12T14:30:00Z
}
```

### Example: TCP Health Check
```
HealthCheck {
  id: "hc-auth-prod-01-tcp"
  instanceId: "inst-auth-prod-01"
  endpointId: null
  name: "tcp-connectivity"
  type: "tcp"
  enabled: true
  interval: 30s
  timeout: 5s
  failuresBeforeUnhealthy: 5
  successesBeforeHealthy: 2
  description: "TCP connection test"
  tcpConnectTimeout: 3s
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-12T14:30:00Z
}
```

---

## 7. HealthState

**Purpose**: Current health status and transition tracking for an instance.

### Properties
```
id: UUID                              # Immutable
instanceId: UUID                     # Foreign key
currentState: enum                   # UNKNOWN, HEALTHY, DEGRADED, UNHEALTHY, DISABLED
consecutiveSuccesses: int            # For UNHEALTHY->HEALTHY transition
consecutiveFailures: int             # For HEALTHY->UNHEALTHY transition
lastTransitionTime: timestamp        # When state last changed
lastCheckTime: timestamp?            # Last time any check ran
metadata: JSON                       # State-specific metadata
updatedAt: timestamp
```

### State Enum
```
UNKNOWN       # No checks configured or never checked
HEALTHY       # All required checks passing
DEGRADED      # Some optional checks failing, required checks passing
UNHEALTHY     # Required checks failing
DISABLED      # Instance disabled or checks disabled
```

### State Transition Rules
```
Entry:      UNKNOWN

            ↓ (no checks configured)

UNKNOWN ----→ (checks configured and run)

            ↓ (all healthy)

HEALTHY ←→ UNHEALTHY
  ↓
(failuresBeforeUnhealthy consecutive failures)
↓
UNHEALTHY ←→ HEALTHY
  ↓
(successesBeforeHealthy consecutive successes)

Optional: HEALTHY ↔ DEGRADED
          (if optional checks failing)

Manual:     Any → DISABLED
```

### Example
```
HealthState {
  id: "hs-auth-prod-01"
  instanceId: "inst-auth-prod-01"
  currentState: HEALTHY
  consecutiveSuccesses: 0
  consecutiveFailures: 0
  lastTransitionTime: 2026-08-12T14:30:00Z
  lastCheckTime: 2026-08-12T14:35:42Z
  metadata: {
    transitionReason: "3 consecutive successes after recovery"
  }
  updatedAt: 2026-08-12T14:35:42Z
}
```

---

## 8. HealthResult

**Purpose**: Individual health check execution record.

### Properties
```
id: UUID                    # Immutable
healthCheckId: UUID        # Foreign key
instanceId: UUID           # Denormalized for queries
timestamp: timestamp       # When check ran
success: bool              # Check passed/failed
latencyMs: int            # Milliseconds to complete
statusCode: int?          # HTTP status code
errorType: string?        # Error classification
errorMessage: string?     # Error details
metadata: JSON            # Additional result data
expiresAt: timestamp?     # For automatic cleanup
```

### Error Types
```
CONNECTION_REFUSED
CONNECTION_TIMEOUT
READ_TIMEOUT
HTTP_ERROR
HTTP_INVALID_STATUS
RESPONSE_PARSE_ERROR
GRPC_ERROR
TCP_CONNECT_FAILED
OTHER
```

### Data Retention
- Store: Last 24 hours of detailed results
- Purge: Older results deleted (configurable)
- Purpose: Recent diagnosis only

### Example
```
HealthResult {
  id: "hr-20260812-143542-001"
  healthCheckId: "hc-auth-prod-01-http"
  instanceId: "inst-auth-prod-01"
  timestamp: 2026-08-12T14:35:42Z
  success: true
  latencyMs: 12
  statusCode: 200
  errorType: null
  errorMessage: null
  metadata: {
    responseTime: "12ms",
    bodySize: 1024
  }
  expiresAt: 2026-08-13T14:35:42Z
}
```

---

## 9. Incident

**Purpose**: Operational event representing a service becoming unhealthy or recovering.

### Properties
```
id: UUID                      # Immutable
instanceId: UUID             # Foreign key
deploymentId: UUID           # Denormalized for queries
environmentId: UUID          # Denormalized for queries
serviceId: UUID              # Denormalized for queries
state: enum                  # OPEN, RESOLVED
openedAt: timestamp         # When incident created
resolvedAt: timestamp?      # When health recovered
durationSeconds: int?       # Total outage duration
reason: string              # Why it became unhealthy
impactSummary: string       # Human readable summary
tags: map<string, string> # Classification
metadata: JSON             # Additional context
createdAt: timestamp
```

### State Enum
```
OPEN       # Incident active, service unhealthy
RESOLVED   # Service recovered, incident closed
```

### Example
```
Incident {
  id: "inc-20260812-143542-001"
  instanceId: "inst-auth-prod-01"
  deploymentId: "deploy-auth-prod-001"
  environmentId: "env-prod-001"
  serviceId: "svc-auth-001"
  state: RESOLVED
  openedAt: 2026-08-12T13:42:12Z
  resolvedAt: 2026-08-12T13:47:54Z
  durationSeconds: 342
  reason: "HTTP health check failed 3 times consecutively"
  impactSummary: "auth-prod-01 became unhealthy and recovered after 5 minutes 42 seconds"
  tags: {
    severity: "high",
    impact: "production"
  }
  metadata: {
    failureMode: "timeout",
    failingCheck: "readiness",
    recoveryMethod: "automatic"
  }
  createdAt: 2026-08-12T13:42:12Z
}
```

---

## 10. AlertPolicy

**Purpose**: Configuration for when/how to alert on incidents.

### Properties
```
id: UUID                          # Immutable
deploymentId: UUID?              # Deployment-specific (or null for global)
environmentId: UUID?             # Environment-specific (or null for global)
enabled: bool                    # Policy active
notifyOn: string[]              # [unhealthy, degraded, recovered]
cooldownMinutes: int            # Prevent duplicate alerts
sendRecoveryNotification: bool   # Alert on recovery
notificationChannelIds: UUID[]  # Which channels to use
filters: JSON                    # Additional filtering
createdAt: timestamp
updatedAt: timestamp
```

### Notification Types
```
unhealthy     # When check starts failing
degraded      # When degraded state reached
recovered     # When health restored
created       # Optional: new instance registered
deleted       # Optional: instance removed
```

### Hierarchy
- Global default (any)
- Environment-specific (if set)
- Deployment-specific (if set)
- Per service (future)

### Example
```
AlertPolicy {
  id: "ap-prod-auth-001"
  deploymentId: "deploy-auth-prod-001"
  environmentId: null
  enabled: true
  notifyOn: ["unhealthy", "recovered"]
  cooldownMinutes: 10
  sendRecoveryNotification: true
  notificationChannelIds: [
    "nc-webhook-pagerduty",
    "nc-email-oncall"
  ]
  filters: {
    minSeverity: "high",
    excludeCheckTypes: ["tcp"]
  }
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-12T14:30:00Z
}
```

---

## 11. NotificationChannel

**Purpose**: Destination for alert notifications.

### Properties
```
id: UUID                    # Immutable
type: string                # webhook, email
name: string                # e.g., "PagerDuty Webhook"
enabled: bool
description: string
configuration: JSON         # Type-specific config
retryPolicy: JSON          # Backoff strategy
tags: map<string, string>
createdAt: timestamp
updatedAt: timestamp
```

### Channel Types

#### Webhook
```
configuration: {
  url: "https://hooks.slack.com/...",
  method: "POST",
  headers: {...},
  signingSecret?: "..."
}
```

#### Email
```
configuration: {
  smtpServer: "smtp.company.com",
  smtpPort: 587,
  fromAddress: "registry@company.com",
  toAddresses: ["oncall@company.com"],
  templateName: "incident"
}
```

### Example
```
NotificationChannel {
  id: "nc-webhook-pagerduty"
  type: "webhook"
  name: "PagerDuty Webhook"
  enabled: true
  description: "Send incidents to PagerDuty"
  configuration: {
    url: "https://events.pagerduty.com/v2/enqueue",
    method: "POST"
  }
  retryPolicy: {
    maxRetries: 3,
    initialBackoffSeconds: 5,
    maxBackoffSeconds: 60
  }
  tags: {
    provider: "pagerduty",
    priority: "high"
  }
  createdAt: 2026-08-01T08:00:00Z
  updatedAt: 2026-08-12T14:30:00Z
}
```

---

## 12. Event

**Purpose**: Immutable record of significant operations for audit trail and UI updates.

### Properties
```
id: UUID
type: string                # ServiceCreated, IncidentOpened, etc
timestamp: timestamp        # When event occurred
resourceType: string        # Service, Instance, Incident, etc
resourceId: UUID           # ID of affected resource
environmentId: UUID?       # Denormalized
serviceId: UUID?           # Denormalized
deploymentId: UUID?        # Denormalized
instanceId: UUID?          # Denormalized
actor: string              # User or system that triggered event
actorId: UUID?             # User ID if applicable
message: string            # Human-readable summary
changes: JSON?             # Before/after for updates
metadata: JSON             # Event-specific data
createdAt: timestamp       # Immutable
expiresAt: timestamp?      # For automatic cleanup
```

### Event Types
```
# Service lifecycle
ServiceCreated
ServiceUpdated
ServiceDeleted
ServiceDisabled
ServiceEnabled

# Deployment lifecycle
DeploymentCreated
DeploymentUpdated
DeploymentDeleted

# Instance lifecycle
InstanceRegistered
InstanceUpdated
InstanceRemoved
InstanceDisabled
InstanceEnabled

# Endpoint lifecycle
EndpointCreated
EndpointUpdated
EndpointDeleted

# Health events
HealthCheckExecuted
HealthStateChanged
HealthCheckFailed

# Incident events
IncidentOpened
IncidentResolved
IncidentAcknowledged (future)

# Alert events
AlertSent
AlertFailed
AlertRetried

# Admin events
ConfigurationChanged
UserCreated
TokenCreated
TokenRevoked
```

### Data Retention
- Store: 30 days (configurable)
- Purge: Older events deleted
- Purpose: Operational timeline

### Example
```
Event {
  id: "evt-20260812-143542-001"
  type: "IncidentOpened"
  timestamp: 2026-08-12T13:42:12Z
  resourceType: "Incident"
  resourceId: "inc-20260812-143542-001"
  environmentId: "env-prod-001"
  serviceId: "svc-auth-001"
  deploymentId: "deploy-auth-prod-001"
  instanceId: "inst-auth-prod-01"
  actor: "system"
  actorId: null
  message: "auth-prod-01 became unhealthy (3 failed health checks)"
  changes: null
  metadata: {
    failingCheckId: "hc-auth-prod-01-http",
    failureReason: "timeout",
    consecutiveFailures: 3
  }
  createdAt: 2026-08-12T13:42:12Z
  expiresAt: 2026-09-11T13:42:12Z
}
```

---

## 13. AuditLog

**Purpose**: Track administrative changes for compliance and debugging.

### Properties
```
id: UUID
timestamp: timestamp        # When change occurred
actor: string              # Username or service
actorId: UUID?             # User/service ID
action: string             # Created, Updated, Deleted, etc
resourceType: string       # Service, Deployment, AlertPolicy, etc
resourceId: UUID          # ID of modified resource
environmentId: UUID?      # Denormalized
changes: JSON             # Before/after values
changeDescription: string # Human-readable change summary
ip: string?               # Source IP
userAgent: string?        # User-Agent header
status: string            # success, failure
errorMessage: string?     # If failed
metadata: JSON            # Additional context
createdAt: timestamp      # Immutable
expiresAt: timestamp?     # For retention policies
```

### Actions
```
Create
Update
Delete
Enable
Disable
Grant
Revoke
Authenticate
TokenCreated
TokenRevoked
```

### Data Retention
- Store: Indefinitely (for compliance)
- Purge: Never (unless configured)
- Purpose: Audit trail and compliance

### Example
```
AuditLog {
  id: "audit-20260812-143542-001"
  timestamp: 2026-08-12T14:35:00Z
  actor: "alex.johnson"
  actorId: "user-001"
  action: "Update"
  resourceType: "HealthCheck"
  resourceId: "hc-auth-prod-01-http"
  environmentId: "env-prod-001"
  changes: {
    interval: {
      before: "30s",
      after: "10s"
    },
    timeout: {
      before: "5s",
      after: "3s"
    }
  }
  changeDescription: "Updated health check interval from 30s to 10s"
  ip: "10.20.1.50"
  userAgent: "curl/7.68.0"
  status: "success"
  errorMessage: null
  metadata: {
    reason: "Performance improvement"
  }
  createdAt: 2026-08-12T14:35:00Z
}
```

---

## 14. User (MVP)

**Purpose**: Local user account for authentication.

### Properties
```
id: UUID
username: string           # Unique, immutable
email: string             # Unique
displayName: string       # Full name
passwordHash: string      # Bcrypt
role: string              # Administrator, Operator, Viewer, Automation
enabled: bool
tags: map<string, string>
createdAt: timestamp
updatedAt: timestamp
lastLoginAt: timestamp?
deletedAt: timestamp?     # Soft delete
```

### Roles
```
Administrator   # Full access
Operator        # Service operations
Viewer          # Read-only
Automation      # API token only (no password)
```

---

## 15. ApiToken

**Purpose**: Token-based access for automation.

### Properties
```
id: UUID
userId: UUID              # Foreign key
name: string              # e.g., "CI/CD Pipeline"
token: string             # Hashed token
scope: string[]          # Permissions: read, write, admin
environmentIds: UUID[]   # If environment-scoped
expiresAt: timestamp?    # Optional expiration
lastUsedAt: timestamp?
enabled: bool
createdAt: timestamp
createdBy: UUID          # User who created token
```

---

## 16. Value Objects & Enums

### Duration
```
type Duration struct {
  value int       // milliseconds
}

methods:
  Seconds()     int
  Milliseconds() int
  String()      string
```

### HealthState
```
enum HealthState:
  UNKNOWN
  HEALTHY
  DEGRADED
  UNHEALTHY
  DISABLED
```

### Protocol
```
enum Protocol:
  HTTP
  HTTPS
  GRPC
  TCP
  UDP
```

### IncidentState
```
enum IncidentState:
  OPEN
  RESOLVED
```

### CheckType
```
enum CheckType:
  HTTP
  HTTPS
  GRPC
  TCP
  UDP
  HEARTBEAT
```

---

## 17. Domain Invariants

### Service Invariants
- Service name globally unique
- Service name immutable
- Service must have at least one deployment to be operational
- Service deletion cascades to all deployments

### Deployment Invariants
- Combination (service, environment) unique
- Cannot change service or environment after creation
- Deletion cascades to all instances

### Instance Invariants
- Instance name unique within deployment
- Instance address must be valid (IP or hostname)
- Cannot be healthy in multiple deployments
- Instance deletion preserves health history

### HealthCheck Invariants
- Health checks execute outside transactions
- Timeout < interval
- Consecutive failure threshold >= 1
- Check execution timeouts after configured duration
- No concurrent executions of same check

### State Transition Invariants
- Transitions deterministic based on consecutive threshold
- State changes only on threshold, not single check
- Incident created only on entry to UNHEALTHY
- Incident resolved only on exit from UNHEALTHY
- Transitions logged as events

### Alert Invariants
- Alert cooldown prevents duplicate notifications
- Recovery notification always sent (no cooldown)
- Alert channels must be reachable before enable
- Multiple alert policies can apply to one incident

---

## 18. Aggregates

### Service Aggregate
```
Root: Service
Children:
  - ServiceDeployment
    - ServiceInstance
      - Endpoint
      - HealthCheck
      - HealthState
  - Tags
  - Metadata
```

### Incident Aggregate
```
Root: Incident
Children:
  - IncidentEvents
  - RelatedAlerts
  - RelatedHealthResults
```

---

## 19. Domain Events

All events implement:
```
interface DomainEvent {
  ID() UUID
  Timestamp() time.Time
  ResourceType() string
  ResourceID() UUID
  Type() string
  Metadata() map[string]interface{}
}
```

Published to event subscribers for:
- Persistence (event store)
- Audit logging
- Alert triggering
- UI updates (via streaming)
- Metrics collection

---

## 20. Key Relationships

```
Environment
  1 ──────► * Service
                │
                1 ──────► * ServiceDeployment
                             │
                             1 ──────► * ServiceInstance
                                          │
                                          ├─ 1 ──────► * Endpoint
                                          ├─ 1 ──────► * HealthCheck
                                          └─ 1 ──────► 1 HealthState
                                                          │
                                                          └─ * HealthResult

ServiceDeployment
  ├─ 1 ──────► * AlertPolicy
  └─ * ◄────── 1 Incident
                   │
                   └─ * ◄────── * AlertAttempt

HealthCheck
  └─ 1 ──────► * HealthResult

AlertPolicy
  └─ * ◄────── * NotificationChannel

User
  └─ 1 ──────► * ApiToken

ServiceInstance
  └─ 1 ──────► * Event
```

This domain model provides a clear, maintainable foundation for the Service Registry system.
