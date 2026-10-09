# Alauda Public API

The stable public model is `Service -> Environment -> Instance -> Endpoint -> Health Check`. Deployment records, UUID relationships, and repository identifiers are internal implementation details. Service catalog records include canonical `healthStatus`; Instance records include canonical `healthState` and `healthy=true` only for Healthy. See [HEALTH_STATE.md](HEALTH_STATE.md).

## Authentication

Send an API token or application key as `Authorization: Bearer <token>`. Discovery clients should receive only `discovery.read`, optionally restricted to permitted environments. Capabilities include `registry.read`, `registry.write`, `health.read`, `health.write`, `health.execute`, `incident.read`, `incident.resolve`, `events.read`, and `admin`.

OpenAPI and Swagger require authentication and any read capability; discovery-only keys can read the contract without administrative privileges.

Legacy `read` maps to public read capabilities; legacy `write` maps to registry/health mutations and operational execution. `admin` permits all capabilities, but never overrides Environment restrictions. Audit data and notification-channel administration require admin. Notification testing is POST-only.

Environment restrictions apply at the backend boundary, including legacy REST and Connect RPCs. Catalog restrictions are applied before pagination. Scoped credentials must supply an explicit authorized Environment or scoped resource filter for legacy operations that cannot enumerate safely; global mutations and notification administration are unavailable to Environment-scoped credentials. Public discovery always requires an Environment key. Existing legacy contracts remain, but authorization bypasses are not compatibility guarantees.

## Registration

`POST /api/v1/services/{serviceKey}/instances` requires an existing Service and Environment. Registration resolves both by key and returns `404` with `service_not_found` or `environment_not_found` when either is missing.

```json
{
  "environment": "stg",
  "instance": {"name": "authentication-01", "address": "lynx-authentication.lynx"},
  "endpoints": [{"name": "default", "protocol": "http", "port": 81, "path": "/", "primary": true}]
}
```

Registration is an atomic UPSERT keyed by Service + Environment + Instance name. Matching endpoints are updated, new endpoints are created, and omitted endpoints are preserved. Omitted mutable fields preserve stored values, including Enabled, description, tags and metadata; explicit empty strings/maps clear those fields where valid. Null is not a clear operation. New resources require address/protocol/port; Enabled defaults true. Set `replaceEndpoints: true` to retire omitted endpoints. Responses contain the full active endpoint set.

Concurrent UPSERTs are serialized transactionally; later successful updates win for the fields they supply. SQLite busy/locked snapshots retry the complete transaction with a bounded backoff; exhaustion returns temporarily_unavailable, never a raw constraint or duplicate resource. A response/transport failure after commit can be retried safely with the same identity.

For a new Instance, one endpoint with Primary omitted defaults Primary=true. Incremental singleton updates never infer promotion. Explicit true atomically demotes the previous Primary; explicit false is respected. Zero Primary endpoints are permitted and discovery uses deterministic endpoint-name fallback. `mode` may be omitted or `upsert`; other modes are rejected. Service/Environment must already exist; disabled Environments cannot register or discover.

Deregistration soft-deletes only the Instance, preserving endpoints, checks and history. Repeated deletion returns 204. Re-registration restores the same identity and retained resources without re-enabling explicitly disabled resources; explicit replacement retires omitted endpoints. Checks bound to inactive endpoints are never executed through the Instance's fallback port. History is intentionally retained. Internal association restoration is automatic and does not appear in the public contract.

## Discovery

Endpoint protocols are `http`, `https`, `tcp`, `udp`, and `grpc`. Only HTTP/HTTPS support an optional path, without query, fragment, authority or control characters. TCP/UDP/gRPC require an empty or omitted path; incompatible supplied paths return `validation_failed` with a field error. Protocol changes clear incompatible retained paths when Path is omitted. Health Check path overrides are separate configuration, not generic gRPC service/method paths.

Discovery and endpoint-management responses omit Path for non-path protocols, including legacy stored values, without rewriting those records. Existing representations remain `http://host:port/path`, `https://host:port/path`, `tcp://host:port`, `udp://host:port`, and logical cleartext `grpc://host:port`. Ports must be 1..65535; no `:0` or irrelevant trailing slash is generated. No new gRPC TLS scheme is introduced.

`GET /api/v1/discovery/{serviceKey}?environment=stg` returns all enabled instances and endpoints. Known unhealthy and disabled health states are excluded by default; unknown health is usable until a check reports otherwise. Add `health=all` to inspect all enabled candidates.

