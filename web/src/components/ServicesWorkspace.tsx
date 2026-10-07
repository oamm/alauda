import { FormEvent, useEffect, useState } from "react";
import { ServiceHealthView } from "./ServiceHealthView";
import { Copy, MoreHorizontal, Plus, Trash2, Pencil } from "lucide-react";

import {
  AvailabilitySummary,
  Endpoint,
  Environment,
  EventRecord,
  HealthCheck,
  HealthStateView,
  Incident,
  Service,
  ServiceDeployment,
  ServiceInstance,
} from "../api";
import {
  formatActivityTimestamp,
  formatEventType,
  formatTimestamp,
  pluralize,
} from "../utils/format";
import {
  ActionGroup,
  Button,
  Card,
  Checkbox,
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Alert,
  EmptyState,
  FormField,
  FormSection,
  IconButton,
  Input,
  SearchInput,
  Stat,
  StatGroup,
  Section,
  Workspace,
  ActivityList,
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
  DefinitionList,
  Inline,
  PageHeader,
  Pagination,
  Select,
  SectionHeader,
  Skeleton,
  StatusBadge,
  Switch,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
  Textarea,
} from "./ui";

type RegistrationFormState = {
  environmentId: string;
  instanceName: string;
  address: string;
  description: string;
  endpoints: Array<{
    name: string;
    protocol: string;
    port: number;
    path: string;
    primary: boolean;
  }>;
  configureHealth: boolean;
  healthName: string;
  healthType: string;
  healthPath: string;
  healthIntervalSeconds: number;
  healthTimeoutSeconds: number;
  healthFailuresBeforeUnhealthy: number;
  healthSuccessesBeforeHealthy: number;
};
type ServiceForm = { name: string; displayName: string; description: string };
type InstanceEditForm = {
  address: string;
  description: string;
  enabled: boolean;
};
type EndpointEditForm = {
  name: string;
  protocol: string;
  port: number;
  path: string;
  enabled: boolean;
  primary: boolean;
};
type HealthForm = {
  instanceId: string;
  endpointId: string;
  name: string;
  type: string;
  path: string;
  expectedStatus: string;
  intervalSeconds: number;
  timeoutSeconds: number;
  failuresBeforeUnhealthy: number;
  successesBeforeHealthy: number;
  description: string;
};

type ServicesWorkspaceProps = {
  loading: boolean;
  error: string;
  environments: Environment[];
  services: Service[];
  filteredServices: Service[];
  visibleServices: Service[];
  deployments: ServiceDeployment[];
  instances: ServiceInstance[];
  endpoints: Endpoint[];
  healthChecks: HealthCheck[];
  healthStates: HealthStateView[];
  incidents: Incident[];
  events: EventRecord[];
  availability: {
    availability24h?: AvailabilitySummary;
    availability7d?: AvailabilitySummary;
    availability30d?: AvailabilitySummary;
  };
  selectedEnvironmentId: string;
  currentEnvironmentName: string;
  selectedServiceId: string;
  onSelectService: (id: string) => void;
  serviceSearch: string;
  setServiceSearch: (value: string) => void;
  serviceHealthFilter: string;
  setServiceHealthFilter: (value: string) => void;
  serviceTagFilter: string;
  setServiceTagFilter: (value: string) => void;
  servicePage: number;
  servicePageCount: number;
  setServicePage: (value: number) => void;
  selectedBulkServiceIds: string[];
  toggleBulkService: (id: string) => void;
  selectVisibleServices: () => void;
  clearBulkSelection: () => void;
  handleCopySelectedServiceIds: () => void;
  handleDeleteSelectedServices: () => void;
  bulkActionRunning: boolean;
  serviceTab: string;
  setServiceTab: (value: string) => void;
  serviceForm: ServiceForm;
  setServiceForm: React.Dispatch<React.SetStateAction<ServiceForm>>;
  showCreateService: boolean;
  setShowCreateService: (value: boolean) => void;
  handleCreateService: (event: FormEvent<HTMLFormElement>) => void;
  editingServiceId: string;
  setEditingServiceId: (value: string) => void;
  serviceEditForm: ServiceForm;
  setServiceEditForm: React.Dispatch<React.SetStateAction<ServiceForm>>;
  handleUpdateService: (event: FormEvent<HTMLFormElement>) => void;
  handleDeleteService: (service: Service) => void;
  savingService: boolean;
  showAddRuntime: boolean;
  setShowAddRuntime: (value: boolean) => void;
  registrationForm: RegistrationFormState;
  setRegistrationForm: React.Dispatch<
    React.SetStateAction<RegistrationFormState>
  >;
  registrationSuccess: string;
  savingRegistration: boolean;
  addRegistrationEndpoint: () => void;
  removeRegistrationEndpoint: (index: number) => void;
  setPrimaryRegistrationEndpoint: (index: number) => void;
  updateRegistrationEndpoint: (
    index: number,
    updates: Partial<RegistrationFormState["endpoints"][number]>,
  ) => void;
  handleRegisterInstance: (event: FormEvent<HTMLFormElement>) => void;
  closeRuntimeDialog: () => void;
  registeredRuntime?: {
    name: string;
    address: string;
    endpoints: Array<{
      id: string;
      protocol: string;
      port: number;
      path: string;
      primary: boolean;
    }>;
  };
  healthFollowUpError: string;
  healthFollowUpConfigured: boolean;
  savingHealthFollowUp: boolean;
  handleCreateRegisteredHealth: () => void;
  editingInstanceId: string;
  setEditingInstanceId: (value: string) => void;
  instanceEditForm: InstanceEditForm;
  setInstanceEditForm: React.Dispatch<React.SetStateAction<InstanceEditForm>>;
  handleUpdateInstance: (event: FormEvent<HTMLFormElement>) => void;
  handleDeleteInstance: (instance: ServiceInstance) => void;
  savingRuntimeEdit: boolean;
  editingEndpointId: string;
  addingEndpointInstanceId: string;
  startAddEndpoint: (instance: ServiceInstance) => void;
  setEditingEndpointId: (value: string) => void;
  setAddingEndpointInstanceId: (value: string) => void;
  endpointEditForm: EndpointEditForm;
  setEndpointEditForm: React.Dispatch<React.SetStateAction<EndpointEditForm>>;
  handleUpdateEndpoint: (event: FormEvent<HTMLFormElement>) => void;
  handleDeleteEndpoint: (endpoint: Endpoint) => void;
  editingHealthCheckId: string;
  setEditingHealthCheckId: (value: string) => void;
  healthForm: HealthForm;
  setHealthForm: React.Dispatch<React.SetStateAction<HealthForm>>;
  handleUpdateHealthCheck: (event: FormEvent<HTMLFormElement>) => void;
  handleDeleteHealthCheck: (check: HealthCheck) => void;
  savingHealthCheck: boolean;
  handleRunHealthCheck: (id: string) => Promise<void>;
  startEditHealthCheck: (check: HealthCheck) => void;
  healthEditForm: Pick<
    HealthCheck,
    | "enabled"
    | "intervalSeconds"
    | "timeoutSeconds"
    | "failuresBeforeUnhealthy"
    | "successesBeforeHealthy"
  > & { description: string };
  setHealthEditForm: React.Dispatch<
    React.SetStateAction<ServicesWorkspaceProps["healthEditForm"]>
  >;
  testResult: string;
  setError: (value: string) => void;
};

