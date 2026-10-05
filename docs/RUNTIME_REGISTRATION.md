# Service Registration

Alauda separates catalog identity from runtime topology. The storage and API
model remains deployment-oriented internally, while public surfaces use a
service-address registration abstraction.

```text
Service
  Deployment in Environment
    Instance
      Endpoint
        HealthCheck

Public abstraction:

Service
  Environment
    Instance
      Address
      Endpoint(s)
      Health monitoring
```

A Service describes what an application is. An Instance describes where one running copy exists. An Endpoint describes how a particular protocol can reach that Instance.

## Concepts

`Service` is the logical catalog identity, such as `checkout`.

`Environment` is the runtime context, such as DEV, QA, STAGING, or PROD.

`Deployment` links one Service to one Environment. There must be only one Deployment for a Service + Environment pair. It is an internal implementation detail and is intentionally hidden from web users.

`Instance` is one concrete running copy of a Deployment. Its network identity is its address. Public surfaces present Instance as the concrete running location of a Service within an Environment.

`Endpoint` is a protocol-specific interface exposed by an Instance. Endpoint owns protocol, port, path, enabled state, and optional primary status.

`HealthCheck` describes how Alauda checks an Instance or Endpoint. In the frontend this is presented as optional `Health monitoring`.

## Example

```text
checkout
  PROD
    checkout-prod-01
      address: 10.0.0.10
      endpoints:
        * https :8080 /
          grpc  :5001
          http  :9090 /metrics
```

The star marks the primary endpoint. Primary endpoint is display/discovery metadata only. It is not traffic routing priority.

## Web Workflow

The standard web workflow should not require a user to create or enter Deployment IDs.

```text
From an already selected Service:

Select Environment
Add instance
Enter name, address, and details
Add one or more Endpoints
Add instance
Optionally configure Health monitoring
```

When registration is submitted, Alauda resolves the Deployment automatically;
the user does not select or create one:

```text
Does Service + Environment deployment exist?

YES -> reuse it
NO  -> create it
```

Then it creates the Instance and Endpoint records as one logical service-address
registration. Registration and health-monitoring configuration are separate
outcomes. If health configuration fails, the service address and its Endpoints
remain available and the user can configure or retry health monitoring.

## Composite Operation

The preferred application operation is `RegisterRuntime`.

All public surfaces should use this operation conceptually. The Web UI, a
future CLI, HTTP clients, and automation should differ only in presentation:

```text
Web UI / CLI / API client
          |
          v
    RegisterRuntime
          |
          v
Service -> Environment -> Address -> Endpoint(s)
          |
          v
internal Deployment -> Instance -> Endpoint records
```

## Public CLI/API Contract

The current compatible request uses `service_id`, `environment_id`, and a
nested `instance` because those fields already exist in the protobuf contract.
They remain supported. A future additive public adapter should accept stable
references and the simpler service-address shape:

```json
{
  "service": "Authentication.Grpc",
  "environment": "staging",
  "name": "authentication-01",
  "address": "lynx-authentication.lynx",
  "description": "staging authentication address",
  "endpoints": [
    { "protocol": "grpc", "port": 81, "primary": true },
    { "protocol": "http", "port": 80, "path": "/api" }
  ]
}
```

`service` resolves by exact Service name. `environment` resolves by key first,
then by unique Environment name. Missing or ambiguous references are errors;
the registration call must not create catalog records implicitly. The adapter
then calls the same `RegisterRuntime` operation, which automatically resolves
or creates the internal Deployment.

For a simple one-endpoint CLI command, the adapter may derive the endpoint name
from protocol when that is unambiguous. It may infer the first endpoint as
primary when no endpoint is marked primary. It must not infer a protocol or
runtime name when doing so could select the wrong registration.

The current operation is create-only. Repeating a request with the same runtime
name in the same Service + Environment fails on the existing Instance
uniqueness constraint; address alone is not a registration identity. A future
`reconcile` mode should match Service + Environment + Runtime name and update
the runtime/endpoints, while an explicit request ID should provide durable
replay protection. These are additive follow-ups and are not current behavior.

Current CLI/API limitations are documented in [API.md](API.md): ID-only
Service/Environment lookup, required runtime and endpoint names, internal IDs
in the response, create-only semantics, and unstructured validation errors.

```json
{
  "serviceId": "svc-checkout",
  "environmentId": "env-prod",
  "instance": {
    "name": "checkout-prod-01",
    "address": "10.0.0.10",
    "description": "Primary checkout instance"
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

The lower-level operations remain supported for CLI, automation, and advanced administration:

- `CreateDeployment`
- `CreateInstance`
- `CreateEndpoint`

## CLI Examples

Existing compatible workflow:

```bash
registryctl deployment create \
  --service-id <service-id> \
  --environment-id <environment-id>

registryctl instance register \
  --deployment-id <deployment-id> \
  --name checkout-a \
  --address 10.0.0.10 \
  --port 8080

registryctl endpoint create \
  --instance-id <instance-id> \
  --name http \
  --protocol 1 \
  --port 8080 \
  --path /
```

Preferred future endpoint-first model:

```bash
registryctl instance register \
  --deployment-id <deployment-id> \
  --name checkout-a \
  --address 10.0.0.10

registryctl endpoint create \
  --instance-id <instance-id> \
  --name http \
  --protocol https \
  --port 8080 \
  --path / \
  --primary
```

If the existing instance `--port` flag must remain for compatibility, it should be explicitly deprecated and documented as a legacy default. New workflows should place ports on Endpoints.

## Validation

Service registration requires:

- service
- environment
- instance name
- instance address

Every endpoint requires:

- name
- protocol
- valid port from `1` to `65535`

HTTP and HTTPS paths may be `/` or another valid relative path. TCP and UDP do not require paths.

At most one endpoint per Instance may be primary.

## Failure Behavior

Service registration is presented as one user action, so it should behave as one logical operation.

Local persistence should be transactional where possible:

```text
resolve/create deployment
create instance
create endpoints
commit
publish events
```

External work, including notifications or health execution, must not occur inside the database transaction.

If health-check creation is not included in the same transaction, Alauda should clearly report whether service registration succeeded and health-check configuration failed.

## Product Boundary

Alauda observes and catalogs runtime topology. It does not route traffic, proxy traffic, load balance, rewrite requests, configure networking, or control service-to-service traffic.
