# MVP 2: Runtime Registration & Management

## Product Boundary

Alauda remains a lightweight, open-source service registry and health administration platform.

It stores and administers:

- where services exist
- which environment they belong to
- which runtime instances exist
- which endpoints they expose
- whether those endpoints are healthy
- what incidents occurred
- who was notified

It remains focused on services, environments, runtime instances, endpoints, health checks, incidents, alerts, events, and administration.

Alauda must not route traffic, proxy traffic, load balance, rewrite requests, configure networking, or control service-to-service traffic. Alauda observes and catalogs runtime topology. It does not participate in traffic.

## Technical Impact Assessment

This iteration should be implemented in small coherent increments. The current code already has the core entities and lower-level create/list/update/delete APIs, plus a basic web form that can create/reuse a deployment, create an instance, and optionally create one endpoint.

### Database Migration

- `service_deployments` already has `UNIQUE(service_id, environment_id)`, which supports the one-deployment-per-service-environment invariant.
- `service_instances.port` currently exists. The preferred model moves port ownership to `endpoints.port`, but removing `Instance.port` would affect persisted data, generated API types, CLI flags, UI display, health fallback behavior, and discovery resolution.
- `endpoints.primary` does not exist and needs an additive migration.
- Endpoint primary uniqueness should be enforced transactionally. SQLite can support this with a partial unique index such as one primary endpoint per non-deleted instance.
- Instance duplicate handling needs review. At minimum, uniqueness should be confirmed or added for `(deployment_id, name)` where `deleted_at IS NULL`.

### Protobuf/API Changes

- Keep `CreateDeployment`, `CreateInstance`, and `CreateEndpoint`.
- Add `RegisterRuntime` as the standard web workflow API.
- Add `primary` to `Endpoint`, `CreateEndpointRequest`, and `UpdateEndpointRequest`.
- Treat `Instance.port` as deprecated if compatibility requires keeping it temporarily.
- Prefer generated ConnectRPC clients as the source of truth where applicable.

### Application-Layer Changes

- Add a transactional registration operation:
  - validate request
  - resolve existing deployment
  - create deployment if missing
  - create instance
  - create endpoints
  - enforce at most one primary endpoint
  - commit
  - publish events after commit
  - return the complete registration result
- Do not perform external work inside the transaction.
- If health-check creation cannot safely be part of the same transaction, create runtime registration first and report health-check creation failure clearly.

### Frontend Changes

- Replace the large deployment-heavy form with a compact guided registration workflow.
- Hide deployment IDs in the normal workflow.
- Support multiple endpoint rows and a single primary endpoint.
- Add optional health-check configuration.
- Add visible success state and clear first-instance empty states.
- Use primary endpoint for compact instance presentation.
- Add environment overview to Service Detail.

### CLI Compatibility

- Existing commands must continue to work:
  - `registryctl deployment create`
  - `registryctl instance register`
  - `registryctl endpoint create`
- If `Instance.port` remains for compatibility, mark it deprecated and direct users to endpoint ports.
- Preferred future CLI registration is instance address first, then endpoint creation with `--port` and optional `--primary`.

### Tests

- Add focused frontend tests for registration behavior, validation, errors, primary endpoint handling, empty states, and selected-service preservation.
- Add backend integration tests for `RegisterRuntime`, deployment reuse/create, duplicate deployment prevention, primary endpoint invariant, rollback behavior, and environment isolation.

### Documentation

- Update `docs/MVP.md`, `docs/DOMAIN.md`, and `docs/API.md`.
- Add or update `docs/RUNTIME_REGISTRATION.md`.
- Explain the hierarchy clearly:

```text
Service
  Deployment in Environment
    Instance
      Endpoint
```

## Domain Model

Keep these concepts separate:

- `Service`: logical catalog identity.
- `Environment`: runtime context such as DEV, QA, STAGING, PROD.
- `Deployment`: a Service within an Environment.
- `Instance`: one concrete running copy of a Deployment.
- `Endpoint`: a protocol-specific addressable interface exposed by an Instance.
- `HealthCheck`: configuration for checking an Instance or Endpoint.

Example:

```text
Service
checkout
   |
   +-- Deployment: QA
   |      |
   |      +-- Instance
   |             address: 10.10.20.15
   |
   |             +-- HTTP :8080
   |             +-- gRPC :5001
   |
   +-- Deployment: PROD
          |
          +-- Instance
          +-- Instance
```

Do not collapse these concepts into one persistence model.

## Instance And Endpoint Ownership

Preferred model:

