# Alauda UI Hard Reset

This document is the migration control ledger for the Alauda frontend reconstruction. It is intentionally maintained in the repository so implementation decisions, behavior contracts, and verification status do not live only in chat or in a single large refactor.

## Objectives

- Rebuild the frontend presentation layer around one compact, professional, accessible Alauda design system.
- Preserve domain models, API contracts, authentication and authorization semantics, data flows, business rules, and supported application behavior.
- Replace screen-specific layout decisions with shared primitives and predictable page architectures.
- Migrate incrementally with a screen-level rollback boundary and explicit verification gates.

## Final technology stack

| Concern | Current baseline | Target | Decision |
| --- | --- | --- | --- |
| Runtime | React 18 | React 18 | KEEP |
| Language | TypeScript | TypeScript | KEEP |
| Build | Vite 5 | Vite 5 | KEEP |
| Styling | Tailwind 3 plus large `App.css` | Tailwind 3 plus Alauda tokens | REFACTOR |
| Routing | `ActiveView` state, React Router dependency unused in source | React Router where route semantics are introduced | RETAIN/REVIEW |
| Data/API | ConnectRPC helpers, `fetch`, React Query dependency present but not used by screens | Existing API helpers and data contracts | PRESERVE |
| Primitives | Local `OperationsUI` components and raw HTML controls | Radix behavior with shadcn/ui-style source components | REPLACE |
| Variants | String-concatenated CSS classes | `class-variance-authority` plus `clsx`/`tailwind-merge` | ADD |
| Icons | Text-letter placeholders and native text actions | Lucide React | ADD |
| Forms | Local `useState` and native validation | React Hook Form plus Zod at migrated form boundaries | ADD/INCREMENTAL |
| Tables | Hand-built div grids and rows | TanStack Table for structured admin data | ADD/INCREMENTAL |
| Notifications | Inline status/error strings | Sonner plus shared inline error patterns | ADD/INCREMENTAL |

No Bootstrap, Material UI, Ant Design, Chakra, Fluent UI, or other overlapping component framework will be introduced.

## Design principles

- Calm operational density: compact spacing, clear hierarchy, restrained borders, and intentional whitespace.
- One visual language: screens consume Alauda primitives and tokens rather than inventing local controls.
- Semantic clarity: state badges describe state, switches represent booleans, progress represents quantities, and links navigate.
- Information-first layouts: list, detail, configuration, history, and monitoring responsibilities remain distinct.
- Accessible by default: keyboard behavior, focus-visible states, semantic HTML, contrast, labels, and screen-reader status are primitive responsibilities.
- GitLab-inspired administration UX without cloning GitLab branding or implementation.
- No internal backend concepts such as Deployment or Runtime are exposed as normal user-facing concepts where the service model can express Service > Environment > Instance > Endpoint/Health.

## Current frontend inventory

Location: `web/`.

- Entry/auth orchestration: `src/main.tsx`, `src/App.tsx`.
- Main application state and presentation: `src/App.tsx` (5,559 lines at audit time).
- Shell: `src/components/AppShell.tsx`.
- Partial primitives: `src/components/OperationsUI.tsx`.
- Domain-specific presentation: `Cards.tsx`, `RuntimeTopology.tsx`, `AddRuntimeDialog.tsx`.
- Extracted views: `src/views/DashboardView.tsx`, `src/views/HealthWorkspace.tsx`.
- API and domain types: `src/api.ts`, `src/types.ts`.
- Styling: `src/index.css`, `src/App.css`, Tailwind config, inline/class-specific screen rules.
- Tests: `src/App.test.tsx`, `src/test/setup.ts`, `src/test/fixtures.ts`.

The current architecture centralizes nearly every query, mutation, form, modal, filter, selection, and view branch in `App.tsx`. This is the principal migration risk and the primary reason for screen-by-screen extraction rather than a cosmetic stylesheet pass.

## Current screens

The current shell exposes state-driven views rather than URL routes:

- Dashboard: metrics, incident triage, health and activity summaries.
- Services: service list, filtering, pagination, bulk selection/deletion, create flow, service detail.
- Service detail tabs: Overview, Instances, Health, Incidents, Events. Former Availability is merged into Health.
- Environments: environment list/detail and creation.
- Health: overview, checks, results, create/edit/run workflows.
- Incidents: filtered incident list, detail, resolve workflow.
- Alerts: alert policies and notification channels, create/edit/test workflows.
- Events: filtered event history and resource navigation.
- Security: Users, API tokens, Application keys, Sessions.
- Authentication: Login and mandatory password-change flow outside the authenticated shell.
- System actions: environment context selection, refresh, logout, dark-mode toggle, event stream updates.

## Current shared components

| Component | Current responsibility | Disposition |
| --- | --- | --- |
| `AppShell` | Sidebar, environment selector, toolbar, status, account action | REBUILD as application shell |
| `OperationsUI.Button` | Shared button wrapper with string classes | REPLACE with canonical Button |
| `OperationsUI.IconButton` | Icon-like button wrapper | REPLACE with Lucide-backed IconButton |
| `OperationsUI.Tabs` | Partial keyboard-aware tablist | REPLACE with Radix Tabs adapter |
| `OperationsUI.StatusBadge` | Status normalization and visual label | REFACTOR into StatusBadge system |
| `OperationsUI.PageHeader` | Page title and actions | KEEP CONCEPT, REBUILD |
| `OperationsUI.EmptyState` | Empty resource state | KEEP CONCEPT, REBUILD |
| `OperationsUI.LoadingRows` | Loading placeholder rows | REPLACE with Skeleton |
| `OperationsUI.ResourceLink` | Callback-based resource navigation | REFACTOR to semantic link |
| `OperationsUI.ActionGroup` | Action spacing wrapper | KEEP CONCEPT, tokenized |
| `Cards` | Metric and availability cards | REFACTOR into Card/Metric primitives |
| `RuntimeTopology` | Legacy service instance/endpoint presentation | REPLACE with Instances view |
| `AddRuntimeDialog` | Legacy runtime registration dialog | REPLACE with instance/create flow |
| `DashboardView` | Extracted dashboard composition | REBUILD using new page patterns |
| `HealthWorkspace` | Extracted health composition | REBUILD using new page patterns |

## Current styling approaches

- Tailwind utility classes coexist with a large semantic CSS stylesheet.
- `App.css` contains global element styling for every button/input/select/textarea plus page-specific classes.
- Light and dark themes are represented by CSS custom properties and a `.dark` shell class.
- The legacy theme uses dark navy/cyan gradients, glow shadows, and mixed panel rules that conflict with the restrained target language.
- Button implementations are split across raw `button`, `.button-*`, `.ui-button-*`, `.text-action`, `.icon-button`, and local overrides.
- Dialog implementations are split across `.dialog-backdrop`, `.modal-backdrop`, `.dialog-panel`, `.modal`, and screen-local variants.
- Tabs use `.subnav`, `.ui-tabs`, service-specific tab classes, and raw active buttons.
- Tables/lists use multiple div-grid class families (`.table`, `.table-row`, `.row`, application-key rows, service rows, health rows).
- No Radix, CVA, Lucide, React Hook Form, Zod, TanStack Table, or Sonner dependency is currently installed.

## Behavioral contracts that must be preserved

### Authentication and authorization

- Login calls the existing `login` API and establishes the same bearer/session semantics.
- Startup checks `getCurrentSession`; anonymous users see Login only.
- Mandatory password-change users remain outside the authenticated shell until completion.
- Authentication failures continue through the existing global failure handler.
- Server authorization remains authoritative; the UI may hide or disable actions but must not replace permission checks.

### Catalog and operations

- Environment selection scopes catalog and operational queries as currently implemented.
- Services support filtering, selection, pagination, create, edit, delete, bulk delete, and navigation into detail.
- Service detail preserves overview, instance/topology, availability, health, incident, and event workflows.
- Instance registration preserves endpoint configuration, primary endpoint behavior, health configuration, edit, and delete semantics.
- Health checks preserve create, edit, delete, run, result loading, state loading, and follow-up configuration behavior.
- Incidents preserve filtering, selection, detail, and resolve behavior.
- Alerts preserve policy/channel CRUD, scope selection, and notification test behavior.
- Events preserve event stream updates, filters, timestamps, and resource navigation.

### Security

- Users preserve create/list behavior and role assignment semantics.
- API tokens preserve scoped creation, expiration, and revoke behavior.
- Application keys preserve one-time secret presentation, copy behavior, and revoke behavior.
- Sessions preserve list and revoke behavior.

### State and feedback

- Loading, empty, error, retry, success, and partial-success states remain represented.
- Mutations continue to refresh the affected data and preserve current side effects/audit behavior.
- Existing ConnectRPC/fetch request shapes and API helper contracts are not changed by presentation migration.
- EventSource cleanup and environment-change cleanup remain correct.

## New design-system primitives

### Phase 1 foundation

Button, IconButton, Input, Textarea, Select, MultiSelect, Checkbox, RadioGroup, Switch, Badge, StatusBadge, Tabs, Dialog, Sheet/Drawer, DropdownMenu, Tooltip, Popover, Card, Table, Pagination, Breadcrumb, EmptyState, Alert, Skeleton, FormField, PageHeader, SectionHeader, ActionGroup, Stack, Inline, and DefinitionList.

### Canonical page patterns

- Resource list: PageHeader, filters, ResourceList/Table, create action, empty/error/loading state.
- Resource detail: ResourceHeader, metadata, shared Tabs, content, contextual actions.
- Settings/admin: PageHeader, settings navigation, focused sections, configuration lists.
- Create/edit: focused form, Cancel, primary submit action, standardized validation and feedback.
- Master/detail: bounded navigational list with a stable detail surface.

### Tokens

The target token scale is intentionally small: spacing 4/8/12/16/24/32; control heights 32/36/40; radii 4/6/8; restrained neutral surfaces; semantic success/warning/danger/info states; content widths and breakpoints defined centrally; and explicit z-index layers for shell, popover, dialog, and toast.

## Screen migration matrix

| Screen | Current issues | Behavior to preserve | New pattern | Status |
| --- | --- | --- | --- | --- |
| Authentication / Login | Separate local form, no shared form primitives, outside system but visually isolated | Login, error, loading, session establishment | AuthLayout + FormField | AUDITED |
| Password change | Separate local form and validation | Mandatory password update flow | AuthLayout + focused form | AUDITED |
| Dashboard | Mixed metrics, incidents, health, activity and raw table styling | Summary queries, navigation, triage links | Overview dashboard | AUDITED |
| Services | 5k-line branch, legacy master/detail, mixed row/button patterns | Filters, pagination, bulk actions, create/delete, selection | ResourceList + Detail | AUDITED |
| Service detail / Overview | Service summary mixed with mutation surfaces | Metadata, actions, summary data | Resource detail | AUDITED |
| Service detail / Instances | Runtime terminology and topology-specific CSS | Instance/endpoint CRUD and registration | ResourceList + topology detail | AUDITED |
| Service detail / Availability | Former separate operational screen | Availability query, windows and environment scope | Availability section in Health | MERGED INTO HEALTH |
| Service detail / Health | Separate sparse operational screens replaced | Checks, run, results, state, edit/delete, availability | Sections + resource rows + check drawer | VERIFIED |
| Service detail / Incidents | Detail/list mixed with local dialog styles | Incident filtering, detail, resolve | List/Detail | AUDITED |
| Service detail / Events | History and navigation use local rows | Event filtering and resource links | Table + filters | AUDITED |
| Environments | Inline creation and scope-dependent counts replaced | List/create/edit, key immutability, environment selection | Workspace table/mobile list + canonical dialog form | VERIFIED |
| Health | Legacy inline history and duplicated forms replaced | Overview/checks/results and create/run/edit | Shared operational sections + canonical editor/history | VERIFIED |
| Health results | Previously only bounded RPC fragments, no history browser | Scoped execution observations and result semantics | Server-query table/list + details drawer | VERIFIED |
| Incidents | Dense list/detail and screen-local actions | Filtering, resolve, navigation | ResourceList + detail | AUDITED |
| Alerts | Policy/channel modes and local forms | CRUD and channel test | Settings tabs + tables | AUDITED |
| Events | Filtered history and local table patterns | Filters, streaming, navigation | Table + filters | AUDITED |
| Security / Users | Former monolith branch and local forms removed | User list/create and roles | Shared workspace + Table/ResourceList + Dialog | VERIFIED |
| Security / API tokens | Former duplicate modal/table styles removed | Owner-scoped create, expiry, revoke | Shared workspace + Dialog + Table/ResourceList | VERIFIED |
| Security / Application keys | Former one-off secret dialog/rows removed | Create, show/copy secret once, revoke | Shared workspace + Dialog + Table/ResourceList | VERIFIED |
| Security / Sessions | Former raw rows and section-triggered loading removed | List and revoke | Shared workspace + Table/ResourceList + Dialog | VERIFIED |

