# Alauda Public API / CLI Remediation

Release verification date: 2026-10-08. Source of truth: the current handlers, repositories, CLI, and authenticated regression tests. This implements the completed audit backlog; it is not a domain-model rewrite.

Subsequent UI integration update: compatibility preservation was explicitly withdrawn. The legacy public catalog/discovery REST facade and RegistryService RPC, generated contract and unused repository have now been deleted. The UI uses public catalog paging and key-addressed UPSERT registration; granular credential scopes, typed errors and non-admin catalog loading have been updated. Active internal editing/health/operational RPCs remain UI dependencies. See UI_API.md for the current boundary. The phase and image verification record below describes the preceding release snapshot, not a promise to retain its removed APIs.

## Public API readiness

**READY for the documented Service Registry / Service Discovery workflow**, after explicit Service and Environment bootstrap. No CRITICAL/HIGH audit finding remains open. This does not claim Consul wire compatibility, load balancing, or replacement of every Consul feature.

The stable integration model is Service -> Environment -> Instance -> Endpoint -> Health Check. Existing repositories, SQLite transactions, Application Keys, domain entities, and health execution remain in use. Internal associations are resolved/restored behind the public facade.

## Findings

| Finding | Severity | Status | Release classification | Correction and evidence |
| --- | --- | --- | --- | --- |
| F01 Credential disclosure | CRITICAL | CLOSED | BLOCKER | Execution-time token resolution, blank flag default, error redaction; sentinel help/usage/error tests. |
| F02 Environment bypass | CRITICAL | CLOSED | BLOCKER | Shared resource-boundary authorization, repository predicates before paging; REST and JSON/protobuf/framed RPC matrix. |
| F03 Primary reconciliation | HIGH | CLOSED | BLOCKER | Transaction demotes existing Primary before writes; ordering, promotion, concurrency, rollback tests. |
| F04 Invalid addresses | HIGH | CLOSED | BLOCKER | Shared protocol-aware host/path/port builder; IPv4/IPv6, scheme, path and boundary tests. |
| F05 Misleading health | HIGH | CLOSED | BLOCKER | Derived health state, explicit policies, Healthy-first ordering; state/eligibility regressions. |
| F06 Excessive read privilege/side effects | HIGH | CLOSED | BLOCKER | Granular capabilities; audit/notification administration requires admin; notification test POST only. |
| F07 Incomplete OpenAPI | HIGH | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Typed stable DTOs, errors, enums, scopes, examples and actual routes; document and response/example validation. |
| F08 Discovery used for management | HIGH | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Separate key-addressed registry queries; disabled/endpointless resources visible; missing Instance error. |
| F09 Destructive omissions | HIGH | CLOSED | BLOCKER | Presence-aware DTOs, preserved fields/maps, explicit Primary false, explicit replacement; reconciliation tests. |
| F10 Error inconsistency | HIGH | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Problem Details across public REST middleware/handlers; typed storage classification; safe legacy Connect messages. |
| F11 CLI integer overflow | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Original int64 port validated before narrowing; simple/tuple/file overflow tests. |
| F12 File/config precedence | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Shared DTO, strict file parsing and flags > file > environment > nonsecret config; precedence tests. |
| F13 Scripting output/exits | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Validated formats, nil-safe tables, requested data under quiet, stderr diagnostics, category exit codes. |
| F14 Key-addressed bootstrap/Health | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Explicit Service/Environment creation and named Health Check list/create/run/results; authenticated tests. |
| F15 Public/internal boundary | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Public route documentation, deprecated legacy OpenAPI operations/CLI commands, retained RPC consumers. |
| F16 Pagination/membership | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Public pageSize/pageToken/nextPageToken, pre-page authorization, direct membership query beyond 200 records. |
| F17 Deregistration lifecycle | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Idempotent soft-delete, retained history/identity restoration, inactive check targets excluded. |
| F18 Ignored mode/retired association | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Only upsert accepted; retired internal association restored; strict body and restore tests. |
| F19 Disabled/stale monitoring | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Disabled Environment exclusion, explicit Unknown/freshness rules, active-check filtering; health regressions. |
| F20 Regression gaps | MEDIUM | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Dedicated security, registration, discovery, lifecycle, CLI and schema suites added. |
| F21 Loose paths/iteration/cache | LOW | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Strict suffixes, single decoding, rows.Err handling, no-store; iteration failure test. |
| F22 Intermittent integration failure | LOW | CLOSED | REQUIRED BEFORE PUBLIC RELEASE | Reproduced Windows SQLite directory-cleanup failure, bounded owned-fixture cleanup, repeated green suites. |

