# Protobuf API Contracts

This service exposes ConnectRPC APIs generated from Protobuf plus a small set of operational REST endpoints:

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/password`
- `GET /api/v1/auth/me`
- `GET /api/v1/auth/sessions`
- `DELETE /api/v1/auth/sessions/{id}`
- `GET|POST /api/v1/auth/users`
- `GET|POST /api/v1/auth/tokens`
- `DELETE /api/v1/auth/tokens/{id}`
- `GET /api/v1/audit-logs`
- `GET /api/v1/events/watch`
- `POST /api/v1/alerts/test/{channelId}`
- `GET /api/v1/ping`

Configured servers require authentication for API requests by default. The explicit public API allowlist is `POST /api/v1/auth/login` and `GET /api/v1/ping`; liveness and readiness are exposed separately at `/healthz` and `/readyz`. Browser sessions use the HttpOnly `alauda_session` cookie, while Bearer tokens remain supported for CLI and automation. Missing or invalid authentication returns `401`; an authenticated principal without the required scope returns `403`. Read requests require `read`, mutations require `write`, and user/token/session administration requires `admin`.

This document defines all Protocol Buffer service definitions and messages for the Service Registry.

The API is organized into logical services around domain boundaries:

- `EnvironmentService`
- `CatalogService` (services management)
- `DeploymentService` (service deployments)
- `InstanceService` (service instances)
- `InstanceService.RegisterRuntime` (composite runtime registration workflow)
- `HealthService` (health checks and state)
- `IncidentService` (incidents)
- `AlertService` (alerts and policies)
- `EventService` (event stream and history)
- `RegistryService` (service lookup for discovery)

### Public registration vocabulary

Public registration uses `Service`, `Environment`, `Instance`, an address,
`Endpoint`, and optional `Health monitoring`. `Deployment` and its database ID
remain internal implementation details. Existing administrative APIs may still
return internal records for compatibility, but normal registration clients
should not need those IDs.

---

## Proto File Structure

```
api/
└── registry/
    ├── v1/
    │   ├── environment.proto
    │   ├── catalog.proto
    │   ├── deployment.proto
    │   ├── instance.proto
    │   ├── endpoint.proto
    │   ├── health.proto
    │   ├── incident.proto
    │   ├── alert.proto
    │   ├── event.proto
    │   ├── registry.proto
    │   └── common.proto
    └── buf.yaml
```

---

## Common Types (common.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";

// Standard error response
message Error {
  string code = 1;
  string message = 2;
  map<string, string> details = 3;
}

// Health state enumeration
enum HealthState {
  HEALTH_STATE_UNSPECIFIED = 0;
  HEALTH_STATE_UNKNOWN = 1;
  HEALTH_STATE_HEALTHY = 2;
  HEALTH_STATE_DEGRADED = 3;
  HEALTH_STATE_UNHEALTHY = 4;
  HEALTH_STATE_DISABLED = 5;
}

// Health check type enumeration
enum HealthCheckType {
  HEALTH_CHECK_TYPE_UNSPECIFIED = 0;
  HEALTH_CHECK_TYPE_HTTP = 1;
  HEALTH_CHECK_TYPE_HTTPS = 2;
  HEALTH_CHECK_TYPE_GRPC = 3;
  HEALTH_CHECK_TYPE_TCP = 4;
  HEALTH_CHECK_TYPE_UDP = 5;
  HEALTH_CHECK_TYPE_HEARTBEAT = 6;
}

// Protocol enumeration
enum Protocol {
  PROTOCOL_UNSPECIFIED = 0;
  PROTOCOL_HTTP = 1;
  PROTOCOL_HTTPS = 2;
  PROTOCOL_GRPC = 3;
  PROTOCOL_TCP = 4;
  PROTOCOL_UDP = 5;
}

// Incident state enumeration
enum IncidentState {
  INCIDENT_STATE_UNSPECIFIED = 0;
  INCIDENT_STATE_OPEN = 1;
  INCIDENT_STATE_RESOLVED = 2;
}

// User role enumeration
enum UserRole {
  USER_ROLE_UNSPECIFIED = 0;
  USER_ROLE_ADMINISTRATOR = 1;
  USER_ROLE_OPERATOR = 2;
  USER_ROLE_VIEWER = 3;
  USER_ROLE_AUTOMATION = 4;
}

// Pagination request
message PaginationRequest {
  int32 page_size = 1;  // Default 50, max 1000
  string page_token = 2;
}

// Pagination response
message PaginationResponse {
  string next_page_token = 1;
  int32 total_size = 2;
}

// Duration value object
message Duration {
  int64 seconds = 1;  // e.g., 10 for 10 seconds
}

// Tag key-value pair
message Tag {
  string key = 1;
  string value = 2;
}

// Server health/version response
message HealthCheckResponse {
  bool alive = 1;
  string version = 2;
  google.protobuf.Timestamp timestamp = 3;
}
```

