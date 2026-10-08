# Current Health State

This is the canonical current-state contract. `HealthRepository.CurrentHealth` reads active checks and their latest completed results. The Service catalog, Instance management, discovery, global Health UI, Service UI and CLI catalog use that projection. No stored `health_states` row or incident count is a fallback for missing results.

## Check

An enabled check on an active target uses its latest completed result, ordered by execution timestamp and then result ID. Success is Healthy; failure is Unhealthy. No result, an invalid/future timestamp, or a result older than the greater of five minutes and three check intervals is Unknown. A disabled check or one bound to a disabled/deleted Endpoint is excluded from Instance aggregation. Monitoring disabled at the deployment level makes the Instance Unknown, even when historical results exist. A disabled Instance is Disabled.

## Aggregate

The same `AggregateHealth` rule applies at each level: enabled active checks form the Instance state; enabled Instances in enabled Environments form the Service state. A Service without active Instances is Unknown. Disabled Instances are excluded from Service aggregation. `monitored` means an Instance has at least one active enabled check with a usable target, regardless of whether it has a result.

| States being aggregated | Result |
| --- | --- |
| None, or all Unknown | Unknown |
| All Healthy | Healthy |
| All Unhealthy | Unhealthy |
| Healthy + Unhealthy | Degraded |
| Healthy + Unknown | Degraded |
| Unhealthy + Unknown, no Healthy | Unhealthy |
| A Degraded child mixed with any other state | Aggregate its Healthy and Unhealthy components using the same rule |

Partial monitoring can therefore never make a Service fully Healthy. Two active Instances with one Unhealthy and one unmonitored return Unhealthy; one Healthy and one unmonitored return Degraded.

Enabled is configuration/lifecycle state; it is not a health result. Historical availability comes from incident/availability history and can differ from the current health state. The latest result is evaluated at request time, so stale results become Unknown without a cache invalidation dependency. A state transition writes its result, stored Instance state, incident action and `health.changed` event in one transaction. The UI refreshes after that event and polls current status every 30 seconds to detect freshness expiry.

Incidents open when an active Instance becomes Unhealthy or Degraded, and auto-resolve only when a new result makes the Instance Healthy. A manual incident resolution does not change current health. A fresh result is authoritative even when a previous incident or stored health-state row says otherwise.

Existing `failuresBeforeUnhealthy` and `successesBeforeHealthy` fields remain accepted for compatibility, but no longer delay current health transitions. The UI does not offer those controls; they should be removed from the public schema in a separate contract cleanup. Their persisted values and consecutive counters are retained for historical compatibility only.

## API and Discovery

Service catalog responses include `healthStatus`; Instance responses include `healthState` and a derived `healthy` boolean that is true only for Healthy. `GET /api/v1/health/status` returns the current Service and Instance status maps and monitoring coverage for the UI. It requires `health.read` and honors Environment restrictions. These map keys are metadata IDs for UI correlation; normal discovery still addresses Service and Environment keys.

Discovery uses the same Instance projection: `healthy` returns only Healthy; default `usable` includes Healthy, Degraded and Unknown; `all` includes known Unhealthy but never disabled lifecycle resources. Resolve prefers Healthy, then deterministic Instance name order. See [SERVICE_DISCOVERY.md](SERVICE_DISCOVERY.md).