const protocolOptions = [
  ["PROTOCOL_HTTP", "HTTP"],
  ["PROTOCOL_HTTPS", "HTTPS"],
  ["PROTOCOL_GRPC", "gRPC"],
  ["PROTOCOL_TCP", "TCP"],
  ["PROTOCOL_UDP", "UDP"],
];

export function ServicesWorkspace(props: ServicesWorkspaceProps) {
  const [activeTab, setActiveTab] = useState(props.serviceTab);
  useEffect(() => setActiveTab(props.serviceTab), [props.serviceTab]);
  const selectedService = props.services.find(
    (service) => service.id === props.selectedServiceId,
  );
  const selectedDeployments = props.deployments.filter(
    (deployment) => deployment.serviceId === props.selectedServiceId,
  );
  const deploymentIds = new Set(
    selectedDeployments.map((deployment) => deployment.id),
  );
  const selectedInstances = props.instances.filter((instance) =>
    deploymentIds.has(instance.deploymentId),
  );
  const instanceIds = new Set(selectedInstances.map((instance) => instance.id));
  const selectedEndpoints = props.endpoints.filter((endpoint) =>
    instanceIds.has(endpoint.instanceId),
  );
  const selectedHealthChecks = props.healthChecks.filter((check) =>
    instanceIds.has(check.instanceId),
  );
  const selectedEvents = props.events.filter(
    (event) => event.serviceId === props.selectedServiceId,
  );
  const status = selectedService
    ? serviceStatus(selectedService, props.incidents)
    : "unknown";

  return (
    <section className="services-workspace-new">
      <PageHeader
        title="Services"
        description={`${pluralize(props.filteredServices.length, "service")} ${props.selectedEnvironmentId ? `in ${props.currentEnvironmentName}` : "across all environments"}.`}
        action={
          <Button
            onClick={() => props.setShowCreateService(true)}
            type="button"
            variant="primary"
          >
            <Plus size={16} />
            Create service
          </Button>
        }
      />
      <div className="services-layout-new">
        <Card className="service-catalog-new">
          <div className="service-filters-new">
            <FormField label="Search">
              <SearchInput
                aria-label="Search services"
                placeholder="Search services"
                value={props.serviceSearch}
                onChange={(event) => props.setServiceSearch(event.target.value)}
              />
            </FormField>
            <div className="service-filter-pair">
              <FormField label="Health">
                <Select
                  value={props.serviceHealthFilter}
                  onChange={(event) =>
                    props.setServiceHealthFilter(event.target.value)
                  }
                >
                  <option value="all">All health</option>
                  <option value="healthy">Healthy</option>
                  <option value="degraded">Degraded</option>
                </Select>
              </FormField>
              <FormField label="Tag">
                <Input
                  aria-label="Filter by tag"
                  placeholder="Any tag"
                  value={props.serviceTagFilter}
                  onChange={(event) =>
                    props.setServiceTagFilter(event.target.value)
                  }
                />
              </FormField>
            </div>
            {props.serviceSearch ||
            props.serviceHealthFilter !== "all" ||
            props.serviceTagFilter ? (
              <Button
                onClick={() => {
                  props.setServiceSearch("");
                  props.setServiceHealthFilter("all");
                  props.setServiceTagFilter("");
                }}
                size="sm"
                type="button"
                variant="link"
              >
                Clear filters
              </Button>
            ) : null}
          </div>
          {props.selectedBulkServiceIds.length ? (
            <div className="service-bulk-new">
              <span>{props.selectedBulkServiceIds.length} selected</span>
              <Button
                onClick={props.selectVisibleServices}
                size="sm"
                type="button"
              >
                Select visible
              </Button>
              <Button
                onClick={props.handleCopySelectedServiceIds}
                size="sm"
                type="button"
                variant="ghost"
              >
                <Copy size={14} />
                Copy IDs
              </Button>
              <Button
                disabled={props.bulkActionRunning}
                onClick={props.handleDeleteSelectedServices}
                size="sm"
                type="button"
                variant="danger"
              >
                {props.bulkActionRunning ? "Deleting" : "Delete"}
              </Button>
              <Button
                onClick={props.clearBulkSelection}
                size="sm"
                type="button"
                variant="ghost"
              >
                Clear
              </Button>
            </div>
          ) : null}
          {props.loading ? (
            <div className="service-skeleton-list">
              <Skeleton className="h-14" />
              <Skeleton className="h-14" />
              <Skeleton className="h-14" />
              <Skeleton className="h-14" />
            </div>
          ) : props.filteredServices.length === 0 ? (
            <EmptyState
              title="No matching services"
              description="No catalog services match the current search, health, and tag filters."
            />
          ) : (
            <div className="service-resource-list">
              {props.visibleServices.map((service) => (
                <ServiceListItem
                  key={service.id}
                  service={service}
                  selected={service.id === props.selectedServiceId}
                  selectedBulk={props.selectedBulkServiceIds.includes(
                    service.id,
                  )}
                  instanceCount={
                    props.instances.filter((instance) =>
                      new Set(
                        props.deployments
                          .filter(
                            (deployment) => deployment.serviceId === service.id,
                          )
                          .map((deployment) => deployment.id),
                      ).has(instance.deploymentId),
                    ).length
                  }
                  status={serviceStatus(service, props.incidents)}
                  onSelect={() => props.onSelectService(service.id)}
                  onToggle={() => props.toggleBulkService(service.id)}
                />
              ))}
            </div>
          )}
          <Pagination
            page={props.servicePage}
            pageCount={props.servicePageCount}
            onPageChange={props.setServicePage}
          />
        </Card>
        <Workspace className="service-detail-new" aria-label="Service details">
          {selectedService ? (
            <>
              <ServiceDetailHeader
                service={selectedService}
                status={status}
                instanceCount={selectedInstances.length}
                endpointCount={selectedEndpoints.length}
                environmentCount={selectedDeployments.length}
                onEdit={() => {
                  props.setEditingServiceId(selectedService.id);
                  props.setServiceEditForm({
                    name: selectedService.name,
                    displayName:
                      selectedService.displayName || selectedService.name,
                    description: selectedService.description ?? "",
                  });
                }}
                onDelete={() => props.handleDeleteService(selectedService)}
                onCopy={() =>
                  navigator.clipboard
                    .writeText(selectedService.id)
                    .then(() => props.setError("Service ID copied."))
                    .catch(() => props.setError("Failed to copy service ID."))
                }
              />
              <Tabs
                value={activeTab}
                onValueChange={(value) => {
                  setActiveTab(value);
                  props.setServiceTab(value);
                }}
              >
                <TabsList aria-label="Service sections">
                  <TabsTrigger value="overview">Overview</TabsTrigger>
                  <TabsTrigger value="instances">Instances</TabsTrigger>
                  <TabsTrigger value="health">Health</TabsTrigger>
                  <TabsTrigger value="incidents">Incidents</TabsTrigger>
                  <TabsTrigger value="events">Events</TabsTrigger>
                </TabsList>
                <TabsContent value="overview">
                  <ServiceOverview
                    environmentCount={selectedDeployments.length}
                    service={selectedService}
                    status={status}
                    instances={selectedInstances}
                    endpoints={selectedEndpoints}
                    healthChecks={selectedHealthChecks}
                    availability={props.availability}
                    events={selectedEvents}
                  />
                </TabsContent>
                <TabsContent value="instances">
                  <ServiceInstances
                    {...props}
                    service={selectedService}
                    deployments={selectedDeployments}
                    instances={selectedInstances}
                    endpoints={selectedEndpoints}
                    healthChecks={selectedHealthChecks}
                    environments={props.environments}
                  />
                </TabsContent>
                <TabsContent value="health">
                  <ServiceHealthView
                    key={selectedService.id}
                    checks={selectedHealthChecks}
                    states={props.healthStates}
                    instances={selectedInstances}
                    endpoints={selectedEndpoints}
                    availability={props.availability}
                    error={props.error}
                    saving={props.savingHealthCheck}
                    onInstances={() => props.setServiceTab("instances")}
                    onEdit={props.startEditHealthCheck}
                    onDelete={props.handleDeleteHealthCheck}
                    onRun={props.handleRunHealthCheck}
                    onConfigure={(instance) => {
                      props.setEditingHealthCheckId("new");
                      props.setHealthForm((current) => ({
                        ...current,
                        instanceId: instance.id,
                        endpointId: "",
                      }));
                    }}
                  />
                </TabsContent>
                <TabsContent value="incidents">
                  <ServiceIncidents
                    incidents={props.incidents.filter(
                      (incident) => incident.serviceId === selectedService.id,
                    )}
                  />
                </TabsContent>
                <TabsContent value="events">
                  <ServiceEvents events={selectedEvents} />
                </TabsContent>
              </Tabs>
            </>
          ) : (
            <EmptyState
              title="Select a service"
              description="Choose a service from the catalog to inspect its configuration and operational state."
            />
          )}
        </Workspace>
      </div>
      <ServiceFormDialog
        open={props.showCreateService}
        title="Create service"
        form={props.serviceForm}
        setForm={props.setServiceForm}
        saving={props.savingService}
        onClose={() => props.setShowCreateService(false)}
        onSubmit={props.handleCreateService}
      />
      {props.editingServiceId ? (
        <ServiceFormDialog
          open
          title="Edit service"
          form={props.serviceEditForm}
          setForm={props.setServiceEditForm}
          saving={props.savingService}
          onClose={() => props.setEditingServiceId("")}
          onSubmit={props.handleUpdateService}
        />
      ) : null}
      {selectedService && props.showAddRuntime && !props.registeredRuntime ? (
        <AddInstanceDialog {...props} service={selectedService} />
      ) : null}
      {selectedService && props.showAddRuntime && props.registeredRuntime ? (
        <RegisteredInstanceSuccess {...props} service={selectedService} />
      ) : null}
      {props.editingHealthCheckId ? (
        <HealthDialog
          {...props}
          instances={selectedInstances}
          endpoints={selectedEndpoints}
        />
      ) : null}
    </section>
  );
}

