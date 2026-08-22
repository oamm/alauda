# Runtime Registration

Alauda separates catalog identity from runtime topology.

```text
Service
  Deployment in Environment
    Instance
      Endpoint
```

A Service describes what an application is. An Instance describes where one running copy exists. An Endpoint describes how a particular protocol can reach that Instance.

## Concepts

`Service` is the logical catalog identity, such as `checkout`.

`Environment` is the runtime context, such as DEV, QA, STAGING, or PROD.

`Deployment` links one Service to one Environment. There must be only one Deployment for a Service + Environment pair.

`Instance` is one concrete running copy of a Deployment. Its runtime network identity is its address.

`Endpoint` is a protocol-specific interface exposed by an Instance. Endpoint owns protocol, port, path, enabled state, and optional primary status.

`HealthCheck` describes how Alauda checks an Instance or Endpoint.

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
Select Service
Select Environment
Register Instance
Add Endpoints
Optionally add Health Check
Review
Register Runtime
```

When registration is submitted, Alauda resolves the Deployment automatically:

```text
Does Service + Environment deployment exist?

YES -> reuse it
NO  -> create it
```

Then it creates the Instance and Endpoint records as one logical runtime registration.

## Composite Operation

The preferred application operation is `RegisterRuntime`.

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

Runtime registration requires:

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

Runtime registration is presented as one user action, so it should behave as one logical operation.

Local persistence should be transactional where possible:

```text
resolve/create deployment
create instance
create endpoints
commit
publish events
```

External work, including notifications or health execution, must not occur inside the database transaction.

If health-check creation is not included in the same transaction, Alauda should clearly report whether runtime registration succeeded and health-check configuration failed.

## Product Boundary

Alauda observes and catalogs runtime topology. It does not route traffic, proxy traffic, load balance, rewrite requests, configure networking, or control service-to-service traffic.