`GET /api/v1/discovery/{serviceKey}/resolve?environment=stg&endpoint=grpc` returns one address. A named endpoint is selected when supplied; otherwise the enabled primary endpoint is selected, falling back to the first enabled endpoint in deterministic name order. Healthy Instances rank before other eligible states; ties use Instance name order. This is deterministic selection, not load balancing. See [SERVICE_DISCOVERY.md](SERVICE_DISCOVERY.md) for the canonical health policy and freshness rules.

`DELETE /api/v1/services/{serviceKey}/instances/{instanceName}?environment=stg` deregisters by human-readable identity.

## Bootstrap and management

Explicit bootstrap uses `POST /api/v1/services` with `name`, optional `displayName`, `description`, `tags`, `metadata`; and `POST /api/v1/environments` with `key`, optional `name`, `description`, `tags`. Both return 201 and require `registry.write` on an unrestricted credential. Registration never creates either implicitly.

Management is independent of discovery eligibility:

- `GET /api/v1/services?environment=stg`
- `GET /api/v1/services/{serviceKey}?environment=stg`
- `GET /api/v1/services/{serviceKey}/instances?environment=stg`
- `GET /api/v1/services/{serviceKey}/instances/{instanceName}?environment=stg`
- `GET /api/v1/services/{serviceKey}/instances/{instanceName}/endpoints?environment=stg`
- `GET /api/v1/environments`
- `GET /api/v1/environments/{environmentKey}`
- `GET /api/v1/environments/{environmentKey}/services`
- `GET /api/v1/environments/{environmentKey}/services/{serviceName}`

Management includes disabled active resources, excludes soft-deleted resources, and requires `registry.read`. Missing named Instances return `instance_not_found`, never an unfiltered list. List pagination uses `pageSize` (1..200, default 50), `pageToken`, and top-level `nextPageToken`. Tokens are offsets returned by the server; pagination is not a snapshot during concurrent changes. Environment-scoped catalog predicates run before pagination. Canonical Environment responses retain the old nested `pagination` field temporarily as an additive compatibility bridge.

Health Check routes use the same resource keys:

- `GET|POST /api/v1/services/{serviceKey}/instances/{instanceName}/health-checks?environment=stg`
- `POST /api/v1/services/{serviceKey}/instances/{instanceName}/health-checks/{checkName}/run?environment=stg`
- `GET /api/v1/services/{serviceKey}/health-results?environment=stg`

Create accepts `name`, `endpoint`, `type` (http/tcp/dns), Enabled, description, tags/metadata and threshold/interval fields documented in OpenAPI. Defaults: enabled=true, intervalSeconds=30, timeoutSeconds=5, failuresBeforeUnhealthy=3, successesBeforeHealthy=2. The Endpoint is resolved internally. Read/write/execute require `health.read` / `health.write` / `health.execute`, respectively. Results accept `instance`, `endpoint`, `check`, `from`, `to` plus public pagination; timestamps are UTC RFC3339. History remains visible after deregistration. Disabled targets and disabled Environments cannot execute checks; scheduled monitoring also respects monitoring-disabled state.

HTTP checks inherit HTTP/HTTPS transport from their Endpoint. Use tcp for a TCP-connect liveness check on a gRPC Endpoint; this does not claim application-level gRPC health-protocol verification. Explicit zero/negative timing and threshold values fail validation rather than silently applying defaults.

## Errors and compatibility

Public REST failures use `application/problem+json` with type/title/status/code/detail and optional field-level `errors`. Stable codes include `validation_failed`, `service_not_found`, `environment_not_found`, `instance_not_found`, `endpoint_not_found`, `no_healthy_instance`, `environment_disabled`, `authentication_required`, `permission_denied`, `resource_conflict`, `rate_limited`, `temporarily_unavailable`, and `server_error`. Internal exception/SQL text is not a public response. Unsupported JSON fields and trailing documents are rejected. Keys use 1..128 ASCII letters/digits/dot/underscore/hyphen and are case-sensitive. Environment display names are not aliases. Paths are decoded once by the HTTP server; slash/percent are not supported key characters. Request body limit is1MiB; audit logging does not truncate handler input.

The obsolete `/api/v1/catalog/*`, `/api/v1/discovery/services/*` and RegistryService discovery RPCs have been removed, including their unused implementations and generated discovery contract. They return 404; no compatibility promise remains. Connect RPCs still used for UI editing, topology and operations are internal contracts, not the public integration API. The UI uses public key-addressed registration and public paginated catalog reads. New integrations should use this document and `/openapi.json`.