function ServiceListItem({
  service,
  selected,
  selectedBulk,
  instanceCount,
  status,
  onSelect,
  onToggle,
}: {
  service: Service;
  selected: boolean;
  selectedBulk: boolean;
  instanceCount: number;
  status: string;
  onSelect: () => void;
  onToggle: () => void;
}) {
  return (
    <div
      className={
        selected ? "service-resource-item selected" : "service-resource-item"
      }
    >
      <Checkbox
        aria-label={`Select ${service.displayName || service.name}`}
        checked={selectedBulk}
        onCheckedChange={onToggle}
      />
      <button
        className="service-resource-button"
        aria-pressed={selected}
        onClick={onSelect}
        type="button"
      >
        <span className="service-resource-title">
          <strong>{service.displayName || service.name}</strong>
          <StatusBadge status={status} />
        </span>
        <span className="service-resource-meta">
          {service.name} · {pluralize(instanceCount, "instance")}
        </span>
      </button>
    </div>
  );
}

function ServiceDetailHeader({
  service,
  status,
  instanceCount,
  endpointCount,
  environmentCount,
  onEdit,
  onDelete,
  onCopy,
}: {
  service: Service;
  status: string;
  instanceCount: number;
  endpointCount: number;
  environmentCount: number;
  onEdit: () => void;
  onDelete: () => void;
  onCopy: () => void;
}) {
  return (
    <div className="service-detail-header-new">
      <div>
        <Inline>
          <h2>{service.displayName || service.name}</h2>
          <StatusBadge status={status} />
        </Inline>
        <p className="service-detail-meta">
          {service.name} · {environmentCount}{" "}
          {environmentCount === 1 ? "environment" : "environments"} ·{" "}
          {instanceCount} {instanceCount === 1 ? "instance" : "instances"} ·{" "}
          {endpointCount} {endpointCount === 1 ? "endpoint" : "endpoints"}
        </p>
        {service.description ? (
          <p className="service-detail-description">{service.description}</p>
        ) : null}
      </div>
      <details className="service-actions-menu">
        <summary aria-label="Service actions">
          <MoreHorizontal size={18} />
        </summary>
        <div>
          <Button onClick={onEdit} size="sm" type="button" variant="ghost">
            Edit service
          </Button>
          <Button onClick={onCopy} size="sm" type="button" variant="ghost">
            <Copy size={14} />
            Copy ID
          </Button>
          <Button onClick={onDelete} size="sm" type="button" variant="danger">
            Delete service
          </Button>
        </div>
      </details>
    </div>
  );
}