```text
Instance
  name
  address
  description
  tags
  metadata

Endpoint
  name
  protocol
  port
  path
  primary
  enabled
  metadata
```

An instance can expose multiple ports:

```text
10.0.0.10

HTTP      :8080
gRPC      :5001
Metrics   :9090
```

Therefore a single `Instance.port` is ambiguous. The network identity of an Instance should be its address. The Endpoint defines how a particular interface is reached.

Compatibility rule: because `Instance.port` already exists in persisted and public API contracts, do not remove it casually. First add endpoint-first behavior, migrate existing non-zero instance ports into a default endpoint where safe, update UI/CLI/docs to prefer endpoint ports, then deprecate and remove `Instance.port` only in an explicit compatibility-breaking release.

## Primary Endpoint

An endpoint may be marked:

```text
primary = true
```

Rules:

- A runtime Instance should have at most one primary endpoint.
- Primary endpoint is optional.
- If exactly one endpoint exists, it may automatically become primary.
- Creating a new primary endpoint should safely replace the previous primary endpoint for that Instance.
- Enforce this invariant transactionally where possible.
- `primary` is display/discovery metadata only. It is not traffic routing priority.

Example:

```text
checkout-prod-01
10.0.0.10

* HTTP
  https://10.0.0.10:8080

  gRPC
  10.0.0.10:5001

  Metrics
  http://10.0.0.10:9090/metrics
```

Use the primary endpoint for compact UI presentation when appropriate.

## Deployment UX

`Deployment` remains an important domain entity, but it should not normally be something an operator must manually manage during runtime registration.

Normal registration workflow:

```text
Select Service
Select Environment
Resolve Deployment
```

Alauda should then:

```text
Does Service + Environment deployment exist?

YES -> reuse it
NO  -> create it
```

Do not require the user to understand or enter deployment IDs in the normal web workflow. Keep explicit deployment management available through advanced API/CLI operations.

Deployment invariant:

```text
one Deployment per Service + Environment
```

Database conceptually:

```sql
UNIQUE(service_id, environment_id)
```

Runtime registration must be idempotent with respect to deployment resolution. Submitting the registration workflow multiple times must not create duplicate deployments for the same Service and Environment.

## RegisterRuntime Operation

Keep existing lower-level API operations:

- `CreateDeployment`
- `CreateInstance`
- `CreateEndpoint`

Add a higher-level application operation for the web registration workflow:

```text
RegisterRuntime
```

Conceptual request:

```json
{
  "serviceId": "...",
  "environmentId": "...",
  "instance": {
    "name": "checkout-prod-01",
    "address": "10.0.0.10",
    "description": ""
  },
  "endpoints": [
    {
      "name": "http",
      "protocol": "https",
      "port": 8080,
      "path": "/",
      "primary": true
    },
    {
      "name": "grpc",
      "protocol": "grpc",
      "port": 5001,
      "primary": false
    }
  ]
}
```

Application flow:

```text
RegisterRuntime
  Validate request
  Resolve existing Deployment
  Create Deployment if missing
  Create Instance
  Create Endpoint(s)
  Commit
  Publish event(s)
  Return complete registration result
```

Runtime registration must behave as one logical operation. Avoid silently leaving confusing partial state. For SQLite, perform local persistence mutations inside one appropriate database transaction. External operations must not occur inside the transaction. If architectural constraints prevent full transactional behavior, implement explicit compensation and document the behavior. The user should receive a clear error explaining whether anything was persisted.

## Registration Wizard

Replace or refine the large registration form into a small guided workflow. Avoid excessive wizard ceremony; the workflow should remain fast.

### Step 1: Runtime Context

```text
Register Runtime

Service
checkout

Environment
PRODUCTION
```

Service may already be selected when initiated from Service Detail. Prefer the global environment selection when available.

### Step 2: Instance

Inputs:

- Instance name
- Address
- Description

Example:

```text
checkout-prod-01
10.0.0.10
Primary checkout instance
```

Address must support IPv4, IPv6, hostname, and DNS name.

### Step 3: Endpoints

Allow one or more endpoints.

Initial endpoint row:

```text
Name
Protocol
Port
Path
Primary
```

Example:

```text
http
HTTPS
8080
/
Primary yes
```

Supported protocol values:

- HTTP
- HTTPS
- gRPC
- TCP
- UDP

Path should only be required or emphasized when relevant.

### Step 4: Health

Health configuration is optional. The operator can choose:

- Configure later
- Add health check

Initial supported configuration should reuse existing health capabilities:

```text
Health Check

Name
readiness

Type
HTTP

Endpoint
http

Path
/healthz

Interval
10 seconds

Timeout
3 seconds

Failures before unhealthy
3

Successes before healthy
2
```

Do not make health configuration mandatory for registration. If the health-check API cannot yet be safely included in the same composite operation, complete runtime registration first and create the health check immediately afterward with clearly handled failure behavior.

### Step 5: Review

Before submission, optionally show a compact review:

```text
checkout
PRODUCTION

Instance
checkout-prod-01
10.0.0.10

Endpoints
* HTTPS :8080 /
  gRPC  :5001

Health
HTTP /healthz every 10s
```

Primary action:

```text
Register Runtime
```

## Success State

After successful registration:

- close or complete the registration workflow
- keep the Service selected
- keep the current Environment selected
- reload the affected catalog/runtime data
- show an inline success message
- display the newly registered Instance
- display all newly registered Endpoints
- update health data if a health check was configured

Example:

```text
Runtime registered successfully.

checkout-prod-01
10.0.0.10

HTTPS :8080
gRPC  :5001
```

Do not rely only on a disappearing toast. Use an accessible visible success state.

## Empty States

When a Service Deployment has no instances, display:

```text
No runtime instances registered.

Alauda knows this service exists, but it does not yet know
where this service is running.

[ Register first instance ]
```

If no deployment exists for the currently selected Environment:

```text
checkout has no runtime registration in QA.

[ Register runtime in QA ]
```

The purpose is to clearly teach the difference between service catalog metadata and runtime registration.

## Service Detail Environment Overview

Improve Service Detail so one logical Service can be understood across environments.

Example:

```text
checkout

ENVIRONMENT     INSTANCES     HEALTH       VERSION

DEV                 1         HEALTHY       3.1.0
QA                  2         HEALTHY       3.0.8
PROD                4         DEGRADED      3.0.5
```

Selecting an Environment should show that Deployment's runtime details. Do not create a separate duplicate Service record per Environment.

## Instance Presentation

Instance cards/tables should show:

- Instance name
- Address
- Environment
- Health
- Primary endpoint
- Endpoint count
- Last health change

Example:

```text
checkout-prod-01

10.0.0.10

Primary
HTTPS :8080

Endpoints
2

Health
HEALTHY
```

Do not display every endpoint in high-density list screens. Use the primary endpoint for summary presentation.

## Endpoint Presentation

Instance detail should expose all endpoints.

Example:

```text
ENDPOINT       PROTOCOL       ADDRESS

* http         HTTPS          10.0.0.10:8080
  grpc         gRPC           10.0.0.10:5001
  metrics      HTTP           10.0.0.10:9090/metrics
```

The address is derived from:

```text
Instance.address + Endpoint.port
```

plus protocol/path where relevant.

## Validation

General validation requires:

- service
- environment
- instance name
- instance address

For every endpoint require:

- name
- protocol
- valid port from `1` to `65535`

HTTP/HTTPS path validation should permit `/` and other valid relative paths. Do not require paths for TCP or UDP. gRPC path/service semantics should follow the existing health/discovery model.

Primary endpoint validation:

- maximum one primary endpoint

Backend validation remains authoritative. Frontend validation exists only for fast feedback.

## Duplicate Handling

Define behavior for duplicate Instance registration. At minimum, detect collisions based on the existing Instance identity rules.

Do not silently create duplicate runtime instances because the user double-clicked Submit. Use disabled submit while a request is pending, request idempotency where practical, and database uniqueness constraints where appropriate. Return a useful conflict error.

## Edit And Delete Management

Keep these management tasks inside MVP 2:

- Edit Instance
- Delete Instance
- Edit Endpoint
- Delete Endpoint

Prefer contextual actions from Service Detail or Instance Detail.

Deletion must clearly explain relationships:

```text
Delete instance checkout-prod-01?

This will also remove:
- 2 endpoints
- 1 health check

Existing incident/event history will be preserved where appropriate.
```

Use existing backend semantics where already defined. Do not silently cascade operational history unless explicitly intended by the domain.

## API Client Updates

Add or update frontend API helpers for:

- `RegisterRuntime`
- `CreateDeployment`
- `CreateInstance`
- `CreateEndpoint`

Keep the lower-level helpers. Prefer `RegisterRuntime` for the standard UI workflow. Generated ConnectRPC clients should remain the source of truth where applicable.

## Focused UI Tests

Add focused frontend coverage for at least:

- register instance into existing deployment
- register instance when deployment does not exist
- register instance with one endpoint
- register instance with multiple endpoints
- primary endpoint handling
- validation failure
- backend error
- successful registration
- empty-state registration action
- selected Service remains active after successful registration

## Backend Integration Tests

Add tests for:

- `RegisterRuntime` creates missing deployment
- `RegisterRuntime` reuses existing deployment
- duplicate registration does not create duplicate deployment
- multiple endpoints are created correctly
- only one primary endpoint exists
- failure rolls back runtime registration
- environment isolation is preserved

## Existing Compatibility

Preserve compatibility for existing CLI/API workflows unless a schema migration is explicitly required.

Existing commands should continue to work:

```bash
registryctl deployment create
registryctl instance register
registryctl endpoint create
```

Preferred future CLI model:

```bash
registryctl instance register \
  --deployment-id <deployment-id> \
  --name checkout-a \
  --address 10.0.0.10
```

then:

```bash
registryctl endpoint create \
  --instance-id <instance-id> \
  --name http \
  --protocol https \
  --port 8080 \
  --path / \
  --primary
```

If compatibility requires retaining an existing port argument temporarily, deprecate it explicitly rather than silently changing semantics.

## Acceptance Criteria

The updated MVP 2 is complete when:

- A user can register a runtime Instance entirely from the web UI.
- A user does not need to manually create or understand a Deployment during the normal workflow.
- Alauda reuses an existing Service + Environment Deployment automatically.
- Alauda creates the Deployment automatically when none exists.
- Duplicate Deployments cannot be created for the same Service + Environment.
- An Instance stores its runtime address.
- One or more Endpoints can be created during registration.
- Ports belong to Endpoints in the preferred domain model.
- One Endpoint can optionally be designated as primary.
- Multiple Endpoints can be registered in the same workflow.
- Health configuration can optionally be added during registration.
- Health configuration can also be skipped and added later.
- Runtime registration behaves as one logical operation.
- Partial failure does not silently leave confusing runtime state.
- Newly created runtime resources immediately appear in Service Detail.
- Services with no Instances display a clear registration empty state.
- Environment overview remains visible and understandable.
- Existing CLI functionality continues to work.
- Instance editing works.
- Instance deletion works.
- Endpoint editing works.
- Endpoint deletion works.
- Focused UI registration tests pass.
- Backend runtime-registration integration tests pass.
- Existing test suites pass.
- Production frontend build passes.
- Go build and tests pass.

## Implementation Status

### Already Completed

- [x] CreateDeployment frontend API helper
- [x] CreateInstance frontend API helper
- [x] CreateEndpoint frontend API helper
- [x] Basic Register Instance form
- [x] Existing deployment reuse/create support
- [x] Instance address support
- [x] Optional endpoint creation
- [x] Catalog reload after registration
- [x] Existing frontend tests passing
- [x] Production frontend build passing

### Refine / Implement

- [ ] Review Instance.port ownership and migration impact
- [ ] Move port ownership to Endpoint if compatibility permits
- [x] Add Endpoint.primary
- [x] Enforce one Deployment per Service + Environment
- [x] Add RegisterRuntime application/API operation
- [x] Make runtime registration transactional
- [x] Auto-resolve/create Deployment in standard UI flow
- [ ] Replace Deployment-heavy form with guided runtime workflow
- [x] Support multiple Endpoint rows
- [x] Add optional Health Check step
- [x] Add registration success state
- [x] Add first-instance empty state
- [x] Add environment comparison to Service Detail
- [x] Add Edit Instance
- [x] Add Delete Instance
- [x] Add Edit Endpoint
- [x] Add Delete Endpoint
- [x] Add focused UI registration tests
- [x] Add backend RegisterRuntime integration tests
- [x] Update documentation

## UX Principle

The user should think:

```text
I have a Service.
It runs in PROD.
Here is one running Instance.
These are the interfaces it exposes.
This is how Alauda should check its health.
```

The user should not have to think:

```text
I need to manually create a Deployment ID
before I can create an Instance.
```

The domain may contain more entities than the user workflow exposes. That is acceptable and intentional.

## Architectural Principle

Preserve the distinction:

```text
Domain simplicity != database flattening

UX simplicity != domain flattening
```

Alauda should keep a correct domain model while presenting a simple operator experience. Do not move address or port onto `Service` simply because it appears easier in a form.

## Implementation Order

Prioritize in this order:

1. Domain invariants
2. Database constraints/migrations
3. `RegisterRuntime` application operation
4. API contracts
5. Frontend registration workflow
6. Empty states and success UX
7. Instance/Endpoint management
8. Tests
9. Documentation
