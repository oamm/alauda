# Service Discovery

Current Check, Instance and Service state aggregation is defined in [HEALTH_STATE.md](HEALTH_STATE.md). Discovery consumes that shared projection.

Alauda discovery is intentionally small: resolve a Service key in an Environment key, inspect candidates, or resolve a named Endpoint. Discovery respects enabled Instances and Endpoints and excludes known unhealthy or disabled Instances by default. Unknown health is retained as a usable candidate so a newly registered service can be discovered before its first check executes.

Addresses use one backend builder. Input addresses are bare DNS names, IPv4 or IPv6, never schemes, embedded ports, credentials or paths. Ports are 1..65535. HTTP/HTTPS include an escaped path (default `/`); query/fragment/authority paths are rejected. IPv6 authorities are bracketed correctly. TCP/UDP use authority URIs. `grpc://host:port` is a logical cleartext gRPC authority, not an HTTP URL or automatic TLS configuration. No grpcs/TLS inference is supported; clients use the separate protocol field to choose transport.

The list returns all eligible candidates. Resolve chooses Healthy first, then other policy-eligible states, ordered by Instance name within each class. Named Endpoint filters before selection; otherwise use Enabled Primary, then deterministic Endpoint-name fallback. This is deterministic selection, not load balancing.

Policies: `health=usable` (default) allows Healthy/Unknown/Degraded and excludes known Unhealthy/Disabled; `health=healthy` requires actual Healthy; `health=all` includes health states but still excludes disabled Instance/Endpoint and disabled Environment. Invalid policies fail validation. `healthy` means actual Healthy, not usable; `healthState` is explicit. No active check, monitoring disabled, or state older than the greater of five minutes and three active-check intervals means Unknown. Unknown is usable until a fresh check reports otherwise. Registration reports this same effective state, never invented health. Discovery replies use `Cache-Control: no-store`; clients may add deliberate short-lived local caching.

Checks bound to disabled/deleted Endpoints are not active checks. Invalid timestamps or timestamps more than one minute in the future also become Unknown. This prevents an unusable historical target or malformed clock value from indefinitely controlling routing.

A C# client needs one request for either GetDiscoveryData (the list endpoint) or GetFullAddress (resolve). Configure its Environment key once, pass it as environment, and consume the server-generated address; no relationship reconstruction or additional ID lookups are needed.
