# Alauda CLI

The executable is `alauda`; `registryctl` remains a compatibility name. Stable registration and discovery use Service / Environment / Instance / Endpoint / Health Check names.

## Canonical commands

```text
alauda environments list | create
alauda services create | list | register | deregister | instances | endpoints | resolve
alauda health checks list | create | run
alauda health results list
```

Bootstrap explicitly with an unrestricted registry.write credential:

```bash
alauda environments create --key stg --display-name Staging
alauda services create --name Authentication.Grpc
alauda services register --name Authentication.Grpc --environment stg --address lynx-authentication.lynx --port 81
alauda services resolve Authentication.Grpc --environment stg --output value
```

The shorthand uses the bare address as Instance name, Endpoint name `default`, kind `http`, path `/`. IPv6 and addresses that cannot form a resource key require explicit `--instance`. Repeating registration is an atomic UPSERT; IDs stay stable. Omitted mutable fields and endpoints are preserved. Only a new singleton with Primary omitted defaults to Primary. Promotion is explicit with `--primary-endpoint`; file `primary: false` is respected. See [PUBLIC_API.md](PUBLIC_API.md) for canonical reconciliation and lifecycle semantics.

## Endpoint semantics

Service is the logical resource; Instance is the running/network location; Endpoint is a consumable interface of that Instance. EndpointKind describes the interface and is passed with `--kind`. Address is hostname/IP only, port is separate, and path is accepted only for HTTP/HTTPS. PostgreSQL `db.internal:5432`, gRPC `lynx-authentication.lynx:81`, and HTTP `http://api.internal:8080/api` are value outputs, not stored addresses.

Path applies only to HTTP/HTTPS. gRPC, PostgreSQL, Redis, TCP, UDP and Custom registration omit it automatically; explicitly supplying a nonempty incompatible path fails before submission. The same rule applies to repeated endpoints and registration files when kind is specified; partial file updates resolve omitted kinds on the server. `--protocol` and file `protocol` are not supported.

```sh
alauda services register --name PostgreSQL --environment development --instance postgres-01 --address 192.168.0.109 --kind postgres --port 5432
alauda services register --name Web --environment development --address web.internal --kind http --port 8080 --path /api
```

Resolve `--output value` prints HTTP/HTTPS as URIs and non-URI endpoint kinds as `host:port`: PostgreSQL `192.168.0.109:5432`, Redis `redis.internal:6379`, gRPC `lynx-authentication.lynx:81`, TCP/UDP `host:port`. Alauda does not generate database, Redis, Kafka or gRPC transport connection strings; applications combine the discovered endpoint with their own credentials, database names, TLS policy and secrets. Internal numeric endpoint commands use HTTP=1, HTTPS=2, GRPC=3, TCP=4, UDP=5, POSTGRES=6, REDIS=7, CUSTOM=8.

```bash
alauda services register --name Authentication.Grpc --environment stg --instance auth-01 --address lynx-authentication.lynx --endpoint default,http,81,/ --endpoint metrics,http,9090,/metrics
alauda services register --file service.yaml
alauda services list --environment stg --page-size 50 --output json
alauda services instances Authentication.Grpc --environment stg
alauda services endpoints Authentication.Grpc --environment stg --instance auth-01
alauda services deregister --name Authentication.Grpc --environment stg --instance auth-01 --yes
alauda health checks create --service Authentication.Grpc --environment stg --instance auth-01 --name alive --endpoint default --type http
alauda health checks list --service Authentication.Grpc --environment stg --instance auth-01
alauda health checks run --service Authentication.Grpc --environment stg --instance auth-01 --name alive
alauda health results list --service Authentication.Grpc --environment stg --from 2026-10-01T00:00:00Z
```

Management lists use registry routes, not discovery; disabled/unhealthy resources remain manageable. Deregistration is idempotent and retains history, endpoints and checks. Re-registration restores identity while preserving omitted state.

## Files and precedence

Files use the REST registration DTO plus a top-level `service` routing key. JSON or a single YAML document is supported; unknown fields are rejected. Files preserve tags, metadata, enabled, primary and replaceEndpoints. Explicit flags override corresponding file fields; a file Environment overrides ALAUDA_ENVIRONMENT, which overrides nonsecret config and defaults. File endpoint arrays are replaced only when repeated --endpoint values are explicitly supplied. Endpoint field overrides require exactly one submitted endpoint; ambiguous multi-endpoint overrides fail validation. Repeated --endpoint accepts name,kind,port[,path]; a single name requires --port. Legacy --endpoints is deprecated.

Example:

```yaml
service: Authentication.Grpc
environment: stg
instance:
  name: auth-01
  address: lynx-authentication.lynx
endpoints:
  - name: default
    kind: http
    port: 81
    path: /
    primary: true
    enabled: true
```

## Authentication and configuration

Use --server or ALAUDA_URL, and --token or ALAUDA_TOKEN (REGISTRY_TOKEN is a compatibility fallback). Tokens are resolved at execution, never printable help defaults, never stored by the CLI, and redacted from final errors. Prefer environment injection over command-line secrets, which may be visible in process listings and shell history. Username/password login remains for interactive legacy administration, not required for automation. Discovery clients should use discovery.read-only Application Keys, optionally Environment-restricted.

A modest optional nonsecret config defaults to ~/.config/alauda/config.yaml; override with --config. Only server, environment, output are accepted. Secret fields are rejected. No credential store or contexts are implemented. Server precedence: explicit flag > ALAUDA_URL > config > localhost:9700. Environment precedence: explicit flag > registration file > ALAUDA_ENVIRONMENT > config. Output: explicit flag > config > pretty.

## Output and exit codes

Public query output supports --output table, json, yaml. Resolve additionally supports value, printing the transport value: HTTP/HTTPS URI, otherwise host:port. Pretty JSON remains for compatibility; legacy protobuf commands retain their JSON/YAML/pretty renderers rather than the public table layouts. Unknown formats fail validation. Machine-readable stdout contains data only; errors and deprecation/page diagnostics use stderr. --quiet suppresses ancillary diagnostics and registration/bootstrap success responses, not requested query data. Resolve value still prints under --quiet.

Public lists accept pageSize/pageToken through --page-size/--page-token where shown in help. Use nextPageToken to continue; the CLI does not silently fetch every page.

| Exit | Meaning                         |
| ---- | ------------------------------- |
| 0    | Success                         |
| 2    | Validation or CLI usage         |
| 3    | Authentication                  |
| 4    | Permission denied               |
| 5    | Resource not found              |
| 6    | Conflict or failed precondition |
| 7    | No eligible discovery candidate |
| 8    | Network or server failure       |

## Compatibility boundary

Singular environment/service/deployment/instance/endpoint, health-check, and old health create/list/run are legacy UUID-based workflows with deprecation warnings. Auth, audit-log, incidents, events, alert-policy and notification-channel remain operational/administrative commands. Legacy read/write/admin credentials remain supported through capability mapping; Environment restrictions are never bypassed for compatibility. Use --help for the exact legacy flags and [API.md](API.md) for the internal/administrative boundary.