function ServiceOverview({
  environmentCount,
  service,
  status,
  instances,
  endpoints,
  healthChecks,
  availability,
  events,
}: {
  environmentCount: number;
  service: Service;
  status: string;
  instances: ServiceInstance[];
  endpoints: Endpoint[];
  healthChecks: HealthCheck[];
  availability: ServicesWorkspaceProps["availability"];
  events: EventRecord[];
}) {
  const windows: Array<[string, AvailabilitySummary | undefined]> = [
    ["24 hours", availability.availability24h],
    ["7 days", availability.availability7d],
    ["30 days", availability.availability30d],
  ];
  return (
    <div className="service-section-new">
      <StatGroup label="Service summary">
        <Stat label="Environments" value={environmentCount} />
        <Stat label="Instances" value={instances.length} />
        <Stat label="Endpoints" value={endpoints.length} />
        <Stat
          label="Status"
          value={status === "healthy" ? "Healthy" : "Degraded"}
        />
      </StatGroup>
      <div className="service-summary-grid-new">
        <Section title="Availability">
          <div className="availability-summary-new">
            {windows.map(([label, summary]) => (
              <div key={label}>
                <span>{label}</span>
                <strong>
                  {typeof summary?.availabilityPercent === "number"
                    ? `${summary.availabilityPercent.toFixed(2)}%`
                    : "No data"}
                </strong>
              </div>
            ))}
          </div>
        </Section>
        <Section title="Monitoring">
          {healthChecks.length ? (
            <div className="compact-list-new">
              {healthChecks.slice(0, 4).map((check) => (
                <div key={check.id}>
                  <strong>{check.name}</strong>
                  <StatusBadge
                    status={check.enabled ? "enabled" : "disabled"}
                  />
                </div>
              ))}
            </div>
          ) : (
            <EmptyState
              title="No health monitoring"
              description={`${service.displayName || service.name} has no health checks configured.`}
            />
          )}
        </Section>
      </div>
      <Section title="Recent activity">
        {events.length ? (
          <ActivityList
            items={events.slice(0, 5).map((event) => ({
              id: event.id,
              time: formatActivityTimestamp(event.timestamp),
              event: formatEventType(event.type),
              context: event.message,
            }))}
          />
        ) : (
          <EmptyState
            title="No recent activity"
            description="No events have been recorded for this service in the selected scope."
          />
        )}
      </Section>
    </div>
  );
}

type InstanceWorkspaceProps = ServicesWorkspaceProps & {
  service: Service;
  deployments: ServiceDeployment[];
  instances: ServiceInstance[];
  endpoints: Endpoint[];
  healthChecks: HealthCheck[];
  environments: Environment[];
};