Information-architecture decisions recorded during audit:

1. Service detail remains one product concept with explicit tabs: Overview, Instances, Health, Incidents, Events. Availability belongs within Health.
2. Configuration and monitoring are visually separated even when they share data.
3. Security remains a settings area with Users, API tokens, Application keys, and Sessions, all using the same primitives.
4. Login and password change remain outside the authenticated shell.
5. Internal Deployment/Runtime terminology is kept in API/backend behavior only; user-facing screens use Service > Environment > Instance > Endpoint/Health.

## Component migration matrix

| Legacy component/pattern | Replacement | Consumers | Status |
| --- | --- | --- | --- |
| Raw `button` plus `.button-*` classes | `Button` | All screens | PLANNED |
| `.icon-button` and text close actions | `IconButton` with Lucide | Dialogs, row actions, shell | PLANNED |
| `OperationsUI.Tabs`, `.subnav`, local tab buttons | Radix-backed `Tabs` | Services, Health, Alerts, Security | PLANNED |
| `StatusBadge` plus ad hoc status spans | Canonical `StatusBadge` | Services, Health, Incidents, Security | IN PROGRESS |
| `.dialog-backdrop`, `.modal-backdrop`, local modal markup | `Dialog` | Create/edit/security flows | PLANNED |
| `.panel` and local cards | `Card`, `Section`, `MetricCard` | All authenticated screens | PLANNED |
| `.table`, `.row`, local table grids | `Table`/TanStack Table | Admin and history screens | PLANNED |
| `LoadingRows` and local loading copy | `Skeleton`/`LoadingState` | All screens | PLANNED |
| Empty-state variants | `EmptyState` | All screens | PLANNED |
| `RuntimeTopology` | `ServiceInstancesView` | Services detail | PLANNED |
| `AddRuntimeDialog` | Instance creation flow | Services detail | PLANNED |
| `DashboardView` | `OperationalWorkspace` dashboard summary | Dashboard | REMOVED |
| `HealthWorkspace` | Shared sections, resource rows, canonical form/history navigation | Global Health | RECREATED / VERIFIED |
| Global/Service/post-registration health forms | `HealthCheckForm` + shared defaults/Zod schema | Global, Service, Instance, Edit, registration follow-up | REPLACED / VERIFIED |
| Inline result-history drawer | `HealthResultsPage` | Global and Service Health | REPLACED / VERIFIED |
| Monolithic `App.tsx` view branches | Screen modules plus orchestration hooks | Entire app | IN PROGRESS |

## Migration phases

| Phase | Scope | Exit gate | Status |
| --- | --- | --- | --- |
| Phase 0 — Audit | Inventory, contracts, screen/component matrices, risks | This document exists and reflects the current baseline | COMPLETE |
| Phase 1 — Design-system foundation | Tokens and canonical primitives, no backend changes | Primitive tests/build pass; no duplicate foundation API added | COMPLETE |
| Phase 2 — Application shell | Login, sidebar, top shell, page container, context/account actions | Auth boundary and shell behavior verified | VERIFIED |
| Phase 3 — Services | Services and all service detail tabs | Service workflows and visual gate verified | VERIFIED |
| Phase 4 — Operations | Dashboard, Environments, Incidents, Alerts, Events | Operational behavior and responsive gate verified | VERIFIED |
| Phase 5 — Security | Users, tokens, application keys, sessions | Security behavior, permissions, and secret handling verified | VERIFIED |
| Phase 6 — Cleanup | Remove old CSS/components/dependencies after consumers migrate | No legacy UI consumers remain | PLANNED |
| Phase 7 — Verification | Visual, responsive, accessibility, behavior, build/test QA | Complete hard-reset definition satisfied | PLANNED |

## Regression checklist

- [ ] Login, logout, session restore, auth failure, and mandatory password change.
- [ ] Environment scope changes reload the same catalog and operational data.
- [ ] Services filter, paginate, select, bulk delete, create, edit, and delete.
- [ ] Instance, endpoint, and health-check mutations preserve API payloads and refresh behavior.
- [ ] Availability, incidents, alerts, and events retain queries, filters, navigation, and side effects.
- [x] Security workflows preserve roles, token scopes/expiry, one-time application-key secret behavior, and session revoke.
- [ ] Loading, empty, error, retry, success, and partial success states exist for migrated screens.
- [ ] EventSource and other subscriptions clean up on scope changes/unmount.
- [ ] Existing frontend and backend tests remain intact.

## Accessibility checklist

- [ ] Every form control has a programmatic label and validation message.
- [ ] Buttons, links, tabs, menus, dialogs, and switches use semantic or accessible primitives.
- [ ] Focus-visible styling and keyboard traversal work at every migrated boundary.
- [ ] Dialog focus trap, Escape close, restore focus, and scroll behavior are verified.
- [ ] Status changes and mutation feedback are announced appropriately.
- [ ] Contrast, target sizes, reduced motion, and zoom at 100/125/150% are verified.

## Responsive checklist

- [ ] Verify 1920, 1440, 1280, 1024, 768, and mobile widths.
- [ ] Verify 100%, 125%, and 150% zoom.
- [ ] No ordinary form/dialog horizontally scrolls.
- [ ] Tables have a deliberate responsive strategy: reflow, priority columns, or bounded scroll only where appropriate.
- [ ] Sidebar, environment context, filters, action groups, list/detail, and tabs remain usable on tablet/mobile.

## Open decisions

- Whether to introduce URL-backed routes for the existing state-driven views in Phase 2 or preserve state navigation until after screen extraction.
- Whether dark mode remains a supported product feature after the light operational system is verified; no behavior removal is authorized by this document.
- Exact icon set mapping for every nav item and status glyph.
- Whether notification toasts should be introduced per migrated mutation or after the shell is complete.

## Known technical debt

- `App.tsx` couples data loading, mutation orchestration, form state, and screen markup.
- React Router and React Query are declared but not currently the primary screen architecture.
- Generated/API naming still includes Deployment and Runtime concepts that must remain hidden in the product UI.
- Existing CSS has overlapping global, semantic, Tailwind, and legacy visual rules.
- Existing UI audit notes describe prior refinements but are not a complete migration ledger.
- No visual regression harness is currently present; verification will need a deliberate browser/screenshot pass.

## Completed migrations

| Date | Change | Verification |
| --- | --- | --- |
| 2026-10-06 | Phase 0 audit and migration-control document created | Current frontend inventory, behavior contracts, matrices, and phase gates recorded |

## Remaining migrations

## Phase 4 Audit: Operational Screens

Phase 4 is audited before implementation. The existing behavior remains in `App.tsx` and the new presentation boundary will consume the same state, API helpers, mutation handlers, environment scope, and navigation callbacks.

| Screen | Primary goal | Current legacy surface | Behavioral contract | New pattern | Status |
|---|---|---|---|---|---|
| Dashboard | Scan system health and exceptions | `DashboardView`, `MetricCard`, `AvailabilityCard`, legacy `OperationsUI` and page CSS | Environment scope, availability windows, service/health counts, open incidents, alert configuration counts, recent events, empty states | Operational summary + exception/activity tables | VERIFIED |
| Environments | Review and create environment scopes | Inline `.panel` form, raw inputs/buttons, legacy table rows | List environments, show key/tier/enabled state, create with key/name/tier/description, select created scope | Resource table + focused create section | VERIFIED |
| Incidents | Triage and resolve service incidents | Raw filter bar/table rows, legacy `StatusBadge`, hand-built modal | State/search filters, selected detail, service navigation, resolve open incidents, environment scope | Compact list/detail + shared Dialog | VERIFIED |
| Alerts | Manage alert policies and notification channels | Raw subnav, inline policy/channel forms, legacy rows/buttons | Policies/channels tabs, create/edit policy, create/edit/test channel, loading/error feedback | Settings-style tabs + dense tables/forms | VERIFIED |
| Events | Chronologically scan registry activity | Raw filter bar and timeline rows, legacy `ResourceLink` | Type/resource/search filters, live-loaded events, service navigation, empty state | Chronological data table | VERIFIED |

### Phase 4 Behavioral Contracts

- `loadCatalog` and `loadOperationalData` remain the source of truth for environment-scoped data and live event refresh.
- Existing create, edit, test, resolve, and navigation handlers remain in `App.tsx`; the new UI does not change API payloads or authorization semantics.
- Dashboard is read-only and routes users to canonical Incidents, Alerts, Events, Services, and Environments workflows.
- Environments preserves `createEnvironment` fields: key, name, tier, and description.
- Incidents preserves state/search filtering, service navigation, selected detail, and `resolveIncident` behavior.
- Alerts preserves policy/channel forms, edit behavior, notification-channel test behavior, and environment/deployment scope fields.
- Events preserves event type/resource/search filtering, live stream data, actor/resource context, and service navigation.

### Phase 4 Component Migration Matrix

| Legacy component or pattern | Replacement | Consumers | Removal condition | Status |
|---|---|---|---|---|
| `DashboardView` | `OperationalWorkspace` dashboard summary | Dashboard | Dashboard verified on replacement | TEMPORARY |
| `MetricCard`, `AvailabilityCard` | `Card`, `StatusBadge`, compact summary rows | Dashboard, Incidents | No Phase 4 screen imports `Cards.tsx` | TEMPORARY |
| legacy `OperationsUI` page helpers | `PageHeader`, `StatusBadge`, `Table`, `EmptyState`, `Button`, `Dialog`, `Select` from `components/ui` | Phase 4 screens | No migrated screen imports legacy helpers | IN PROGRESS |
| raw `.panel`/`.table` operational markup | `OperationalWorkspace` sections and shared UI primitives | Phase 4 screens | All five screens verified | IN PROGRESS |
| hand-built incident modal | shared Radix `Dialog` | Incidents | Incident detail parity verified | DESIGNED |

### Phase 4 Information Architecture Decisions

- Dashboard summarizes exceptions and recent activity; it does not duplicate full management lists.
- Environments owns environment creation and scope selection context.
- Incidents owns incident resolution and detail inspection.
- Alerts separates policy routing from notification channels with the existing tabs state.
- Events remains a chronological read-oriented screen; event rows stay compact and resource links navigate to canonical Services context.

### Phase 4 Responsive and Accessibility Review Plan

Code/layout review covers 1920, 1440, 1280, 1024, 768, and mobile constraints, including wrapping tables, action groups, filters, and dialog bounds. Keyboard focus, semantic headings/tables, labeled controls, status text, and shared Dialog behavior are reviewed during implementation. No Playwright/browser runner exists in this repository, so browser screenshot verification is documented as unavailable rather than introducing a new test dependency.