## Phase closure record

The initial plan prioritized security and transactional correctness before contract breadth. Each phase used focused tests before proceeding. Later regression/release work added further coverage without weakening earlier behavior.

| Phase | Findings/behavior closed | Main changed components | Tests/results | Compatibility impact / remaining blocker at that point |
| --- | --- | --- | --- | --- |
| A Security | F01, F02, F06: token safety, global Environment boundary, no read-side notification execution | CLI main/common; auth middleware/RBAC/audit; API environment authorization; scoped storage lists; alert REST | CLI security, auth, scoped REST/RPC tests passed | Legacy broad scopes mapped; insecure access intentionally denied. Registration/discovery blockers remained. |
| B Reconciliation | F03, F09, F18: atomic promotion, presence-aware patches, restoration, supported mode | shared contract DTO; runtime repository; public facade | Promotion, omission, replacement, restore, concurrent UPSERT and rollback tests passed | Legacy RegisterRuntime remains create-only; public UPSERT behavior corrected. Address/health blockers remained. |
| C Discovery/address | F04, F05, F11, F19: canonical address, truthful health, explicit policies, freshness | address package; API facade; discovery-health SQL; registry/health repositories; executor; CLI parser | Address/health policy, disabled resources, overflow and legacy discovery tests passed | Invalid/disabled/unusable legacy candidates now excluded. Contract/CLI blockers remained. |
| D Contract consistency | F10, F21: typed errors, strict decoding/routes, intact body, iteration handling | problem package; API error wrapper/helpers; auth middleware/audit; public facade | Problem Details, storage failure, >64KiB body, unknown/trailing JSON, rows iteration tests passed | Public error bodies now structured; Connect keeps its wire envelope. Management/OpenAPI blockers remained. |
| E Management | F08, F14, F16: named management/bootstrap/Health, independent of discovery | public management/health/results handlers; REST resources; Service membership query; CLI commands | Bootstrap, disabled-resource management, key Health execution/history, beyond-first-page tests passed | Additive named routes; existing ID routes retained. CLI/schema blockers remained. |
| F Least privilege | F02, F06: discovery-only key and legacy capability mapping | auth RBAC/middleware; backend resource boundary | Discovery-only allowed/denied tests, admin denial, resource-reference matrix passed | read/write continue via documented mapping; admin cannot bypass Environment scope. CLI/schema blockers remained. |
| G CLI | F12, F13, F15: grammar, files/config, formats, quiet, tables, exits | CLI common/main/public commands and legacy deprecations | Precedence, output, exit/redaction, malformed-server and authenticated workflow tests passed | registryctl alias/legacy commands retained; legacy warnings go to stderr. Lifecycle/schema verification remained. |
| H Lifecycle | F17: retained checks/history, restore stable identity, repeated delete | runtime/health repositories; public health/results/facade | Real HTTP check, inactive endpoint, retained results, restore and repeated deregistration passed | Idempotent 204 and intentional soft-delete documented. Paging/schema verification remained. |
| I Pagination | F16: consistent public paging and scoped catalog membership | public management/REST resources; repository scope predicates; CLI paging | >200 membership, authorization-before-page-size=1, management paging passed | Existing Environment nested pagination temporarily retained additively. Schema/release verification remained. |
| J OpenAPI | F07: accurate typed public document | OpenAPI builder/tests; DTOs; kin-openapi validation dependency | Document, examples, actual registration/resolve response schemas, mounted routes passed | Legacy operations explicitly deprecated, not removed. Release repetition remained. |
| K Deprecation | F15: explicit PUBLIC/LEGACY/INTERNAL/ADMIN boundaries | CLI deprecation fields; OpenAPI; canonical API/CLI/runtime/migration docs | Modern/legacy command tests passed | Known frontend RPC and old clients retained; future removal requires migration. Release repetition remained. |
| L Regression/release | F20, F22 and transport hardening | API/CLI regression suites, scoped RPC matrix, owned SQLite fixtures | Full suite, integration x5, API x20, build and diff checks passed | Framed Connect JSON preserved for allowed requests; invalid/compressed frames fail closed. No release blocker remains. |

