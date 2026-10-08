import { FormEvent } from "react";
import { EnvironmentsWorkspace } from "./EnvironmentsWorkspace";
import {
  AlertPolicy,
  AvailabilitySummary,
  Environment,
  EventRecord,
  Incident,
  NotificationChannel,
  ServiceDeployment,
  ServiceInstance,
  Service,
} from "../api";
import {
  formatActivityTimestamp,
  formatAvailability,
  formatDuration,
  formatEventType,
  formatTimestamp,
} from "../utils/format";
import { Badge, StatusBadge } from "./ui/badge";
import { Button } from "./ui/button";
import {
  Card,
  EmptyState,
  FilterBar,
  PageHeader,
  SectionHeader,
  Workspace,
  Section,
  Stat,
  StatGroup,
  ActivityList,
} from "./ui/layout";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { FormField, Input, SearchInput, Textarea } from "./ui/form";
import { Checkbox, Select } from "./ui/controls";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "./ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";

type PolicyForm = {
  environmentId: string;
  deploymentId: string;
  notifyOn: string;
  cooldownMinutes: number;
  sendRecoveryNotification: boolean;
  channelIds: string;
};
type ChannelForm = {
  type: string;
  name: string;
  description: string;
  url: string;
  smtpHost: string;
  smtpPort: string;
  from: string;
  to: string;
};

type OperationalWorkspaceProps = {
  view: "dashboard" | "environments" | "incidents" | "alerts" | "events";
  loading: boolean;
  environments: Environment[];
  services: Service[];
  deployments: ServiceDeployment[];
  instances: ServiceInstance[];
  incidents: Incident[];
  filteredIncidents: Incident[];
  selectedIncident?: Incident;
  events: EventRecord[];
  filteredEvents: EventRecord[];
  availability: {
    availability24h?: AvailabilitySummary;
    availability7d?: AvailabilitySummary;
    availability30d?: AvailabilitySummary;
  };
  healthChecksCount: number;
  alertPolicies: AlertPolicy[];
  notificationChannels: NotificationChannel[];
  selectedEnvironmentId: string;
  currentEnvironmentName: string;
  incidentStateFilter: string;
  incidentSearch: string;
  eventTypeFilter: string;
  eventResourceFilter: string;
  eventSearch: string;
  eventTypes: string[];
  eventResourceTypes: string[];
  alertsSection: "policies" | "channels";
  policyForm: PolicyForm;
  channelForm: ChannelForm;
  editingPolicyId: string;
  editingChannelId: string;
  savingPolicy: boolean;
  savingChannel: boolean;
  testingChannelId: string;
  resolvingIncidentId: string;
  onIncidentStateFilter: (value: string) => void;
  onIncidentSearch: (value: string) => void;
  onEventTypeFilter: (value: string) => void;
  onEventResourceFilter: (value: string) => void;
  onEventSearch: (value: string) => void;
  onAlertsSection: (value: "policies" | "channels") => void;
  onEnvironmentSaved: (
    environment: Environment,
    created: boolean,
  ) => Promise<void>;
  onViewEnvironmentServices: (environmentId: string) => void;
  onPolicyForm: (value: PolicyForm) => void;
  onChannelForm: (value: ChannelForm) => void;
  onCreatePolicy: (event: FormEvent<HTMLFormElement>) => void;
  onCreateChannel: (event: FormEvent<HTMLFormElement>) => void;
  onCancelPolicy: () => void;
  onCancelChannel: () => void;
  onEditPolicy: (policy: AlertPolicy) => void;
  onEditChannel: (channel: NotificationChannel) => void;
  onTestChannel: (id: string) => void;
  onResolveIncident: (id: string) => void;
  onSelectedIncident: (id: string) => void;
  onViewChange: (
    view: "environments" | "services" | "incidents" | "alerts" | "events",
  ) => void;
  onSelectService: (id: string) => void;
};

const fieldClass = "grid gap-1.5 text-sm font-medium text-[var(--text)]";
const muted = "text-xs text-[var(--text-muted)]";

export function OperationalWorkspace(props: OperationalWorkspaceProps) {
  if (props.view === "dashboard") return <Dashboard {...props} />;
  if (props.view === "environments")
    return (
      <EnvironmentsWorkspace
        environments={props.environments}
        loading={props.loading}
        selectedEnvironmentId={props.selectedEnvironmentId}
        onSaved={props.onEnvironmentSaved}
        onViewServices={props.onViewEnvironmentServices}
      />
    );
  if (props.view === "incidents") return <Incidents {...props} />;
  if (props.view === "alerts") return <Alerts {...props} />;
  return <Events {...props} />;
}