Phase 1 foundation and all screen migrations remain. A screen may only move from MIGRATED to VERIFIED after the behavioral, visual, responsive, and accessibility gates are recorded here.

## Phase 1 checkpoint notes

- Added the target UI dependencies and lockfile entries: Radix Dialog/Dropdown/Popover/Tabs/Tooltip/Slot, CVA, `clsx`, `tailwind-merge`, Lucide React, React Hook Form, Zod, TanStack Table, Sonner, and Testing Library User Event.
- Added source-owned primitives under `web/src/components/ui/` with shared `cn` utility and primitive tests.
- Foundation includes Button/IconButton, Input/Textarea/FormField, Select, Checkbox, Switch, Badge/StatusBadge, Radix Dialog/Tabs, Card, Table cells, Alert, Skeleton, EmptyState, PageHeader, SectionHeader, Breadcrumb, Pagination, Stack/Inline/DefinitionList, and Sonner adapter.
- `npm test -- --run`: PASS (2 files, 14 tests); jsdom reports the existing canvas `getContext` not-implemented warning from the accessibility test setup.
- `npm run build`: PASS after importing Vitest globals in the primitive test.
- `git diff --check`: PASS.
- Rebuilt `AppShell` around Lucide navigation icons, the canonical Button/Select/StatusBadge primitives, semantic navigation state, responsive shell tokens, and the existing callbacks. Existing App behavior tests pass after preserving accessible names for incident navigation and environment selection.
- `npm run lint`: BLOCKED by the existing repository state: `web/` has an ESLint script but no ESLint configuration file. This is tracked as tooling debt and must be resolved before final verification.
- `npm install` reports 24 existing dependency audit vulnerabilities (7 moderate, 17 high). No forced audit upgrade was applied during the UI migration.

## Audit report summary

### Frontend technology inventory

React 18, TypeScript, Vite, Tailwind CSS, React Router dependency, React Query dependency, ConnectRPC clients, Vitest, Testing Library, jsdom, axe-core, ESLint, and Prettier. Only React, Vite, Tailwind, ConnectRPC, Vitest, Testing Library, and related tooling are visibly used by the current frontend source; Router and React Query are not the dominant screen architecture.

### Current styling systems

Tailwind utilities; CSS custom properties in `App.css`; global element selectors; semantic class names; screen-specific layout classes; multiple button, dialog, tab, row, table, and status class families; optional dark-mode overrides.

### Current component inventory

The shared components and dispositions are listed above. The largest local component is the monolithic `App.tsx`, followed by domain-specific topology, dashboard, health, and security render branches.

### Known UX problems

Mixed workflows, duplicated controls, inconsistent dialogs, raw forms, ambiguous status/boolean presentation, internal runtime terminology, state-driven navigation without URL semantics, and screen-local layout decisions.

### Known visual inconsistencies

Navy/cyan glow styling, light neutral panels, Tailwind slate utilities, raw HTML controls, local button variants, local tables, and multiple modal treatments coexist. The current dark-mode implementation further duplicates token decisions.

### Known duplicated controls

Buttons, icon buttons, close controls, tabs, status badges, dialogs, panel/card shells, table/list rows, loading placeholders, filters, and empty states.

### Current information-architecture issues

Service list/detail/configuration/history concerns are interleaved; health has multiple responsibilities in one workspace; security sections are controlled by hidden panels inside the monolith; alerts mix policy and channel administration; create/edit forms are frequently embedded beside lists; and backend runtime terminology leaks into the service experience.

### Main regression risks

The primary risks are losing hidden mutation side effects while extracting from `App.tsx`, changing environment-scope query timing, breaking auth boundary semantics, mishandling one-time application-key secrets, losing EventSource cleanup, and accidentally changing permission-sensitive action visibility. Each screen migration must preserve behavior through existing tests plus focused interaction checks.

## Phase 2 verification checkpoint

Status: VERIFIED for the implemented shell acceptance surface. Sidebar navigation, active state, environment selector, refresh, dark mode, logout, incident count, authorization-gated Security visibility, responsive shell rules, and focus-visible controls remain wired to existing callbacks and state. Browser viewport and zoom verification remains part of Phase 7 because no visual regression harness exists.

## Phase 3 Services checkpoint

Status: IN PROGRESS.

The legacy presentation was a monolithic Services branch in `App.tsx`, supplemented by `RuntimeTopology` and `AddRuntimeDialog`, with inline tables, drawers, status widgets, and local CSS. The new presentation is `ServicesWorkspace`: compact catalog, resource detail header, shared tabs, Overview, Instances, Availability, Health, Incidents, and Events, using the shared primitives.

Behavior preserved includes service search, health/tag filtering, pagination, selection and bulk actions, environment scoping, service CRUD, instance registration/edit/remove, endpoint CRUD, health actions, availability summaries, incident/event navigation, and loading/empty/error states. Internal Deployment/Runtime terminology remains out of the visible product model.

Intentional decisions: informational Enabled state uses `StatusBadge`; editable instance/endpoint state uses `Switch`; Instances owns topology and creation; Availability has no duplicate Add instance action; and service detail uses one canonical tab system. The visible legacy AddRuntimeDialog mount was removed. Its source remains temporarily available only for rollback until follow-up parity is verified.

| Services screen | Status | Notes |
| --- | --- | --- |
| Services list | IN PROGRESS | New catalog, filters, selection, pagination, loading, and empty state. |
| Service detail shell | IN PROGRESS | New resource header and shared tabs. |
| Overview | IN PROGRESS | Summary metrics and operational summaries. |
| Instances | IN PROGRESS | New instance cards, switches, endpoint summary, edit/remove, and Add Instance dialog. |
| Availability | MERGED INTO HEALTH | Windows and operational context now belong to unified Health. |
| Health | IN PROGRESS | Service-scoped checks/results; follow-up configuration parity remains. |

Verification checkpoint: `npm run build` and `git diff --check` pass. `npm test -- --run` currently passes 12 of 14 tests; the remaining two Services workflow tests require exact service context and the new dialog's health follow-up action. Lint remains unavailable because no ESLint configuration exists.

Remaining Phase 3 debt: finish Add Instance success/follow-up state, add endpoint creation from an existing instance through EndpointEditor, perform browser visual verification at required widths/zoom, and remove temporary legacy sources after all consumers are gone.

## Phase 3 verification checkpoint

Status: VERIFIED for the migrated Services surface.

| Area | Status | Verification |
| --- | --- | --- |
| Services list | VERIFIED | Catalog search/filter/pagination/selection, loading, empty, and error behavior covered by the application suite and static layout review. |
| Service detail shell | VERIFIED | Resource header, status, metadata, overflow actions, and canonical tabs use shared primitives. |
| Overview | VERIFIED | Summary metrics and operational summaries render without duplicated management actions. |
| Instances | VERIFIED | Instance CRUD, enable/disable Switch, endpoint summary, endpoint creation, and empty state use the new surface. |
| Availability | MERGED INTO HEALTH | Preserved read-oriented windows and scope; no instance creation action. |
| Service Health | VERIFIED | Service-scoped monitoring and optional post-registration health follow-up, including retry without duplicate registration. |
| Incidents | VERIFIED | Shared status and empty-state language. |
| Events | VERIFIED | Shared list and empty-state language. |
| Add Instance | VERIFIED | New Dialog/Form flow preserves service context, environment, instance fields, endpoint validation, loading, API errors, success, and Done. |
| Add Endpoint | VERIFIED | Instance-level EndpointEditor uses the existing create endpoint API and shared fields. |
| Responsive QA | VERIFIED | CSS review covers 1920, 1440, 1280, 1024, 768, and mobile breakpoints plus wrapping rules; no Playwright/browser runner is installed in this repository. |
| Accessibility QA | VERIFIED | Automated axe coverage passes; labels, focus-visible controls, dialog semantics, keyboard tabs, switches, and icon-button names reviewed. |

Intentional UX changes: Add Instance canonical home is Instances; Availability is operational/read-oriented; informational Enabled state is `StatusBadge`; editable Enabled state is `Switch`; post-registration health is optional and retry never resubmits instance creation.

Legacy dependency disposition: `AddRuntimeDialog.tsx` was removed after the replacement flow and tests passed. `RuntimeTopology.tsx` remains TEMPORARY only because an unreachable rollback branch in `App.tsx` still type-checks the old presentation; it must be removed with that branch during Phase 6 cleanup.

Final Phase 3 gate: `npm test -- --run` PASS (14/14), `npm run build` PASS, `git diff --check` PASS. Lint remains UNAVAILABLE because no ESLint configuration exists.

## Phase 4 Verification Checkpoint

| Screen | Status | Notes |
|---|---|---|
| Dashboard | VERIFIED | Compact operational summary; exceptions and activity link to canonical screens |
| Environments | VERIFIED | Resource table and focused create form preserve environment API fields |
| Incidents | VERIFIED | Shared status badges, filters, list/detail Dialog, service navigation, and resolve action |
| Alerts | VERIFIED | Shared Tabs, dense policy/channel tables, preserved create/edit/test flows |
| Events | VERIFIED | Chronological table, type/resource/search filters, actor/resource context |
| Responsive QA | VERIFIED | Code/layout review covers 1920, 1440, 1280, 1024, 768, and mobile constraints; no browser runner exists |
| Accessibility QA | VERIFIED | Semantic tables, labels, Radix Dialog focus handling, keyboard Tabs, visible status text, and focus-visible primitives reviewed |

### Phase 4 Intentional UX Changes

- Dashboard is read-only and routes to canonical operational workflows instead of duplicating management interfaces.
- Incidents use the shared Radix Dialog instead of a hand-built modal backdrop.
- Alerts use the canonical Tabs primitive for policy/channel separation.
- Events display human-readable event labels while retaining the raw event type in accessible text and preserving raw filter values.
- Environment creation is a focused design-system form rather than an always-open legacy disclosure.

### Phase 4 Remaining Technical Debt

- The old operational JSX branches remain unreachable in `App.tsx` as rollback scaffolding and are scheduled for Phase 6 cleanup after broader screen extraction.
- `OperationsUI.tsx` and `Cards.tsx` remain because the not-yet-migrated global Health and legacy fallback branches still type-check against them; no Phase 4 route renders them.
- No Playwright/browser runner exists, so screenshot-based visual verification and live zoom checks remain a documented tooling limitation.
- Lint remains UNAVAILABLE because the repository has no ESLint configuration; no lint configuration was introduced for Phase 4.
- Runtime compatibility note: some deployed API versions return 404 for `GetInstanceHealthState`; the frontend now treats instance state as unavailable instead of issuing the request during catalog loading, and availability rendering tolerates incomplete summaries.

## Compactness and Hierarchy Refinement

### 2026-10-07 Form and Control Audit

Status: IMPLEMENTED; verification results and limitations are recorded below. This pass rebuilds the source-owned controls before updating their consumers.

- Legacy global button/input rules impose padding and minimum height on Radix Checkbox/Switch. Their intended dimensions are not their rendered dimensions.
- Input, Select, and Textarea use separate presentation strings. Select has no consistent chevron; helper/error text has no aria relationship to the control.
- Services repeats Catalog/count and Selected service labels, nests summary cards within the detail card, and hardcodes the Overview environment count.
- Operational filters duplicate row markup; alert recovery uses a raw checkbox. Pagination also bypasses Button.
- Existing mutation handlers, required fields, API payloads, permission gates, and controlled form state are retained. Foundation components own presentation and accessible relationships.

Planned replacements: canonical form controls and field context; shared FormSection/SearchInput/PasswordInput/MultiSelect/RadioGroup; canonical control stylesheet isolated from legacy rules; compact Stat; shared FilterBar consumers; unframed Services details and summary sections.

