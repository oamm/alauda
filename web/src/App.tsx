import { FormEvent, useEffect, useMemo, useState } from "react";

import "./App.css";
import { newEndpoint, validateEndpoint, validateEndpointCollection, type EndpointFormValue } from "./lib/endpoint-form";
import "./components/ui/ui.css";
import { AppShell } from "./components/AppShell";
import { FormField, Input, PasswordInput, Alert, Button as FormButton } from "./components/ui";
import { AvailabilityCard, MetricCard } from "./components/Cards";
import {
  EmptyState,
  Button,
  IconButton,
  PageHeader,
  ResourceLink,
  StatusBadge,
} from "./components/OperationsUI";
import { ActiveView } from "./types";
import {
  formatActivityTimestamp,
  formatDuration,
  formatEventType,
  formatTimestamp,
  pluralize,
} from "./utils/format";
import { RuntimeTopology } from "./components/RuntimeTopology";
import { HealthWorkspace } from "./views/HealthWorkspace";
import { HealthResultsPage } from "./components/HealthResultsPage";
import { ServicesWorkspace } from "./components/ServicesWorkspace";
import { OperationalWorkspace } from "./components/OperationalWorkspace";
import {
  AlertPolicy,
  ApplicationKey,
  changePassword,
  ApiToken,
  createApiToken,
  createApplicationKey,
  createAlertPolicy,
  createEndpoint,
  createNotificationChannel,
  createService,
  createUser,
  AvailabilitySummary,
  deleteEndpoint,
  deleteHealthCheck,
  deleteInstance,
  deleteService,
  Endpoint,
  Environment,
  EventRecord,
  HealthCheck,
  HealthStateView,
  Incident,
  listApiTokens,
  listApplicationKeys,
  NotificationChannel,
  ServiceDeployment,
  ServiceInstance,
  eventStreamUrl,
  getAvailability,
  getCurrentSession,
  listAlertPolicies,
  listDeployments,
  listEndpoints,
  listHealthChecks,
  listEnvironments,
  listEvents,
  listIncidents,
  listInstances,
  listNotificationChannels,
  listServices,
  listSessions,
  listUsers,
  login,
  logout,
  registerRuntime,
  resolveIncident,
  revokeApiToken,
  revokeApplicationKey,
  revokeSession,
  runHealthCheck,
  Service,
  setBearerToken,
  setAuthenticationFailureHandler,
  testNotificationChannel,
  updateAlertPolicy,
  updateEndpoint,
  updateInstance,
  updateNotificationChannel,
  updateService,
  UserAccount,
} from "./api";

function endpointMutationError(error: unknown, name: string): string {
  const message = error instanceof Error ? error.message : "Failed to update endpoint";
  try {
    const response = JSON.parse(message);
    if (response?.code === "already_exists") {
      return `An endpoint named "${name}" already exists in this instance. Choose a different name.`;
    }
    if (typeof response?.message === "string") return response.message;
  } catch {
    // Connect errors may also be plain text.
  }
  return message;
}

function newRegistrationEndpoint(overrides: Partial<EndpointFormValue> = {}) {
  return newEndpoint([], overrides);
}