function Dashboard({
  services,
  environments,
  incidents,
  events,
  availability,
  alertPolicies,
  notificationChannels,
  healthChecksCount,
  currentEnvironmentName,
  onViewChange,
  onSelectedIncident,
  onSelectService,
}: OperationalWorkspaceProps) {
  const open = incidents.filter(
    (incident) => incident.state === "INCIDENT_STATE_OPEN",
  );
  const availability24h = availability.availability24h?.availabilityPercent;
  return (
    <div className="grid gap-5">
      <PageHeader
        title="Dashboard"
        description={`Operational summary for ${currentEnvironmentName.toLowerCase()}.`}
      />
      <Workspace aria-label="Operational summary">
        <StatGroup label="System summary">
          <Summary
            label="Availability"
            value={formatAvailability(availability24h)}
            detail="Last 24 hours"
            status={
              availability24h === undefined
                ? "unknown"
                : availability24h < 99
                  ? "degraded"
                  : "healthy"
            }
          />
          <Summary
            label="Services"
            value={String(services.length)}
            detail={`${environments.length} environments`}
          />
          <Summary
            label="Open incidents"
            value={String(open.length)}
            detail="Needs triage"
            status={open.length ? "open" : "healthy"}
          />
          <Summary
            label="Alert policies"
            value={String(alertPolicies.length)}
            detail={`${notificationChannels.length} channels · ${healthChecksCount} checks`}
          />
        </StatGroup>
        <div className="mt-4 grid gap-5 xl:grid-cols-2">
          <Section
            title="Exceptions"
            description="Active incidents in the current scope"
            action={
              <Button
                variant="link"
                size="sm"
                onClick={() => onViewChange("incidents")}
              >
                View incidents
              </Button>
            }
          >
            {open.length === 0 ? (
              <EmptyState
                title="No active incidents"
                description="All monitored services are operating normally in this scope."
              />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Status</TableHead>
                    <TableHead>Reason</TableHead>
                    <TableHead>Service</TableHead>
                    <TableHead>Opened</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {open.slice(0, 6).map((incident) => (
                    <TableRow
                      key={incident.id}
                      onClick={() => onSelectedIncident(incident.id)}
                      className="cursor-pointer"
                    >
                      <TableCell>
                        <StatusBadge status="open" />
                      </TableCell>
                      <TableCell>
                        <strong className="font-medium">
                          {incident.reason || "Open incident"}
                        </strong>
                        <div className={muted}>
                          {incident.impactSummary || "Impact not recorded"}
                        </div>
                      </TableCell>
                      <TableCell>
                        <Button
                          variant="link"
                          size="sm"
                          onClick={(event) => {
                            event.stopPropagation();
                            onSelectService(incident.serviceId);
                            onViewChange("services");
                          }}
                        >
                          {incident.serviceId || incident.instanceId}
                        </Button>
                      </TableCell>
                      <TableCell className={muted}>
                        {formatTimestamp(incident.openedAt)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Section>
          <Section
            title="Recent events"
            description="Latest activity for the current scope"
            action={
              <Button
                variant="link"
                size="sm"
                onClick={() => onViewChange("events")}
              >
                View events
              </Button>
            }
          >
            {events.length === 0 ? (
              <EmptyState
                title="No recent activity"
                description="No registry events have been recorded for the selected scope."
              />
            ) : (
              <ActivityList
                items={events.slice(0, 8).map((event) => ({
                  id: event.id,
                  time: formatActivityTimestamp(event.timestamp),
                  event: formatEventType(event.type),
                  context: event.message,
                }))}
              />
            )}
          </Section>
        </div>
      </Workspace>
    </div>
  );
}

function Summary({
  label,
  value,
  detail,
  status,
}: {
  label: string;
  value: string;
  detail: string;
  status?: string;
}) {
  return (
    <Stat
      label={label}
      detail={detail}
      value={
        <span className="inline-flex flex-wrap items-center gap-2">
          {value}
          {status ? <StatusBadge status={status} /> : null}
        </span>
      }
    />
  );
}

function Incidents({
  filteredIncidents,
  incidents,
  selectedIncident,
  incidentStateFilter,
  incidentSearch,
  onIncidentStateFilter,
  onIncidentSearch,
  onSelectedIncident,
  onResolveIncident,
  resolvingIncidentId,
  onSelectService,
  onViewChange,
}: OperationalWorkspaceProps) {
  const open = incidents.filter(
    (incident) => incident.state === "INCIDENT_STATE_OPEN",
  ).length;
  return (
    <div className="grid gap-5">
      <PageHeader
        title="Incidents"
        description={`${open} open incidents require attention in the current environment scope.`}
      />
      <Card>
        <FilterBar>
          <label className={fieldClass}>
            State
            <Select
              value={incidentStateFilter}
              onChange={(event) => onIncidentStateFilter(event.target.value)}
            >
              <option value="all">All incidents</option>
              <option value="open">Open</option>
              <option value="resolved">Resolved</option>
            </Select>
          </label>
          <label className={`${fieldClass} min-w-60`}>
            Search
            <SearchInput
              value={incidentSearch}
              onChange={(event) => onIncidentSearch(event.target.value)}
              placeholder="Search incidents"
            />
          </label>
          <span className={`ml-auto ${muted}`}>
            {filteredIncidents.length} shown
          </span>
        </FilterBar>
        {filteredIncidents.length === 0 ? (
          <EmptyState
            title="No incidents match"
            description="No incidents match the current state and search filters."
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Status</TableHead>
                <TableHead>Reason</TableHead>
                <TableHead>Service</TableHead>
                <TableHead>Environment</TableHead>
                <TableHead>Started</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredIncidents.map((incident) => (
                <TableRow key={incident.id}>
                  <TableCell>
                    <StatusBadge
                      status={incident.state
                        .replace("INCIDENT_STATE_", "")
                        .toLowerCase()}
                    />
                  </TableCell>
                  <TableCell>
                    <strong className="font-medium">
                      {incident.reason || "No reason recorded"}
                    </strong>
                    <div className={muted}>
                      {incident.impactSummary || "Impact not recorded"}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="link"
                      size="sm"
                      onClick={() => {
                        onSelectService(incident.serviceId);
                        onViewChange("services");
                      }}
                    >
                      {incident.serviceId || incident.instanceId}
                    </Button>
                  </TableCell>
                  <TableCell className={muted}>
                    {incident.environmentId}
                  </TableCell>
                  <TableCell className={muted}>
                    {formatTimestamp(incident.openedAt)}
                  </TableCell>
                  <TableCell className={muted}>
                    {incident.resolvedAt
                      ? formatDuration(incident.durationSeconds)
                      : "Open"}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => onSelectedIncident(incident.id)}
                      >
                        Details
                      </Button>
                      <Button
                        size="sm"
                        variant="secondary"
                        disabled={
                          incident.state !== "INCIDENT_STATE_OPEN" ||
                          resolvingIncidentId === incident.id
                        }
                        onClick={() => onResolveIncident(incident.id)}
                      >
                        {resolvingIncidentId === incident.id
                          ? "Resolving"
                          : "Resolve"}
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
      <Dialog
        open={Boolean(selectedIncident)}
        onOpenChange={(open) => {
          if (!open) onSelectedIncident("");
        }}
      >
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>Incident Detail</DialogTitle>
          </DialogHeader>
          {selectedIncident ? (
            <DialogBody>
              <dl className="grid grid-cols-[minmax(100px,auto)_1fr] gap-x-5 gap-y-3 text-sm">
                <dt className={muted}>Status</dt>
                <dd>
                  <StatusBadge
                    status={selectedIncident.state
                      .replace("INCIDENT_STATE_", "")
                      .toLowerCase()}
                  />
                </dd>
                <dt className={muted}>Reason</dt>
                <dd>{selectedIncident.reason || "None"}</dd>
                <dt className={muted}>Impact</dt>
                <dd>{selectedIncident.impactSummary || "None"}</dd>
                <dt className={muted}>Service</dt>
                <dd>
                  <Button
                    variant="link"
                    size="sm"
                    onClick={() => {
                      onSelectService(selectedIncident.serviceId);
                      onSelectedIncident("");
                      onViewChange("services");
                    }}
                  >
                    {selectedIncident.serviceId}
                  </Button>
                </dd>
                <dt className={muted}>Instance</dt>
                <dd>{selectedIncident.instanceId}</dd>
                <dt className={muted}>Opened</dt>
                <dd>{formatTimestamp(selectedIncident.openedAt)}</dd>
                <dt className={muted}>Resolved</dt>
                <dd>{formatTimestamp(selectedIncident.resolvedAt)}</dd>
              </dl>
            </DialogBody>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  );
}

function Alerts({
  alertPolicies,
  notificationChannels,
  alertsSection,
  onAlertsSection,
  policyForm,
  channelForm,
  onPolicyForm,
  onChannelForm,
  editingPolicyId,
  editingChannelId,
  savingPolicy,
  savingChannel,
  testingChannelId,
  onCreatePolicy,
  onCreateChannel,
  onCancelPolicy,
  onCancelChannel,
  onEditPolicy,
  onEditChannel,
  onTestChannel,
  environments,
  selectedEnvironmentId,
}: OperationalWorkspaceProps) {
  return (
    <div className="grid gap-5">
      <PageHeader
        title="Alerts"
        description="Configure health notification policies and the channels that deliver them."
      />
      <Tabs
        value={alertsSection}
        onValueChange={(value) =>
          onAlertsSection(value as "policies" | "channels")
        }
      >
        <TabsList className="border-b border-[var(--border)]">
          <TabsTrigger value="policies">
            Policies ({alertPolicies.length})
          </TabsTrigger>
          <TabsTrigger value="channels">
            Channels ({notificationChannels.length})
          </TabsTrigger>
        </TabsList>
        <TabsContent value="policies" className="mt-4">
          <Card>
            <SectionHeader
              title="Alert policies"
              description="Route health transitions to notification channels."
            />
            {alertPolicies.length === 0 ? (
              <EmptyState
                title="No alert policies"
                description="Create a policy to route health transitions or incident changes to a notification channel."
              />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Scope</TableHead>
                    <TableHead>Triggers</TableHead>
                    <TableHead>Cooldown</TableHead>
                    <TableHead>Channels</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {alertPolicies.map((policy) => (
                    <TableRow key={policy.id}>
                      <TableCell>
                        {policy.deploymentId ||
                          policy.environmentId ||
                          "Global"}
                      </TableCell>
                      <TableCell className={muted}>
                        {policy.notifyOn?.join(", ") || "No triggers"}
                      </TableCell>
                      <TableCell>{policy.cooldownMinutes}m</TableCell>
                      <TableCell>{policy.channelIds?.length ?? 0}</TableCell>
                      <TableCell>
                        <StatusBadge
                          status={policy.enabled ? "enabled" : "disabled"}
                        />
                      </TableCell>
                      <TableCell>
                        <Button
                          size="sm"
                          variant="ghost"
                          onClick={() => onEditPolicy(policy)}
                        >
                          Edit
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>
          <PolicyForm
            {...{
              policyForm,
              onPolicyForm,
              editingPolicyId,
              savingPolicy,
              onCreatePolicy,
              onCancelPolicy,
              environments,
              selectedEnvironmentId,
            }}
          />
        </TabsContent>
        <TabsContent value="channels" className="mt-4">
          <Card>
            <SectionHeader
              title="Notification channels"
              description="Destinations available to alert policies."
            />
            {notificationChannels.length === 0 ? (
              <EmptyState
                title="No notification channels"
                description="Add a webhook or email destination before assigning policies."
              />
            ) : (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Type</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {notificationChannels.map((channel) => (
                    <TableRow key={channel.id}>
                      <TableCell>
                        <strong className="font-medium">{channel.name}</strong>
                        <div className={muted}>
                          {channel.description || "No description"}
                        </div>
                      </TableCell>
                      <TableCell>{channel.type}</TableCell>
                      <TableCell>
                        <StatusBadge
                          status={channel.enabled ? "enabled" : "disabled"}
                        />
                      </TableCell>
                      <TableCell>
                        <div className="flex gap-2">
                          <Button
                            size="sm"
                            variant="ghost"
                            disabled={
                              !channel.enabled ||
                              testingChannelId === channel.id
                            }
                            onClick={() => onTestChannel(channel.id)}
                          >
                            {testingChannelId === channel.id
                              ? "Testing"
                              : "Test"}
                          </Button>
                          <Button
                            size="sm"
                            variant="ghost"
                            onClick={() => onEditChannel(channel)}
                          >
                            Edit
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
          </Card>
          <ChannelForm
            {...{
              channelForm,
              onChannelForm,
              editingChannelId,
              savingChannel,
              onCreateChannel,
              onCancelChannel,
            }}
          />
        </TabsContent>
      </Tabs>
    </div>
  );
}

function PolicyForm({
  policyForm,
  onPolicyForm,
  editingPolicyId,
  savingPolicy,
  onCreatePolicy,
  onCancelPolicy,
  environments,
  selectedEnvironmentId,
}: Pick<
  OperationalWorkspaceProps,
  | "policyForm"
  | "onPolicyForm"
  | "editingPolicyId"
  | "savingPolicy"
  | "onCreatePolicy"
  | "onCancelPolicy"
  | "environments"
  | "selectedEnvironmentId"
>) {
  return (
    <Card className="mt-4">
      <SectionHeader
        title={editingPolicyId ? "Edit alert policy" : "Create alert policy"}
        action={
          editingPolicyId ? (
            <Button variant="ghost" size="sm" onClick={onCancelPolicy}>
              Cancel
            </Button>
          ) : null
        }
      />
      <form
        className="alauda-settings-form sm:grid-cols-2"
        onSubmit={onCreatePolicy}
      >
        <FormField label="Environment">
          <Select
            value={policyForm.environmentId || selectedEnvironmentId}
            onChange={(event) =>
              onPolicyForm({ ...policyForm, environmentId: event.target.value })
            }
          >
            <option value="">Select environment</option>
            {environments.map((environment) => (
              <option key={environment.id} value={environment.id}>
                {environment.name}
              </option>
            ))}
          </Select>
        </FormField>
        <FormField label="Environment scope reference">
          <Input
            value={policyForm.deploymentId}
            onChange={(event) =>
              onPolicyForm({ ...policyForm, deploymentId: event.target.value })
            }
          />
        </FormField>
        <FormField label="Notify on" hint="Comma-separated transitions">
          <Input
            required
            value={policyForm.notifyOn}
            onChange={(event) =>
              onPolicyForm({ ...policyForm, notifyOn: event.target.value })
            }
          />
        </FormField>
        <FormField label="Channel IDs" hint="Comma-separated channel IDs">
          <Input
            required
            value={policyForm.channelIds}
            onChange={(event) =>
              onPolicyForm({ ...policyForm, channelIds: event.target.value })
            }
          />
        </FormField>
        <FormField label="Cooldown minutes">
          <Input
            min="0"
            type="number"
            value={policyForm.cooldownMinutes}
            onChange={(event) =>
              onPolicyForm({
                ...policyForm,
                cooldownMinutes: Number(event.target.value),
              })
            }
          />
        </FormField>
        <label className="flex items-center gap-2 text-sm">
          <Checkbox
            checked={policyForm.sendRecoveryNotification}
            onCheckedChange={(checked) =>
              onPolicyForm({
                ...policyForm,
                sendRecoveryNotification: checked === true,
              })
            }
          />
          Send recovery notification
        </label>
        <div className="flex justify-end gap-2 sm:col-span-2">
          <Button type="submit" loading={savingPolicy}>
            {savingPolicy
              ? "Saving"
              : editingPolicyId
                ? "Save policy"
                : "Create policy"}
          </Button>
        </div>
      </form>
    </Card>
  );
}

function ChannelForm({
  channelForm,
  onChannelForm,
  editingChannelId,
  savingChannel,
  onCreateChannel,
  onCancelChannel,
}: Pick<
  OperationalWorkspaceProps,
  | "channelForm"
  | "onChannelForm"
  | "editingChannelId"
  | "savingChannel"
  | "onCreateChannel"
  | "onCancelChannel"
>) {
  return (
    <Card className="mt-4">
      <SectionHeader
        title={
          editingChannelId
            ? "Edit notification channel"
            : "Create notification channel"
        }
        action={
          editingChannelId ? (
            <Button variant="ghost" size="sm" onClick={onCancelChannel}>
              Cancel
            </Button>
          ) : null
        }
      />
      <form
        className="alauda-settings-form sm:grid-cols-2"
        onSubmit={onCreateChannel}
      >
        <FormField label="Type">
          <Select
            value={channelForm.type}
            onChange={(event) =>
              onChannelForm({ ...channelForm, type: event.target.value })
            }
          >
            <option value="webhook">Webhook</option>
            <option value="email">Email</option>
          </Select>
        </FormField>
        <FormField label="Name">
          <Input
            required
            value={channelForm.name}
            onChange={(event) =>
              onChannelForm({ ...channelForm, name: event.target.value })
            }
          />
        </FormField>
        <FormField label="Description">
          <Textarea
            value={channelForm.description}
            onChange={(event) =>
              onChannelForm({ ...channelForm, description: event.target.value })
            }
          />
        </FormField>
        {channelForm.type === "webhook" ? (
          <FormField label="Webhook URL">
            <Input
              required
              value={channelForm.url}
              onChange={(event) =>
                onChannelForm({ ...channelForm, url: event.target.value })
              }
            />
          </FormField>
        ) : (
          <>
            <FormField label="SMTP host">
              <Input
                required
                value={channelForm.smtpHost}
                onChange={(event) =>
                  onChannelForm({
                    ...channelForm,
                    smtpHost: event.target.value,
                  })
                }
              />
            </FormField>
            <FormField label="SMTP port">
              <Input
                required
                value={channelForm.smtpPort}
                onChange={(event) =>
                  onChannelForm({
                    ...channelForm,
                    smtpPort: event.target.value,
                  })
                }
              />
            </FormField>
            <FormField label="From">
              <Input
                required
                value={channelForm.from}
                onChange={(event) =>
                  onChannelForm({ ...channelForm, from: event.target.value })
                }
              />
            </FormField>
            <FormField label="To">
              <Input
                required
                value={channelForm.to}
                onChange={(event) =>
                  onChannelForm({ ...channelForm, to: event.target.value })
                }
              />
            </FormField>
          </>
        )}
        <div className="flex justify-end gap-2 sm:col-span-2">
          <Button type="submit" loading={savingChannel}>
            {savingChannel
              ? "Saving"
              : editingChannelId
                ? "Save channel"
                : "Create channel"}
          </Button>
        </div>
      </form>
    </Card>
  );
}

function Events({
  filteredEvents,
  eventTypeFilter,
  eventResourceFilter,
  eventSearch,
  eventTypes,
  eventResourceTypes,
  onEventTypeFilter,
  onEventResourceFilter,
  onEventSearch,
}: OperationalWorkspaceProps) {
  return (
    <div className="grid gap-5">
      <PageHeader
        title="Events"
        description="Chronological registry activity for the selected environment scope."
      />
      <Card>
        <FilterBar>
          <label className={fieldClass}>
            Type
            <Select
              value={eventTypeFilter}
              onChange={(event) => onEventTypeFilter(event.target.value)}
            >
              <option value="">All event types</option>
              {eventTypes.map((type) => (
                <option key={type} value={type}>
                  {formatEventType(type)}
                </option>
              ))}
            </Select>
          </label>
          <label className={fieldClass}>
            Resource
            <Select
              value={eventResourceFilter}
              onChange={(event) => onEventResourceFilter(event.target.value)}
            >
              <option value="all">All resources</option>
              {eventResourceTypes.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </Select>
          </label>
          <label className={`${fieldClass} min-w-60`}>
            Search
            <SearchInput
              value={eventSearch}
              onChange={(event) => onEventSearch(event.target.value)}
              placeholder="Search events"
            />
          </label>
          <span className={`ml-auto ${muted}`}>
            {filteredEvents.length} shown
          </span>
        </FilterBar>
        {filteredEvents.length === 0 ? (
          <EmptyState
            title="No events match"
            description="No registry events match the current type, resource, and search filters."
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead>
                <TableHead>Event</TableHead>
                <TableHead>Resource</TableHead>
                <TableHead>Actor</TableHead>
                <TableHead>Message</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredEvents.map((event) => (
                <TableRow key={event.id}>
                  <TableCell
                    className={muted}
                    title={formatTimestamp(event.timestamp)}
                  >
                    {formatActivityTimestamp(event.timestamp)}
                  </TableCell>
                  <TableCell>
                    <Badge>
                      {formatEventType(event.type)}
                      <span className="sr-only">{event.type}</span>
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {event.resourceType}:{event.resourceId}
                  </TableCell>
                  <TableCell className={muted}>{event.actor}</TableCell>
                  <TableCell>{event.message}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