The refinement audit found repeated active-view titles in the shell and page header, an unnecessary `Alauda` eyebrow on every new page, generous header/card/dialog padding, tall sidebar spacing, and Services-specific repetition between `Services`, `Service catalog`, and the selected-resource breadcrumb. Phase 4 operational screens also inherited larger-than-needed utility gaps from the initial foundation.

### Compactness Rules Adopted

- One primary page title per screen; the authenticated shell provides environment context and controls, while the content page owns the title.
- Page descriptions are concise supporting context, not a second title or paragraph-length explanation.
- Product/category eyebrows are removed when the page title already establishes orientation.
- Section headers use compact vertical padding and descriptions only where they add operational context.
- Tabs, list rows, metric tiles, dialogs, and page containers use compact spacing tokens by default.
- Services relies on its page title and selected resource name; redundant `Catalog` and `Selected service` headings are removed.
- Empty states remain breathable but no longer reserve oversized vertical blocks.
- Density belongs to shared components and layout tokens, not application-wide overrides of utility classes. The old content-region `gap-5` override is removed.

### Affected Shared Primitives and Screens

- Refined `PageHeader`, `SectionHeader`, `EmptyState`, dialog header/body/footer spacing, shell/topbar, sidebar spacing, and global content density.
- Refined Services catalog/detail hierarchy, service rows, detail header, summary metrics, tabs, and section spacing.
- Applied inherited density improvements to Dashboard, Environments, Incidents, Alerts, and Events through shared primitives and content spacing.

### Compactness Re-verification

Screens re-reviewed: Dashboard, Services list/detail, Environments, Incidents, Alerts, Events, authenticated shell/sidebar, dialogs, and forms. The pass preserves the existing responsive breakpoints and keyboard/accessibility primitives. Remaining debt is the broader legacy CSS and old fallback branches scheduled for Phase 6 cleanup; no new screen-specific density exception was introduced.

## Visual Composition Correction

Status: IMPLEMENTED AND REVIEWED. Audit found that unframing the selected Services detail area removed its major-region boundary; individual overview stats no longer read as one summary; Availability/Monitoring headers have inconsistent content alignment; activity rows distribute short text across excessive space. Dashboard used separate cards for tiny metrics and a different activity layout.

Proposed correction: three levels only (page, workspace, internal section); shared Workspace, Section, StatGroup and ActivityList patterns. Preserve compact control sizes, filters, API calls, selected-service state, event ordering/limits and canonical actions. Do not restore Catalog/Selected service headings. Services and Dashboard are the immediate consumers; existing admin list surfaces remain intact rather than gaining nested containers.

### Authoritative Visual Hierarchy

- Level 1: existing application page background. Level 2: major resource workspace or list surface, neutral background, one subtle border, compact 16px padding, restrained radius and no strong shadow. Level 3: transparent internal Section, heading/action alignment, optional subtle top divider and 8px header/content rhythm. Do not introduce further nested cards for summary text.
- Workspace anchors resource identity and tabs together. Services catalog and selected resource remain two clear regions; resource name establishes selection context, not another eyebrow or repeated page title.
- Section is the canonical internal-section primitive with an accessible heading, optional description, optional actions, content and optional divider. SectionHeader remains appropriate for existing table/list region headers; it is not an extra wrapper around Section.
- StatGroup presents related metrics in one light summary strip with internal separators; Stat owns label/value and optional detail. No individual metric cards. Four summary segments wrap to two columns on mobile.
- ActivityList uses one semantic list with consistent timestamp, event and context columns. Related values remain in one source-order row; narrow layouts move context below the timestamp/event pair. Ordering and limits remain screen-owned, not altered by the presentation primitive.
- Services Overview uses a 60/40 Availability/Monitoring split at wide widths and stacked sections on mobile. All three availability windows occupy aligned columns. Workspace header, compact tabs, summary and divided sections maintain predictable spacing.
- Catalog selection uses a muted neutral surface plus an accent edge, rather than a strong colored fill. Compact row separators distinguish resources without individually boxing them. Checkbox, badge and pagination semantics remain unchanged.
- These rules supersede the earlier recommendation to leave the selected detail pane entirely unframed; the older compactness checkpoint describes historical implementation, not the final surface contract.

### Replacements and Verification

- Shared primitives introduced/refined: Workspace, Section, StatGroup, Stat (optional supporting detail), ActivityList and section-aware EmptyState spacing. Compact Button/Input/Select/Tabs sizing is unchanged; sidebar is unchanged.
- Services: selected detail section replaced by Workspace; separate metric layout replaced by StatGroup; Availability/Monitoring/Recent activity use Section; space-between activity rows replaced by ActivityList. Obsolete service metric CSS removed.
- Dashboard: four standalone Summary cards replaced by StatGroup/Stat; two internal section cards replaced by shared Section inside one Workspace; event rows use ActivityList. Existing table, service navigation and View incidents/events actions are preserved.
- Existing application tests retained. Added tests for named sections/action callbacks, grouped metrics, and activity source order. No backend/API/authentication/authorization changes, dependencies or whole source-file deletions in this correction.
- Air browser fixture review: Services DOM/CSS snapshots at 1920, 1440, 1280, 1024, 768 and 390px found no page overflow; workspace neutral surface, stats segmentation, section proportions and activity column wrapping were measured. Dashboard at the current 488px viewport has one workspace, one stat group, two named internal sections, no nested Cards and no page overflow.
- Browser review used existing fixtures in memory only; it did not mutate backend data. It is not screenshot-based certification of every live workflow, zoom or data variant. The unmocked preview event stream still returns 404, as already documented.
- Automated gates: `npm test -- --run` PASS (20/20); `npm run build` PASS (includes TypeScript); `git diff --check` PASS. Nonfatal existing Vite/jsdom and line-ending warnings remain. Lint remains UNAVAILABLE - no ESLint configuration exists. Previously documented portal-theme/zoom and legacy cleanup debt remains unchanged.

## Form and Control Standards

### Authoritative Contract

- Input, PasswordInput, SearchInput, Textarea, Select, MultiSelect, Checkbox, RadioGroup, Switch, FormField, FormSection, FieldDescription and FieldError share the source-owned form system. No additional framework or dependency was added.
- Standard input/select height: 36px. Button heights: 36px default, 32px small; icon actions: 32px. Checkbox/radio visual size: 16px; Switch: 32 x 20px. Labels provide the larger clickable area where applicable.
- Labels: 12px medium; controls: 13px; helper/error text: 12px regular. Label/control gap: 4px; field-group gap: 12px. Settings forms have a readable 680px maximum width and collapse to one column on narrow screens.
- Controls use a subtle neutral border, surface background, 5px radius, readable placeholders and a consistent focus-visible outline. Disabled controls use muted surfaces; readonly text remains readable; invalid state uses a restrained semantic border plus linked error text, never color alone.
- FormField supplies stable IDs, accessible names, linked descriptions and aria-invalid. Explicit accessible names (including numbered endpoint fields) take precedence. MultiSelect options have unique IDs and independently readable labels.
- Select retains native keyboard behavior, with shared appearance and a decorative chevron. Password reveal is a named, non-submit icon action; it does not alter the value or password validation.
- FormSection uses a fieldset/legend and restrained separator, not another card. Dialogs use the same controls as pages, with scrolling confined to their bodies. Controlled dialogs restore focus to the opener on close; consumer autofocus overrides remain supported.
- Filters use shared SearchInput/Select and FilterBar wrapping. Pagination uses canonical Buttons. Informational booleans remain StatusBadge; editable booleans remain Checkbox/Switch.
- VERIFIED requires behavior tests, design-system compliance, compactness review, accessibility review and responsive review; automated passing tests alone are not visual certification.

### Replacement and Consumer Checkpoint

| Component / surface | Disposition | Behavior preserved / intentional presentation change |
| --- | --- | --- |
| Input / Textarea / Select / FormField | REBUILT | Controlled values, native validation, ref support and event callbacks retained; one visual family and accessible field relationships |
| Checkbox / Switch | REBUILT | Radix boolean and keyboard semantics retained; dimensions protected against legacy global button rules |
| PasswordInput / SearchInput / MultiSelect / RadioGroup | SHARED FOUNDATION | Reusable controls, accessible names and option IDs; no overlapping UI library |
| Button / IconButton / Pagination | REFINED | Canonical variants/sizes; default non-submit behavior; pagination callbacks and boundaries unchanged |
| Dialog / DialogDescription | REFINED | Radix focus trap and Escape retained; focus restoration fixed; generated hidden context text removed |
| Services catalog and detail | RECOMPOSED | Search/filter/selection/pagination and tabs retained; redundant headings removed; unframed detail and summary sections replace nested cards |
| Overview metrics | REPLACED by Stat | Compact inline summary; environment count now derived from selected deployments, not hardcoded |
| Add Instance / endpoint fields | REFINED | Existing registration state, endpoint primary selection, API payloads and validation unchanged; shared form section without redundant Connections title |
| Login / password change | REBUILT presentation | Authentication handlers, required fields, autocomplete and 12-character password rule retained; canonical fields, password reveal, buttons and errors |
| Operational filters / configuration | REFINED | Shared FilterBar/SearchInput; bounded forms; recovery checkbox now Radix; existing mutation handlers preserved |

No whole source component was deleted in this refinement. Obsolete metric markup, nested detail/summary Card wrappers, raw authentication form controls, raw pagination buttons, raw recovery checkbox and hidden context hacks were replaced. Existing rollback branches and their consumers are not removed prematurely.

### Review and Verification

- Existing application behavior suite retained. Added focused tests for field descriptions/errors, password reveal, multi-select option naming/selection and controlled-dialog Escape/focus restoration. The service-context assertion now checks the dialog's accessible description rather than obsolete hidden text.
- Browser fixture QA used the Air preview tool, without installing a browser runner or changing backend data. At the active 488px viewport, inputs/selects measured 36px, checkbox/radio measured 16px, and the Add Instance dialog stayed within the viewport without horizontal overflow. Shift+Tab remained trapped inside the dialog; Escape closed it. The focus-return issue discovered during this review was fixed centrally and covered by an automated test.
- Rendered DOM/CSS snapshot checks in isolated iframe viewports at 1920, 1440, 1280, 1024, 768 and 390px found no page overflow. Catalog width was 320px at wide widths; layout stacked below 1100px. These are layout measurements, not full live-workflow or screenshot regression certification at every viewport.
- Re-reviewed shell environment control, Services catalog/detail/Overview/Instances and dialog composition; Dashboard/Environments/Incidents/Alerts/Events inherit shared control/table improvements. No authentication, authorization, API or business-rule changes were intended.
- Final automated gates: `npm test -- --run` PASS (18/18); `npm run build` PASS (includes `tsc`); `git diff --check` PASS. Existing nonfatal jsdom canvas and Vite plugin deprecation warnings remain. Lint: UNAVAILABLE - no ESLint configuration exists.

### Remaining Design Debt

- Global Health and Security are not fully migrated; their legacy presentation remains scheduled for the subsequent migration phases, rather than being superficially styled in this pass.
- Legacy global CSS and unreachable rollback JSX remain for Phase 6. New controls own their presentation in `ui/ui.css`; removing the legacy cascade requires auditing remaining live consumers.
- Native selects are deliberately retained for reliable keyboard and form behavior; platform dropdown menus are not pixel-identical across operating systems.
- Dark-theme portal inheritance, full live zoom QA, long-data fixtures across every operational screen, and screenshot-based visual regression remain follow-up checks. Browser fixture QA cannot prove deployed backend compatibility; the preview's event stream was not mocked and returned 404.

## Instances Progressive Disclosure

Status: VERIFIED for the replacement presentation and tested workflows, subject to the browser/fixture limitations below. This supersedes the previous instance-card presentation documented in the Phase 3 checkpoint.

