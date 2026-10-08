import { FormEvent, useEffect, useState } from "react";
import { ServiceHealthView } from "./ServiceHealthView";
import { HealthCheckForm } from "./HealthCheckForm";
import { EndpointEditor } from "./EndpointEditor";
import {
  validateEndpointCollection,
  type EndpointFormValue,
} from "../lib/endpoint-form";
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
  endpoints: EndpointFormValue[];
};
type ServiceForm = { name: string; displayName: string; description: string };
type InstanceEditForm = {
  address: string;
  description: string;
  enabled: boolean;
};
type EndpointEditForm = EndpointFormValue;

type ServicesWorkspaceProps = {
  onHealthSaved: () => Promise<void>;
  onHealthResults: (serviceId?: string, checkId?: string) => void;
  onConfigureInstance?: (instance: ServiceInstance) => void;
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
    instanceId: string;
    name: string;
    address: string;
    endpoints: Endpoint[];
  };
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
  healthTargetInstanceId: string;
  setHealthTargetInstanceId: (id: string) => void;
  handleDeleteHealthCheck: (check: HealthCheck) => void;
  savingHealthCheck: boolean;
  handleRunHealthCheck: (id: string) => Promise<void>;
  startEditHealthCheck: (check: HealthCheck) => void;
  testResult: string;
  setError: (value: string) => void;
};

