# UI/UX Information Architecture Audit

## Screen map

| Current screen | Problem | Proposed screen(s) | Navigation |
| --- | --- | --- | --- |
| Dashboard | Summary, incidents, events, and availability compete for attention. | Dashboard overview | Summary links open the relevant focused resource view. |
| Services | List, service detail, runtime topology, instance and endpoint management, health configuration, events, and create/edit workflows are stacked together. | Services list; Service detail; Runtime/availability; contextual forms | Select a service for detail. Use detail tabs for Overview, Runtime, Health, Incidents, and Events. Create/Edit opens a dialog or focused route. |
| Environments | The environment list and create form are separate goals rendered together. | Environments list; Environment create/edit dialog | Primary Create action opens the form. |
| Health | Global summary, instance health, checks, selected-check detail, results, and creation were all visible simultaneously. | Health Overview; Health Checks; Health Check detail; Results; Create/Edit workflow | Health sub-navigation. Select a check for detail. Create opens a dialog. |
| Incidents | List and selected incident detail are combined, but detail is already contextual. | Incidents list; Incident detail | Select Details and preserve list filters when returning. |
| Alerts | Policies and notification channels are two resource types, with both create/edit forms permanently visible. | Alert Policies; Notification Channels; focused policy/channel forms | Sub-navigation or focused dialogs for create/edit. |
| Security | Session, users, API tokens, and their forms are independent workflows. | Session; Users; API Tokens | Security sub-navigation; create actions open focused dialogs. |
| Events | Timeline and filters have one primary goal. | Events timeline | Resource links provide contextual navigation. |

## Implemented in this pass

Health now follows the resource-management pattern:

- Overview shows global status and instance health.
- Checks shows the list and opens one selected check's detail inline only after selection.
- Results is a separate contextual view for the selected check.
- Create health check is an intentional dialog action and is not rendered below the list.
- Existing health API calls and payloads are unchanged.
- Security now separates Session, Users, and API tokens into focused subviews.
- Alerts now separates Alert policies and Notification channels into focused subviews, with create/edit forms disclosed contextually.
- Environments keeps its list as the default view and discloses creation only on demand.

## Audit rules for follow-up screens

Each resource should have a list, detail, and form boundary where complexity warrants it. The default view should expose the primary goal, while configuration, history, and advanced metadata should require an intentional action. Shared page primitives should establish consistent headers, filters, status summaries, tables, detail sections, empty states, and form workflows without hiding materially different domain behavior behind a generic abstraction.

## Global refinement findings

| Global issue | Root cause | Affected areas | Shared fix |
| --- | --- | --- | --- |
| Control sizing and action alignment varied | Individual row, wizard, toolbar, and page-action selectors overrode the base button contract. | Navigation, headers, tables, service detail, dialogs | Shared control tokens and normalized button sizing, wrapping, focus, and icon hit targets. |
| Dialogs used competing width and overflow rules | Runtime, health, and incident dialogs used separate shells and allowed the whole dialog to scroll. | Add Instance, Health, Incident, service workflows | Bounded dialog sizing, body-only scrolling for runtime registration, `min-width: 0`, and responsive form grids. |
| Form rows forced wide layouts | Repeated editor grids used fixed minimum columns and did not collapse consistently. | Endpoint editors, runtime registration, inline edits, health and alert forms | Shared `minmax(0, 1fr)` grids with 900px and 640px breakpoints. |
| Dense pages had mixed resource workflows | List, detail, history, and creation concerns lived in one render branch. | Health, Security, Alerts, Services | Focused subviews, progressive disclosure, contextual forms, and the screen map above. |
| Metadata and long identifiers could expand rows | Grid children lacked consistent shrink/wrap behavior. | Tables, topology, detail lists, events | Shared child `min-width: 0` and `overflow-wrap: anywhere` rules. |

## Refinement status

Implemented shared changes include spacing tokens (`4/8/12/16/24/32`), global box sizing, bounded page containers, normalized typography levels, button/control dimensions, responsive form grids, shared tab/sub-navigation behavior, status sizing, and dialog overflow rules. Existing API contracts and domain terminology remain unchanged.

Remaining page-specific exceptions are the service workspace topology and wizard, which intentionally use denser layouts for hierarchical deployment and multi-step registration workflows. They now inherit the global sizing and overflow rules, but their domain-specific hierarchy should remain visually distinct.

## Interaction audit

| Symptom | Shared root cause | Affected screens | Systemic correction |
| --- | --- | --- | --- |
| Buttons and action groups varied by screen | Raw buttons accumulated local dimensions and spacing rules. | All pages, dialogs, endpoint rows, service detail | Added `Button`, `IconButton`, and `ActionGroup` primitives with explicit variants and sizes. |
| Tabs could inherit surrounding layout height | Service tabs and sub-navigation were separate implementations. | Services and Health | Added a shared keyboard-aware `Tabs` primitive with fixed control height and active underline. |
| Resource navigation used mutation-style buttons | `ResourceLink` rendered a button for navigation callbacks. | Incidents, Events, Services, Dashboard | Resource references now render semantic links while preserving SPA callbacks. |
| Dialog close actions differed | Runtime and incident dialogs used separate close controls. | Add Instance and Incident detail | Standardized icon-button close labels and hit targets; Add Instance also closes on Escape. |
| Endpoint removal consumed row width | Removal was a text action in repeated editor rows. | Runtime registration and endpoint editing | Runtime registration now uses a compact accessible icon button. |
| Health Results duplicated tab navigation | Results included a second Back action already represented by the tab strip. | Health | Removed the redundant navigation control. |

The interaction primitives live in `web/src/components/OperationsUI.tsx`; their shared sizing, focus, overflow, and responsive rules live in `web/src/App.css`.

## Density refinement

The frontend now has a compact operational density contract layered over the shared spacing tokens. Lists, tables, metadata, filters, pagination, panel headings, and repeated rows use reduced content-driven rhythm; forms and dialogs retain the default control size. The Services master/detail layout uses a bounded list column (`clamp(300px, 25vw, 400px)`) so detail content receives the remaining desktop width. Short service metadata is grouped through the shared `Metadata` component, while mobile layouts collapse the same structures to one column without introducing horizontal scrolling.