## Verified registration and discovery contract

Registration requires pre-existing Service and Environment keys. It never silently creates typo'd resources. The public identity is Service key + Environment key + Instance name. Shorthand uses the bare hostname as Instance name, endpoint default, protocol http, port supplied by the caller, path /.

UPSERT preserves omitted fields and omitted endpoints. Explicit values update; explicit empty maps clear maps; null does not clear fields. replaceEndpoints retires omitted endpoints. A new singleton with Primary omitted becomes Primary; incremental singleton submissions do not promote. Explicit Primary true demotes the previous Primary in the same transaction. Explicit false is respected. No Primary uses enabled endpoint-name ordering, not arbitrary database order.

Discovery requires Service key + Environment key and one HTTP request. List returns all eligible candidates. Resolve selects a named Endpoint, otherwise enabled Primary, otherwise enabled endpoint-name fallback. Eligible Healthy Instances precede Unknown/Degraded, with Instance name as tie-breaker. No round-robin or load balancing is claimed.

health=usable is default and excludes known Unhealthy/Disabled while allowing Unknown/Degraded. health=healthy requires actual Healthy. health=all still excludes disabled lifecycle resources. Disabled Environment, Instance and Endpoint are never normal candidates. No active check, disabled monitoring, or stale state becomes Unknown. Freshness rules and protocol/gRPC semantics are canonical in SERVICE_DISCOVERY.md.

Deregistration returns 204 repeatedly, soft-deletes the Instance, preserves resources/history, and restoration reuses identity. Inactive endpoint-bound checks cannot execute through a port-0 fallback.

## Final public route inventory

All paths below begin with /api/v1. Bearer credentials use granular capabilities; Environment scope is enforced independently of capability.

| Method | Path | Capability |
| --- | --- | --- |
| GET / POST | /services | registry.read / registry.write |
| GET | /services/{serviceKey} | registry.read |
| GET / POST | /services/{serviceKey}/instances | registry.read / registry.write |
| GET / DELETE | /services/{serviceKey}/instances/{instanceName} | registry.read / registry.write |
| GET | /services/{serviceKey}/instances/{instanceName}/endpoints | registry.read |
| GET / POST | /services/{serviceKey}/instances/{instanceName}/health-checks | health.read / health.write |
| POST | /services/{serviceKey}/instances/{instanceName}/health-checks/{checkName}/run | health.execute |
| GET | /services/{serviceKey}/health-results | health.read |
| GET / POST | /environments | registry.read / registry.write |
| GET | /environments/{environmentKey} | registry.read |
| GET | /environments/{environmentKey}/services | registry.read |
| GET | /environments/{environmentKey}/services/{serviceName} | registry.read |
| GET | /discovery/{serviceKey} | discovery.read |
| GET | /discovery/{serviceKey}/resolve | discovery.read |

Resource operations beneath a Service require environment; catalog listing may enumerate authorized Environment membership. Bootstrap requires an unrestricted registry.write credential. OpenAPI/Swagger are authenticated and available to any read capability. PUBLIC_API.md and generated OpenAPI contain exact schemas, query parameters, error codes, pagination and examples.

## Final CLI command tree

```text
alauda environments list | create
alauda services create | list | register | deregister | instances | endpoints | resolve
alauda health checks list | create | run
alauda health results list
```

Legacy singular environment/service/deployment/instance/endpoint, health-check and old health create/list/run remain deprecated. Existing auth/audit/incidents/events/alert/notification operational commands remain; they are not the stable registration/discovery grammar.