### Audit and Architecture Decision

The previous instance cards mixed identity, health state, endpoint summaries, edit forms and endpoint creation, then repeated endpoints in a separate service-wide list. This obscured subresource ownership and increased default-view complexity. The old markup was replaced, not restyled.

Selected model: environment-grouped compact table -> instance detail Dialog -> focused endpoint editor state within the same Dialog. Current metadata and actions do not require a dedicated route or another persistent master/detail pane. A single controlled Dialog avoids nested focus traps and stays within the service context.

- List: instance name (accessible details action), address, endpoint count, monitoring check count/not configured and current status. Environment groups remain compact. No full endpoint properties, inline editors, descriptions, or health internals are permanently exposed.
- Instance detail: Endpoints / Monitoring / Details canonical Tabs. Endpoints are a structured table with protocol, port, path, primary designation, enabled state and named edit/remove icon actions. Primary and enabled are separate facts; a primary endpoint no longer hides its disabled state.
- Endpoint create/edit: reuses the single existing EndpointEditor in a focused dialog state. It owns protocol/port/path/primary/enabled only, not instance creation. Escape/Cancel returns to details; successful save returns after the existing handler clears editor state. Failed saves retain fields and expose a shared Alert inside the dialog.
- Monitoring: the list shows only configured check count; instance details show configured/enabled counts and a link to Service Health. Check configuration, manual runs and executions belong to Health. Configure health closes instance details and hands the selected instance to Service Health's configuration dialog.
- Details: metadata and focused instance edit form; editable enabled state uses Switch. Remove instance stays here, with the existing dependency-aware confirmation. Delete endpoint remains owned by its instance detail table.
- Add instance remains the list's canonical create action. Empty list/endpoints/monitoring states expose one relevant CTA each.

### Components and Behavioral Contract

- Recreated ServiceInstances as a summary list; introduced InstanceDetails with locally owned selection/tab state. Removed InstanceRow and EndpointRow presentation functions and the duplicate service-wide endpoint block. No whole source file was deleted.
- Reused Section, Table, Tabs, Dialog, DefinitionList, EmptyState, FormField, Select, Input, Textarea, Switch, StatusBadge, Button/IconButton and EndpointEditor. No new framework, dependency, page-specific layout exception or route introduced.
- Removed unused instance-card/subsection/inline-editor/endpoint-row/row-overflow styling after their consumers disappeared. Registration endpoint-draft styling remains because Add Instance still uses it.
- Existing App mutation handlers, API payloads, primary rules, authentication/authorization, deletion confirmations, environment/service scoping and catalog refresh remain unchanged. Dialog receives the existing mutation error state so API errors are readable within the active workflow.
- Disabled instance state takes precedence over stale health information in the summary; configured checks are summarized without falsely implying they are all enabled. No frontend-only primary-endpoint rules were invented.

### Verification and Limitations

- Tests cover progressive disclosure, detail endpoint/health rendering, selected-instance endpoint creation payload, primary/enabled edit payload, instance enabled mutation, API-error retention, endpoint/monitoring empty states, health handoff and Escape/focus restoration.
- Test fixture mutation responses were completed to match existing API contracts. jsdom received a no-layout ResizeObserver shim required by Radix switches in forms; production code/response validation was not weakened.
- Rendered list DOM/CSS checks at 1920, 1440, 1280, 1024, 768 and 390px found no page overflow. The five-column table intentionally scrolls inside the shared Table region at narrow widths. At the active 488px browser width, the detail dialog and focused endpoint form had no horizontal dialog overflow; the endpoint data table scrolls within its own region.
- Automated keyboard test verifies endpoint-editor autofocus (Name after the corrective fix below), Escape back to details, then Escape close and focus return to the instance-list trigger. Shared Tabs/Dialog keyboard and focus semantics remain intact.
- Browser checks used in-memory fixtures, not backend mutations. Full live-data screenshot/zoom certification, portal dark-theme inheritance and broad long-data scenarios remain documented design debt; the fixture preview's unmocked event stream still returned 404.
- Final gates: `npm test -- --run` PASS (27/27), `npm run build` PASS (includes TypeScript), `git diff --check` PASS. Existing nonfatal Vite/jsdom and line-ending warnings remain. Lint remains UNAVAILABLE - no ESLint configuration exists.

### Endpoint Name Collision Correction

Status: VERIFIED. A reported Connect `already_exists` response exposed a presentation regression: Add Endpoint initialized every name to `http`, while EndpointEditor omitted the name control. Storage enforces `UNIQUE(instance_id, name)`, so users could not correct collisions.

- Restored required editable Name in the shared create/edit editor, with linked duplicate-name validation and Name autofocus. Names are trimmed on submission, validated for nonempty content, and checked against known endpoints of the same instance, excluding the endpoint being edited. Case-sensitive matching follows the existing storage constraint.
- Add Endpoint suggests `http`, then the first available `http-2`, `http-3`, etc. in that instance; changing protocol does not silently replace the chosen identity.
- Backend Connect `already_exists` is translated to an actionable duplicate-name message. Other structured endpoint errors retain their server message; plain-text errors still work. Backend uniqueness and API contracts remain unchanged, and failures keep the editor/values available for correction.
- Suggestions use the loaded active catalog only. Concurrent active creation can still collide at the server and is handled as an error, not silently overwritten or automatically retried. The backend correction below resolves collisions with soft-deleted names.
- Added regression coverage for collision-free suggestions, preflight duplicate rejection, successful distinct-name payloads and the exact reported server JSON error. Existing edit tests verify that an endpoint may retain its own name.
- Gates: `npm test -- --run` PASS (29/29); `npm run build` PASS (includes TypeScript); `git diff --check` PASS. No backend change, dependency, or lint configuration added.

## Unified Service Health Migration

Status: AUDITED -> DESIGNED -> IN PROGRESS -> MIGRATED -> VERIFIED, with the fixture/browser limitations below. This checkpoint supersedes the separate Availability/Health presentation in earlier phase records.

### Service Detail Navigation

Overview / Instances / Health / Incidents / Events.

Former Availability tab: **MERGED INTO HEALTH**. Existing `/services/:id/availability` URLs select Health and are canonicalized to `/services/:id/health` using history replacement, without adding a navigation entry. No visible Availability tab remains. Overview is unchanged.

### Audit and Health Ownership

- Availability previously owned historical windows and scoped instance summaries; Health owned check configuration and manual runs with little results context. Separate sparse screens obscured their operational relationship.
- Health now owns current service health, monitoring, historical availability and check observations. Backend Availability, HealthCheck, HealthResult and instance-state entities remain separate.
- Current status uses a compact StatGroup: instances, monitored instances (enabled checks), healthy and unhealthy. Missing instance state is explicitly Unknown, not inferred from enabled state, lack of incidents, or old results.
- Availability uses the existing 24-hour/7-day/30-day queries, environment scope, percentage formatting, downtime and incident semantics. Three compact windows share one no-data explanation.
- Health checks use wrapping resource rows with distinct identity/type, interval, human-readable instance/endpoint target, Enabled/Disabled, latest observation and Run action. Creation is a focused Dialog, not a permanent form.
- Recent results show at most five observations on the main surface. View all results navigates to the canonical `/health/results` search page; check selection opens configuration, counters and a three-execution preview with a check-scoped history link. Unsupported or missing data is not fabricated.
- Instances retain monitoring counts/not-configured summaries. Instance detail Monitoring provides enabled/configured counts and a Service Health link, not duplicated check configuration or Run actions. Topology and endpoint mutations remain in Instances.

### Component Migration and Behavioral Parity

| Previous component/pattern | Replacement | Classification/status |
| --- | --- | --- |
| ServiceAvailability presentation and tab | ServiceHealthView Availability section | REMOVE / MERGED INTO HEALTH |
| ServiceHealth presentation | ServiceHealthView resource rows, results and drawer | REPLACED / VERIFIED |
| Instance inline check/run list | Monitoring summary and Health handoff | REPLACED / VERIFIED |
| HealthDialog bound to creation state during edit | Canonical HealthCheckForm with actual persisted check values | RECREATED / VERIFIED |
| Availability API/calculation helpers | Same helpers passed into Health | KEEP |
| Unreachable rollback Services/Availability/Health JSX in App | Removed after shared view/form parity tests | REMOVED |

- Removed dead Availability/old-health list CSS only after checking live consumers. Overview's availability presentation remains intact.
- Shared Drawer wraps the canonical Radix Dialog with right-edge placement, preserving focus trap, Escape and restoration. DialogHeader reserves close-control space. ResourceList/ResourceRow provide semantic, wrapping operational rows. StatGroup adds reusable two/three/four-column support.
- Existing create/update/delete/run handlers and authorization are retained. Edit now changes the actual edit-form interval, timeout, thresholds, description and enabled state rather than accidentally changing creation state. Unsupported identity/target edits are not offered.
- Result requests are restricted to checks in the selected service/environment. Responses are scoped, deduplicated and chronological; cancelled/obsolete requests cannot replace current results. Loading, error/retry, no monitoring and no results use shared primitives.
- Manual runs update current instance state only from the returned backend state and refresh observations. Instance/endpoint creation, primary rules and optional post-registration health failure/retry semantics are preserved. No backend mutation or dependency change was required.

### Verification and Remaining Debt

- Regression coverage: canonical tabs and old URL alias; unchanged availability percentages; current-state Unknown/actual-state behavior; scoped result requests; no monitoring/no availability; monitoring without results; manual runs; check-detail navigation; create/edit payloads; environment-filtered availability; five-row main history bound; drawer accessibility, Escape and focus restoration. Existing optional post-registration failure fixture now explicitly returns HTTP 500 instead of relying on an incomplete successful response.
- Rendered Health DOM/CSS fixture checks at 1920, 1440, 1280, 1024, 768 and 390px found no horizontal page overflow. Availability remains three compact columns; resource rows wrap. These are layout measurements, not full live screenshots or browser-zoom certification.
- Live fixture preview at 466x513: check drawer stays inside the viewport (434px wide), has no horizontal overflow, and Escape restores focus to the selected check. Automated axe WCAG 2A/AA review of the drawer passes; existing shared focus/keyboard tests remain intact.
- Browser QA uses in-memory fixtures, not production backend mutations. The fixture's unmocked event stream returns 404. Full live-data zoom/screenshot verification and portal dark-theme inheritance remain existing debt.
- Current-state catalog loading avoids the unavailable GetInstanceHealthState endpoint. Absent state remains Unknown; a successful manual run can provide actual state. A compatible state-loading capability remains deployment/API debt, not inferred frontend health.
- Results API currently retrieves up to 20 observations per check. The drawer displays available recent history, not an invented exhaustive history/pagination contract. A full historical explorer is deferred.
- Final gates: `npm test -- --run` PASS (37/37); `npm run build` PASS (includes TypeScript); `git diff --check` PASS. Existing nonfatal Vite/jsdom and line-ending warnings remain. Lint: UNAVAILABLE - no ESLint configuration exists.

## Endpoint Soft-Delete Name Reuse

Status: VERIFIED in repository tests; deployment requires the updated backend. Follow-up to the endpoint-name UI correction: Delete hides endpoints using `deleted_at`, but the existing `UNIQUE(instance_id, name)` constraint covers deleted rows too. An apparently empty instance could therefore reject a name that belonged to a deleted endpoint. Frontend suggestions cannot resolve invisible tombstones reliably.

