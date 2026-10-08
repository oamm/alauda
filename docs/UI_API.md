# UI API Integration

The UI uses the same public registration semantics as alauda: existing Service and Environment, key-addressed atomic UPSERT, non-destructive omission, explicit endpoint Primary/Enabled values and server-built addresses.

| UI workflow                               | Current contract                                                      |
| ----------------------------------------- | --------------------------------------------------------------------- |
| Environment selector/catalog              | GET /api/v1/environments, all public pages                            |
| Service catalog                           | GET /api/v1/services?environment={key}, all public pages              |
| Create Service                            | POST /api/v1/services                                                 |
| Add/register Instance with Endpoints      | POST /api/v1/services/{serviceKey}/instances, Environment key in body |
| Edit/delete Service or Environment        | Internal CatalogService/EnvironmentService RPC                        |
| Topology and Instance/Endpoint editing    | Internal DeploymentService/InstanceService/EndpointService RPC        |
| Health editing/results/operations         | Internal HealthService and browser-specific health projections        |
| Incidents, Events, notifications          | Internal operational RPC/REST handlers                                |
| Sessions, users, tokens, Application Keys | Administrative REST handlers                                          |

Registration never sends serviceId, environmentId or deploymentId. Public endpoint protocols are lowercase; the API adapter translates existing form enum values at the boundary. Server canonical endpoint addresses are used for registration output rather than recomposed by the UI. Internal topology IDs remain hidden implementation references until the remaining editing/query workflows migrate.

Public catalog pages use pageSize/pageToken/nextPageToken. The UI follows every page and guards against repeated tokens. Internal topology paging remains unchanged. Management data is never substituted with health-filtered discovery data.

API failures are typed with status/code/field errors. Problem Details and Connect error envelopes render readable messages instead of raw JSON. Existing Endpoint and Health Check conflict messages use the preserved error code. HTTP 401 still returns the browser to the sign-in boundary.

Application Key creation defaults to discovery.read, offers granular capabilities, and retains Environment restriction controls. API token creation defaults to registry.read. Existing broad read/write credentials can still be displayed, but are no longer the options offered for new credentials.

Non-admin sessions do not fetch notification channels or alert policies during catalog loading and do not show Alerts/Security administration navigation. Backend authorization remains authoritative; UI visibility is not a substitute for permissions.

Removed APIs: `/api/v1/catalog/*`, `/api/v1/discovery/services/*` and RegistryService discovery RPCs. No UI consumer used these APIs; their deletion does not affect the UI. Active internal CRUD/health RPCs cannot be deleted before their UI replacements exist. Public edit/delete routes for every resource are not yet available, so this is an explicit migration boundary, not a commitment to preserve deprecated APIs.

Development: Vite proxies /api, /registry and /openapi.json to localhost:9700 by default. Set VITE_API_PROXY_TARGET to another backend URL for an isolated preview. Docker embeds the freshly built frontend before compiling the server; native embedded builds require synchronizing web/dist into internal/webstatic/dist (the existing run-app script does this).

Regression coverage includes named registration without IDs, protocol conversion, repeated calls, canonical addresses, all catalog pages, Problem Details, session expiry, granular credential selection and non-admin catalog loading.
