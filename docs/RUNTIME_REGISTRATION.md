# Runtime Registration

The stable public contract is Service -> Environment -> Instance -> Endpoint -> Health Check. See [PUBLIC_API.md](PUBLIC_API.md) for the canonical REST schema, reconciliation, errors and lifecycle semantics; [CLI.md](CLI.md) documents the same model through flags and files.

Service and Environment are explicit bootstrap prerequisites. Registration does not create them on typo. The public facade resolves their exact case-sensitive keys and uses the existing repositories/transaction internally. Instance identity is Service + Environment + Instance name; Endpoint identity is Instance + Endpoint name.

## Endpoint semantics

Service is the logical resource. Instance is a running/network location with a host-only address. Endpoint is a consumable interface on that Instance. EndpointKind describes the interface; it is not a mandatory URI scheme. Address, port and optional kind-dependent path stay structured.

```bash
alauda services register --name Authentication.Grpc --environment stg --address lynx-authentication.lynx --port 81
alauda services resolve Authentication.Grpc --environment stg --output value
alauda services register --name PostgreSQL --environment development --instance postgres-01 --address 192.168.0.109 --kind postgres --port 5432
```

Registration is an atomic idempotent UPSERT. Omitted fields/endpoints preserve state. Explicit replacement retires omitted endpoints; explicit Primary promotion demotes the old Primary before writing the new one. Primary is inferred only for a new singleton when omitted. Health Check configuration is a separate operation; registration never invents a Healthy result.

Endpoint kind describes what the endpoint is, not a mandatory URI scheme. HTTP/HTTPS support path and URI value formatting. PostgreSQL, Redis, gRPC, TCP, UDP and Custom remain structured address + port + kind. Alauda does not store or generate connection strings, credentials, database names, Redis DB numbers, Kafka security settings or gRPC TLS policy; consuming applications and secret/config systems own those details.

RegisterRuntime and internal Connect RPCs remain administrative implementation details. Deployment relationships are not part of the public registration or discovery facade. Frontend editing and administrative tooling use the same EndpointKind validation rules as the public REST and CLI contracts.
