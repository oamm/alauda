# Service Discovery

Current Check, Instance and Service state aggregation is defined in [HEALTH_STATE.md](HEALTH_STATE.md). Discovery consumes that shared projection.

Alauda discovery is intentionally small: resolve a Service key in an Environment key, inspect candidates, or resolve a named Endpoint. Discovery respects enabled Instances and Endpoints and excludes known unhealthy or disabled Instances by default. Unknown health is retained as a usable candidate so a newly registered service can be discovered before its first check executes.

## Endpoint semantics

Service is the logical resource. Instance is the running/network location with a host-only address. Endpoint is a consumable interface of that Instance. EndpointKind describes that interface. Address is hostname/IP only, port is numeric, and path is optional only for HTTP/HTTPS.

Alauda discovers network endpoints. Canonical stored addresses are bare DNS names, IPv4 or IPv6, never schemes, embedded ports, credentials or paths. Registration rejects addresses that include a URI scheme, embedded port, credentials, query, fragment or path. Ports are 1..65535. Endpoint kind is explicit: `HTTP`, `HTTPS`, `GRPC`, `POSTGRES`, `REDIS`, `TCP`, `UDP`, or `CUSTOM`. Only HTTP/HTTPS have a URI scheme and optional path. gRPC, PostgreSQL, Redis, TCP, UDP and Custom return structured address + port + kind, and value formatting is host:port.

Example resolve payloads:

```json
{"service":"postgres","environment":"development","instance":{"name":"postgres-01","address":"192.168.0.109"},"endpoint":{"name":"default","kind":"POSTGRES","port":5432,"primary":true,"enabled":true},"resolvedValue":"192.168.0.109:5432"}
{"service":"api","environment":"development","instance":{"name":"api-01","address":"api.internal"},"endpoint":{"name":"default","kind":"HTTP","port":8080,"path":"/","primary":true,"enabled":true},"resolvedValue":"http://api.internal:8080/"}
```

Alauda does not manufacture technology-specific connection strings such as PostgreSQL DSNs, Redis URIs, Kafka bootstrap security settings or gRPC TLS URLs. Consumers own credentials, database names, Redis DB numbers, Kafka auth/TLS material and gRPC cleartext/TLS transport policy.

The list returns all eligible candidates. Resolve chooses Healthy first, then other policy-eligible states, ordered by Instance name within each class. Named Endpoint filters before selection; otherwise use Enabled Primary, then deterministic Endpoint-name fallback. This is deterministic selection, not load balancing.

Policies: `health=usable` (default) allows Healthy/Unknown/Degraded and excludes known Unhealthy/Disabled; `health=healthy` requires actual Healthy; `health=all` includes health states but still excludes disabled Instance/Endpoint and disabled Environment. Invalid policies fail validation. `healthy` means actual Healthy, not usable; `healthState` is explicit. No active check, monitoring disabled, or state older than the greater of five minutes and three active-check intervals means Unknown. Unknown is usable until a fresh check reports otherwise. Registration reports this same effective state, never invented health. Discovery replies use `Cache-Control: no-store`; clients may add deliberate short-lived local caching.

Checks bound to disabled/deleted Endpoints are not active checks. Invalid timestamps or timestamps more than one minute in the future also become Unknown. This prevents an unusable historical target or malformed clock value from indefinitely controlling routing.

A C# client needs one request for either the list endpoint or resolve. Prefer a structured model such as `DiscoveryEndpoint(string Address, int Port, EndpointKind Kind, string? Path)`. Compatibility helpers named like `GetFullAddress` should format HTTP/HTTPS as URI and non-URI kinds as host:port; they must not assume `http://` or invent `grpc://`/`tcp://`.