---

## Environments (environment.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "registry/v1/common.proto";

service EnvironmentService {
  rpc CreateEnvironment(CreateEnvironmentRequest) returns (CreateEnvironmentResponse);
  rpc UpdateEnvironment(UpdateEnvironmentRequest) returns (UpdateEnvironmentResponse);
  rpc DeleteEnvironment(DeleteEnvironmentRequest) returns (google.protobuf.Empty);
  rpc GetEnvironment(GetEnvironmentRequest) returns (GetEnvironmentResponse);
  rpc ListEnvironments(ListEnvironmentsRequest) returns (ListEnvironmentsResponse);
}

message Environment {
  string id = 1;
  string key = 2;  // Unique identifier
  string name = 3;
  string description = 4;
  bool enabled = 5;
  string tier = 6;  // Optional: production, staging, development
  repeated Tag tags = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
}

message CreateEnvironmentRequest {
  string key = 1;
  string name = 2;
  string description = 3;
  bool enabled = 4;
  string tier = 5;
  repeated Tag tags = 6;
}

message CreateEnvironmentResponse {
  Environment environment = 1;
}

message UpdateEnvironmentRequest {
  string id = 1;
  string name = 2;
  string description = 3;
  bool enabled = 4;
  string tier = 5;
  repeated Tag tags = 6;
}

message UpdateEnvironmentResponse {
  Environment environment = 1;
}

message DeleteEnvironmentRequest {
  string id = 1;
}

message GetEnvironmentRequest {
  string id = 1;
}

message GetEnvironmentResponse {
  Environment environment = 1;
}

message ListEnvironmentsRequest {
  PaginationRequest pagination = 1;
  bool enabled_only = 2;
}

message ListEnvironmentsResponse {
  repeated Environment environments = 1;
  PaginationResponse pagination = 2;
}
```

---

## Services Catalog (catalog.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "registry/v1/common.proto";

service CatalogService {
  rpc CreateService(CreateServiceRequest) returns (CreateServiceResponse);
  rpc UpdateService(UpdateServiceRequest) returns (UpdateServiceResponse);
  rpc DeleteService(DeleteServiceRequest) returns (google.protobuf.Empty);
  rpc GetService(GetServiceRequest) returns (GetServiceResponse);
  rpc ListServices(ListServicesRequest) returns (ListServicesResponse);
}

message Service {
  string id = 1;
  string name = 2;  // Unique, immutable
  string display_name = 3;
  string description = 4;
  repeated Tag tags = 5;
  map<string, string> metadata = 6;
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
}

message CreateServiceRequest {
  string name = 1;
  string display_name = 2;
  string description = 3;
  repeated Tag tags = 4;
  map<string, string> metadata = 5;
}

message CreateServiceResponse {
  Service service = 1;
}

message UpdateServiceRequest {
  string id = 1;
  string display_name = 2;
  string description = 3;
  repeated Tag tags = 4;
  map<string, string> metadata = 5;
}

message UpdateServiceResponse {
  Service service = 1;
}

message DeleteServiceRequest {
  string id = 1;
}

message GetServiceRequest {
  string id = 1;
}

message GetServiceResponse {
  Service service = 1;
}

message ListServicesRequest {
  PaginationRequest pagination = 1;
  string environment_id = 2;  // Optional filter
  repeated Tag tag_filters = 3;
}

message ListServicesResponse {
  repeated Service services = 1;
  PaginationResponse pagination = 2;
}
```

---

## Deployments (deployment.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "registry/v1/common.proto";