function AuthenticatedApp({ onLogout }: { onLogout: () => Promise<void> }) {
  const initialPath = parseApplicationPath(window.location.pathname);
  const [activeView, setActiveView] = useState<ActiveView>(initialPath.view);
  const [healthResultsRoute,setHealthResultsRoute] = useState(window.location.pathname === "/health/results");
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [services, setServices] = useState<Service[]>([]);
  const [deployments, setDeployments] = useState<ServiceDeployment[]>([]);
  const [instances, setInstances] = useState<ServiceInstance[]>([]);
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [healthChecks, setHealthChecks] = useState<HealthCheck[]>([]);
  const [healthStates, setHealthStates] = useState<HealthStateView[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [events, setEvents] = useState<EventRecord[]>([]);
  const [notificationChannels, setNotificationChannels] = useState<
    NotificationChannel[]
  >([]);
  const [alertPolicies, setAlertPolicies] = useState<AlertPolicy[]>([]);
  const [users, setUsers] = useState<UserAccount[]>([]);
  const [apiTokens, setApiTokens] = useState<ApiToken[]>([]);
  const [applicationKeys, setApplicationKeys] = useState<ApplicationKey[]>([]);
  const [sessions, setSessions] = useState<import("./api").Session[]>([]);
  const [availability, setAvailability] = useState<{
    availability24h?: AvailabilitySummary;
    availability7d?: AvailabilitySummary;
    availability30d?: AvailabilitySummary;
  }>({});
  const [serviceAvailability, setServiceAvailability] = useState<{
    availability24h?: AvailabilitySummary;
    availability7d?: AvailabilitySummary;
    availability30d?: AvailabilitySummary;
  }>({});
  const [selectedEnvironmentId, setSelectedEnvironmentId] = useState("");
  const [selectedServiceId, setSelectedServiceId] = useState(initialPath.serviceId);
  const [serviceSearch, setServiceSearch] = useState("");
  const [serviceHealthFilter, setServiceHealthFilter] = useState("all");
  const [serviceTagFilter, setServiceTagFilter] = useState("");
  const [servicePage, setServicePage] = useState(1);
  const [serviceTab, setServiceTab] = useState<
    "overview" | "instances" | "availability" | "health" | "incidents" | "events"
  >(initialPath.serviceTab);
  const [showCreateService, setShowCreateService] = useState(false);
  const [showAddRuntime, setShowAddRuntime] = useState(false);
  const [healthStatusFilter, setHealthStatusFilter] = useState("all");
  const [selectedBulkServiceIds, setSelectedBulkServiceIds] = useState<
    string[]
  >([]);
  const [incidentStateFilter, setIncidentStateFilter] = useState("all");
  const [incidentSearch, setIncidentSearch] = useState("");
  const [selectedIncidentId, setSelectedIncidentId] = useState("");
  const [eventTypeFilter, setEventTypeFilter] = useState("");
  const [eventResourceFilter, setEventResourceFilter] = useState("all");
  const [eventSearch, setEventSearch] = useState("");
  const [editingChannelId, setEditingChannelId] = useState("");
  const [editingPolicyId, setEditingPolicyId] = useState("");
  const [editingServiceId, setEditingServiceId] = useState("");
  const [editingInstanceId, setEditingInstanceId] = useState("");
  const [editingEndpointId, setEditingEndpointId] = useState("");
  const [addingEndpointInstanceId, setAddingEndpointInstanceId] = useState("");
  const [editingHealthCheckId, setEditingHealthCheckId] = useState("");
  const [securitySection, setSecuritySection] = useState<
    "users" | "tokens" | "applicationKeys" | "sessions"
  >("users");
  const [showCreateApplicationKey, setShowCreateApplicationKey] = useState(false);
  const [applicationKeySecret, setApplicationKeySecret] = useState("");
  const [showCreateUser, setShowCreateUser] = useState(false);
  const [showCreateApiToken, setShowCreateApiToken] = useState(false);
  const [alertsSection, setAlertsSection] = useState<"policies" | "channels">(
    "policies",
  );
  const [darkMode, setDarkMode] = useState(false);
  const [loading, setLoading] = useState(true);
  const [savingService, setSavingService] = useState(false);
  const [savingRegistration, setSavingRegistration] = useState(false);
  const [savingRuntimeEdit, setSavingRuntimeEdit] = useState(false);
  const [savingHealthCheck, setSavingHealthCheck] = useState(false);
  const [savingChannel, setSavingChannel] = useState(false);
  const [savingPolicy, setSavingPolicy] = useState(false);
  const [bulkActionRunning, setBulkActionRunning] = useState(false);
  const [resolvingIncidentId, setResolvingIncidentId] = useState("");
  const [testingChannelId, setTestingChannelId] = useState("");
  const [testResult, setTestResult] = useState("");
  const [authToken] = useState("");
  const [authMessage, setAuthMessage] = useState("");
  const [error, setError] = useState("");
  const [registrationSuccess, setRegistrationSuccess] = useState("");
  const [registeredRuntime, setRegisteredRuntime] = useState<{
    name: string;
    address: string;
    instanceId: string;
    endpoints: Endpoint[];
  } | null>(null);
  const [serviceForm, setServiceForm] = useState({
    name: "",
    displayName: "",
    description: "",
  });
  const [serviceEditForm, setServiceEditForm] = useState({
    name: "",
    displayName: "",
    description: "",
  });
  const [registrationStep, setRegistrationStep] = useState(0);
  const [registrationForm, setRegistrationForm] = useState({
    environmentId: "",
    instanceName: "",
    address: "",
    description: "",
    endpoints: [newRegistrationEndpoint()],
  });
  const [instanceEditForm, setInstanceEditForm] = useState({
    address: "",
    description: "",
    enabled: true,
  });
  const [endpointEditForm, setEndpointEditForm] = useState({
    name: "",
    protocol: "PROTOCOL_HTTP",
    port: 8080,
    path: "/",
    enabled: true,
    primary: false,
  });
  const [healthTargetInstanceId, setHealthTargetInstanceId] = useState("");
  const [channelForm, setChannelForm] = useState({
    type: "webhook",
    name: "",
    description: "",
    url: "",
    smtpHost: "",
    smtpPort: "587",
    from: "",
    to: "",
  });
  const [policyForm, setPolicyForm] = useState({
    environmentId: "",
    deploymentId: "",
    notifyOn: "unhealthy,recovered",
    cooldownMinutes: 15,
    sendRecoveryNotification: true,
    channelIds: "",
  });
  const [userForm, setUserForm] = useState({
    username: "",
    email: "",
    displayName: "",
    password: "",
    role: "Viewer",
  });
  const [tokenForm, setTokenForm] = useState({
    userId: "",
    name: "",
    scopes: "read",
    expiresInHours: "720",
  });
  const [applicationKeyForm, setApplicationKeyForm] = useState({
    name: "",
    scopes: "read",
    environmentIds: "",
    expiresInHours: "720",
  });

  useEffect(() => {
    setBearerToken(authToken);
  }, [authToken]);

  useEffect(() => {
    if (!showCreateApplicationKey && !showCreateUser && !showCreateApiToken) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        closeApplicationKeyFlow();
        setShowCreateUser(false);
        setShowCreateApiToken(false);
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [showCreateApplicationKey, showCreateUser, showCreateApiToken]);

  const selectedServiceDeployments = useMemo(
    () =>
      deployments.filter(
        (deployment) => deployment.serviceId === selectedServiceId,
      ),
    [deployments, selectedServiceId],
  );
  const selectedEnvironment = useMemo(
    () =>
      environments.find(
        (environment) =>
          environment.id ===
          (selectedEnvironmentId || registrationForm.environmentId),
      ),
    [environments, registrationForm.environmentId, selectedEnvironmentId],
  );
  const selectedEnvironmentDeployment = useMemo(
    () =>
      selectedServiceDeployments.find(
        (deployment) =>
          deployment.environmentId ===
          (selectedEnvironmentId || registrationForm.environmentId),
      ),
    [
      registrationForm.environmentId,
      selectedEnvironmentId,
      selectedServiceDeployments,
    ],
  );
  const selectedServiceInstances = useMemo(() => {
    const deploymentIDs = new Set(
      selectedServiceDeployments.map((deployment) => deployment.id),
    );
    return instances.filter((instance) =>
      deploymentIDs.has(instance.deploymentId),
    );
  }, [instances, selectedServiceDeployments]);
  const selectedEnvironmentInstances = useMemo(() => {
    if (!selectedEnvironmentDeployment) {
      return [];
    }
    return instances.filter(
      (instance) => instance.deploymentId === selectedEnvironmentDeployment.id,
    );
  }, [instances, selectedEnvironmentDeployment]);
  const selectedServiceInstanceById = useMemo(
    () =>
      new Map(
        selectedServiceInstances.map((instance) => [instance.id, instance]),
      ),
    [selectedServiceInstances],
  );
  const servicePageSize = 10;
  const filteredServices = useMemo(() => {
    const query = serviceSearch.trim().toLowerCase();
    const tagQuery = serviceTagFilter.trim().toLowerCase();
    return services.filter((service) => {
      const searchText = [
        service.name,
        service.displayName,
        service.description,
        formatMap(service.tags),
        formatMap(service.metadata),
      ]
        .join(" ")
        .toLowerCase();
      if (query && !searchText.includes(query)) {
        return false;
      }
      if (
        tagQuery &&
        !formatMap(service.tags).toLowerCase().includes(tagQuery)
      ) {
        return false;
      }
      if (serviceHealthFilter === "all") {
        return true;
      }
      const serviceStatus = serviceOperationalStatus(service, incidents);
      return serviceHealthFilter === serviceStatus;
    });
  }, [
    incidents,
    serviceHealthFilter,
    serviceSearch,
    serviceTagFilter,
    services,
  ]);
  const servicePageCount = Math.max(
    1,
    Math.ceil(filteredServices.length / servicePageSize),
  );
  const visibleServices = filteredServices.slice(
    (servicePage - 1) * servicePageSize,
    servicePage * servicePageSize,
  );
  const filteredIncidents = useMemo(() => {
    const query = incidentSearch.trim().toLowerCase();
    return incidents.filter((incident) => {
      if (
        incidentStateFilter !== "all" &&
        formatIncidentState(incident.state) !== incidentStateFilter
      ) {
        return false;
      }
      if (!query) {
        return true;
      }
      return [
        incident.id,
        incident.reason,
        incident.impactSummary,
        incident.instanceId,
        incident.deploymentId,
        incident.serviceId,
        incident.environmentId,
      ]
        .join(" ")
        .toLowerCase()
        .includes(query);
    });
  }, [incidentSearch, incidentStateFilter, incidents]);
  const selectedIncident = useMemo(
    () => incidents.find((incident) => incident.id === selectedIncidentId),
    [incidents, selectedIncidentId],
  );
  const filteredEvents = useMemo(() => {
    const query = eventSearch.trim().toLowerCase();
    return events.filter((event) => {
      if (eventTypeFilter && event.type !== eventTypeFilter) {
        return false;
      }
      if (
        eventResourceFilter !== "all" &&
        event.resourceType !== eventResourceFilter
      ) {
        return false;
      }
      if (!query) {
        return true;
      }
      return [
        event.type,
        event.message,
        event.actor,
        event.resourceType,
        event.resourceId,
      ]
        .join(" ")
        .toLowerCase()
        .includes(query);
    });
  }, [eventResourceFilter, eventSearch, eventTypeFilter, events]);
  const eventTypes = useMemo(
    () => Array.from(new Set(events.map((event) => event.type))).sort(),
    [events],
  );
  const eventResourceTypes = useMemo(
    () => Array.from(new Set(events.map((event) => event.resourceType))).sort(),
    [events],
  );

  async function loadCatalog(environmentId = selectedEnvironmentId) {
    setError("");
    const [
      nextEnvironments,
      nextServices,
      nextHealthChecks,
      nextIncidents,
      nextEvents,
      nextAvailability,
      nextChannels,
      nextPolicies,
    ] = await Promise.all([
      listEnvironments(),
      listServices(environmentId),
      listHealthChecks(),
      listIncidents({ environmentId: environmentId || undefined }),
      listEvents({ environmentId: environmentId || undefined }),
      getAvailability({ environmentId: environmentId || undefined }),
      listNotificationChannels(),
      listAlertPolicies({ environmentId: environmentId || undefined }),
    ]);
    setEnvironments(nextEnvironments);
    setServices(nextServices);
    setHealthChecks(nextHealthChecks);
    setIncidents(nextIncidents);
    setEvents(nextEvents);
    setAvailability(nextAvailability);
    setNotificationChannels(nextChannels);
    setAlertPolicies(nextPolicies);
    const nextDeployments = await listDeployments({
      environmentId: environmentId || undefined,
    });
    const visibleServiceIDs = new Set(
      nextServices.map((service) => service.id),
    );
    const scopedDeployments = nextDeployments.filter((deployment) =>
      visibleServiceIDs.has(deployment.serviceId),
    );
    const nextInstances = (
      await Promise.all(
        scopedDeployments.map((deployment) => listInstances(deployment.id)),
      )
    ).flat();
    const nextEndpoints = (
      await Promise.all(
        nextInstances.map((instance) => listEndpoints(instance.id)),
      )
    ).flat();
    // Some deployed API versions do not expose instance state yet. Keep the
    // topology usable and let the service UI render an explicit unknown state.
    const nextHealthStates: HealthStateView[] = [];
    setDeployments(scopedDeployments);
    setInstances(nextInstances);
    setEndpoints(nextEndpoints);
    setHealthStates(nextHealthStates);
    setSelectedServiceId((current) => {
      if (nextServices.some((service) => service.id === current)) {
        return current;
      }
      return nextServices[0]?.id ?? "";
    });
    setServicePage(1);
  }

  async function loadOperationalData(environmentId = selectedEnvironmentId) {
    const [nextIncidents, nextEvents, nextAvailability] = await Promise.all([
      listIncidents({ environmentId: environmentId || undefined }),
      listEvents({ environmentId: environmentId || undefined }),
      getAvailability({ environmentId: environmentId || undefined }),
    ]);
    setIncidents(nextIncidents);
    setEvents(nextEvents);
    setAvailability(nextAvailability);
  }

  async function loadSecurityData(userId = tokenForm.userId) {
    const nextUsers = await listUsers();
    setUsers(nextUsers);
    const selectedUserId = userId || nextUsers[0]?.id || "";
    if (!tokenForm.userId && selectedUserId) {
      setTokenForm((current) => ({ ...current, userId: selectedUserId }));
    }
    if (selectedUserId) {
      setApiTokens(await listApiTokens(selectedUserId));
    } else {
      setApiTokens([]);
    }
    setApplicationKeys(await listApplicationKeys());
  }

  async function loadSessionData() {
    setSessions(await listSessions());
  }

  useEffect(() => {
    loadCatalog()
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : "Failed to load catalog");
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    const source = new EventSource(
      eventStreamUrl({ environmentId: selectedEnvironmentId || undefined }),
    );
    source.addEventListener("registry-event", (message) => {
      const event = JSON.parse((message as MessageEvent).data) as EventRecord;
      setEvents((current) => [event, ...current].slice(0, 100));
      loadOperationalData().catch(() => undefined);
    });
    source.onerror = () => {
      source.close();
    };
    return () => source.close();
  }, [selectedEnvironmentId]);

  useEffect(() => {
    setServicePage(1);
  }, [serviceHealthFilter, serviceSearch, serviceTagFilter]);

  useEffect(() => {
    setServicePage((current) => Math.min(current, servicePageCount));
  }, [servicePageCount]);

  useEffect(() => {
    const handlePopState = () => {
      const path = parseApplicationPath(window.location.pathname);
      setActiveView(path.view);
      setHealthResultsRoute(window.location.pathname === "/health/results");
      if (path.serviceId) setSelectedServiceId(path.serviceId);
      if (path.serviceTab) setServiceTab(path.serviceTab);
    };
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  useEffect(() => {
    const nextPath = activeView === "health" && healthResultsRoute ? "/health/results" : applicationPath(activeView, selectedServiceId, serviceTab);
    if (window.location.pathname !== nextPath) {
      window.history.replaceState({}, "", nextPath);
    }
  }, [activeView, selectedServiceId, serviceTab, healthResultsRoute]);

  function navigateHealthResults(serviceId?:string,checkId?:string,instanceId?:string,environmentId=selectedEnvironmentId) {
    const params=new URLSearchParams();if(serviceId)params.set("serviceId",serviceId);if(checkId)params.set("checkId",checkId);if(instanceId)params.set("instanceId",instanceId);if(environmentId)params.set("environmentId",environmentId);
    setActiveView("health");setHealthResultsRoute(true);window.history.pushState({},"",`/health/results${params.size?`?${params}`:""}`);
  }
  function navigateTo(view: ActiveView) {
    setHealthResultsRoute(false);
    setActiveView(view);
    const nextPath = applicationPath(view, view === "services" ? selectedServiceId : "", serviceTab);
    if (window.location.pathname !== nextPath) {
      window.history.pushState({}, "", nextPath);
    }
  }

  async function navigateServiceHealth(serviceId:string,environmentId:string) {
    if(environmentId!==selectedEnvironmentId)await handleEnvironmentChange(environmentId);
    setSelectedServiceId(serviceId);setServiceTab("health");setHealthResultsRoute(false);setActiveView("services");
    window.history.pushState({},"",`/services/${encodeURIComponent(serviceId)}/health`);
  }

  useEffect(() => {
    setRegistrationForm((current) => ({
      ...current,
      environmentId:
        selectedEnvironmentId ||
        current.environmentId ||
        environments[0]?.id ||
        "",
    }));
  }, [environments, selectedEnvironmentId, selectedServiceId]);

  useEffect(() => {
    if (!selectedServiceId) {
      setServiceAvailability({});
      return;
    }
    getAvailability({
      environmentId: selectedEnvironmentId || undefined,
      serviceId: selectedServiceId,
    })
      .then(setServiceAvailability)
      .catch(() => setServiceAvailability({}));
  }, [selectedEnvironmentId, selectedServiceId]);

  async function handleEnvironmentChange(environmentId: string) {
    setSelectedEnvironmentId(environmentId);
    setLoading(true);
    try {
      await loadCatalog(environmentId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load catalog");
    } finally {
      setLoading(false);
    }
  }

  async function handleLogout() {
    await onLogout();
    setUsers([]);
    setApiTokens([]);
  }

  async function handleCreateUser(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setAuthMessage("");
    try {
      const user = await createUser(userForm);
      setUserForm({
        username: "",
        email: "",
        displayName: "",
        password: "",
        role: "Viewer",
      });
      setAuthMessage(`Created user ${user.username}.`);
      await loadSecurityData(user.id);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create user");
    }
  }

  async function handleCreateApiToken(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setAuthMessage("");
    try {
      const expiresIn = Number(tokenForm.expiresInHours);
      const expiresAt =
        Number.isFinite(expiresIn) && expiresIn > 0
          ? new Date(Date.now() + expiresIn * 60 * 60 * 1000).toISOString()
          : undefined;
      const response = await createApiToken({
        userId: tokenForm.userId,
        name: tokenForm.name,
        scopes: tokenForm.scopes
          .split(",")
          .map((scope) => scope.trim())
          .filter(Boolean),
        expiresAt,
      });
      setTokenForm((current) => ({ ...current, name: "" }));
      setAuthMessage(
        response.secret
          ? `Token created. Secret: ${response.secret}`
          : "Token created.",
      );
      await loadSecurityData(tokenForm.userId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create token");
    }
  }

  async function handleRevokeApiToken(id: string) {
    setError("");
    setAuthMessage("");
    try {
      await revokeApiToken(id);
      setAuthMessage("Token revoked.");
      await loadSecurityData(tokenForm.userId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke token");
    }
  }

  async function handleCreateApplicationKey(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setAuthMessage("");
    try {
      const expiresIn = Number(applicationKeyForm.expiresInHours);
      const expiresAt = Number.isFinite(expiresIn) && expiresIn > 0
        ? new Date(Date.now() + expiresIn * 60 * 60 * 1000).toISOString()
        : undefined;
      const response = await createApplicationKey({
        name: applicationKeyForm.name,
        scopes: applicationKeyForm.scopes.split(",").map((scope) => scope.trim()).filter(Boolean),
        environmentIds: applicationKeyForm.environmentIds.split(",").map((id) => id.trim()).filter(Boolean),
        expiresAt,
      });
      setApplicationKeyForm((current) => ({ ...current, name: "" }));
      setApplicationKeySecret(response.secret ?? "");
      setAuthMessage("Application key created.");
      setApplicationKeys(await listApplicationKeys());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create application key");
    }
  }

  function closeApplicationKeyFlow() {
    setShowCreateApplicationKey(false);
    setApplicationKeySecret("");
    setApplicationKeyForm({ name: "", scopes: "read", environmentIds: "", expiresInHours: "720" });
  }

  async function copyApplicationKey() {
    if (!applicationKeySecret) return;
    await navigator.clipboard.writeText(applicationKeySecret);
    setAuthMessage("Application key copied.");
  }

  async function handleRevokeApplicationKey(id: string) {
    setError("");
    setAuthMessage("");
    try {
      await revokeApplicationKey(id);
      setAuthMessage("Application key revoked.");
      setApplicationKeys(await listApplicationKeys());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to revoke application key");
    }
  }

  async function handleCreateService(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingService(true);
    setError("");
    try {
      const service = await createService(serviceForm);
      setServiceForm({ name: "", displayName: "", description: "" });
      await loadCatalog(selectedEnvironmentId);
      setSelectedServiceId(service.id);
      setServiceTab("overview");
      setShowCreateService(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create service");
    } finally {
      setSavingService(false);
    }
  }


  async function handleUpdateService(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const current = services.find((service) => service.id === editingServiceId);
    if (!current) {
      return;
    }
    setSavingService(true);
    setError("");
    try {
      const service = await updateService({
        id: current.id,
        displayName: serviceEditForm.displayName,
        description: serviceEditForm.description,
        tags: current.tags ?? {},
        metadata: current.metadata ?? {},
      });
      await loadCatalog(selectedEnvironmentId);
      setSelectedServiceId(service.id);
      setEditingServiceId("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update service");
    } finally {
      setSavingService(false);
    }
  }

  async function handleDeleteService(service: Service) {
    const confirmed = window.confirm(
      `Delete service ${service.displayName || service.name}?`,
    );
    if (!confirmed) {
      return;
    }
    setSavingService(true);
    setError("");
    try {
      await deleteService(service.id);
      await loadCatalog(selectedEnvironmentId);
      setEditingServiceId("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to delete service");
    } finally {
      setSavingService(false);
    }
  }

  function updateRegistrationEndpoint(
    index: number,
    updates: Partial<ReturnType<typeof newRegistrationEndpoint>>,
  ) {
    setRegistrationForm((current) => ({
      ...current,
      endpoints: current.endpoints.map((endpoint, endpointIndex) =>
        endpointIndex === index ? { ...endpoint, ...updates } :
          updates.primary ? { ...endpoint, primary: false } : endpoint,
      ),
    }));
  }

  function addRegistrationEndpoint() {
    setRegistrationForm((current) => ({
      ...current,
      endpoints: [
        ...current.endpoints,
        newEndpoint(current.endpoints.map(endpoint => endpoint.name.trim())),
      ],
    }));
  }

  function removeRegistrationEndpoint(index: number) {
    setRegistrationForm((current) => {
      const endpoints = current.endpoints.filter((_, item) => item !== index);
      if (endpoints.length === 1 && !endpoints[0].primary) {
        endpoints[0] = { ...endpoints[0], primary: true };
      }
      return { ...current, endpoints };
    });
  }

  function setPrimaryRegistrationEndpoint(index: number) {
    setRegistrationForm((current) => ({
      ...current,
      endpoints: current.endpoints.map((endpoint, endpointIndex) => ({
        ...endpoint,
        primary: endpointIndex === index,
      })),
    }));
  }

  function focusRegistrationForm() {
    setShowAddRuntime(true);
    setRegistrationStep(1);
    window.setTimeout(() => {
      document
        .querySelector<HTMLInputElement>("[data-registration-instance-name]")
        ?.focus();
    }, 0);
  }

  function startEditInstance(instance: ServiceInstance) {
    setEditingInstanceId(instance.id);
    setInstanceEditForm({
      address: instance.address,
      description: instance.description ?? "",
      enabled: instance.enabled,
    });
  }

  function startEditEndpoint(endpoint: Endpoint) {
    setAddingEndpointInstanceId("");
    setEditingEndpointId(endpoint.id);
    setEndpointEditForm({
      name: endpoint.name,
      protocol: endpoint.protocol,
      port: endpoint.port,
      path: endpoint.path ?? "",
      enabled: endpoint.enabled,
      primary: endpoint.primary,
    });
  }

  function startAddEndpoint(instance: ServiceInstance) {
    const names = endpoints.filter(endpoint => endpoint.instanceId === instance.id).map(endpoint => endpoint.name);
    setEditingEndpointId("");
    setAddingEndpointInstanceId(instance.id);
    setEndpointEditForm(newEndpoint(names, {
      port: primaryEndpointForInstance(endpoints, instance.id)?.port || 8080,
    }));
  }

  async function handleUpdateInstance(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editingInstanceId) {
      return;
    }
    const current = instances.find(
      (instance) => instance.id === editingInstanceId,
    );
    setSavingRuntimeEdit(true);
    setError("");
    try {
      await updateInstance({
        id: editingInstanceId,
        address: instanceEditForm.address,
        port: current?.port ?? 0,
        description: instanceEditForm.description,
        enabled: instanceEditForm.enabled,
        tags: current?.tags ?? {},
        metadata: current?.metadata ?? {},
      });
      setEditingInstanceId("");
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to update instance",
      );
    } finally {
      setSavingRuntimeEdit(false);
    }
  }

  async function handleDeleteInstance(instance: ServiceInstance) {
    const relatedEndpoints = endpoints.filter(
      (endpoint) => endpoint.instanceId === instance.id,
    ).length;
    const relatedChecks = healthChecks.filter(
      (check) => check.instanceId === instance.id,
    ).length;
    const confirmed = window.confirm(
      `Delete instance ${instance.name}?\n\nThis will also remove:\n- ${pluralize(
        relatedEndpoints,
        "endpoint",
      )}\n- ${pluralize(
        relatedChecks,
        "health check",
      )}\n\nExisting incident/event history will be preserved where appropriate.`,
    );
    if (!confirmed) {
      return;
    }
    setSavingRuntimeEdit(true);
    setError("");
    try {
      await deleteInstance(instance.id);
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete instance",
      );
    } finally {
      setSavingRuntimeEdit(false);
    }
  }

  async function handleUpdateEndpoint(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!editingEndpointId && !addingEndpointInstanceId) {
      return;
    }
    const current = endpoints.find(
      (endpoint) => endpoint.id === editingEndpointId,
    );
    const name = endpointEditForm.name.trim();
    const instanceId = addingEndpointInstanceId || current?.instanceId;
    const validation = validateEndpoint(endpointEditForm, endpoints.filter(endpoint => endpoint.instanceId === instanceId && endpoint.id !== editingEndpointId).map(endpoint => endpoint.name));
    if (Object.keys(validation).length) {
      setError(Object.values(validation)[0] || "Check endpoint details.");
      return;
    }
    setSavingRuntimeEdit(true);
    setError("");
    try {
      if (addingEndpointInstanceId) {
        await createEndpoint({
          instanceId: addingEndpointInstanceId,
          name,
          protocol: endpointEditForm.protocol,
          port: endpointEditForm.port,
          path: endpointEditForm.path,
          enabled: endpointEditForm.enabled,
          primary: endpointEditForm.primary,
        });
      } else {
        await updateEndpoint({
          id: editingEndpointId,
          name,
          protocol: endpointEditForm.protocol,
          port: endpointEditForm.port,
          path: endpointEditForm.path,
          enabled: endpointEditForm.enabled,
          primary: endpointEditForm.primary,
          tags: current?.tags ?? {},
          metadata: current?.metadata ?? {},
        });
      }
      setEditingEndpointId("");
      setAddingEndpointInstanceId("");
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(
        endpointMutationError(err, name),
      );
    } finally {
      setSavingRuntimeEdit(false);
    }
  }

  function startEditHealthCheck(check: HealthCheck) {
    setEditingHealthCheckId(check.id);
  }



  async function handleDeleteHealthCheck(check: HealthCheck) {
    const confirmed = window.confirm(`Delete health check ${check.name}?`);
    if (!confirmed) {
      return;
    }
    setSavingHealthCheck(true);
    setError("");
    try {
      await deleteHealthCheck(check.id);
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete health check",
      );
    } finally {
      setSavingHealthCheck(false);
    }
  }

  async function handleDeleteEndpoint(endpoint: Endpoint) {
    const relatedChecks = healthChecks.filter(
      (check) => check.endpointId === endpoint.id,
    ).length;
    const confirmed = window.confirm(
      `Delete endpoint ${endpoint.name}?\n\nThis will also remove or detach:\n- ${pluralize(
        relatedChecks,
        "health check",
      )}\n\nExisting incident/event history will be preserved where appropriate.`,
    );
    if (!confirmed) {
      return;
    }
    setSavingRuntimeEdit(true);
    setError("");
    try {
      await deleteEndpoint(endpoint.id);
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete endpoint",
      );
    } finally {
      setSavingRuntimeEdit(false);
    }
  }

  async function handleRegisterInstance(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const serviceId = selectedServiceId;
    if (!serviceId) {
      return;
    }
    const endpointErrors = validateEndpointCollection(registrationForm.endpoints);
    if (!registrationForm.environmentId || !registrationForm.instanceName.trim() || !registrationForm.address.trim()) {
      setError("Choose an environment and enter instance details.");
      return;
    }
    if (endpointErrors.some(errors => Object.keys(errors).length)) {
      setError("Check the endpoint fields before adding this instance.");
      return;
    }
    setSavingRegistration(true);
    setError("");
    setRegistrationSuccess("");
    try {
      const registration = await registerRuntime({
        serviceId,
        environmentId: registrationForm.environmentId,
        instance: {
          name: registrationForm.instanceName,
          address: registrationForm.address,
          description: registrationForm.description,
          enabled: true,
        },
        endpoints: registrationForm.endpoints.map((endpoint) => ({
          name: endpoint.name.trim(),
          protocol: endpoint.protocol,
          port: endpoint.port,
          path: endpoint.path,
          enabled: endpoint.enabled,
          primary: endpoint.primary,
        })),
      });

      setRegistrationForm((current) => ({
        ...current,
        instanceName: "",
        address: "",
        description: "",
        endpoints: [newRegistrationEndpoint()],
      }));
      setRegistrationStep(0);
      await loadCatalog(selectedEnvironmentId);
      setSelectedServiceId(serviceId);
      setServiceTab("instances");
      if (!registration.instance) {
        throw new Error("RegisterRuntime returned no instance");
      }
      setRegisteredRuntime({
        name: registration.instance.name,
        address: registration.instance.address,
        instanceId: registration.instance.id,
        endpoints: registration.endpoints ?? [],
      });
      setRegistrationSuccess("Instance added successfully.");
      setAuthMessage(
        `Instance added successfully: ${registration.instance.name}.`,
      );
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to register service",
      );
    } finally {
      setSavingRegistration(false);
    }
  }


  function closeRuntimeDialog() {
    setShowAddRuntime(false);
    setRegisteredRuntime(null);
    setRegistrationSuccess("");
  }

  async function handleDeleteSelectedServices() {
    if (selectedBulkServiceIds.length === 0) {
      return;
    }
    const confirmed = window.confirm(
      `Delete ${pluralize(selectedBulkServiceIds.length, "selected service")}?`,
    );
    if (!confirmed) {
      return;
    }
    setBulkActionRunning(true);
    setError("");
    try {
      await Promise.all(selectedBulkServiceIds.map((id) => deleteService(id)));
      setSelectedBulkServiceIds([]);
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to delete services",
      );
    } finally {
      setBulkActionRunning(false);
    }
  }

  async function handleCopySelectedServiceIds() {
    if (selectedBulkServiceIds.length === 0) {
      return;
    }
    setError("");
    try {
      await navigator.clipboard.writeText(selectedBulkServiceIds.join(","));
      setTestResult("Selected service IDs copied.");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to copy service IDs",
      );
    }
  }

  function toggleBulkService(id: string) {
    setSelectedBulkServiceIds((current) =>
      current.includes(id)
        ? current.filter((item) => item !== id)
        : [...current, id],
    );
  }

  function selectVisibleServices() {
    setSelectedBulkServiceIds((current) =>
      Array.from(
        new Set([...current, ...visibleServices.map((service) => service.id)]),
      ),
    );
  }



  async function handleRunHealthCheck(id: string) {
    if (!id) {
      return;
    }
    setError("");
    try {
      const response = await runHealthCheck(id);
      if (response.state) {
        const state = response.state;
        setHealthStates(current => [...current.filter(item => item.instanceId !== state.instanceId), state]);
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to run health check",
      );
    }
  }


  async function handleResolveIncident(id: string) {
    setResolvingIncidentId(id);
    setError("");
    try {
      await resolveIncident(id);
      await loadOperationalData(selectedEnvironmentId);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to resolve incident",
      );
    } finally {
      setResolvingIncidentId("");
    }
  }

  async function handleCreateChannel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingChannel(true);
    setError("");
    setTestResult("");
    try {
      const configuration =
        channelForm.type === "webhook"
          ? compactMap({ url: channelForm.url })
          : compactMap({
              smtp_host: channelForm.smtpHost,
              smtp_port: channelForm.smtpPort,
              from: channelForm.from,
              to: channelForm.to,
            });
      if (editingChannelId) {
        const current = notificationChannels.find(
          (channel) => channel.id === editingChannelId,
        );
        await updateNotificationChannel({
          id: editingChannelId,
          name: channelForm.name,
          enabled: current?.enabled ?? true,
          description: channelForm.description,
          configuration,
          retryPolicy: current?.retryPolicy ?? {},
          tags: current?.tags ?? {},
        });
      } else {
        await createNotificationChannel({
          type: channelForm.type,
          name: channelForm.name,
          enabled: true,
          description: channelForm.description,
          configuration,
          retryPolicy: {},
          tags: {},
        });
      }
      setChannelForm({
        type: "webhook",
        name: "",
        description: "",
        url: "",
        smtpHost: "",
        smtpPort: "587",
        from: "",
        to: "",
      });
      setEditingChannelId("");
      await loadCatalog(selectedEnvironmentId);
      setActiveView("alerts");
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to create notification channel",
      );
    } finally {
      setSavingChannel(false);
    }
  }

  async function handleCreatePolicy(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingPolicy(true);
    setError("");
    try {
      if (editingPolicyId) {
        const current = alertPolicies.find(
          (policy) => policy.id === editingPolicyId,
        );
        await updateAlertPolicy({
          id: editingPolicyId,
          enabled: current?.enabled ?? true,
          notifyOn: splitCSV(policyForm.notifyOn),
          cooldownMinutes: policyForm.cooldownMinutes,
          sendRecoveryNotification: policyForm.sendRecoveryNotification,
          filters: current?.filters ?? {},
          channelIds: splitCSV(policyForm.channelIds),
        });
      } else {
        await createAlertPolicy({
          deploymentId: policyForm.deploymentId,
          environmentId: policyForm.environmentId || selectedEnvironmentId,
          enabled: true,
          notifyOn: splitCSV(policyForm.notifyOn),
          cooldownMinutes: policyForm.cooldownMinutes,
          sendRecoveryNotification: policyForm.sendRecoveryNotification,
          filters: {},
          channelIds: splitCSV(policyForm.channelIds),
        });
      }
      setPolicyForm({
        environmentId: "",
        deploymentId: "",
        notifyOn: "unhealthy,recovered",
        cooldownMinutes: 15,
        sendRecoveryNotification: true,
        channelIds: "",
      });
      setEditingPolicyId("");
      await loadCatalog(selectedEnvironmentId);
      setActiveView("alerts");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to create policy");
    } finally {
      setSavingPolicy(false);
    }
  }

  async function handleTestChannel(id: string) {
    setTestingChannelId(id);
    setError("");
    setTestResult("");
    try {
      await testNotificationChannel(id);
      setTestResult("Test notification sent.");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to test notification",
      );
    } finally {
      setTestingChannelId("");
    }
  }

  function editPolicy(policy: AlertPolicy) {
    setEditingPolicyId(policy.id);
    setPolicyForm({
      environmentId: policy.environmentId ?? "",
      deploymentId: policy.deploymentId ?? "",
      notifyOn: policy.notifyOn?.join(",") || "unhealthy,recovered",
      cooldownMinutes: policy.cooldownMinutes,
      sendRecoveryNotification: policy.sendRecoveryNotification,
      channelIds: policy.channelIds?.join(",") ?? "",
    });
  }

  function editChannel(channel: NotificationChannel) {
    setEditingChannelId(channel.id);
    setChannelForm({
      type: channel.type,
      name: channel.name,
      description: channel.description ?? "",
      url: channel.configuration?.url ?? "",
      smtpHost: channel.configuration?.smtp_host ?? "",
      smtpPort: channel.configuration?.smtp_port ?? "587",
      from: channel.configuration?.from ?? "",
      to: channel.configuration?.to ?? "",
    });
  }

  function cancelPolicyEdit() {
    setEditingPolicyId("");
    setPolicyForm({
      environmentId: "",
      deploymentId: "",
      notifyOn: "unhealthy,recovered",
      cooldownMinutes: 15,
      sendRecoveryNotification: true,
      channelIds: "",
    });
  }

  function cancelChannelEdit() {
    setEditingChannelId("");
    setChannelForm({
      type: "webhook",
      name: "",
      description: "",
      url: "",
      smtpHost: "",
      smtpPort: "587",
      from: "",
      to: "",
    });
  }

  const currentEnvironmentName =
    environments.find((environment) => environment.id === selectedEnvironmentId)
      ?.name ?? "All environments";
  const openIncidentCount = incidents.filter(
    (incident) => incident.state === "INCIDENT_STATE_OPEN",
  ).length;
  const degradedServiceCount = services.filter(
    (service) => serviceOperationalStatus(service, incidents) === "degraded",
  ).length;
  void savingRuntimeEdit;
  void registrationSuccess;
  void registrationStep;
  void selectedEnvironment;
  void selectedEnvironmentInstances;
  void selectedServiceInstanceById;
  void addRegistrationEndpoint;
  void removeRegistrationEndpoint;
  void setPrimaryRegistrationEndpoint;
  void focusRegistrationForm;
  void RuntimeTopology;
  void startEditInstance;
  void startEditEndpoint;
  void handleUpdateInstance;
  void handleDeleteInstance;
  void handleUpdateEndpoint;
  void handleDeleteEndpoint;
  void formatEndpointSummary;
  void formatEndpointUrl;

  const operationalProps = {
    loading,
    environments,
    services,
    deployments,
    instances,
    incidents,
    filteredIncidents,
    selectedIncident,
    events,
    filteredEvents,
    availability,
    healthChecksCount: healthChecks.length,
    alertPolicies,
    notificationChannels,
    selectedEnvironmentId,
    currentEnvironmentName,
    incidentStateFilter,
    incidentSearch,
    eventTypeFilter,
    eventResourceFilter,
    eventSearch,
    eventTypes,
    eventResourceTypes,
    alertsSection,
    policyForm,
    channelForm,
    editingPolicyId,
    editingChannelId,
    savingPolicy,
    savingChannel,
    testingChannelId,
    resolvingIncidentId,
    onIncidentStateFilter: setIncidentStateFilter,
    onIncidentSearch: setIncidentSearch,
    onEventTypeFilter: setEventTypeFilter,
    onEventResourceFilter: setEventResourceFilter,
    onEventSearch: setEventSearch,
    onAlertsSection: setAlertsSection,
    onEnvironmentSaved: async (environment:Environment,created:boolean) => {
      const scope=created?environment.id:selectedEnvironmentId;
      if(created)setSelectedEnvironmentId(scope);
      await loadCatalog(scope);
    },
    onViewEnvironmentServices: async (environmentId:string) => {
      await handleEnvironmentChange(environmentId);
      navigateTo("services");
    },
    onPolicyForm: setPolicyForm,
    onChannelForm: setChannelForm,
    onCreatePolicy: handleCreatePolicy,
    onCreateChannel: handleCreateChannel,
    onCancelPolicy: cancelPolicyEdit,
    onCancelChannel: cancelChannelEdit,
    onEditPolicy: editPolicy,
    onEditChannel: editChannel,
    onTestChannel: (id: string) => { void handleTestChannel(id); },
    onResolveIncident: (id: string) => { void handleResolveIncident(id); },
    onSelectedIncident: setSelectedIncidentId,
    onViewChange: (view: "environments" | "services" | "incidents" | "alerts" | "events") => navigateTo(view),
    onSelectService: setSelectedServiceId,
  };

  return (
    <AppShell
      activeView={activeView}
      currentEnvironmentName={currentEnvironmentName}
      darkMode={darkMode}
      degradedServiceCount={degradedServiceCount}
      environments={environments}
      openIncidentCount={openIncidentCount}
      selectedEnvironmentId={selectedEnvironmentId}
      onEnvironmentChange={handleEnvironmentChange}
      onRefresh={() => loadCatalog()}
      onLogout={() => { void handleLogout(); }}
      onSecurityOpen={() => {
        navigateTo("security");
        loadSecurityData().catch((err: unknown) =>
          setError(
            err instanceof Error
              ? err.message
              : "Failed to load security data",
          ),
        );
      }}
      onToggleDarkMode={() => setDarkMode((value) => !value)}
      onViewChange={navigateTo}
    >
      {loading ? <div className="status">Loading registry data...</div> : null}

      {error && !healthResultsRoute && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      {!healthResultsRoute && testResult && <div className="success">{testResult}</div>}
      {!healthResultsRoute && authMessage && <div className="success">{authMessage}</div>}

      {activeView === "dashboard" ? (
        <OperationalWorkspace {...operationalProps} view="dashboard" />
      ) : activeView === "services" ? (
        <ServicesWorkspace
          onHealthSaved={()=>loadCatalog(selectedEnvironmentId)}
          onHealthResults={navigateHealthResults}
          onHealthChecks={(serviceId) => {
            setActiveView("health");
            setHealthResultsRoute(false);
            window.history.pushState({}, "", `/health/checks?serviceId=${encodeURIComponent(serviceId)}`);
          }}
          error={error}
          loading={loading}
          environments={environments}
          services={services}
          filteredServices={filteredServices}
          visibleServices={visibleServices}
          deployments={deployments}
          instances={instances}
          endpoints={endpoints}
          healthChecks={healthChecks}
          healthStates={healthStates}
          incidents={incidents}
          events={events}
          availability={serviceAvailability}
          selectedEnvironmentId={selectedEnvironmentId}
          currentEnvironmentName={currentEnvironmentName}
          selectedServiceId={selectedServiceId}
          onSelectService={setSelectedServiceId}
          serviceSearch={serviceSearch}
          setServiceSearch={setServiceSearch}
          serviceHealthFilter={serviceHealthFilter}
          setServiceHealthFilter={setServiceHealthFilter}
          serviceTagFilter={serviceTagFilter}
          setServiceTagFilter={setServiceTagFilter}
          servicePage={servicePage}
          servicePageCount={servicePageCount}
          setServicePage={setServicePage}
          selectedBulkServiceIds={selectedBulkServiceIds}
          toggleBulkService={toggleBulkService}
          selectVisibleServices={selectVisibleServices}
          clearBulkSelection={() => setSelectedBulkServiceIds([])}
          handleCopySelectedServiceIds={handleCopySelectedServiceIds}
          handleDeleteSelectedServices={handleDeleteSelectedServices}
          bulkActionRunning={bulkActionRunning}
          serviceTab={serviceTab}
          setServiceTab={(value) => setServiceTab(value as typeof serviceTab)}
          serviceForm={serviceForm}
          setServiceForm={setServiceForm}
          showCreateService={showCreateService}
          setShowCreateService={setShowCreateService}
          handleCreateService={handleCreateService}
          editingServiceId={editingServiceId}
          setEditingServiceId={setEditingServiceId}
          serviceEditForm={serviceEditForm}
          setServiceEditForm={setServiceEditForm}
          handleUpdateService={handleUpdateService}
          handleDeleteService={handleDeleteService}
          savingService={savingService}
          showAddRuntime={showAddRuntime}
          setShowAddRuntime={setShowAddRuntime}
          registrationForm={registrationForm}
          setRegistrationForm={setRegistrationForm}
          registrationSuccess={registrationSuccess}
          savingRegistration={savingRegistration}
          addRegistrationEndpoint={addRegistrationEndpoint}
          removeRegistrationEndpoint={removeRegistrationEndpoint}
          setPrimaryRegistrationEndpoint={setPrimaryRegistrationEndpoint}
          updateRegistrationEndpoint={updateRegistrationEndpoint}
          handleRegisterInstance={handleRegisterInstance}
          closeRuntimeDialog={closeRuntimeDialog}
          registeredRuntime={registeredRuntime ?? undefined}
          editingInstanceId={editingInstanceId}
          setEditingInstanceId={setEditingInstanceId}
          instanceEditForm={instanceEditForm}
          setInstanceEditForm={setInstanceEditForm}
          handleUpdateInstance={handleUpdateInstance}
          handleDeleteInstance={handleDeleteInstance}
          savingRuntimeEdit={savingRuntimeEdit}
          editingEndpointId={editingEndpointId}
          addingEndpointInstanceId={addingEndpointInstanceId}
          startAddEndpoint={startAddEndpoint}
          setEditingEndpointId={setEditingEndpointId}
          setAddingEndpointInstanceId={setAddingEndpointInstanceId}
          endpointEditForm={endpointEditForm}
          setEndpointEditForm={setEndpointEditForm}
          handleUpdateEndpoint={handleUpdateEndpoint}
          handleDeleteEndpoint={handleDeleteEndpoint}
          editingHealthCheckId={editingHealthCheckId}
          setEditingHealthCheckId={setEditingHealthCheckId}
          healthTargetInstanceId={healthTargetInstanceId}
          setHealthTargetInstanceId={setHealthTargetInstanceId}
          handleDeleteHealthCheck={handleDeleteHealthCheck}
          savingHealthCheck={savingHealthCheck}
          handleRunHealthCheck={handleRunHealthCheck}
          startEditHealthCheck={startEditHealthCheck}
          testResult={testResult}
          setError={setError}
        />
      ) : activeView === "environments" ? (
        <OperationalWorkspace {...operationalProps} view="environments" />
      ) : activeView === "health" ? (
        healthResultsRoute ? <HealthResultsPage services={services} environments={environments} instances={instances} endpoints={endpoints} checks={healthChecks} deployments={deployments} onService={(serviceId, environmentId)=>void navigateServiceHealth(serviceId, environmentId)} onInstance={(instanceId, serviceId, environmentId)=>navigateHealthResults(serviceId, undefined, instanceId, environmentId)} onBack={()=>{
          const serviceId=new URLSearchParams(window.location.search).get("serviceId");
          if(serviceId){setSelectedServiceId(serviceId);setServiceTab("health");setHealthResultsRoute(false);setActiveView("services");window.history.pushState({},"",`/services/${encodeURIComponent(serviceId)}/health`)}
          else navigateTo("health");
        }}/> : <HealthWorkspace services={services} environments={environments} deployments={deployments} endpoints={endpoints} error={error}
          instances={instances} healthChecks={healthChecks} healthStates={healthStates} selectedEnvironmentId={selectedEnvironmentId}
          healthStatusFilter={healthStatusFilter} setHealthStatusFilter={setHealthStatusFilter}
          onRun={handleRunHealthCheck} onDelete={handleDeleteHealthCheck} onSaved={()=>loadCatalog(selectedEnvironmentId)}
          loading={loading}
          onServiceHealth={(serviceId,environmentId)=>void navigateServiceHealth(serviceId,environmentId)}
          onInstanceResults={(instanceId,serviceId,environmentId)=>navigateHealthResults(serviceId,undefined,instanceId,environmentId)}
          onResults={checkId=>navigateHealthResults(undefined,checkId)}/>

      ) : activeView === "incidents" ? (
        <OperationalWorkspace {...operationalProps} view="incidents" />
      ) : false ? (
        <section className="content-grid incidents-grid">
          <PageHeader
            title="Incidents"
            context={`${filteredIncidents.length} incidents · ${openIncidentCount} open`}
            description="Triage active breakage, inspect impact, and resolve incidents without losing service context."
          />
          <div className="panel">
            <div className="panel-heading">
              <h2>Availability</h2>
              <span>{selectedEnvironmentId ? "Filtered" : "Global"}</span>
            </div>
            <div className="availability-grid">
              <AvailabilityCard
                label="24h"
                summary={availability.availability24h}
              />
              <AvailabilityCard
                label="7d"
                summary={availability.availability7d}
              />
              <AvailabilityCard
                label="30d"
                summary={availability.availability30d}
              />
            </div>
          </div>

          <div className="panel">
            <div className="panel-heading">
              <h2>Incidents</h2>
              <span>{filteredIncidents.length} shown</span>
            </div>
            <div className="metrics-grid compact-metrics">
              <MetricCard
                label="Open"
                value={
                  incidents.filter(
                    (incident) => incident.state === "INCIDENT_STATE_OPEN",
                  ).length
                }
                detail="active incidents"
              />
              <MetricCard
                label="Resolved"
                value={
                  incidents.filter(
                    (incident) => incident.state === "INCIDENT_STATE_RESOLVED",
                  ).length
                }
                detail="history"
              />
            </div>
            <div className="filter-bar">
              <label>
                State
                <select
                  value={incidentStateFilter}
                  onChange={(event) =>
                    setIncidentStateFilter(event.target.value)
                  }
                >
                  <option value="all">All incidents</option>
                  <option value="open">Open</option>
                  <option value="resolved">Resolved</option>
                </select>
              </label>
              <label>
                Search
                <input
                  value={incidentSearch}
                  onChange={(event) => setIncidentSearch(event.target.value)}
                />
              </label>
            </div>
            <div className="table">
              {filteredIncidents.length === 0 ? (
                <EmptyState
                  title="No incidents match"
                  description="No incidents match the current state and search filters."
                />
              ) : (
                filteredIncidents.map((incident) => (
                  <div className="incident-row" key={incident.id}>
                    <div>
                      <StatusBadge
                        status={formatIncidentState(incident.state)}
                      />
                      <span>{incident.reason || "No reason recorded"}</span>
                    </div>
                    <ResourceLink
                      onClick={() => {
                        setSelectedServiceId(incident.serviceId);
                        setActiveView("services");
                      }}
                    >
                      {incident.serviceId || incident.instanceId}
                    </ResourceLink>
                    <span>{formatTimestamp(incident.openedAt)}</span>
                    <span>
                      {incident.resolvedAt
                        ? formatDuration(incident.durationSeconds)
                        : "Open"}
                    </span>
                    <button
                      disabled={
                        incident.state !== "INCIDENT_STATE_OPEN" ||
                        resolvingIncidentId === incident.id
                      }
                      onClick={() => handleResolveIncident(incident.id)}
                      type="button"
                    >
                      {resolvingIncidentId === incident.id
                        ? "Resolving"
                        : "Resolve"}
                    </button>
                    <button
                      onClick={() => setSelectedIncidentId(incident.id)}
                      type="button"
                    >
                      Details
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
          {selectedIncident ? (
            <div aria-modal="true" className="modal-backdrop" role="dialog">
              <div className="modal">
                <div className="panel-heading">
                  <h2>Incident Detail</h2>
                  <IconButton
                    label="Close dialog"
                    onClick={() => setSelectedIncidentId("")}
                    type="button"
                  >
                    x
                  </IconButton>
                </div>
                <dl className="detail-list">
                  <dt>ID</dt>
                  <dd>{selectedIncident!.id}</dd>
                  <dt>State</dt>
                  <dd>
                    <StatusBadge
                      status={formatIncidentState(selectedIncident!.state)}
                    />
                  </dd>
                  <dt>Reason</dt>
                  <dd>{selectedIncident!.reason || "None"}</dd>
                  <dt>Impact</dt>
                  <dd>{selectedIncident!.impactSummary || "None"}</dd>
                  <dt>Service</dt>
                  <dd>
                    <ResourceLink
                      onClick={() => {
                        setSelectedServiceId(selectedIncident!.serviceId);
                        setSelectedIncidentId("");
                        setActiveView("services");
                      }}
                    >
                      {selectedIncident!.serviceId}
                    </ResourceLink>
                  </dd>
                  <dt>Instance</dt>
                  <dd>{selectedIncident!.instanceId}</dd>
                  <dt>Opened</dt>
                  <dd>{formatTimestamp(selectedIncident!.openedAt)}</dd>
                  <dt>Resolved</dt>
                  <dd>{formatTimestamp(selectedIncident!.resolvedAt)}</dd>
                  <dt>Duration</dt>
                  <dd>{formatDuration(selectedIncident!.durationSeconds)}</dd>
                  <dt>Metadata</dt>
                  <dd>{formatMap(selectedIncident!.metadata)}</dd>
                </dl>
              </div>
            </div>
          ) : null}
        </section>
      ) : activeView === "alerts" ? (
        <OperationalWorkspace {...operationalProps} view="alerts" />
      ) : false ? (
        <section className="content-grid alerts-grid">
          <PageHeader
            title="Alerts"
            context={`${alertPolicies.length} policies · ${notificationChannels.length} channels`}
            description="Keep notification channels separate from policies so routing and triggers stay clear."
          />
          <nav className="subnav" aria-label="Alert views">
            <button className={alertsSection === "policies" ? "active" : ""} type="button" onClick={() => setAlertsSection("policies")}>Alert policies</button>
            <button className={alertsSection === "channels" ? "active" : ""} type="button" onClick={() => setAlertsSection("channels")}>Notification channels</button>
          </nav>
          {alertsSection === "policies" ? (
          <div className="panel">
            <div className="panel-heading">
              <h2>Alert Policies</h2>
              <span>{alertPolicies.length} total</span>
            </div>
            <div className="table">
              {alertPolicies.length === 0 ? (
                <EmptyState
                  title="No alert policies"
                  description="Create a policy to route health transitions or incident changes to a notification channel."
                />
              ) : (
                alertPolicies.map((policy) => (
                  <div className="policy-row" key={policy.id}>
                    <div>
                      <StatusBadge
                        status={policy.enabled ? "enabled" : "disabled"}
                      />
                      <span>{formatPolicyScope(policy)}</span>
                    </div>
                    <span>{policy.notifyOn?.join(", ") || "No triggers"}</span>
                    <span>{policy.cooldownMinutes}m cooldown</span>
                    <span>{policy.channelIds?.length ?? 0} channels</span>
                    <button onClick={() => editPolicy(policy)} type="button">
                      Edit
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>

          ) : (
          <div className="panel">
            <div className="panel-heading">
              <h2>Notification Channels</h2>
              <span>{notificationChannels.length} total</span>
            </div>
            <div className="table">
              {notificationChannels.length === 0 ? (
                <EmptyState
                  title="No notification channels"
                  description="Add a webhook or email destination before assigning policies."
                />
              ) : (
                notificationChannels.map((channel) => (
                  <div className="channel-row" key={channel.id}>
                    <div>
                      <strong>{channel.name}</strong>
                      <span>{channel.type}</span>
                    </div>
                    <span>
                      {channel.description || "Description unavailable"}
                    </span>
                    <StatusBadge
                      status={channel.enabled ? "enabled" : "disabled"}
                    />
                    <button
                      disabled={
                        !channel.enabled || testingChannelId === channel.id
                      }
                      onClick={() => handleTestChannel(channel.id)}
                      type="button"
                    >
                      {testingChannelId === channel.id ? "Testing" : "Test"}
                    </button>
                    <button onClick={() => editChannel(channel)} type="button">
                      Edit
                    </button>
                  </div>
                ))
              )}
            </div>
          </div>
          )}

          <details className="workflow-disclosure">
            <summary>{editingPolicyId ? "Edit alert policy" : "Create alert policy"}</summary>
          <form className="panel form-panel" onSubmit={handleCreatePolicy}>
            <div className="panel-heading">
              <h2>
                {editingPolicyId ? "Edit Alert Policy" : "Create Alert Policy"}
              </h2>
              {editingPolicyId ? (
                <button onClick={cancelPolicyEdit} type="button">
                  Cancel
                </button>
              ) : null}
            </div>
            <label>
              Environment
              <select
                value={policyForm.environmentId || selectedEnvironmentId}
                onChange={(event) =>
                  setPolicyForm((current) => ({
                    ...current,
                    environmentId: event.target.value,
                  }))
                }
              >
                <option value="">Select environment</option>
                {environments.map((environment) => (
                  <option key={environment.id} value={environment.id}>
                    {environment.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Environment scope reference
              <input
                value={policyForm.deploymentId}
                onChange={(event) =>
                  setPolicyForm((current) => ({
                    ...current,
                    deploymentId: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Notify on
              <input
                required
                value={policyForm.notifyOn}
                onChange={(event) =>
                  setPolicyForm((current) => ({
                    ...current,
                    notifyOn: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Channel IDs
              <input
                required
                value={policyForm.channelIds}
                onChange={(event) =>
                  setPolicyForm((current) => ({
                    ...current,
                    channelIds: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Cooldown minutes
              <input
                min="0"
                type="number"
                value={policyForm.cooldownMinutes}
                onChange={(event) =>
                  setPolicyForm((current) => ({
                    ...current,
                    cooldownMinutes: Number(event.target.value),
                  }))
                }
              />
            </label>
            <label className="checkbox-label">
              <input
                checked={policyForm.sendRecoveryNotification}
                type="checkbox"
                onChange={(event) =>
                  setPolicyForm((current) => ({
                    ...current,
                    sendRecoveryNotification: event.target.checked,
                  }))
                }
              />
              Send recovery notification
            </label>
            <button disabled={savingPolicy} type="submit">
              {savingPolicy
                ? "Saving"
                : editingPolicyId
                  ? "Save policy"
                  : "Create policy"}
            </button>
          </form>
          </details>

          <details className="workflow-disclosure">
            <summary>{editingChannelId ? "Edit notification channel" : "Create notification channel"}</summary>
          <form className="panel form-panel" onSubmit={handleCreateChannel}>
            <div className="panel-heading">
              <h2>
                {editingChannelId
                  ? "Edit Notification Channel"
                  : "Create Notification Channel"}
              </h2>
              {editingChannelId ? (
                <button onClick={cancelChannelEdit} type="button">
                  Cancel
                </button>
              ) : null}
            </div>
            <label>
              Type
              <select
                value={channelForm.type}
                onChange={(event) =>
                  setChannelForm((current) => ({
                    ...current,
                    type: event.target.value,
                  }))
                }
              >
                <option value="webhook">Webhook</option>
                <option value="email">Email</option>
              </select>
            </label>
            <label>
              Name
              <input
                required
                value={channelForm.name}
                onChange={(event) =>
                  setChannelForm((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Description
              <textarea
                value={channelForm.description}
                onChange={(event) =>
                  setChannelForm((current) => ({
                    ...current,
                    description: event.target.value,
                  }))
                }
              />
            </label>
            {channelForm.type === "webhook" ? (
              <label>
                Webhook URL
                <input
                  required
                  value={channelForm.url}
                  onChange={(event) =>
                    setChannelForm((current) => ({
                      ...current,
                      url: event.target.value,
                    }))
                  }
                />
              </label>
            ) : (
              <div className="form-fields-grid">
                <label>
                  SMTP host
                  <input
                    required
                    value={channelForm.smtpHost}
                    onChange={(event) =>
                      setChannelForm((current) => ({
                        ...current,
                        smtpHost: event.target.value,
                      }))
                    }
                  />
                </label>
                <label>
                  SMTP port
                  <input
                    required
                    value={channelForm.smtpPort}
                    onChange={(event) =>
                      setChannelForm((current) => ({
                        ...current,
                        smtpPort: event.target.value,
                      }))
                    }
                  />
                </label>
                <label>
                  From
                  <input
                    required
                    value={channelForm.from}
                    onChange={(event) =>
                      setChannelForm((current) => ({
                        ...current,
                        from: event.target.value,
                      }))
                    }
                  />
                </label>
                <label>
                  To
                  <input
                    required
                    value={channelForm.to}
                    onChange={(event) =>
                      setChannelForm((current) => ({
                        ...current,
                        to: event.target.value,
                      }))
                    }
                  />
                </label>
              </div>
            )}
            <button disabled={savingChannel} type="submit">
              {savingChannel
                ? "Saving"
                : editingChannelId
                  ? "Save channel"
                  : "Create channel"}
            </button>
          </form>
          </details>
        </section>
      ) : activeView === "security" ? (
        <section className="content-grid alerts-grid">
          <PageHeader
            title="Security"
            context="Administrative controls"
            description="Manage users, application credentials, API tokens, and session access without exposing secrets in normal lists."
          />
          <nav className="subnav" aria-label="Security views">
            <button className={securitySection === "users" ? "active" : ""} type="button" onClick={() => setSecuritySection("users")}>Users</button>
            <button className={securitySection === "tokens" ? "active" : ""} type="button" onClick={() => setSecuritySection("tokens")}>API tokens</button>
            <button className={securitySection === "applicationKeys" ? "active" : ""} type="button" onClick={() => { setSecuritySection("applicationKeys"); listApplicationKeys().then(setApplicationKeys).catch((err: unknown) => setError(err instanceof Error ? err.message : "Failed to load application keys")); }}>Application keys</button>
            <button className={securitySection === "sessions" ? "active" : ""} type="button" onClick={() => { setSecuritySection("sessions"); loadSessionData().catch((err: unknown) => setError(err instanceof Error ? err.message : "Failed to load sessions")); }}>Sessions</button>
          </nav>
          <section className="settings-page" hidden={securitySection !== "sessions"}>
            <div className="settings-page-header"><div><h2>Sessions</h2><p>Review active browser sessions and revoke access you no longer recognize.</p></div><button type="button" onClick={() => loadSessionData().catch((err: unknown) => setError(err instanceof Error ? err.message : "Failed to load sessions"))}>Refresh sessions</button></div>
            <div className="application-key-table session-table" role="table" aria-label="Sessions">
              <div className="application-key-table-header" role="row"><span>User</span><span>Client</span><span>Last active</span><span>Created</span><span>Expires</span><span>Status</span><span>Actions</span></div>
              {sessions.length === 0 ? <EmptyState title="No active sessions" description="Authenticated browser sessions will appear here when they are available." /> : sessions.map((session) => <div className="application-key-table-row" key={session.id} role="row"><strong>{session.userId}</strong><span>{session.userAgent || "Unknown client"}</span><span>{formatTimestamp(session.lastActivityAt)}</span><span>{formatTimestamp(session.createdAt)}</span><span>{formatTimestamp(session.expiresAt)}</span><StatusBadge status={session.revokedAt ? "revoked" : "active"} label={session.revokedAt ? "Revoked" : "Active"} />{!session.revokedAt ? <button className="text-action danger-action" type="button" onClick={() => revokeSession(session.id).then(loadSessionData).catch((err: unknown) => setError(err instanceof Error ? err.message : "Failed to revoke session"))}>Revoke</button> : <span>—</span>}</div>)}
            </div>
          </section>
          <section className="settings-page" hidden={securitySection !== "users"}>
            <div className="settings-page-header"><div><h2>Users</h2><p>Manage the people and roles that can access this Alauda instance.</p></div><button type="button" onClick={() => setShowCreateUser(true)}>Create user</button></div>
            <div className="application-key-table" role="table" aria-label="Users"><div className="application-key-table-header" role="row"><span>Username</span><span>Display name</span><span>Email</span><span>Role</span><span>Created</span><span>Status</span><span>Actions</span></div>{users.length === 0 ? <EmptyState title="No users yet" description="Create an account to grant access to the Alauda administration interface." action={<button type="button" onClick={() => setShowCreateUser(true)}>Create user</button>} /> : users.map((user) => <div className="application-key-table-row" key={user.id} role="row"><strong>{user.username}</strong><span>{user.displayName || "—"}</span><span>{user.email}</span><span>{user.role}</span><span>{user.createdAt ? new Date(user.createdAt).toLocaleDateString() : "Not available"}</span><StatusBadge status={user.enabled ? "enabled" : "disabled"} label={user.enabled ? "Active" : "Disabled"} /><span>—</span></div>)}</div>
          </section>
          {false && <form className="panel form-panel" onSubmit={handleCreateUser}>
            <div className="panel-heading">
              <h2>Users</h2>
              <button
                type="button"
                onClick={() =>
                  loadSecurityData().catch((err: unknown) =>
                    setError(
                      err instanceof Error
                        ? err.message
                        : "Failed to load users",
                    ),
                  )
                }
              >
                Refresh
              </button>
            </div>
            <div className="table compact-table">
              {users.length === 0 ? (
                <EmptyState
                  title="No users loaded"
                  description="Refresh security data or sign in with an account that can list users."
                />
              ) : (
                users.map((user) => (
                  <div className="table-row" key={user.id}>
                    <strong>{user.username}</strong>
                    <span>{user.role}</span>
                    <span>{user.email}</span>
                    <StatusBadge
                      status={user.enabled ? "enabled" : "disabled"}
                    />
                  </div>
                ))
              )}
            </div>
            <label>
              Username
              <input
                required
                value={userForm.username}
                onChange={(event) =>
                  setUserForm((current) => ({
                    ...current,
                    username: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Email
              <input
                required
                type="email"
                value={userForm.email}
                onChange={(event) =>
                  setUserForm((current) => ({
                    ...current,
                    email: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Display name
              <input
                required
                value={userForm.displayName}
                onChange={(event) =>
                  setUserForm((current) => ({
                    ...current,
                    displayName: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Password
              <input
                required
                type="password"
                value={userForm.password}
                onChange={(event) =>
                  setUserForm((current) => ({
                    ...current,
                    password: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Role
              <select
                value={userForm.role}
                onChange={(event) =>
                  setUserForm((current) => ({
                    ...current,
                    role: event.target.value,
                  }))
                }
              >
                <option value="Administrator">Administrator</option>
                <option value="Operator">Operator</option>
                <option value="Viewer">Viewer</option>
                <option value="Automation">Automation</option>
              </select>
            </label>
            <button type="submit">Create user</button>
          </form>}
          {showCreateUser ? <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setShowCreateUser(false); }}><div className="modal form-panel application-key-modal" role="dialog" aria-modal="true" aria-labelledby="create-user-title"><form onSubmit={async (event) => { await handleCreateUser(event); setShowCreateUser(false); }}><div className="modal-header"><div><h2 id="create-user-title">Create user</h2><span>Give a person access to the Alauda administration interface.</span></div><button className="text-action" type="button" onClick={() => setShowCreateUser(false)}>Close</button></div><div className="modal-body application-key-form-body"><label>Username<input required autoFocus value={userForm.username} onChange={(event) => setUserForm((current) => ({ ...current, username: event.target.value }))} /></label><label>Email<input required type="email" value={userForm.email} onChange={(event) => setUserForm((current) => ({ ...current, email: event.target.value }))} /></label><label>Display name<input required value={userForm.displayName} onChange={(event) => setUserForm((current) => ({ ...current, displayName: event.target.value }))} /></label><label>Password<input required type="password" value={userForm.password} onChange={(event) => setUserForm((current) => ({ ...current, password: event.target.value }))} /></label><label>Role<select value={userForm.role} onChange={(event) => setUserForm((current) => ({ ...current, role: event.target.value }))}><option value="Administrator">Administrator</option><option value="Operator">Operator</option><option value="Viewer">Viewer</option><option value="Automation">Automation</option></select></label></div><div className="dialog-actions"><button className="text-action" type="button" onClick={() => setShowCreateUser(false)}>Cancel</button><button type="submit">Create user</button></div></form></div></div> : null}
          <section className="settings-page" hidden={securitySection !== "tokens"}>
            <div className="settings-page-header"><div><h2>API tokens</h2><p>Manage user-scoped credentials for scripts and automation without exposing secrets in the list.</p></div><button type="button" onClick={() => setShowCreateApiToken(true)}>Create API token</button></div>
            <div className="application-key-table" role="table" aria-label="API tokens"><div className="application-key-table-header" role="row"><span>Name</span><span>User</span><span>Scopes</span><span>Created</span><span>Expires</span><span>Status</span><span>Actions</span></div>{apiTokens.length === 0 ? <EmptyState title="No API tokens yet" description="Create a scoped token for a user or automation workflow." action={<button type="button" onClick={() => setShowCreateApiToken(true)}>Create API token</button>} /> : apiTokens.map((token) => <div className="application-key-table-row" key={token.id} role="row"><strong>{token.name}</strong><span>{users.find((user) => user.id === token.userId)?.username ?? token.userId}</span><span>{token.scopes.join(", ")}</span><span>{token.createdAt ? new Date(token.createdAt).toLocaleDateString() : "Not available"}</span><span>{token.expiresAt ? new Date(token.expiresAt).toLocaleDateString() : "Never"}</span><StatusBadge status={token.enabled ? "enabled" : "disabled"} label={token.enabled ? "Active" : "Revoked"} /><button className="text-action danger-action" type="button" onClick={() => handleRevokeApiToken(token.id)}>Revoke</button></div>)}</div>
          </section>
          {false && <form className="panel form-panel" onSubmit={handleCreateApiToken}>
            <div className="panel-heading">
              <h2>API Tokens</h2>
              <span>{apiTokens.length} loaded</span>
            </div>
            <label>
              User
              <select
                required
                value={tokenForm.userId}
                onChange={(event) => {
                  const userId = event.target.value;
                  setTokenForm((current) => ({ ...current, userId }));
                  loadSecurityData(userId).catch((err: unknown) =>
                    setError(
                      err instanceof Error
                        ? err.message
                        : "Failed to load tokens",
                    ),
                  );
                }}
              >
                <option value="">Select user</option>
                {users.map((user) => (
                  <option key={user.id} value={user.id}>
                    {user.username}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Token name
              <input
                required
                value={tokenForm.name}
                onChange={(event) =>
                  setTokenForm((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Scopes
              <input
                value={tokenForm.scopes}
                onChange={(event) =>
                  setTokenForm((current) => ({
                    ...current,
                    scopes: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Expires in hours
              <input
                type="number"
                min="1"
                value={tokenForm.expiresInHours}
                onChange={(event) =>
                  setTokenForm((current) => ({
                    ...current,
                    expiresInHours: event.target.value,
                  }))
                }
              />
            </label>
            <button type="submit">Create token</button>
            <div className="table compact-table">
              {apiTokens.length === 0 ? (
                <EmptyState
                  title="No API tokens loaded"
                  description="Select a user to inspect existing tokens or create a new scoped token."
                />
              ) : (
                apiTokens.map((token) => (
                  <div className="table-row" key={token.id}>
                    <strong>{token.name}</strong>
                    <span>{token.scopes.join(", ")}</span>
                    <StatusBadge
                      status={token.enabled ? "enabled" : "disabled"}
                      label={token.enabled ? "Enabled" : "Revoked"}
                    />
                    <button
                      type="button"
                      onClick={() => handleRevokeApiToken(token.id)}
                    >
                      Revoke
                    </button>
                  </div>
                ))
              )}
            </div>
          </form>}
          {showCreateApiToken ? <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) setShowCreateApiToken(false); }}><div className="modal form-panel application-key-modal" role="dialog" aria-modal="true" aria-labelledby="create-api-token-title"><form onSubmit={async (event) => { await handleCreateApiToken(event); setShowCreateApiToken(false); }}><div className="modal-header"><div><h2 id="create-api-token-title">Create API token</h2><span>Create a scoped credential for a user or automation workflow.</span></div><button className="text-action" type="button" onClick={() => setShowCreateApiToken(false)}>Close</button></div><div className="modal-body application-key-form-body"><label>User<select required autoFocus value={tokenForm.userId} onChange={(event) => setTokenForm((current) => ({ ...current, userId: event.target.value }))}><option value="">Select user</option>{users.map((user) => <option key={user.id} value={user.id}>{user.username}</option>)}</select></label><label>Token name<input required value={tokenForm.name} onChange={(event) => setTokenForm((current) => ({ ...current, name: event.target.value }))} /></label><fieldset><legend>Scopes</legend><span className="field-help">Use the minimum permissions required by the workflow.</span><div className="choice-grid">{["read", "write", "admin"].map((scope) => { const selected = tokenForm.scopes.split(",").includes(scope); return <label className="choice-row" key={scope}><input type="checkbox" checked={selected} onChange={() => setTokenForm((current) => { const scopes = current.scopes.split(",").filter(Boolean); return { ...current, scopes: selected ? scopes.filter((value) => value !== scope).join(",") : [...scopes, scope].join(",") }; })} /><span><strong>{scope}</strong></span></label>; })}</div></fieldset><label>Expiration<select value={tokenForm.expiresInHours} onChange={(event) => setTokenForm((current) => ({ ...current, expiresInHours: event.target.value }))}><option value="168">7 days</option><option value="720">30 days</option><option value="2160">90 days</option><option value="">Never</option></select></label></div><div className="dialog-actions"><button className="text-action" type="button" onClick={() => setShowCreateApiToken(false)}>Cancel</button><button type="submit">Create API token</button></div></form></div></div> : null}
          <section className="settings-page" hidden={securitySection !== "applicationKeys"}>
            <div className="settings-page-header">
              <div>
                <h2>Application keys</h2>
                <p>Manage credentials used by external services and automation to access Alauda.</p>
              </div>
              <button type="button" onClick={() => { setApplicationKeySecret(""); setShowCreateApplicationKey(true); }}>Create application key</button>
            </div>
            <div className="application-key-table" role="table" aria-label="Application keys">
              <div className="application-key-table-header" role="row">
                <span>Name</span><span>Scopes</span><span>Environments</span><span>Created</span><span>Expires</span><span>Status</span><span>Actions</span>
              </div>
              {applicationKeys.length === 0 ? (
                <EmptyState title="No application keys yet" description="Application keys are intended for machine-to-machine authentication, automation, and external service clients." action={<button type="button" onClick={() => setShowCreateApplicationKey(true)}>Create application key</button>} />
              ) : applicationKeys.map((key) => {
                const keyEnvironments = key.environmentIds?.length
                  ? key.environmentIds.map((id) => environments.find((environment) => environment.id === id)?.name ?? id).join(", ")
                  : "All environments";
                const expired = Boolean(key.expiresAt && new Date(key.expiresAt).getTime() < Date.now());
                return <div className="application-key-table-row" key={key.id} role="row">
                  <strong>{key.name}</strong>
                  <span>{key.scopes.join(", ")}</span>
                  <span>{keyEnvironments}</span>
                  <span>{key.createdAt ? new Date(key.createdAt).toLocaleDateString() : "Not available"}</span>
                  <span>{key.expiresAt ? new Date(key.expiresAt).toLocaleDateString() : "Never"}</span>
                  <StatusBadge status={expired ? "warning" : key.enabled ? "enabled" : "disabled"} label={expired ? "Expired" : key.enabled ? "Active" : "Revoked"} />
                  {key.enabled ? <button className="text-action danger-action" type="button" onClick={() => handleRevokeApplicationKey(key.id)}>Revoke</button> : <span>—</span>}
                </div>;
              })}
            </div>
          </section>
          {showCreateApplicationKey ? <div className="modal-backdrop" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) closeApplicationKeyFlow(); }}>
            <div className="modal form-panel application-key-modal" role="dialog" aria-modal="true" aria-labelledby="create-application-key-title">
              {applicationKeySecret ? <div className="application-key-secret-step">
                <div className="modal-header"><div><h2 id="create-application-key-title">Application key created</h2><span>Copy this key now. It will not be shown again.</span></div></div>
                <div className="secret-callout"><code>{applicationKeySecret}</code><button type="button" onClick={() => void copyApplicationKey()}>Copy</button></div>
                <div className="dialog-actions"><button type="button" onClick={closeApplicationKeyFlow}>Done</button></div>
              </div> : <form onSubmit={handleCreateApplicationKey}>
                <div className="modal-header"><div><h2 id="create-application-key-title">Create application key</h2><span>Create a machine credential for external services and automation.</span></div><IconButton label="Close dialog" type="button" onClick={closeApplicationKeyFlow}>x</IconButton></div>
                <div className="modal-body application-key-form-body">
                  <label>Name<input required autoFocus value={applicationKeyForm.name} onChange={(event) => setApplicationKeyForm((current) => ({ ...current, name: event.target.value }))} placeholder="payments-production" /><span className="field-help">Choose a name that identifies the consuming service.</span></label>
                  <fieldset><legend>Scopes</legend><span className="field-help">Select the permissions this credential needs.</span><div className="choice-grid">{["read", "write", "admin"].map((scope) => { const selected = applicationKeyForm.scopes.split(",").includes(scope); return <label className="choice-row" key={scope}><input type="checkbox" checked={selected} onChange={() => setApplicationKeyForm((current) => { const scopes = current.scopes.split(",").filter(Boolean); return { ...current, scopes: selected ? scopes.filter((value) => value !== scope).join(",") : [...scopes, scope].join(",") }; })} /><span><strong>{scope}</strong><small>{scope === "read" ? "View registry resources" : scope === "write" ? "Change registry resources" : "Administrative access"}</small></span></label>; })}</div></fieldset>
                  <fieldset><legend>Environments</legend><span className="field-help">Limit this key to selected environments, or allow all environments.</span><label className="choice-row environment-all-choice"><input type="checkbox" checked={!applicationKeyForm.environmentIds} onChange={(event) => setApplicationKeyForm((current) => ({ ...current, environmentIds: event.target.checked ? "" : environments[0]?.id ?? "" }))} /><span><strong>All environments</strong><small>Allow access across the registry</small></span></label><div className="choice-grid environment-choices">{environments.map((environment) => { const selected = applicationKeyForm.environmentIds.split(",").filter(Boolean).includes(environment.id); return <label className="choice-row" key={environment.id}><input type="checkbox" disabled={!applicationKeyForm.environmentIds} checked={selected} onChange={() => setApplicationKeyForm((current) => { const ids = current.environmentIds.split(",").filter(Boolean); const next = selected ? ids.filter((id) => id !== environment.id) : [...ids, environment.id]; return { ...current, environmentIds: next.join(",") }; })} /><span><strong>{environment.name}</strong><small>{environment.key}</small></span></label>; })}</div></fieldset>
                  <label>Expiration<select value={applicationKeyForm.expiresInHours} onChange={(event) => setApplicationKeyForm((current) => ({ ...current, expiresInHours: event.target.value }))}><option value="168">7 days</option><option value="720">30 days</option><option value="2160">90 days</option><option value="">Never</option></select></label>
                </div>
                <div className="dialog-actions"><Button variant="ghost" type="button" onClick={closeApplicationKeyFlow}>Cancel</Button><Button variant="primary" type="submit">Create application key</Button></div>
              </form>}
            </div>
          </div> : null}
        </section>
      ) : activeView === "events" ? (
        <OperationalWorkspace {...operationalProps} view="events" />
      ) : (
        <section className="content-grid events-grid">
          <PageHeader
            title="Events"
            context={`${filteredEvents.length} events · live stream enabled`}
            description="Chronological registry activity with resource references for the selected environment scope."
          />
          <div className="panel">
            <div className="panel-heading">
              <h2>Events Timeline</h2>
              <span>Live / {filteredEvents.length} shown</span>
            </div>
            <div className="filter-bar">
              <label>
                Type
                <select
                  value={eventTypeFilter}
                  onChange={(event) => setEventTypeFilter(event.target.value)}
                >
                  <option value="">All event types</option>
                  {eventTypes.map((type) => (
                    <option key={type} value={type}>
                      {type}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Resource
                <select
                  value={eventResourceFilter}
                  onChange={(event) =>
                    setEventResourceFilter(event.target.value)
                  }
                >
                  <option value="all">All resources</option>
                  {eventResourceTypes.map((type) => (
                    <option key={type} value={type}>
                      {type}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Search
                <input
                  value={eventSearch}
                  onChange={(event) => setEventSearch(event.target.value)}
                />
              </label>
            </div>
            <div className="table">
              {filteredEvents.length === 0 ? (
                <EmptyState
                  title="No events match"
                  description="No registry events match the current type, resource, and search filters."
                />
              ) : (
                filteredEvents.map((event) => (
                  <div className="event-row timeline-row" key={event.id}>
                    <time title={formatTimestamp(event.timestamp)}>
                      {formatActivityTimestamp(event.timestamp)}
                    </time>
                    <strong>{formatEventType(event.type)}</strong>
                    <span>{event.message}</span>
                    <ResourceLink
                      onClick={
                        event.serviceId
                          ? () => {
                              setSelectedServiceId(event.serviceId ?? "");
                              setActiveView("services");
                            }
                          : undefined
                      }
                    >
                      {event.resourceType}:{event.resourceId}
                    </ResourceLink>
                    <span>{event.actor}</span>
                  </div>
                ))
              )}
            </div>
          </div>
        </section>
      )}
    </AppShell>
  );
}



function formatProtocol(value: string) {
  return value.replace("PROTOCOL_", "").toLowerCase();
}



function primaryEndpointForInstance(endpoints: Endpoint[], instanceId: string) {
  const instanceEndpoints = endpoints.filter(
    (endpoint) => endpoint.instanceId === instanceId,
  );
  return (
    instanceEndpoints.find((endpoint) => endpoint.primary) ??
    instanceEndpoints[0]
  );
}

function formatEndpointSummary(endpoint?: Endpoint) {
  if (!endpoint) {
    return "No endpoints";
  }
  const protocol = formatProtocol(endpoint.protocol).toUpperCase();
  const path =
    endpoint.path &&
    (endpoint.protocol === "PROTOCOL_HTTP" ||
      endpoint.protocol === "PROTOCOL_HTTPS")
      ? ` ${endpoint.path}`
      : "";
  return `${endpoint.primary ? "Primary " : ""}${protocol} :${endpoint.port}${path}`;
}

function formatEndpointUrl(endpoint: Endpoint, instance?: ServiceInstance) {
  const host = instance?.address || endpoint.instanceId;
  const port = endpoint.port || instance?.port || 0;
  const path = endpoint.path || "";
  switch (endpoint.protocol) {
    case "PROTOCOL_HTTP":
      return `http://${host}:${port}${path || "/"}`;
    case "PROTOCOL_HTTPS":
      return `https://${host}:${port}${path || "/"}`;
    case "PROTOCOL_GRPC":
      return `grpc://${host}:${port}${path}`;
    case "PROTOCOL_TCP":
      return `tcp://${host}:${port}`;
    case "PROTOCOL_UDP":
      return `udp://${host}:${port}`;
    default:
      return `${host}:${port}${path}`;
  }
}

function formatIncidentState(value: string) {
  return value.replace("INCIDENT_STATE_", "").toLowerCase();
}

function formatMap(value?: Record<string, string>) {
  if (!value || Object.keys(value).length === 0) {
    return "None";
  }
  return Object.entries(value)
    .map(([key, item]) => `${key}: ${item}`)
    .join(", ");
}

function splitCSV(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function compactMap(value: Record<string, string>) {
  return Object.fromEntries(
    Object.entries(value).filter(([, item]) => item.trim() !== ""),
  );
}

function formatPolicyScope(policy: AlertPolicy) {
  if (policy.deploymentId) {
    return `Environment scope ${policy.deploymentId}`;
  }
  if (policy.environmentId) {
    return `Environment ${policy.environmentId}`;
  }
  return "No scope";
}

const serviceTabs = ["overview", "instances", "health", "incidents", "events"] as const;
type ServiceTabPath = (typeof serviceTabs)[number];

function parseApplicationPath(pathname: string): { view: ActiveView; serviceId: string; serviceTab: ServiceTabPath } {
  const parts = pathname.split("/").filter(Boolean).map((part) => decodeURIComponent(part));
  const view = parts[0] as ActiveView | undefined;
  if (view === "services") {
    const serviceTab = parts[2] === "availability" ? "health" : serviceTabs.includes(parts[2] as ServiceTabPath) ? parts[2] as ServiceTabPath : "overview";
    return { view, serviceId: parts[1] ?? "", serviceTab };
  }
  return {
    view: view && ["dashboard", "environments", "health", "incidents", "alerts", "security", "events"].includes(view) ? view : "dashboard",
    serviceId: "",
    serviceTab: "overview",
  };
}

function applicationPath(view: ActiveView, serviceId: string, serviceTab: ServiceTabPath | "availability") {
  if (view === "services") {
    return serviceId ? `/services/${encodeURIComponent(serviceId)}/${serviceTab === "availability" ? "health" : serviceTab}` : "/services";
  }
  return `/${view}`;
}

function serviceOperationalStatus(service: Service, incidents: Incident[]) {
  const hasOpenIncident = incidents.some(
    (incident) =>
      incident.serviceId === service.id &&
      incident.state === "INCIDENT_STATE_OPEN",
  );
  return hasOpenIncident ? "degraded" : "healthy";
}

function LoginPage({ onAuthenticated }: { onAuthenticated: (user: UserAccount, mustChangePassword: boolean) => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      const response = await login({ username, password });
      if (!response.user) throw new Error("Login returned no user");
      onAuthenticated(response.user, Boolean(response.mustChangePassword ?? response.user.mustChangePassword));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Sign in failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <main className="auth-boundary">
      <section className="auth-card" aria-labelledby="login-title">
        <div className="brand-lockup auth-brand"><div className="brand-mark" aria-hidden="true">A</div><div><h1>Alauda</h1><p>Service Registry</p></div></div>
        <h2 id="login-title">Sign in to Alauda</h2>
        <form className="alauda-auth-form" onSubmit={submit}>
          <FormField label="Username"><Input autoComplete="username" value={username} onChange={(event) => setUsername(event.target.value)} required /></FormField>
          <FormField label="Password"><PasswordInput autoComplete="current-password" value={password} onChange={(event) => setPassword(event.target.value)} required /></FormField>
          {error ? <Alert tone="danger" title={error} /> : null}
          <FormButton variant="primary" disabled={saving} aria-busy={saving} type="submit">{saving ? "Signing in" : "Sign in"}</FormButton>
        </form>
      </section>
    </main>
  );
}

function PasswordChangePage({ onComplete }: { onComplete: () => void }) {
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving(true);
    setError("");
    try {
      await changePassword({ newPassword, confirmPassword });
      onComplete();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Password update failed");
    } finally {
      setSaving(false);
    }
  }

  return (
    <main className="auth-boundary">
      <section className="auth-card" aria-labelledby="password-title">
        <div className="brand-lockup auth-brand"><div className="brand-mark" aria-hidden="true">A</div><div><h1>Alauda</h1><p>Service Registry</p></div></div>
        <h2 id="password-title">Set a new password</h2>
        <p>For security, replace the temporary administrator password before continuing.</p>
        <form className="alauda-auth-form" onSubmit={submit}>
          <FormField label="New password" hint="At least 12 characters"><PasswordInput autoComplete="new-password" minLength={12} value={newPassword} onChange={(event) => setNewPassword(event.target.value)} required /></FormField>
          <FormField label="Confirm password"><PasswordInput autoComplete="new-password" minLength={12} value={confirmPassword} onChange={(event) => setConfirmPassword(event.target.value)} required /></FormField>
          {error ? <Alert tone="danger" title={error} /> : null}
          <FormButton variant="primary" disabled={saving} aria-busy={saving} type="submit">{saving ? "Updating" : "Update password"}</FormButton>
        </form>
      </section>
    </main>
  );
}

function App() {
  const [authState, setAuthState] = useState<"loading" | "anonymous" | "authenticated" | "must-change">("loading");

  useEffect(() => {
    setAuthenticationFailureHandler(() => {
      setAuthState("anonymous");
    });
    getCurrentSession()
      .then((response) => {
        if (!response.user) throw new Error("Session returned no user");
        setAuthState(response.mustChangePassword ? "must-change" : "authenticated");
      })
      .catch(() => setAuthState("anonymous"));
    return () => setAuthenticationFailureHandler(undefined);
  }, []);

  async function handleLogout() {
    try { await logout(); } finally { setAuthState("anonymous"); }
  }

  if (authState === "loading") return <main className="auth-boundary"><div className="status">Loading authentication...</div></main>;
  if (authState === "anonymous") return <LoginPage onAuthenticated={(_nextUser, mustChange) => { setAuthState(mustChange ? "must-change" : "authenticated"); }} />;
  if (authState === "must-change") return <PasswordChangePage onComplete={() => { setAuthState("anonymous"); }} />;
  return <AuthenticatedApp onLogout={handleLogout} />;
}

export default App;