export function ServicesWorkspace(props: ServicesWorkspaceProps) {
  const [healthInstanceContext, setHealthInstanceContext] = useState<string>();
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
                    onConfigureInstance={(instance) =>
                      setHealthInstanceContext(instance.id)
                    }
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
                    onResults={(checkId) =>
                      props.onHealthResults(selectedService.id, checkId)
                    }
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
                      setHealthInstanceContext(undefined);
                      props.onConfigureInstance?.(instance);
                      props.setEditingHealthCheckId("new");
                      props.setHealthTargetInstanceId(instance.id);
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
          lockedInstanceId={healthInstanceContext}
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
    props.onConfigureInstance?.(instance);
    props.setHealthTargetInstanceId(instance.id);
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
      <DialogContent size={editorOpen ? "md" : "lg"}>
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
          {props.error &&
          !(editorOpen && props.error.startsWith("An endpoint named")) ? (
            <Alert tone="danger" title={props.error} />
          ) : null}
          {editorOpen ? (
            <EndpointForm {...props} onClose={closeEditor} />
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

function EndpointForm(
  props: ServicesWorkspaceProps & {
    onClose: () => void;
    endpoints: Endpoint[];
  },
) {
  const instanceId =
    props.addingEndpointInstanceId ||
    props.endpoints.find((endpoint) => endpoint.id === props.editingEndpointId)
      ?.instanceId;
  const existingNames = props.endpoints
    .filter(
      (endpoint) =>
        endpoint.instanceId === instanceId &&
        endpoint.id !== props.editingEndpointId,
    )
    .map((endpoint) => endpoint.name);
  const nameError =
    props.error.startsWith("An endpoint named") &&
    props.error.includes(`"${props.endpointEditForm.name.trim()}"`)
      ? props.error
      : undefined;
  return (
    <form className="grid gap-3" onSubmit={props.handleUpdateEndpoint}>
      <EndpointEditor
        value={props.endpointEditForm}
        onChange={props.setEndpointEditForm}
        existingNames={existingNames}
        errors={nameError ? { name: nameError } : {}}
        autoFocus
        disabled={props.savingRuntimeEdit}
      />
      <ActionGroup>
        <Button onClick={props.onClose} type="button" variant="ghost">
          Cancel
        </Button>
        <Button
          disabled={props.savingRuntimeEdit}
          type="submit"
          variant="primary"
        >
          Save
        </Button>
      </ActionGroup>
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
                description="Names are unique within this instance. At most one endpoint can be primary."
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
                    <fieldset
                      className="min-w-0 border-t border-[var(--border)] pt-3"
                      key={index}
                      aria-label={`Endpoint ${index + 1}`}
                    >
                      <EndpointEditor
                        value={endpoint}
                        labelPrefix={
                          props.registrationForm.endpoints.length > 1
                            ? `Endpoint ${index + 1}`
                            : ""
                        }
                        existingNames={props.registrationForm.endpoints
                          .filter((_, other) => other !== index)
                          .map((value) => value.name.trim())}
                        errors={
                          validateEndpointCollection(
                            props.registrationForm.endpoints,
                          )[index]
                        }
                        disabled={props.savingRegistration}
                        primaryLocked={
                          props.registrationForm.endpoints.length === 1
                        }
                        onChange={(value) =>
                          props.updateRegistrationEndpoint(index, value)
                        }
                        onRemove={() => props.removeRegistrationEndpoint(index)}
                      />
                    </fieldset>
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
                  Instance added. Health monitoring can be configured from
                  Health.
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
  const [configured, setConfigured] = useState(false);
  const runtime = props.registeredRuntime;
  if (!runtime) return null;
  const instance = props.instances.find((i) => i.id === runtime.instanceId) || {
    id: runtime.instanceId,
    name: runtime.name,
    address: runtime.address,
    deploymentId: "",
    port: runtime.endpoints[0]?.port || 0,
    enabled: true,
  };
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) props.closeRuntimeDialog();
      }}
    >
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>Instance added</DialogTitle>
          <DialogDescription>
            {runtime.name} · {runtime.address}
          </DialogDescription>
        </DialogHeader>
        <DialogBody>
          {!configuring ? (
            <div className="grid gap-2">
              <strong>Endpoints</strong>
              {runtime.endpoints.map((e) => (
                <span key={e.id}>
                  {e.name} · {formatProtocol(e.protocol)} :{e.port}
                  {e.path || ""}
                </span>
              ))}
              <p role="status">
                {configured
                  ? "Health monitoring configured."
                  : "Health monitoring is not configured."}
              </p>
            </div>
          ) : (
            <HealthCheckForm
              instances={[instance]}
              endpoints={runtime.endpoints}
              instanceId={instance.id}
              endpointId={
                runtime.endpoints.find((e) => e.primary)?.id ||
                runtime.endpoints[0]?.id
              }
              onCancel={() => setConfiguring(false)}
              onSaved={async () => {
                setConfigured(true);
                setConfiguring(false);
                await props.onHealthSaved();
              }}
            />
          )}
        </DialogBody>
        {!configuring ? (
          <DialogFooter>
            {!configured ? (
              <Button onClick={() => setConfiguring(true)}>
                Configure health
              </Button>
            ) : null}
            <Button variant="primary" onClick={props.closeRuntimeDialog}>
              Done
            </Button>
          </DialogFooter>
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

function HealthDialog(
  props: ServicesWorkspaceProps & { lockedInstanceId?: string },
) {
  const check = props.healthChecks.find(
    (c) => c.id === props.editingHealthCheckId,
  );
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) props.setEditingHealthCheckId("");
      }}
    >
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>
            {check ? "Edit health check" : "Configure health"}
          </DialogTitle>
        </DialogHeader>
        <DialogBody>
          <HealthCheckForm
            key={props.editingHealthCheckId}
            check={check}
            instances={props.instances}
            endpoints={props.endpoints}
            serviceId={props.selectedServiceId}
            initialInstanceId={props.healthTargetInstanceId}
            instanceId={props.lockedInstanceId}
            environmentId={props.selectedEnvironmentId}
            onCancel={() => props.setEditingHealthCheckId("")}
            onSaved={async () => {
              props.setEditingHealthCheckId("");
              await props.onHealthSaved();
            }}
          />
        </DialogBody>
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
    environments.find((environment) => environment.id === id)?.name ||
    "Unknown environment"
  );
}
function formatProtocol(value: string) {
  return value.replace("PROTOCOL_", "").toUpperCase();
}
function formatHealthState(value: string) {
  return value.replace("HEALTH_STATE_", "").toLowerCase();
}