service DeploymentService {
  rpc CreateDeployment(CreateDeploymentRequest) returns (CreateDeploymentResponse);
  rpc UpdateDeployment(UpdateDeploymentRequest) returns (UpdateDeploymentResponse);
  rpc DeleteDeployment(DeleteDeploymentRequest) returns (google.protobuf.Empty);
  rpc GetDeployment(GetDeploymentRequest) returns (GetDeploymentResponse);
  rpc ListDeployments(ListDeploymentsRequest) returns (ListDeploymentsResponse);
}

message ServiceDeployment {
  string id = 1;
  string service_id = 2;
  string environment_id = 3;
  bool health_enabled = 4;
  bool alerts_enabled = 5;
  int32 alert_cooldown_minutes = 6;
  repeated Tag tags = 7;
  map<string, string> metadata = 8;
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

message CreateDeploymentRequest {
  string service_id = 1;
  string environment_id = 2;
  bool health_enabled = 3;
  bool alerts_enabled = 4;
  int32 alert_cooldown_minutes = 5;
  repeated Tag tags = 6;
  map<string, string> metadata = 7;
}

message CreateDeploymentResponse {
  ServiceDeployment deployment = 1;
}

message UpdateDeploymentRequest {
  string id = 1;
  bool health_enabled = 2;
  bool alerts_enabled = 3;
  int32 alert_cooldown_minutes = 4;
  repeated Tag tags = 5;
  map<string, string> metadata = 6;
}

message UpdateDeploymentResponse {
  ServiceDeployment deployment = 1;
}

message DeleteDeploymentRequest {
  string id = 1;
}

message GetDeploymentRequest {
  string id = 1;
}

message GetDeploymentResponse {
  ServiceDeployment deployment = 1;
}

message ListDeploymentsRequest {
  PaginationRequest pagination = 1;
  string service_id = 2;  // Optional filter
  string environment_id = 3;  // Optional filter
}

message ListDeploymentsResponse {
  repeated ServiceDeployment deployments = 1;
  PaginationResponse pagination = 2;
}
```

---

## Instances (instance.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "registry/v1/common.proto";

service InstanceService {
  rpc RegisterInstance(RegisterInstanceRequest) returns (RegisterInstanceResponse);
  rpc UpdateInstance(UpdateInstanceRequest) returns (UpdateInstanceResponse);
  rpc RemoveInstance(RemoveInstanceRequest) returns (google.protobuf.Empty);
  rpc GetInstance(GetInstanceRequest) returns (GetInstanceResponse);
  rpc ListInstances(ListInstancesRequest) returns (ListInstancesResponse);
}

message ServiceInstance {
  string id = 1;
  string deployment_id = 2;
  string name = 3;
  string address = 4;
  int32 port = 5;  // Legacy optional default; prefer Endpoint.port for new workflows
  string description = 6;
  bool enabled = 7;
  repeated Tag tags = 8;
  map<string, string> metadata = 9;
  google.protobuf.Timestamp created_at = 10;
  google.protobuf.Timestamp updated_at = 11;
  google.protobuf.Timestamp last_seen_at = 12;  // Optional
}

message RegisterInstanceRequest {
  string deployment_id = 1;
  string name = 2;
  string address = 3;
  int32 port = 4;  // Legacy optional default; prefer creating Endpoint records
  string description = 5;
  repeated Tag tags = 6;
  map<string, string> metadata = 7;
}

message RegisterInstanceResponse {
  ServiceInstance instance = 1;
}

message UpdateInstanceRequest {
  string id = 1;
  string name = 2;  // Optional update
  string description = 3;
  bool enabled = 4;
  repeated Tag tags = 5;
  map<string, string> metadata = 6;
}

message UpdateInstanceResponse {
  ServiceInstance instance = 1;
}

message RemoveInstanceRequest {
  string id = 1;
}

message GetInstanceRequest {
  string id = 1;
}

message GetInstanceResponse {
  ServiceInstance instance = 1;
}

message ListInstancesRequest {
  PaginationRequest pagination = 1;
  string deployment_id = 2;  // Optional filter
  string environment_id = 3;  // Optional filter
  string service_id = 4;  // Optional filter
  bool enabled_only = 5;
}

message ListInstancesResponse {
  repeated ServiceInstance instances = 1;
  PaginationResponse pagination = 2;
}
```

---

## Endpoints (endpoint.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/empty.proto";
import "registry/v1/common.proto";

service EndpointService {
  rpc CreateEndpoint(CreateEndpointRequest) returns (CreateEndpointResponse);
  rpc UpdateEndpoint(UpdateEndpointRequest) returns (UpdateEndpointResponse);
  rpc DeleteEndpoint(DeleteEndpointRequest) returns (google.protobuf.Empty);
  rpc ListEndpoints(ListEndpointsRequest) returns (ListEndpointsResponse);
}

message Endpoint {
  string id = 1;
  string instance_id = 2;
  string name = 3;
  Protocol protocol = 4;
  int32 port = 5;
  string path = 6;  // Optional
  string description = 7;
  bool enabled = 8;
  repeated Tag tags = 9;
  map<string, string> metadata = 10;
  bool primary = 11;  // Optional display/discovery default for the instance
}

message CreateEndpointRequest {
  string instance_id = 1;
  string name = 2;
  Protocol protocol = 3;
  int32 port = 4;
  string path = 5;  // Optional
  string description = 6;
  repeated Tag tags = 7;
  map<string, string> metadata = 8;
  bool primary = 9;
}

message CreateEndpointResponse {
  Endpoint endpoint = 1;
}

message UpdateEndpointRequest {
  string id = 1;
  string name = 2;
  Protocol protocol = 3;
  int32 port = 4;
  string path = 5;  // Optional
  string description = 6;
  bool enabled = 7;
  repeated Tag tags = 8;
  map<string, string> metadata = 9;
  bool primary = 10;
}

message UpdateEndpointResponse {
  Endpoint endpoint = 1;
}

message DeleteEndpointRequest {
  string id = 1;
}

message ListEndpointsRequest {
  string instance_id = 1;
  Protocol protocol_filter = 2;  // Optional filter
}

message ListEndpointsResponse {
  repeated Endpoint endpoints = 1;
}
```

---

## Service Registration (`InstanceService.RegisterRuntime`)

```protobuf
syntax = "proto3";

package registry.v1;

import "registry/v1/catalog.proto";
import "registry/v1/common.proto";
service InstanceService {
  rpc RegisterRuntime(RegisterRuntimeRequest) returns (RegisterRuntimeResponse);
}

message RegisterRuntimeRequest {
  string service_id = 1;
  string environment_id = 2;
  RuntimeInstanceRegistration instance = 3;
  repeated RuntimeEndpointRegistration endpoints = 4;
}

message RuntimeInstanceRegistration {
  string name = 1;
  string address = 2;
  string description = 3;
  map<string, string> tags = 4;
  map<string, string> metadata = 5;
}

message RuntimeEndpointRegistration {
  string name = 1;
  Protocol protocol = 2;
  int32 port = 3;
  string path = 4;
  bool primary = 5;
  bool enabled = 6;
  map<string, string> metadata = 7;
}

message RegisterRuntimeResponse {
  ServiceDeployment deployment = 1;
  ServiceInstance instance = 2;
  repeated Endpoint endpoints = 3;
}
```

`RegisterRuntime` is the preferred API for the standard web workflow. It resolves or creates the unique Service + Environment deployment, creates the runtime Instance, creates one or more Endpoints, enforces the one-primary-endpoint invariant, and returns the complete runtime registration result.

### Public CLI-friendly contract

The current wire request is ID-based and nested for compatibility. A future
additive public adapter over this same operation should accept stable public
references and flatten the service-address input:

The additive protobuf direction is:

```protobuf
message RegisterRuntimeRequest {
  string service_id = 1;       // existing clients
  string environment_id = 2;   // existing clients
  RuntimeInstanceRegistration instance = 3;
  repeated RuntimeEndpointRegistration endpoints = 4;
  string service = 5;          // public stable service name
  string environment = 6;      // public environment key/name
  string request_id = 7;       // future replay protection
  bool reconcile = 8;          // future create-or-update behavior
}
```

Fields 5-8 are design direction only until the server implements lookup,
precedence, and idempotency semantics.

```json
{
  "service": "Authentication.Grpc",
  "environment": "staging",
  "name": "authentication-01",
  "address": "lynx-authentication.lynx",
  "endpoints": [
    { "protocol": "grpc", "port": 81, "primary": true }
  ]
}
```

The adapter resolves `service` by exact stable service name, resolves
`environment` by key first and then by unique name, rejects missing or
ambiguous references, and invokes the same RegisterRuntime operation. It must
not create a Service or Environment implicitly. Endpoint names may be derived
from protocol only when unique; otherwise callers must provide them. The first
endpoint may become primary only when no endpoint is marked primary.

The backend translation remains:

```text
Service + Environment + Address
  -> resolve Service
  -> resolve Environment
  -> find/create Deployment
  -> create Instance
  -> create Endpoint(s)
```

Health monitoring is a separate follow-up operation and must not be required
for service registration or rolled back with the address registration.

### Idempotency and reconciliation

Current behavior is create-only. A repeated request with the same runtime name
within the same Service + Environment Deployment hits the database uniqueness
constraint; a different name creates another Instance. Address is not a
uniqueness key, and no request idempotency key is honored. Clients must not
retry blindly after an unknown result.

The recommended future `reconcile=true` behavior is to match
Service + Environment + Runtime name, update address/description and reconcile
endpoints when found, or create the runtime when missing. An explicit request
ID should provide replay protection for create requests, backed by durable
request-result storage or an equivalent transactional mechanism.

If a name is omitted by a future CLI, it may derive one deterministically from
service and address only after documented normalization rules. It must not
silently choose a name when multiple existing runtimes could match.

Current CLI limitations are: Service and Environment lookup require database
IDs; runtime and endpoint names are required; responses contain internal
Deployment and Instance records; and validation errors are not yet structured
per public field. These should be addressed additively in a public adapter,
without creating a separate CLI persistence workflow.

The persistence portion should behave as one logical operation. For SQLite, deployment resolution, instance creation, and endpoint creation should occur inside one transaction where possible. Events should be published after commit. If health-check creation is not included in the same transaction, the response/error must clearly indicate whether runtime registration succeeded and health-check creation failed.

---

## Health (health.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "registry/v1/common.proto";

service HealthService {
  rpc GetHealth(GetHealthRequest) returns (GetHealthResponse);
  rpc ListHealthChecks(ListHealthChecksRequest) returns (ListHealthChecksResponse);
  rpc CreateHealthCheck(CreateHealthCheckRequest) returns (CreateHealthCheckResponse);
  rpc UpdateHealthCheck(UpdateHealthCheckRequest) returns (UpdateHealthCheckResponse);
  rpc DeleteHealthCheck(DeleteHealthCheckRequest) returns (google.protobuf.Empty);
  rpc RunHealthCheck(RunHealthCheckRequest) returns (RunHealthCheckResponse);
  rpc GetHealthResults(GetHealthResultsRequest) returns (GetHealthResultsResponse);
  rpc WatchHealthEvents(google.protobuf.Empty) returns (stream HealthEvent);
}

message HealthState {
  string instance_id = 1;
  HealthState current_state = 2;
  int32 consecutive_successes = 3;
  int32 consecutive_failures = 4;
  google.protobuf.Timestamp last_transition_time = 5;
  google.protobuf.Timestamp last_check_time = 6;
  map<string, string> metadata = 7;
}

message HealthCheck {
  string id = 1;
  string instance_id = 2;
  string endpoint_id = 3;  // Optional
  string name = 4;
  HealthCheckType type = 5;
  bool enabled = 6;
  int32 interval_seconds = 7;
  int32 timeout_seconds = 8;
  int32 failures_before_unhealthy = 9;
  int32 successes_before_healthy = 10;
  string description = 11;
  repeated Tag tags = 12;
  map<string, string> metadata = 13;  // Type-specific config
  google.protobuf.Timestamp created_at = 14;
  google.protobuf.Timestamp updated_at = 15;
}

message HealthCheckResult {
  string id = 1;
  string health_check_id = 2;
  string instance_id = 3;
  google.protobuf.Timestamp timestamp = 4;
  bool success = 5;
  int32 latency_ms = 6;
  int32 status_code = 7;  // HTTP status or error code
  string error_type = 8;  // CONNECTION_REFUSED, TIMEOUT, etc
  string error_message = 9;
  map<string, string> metadata = 10;
}

message GetHealthRequest {
  string instance_id = 1;
}

message GetHealthResponse {
  HealthState health = 1;
}

message ListHealthChecksRequest {
  string instance_id = 1;
  bool enabled_only = 2;
}

message ListHealthChecksResponse {
  repeated HealthCheck checks = 1;
}

message CreateHealthCheckRequest {
  string instance_id = 1;
  string endpoint_id = 2;  // Optional
  string name = 3;
  HealthCheckType type = 4;
  int32 interval_seconds = 5;
  int32 timeout_seconds = 6;
  int32 failures_before_unhealthy = 7;
  int32 successes_before_healthy = 8;
  string description = 9;
  repeated Tag tags = 10;
  map<string, string> metadata = 11;  // Type-specific config
}

message CreateHealthCheckResponse {
  HealthCheck check = 1;
}

message UpdateHealthCheckRequest {
  string id = 1;
  int32 interval_seconds = 2;  // Optional
  int32 timeout_seconds = 3;  // Optional
  int32 failures_before_unhealthy = 4;  // Optional
  int32 successes_before_healthy = 5;  // Optional
  bool enabled = 6;
  repeated Tag tags = 7;
  map<string, string> metadata = 8;
}

message UpdateHealthCheckResponse {
  HealthCheck check = 1;
}

message DeleteHealthCheckRequest {
  string id = 1;
}

message RunHealthCheckRequest {
  string health_check_id = 1;
}

message RunHealthCheckResponse {
  HealthCheckResult result = 1;
}

message GetHealthResultsRequest {
  string health_check_id = 1;
  int32 limit = 2;  // Default 50
}

message GetHealthResultsResponse {
  repeated HealthCheckResult results = 1;
}

message HealthEvent {
  string id = 1;
  google.protobuf.Timestamp timestamp = 2;
  string instance_id = 3;
  HealthState old_state = 4;  // Previous state
  HealthState new_state = 5;  // New state
  string reason = 6;
}
```

---

## Incidents (incident.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";
import "registry/v1/common.proto";

service IncidentService {
  rpc ListIncidents(ListIncidentsRequest) returns (ListIncidentsResponse);
  rpc GetIncident(GetIncidentRequest) returns (GetIncidentResponse);
  rpc WatchIncidents(google.protobuf.Empty) returns (stream IncidentEvent);
}

message Incident {
  string id = 1;
  string instance_id = 2;
  string deployment_id = 3;
  string environment_id = 4;
  string service_id = 5;
  IncidentState state = 6;
  google.protobuf.Timestamp opened_at = 7;
  google.protobuf.Timestamp resolved_at = 8;  // Optional
  int32 duration_seconds = 9;  // Optional
  string reason = 10;
  string impact_summary = 11;
  repeated Tag tags = 12;
  map<string, string> metadata = 13;
}

message ListIncidentsRequest {
  PaginationRequest pagination = 1;
  IncidentState state_filter = 2;  // Optional
  string environment_id = 3;  // Optional
  string service_id = 4;  // Optional
  string instance_id = 5;  // Optional
}

message ListIncidentsResponse {
  repeated Incident incidents = 1;
  PaginationResponse pagination = 2;
}

message GetIncidentRequest {
  string id = 1;
}

message GetIncidentResponse {
  Incident incident = 1;
}

message IncidentEvent {
  string id = 1;
  google.protobuf.Timestamp timestamp = 2;
  string incident_id = 3;
  IncidentState state = 4;
  string reason = 5;
}
```

---

## Alerts (alert.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";
import "google/protobuf/empty.proto";
import "registry/v1/common.proto";

service AlertService {
  rpc ListAlertPolicies(ListAlertPoliciesRequest) returns (ListAlertPoliciesResponse);
  rpc CreateAlertPolicy(CreateAlertPolicyRequest) returns (CreateAlertPolicyResponse);
  rpc UpdateAlertPolicy(UpdateAlertPolicyRequest) returns (UpdateAlertPolicyResponse);
  rpc DeleteAlertPolicy(DeleteAlertPolicyRequest) returns (google.protobuf.Empty);
  rpc GetNotificationChannel(GetNotificationChannelRequest) returns (GetNotificationChannelResponse);
  rpc CreateNotificationChannel(CreateNotificationChannelRequest) returns (CreateNotificationChannelResponse);
  rpc UpdateNotificationChannel(UpdateNotificationChannelRequest) returns (UpdateNotificationChannelResponse);
  rpc TestNotificationChannel(TestNotificationChannelRequest) returns (TestNotificationChannelResponse);
  rpc ListNotificationChannels(ListNotificationChannelsRequest) returns (ListNotificationChannelsResponse);
}

message NotificationChannel {
  string id = 1;
  string type = 2;  // webhook, email
  string name = 3;
  bool enabled = 4;
  string description = 5;
  map<string, string> configuration = 6;  // Type-specific config
  map<string, string> retry_policy = 7;
  repeated Tag tags = 8;
  google.protobuf.Timestamp created_at = 9;
  google.protobuf.Timestamp updated_at = 10;
}

message AlertPolicy {
  string id = 1;
  string deployment_id = 2;  // Optional
  string environment_id = 3;  // Optional
  bool enabled = 4;
  repeated string notify_on = 5;  // ["unhealthy", "recovered"]
  int32 cooldown_minutes = 6;
  bool send_recovery_notification = 7;
  repeated string notification_channel_ids = 8;
  map<string, string> filters = 9;  // Optional filtering
  google.protobuf.Timestamp created_at = 10;
  google.protobuf.Timestamp updated_at = 11;
}

message ListAlertPoliciesRequest {
  PaginationRequest pagination = 1;
  string deployment_id = 2;  // Optional
  string environment_id = 3;  // Optional
}

message ListAlertPoliciesResponse {
  repeated AlertPolicy policies = 1;
  PaginationResponse pagination = 2;
}

message CreateAlertPolicyRequest {
  string deployment_id = 1;  // Optional
  string environment_id = 2;  // Optional
  bool enabled = 3;
  repeated string notify_on = 4;
  int32 cooldown_minutes = 5;
  bool send_recovery_notification = 6;
  repeated string notification_channel_ids = 7;
  map<string, string> filters = 8;
}

message CreateAlertPolicyResponse {
  AlertPolicy policy = 1;
}

message UpdateAlertPolicyRequest {
  string id = 1;
  bool enabled = 2;
  repeated string notify_on = 3;
  int32 cooldown_minutes = 4;
  bool send_recovery_notification = 5;
  repeated string notification_channel_ids = 6;
  map<string, string> filters = 7;
}

message UpdateAlertPolicyResponse {
  AlertPolicy policy = 1;
}

message DeleteAlertPolicyRequest {
  string id = 1;
}

message CreateNotificationChannelRequest {
  string type = 1;  // webhook, email
  string name = 2;
  string description = 3;
  map<string, string> configuration = 4;
  map<string, string> retry_policy = 5;
  repeated Tag tags = 6;
}

message CreateNotificationChannelResponse {
  NotificationChannel channel = 1;
}

message UpdateNotificationChannelRequest {
  string id = 1;
  string name = 2;
  string description = 3;
  bool enabled = 4;
  map<string, string> configuration = 5;
  map<string, string> retry_policy = 6;
  repeated Tag tags = 7;
}

message UpdateNotificationChannelResponse {
  NotificationChannel channel = 1;
}

message GetNotificationChannelRequest {
  string id = 1;
}

message GetNotificationChannelResponse {
  NotificationChannel channel = 1;
}

message ListNotificationChannelsRequest {
  PaginationRequest pagination = 1;
  string type_filter = 2;  // Optional
  bool enabled_only = 3;
}

message ListNotificationChannelsResponse {
  repeated NotificationChannel channels = 1;
  PaginationResponse pagination = 2;
}

message TestNotificationChannelRequest {
  string channel_id = 1;
}

message TestNotificationChannelResponse {
  bool success = 1;
  string message = 2;
}
```

---

## Events (event.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "google/protobuf/timestamp.proto";
import "registry/v1/common.proto";

service EventService {
  rpc ListEvents(ListEventsRequest) returns (ListEventsResponse);
  rpc WatchEvents(google.protobuf.Empty) returns (stream Event);
}

message Event {
  string id = 1;
  string type = 2;  // ServiceCreated, IncidentOpened, etc
  google.protobuf.Timestamp timestamp = 3;
  string resource_type = 4;  // Service, Instance, Incident, etc
  string resource_id = 5;
  string environment_id = 6;  // Optional
  string service_id = 7;  // Optional
  string deployment_id = 8;  // Optional
  string instance_id = 9;  // Optional
  string actor = 10;  // Username or "system"
  string actor_id = 11;  // Optional
  string message = 12;
  map<string, string> changes = 13;  // Before/after for updates
  map<string, string> metadata = 14;
}

message ListEventsRequest {
  PaginationRequest pagination = 1;
  string type_filter = 2;  // Optional
  string environment_id = 3;  // Optional
  string service_id = 4;  // Optional
  string resource_type = 5;  // Optional
  google.protobuf.Timestamp since = 6;  // Optional
}

message ListEventsResponse {
  repeated Event events = 1;
  PaginationResponse pagination = 2;
}
```

---

## Registry Lookup (registry.proto)

```protobuf
syntax = "proto3";

package registry.v1;

import "registry/v1/common.proto";

service RegistryService {
  rpc ResolveService(ResolveServiceRequest) returns (ResolveServiceResponse);
  rpc ResolveEndpoint(ResolveEndpointRequest) returns (ResolveEndpointResponse);
}

// Lightweight service discovery response
message ResolveServiceRequest {
  string service_name = 1;
  string environment_key = 2;
  bool healthy_only = 3;  // Default: true
  repeated Tag tag_filters = 4;  // Optional tag-based filtering
}

message ResolvedInstance {
  string id = 1;
  string name = 2;
  string address = 3;
  int32 port = 4;
  HealthState status = 5;
  int32 latency_ms = 6;  // Latest health check latency
  repeated ResolvedEndpoint endpoints = 7;
  repeated Tag tags = 8;
}

message ResolvedEndpoint {
  string id = 1;
  string name = 2;
  Protocol protocol = 3;
  int32 port = 4;
  string path = 5;
  bool enabled = 6;
}

message ResolveServiceResponse {
  string service = 1;
  string environment = 2;
  repeated ResolvedInstance instances = 3;
  int32 total_instances = 4;
  int32 healthy_instances = 5;
  int32 unhealthy_instances = 6;
}

// Single endpoint lookup
message ResolveEndpointRequest {
  string service_name = 1;
  string environment_key = 2;
  string endpoint_name = 3;  // e.g., "http", "grpc"
  bool healthy_only = 4;
}

message ResolvedEndpointInstance {
  string instance_name = 1;
  string address = 2;
  int32 port = 3;
  string url = 4;  // Computed: e.g., "https://10.20.1.15:8080"
  HealthState status = 5;
  int32 latency_ms = 6;
}

message ResolveEndpointResponse {
  repeated ResolvedEndpointInstance instances = 1;
  int32 total_instances = 2;
  int32 healthy_instances = 3;
}
```

---

## buf.yaml Configuration

```yaml
version: v1
build:
  roots:
    - api
generate:
  - remote: buf.build/grpc/go:v1.3.0
    out: gen/go
  - remote: buf.build/grpc-web/typescript:v0.3.1
    out: gen/ts
lint:
  use:
    - DEFAULT
breaking:
  use:
    - FILE
```

---

## buf.gen.yaml Configuration

```yaml
version: v1
managed:
  enabled: true
plugins:
  - remote: buf.build/protocolbuffers/go:v1.32.0
    out: gen/go
  - remote: buf.build/grpc-go:v1.3.0
    out: gen/go
  - remote: buf.build/grpc-web/typescript:v0.3.1
    out: gen/ts
    opt:
      - import_style=commonjs
```

---

## API Design Notes

### Naming Conventions

- Service methods follow standard CRUD: Create, Get, List, Update, Delete
- Stream methods prefixed with "Watch"
- Request messages suffixed with "Request"
- Response messages suffixed with "Response"
- Request/response pairs for each method

### Error Handling

- Standard gRPC error codes used
- Error details in message bodies when relevant
- Client responsible for retry logic (with backoff)

### Pagination

- `PaginationRequest` with `page_size` and `page_token`
- `PaginationResponse` with `next_page_token` and `total_size`
- Default page size: 50
- Maximum page size: 1000

### Streaming

- `WatchHealthEvents`: Real-time health state changes
- `WatchIncidents`: Real-time incident lifecycle
- `WatchEvents`: Real-time operational events
- Reconnection with backoff on disconnect

### Metadata

- Flexible `map<string, string>` fields for type-specific configuration
- Not indexed, not queryable
- Example: HTTP check metadata = `{"expectedStatus": "[200,201]", "followRedirects": "false"}`

### Service Lookup

- `ResolveService`: Primary discovery API (returns all instances with health)
- `ResolveEndpoint`: Convenience for single-endpoint lookup
- Both return lightweight responses (no full entity details)
- Designed for service-to-service discovery (applications asking "where is this service?")

This API provides a clean, idiomatic gRPC interface for all registry operations.