Public queries support table/json/yaml; resolve also value. Quiet preserves requested query output and suppresses ancillary messages. Errors/diagnostics use stderr. Exit codes: 0 success, 2 validation, 3 authentication, 4 permission, 5 not found, 6 conflict/precondition, 7 no eligible discovery candidate, 8 network/server failure. Flags/file/environment/nonsecret config precedence and exceptions are documented in CLI.md. No bearer token is stored by config or displayed as a flag default.

## Verification

Executed successfully after the final runtime changes:

```text
go test ./... -count=1
go test ./... -count=10
go test ./tests/integration -count=5
go test ./internal/api -count=20
go test ./internal/api -run OpenAPI -count=1
go build ./...
git -c core.safecrlf=false diff --check
```

The Environment authorization matrix covers 176 resource-reference cases across JSON, protobuf, framed JSON and framed protobuf. Allowed framed JSON is separately verified to preserve the complete request body. Authenticated CLI tests cover bootstrap, shorthand repeated registration/stable IDs, management, value/JSON/YAML/table/quiet, missing resources, authentication, and repeated deregistration.

### Packaged Consul replacement workflow

Executed against a disposable server using the final Docker image and its bundled alauda executable, with an unrestricted registry.read/registry.write Application Key for registration and a discovery.read-only key restricted to stg for resolution:

```text
alauda services register --name Authentication.Grpc --environment stg --address lynx-authentication.lynx --port 81 --output json
alauda services register --name Authentication.Grpc --environment stg --address lynx-authentication.lynx --port 81 --output json
alauda services resolve Authentication.Grpc --environment stg --output value
```

Both registrations and resolution exited 0. Both registrations returned Instance ID ef151da7-9276-4dd8-9ca3-8705aed7e92f and default Endpoint ID eb091776-5655-413c-939d-78fbff20754e. Resolve stdout was exactly http://lynx-authentication.lynx:81/ followed by a newline, also under --quiet. No Deployment fields or credential appeared in public output.

The same discovery-only key resolving prod exited 4. Registry listing and registration with that key exited 4. Missing registration flags exited 2. Root help, command help, validation usage and permission errors were checked for the actual disposable credential; it was absent.

Production image built locally as alauda/service-registry:public-api-v1 and public-api-candidate, image ID sha256:878f1ff91d7dc9e6011efb7b6dc1b437c000919319237b857862ef39796ed93f. It contains the registry server, alauda CLI and registryctl compatibility alias, running as a non-root user. A Windows executable was built as bin/alauda.exe. No remote registry push was performed; a publishing destination and credentials must be chosen explicitly. Disposable verification containers were removed after verification.

The first final full run reproduced a Windows TempDir removal failure after successful integration assertions. Its complete JSON log was retained outside the repository as Alauda-release-full-first-failure.jsonl. A later full run caught the same removal failure in the alert fixture after its assertions passed; the full command output was captured. All file-backed API, integration, alert, health-storage and auth-security database fixtures now share internal/testutil's verified owned-directory cleanup with bounded post-close removal retries; persistent cleanup errors still fail tests. Ten consecutive full-suite runs then passed, with complete JSON output retained as Alauda-release-full-repeat.jsonl. This changes fixture cleanup, not production SQLite persistence or assertions.

## Compatibility and remaining debt

No legacy RPC or backend entity was removed. Legacy read/write/admin keys remain mapped. Authorization tightening, POST-only notification execution, rejection of invalid addresses/modes/body fields, and structured public errors are intentional safety corrections, not compatibility bypasses. Legacy discovery now honors enabled lifecycle and effective-health safety.

Deferred after the first stable public release: migration/removal of internal UUID/RPC consumers, unification of browser/internal legacy pagination, a production C# client implementation (the common API flow is already one request), optional secure OS credential storage, and snapshot pagination if needed. No complex contexts, Consul wire compatibility, load balancing or inferred gRPC TLS support was added. These are not open F01-F22 blockers.

Canonical documentation: PUBLIC_API.md, CLI.md, SERVICE_DISCOVERY.md, RUNTIME_REGISTRATION.md, CONSUL_MIGRATION.md, API.md and DOCKER.md. Behavior descriptions live in those documents rather than a separate CLI domain model.