- EndpointRepository.Create now restores a deleted endpoint with the same instance/name, following the existing HealthRepository restoration pattern. It retains the original endpoint identity/creation time and references, clears `deleted_at`, and replaces protocol, port, path, enabled, primary, tags and metadata with the submitted settings. Existing references now point to the restored endpoint; checks are not independently created or enabled.
- Restoration and primary-endpoint reassignment occur in the existing transaction. Active duplicate names still fail with the existing uniqueness error; failed creation rolls back primary changes and never overwrites active configuration. Deleted names in other instances remain untouched.
- No schema migration, manual deletion, API payload change, authentication/authorization change or frontend visual change. Existing deployed tombstones are handled when CreateEndpoint is retried against the updated backend. Backend rebuild/restart or redeployment is required; refreshing the frontend alone cannot apply this fix.
- Regression tests cover an empty visible list after deletion, repeated restoration, replacement of stale settings/tags/metadata, primary and non-primary restoration, active duplicate rejection with primary rollback, and instance isolation.
- Gates: `go test ./...` PASS (including storage/API/integration); `npm test -- --run` PASS (37/37); `npm run build` PASS (includes TypeScript); `git diff --check` PASS. No running backend was modified or production database mutated. Lint remains UNAVAILABLE - no ESLint configuration exists.

## Endpoint Creation Semantics and Shared Form

Status: AUDITED -> DESIGNED -> IN PROGRESS -> MIGRATED -> VERIFIED for the shared editor and tested contracts, subject to fixture verification limits below. Audit: endpoint storage already enforces case-sensitive `UNIQUE(instance_id, name)`, not global/service-wide uniqueness. Both create/update payloads support Name, Protocol, Port, Path, Primary and Enabled. Registration's presentation omitted Name/Enabled and hardcoded Enabled=true; its endpoint storage also coerced false to true.

- One controlled EndpointEditor owns all six fields in the same order and responsive one/two-column grid. Add Instance, Add Endpoint and Edit Endpoint consume it; flow wrappers own submission/footer and collections, not independent field markup. Canonical FormField/Input/Select/Switch/IconButton preserve labels, validation association, dimensions and compact rhythm. Endpoint dialogs use md; repeated registration uses lg.
- One EndpointFormValue and Zod schema own required trimmed names, supported protocols, integer ports 1-65535, optional string path, and boolean primary/enabled. Instance-local existingNames exclude the edited endpoint. Collection validation uses the same schema plus at-most-one-primary. No protocol/port/path uniqueness rule is invented; distinct names may share an address.
- Names default to editable `default`, then the first unused `default-2`, `default-3`, etc. Names from other instances never influence suggestions/validation. Backend duplicate responses attach an actionable name error rather than exposing constraint text. Concurrent active creation still relies on server uniqueness.
- Newly created endpoints default Enabled=true in both frontend flows. Registration submits the selected enabled value; the backend no longer forces false to true. This is an intentional correction to respect the existing boolean request, not a schema/API change. Omitted proto3 enabled also evaluates false; clients requiring enabled endpoints should send true explicitly.
- Primary defaults true for the first endpoint, false for additions. Selecting another registration draft clears other primaries. Backend allows at most one primary, not exactly one in all cases; registration auto-promotes a sole endpoint as before, so its single-draft primary switch is locked on. Standalone create/edit preserve the existing optional-primary semantics and backend transactional reassignment. Zero endpoints are allowed by registration: final draft removal is available; recreating the first draft restores default primary. Path remains optional/unrestricted as supported by the API; no HTTP-only requirement is imposed on TCP/UDP/gRPC.
- Removed duplicate standalone/registration field JSX, raw primary radio, and obsolete endpoint-draft card/grid CSS. Health dialogs continue to use the same underlying form primitives without unrelated changes.
- No uniqueness migration required. Existing endpoint soft-delete restoration remains intact. Endpoint update-handler uniqueness errors now use the existing `already_exists` mapping, matching create errors so concurrent rename conflicts receive the same field-level message. Backend redeployment is needed for registration Enabled=false support and the error mapping.

### Endpoint Form Verification

- Shared validation/default tests cover deterministic suggestions, name trimming/case sensitivity/instance scope, required name, supported protocols, integer ports, boolean values, optional paths, same-address distinct names, at-most-one-primary and empty collections.
- App tests cover registration editable names/unique defaults/multiple drafts/removal/primary reassignment/Enabled=false/duplicate prevention; first-draft recreation and endpoint-free registration; standalone default/custom name and Enabled=false; editable current name, own-name exclusion and duplicate rename rejection; server collision field errors; existing Escape/focus behavior. Existing health follow-up and endpoint-restoration tests remain intact.
- Backend tests verify two instances each registering `default`, disabled/enabled values persisted, primary behavior retained, and duplicate endpoint rename returning `already_exists`. No backend validation tests were weakened.
- Repeated Add Instance editor passes automated axe WCAG 2A/AA checks. Field errors remain associated through FormField; all six controls and remove actions have names. Standalone Name receives initial focus.
- Rendered fixture DOM/CSS measurements at 1920, 1440, 1280, 1024, 768 and 390px found no horizontal page/dialog overflow in registration and standalone editors. Desktop dialog widths are bounded at 672px/512px respectively; both measured 358px wide at 390px. Active previews also had no dialog overflow at 466px (registration) and 218px (standalone). This is fixture/layout review, not complete production screenshots/zoom QA. Browser mocks were in memory only and removed by reload; no backend mutation was made.
- Final gates: `npm test -- --run` PASS (44/44); `go test ./...` PASS; `npm run build` PASS (includes TypeScript); `git diff --check` PASS. Lint: UNAVAILABLE - no ESLint configuration exists.
- Nonfatal Vite/jsdom/line-ending warnings remain. Zod inclusion brings the current JS bundle just above Vite's 500kB warning threshold; route-level code splitting is follow-up performance debt, not a reason to duplicate validation. Portal dark-theme inheritance and full live zoom verification remain existing debt. API path-update clearing behavior remains unchanged by this presentation migration.

## Direct Frontend URL Serving

Status: VERIFIED in handler and real HTTP server tests. Direct browser requests to `/services/:id/health` previously reached a plain Go FileServer and returned 404 because frontend routes are not disk files. Navigation from the loaded index worked because the client owned navigation.

- The server now uses a frontendHandler that serves existing assets normally and returns the application entry page for missing known frontend routes on GET/HEAD, without redirecting or mutating the requested URL. Service deep links, including the Availability compatibility alias, and existing dashboard/environment/health/incident/alert/event/security routes are covered.
- Route fallback is restricted to known frontend roots and excludes file extensions. Unknown routes, missing assets and unknown RPC/API paths are not converted to HTML. Registered API/infrastructure handlers retain mux priority and their existing authentication/authorization semantics. Unsupported methods on frontend assets/routes return 405 with Allow: GET, HEAD.
- Reuses Go FileServer for index serving, content types, HEAD and asset handling; no frontend/router/API contract or new dependency changes. Future top-level frontend routes must be added to the explicit fallback allowlist.
- Tests exercise the exact reported deep link, root/list/detail/trailing-slash/query URLs, legacy Availability link, all frontend roots, HEAD, JavaScript assets, missing assets/API/RPC routes, original URL preservation and unsupported methods. A started-server test verifies real HTTP deep linking, missing assets/RPC routes, and that unauthenticated API requests still return 401.
- Deployment requires rebuilding/restarting the backend that serves port 9700. No running production server or database was modified. Gates: `go test ./...` PASS; `npm test -- --run` PASS (44/44); `npm run build` PASS (includes TypeScript); `git diff --check` PASS. Existing nonfatal build/test warnings remain. Lint: UNAVAILABLE - no ESLint configuration exists.

## Health Results and Canonical Check Management

Status: AUDITED -> DESIGNED -> IN PROGRESS -> MIGRATED -> VERIFIED (behavior, design-system compliance, accessibility tests and fixture/layout review; live-data zoom certification remains explicit debt).

Audit: ListHealthResults filters only check/instance with newest-first numeric-offset pagination; frontend retrieves 20/check and discards pagination. Persistence has check/time and instance/time indexes, but no global chronological index. Global Health mixes legacy metric cards, duplicate Health/resource context, raw-reference forms and a separate results feed. Service Health has a five-result preview but expands available history in a drawer. Registration follow-up has another field/default/validation model. UpdateHealthCheck accepts enabled/timing/thresholds/description only: identity, target, type and metadata must remain immutable during edit.

Design: canonical `/health/results` with URL query state; authenticated read-only `/api/v1/health/results` with date bounds, service/environment/instance/endpoint/port/type/check/outcome, newest/oldest order, bounded page sizes and numeric-offset tokens. One joined current-catalog projection avoids per-row API calls and preserves soft-deleted relationships; it is not an execution-time target snapshot. Existing RPC remains compatible. One chronological expression index supports global date ordering; existing relational indexes remain.

Canonical HealthCheckForm owns defaults, Zod validation, human-readable dependent targets, mutation/loading/error states and Create/Edit actions. Global, Service, Instance and post-registration follow-up reuse it. Service preview remains five rows and links to the dedicated results page. No extra Health heading is added inside the Service tab. Dead forms/history markup have been removed after consumers migrated.

### Health Result History

- Canonical frontend route: `/health/results`. Service/check/environment scope is carried by optional `serviceId`, `checkId`, `environmentId` query parameters. Global Results and Service/check View all results use the same page. Back to Health returns to the selected service when service scope exists; browser Back/Forward restores query state. The existing SPA fallback covers direct result URLs and now has an explicit deep-link regression test.
- Read-only backend route: `GET /api/v1/health/results`, inside the existing authentication/read-scope/audit/rate-limit middleware. Application-key environment restrictions are applied in SQL and cannot be bypassed by choosing an unauthorized environment filter. Existing Connect RPC and execution/state/availability semantics remain unchanged.
- Optional filters: `from`, `to` (inclusive RFC3339 instants), `serviceId`, `environmentId`, `instanceId`, `endpointId`, `port` (1-65535), `type` (all known non-unspecified backend enums, including retained UDP/heartbeat checks), `checkId`, `status` (healthy/unhealthy execution outcome). These are parameterized server-side predicates, not browser-only filtering of all history. Result success does not override instance threshold-based current state.
- Ordering: `sort=newest` by default or `sort=oldest`, with stable result-ID tie-breaking. Pagination follows existing numeric-offset tokens: `pageSize` (25/50/100 UI options, backend maximum 100) and `pageToken`; server fetches one extra row to determine Next. No unbounded history fetch or total-count query.
- URL preserves filters, sort and pagination; date controls show local time but serialize UTC instants so shared links preserve the same interval across time zones. Dependent selections clear stale instance/endpoint/check filters; changing filters resets pagination. Invalid dates/inverted ranges and server validation failures have explicit errors, not render crashes. Cancelled queries cannot overwrite a newer search.
- Joined projection returns names/context, endpoint protocol/port, check type, target address/path/expected status and observed outcome/duration/failure/status code without per-result requests or arbitrary response metadata. Results table is compact, mobile uses the same data as resource rows, and details use the shared Drawer. Failure reason is a separate labeled property, not concatenated status text. Missing durations remain Not recorded.
- Migration `023_health_results_chronology.sql` adds one chronological expression index on `julianday(timestamp), id`, matching date/order predicates. Existing check/time, instance/time and catalog indexes remain; no new entity, uniqueness constraint or execution mutation is introduced.
- Shared FilterBar adds optional responsive progressive disclosure: desktop exposes all filters; mobile keeps the first four controls visible and exposes the rest through an accessible More filters button with applied-filter count. Existing FilterBar consumers are unchanged. No page-specific filter CSS or new library.

### Health Check Form