function ServiceInstances(props: InstanceWorkspaceProps) {
  const [selectedId, setSelectedId] = useState("");
  const selected = props.instances.find(
    (instance) => instance.id === selectedId,
  );
  function closeInstance() {
    props.setEditingInstanceId("");
    props.setEditingEndpointId("");
    props.setAddingEndpointInstanceId("");
    setSelectedId("");
  }
  function openInstance(instance: ServiceInstance) {
    closeInstance();
    props.setError("");
    setSelectedId(instance.id);
  }
  return (
    <div className="service-section-new">
      <SectionHeader
        title="Instances"
        action={
          props.instances.length ? (
            <Button
              onClick={() => props.setShowAddRuntime(true)}
              variant="primary"
            >
              <Plus size={16} />
              Add instance
            </Button>
          ) : undefined
        }
      />
      {props.instances.length === 0 ? (
        <EmptyState
          title="No instances yet"
          description="Add an instance to register service addresses and endpoints."
          action={
            <Button
              onClick={() => props.setShowAddRuntime(true)}
              variant="primary"
            >
              <Plus size={14} />
              Add instance
            </Button>
          }
        />
      ) : (
        <div className="grid gap-4">
          {props.deployments.map((deployment) => {
            const instances = props.instances.filter(
              (instance) => instance.deploymentId === deployment.id,
            );
            return (
              <Section
                key={deployment.id}
                title={environmentName(
                  props.environments,
                  deployment.environmentId,
                )}
                description={pluralize(instances.length, "instance")}
                divider={false}
              >
                <Table
                  aria-label={`Instances in ${environmentName(props.environments, deployment.environmentId)}`}
                >
                  <TableHeader>
                    <TableRow>
                      <TableHead>Instance</TableHead>
                      <TableHead>Address</TableHead>
                      <TableHead>Endpoints</TableHead>
                      <TableHead>Monitoring</TableHead>
                      <TableHead>Status</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {instances.map((instance) => {
                      const checks = props.healthChecks.filter(
                        (check) => check.instanceId === instance.id,
                      );
                      const state = props.healthStates.find(
                        (item) => item.instanceId === instance.id,
                      );
                      return (
                        <TableRow key={instance.id}>
                          <TableCell>
                            <Button
                              variant="link"
                              onClick={() => openInstance(instance)}
                              aria-label={`View details for ${instance.name}`}
                            >
                              {instance.name}
                            </Button>
                          </TableCell>
                          <TableCell>
                            {instance.address}
                            {instance.port ? `:${instance.port}` : ""}
                          </TableCell>
                          <TableCell>
                            {
                              props.endpoints.filter(
                                (endpoint) =>
                                  endpoint.instanceId === instance.id,
                              ).length
                            }
                          </TableCell>
                          <TableCell>
                            <span className="text-xs text-[var(--text-muted)]">
                              {checks.length
                                ? pluralize(checks.length, "health check")
                                : "Not configured"}
                            </span>
                          </TableCell>
                          <TableCell>
                            <StatusBadge
                              status={
                                !instance.enabled
                                  ? "disabled"
                                  : state?.currentState
                                    ? formatHealthState(state.currentState)
                                    : "enabled"
                              }
                            />
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </Section>
            );
          })}
        </div>
      )}
      {selected ? (
        <InstanceDetails
          key={selected.id}
          {...props}
          instance={selected}
          onClose={closeInstance}
        />
      ) : null}
    </div>
  );
}

function InstanceDetails(
  props: InstanceWorkspaceProps & {
    instance: ServiceInstance;
    onClose: () => void;
  },
) {
  const [tab, setTab] = useState("endpoints");
  const instance = props.instance;
  const endpoints = props.endpoints.filter(
    (endpoint) => endpoint.instanceId === instance.id,
  );
  const checks = props.healthChecks.filter(
    (check) => check.instanceId === instance.id,
  );
  const deployment = props.deployments.find(
    (item) => item.id === instance.deploymentId,
  );
  const environment = deployment
    ? environmentName(props.environments, deployment.environmentId)
    : "Unknown environment";
  const editorOpen =
    props.addingEndpointInstanceId === instance.id ||
    endpoints.some((endpoint) => endpoint.id === props.editingEndpointId);
  function closeEditor() {
    props.setEditingEndpointId("");
    props.setAddingEndpointInstanceId("");
    props.setError("");
  }
  function editInstance() {
    props.setError("");
    props.setEditingInstanceId(instance.id);
    props.setInstanceEditForm({
      address: instance.address,
      description: instance.description ?? "",
      enabled: instance.enabled,
    });
    setTab("details");
  }
  function configureHealth() {
    props.onClose();
    props.setServiceTab("health");
    props.setHealthForm((current) => ({ ...current, instanceId: instance.id }));
    props.setEditingHealthCheckId("new");
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) {
          if (editorOpen) closeEditor();
          else props.onClose();
        }
      }}
    >
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>
            {editorOpen
              ? props.addingEndpointInstanceId
                ? "Add endpoint"
                : "Edit endpoint"
              : instance.name}
          </DialogTitle>
          <DialogDescription>
            {editorOpen ? instance.name : instance.address} · {environment}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          {props.error ? <Alert tone="danger" title={props.error} /> : null}
          {editorOpen ? (
            <EndpointEditor {...props} onClose={closeEditor} />
          ) : (
            <>
              <Inline>
                <StatusBadge
                  status={instance.enabled ? "enabled" : "disabled"}
                />
                <Button size="sm" variant="ghost" onClick={editInstance}>
                  Edit instance
                </Button>
              </Inline>
              <Tabs value={tab} onValueChange={setTab}>
                <TabsList aria-label="Instance sections">
                  <TabsTrigger value="endpoints">Endpoints</TabsTrigger>
                  <TabsTrigger value="monitoring">Monitoring</TabsTrigger>
                  <TabsTrigger value="details">Details</TabsTrigger>
                </TabsList>
                <TabsContent value="endpoints">
                  <Section
                    title="Endpoints"
                    divider={false}
                    action={
                      endpoints.length ? (
                        <Button
                          size="sm"
                          onClick={() => {
                            props.setError("");
                            props.startAddEndpoint(instance);
                          }}
                        >
                          <Plus size={14} />
                          Add endpoint
                        </Button>
                      ) : undefined
                    }
                  >
                    {endpoints.length ? (
                      <Table aria-label="Instance endpoints">
                        <TableHeader>
                          <TableRow>
                            <TableHead>Protocol</TableHead>
                            <TableHead>Port</TableHead>
                            <TableHead>Path</TableHead>
                            <TableHead>Primary</TableHead>
                            <TableHead>Status</TableHead>
                            <TableHead>Actions</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {endpoints.map((endpoint) => (
                            <TableRow key={endpoint.id}>
                              <TableCell>
                                {formatProtocol(endpoint.protocol)}
                              </TableCell>
                              <TableCell>:{endpoint.port}</TableCell>
                              <TableCell>{endpoint.path || "-"}</TableCell>
                              <TableCell>
                                {endpoint.primary ? "Primary" : "-"}
                              </TableCell>
                              <TableCell>
                                <StatusBadge
                                  status={
                                    endpoint.enabled ? "enabled" : "disabled"
                                  }
                                />
                              </TableCell>
                              <TableCell>
                                <Inline>
                                  <IconButton
                                    label={`Edit endpoint ${endpoint.name}`}
                                    variant="ghost"
                                    onClick={() => {
                                      props.setError("");
                                      props.setAddingEndpointInstanceId("");
                                      props.setEditingEndpointId(endpoint.id);
                                      props.setEndpointEditForm({
                                        name: endpoint.name,
                                        protocol: endpoint.protocol,
                                        port: endpoint.port,
                                        path: endpoint.path,
                                        primary: endpoint.primary,
                                        enabled: endpoint.enabled,
                                      });
                                    }}
                                  >
                                    <Pencil size={14} />
                                  </IconButton>
                                  <IconButton
                                    label={`Remove endpoint ${endpoint.name}`}
                                    disabled={props.savingRuntimeEdit}
                                    variant="ghost"
                                    onClick={() =>
                                      props.handleDeleteEndpoint(endpoint)
                                    }
                                  >
                                    <Trash2 size={14} />
                                  </IconButton>
                                </Inline>
                              </TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    ) : (
                      <EmptyState
                        title="No endpoints yet"
                        description="Add an endpoint for this instance."
                        action={
                          <Button
                            size="sm"
                            onClick={() => {
                              props.setError("");
                              props.startAddEndpoint(instance);
                            }}
                          >
                            <Plus size={14} />
                            Add endpoint
                          </Button>
                        }
                      />
                    )}
                  </Section>
                </TabsContent>
                <TabsContent value="monitoring">
                  <Section title="Monitoring" divider={false}>
                    {checks.length ? (
                      <>
                        <p className="text-sm">
                          {pluralize(checks.length, "health check")} configured
                          · {checks.filter((check) => check.enabled).length}{" "}
                          enabled
                        </p>
                        <Button
                          size="sm"
                          variant="link"
                          onClick={() => {
                            props.onClose();
                            props.setServiceTab("health");
                          }}
                        >
                          View service health
                        </Button>
                      </>
                    ) : (
                      <EmptyState
                        title="No monitoring configured"
                        description="Configure a health check for this instance."
                        action={
                          <Button size="sm" onClick={configureHealth}>
                            Configure health
                          </Button>
                        }
                      />
                    )}
                  </Section>
                </TabsContent>
                <TabsContent value="details">
                  <Section title="Details" divider={false}>
                    {props.editingInstanceId === instance.id ? (
                      <form
                        className="grid gap-3"
                        onSubmit={props.handleUpdateInstance}
                      >
                        <FormField label="Address">
                          <Input
                            required
                            value={props.instanceEditForm.address}
                            autoFocus
                            onChange={(event) =>
                              props.setInstanceEditForm((current) => ({
                                ...current,
                                address: event.target.value,
                              }))
                            }
                          />
                        </FormField>
                        <FormField label="Description">
                          <Textarea
                            value={props.instanceEditForm.description}
                            onChange={(event) =>
                              props.setInstanceEditForm((current) => ({
                                ...current,
                                description: event.target.value,
                              }))
                            }
                          />
                        </FormField>
                        <FormField label="Enabled">
                          <Switch
                            checked={props.instanceEditForm.enabled}
                            onCheckedChange={(enabled) =>
                              props.setInstanceEditForm((current) => ({
                                ...current,
                                enabled,
                              }))
                            }
                          />
                        </FormField>
                        <ActionGroup>
                          <Button
                            variant="ghost"
                            onClick={() => props.setEditingInstanceId("")}
                          >
                            Cancel
                          </Button>
                          <Button
                            type="submit"
                            variant="primary"
                            disabled={props.savingRuntimeEdit}
                          >
                            Save instance
                          </Button>
                        </ActionGroup>
                      </form>
                    ) : (
                      <DefinitionList>
                        <dt>Address</dt>
                        <dd className="break-all">{instance.address}</dd>
                        <dt>Description</dt>
                        <dd>{instance.description || "No description"}</dd>
                        <dt>Environment</dt>
                        <dd>{environment}</dd>
                      </DefinitionList>
                    )}
                    <div className="mt-4 border-t border-[var(--border)] pt-3">
                      <Button
                        size="sm"
                        variant="danger"
                        disabled={props.savingRuntimeEdit}
                        onClick={() => props.handleDeleteInstance(instance)}
                      >
                        Remove instance
                      </Button>
                    </div>
                  </Section>
                </TabsContent>
              </Tabs>
            </>
          )}
        </DialogBody>
        {!editorOpen ? (
          <DialogFooter>
            <Button onClick={props.onClose}>Done</Button>
          </DialogFooter>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function EndpointEditor(
  props: ServicesWorkspaceProps & {
    endpoint?: Endpoint;
    onClose: () => void;
    service: Service;
    deployments: ServiceDeployment[];
    instances: ServiceInstance[];
    endpoints: Endpoint[];
    healthChecks: HealthCheck[];
    environments: Environment[];
  },
) {
  const instanceId =
    props.addingEndpointInstanceId ||
    props.endpoints.find((endpoint) => endpoint.id === props.editingEndpointId)
      ?.instanceId;
  const duplicateName = props.endpoints.some(
    (endpoint) =>
      endpoint.instanceId === instanceId &&
      endpoint.id !== props.editingEndpointId &&
      endpoint.name === props.endpointEditForm.name.trim(),
  );
  return (
    <form
      className="grid gap-3 sm:grid-cols-2"
      onSubmit={props.handleUpdateEndpoint}
    >
      <FormField
        label="Name"
        hint="Unique within this instance."
        error={
          duplicateName
            ? "An endpoint with this name already exists on this instance."
            : undefined
        }
      >
        <Input
          required
          autoFocus
          value={props.endpointEditForm.name}
          onChange={(event) =>
            props.setEndpointEditForm((current) => ({
              ...current,
              name: event.target.value,
            }))
          }
        />
      </FormField>
      <FormField label="Protocol">
        <Select
          value={props.endpointEditForm.protocol}
          onChange={(event) =>
            props.setEndpointEditForm((current) => ({
              ...current,
              protocol: event.target.value,
            }))
          }
        >
          {protocolOptions.map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </Select>
      </FormField>
      <FormField label="Port">
        <Input
          max={65535}
          min={1}
          required
          type="number"
          value={props.endpointEditForm.port}
          onChange={(event) =>
            props.setEndpointEditForm((current) => ({
              ...current,
              port: Number(event.target.value),
            }))
          }
        />
      </FormField>
      <FormField label="Path">
        <Input
          value={props.endpointEditForm.path}
          onChange={(event) =>
            props.setEndpointEditForm((current) => ({
              ...current,
              path: event.target.value,
            }))
          }
        />
      </FormField>
      <FormField label="Primary">
        <Switch
          checked={props.endpointEditForm.primary}
          onCheckedChange={(checked) =>
            props.setEndpointEditForm((current) => ({
              ...current,
              primary: checked,
            }))
          }
        />
      </FormField>
      <FormField label="Enabled">
        <Switch
          checked={props.endpointEditForm.enabled}
          onCheckedChange={(checked) =>
            props.setEndpointEditForm((current) => ({
              ...current,
              enabled: checked,
            }))
          }
        />
      </FormField>
      <Inline>
        <Button
          disabled={props.savingRuntimeEdit}
          type="submit"
          variant="primary"
        >
          Save
        </Button>
        <Button onClick={props.onClose} type="button" variant="ghost">
          Cancel
        </Button>
      </Inline>
    </form>
  );
}

function ServiceIncidents({ incidents }: { incidents: Incident[] }) {
  return (
    <div className="service-section-new">
      <SectionHeader
        title="Incidents"
        description="Operational incidents associated with this service."
      />
      {incidents.length ? (
        <div className="compact-list-new">
          {incidents.map((incident) => (
            <div key={incident.id}>
              <strong>{incident.reason}</strong>
              <StatusBadge
                status={incident.state
                  .replace("INCIDENT_STATE_", "")
                  .toLowerCase()}
              />
            </div>
          ))}
        </div>
      ) : (
        <EmptyState
          title="No incidents"
          description="No incidents have been recorded for this service in the selected scope."
        />
      )}
    </div>
  );
}
function ServiceEvents({ events }: { events: EventRecord[] }) {
  return (
    <div className="service-section-new">
      <SectionHeader
        title="Events"
        description="Historical activity for this service."
      />
      {events.length ? (
        <div className="compact-list-new">
          {events.map((event) => (
            <div key={event.id}>
              <span>{formatTimestamp(event.timestamp)}</span>
              <strong>{formatEventType(event.type)}</strong>
              <small>{event.message}</small>
            </div>
          ))}
        </div>
      ) : (
        <EmptyState
          title="No events"
          description="No service events have been recorded in the selected scope."
        />
      )}
    </div>
  );
}

function ServiceFormDialog({
  open,
  title,
  form,
  setForm,
  saving,
  onClose,
  onSubmit,
}: {
  open: boolean;
  title: string;
  form: ServiceForm;
  setForm: React.Dispatch<React.SetStateAction<ServiceForm>>;
  saving: boolean;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <Dialog open={open} onOpenChange={(value) => !value && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            Keep catalog identity and description clear for operators.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit}>
          <DialogBody>
            <div className="form-stack-new">
              <FormField label="Name">
                <Input
                  required
                  value={form.name}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                />
              </FormField>
              <FormField label="Display name">
                <Input
                  required
                  value={form.displayName}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      displayName: event.target.value,
                    }))
                  }
                />
              </FormField>
              <FormField label="Description">
                <Textarea
                  value={form.description}
                  onChange={(event) =>
                    setForm((current) => ({
                      ...current,
                      description: event.target.value,
                    }))
                  }
                />
              </FormField>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button onClick={onClose} type="button" variant="ghost">
              Cancel
            </Button>
            <Button
              disabled={saving}
              loading={saving}
              type="submit"
              variant="primary"
            >
              {title.startsWith("Edit") ? "Save service" : "Create service"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function AddInstanceDialog(
  props: ServicesWorkspaceProps & { service: Service },
) {
  return (
    <Dialog
      open={props.showAddRuntime}
      onOpenChange={(value) => !value && props.closeRuntimeDialog()}
    >
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>Add instance</DialogTitle>
          <DialogDescription>
            {props.service.displayName || props.service.name} · add an
            environment instance and its endpoints.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={props.handleRegisterInstance}>
          <DialogBody>
            <div className="form-stack-new">
              <FormField label="Environment">
                <Select
                  aria-label="Environment"
                  required
                  value={props.registrationForm.environmentId}
                  onChange={(event) =>
                    props.setRegistrationForm((current) => ({
                      ...current,
                      environmentId: event.target.value,
                    }))
                  }
                >
                  <option value="">Select environment</option>
                  {props.environments.map((environment) => (
                    <option key={environment.id} value={environment.id}>
                      {environment.name}
                    </option>
                  ))}
                </Select>
              </FormField>
              <div className="form-grid-new">
                <FormField label="Instance name">
                  <Input
                    required
                    value={props.registrationForm.instanceName}
                    onChange={(event) =>
                      props.setRegistrationForm((current) => ({
                        ...current,
                        instanceName: event.target.value,
                      }))
                    }
                  />
                </FormField>
                <FormField label="Address / hostname">
                  <Input
                    required
                    value={props.registrationForm.address}
                    onChange={(event) =>
                      props.setRegistrationForm((current) => ({
                        ...current,
                        address: event.target.value,
                      }))
                    }
                  />
                </FormField>
              </div>
              <FormField label="Description">
                <Textarea
                  value={props.registrationForm.description}
                  onChange={(event) =>
                    props.setRegistrationForm((current) => ({
                      ...current,
                      description: event.target.value,
                    }))
                  }
                />
              </FormField>
              <FormSection
                title="Endpoints"
                description="Choose one primary endpoint."
              >
                <div className="endpoint-editor-section-new">
                  <ActionGroup>
                    <Button
                      onClick={props.addRegistrationEndpoint}
                      size="sm"
                      type="button"
                    >
                      <Plus size={14} />
                      Add endpoint
                    </Button>
                  </ActionGroup>
                  {props.registrationForm.endpoints.map((endpoint, index) => (
                    <div className="endpoint-draft-new" key={index}>
                      <FormField label="Protocol">
                        <Select
                          aria-label={`Endpoint ${index + 1} protocol`}
                          value={endpoint.protocol}
                          onChange={(event) =>
                            props.updateRegistrationEndpoint(index, {
                              protocol: event.target.value,
                            })
                          }
                        >
                          {protocolOptions.map(([value, label]) => (
                            <option key={value} value={value}>
                              {label}
                            </option>
                          ))}
                        </Select>
                      </FormField>
                      <FormField label="Port">
                        <Input
                          aria-label={`Endpoint ${index + 1} port`}
                          max={65535}
                          min={1}
                          required
                          type="number"
                          value={endpoint.port}
                          onChange={(event) =>
                            props.updateRegistrationEndpoint(index, {
                              port: Number(event.target.value),
                            })
                          }
                        />
                      </FormField>
                      <FormField label="Path">
                        <Input
                          aria-label={`Endpoint ${index + 1} path`}
                          value={endpoint.path}
                          onChange={(event) =>
                            props.updateRegistrationEndpoint(index, {
                              path: event.target.value,
                            })
                          }
                        />
                      </FormField>
                      <FormField label="Primary">
                        <input
                          className="alauda-radio"
                          aria-label={`Endpoint ${index + 1} primary`}
                          checked={endpoint.primary}
                          name="primary-endpoint"
                          type="radio"
                          onChange={() =>
                            props.setPrimaryRegistrationEndpoint(index)
                          }
                        />
                      </FormField>
                      <IconButton
                        disabled={props.registrationForm.endpoints.length === 1}
                        label="Remove endpoint"
                        onClick={() => props.removeRegistrationEndpoint(index)}
                        type="button"
                        variant="ghost"
                      >
                        <Trash2 size={14} />
                      </IconButton>
                    </div>
                  ))}
                </div>
              </FormSection>
              {props.registrationSuccess ? (
                <p className="form-success-new" role="status">
                  {props.registrationSuccess}
                </p>
              ) : null}
              {props.registeredRuntime ? (
                <p className="form-success-new" role="status">
                  Instance added.{" "}
                  {props.healthFollowUpConfigured
                    ? "Health monitoring configured."
                    : "Health monitoring can be configured from Health."}
                </p>
              ) : null}
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              onClick={props.closeRuntimeDialog}
              type="button"
              variant="ghost"
            >
              Cancel
            </Button>
            <Button
              disabled={props.savingRegistration}
              loading={props.savingRegistration}
              type="submit"
              variant="primary"
            >
              Add instance
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function RegisteredInstanceSuccess(
  props: ServicesWorkspaceProps & { service: Service },
) {
  const [configuring, setConfiguring] = useState(false);
  const runtime = props.registeredRuntime;
  if (!runtime) return null;
  return (
    <Dialog open onOpenChange={(value) => !value && props.closeRuntimeDialog()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>Instance added</DialogTitle>
          <DialogDescription>
            The instance and its endpoints are ready.
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          <div className="success-summary-new">
            <strong>{runtime.name}</strong>
            <span>{runtime.address}</span>
            <div>
              <span>Endpoints</span>
              {runtime.endpoints.map((endpoint) => (
                <span key={endpoint.id}>
                  {formatProtocol(endpoint.protocol)} :{endpoint.port}
                  {endpoint.path || ""}
                </span>
              ))}
            </div>
          </div>
          {configuring && !props.healthFollowUpError ? (
            <div className="form-grid-new">
              <FormField label="Check name">
                <Input
                  required
                  value={props.registrationForm.healthName}
                  onChange={(event) =>
                    props.setRegistrationForm((current) => ({
                      ...current,
                      healthName: event.target.value,
                    }))
                  }
                />
              </FormField>
              <FormField label="Check type">
                <Select
                  value={props.registrationForm.healthType}
                  onChange={(event) =>
                    props.setRegistrationForm((current) => ({
                      ...current,
                      healthType: event.target.value,
                    }))
                  }
                >
                  <option value="HEALTH_CHECK_TYPE_HTTP">HTTP</option>
                  <option value="HEALTH_CHECK_TYPE_HTTPS">HTTPS</option>
                  <option value="HEALTH_CHECK_TYPE_TCP">TCP</option>
                  <option value="HEALTH_CHECK_TYPE_GRPC">gRPC</option>
                </Select>
              </FormField>
              <FormField label="Interval (seconds)">
                <Input
                  min={1}
                  type="number"
                  value={props.registrationForm.healthIntervalSeconds}
                  onChange={(event) =>
                    props.setRegistrationForm((current) => ({
                      ...current,
                      healthIntervalSeconds: Number(event.target.value),
                    }))
                  }
                />
              </FormField>
              <FormField label="Timeout (seconds)">
                <Input
                  min={1}
                  type="number"
                  value={props.registrationForm.healthTimeoutSeconds}
                  onChange={(event) =>
                    props.setRegistrationForm((current) => ({
                      ...current,
                      healthTimeoutSeconds: Number(event.target.value),
                    }))
                  }
                />
              </FormField>
            </div>
          ) : null}
          {props.healthFollowUpError ? (
            <Alert
              tone="danger"
              title="Health monitoring could not be configured."
            >
              The instance and its endpoints already exist.
            </Alert>
          ) : props.healthFollowUpConfigured ? (
            <p className="form-success-new" role="status">
              Health monitoring configured.
            </p>
          ) : (
            <p>Health monitoring is not configured.</p>
          )}
        </DialogBody>
        <DialogFooter>
          {configuring && !props.healthFollowUpError ? (
            <>
              <Button
                onClick={() => setConfiguring(false)}
                type="button"
                variant="ghost"
              >
                Cancel
              </Button>
              <Button
                disabled={props.savingHealthFollowUp}
                loading={props.savingHealthFollowUp}
                onClick={props.handleCreateRegisteredHealth}
                type="button"
                variant="primary"
              >
                Save health monitoring
              </Button>
            </>
          ) : (
            <>
              <Button
                onClick={() => setConfiguring(true)}
                type="button"
                variant="secondary"
              >
                {props.healthFollowUpError ? "Retry" : "Configure health"}
              </Button>
              <Button
                onClick={props.closeRuntimeDialog}
                type="button"
                variant="primary"
              >
                Done
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function HealthDialog(props: ServicesWorkspaceProps) {
  const editing = props.editingHealthCheckId !== "new";
  const check = props.healthChecks.find(
    (check) => check.id === props.editingHealthCheckId,
  );
  const values = editing ? props.healthEditForm : props.healthForm;
  function timing(
    key:
      | "intervalSeconds"
      | "timeoutSeconds"
      | "failuresBeforeUnhealthy"
      | "successesBeforeHealthy",
    value: number,
  ) {
    if (editing)
      props.setHealthEditForm((current) => ({ ...current, [key]: value }));
    else props.setHealthForm((current) => ({ ...current, [key]: value }));
  }
  return (
    <Dialog
      open
      onOpenChange={(open) => !open && props.setEditingHealthCheckId("")}
    >
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>
            {editing ? "Edit health check" : "Configure health"}
          </DialogTitle>
          <DialogDescription>
            {editing
              ? check?.name || "Health check"
              : "Monitor an instance or one of its endpoints."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={props.handleUpdateHealthCheck}>
          <DialogBody>
            {props.error ? <Alert tone="danger" title={props.error} /> : null}
            <div className="grid gap-3 sm:grid-cols-2">
              {!editing ? (
                <>
                  <FormField label="Instance">
                    <Select
                      required
                      value={props.healthForm.instanceId}
                      onChange={(event) =>
                        props.setHealthForm((current) => ({
                          ...current,
                          instanceId: event.target.value,
                          endpointId: "",
                        }))
                      }
                    >
                      <option value="">Select instance</option>
                      {props.instances.map((instance) => (
                        <option key={instance.id} value={instance.id}>
                          {instance.name}
                        </option>
                      ))}
                    </Select>
                  </FormField>
                  <FormField label="Endpoint">
                    <Select
                      value={props.healthForm.endpointId}
                      onChange={(event) =>
                        props.setHealthForm((current) => ({
                          ...current,
                          endpointId: event.target.value,
                        }))
                      }
                    >
                      <option value="">Instance address</option>
                      {props.endpoints
                        .filter(
                          (endpoint) =>
                            endpoint.instanceId === props.healthForm.instanceId,
                        )
                        .map((endpoint) => (
                          <option key={endpoint.id} value={endpoint.id}>
                            {endpoint.name} ·{" "}
                            {formatProtocol(endpoint.protocol)} :{endpoint.port}
                          </option>
                        ))}
                    </Select>
                  </FormField>
                  <FormField label="Check name">
                    <Input
                      required
                      value={props.healthForm.name}
                      onChange={(event) =>
                        props.setHealthForm((current) => ({
                          ...current,
                          name: event.target.value,
                        }))
                      }
                    />
                  </FormField>
                  <FormField label="Type">
                    <Select
                      value={props.healthForm.type}
                      onChange={(event) =>
                        props.setHealthForm((current) => ({
                          ...current,
                          type: event.target.value,
                        }))
                      }
                    >
                      <option value="HEALTH_CHECK_TYPE_HTTP">HTTP</option>
                      <option value="HEALTH_CHECK_TYPE_HTTPS">HTTPS</option>
                      <option value="HEALTH_CHECK_TYPE_TCP">TCP</option>
                      <option value="HEALTH_CHECK_TYPE_GRPC">gRPC</option>
                    </Select>
                  </FormField>
                  <FormField label="Path">
                    <Input
                      value={props.healthForm.path}
                      onChange={(event) =>
                        props.setHealthForm((current) => ({
                          ...current,
                          path: event.target.value,
                        }))
                      }
                    />
                  </FormField>
                  <FormField label="Expected status">
                    <Input
                      value={props.healthForm.expectedStatus}
                      onChange={(event) =>
                        props.setHealthForm((current) => ({
                          ...current,
                          expectedStatus: event.target.value,
                        }))
                      }
                    />
                  </FormField>
                </>
              ) : (
                <FormField label="Enabled">
                  <Switch
                    checked={props.healthEditForm.enabled}
                    onCheckedChange={(enabled) =>
                      props.setHealthEditForm((current) => ({
                        ...current,
                        enabled,
                      }))
                    }
                  />
                </FormField>
              )}
              <FormField label="Interval seconds">
                <Input
                  required
                  min={1}
                  type="number"
                  value={values.intervalSeconds}
                  onChange={(event) =>
                    timing("intervalSeconds", Number(event.target.value))
                  }
                />
              </FormField>
              <FormField label="Timeout seconds">
                <Input
                  required
                  min={1}
                  type="number"
                  value={values.timeoutSeconds}
                  onChange={(event) =>
                    timing("timeoutSeconds", Number(event.target.value))
                  }
                />
              </FormField>
              <FormField label="Failures before unhealthy">
                <Input
                  required
                  min={1}
                  type="number"
                  value={values.failuresBeforeUnhealthy}
                  onChange={(event) =>
                    timing(
                      "failuresBeforeUnhealthy",
                      Number(event.target.value),
                    )
                  }
                />
              </FormField>
              <FormField label="Successes before healthy">
                <Input
                  required
                  min={1}
                  type="number"
                  value={values.successesBeforeHealthy}
                  onChange={(event) =>
                    timing("successesBeforeHealthy", Number(event.target.value))
                  }
                />
              </FormField>
              <FormField label="Description">
                <Textarea
                  value={values.description}
                  onChange={(event) =>
                    editing
                      ? props.setHealthEditForm((current) => ({
                          ...current,
                          description: event.target.value,
                        }))
                      : props.setHealthForm((current) => ({
                          ...current,
                          description: event.target.value,
                        }))
                  }
                />
              </FormField>
            </div>
          </DialogBody>
          <DialogFooter>
            <Button
              variant="ghost"
              onClick={() => props.setEditingHealthCheckId("")}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={props.savingHealthCheck}
              loading={props.savingHealthCheck}
            >
              Save health check
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function serviceStatus(service: Service, incidents: Incident[]) {
  return incidents.some(
    (incident) =>
      incident.serviceId === service.id &&
      incident.state === "INCIDENT_STATE_OPEN",
  )
    ? "degraded"
    : "healthy";
}
function environmentName(environments: Environment[], id: string) {
  return (
    environments.find((environment) => environment.id === id)?.name ??
    `Environment ${id}`
  );
}
function formatProtocol(value: string) {
  return value.replace("PROTOCOL_", "").toUpperCase();
}
function formatHealthState(value: string) {
  return value.replace("HEALTH_STATE_", "").toLowerCase();
}