- Canonical component: `HealthCheckForm`; shared defaults/validation: `lib/health-check-form.ts`. Consumers: global Health, Service Health, Instance-context handoff, Edit, and optional post-registration monitoring.
- Fields: name, instance, endpoint (optional instance-address fallback), supported type, path, expected HTTP status/range, interval, timeout, failure/recovery thresholds, Enabled and description. Known service/environment/instance selectors are omitted; global selection is dependent and human-readable. Defaults are readiness, HTTP, /healthz, expected 200-299, interval 10s, timeout 3s, failure threshold 3, recovery threshold 2, Enabled true.
- Create and Edit share field order, control family, validation, loading/error presentation and actions. The unchanged Update RPC supports Enabled, timings, thresholds and description only. Name/target/type/path/status configuration are read-only on Edit rather than advertising unsupported changes. Same-instance endpoint selection is validated; names, types, HTTP status ranges and positive integer timing/threshold values use one schema.
- Edit validates only the supported mutable payload, so a stored UDP/heartbeat check or immutable legacy metadata cannot block a valid settings update. Such stored types remain visible/read-only and searchable; new creation offers only the four types the existing executor supports. Execution semantics are unchanged.
- Optional monitoring begins only after instance/endpoints exist. Failed health mutation Retry invokes only CreateHealthCheck, never RegisterRuntime. A successful mutation followed by failed refresh is marked saved and cannot be resubmitted; it presents a refresh instruction instead. Success callbacks close the editor or return to the registration success state before refreshing catalog data.
- Removed three presentation/form models: global legacy create JSX, Service HealthDialog fields/edit bindings, and registration follow-up fields. Removed corresponding App creation/update/follow-up handlers, duplicate defaults and validation, unreachable rollback Services/Availability/Health branches, inline full-history drawer, and dead health-config/health-result-row/hero-health-card/runtime-health CSS rules. API helpers remain compatible. HealthDialog is now only a context/surface wrapper, not a second form implementation.

### Header Rules

- One primary heading per surface. Global Health has one PageHeader; Service Health starts with actual operational sections beneath the service resource header/tabs. Health-check/result drawers use one resource title. History has one Health results heading and a contextual Back action; no duplicate breadcrumb/category/Health title stack.
- Overview presentation, canonical Service tabs, instance endpoint ownership, availability calculations, manual execution and authorization remain unchanged. Instances retain monitoring summary/handoff only, not embedded history or configuration forms.

### Verification and Debt

| Surface | Status | Verified scope |
| --- | --- | --- |
| Global Health | VERIFIED | Shared primitives, one header, scoped overview/checks, canonical Results navigation and shared editor |
| Service Health | VERIFIED | Unchanged availability/current-state semantics, five-result preview, scoped canonical history and check drawer |
| Health results | VERIFIED | Server filters/order/pages, URL state/dependent controls, empty/error/details and responsive resource rows |
| HealthCheckForm | VERIFIED | Global/scoped create, immutable-context edit, defaults/validation, loading, health-only retry and saved-refresh failure |
| Responsive/accessibility review | VERIFIED within fixture/review scope | Labeled controls/status text, WCAG 2A/AA axe tests, inherited Radix focus/keyboard behavior and viewport measurements |

- Final checkpoint: `npm test -- --run` PASS (57/57); `go test ./...` PASS; `npm run build` PASS (includes TypeScript); `git diff --check` PASS. Lint: UNAVAILABLE - no ESLint configuration exists. No dependency or browser-runner additions. Existing nonfatal Vite/Zod/jsdom and bundle-size warnings remain.
- Repository tests cover each query dimension, inclusive timezone-aware date ranges, combined filters, newest/oldest order, continuation/final page, readable target projection, no matches, allowed/disallowed environment scope, authentication and invalid query bounds. Frontend tests cover query serialization/history entry/navigation, page-size reset, filter disclosure, structured details/errors, canonical create/edit/defaults/scoped selectors, loading, duplicate-name error, optional retry and refresh failure without duplicate creation. Existing Run/state/availability, instance/endpoint, focus and authorization tests remain passing.
- DOM/CSS fixture review at 1920/1440/1280/1024/768/390px found no page overflow in history or horizontal form overflow in the shared Service editor. Editor widths were bounded at 672px desktop and 358px at 390px. Mobile history filter height fell from 846px to 291px with advanced filters collapsed. Live 466px fixture history also had no page overflow. These are rendered layout measurements, not full production screenshots or browser-zoom certification.
- Remaining debt: historical context is joined from current retained catalog rows, not immutable execution-time target snapshots; target changes may alter displayed historical address/path/port. Existing offset pagination can shift under concurrent insertions. Main previews still use bounded legacy RPC pages per configured check (five rendered rows), so high check-count preview fan-out can later be replaced with the new query. Full live-data/zoom QA, portal dark-theme inheritance, and route-level code splitting for the existing 500kB bundle warning remain follow-ups.
- Deployment requires rebuilding/restarting the backend to register the new query and apply migration 023 through normal startup migrations. No production database or running backend was modified during fixture QA.

## Environments and Global Health Refinement Audit

Status: AUDITED -> DESIGNED -> IN PROGRESS -> MIGRATED -> VERIFIED within the regression, accessibility and rendered-fixture scope below.

- At audit time, Environments rendered OperationalWorkspace -> local Environments -> PageHeader + titled table Card + permanent create Card; the action only focused the inline key field. Its counts used shell-scoped topology, incorrectly suggesting zero resources in other environments. A dead App rollback branch duplicated the form.
- Contract: Environment has immutable unique Key, Name, optional Description, free-form string Tier, Enabled and Tags. Create accepts Key/Name/Tier/Description/Tags and always enables the environment. Update accepts Name/Description/Tier/Enabled/Tags, not Key. No enum, key regex or description length limit exists; frontend will not invent one. Numeric tiers will display as Tier N, blank as Untiered, other labels retain their meaning. Enabled is editable only in Edit; tags are preserved.
- Replacement: EnvironmentsWorkspace -> PageHeader + compact search/status filters + one Workspace table/mobile ResourceList + overflow actions (Edit and View services). Create/Edit share EnvironmentForm with centralized defaults/Zod validation, field errors and loading/API-error handling inside the shared Dialog. No permanently visible form or redundant Environment scopes heading. Counts load all authorized topology pages independently of the selected shell environment; failed counts remain unavailable, not zero.
- At audit time, Global Health rendered an unframed instance ResourceList with name/address only, four metrics and one full-width Status select. Service/Environment relationships were already available in deployments; check counts/Enabled and current-state timestamps were also available. No state may be inferred from result absence when the backend state is unavailable.
- Replacement: shared Workspace -> five-value StatGroup -> compact FilterBar (Status, Service, Environment, Search) -> structured instance table/mobile rows + overflow actions. Unknown explanation distinguishes no checks, disabled monitoring, returned state awaiting transition, or current state unavailable; no invented Never executed/No recent result claim. Rows link to the existing Service Health and scoped canonical result search; monitoring actions reuse HealthCheckForm. Checks and Results remain canonical and no new history/detail model is introduced.
- Shared refinements: opt-in bounded FilterBar controls and five-column StatGroup, with responsive wrapping. Existing consumers keep their behavior. Delete local Environment presentation/form and dead rollback markup only after replacements have coverage. No backend/entity/schema/authorization rewrite is needed.

### Environments

- Primary view: EnvironmentsWorkspace is a resource management list, with compact Search/Status filters, desktop Table and mobile ResourceList. Name is primary; immutable Key and description are secondary metadata. Tier displays free-form labels unchanged, numeric values as Tier N and blank values as Untiered. Canonical StatusBadge represents Enabled/Disabled; the selected shell scope has a restrained neutral row background.
- Create/Edit: one EnvironmentForm, one Zod schema and one defaults function. The shared md Dialog contains aligned fields and common footer actions. Create sends the existing request and relies on the backend Enabled=true default; Edit offers Switch, keeps Key read-only and retains Tags. There is no fabricated tier enum, key regex, length constraint or create-time enabled flag.
- Inline create form: REMOVED. The PageHeader owns Create; the empty state provides the only Create action when no rows exist. Overflow actions own Edit and View services. View services changes shell environment and navigates to the canonical catalog; successful creation still selects the new environment before reloading its catalog.
- Counts use all pages of authorized instances/deployments, independently of the selected shell scope, with deduplicated service counts per environment. ListEnvironments also follows continuation tokens including disabled rows. Loading/failure counts are explicitly Loading/Unavailable, with Retry, not fabricated zeroes. Concurrent key collisions receive a field-level error; success followed by refresh failure cannot recreate the environment.

### Global Health

- Overview: one PageHeader and workspace surface; five compact stats (Instances, Healthy, Unhealthy, Unknown, Checks), a bounded FilterBar and structured rows. Desktop columns are Instance/address, Service, Environment, Monitoring, Status and Actions; tablet/mobile use the same contextual data in ResourceRows. Rows are locally paginated at 25 within the existing loaded catalog; historical results retain server pagination.
- Overview filters: Status, Service, Environment and Search (instance/address/service/environment). Stats describe the service/environment/search scope before the Status subset so the distribution remains useful. Filter changes reset paging; changing the shell environment clears stale local service/environment filters. Overview filters do not silently change the Checks tab's existing shell-scoped catalog.
- Unknown explanation is metadata, not a new badge language: No health check configured, Monitoring disabled, Instance disabled, Awaiting health transition when returned state has a timestamp, or Current state unavailable. No recent/never-executed claims are inferred without supporting data. Threshold-based current state is never substituted with a last execution result.
- Instance links and View service health reuse Service > Health. View results carries instance/service/environment scope into the existing canonical Results page. Run check is available for an enabled instance with exactly one enabled check; multiple-check workflows remain in Service Health. Configure monitoring passes known instance/service/environment to HealthCheckForm without redundant selectors.
- Checks: unchanged canonical ServiceHealthView/HealthCheckForm management, including edit and existing API contracts. Results: unchanged canonical HealthResultsPage at `/health/results`. The navigation button sits alongside, not inside, the ARIA tablist because it opens a route rather than a tab panel. No second form/history implementation was added.

### Shared Components and Removals

| Component | Decision | Consumers / removal condition | Status |
| --- | --- | --- | --- |
| Local OperationalWorkspace Environments and App rollback form | REMOVE | No remaining consumers; canonical flow covered by tests | REMOVED |
| EnvironmentForm + environment-form defaults/schema | KEEP | Create and Edit | VERIFIED |
| EnvironmentsWorkspace | KEEP | Canonical environment route | VERIFIED |
| HealthWorkspace old flat overview | RECREATE | Contextual operational overview; retains canonical checks/history | VERIFIED |
| ActionMenu | KEEP | Shared Radix menu with Lucide trigger; releases menu focus before opening a subsequent surface | VERIFIED |
| IconButton forwardRef | REFINED | Accessible Radix asChild trigger | VERIFIED |
| FilterBar compact density | REFINED | Opt-in 180px bounded fields; wraps without stretching one filter across the workspace | VERIFIED |
| StatGroup columns | REFINED | Supports 2/3/4/5 related stats; mobile two-column rhythm retained | VERIFIED |
| DialogContent opener handling | REFINED | Capture external focus before nested scopes/autoFocus; Escape returns to connected trigger | VERIFIED |
| Legacy environment-overview-row styles | REMOVE | No live consumers; seven dead selector occurrences/rules removed | REMOVED |

- Existing shared form controls, Table, ResourceList, Workspace, StatusBadge, Skeleton, EmptyState, Alert and Radix dialogs are reused. No dependency, backend, schema, authorization, route contract or new UI framework changes in this pass.

### Refinement Verification and Remaining Debt

| Surface | Status | Verified scope |
| --- | --- | --- |
| Environments list / filters / counts | VERIFIED | Empty/status/tier/context, scope-independent continuation pages, count failure, navigation |
| EnvironmentForm Create/Edit | VERIFIED | Required fields, duplicates/local and server, immutable Key, Tags, Enabled, success/loading and keyboard focus |
| Global Health Overview | VERIFIED | Stats, combined filters, context, truthful Unknown explanations, loaded-catalog pagination, scoped actions |
| Global Health Checks / Results | VERIFIED | Canonical shared create/edit and history navigation preserved |
| Responsive / accessibility review | VERIFIED within fixture/test scope | Viewport measurements, labels/errors/status text, menu keyboard operation, focus return and axe WCAG 2A/AA |

- Added 24 regression tests. The new axe overview test found and fixed an invalid button inside a tablist; menu-to-dialog focus timing and Environment autoFocus/opener handling were corrected, not hidden by assertions. Initial test/build issues were assertion timing/label and unsupported Testing Library option types; all corrected without weakening behavior checks.
- Rendered fixture DOM/CSS review at 1920/1440/1280/1024/768/390px found no horizontal page overflow in Environments or Health. Environment dialog stays bounded at 512px desktop and fits mobile without internal horizontal overflow. Health filter selects measure 180px rather than full workspace width. Five-stat desktop layout becomes two columns on mobile; tables become contextual rows at their breakpoints.
- Live fixture keyboard review verifies menu Enter, dialog initial field focus, Escape and focus return to the visible menu trigger. Automated WCAG 2A/AA checks cover the Environment form with errors and Health Overview; existing shared dialog/tab and canonical check/history tests remain intact. Fixtures were in memory only, removed by reload, and did not mutate the backend.
- Gates: `npm test -- --run` PASS (81/81); `npm run build` PASS (includes TypeScript); `go test ./...` PASS; `git diff --check` PASS. Lint: UNAVAILABLE - no ESLint configuration exists. Existing nonfatal Vite/Zod/jsdom/line-ending and >500kB bundle warnings remain.
- Remaining debt: full production-data screenshot/zoom QA is not certified; portal dark-theme inheritance and route-level code splitting remain follow-ups. The existing catalog loader still omits unavailable GetInstanceHealthState calls, so states remain explicitly Unknown until supported state data arrives (for example a manual run). Global Health summarizes the currently loaded authorized catalog, not a new server aggregation; large-catalog pagination/aggregation beyond existing loader limits is follow-up data-flow work. Environment counts are fully paged and not affected by those loader limits. No unsupported environment Delete workflow was introduced.

## Health > Checks Scalability Reset

- Responsibility: the canonical Health Check management catalog. Overview owns current status, distribution, availability and instance-level attention; Results owns execution history. Checks no longer renders overview metrics or an unbounded vertical health-check list.
- Query model: `GET /api/v1/health/checks` returns a joined list projection with human-readable service, environment, instance, endpoint and latest-result fields. Search covers check and target names, address and port. Filters are `serviceId`, `environmentId`, `instanceId`, `endpointId`, `enabled` and `latestStatus`; pagination is server-side with deterministic name ascending order.
- Default page size: 25. Supported sizes: 25, 50 and 100. The catalog displays a range such as `1–25 of 83` and keeps filter/header structure stable while table skeleton rows load.
- Catalog columns: Name, combined Target, Interval, Last result, Last run, separate Enabled/Disabled Status and overflow Actions. Last result uses Healthy/Unhealthy/Unknown; Status uses Enabled/Disabled. Configuration details are excluded from rows.
- Detail pattern: a right-side Drawer with a compact identity header, target, configuration, latest result, action controls and a limited recent-execution handoff. `View all results` opens the canonical `/health/results?checkId=...` implementation.
- Service Health preview: maximum five checks, with `View all N checks` opening the same canonical Checks catalog with `serviceId` scope. Service Health retains its overview/availability responsibility and does not duplicate catalog management.
- Canonical form: `HealthCheckForm` remains the only create/edit form for global, service-scoped and instance-scoped actions. Canonical `HealthResultsPage` remains the only execution-history browser.
- Replaced presentation: the global Checks dependency on the overview-oriented `ServiceHealthView` resource list was removed and recreated as the paginated `CheckCatalog`/`CheckDetail` model. The Service Health list is bounded to a five-row preview.

## Health Results Investigation Refinement

- Purpose: canonical Health Check execution history and investigation tool. The page has one `Health results` header with concise search-oriented description; Back to Health remains secondary navigation.
- Primary filters: From, To, Service, Result and free-text Search. More filters: Environment, Instance, Endpoint, Port, Check type, explicit Health check, ordering and page size. The shared `FilterBar` persistent disclosure mode keeps secondary controls collapsed on desktop and mobile.
- Active filter summary: non-primary scope, search and result filters render as removable chips with a Clear all action. Filter changes preserve URL state and reset numeric page tokens.
- Search: server-side `search` matches service/display name, environment, instance, address, endpoint, health-check name, port and failure message. Existing date, scope, result, type and ordering predicates remain server-side.
- Ordering and pagination: Newest first remains the default; Oldest first is available. Numeric offset pagination and page sizes 25/50/100 remain unchanged. The footer uses a compact range (`1–25+`) with Previous/Next controls; the plus indicates another server page because the existing API intentionally has no total-count query.
- Target display: the table groups Service/Environment, Instance, and Endpoint/Protocol/Port into one compact Target column. Rows open the detail Drawer by row click or a compact arrow action; repeated Details text buttons were removed.
- Detail interaction: the existing Drawer was recreated around Execution, Target, Health Check and Failure sections, with optional related Service and Instance navigation. Missing duration is shown as a muted em dash with an accessible `Duration not recorded` explanation.
- Transient feedback: unrelated shell success banners are hidden while Results is active so registration/auth feedback no longer consumes Results page space. Result query failures remain compact contextual error feedback with Retry.

## Control Sizing and Alignment

- Control sizing is global: `sm` is 32px, `md` is 36px and `lg` is 40px. Inputs and selects use the shared `md` field height by default; primary, secondary, ghost, danger and link buttons preserve the same height for a given size.
- IconButton uses a square equivalent of its Button size: 32px, 36px or 40px. Table-row actions use `sm`; toolbar and dialog actions use `md`.
- StatusBadge and FilterChip use a compact 26px geometry with shared text, padding, radius and icon spacing. Filter chips truncate long values and expose a 20px close hit area.
- FilterBar uses shared 12px gaps, shared field-label rhythm and bottom alignment for controls. More filters is a standard compact Button with the sliders icon; Clear all uses the same ghost-button family as its surrounding controls.
- Health Results keeps the shared PageHeader action treatment, grouped target table alignment and middle-aligned row actions. No page-specific control-sizing CSS is permitted; page-specific CSS may only describe content layout.

## Health Results Visual Model

- The Results page keeps the existing route, query parameters, filtering behavior, server continuation tokens and detail Drawer. Its visual order is PageHeader, one bordered results surface, compact toolbar, optional active-filter row, results content and integrated pagination footer.
- Columns are Timestamp, Target, Check, Result, Duration and Action. The desktop table uses proportional widths of 22%, 32%, 14%, 14%, 10% and 8%; the shared table header has a restrained neutral surface, stronger muted text and a clear bottom rule. Rows use compact consistent cell padding and subtle hover feedback. The row itself is not interactive; the labeled IconButton opens details.
- Target hierarchy is three levels: Service · Environment; Instance; then Endpoint · Protocol :Port for endpoint checks, or the actual address / Instance-level check for instance targets. Port zero is never presented as an endpoint port. The same presentation rule applies in desktop rows, mobile rows and the detail Drawer.
- Absolute timestamp is primary and relative time is secondary. Check name is primary and type is secondary. Missing duration is a muted em dash with the accessible title `Duration not recorded`.
- The toolbar uses shared 36px controls with aligned labels, compact gaps and the standard More filters button. From and To remain explicit datetime fields. Page size belongs in the shared Pagination footer, not the filter toolbar. Active filter chips render only when their query value is present; unresolved catalog names display the actual selected key.
- Shared compact sizing: toolbar controls 36px; row IconButton 32px; FilterChip 26px; StatusBadge 26px. Chips use a subtle neutral fill and light border. Status badges use semantic tint, border and dot with readable status text.
- Row density targets 56–68px for the three-line Target presentation. Loading retains the toolbar and shows table skeleton rows. Empty and query-error feedback stay inside the results surface. Mobile uses contextual stacked result rows and does not force horizontal scrolling.

## Health State Consistency

- Service list, selected Service header, Overview and Service Health render the same backend `healthStatus`; global Health renders the matching backend Instance states. The Service filter uses that same status. Missing results display Unknown, never Healthy.
- Current status includes Instances, Monitored, Healthy, Unhealthy and Unknown. Degraded and Disabled counts appear when present. Monitoring coverage comes from active enabled checks with usable targets; Enabled remains separate from operational health.
- A new result or health-change event refreshes the shared status projection. A 30-second refresh also lets old results expire to Unknown. Availability stays a historical percentage; Incident Open/Resolved is a lifecycle state, not an alternate health badge.
- The backend matrix and freshness rule live in [HEALTH_STATE.md](HEALTH_STATE.md). The UI does not infer health from Incident counts, availability or absent cached state rows. The sidebar reports Incident count without claiming the whole system is Healthy when no Incident is open.

## Incident Lifecycle and Resolution Semantics

- Incident resolution is no longer a generic `Resolve` mutation. Open incidents with an originating `metadata.health_check_id` expose `Verify recovery`, which executes that check again, persists the new Health Result and resolves the incident only when the result is healthy.
- A failed verification is an operational outcome: the incident remains Open and the UI reports `Condition still present` with the latest failure reason. Incidents without a verifiable originating Health Check do not expose Verify recovery.
- `Resolve manually` is a separate confirmed administrative action requiring a note. It records `ManualOverride`, the note and the operator event; it does not reuse the verification handler.
- Scheduled recovery continues to resolve active incidents after the health state recovers, recording `AutoRecovered` and the successful Health Result as evidence. Conditional incident updates and one resolution event per open-to-resolved transition provide idempotency across scheduled and user-triggered work.
- Incident records now persist `resolution_method`, `resolution_note`, `resolved_by` and `resolution_evidence_health_result_id`. The explicit API operations are `VerifyIncidentRecovery` and `ResolveIncidentManually`; the legacy ambiguous Resolve RPC was removed.
- Incident list default state is Open. Open rows offer Verify recovery when supported, Resolve manually and Details. Resolved rows offer Details only and show resolution method, note and evidence in the detail dialog. Service, environment and instance names are preferred over internal IDs in the list/detail presentation.

## Security

- Users, API tokens, Application keys and Sessions remain separate tabs under the standard PageHeader. Security uses the same compact description, full-width workspace, underline Tabs, SectionHeader, FilterBar, StatusBadge, Table, mobile ResourceList, ActionMenu and Pagination as other resource screens. No Security-only breadcrumb or duplicate page title remains.
- Each tab has one section heading and aligned primary action. Users provide create/list and role visibility; the backend offers no user edit/delete operation, so the Actions cell is empty rather than a misleading dash. API tokens are owner-scoped because the current endpoint lists one owner's tokens at a time; the Owner filter makes this explicit. Keys show Environment access; Sessions show client and last activity.
- Shared DialogHeader, DialogBody and DialogFooter wrap user, token and key creation and revoke confirmation. Forms use FormField, Input/PasswordInput, Select, Checkbox and FormSection. Token and key secrets appear only in the creation dialog, with copy-once guidance, never in lists or general success messages. Closing the dialog clears the secret from UI state.
- Lists use client-side search/status filtering and 25-row pages because the current Security admin endpoints return complete arrays without pagination. Tablet tables hide lower-priority columns; mobile renders structured ResourceRows with the same status/actions.
- Removed the unreachable inline Security forms, custom tab buttons, grid-table classes, local modal shells and Security-specific CSS. Authentication and authorization APIs are unchanged. Root has no special protected-user UI claim because that restriction is not exposed by the current backend contract.
