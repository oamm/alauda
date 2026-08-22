import { FormEvent, useEffect, useMemo, useState } from "react";

import "./App.css";
import { AppShell } from "./components/AppShell";
import { AvailabilityCard, MetricCard } from "./components/Cards";
import {
  EmptyState,
  LoadingRows,
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
import { DashboardView } from "./views/DashboardView";
import {
  AlertPolicy,
  ApiToken,
  createApiToken,
  createHealthCheck,
  createEnvironment,
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
  HealthResult,
  HealthStateView,
  Incident,
  listApiTokens,
  NotificationChannel,
  ServiceDeployment,
  ServiceInstance,
  eventStreamUrl,
  getAvailability,
  getInstanceHealthState,
  listAlertPolicies,
  listDeployments,
  listEndpoints,
  listHealthChecks,
  listEnvironments,
  listEvents,
  listHealthResults,
  listIncidents,
  listInstances,
  listNotificationChannels,
  listServices,
  listUsers,
  login,
  registerRuntime,
  resolveIncident,
  revokeApiToken,
  runHealthCheck,
  Service,
  setBearerToken,
  testNotificationChannel,
  updateAlertPolicy,
  updateEndpoint,
  updateHealthCheck,
  updateInstance,
  updateNotificationChannel,
  updateService,
  UserAccount,
} from "./api";

function newRegistrationEndpoint(
  overrides: Partial<{
    name: string;
    protocol: string;
    port: number;
    path: string;
    primary: boolean;
  }> = {},
) {
  return {
    name: "http",
    protocol: "PROTOCOL_HTTP",
    port: 8080,
    path: "/",
    primary: true,
    ...overrides,
  };
}

function App() {
  const [activeView, setActiveView] = useState<ActiveView>("dashboard");
  const [environments, setEnvironments] = useState<Environment[]>([]);
  const [services, setServices] = useState<Service[]>([]);
  const [deployments, setDeployments] = useState<ServiceDeployment[]>([]);
  const [instances, setInstances] = useState<ServiceInstance[]>([]);
  const [endpoints, setEndpoints] = useState<Endpoint[]>([]);
  const [healthChecks, setHealthChecks] = useState<HealthCheck[]>([]);
  const [healthStates, setHealthStates] = useState<HealthStateView[]>([]);
  const [healthResults, setHealthResults] = useState<HealthResult[]>([]);
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [events, setEvents] = useState<EventRecord[]>([]);
  const [notificationChannels, setNotificationChannels] = useState<
    NotificationChannel[]
  >([]);
  const [alertPolicies, setAlertPolicies] = useState<AlertPolicy[]>([]);
  const [users, setUsers] = useState<UserAccount[]>([]);
  const [apiTokens, setApiTokens] = useState<ApiToken[]>([]);
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
  const [latestState, setLatestState] = useState<HealthStateView | null>(null);
  const [selectedEnvironmentId, setSelectedEnvironmentId] = useState("");
  const [selectedServiceId, setSelectedServiceId] = useState("");
  const [selectedHealthCheckId, setSelectedHealthCheckId] = useState("");
  const [serviceSearch, setServiceSearch] = useState("");
  const [serviceHealthFilter, setServiceHealthFilter] = useState("all");
  const [serviceTagFilter, setServiceTagFilter] = useState("");
  const [servicePage, setServicePage] = useState(1);
  const [serviceTab, setServiceTab] = useState<
    "overview" | "runtime" | "health" | "incidents" | "events"
  >("overview");
  const [showCreateService, setShowCreateService] = useState(false);
  const [showAddRuntime, setShowAddRuntime] = useState(false);
  const [showRuntimeHealth, setShowRuntimeHealth] = useState(false);
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
  const [darkMode, setDarkMode] = useState(false);
  const [loading, setLoading] = useState(true);
  const [savingService, setSavingService] = useState(false);
  const [savingRegistration, setSavingRegistration] = useState(false);
  const [savingRuntimeEdit, setSavingRuntimeEdit] = useState(false);
  const [savingEnvironment, setSavingEnvironment] = useState(false);
  const [savingHealthCheck, setSavingHealthCheck] = useState(false);
  const [savingChannel, setSavingChannel] = useState(false);
  const [savingPolicy, setSavingPolicy] = useState(false);
  const [bulkActionRunning, setBulkActionRunning] = useState(false);
  const [runningHealthCheck, setRunningHealthCheck] = useState(false);
  const [resolvingIncidentId, setResolvingIncidentId] = useState("");
  const [testingChannelId, setTestingChannelId] = useState("");
  const [testResult, setTestResult] = useState("");
  const [authToken, setAuthToken] = useState(() =>
    typeof window === "undefined"
      ? ""
      : (window.localStorage.getItem("registryToken") ?? ""),
  );
  const [authMessage, setAuthMessage] = useState("");
  const [error, setError] = useState("");
  const [registrationSuccess, setRegistrationSuccess] = useState("");
  const [serviceForm, setServiceForm] = useState({
    name: "",
    displayName: "",
    description: "",
  });
  const [serviceEditForm, setServiceEditForm] = useState({
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
    configureHealth: false,
    healthName: "readiness",
    healthType: "HEALTH_CHECK_TYPE_HTTP",
    healthPath: "/healthz",
    healthIntervalSeconds: 10,
    healthTimeoutSeconds: 3,
    healthFailuresBeforeUnhealthy: 3,
    healthSuccessesBeforeHealthy: 2,
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
  const [environmentForm, setEnvironmentForm] = useState({
    key: "",
    name: "",
    tier: "",
    description: "",
  });
  const [healthForm, setHealthForm] = useState({
    instanceId: "",
    endpointId: "",
    name: "",
    type: "HEALTH_CHECK_TYPE_HTTP",
    intervalSeconds: 10,
    timeoutSeconds: 3,
    failuresBeforeUnhealthy: 3,
    successesBeforeHealthy: 2,
    description: "",
    path: "/healthz",
    expectedStatus: "200-299",
  });
  const [healthEditForm, setHealthEditForm] = useState({
    enabled: true,
    intervalSeconds: 10,
    timeoutSeconds: 3,
    failuresBeforeUnhealthy: 3,
    successesBeforeHealthy: 2,
    description: "",
  });
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
  const [loginForm, setLoginForm] = useState({
    username: "",
    password: "",
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

  useEffect(() => {
    setBearerToken(authToken);
    if (typeof window !== "undefined") {
      if (authToken) {
        window.localStorage.setItem("registryToken", authToken);
      } else {
        window.localStorage.removeItem("registryToken");
      }
    }
  }, [authToken]);

  const selectedService = useMemo(
    () => services.find((service) => service.id === selectedServiceId),
    [selectedServiceId, services],
  );
  const selectedHealthCheck = useMemo(
    () => healthChecks.find((check) => check.id === selectedHealthCheckId),
    [healthChecks, selectedHealthCheckId],
  );
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
  const selectedServiceEndpoints = useMemo(() => {
    const instanceIDs = new Set(
      selectedServiceInstances.map((instance) => instance.id),
    );
    return endpoints.filter((endpoint) => instanceIDs.has(endpoint.instanceId));
  }, [endpoints, selectedServiceInstances]);
  const selectedServiceHealthChecks = useMemo(() => {
    const instanceIDs = new Set(
      selectedServiceInstances.map((instance) => instance.id),
    );
    return healthChecks.filter((check) => instanceIDs.has(check.instanceId));
  }, [healthChecks, selectedServiceInstances]);
  const selectedServiceEvents = useMemo(
    () => events.filter((event) => event.serviceId === selectedServiceId),
    [events, selectedServiceId],
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
  const healthStateByInstanceId = useMemo(
    () => new Map(healthStates.map((state) => [state.instanceId, state])),
    [healthStates],
  );
  const filteredHealthInstances = useMemo(() => {
    return instances.filter((instance) => {
      if (healthStatusFilter === "all") {
        return true;
      }
      const state = healthStateByInstanceId.get(instance.id);
      const currentState =
        state?.currentState?.replace("HEALTH_STATE_", "").toLowerCase() ??
        "unknown";
      return healthStatusFilter === currentState;
    });
  }, [healthStateByInstanceId, healthStatusFilter, instances]);
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
    const nextHealthStates = (
      await Promise.all(
        nextInstances.map((instance) =>
          getInstanceHealthState(instance.id).catch(() => null),
        ),
      )
    ).filter((state): state is HealthStateView => state !== null);
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
    setSelectedHealthCheckId((current) => {
      if (nextHealthChecks.some((check) => check.id === current)) {
        return current;
      }
      return nextHealthChecks[0]?.id ?? "";
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

  async function handleLogin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setAuthMessage("");
    try {
      const response = await login(loginForm);
      if (!response.token) {
        throw new Error("Login returned no token");
      }
      setAuthToken(response.token);
      setLoginForm({ username: "", password: "" });
      setAuthMessage("Signed in.");
      await loadSecurityData();
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to sign in");
    }
  }

  function handleLogout() {
    setAuthToken("");
    setUsers([]);
    setApiTokens([]);
    setAuthMessage("Signed out.");
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

  function startEditService(service: Service) {
    setEditingServiceId(service.id);
    setServiceEditForm({
      displayName: service.displayName || service.name,
      description: service.description ?? "",
    });
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
        endpointIndex === index ? { ...endpoint, ...updates } : endpoint,
      ),
    }));
  }

  function addRegistrationEndpoint() {
    setRegistrationForm((current) => ({
      ...current,
      endpoints: [
        ...current.endpoints,
        newRegistrationEndpoint({
          name: "",
          primary: current.endpoints.length === 0,
        }),
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
    setEditingEndpointId("");
    setAddingEndpointInstanceId(instance.id);
    setEndpointEditForm({
      name: "http",
      protocol: "PROTOCOL_HTTP",
      port: primaryEndpointForInstance(endpoints, instance.id)?.port || 8080,
      path: "/",
      enabled: true,
      primary: !endpoints.some((endpoint) => endpoint.instanceId === instance.id),
    });
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
    setSavingRuntimeEdit(true);
    setError("");
    try {
      if (addingEndpointInstanceId) {
        await createEndpoint({
          instanceId: addingEndpointInstanceId,
          name: endpointEditForm.name,
          protocol: endpointEditForm.protocol,
          port: endpointEditForm.port,
          path: endpointEditForm.path,
          enabled: endpointEditForm.enabled,
          primary: endpointEditForm.primary,
        });
      } else {
        await updateEndpoint({
          id: editingEndpointId,
          name: endpointEditForm.name,
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
        err instanceof Error ? err.message : "Failed to update endpoint",
      );
    } finally {
      setSavingRuntimeEdit(false);
    }
  }

  function startEditHealthCheck(check: HealthCheck) {
    setEditingHealthCheckId(check.id);
    setHealthEditForm({
      enabled: check.enabled,
      intervalSeconds: check.intervalSeconds,
      timeoutSeconds: check.timeoutSeconds,
      failuresBeforeUnhealthy: check.failuresBeforeUnhealthy,
      successesBeforeHealthy: check.successesBeforeHealthy,
      description: check.description ?? "",
    });
  }

  function startConfigureHealth(instance: ServiceInstance, endpoint?: Endpoint) {
    setEditingHealthCheckId("new");
    setHealthForm((current) => ({
      ...current,
      instanceId: instance.id,
      endpointId: endpoint?.id ?? "",
      name: "readiness",
      type: endpoint ? healthTypeForEndpoint(endpoint.protocol) : current.type,
    }));
  }

  async function handleUpdateHealthCheck(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingHealthCheck(true);
    setError("");
    try {
      if (editingHealthCheckId === "new") {
        const check = await createHealthCheck({
          instanceId: healthForm.instanceId,
          endpointId: healthForm.endpointId,
          name: healthForm.name,
          type: healthForm.type,
          enabled: true,
          intervalSeconds: healthForm.intervalSeconds,
          timeoutSeconds: healthForm.timeoutSeconds,
          failuresBeforeUnhealthy: healthForm.failuresBeforeUnhealthy,
          successesBeforeHealthy: healthForm.successesBeforeHealthy,
          description: healthForm.description,
          metadata: compactMap({
            path: healthForm.path,
            expectedStatus: healthForm.expectedStatus,
          }),
        });
        setSelectedHealthCheckId(check.id);
      } else {
        await updateHealthCheck({
          id: editingHealthCheckId,
          enabled: healthEditForm.enabled,
          intervalSeconds: healthEditForm.intervalSeconds,
          timeoutSeconds: healthEditForm.timeoutSeconds,
          failuresBeforeUnhealthy: healthEditForm.failuresBeforeUnhealthy,
          successesBeforeHealthy: healthEditForm.successesBeforeHealthy,
          description: healthEditForm.description,
        });
      }
      setEditingHealthCheckId("");
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to save health check",
      );
    } finally {
      setSavingHealthCheck(false);
    }
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
    const primaryEndpoint = registrationForm.endpoints[0];
    if (
      !registrationForm.environmentId ||
      !registrationForm.instanceName.trim() ||
      !registrationForm.address.trim() ||
      !primaryEndpoint?.name.trim() ||
      primaryEndpoint.port < 1 ||
      primaryEndpoint.port > 65535
    ) {
      setError("Environment, instance name, address, endpoint name, and a valid endpoint port are required.");
      return;
    }
    if (
      registrationForm.configureHealth &&
      (!registrationForm.healthName.trim() ||
        registrationForm.healthIntervalSeconds < 1 ||
        registrationForm.healthTimeoutSeconds < 1)
    ) {
      setError("Health check name, interval, and timeout are required when health monitoring is configured.");
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
          name: endpoint.name,
          protocol: endpoint.protocol,
          port: endpoint.port,
          path: endpoint.path,
          enabled: true,
          primary: endpoint.primary,
        })),
      });

      let healthMessage = "";
      if (registrationForm.configureHealth) {
        try {
          const registeredInstance = registration.instance;
          if (!registeredInstance) {
            throw new Error("RegisterRuntime returned no instance");
          }
          const endpoint =
            registration.endpoints?.find((item) => item.primary) ??
            registration.endpoints?.[0];
          const check = await createHealthCheck({
            instanceId: registeredInstance.id,
            endpointId: endpoint?.id ?? "",
            name: registrationForm.healthName,
            type: registrationForm.healthType,
            enabled: true,
            intervalSeconds: registrationForm.healthIntervalSeconds,
            timeoutSeconds: registrationForm.healthTimeoutSeconds,
            failuresBeforeUnhealthy:
              registrationForm.healthFailuresBeforeUnhealthy,
            successesBeforeHealthy:
              registrationForm.healthSuccessesBeforeHealthy,
            description: "Created during runtime registration",
            metadata: {
              path: endpoint?.path || primaryEndpoint.path || "/",
              expectedStatus: "200-299",
            },
          });
          healthMessage = ` Health check ${check.name} created.`;
          setSelectedHealthCheckId(check.id);
        } catch (healthErr) {
          setError(
            healthErr instanceof Error
              ? `Runtime registered, but health check creation failed: ${healthErr.message}`
              : "Runtime registered, but health check creation failed.",
          );
        }
      }

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
      setServiceTab("runtime");
      setShowAddRuntime(false);
      setRegistrationSuccess(
        `Runtime registered successfully: ${registration.instance?.name ?? "instance"}.${healthMessage}`,
      );
      setAuthMessage(
        `Runtime registered successfully: ${registration.instance?.name ?? "instance"}.${healthMessage}`,
      );
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to register instance",
      );
    } finally {
      setSavingRegistration(false);
    }
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

  async function handleCreateEnvironment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingEnvironment(true);
    setError("");
    try {
      const environment = await createEnvironment(environmentForm);
      setEnvironmentForm({ key: "", name: "", tier: "", description: "" });
      await loadCatalog(selectedEnvironmentId);
      setSelectedEnvironmentId(environment.id);
      setActiveView("environments");
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create environment",
      );
    } finally {
      setSavingEnvironment(false);
    }
  }

  async function handleCreateHealthCheck(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSavingHealthCheck(true);
    setError("");
    try {
      const check = await createHealthCheck({
        instanceId: healthForm.instanceId,
        endpointId: healthForm.endpointId,
        name: healthForm.name,
        type: healthForm.type,
        enabled: true,
        intervalSeconds: healthForm.intervalSeconds,
        timeoutSeconds: healthForm.timeoutSeconds,
        failuresBeforeUnhealthy: healthForm.failuresBeforeUnhealthy,
        successesBeforeHealthy: healthForm.successesBeforeHealthy,
        description: healthForm.description,
        metadata: {
          path: healthForm.path,
          expectedStatus: healthForm.expectedStatus,
        },
      });
      setSelectedHealthCheckId(check.id);
      setActiveView("health");
      await loadCatalog(selectedEnvironmentId);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to create health check",
      );
    } finally {
      setSavingHealthCheck(false);
    }
  }

  async function handleRunHealthCheck(id = selectedHealthCheckId) {
    if (!id) {
      return;
    }
    setRunningHealthCheck(true);
    setError("");
    try {
      const response = await runHealthCheck(id);
      setLatestState(response.state ?? null);
      setHealthResults(await listHealthResults(id));
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to run health check",
      );
    } finally {
      setRunningHealthCheck(false);
    }
  }

  async function handleSelectHealthCheck(id: string) {
    setSelectedHealthCheckId(id);
    setLatestState(null);
    setHealthResults(await listHealthResults(id));
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
  const selectedServiceRuntimeSummary = [
    pluralize(selectedServiceDeployments.length, "environment"),
    pluralize(selectedServiceInstances.length, "instance"),
    pluralize(selectedServiceEndpoints.length, "endpoint"),
  ].join(" · ");
  const selectedServiceLastActivity =
    selectedServiceEvents.length > 0
      ? formatActivityTimestamp(selectedServiceEvents[0].timestamp)
      : "No activity";
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
  void startEditInstance;
  void startEditEndpoint;
  void handleUpdateInstance;
  void handleDeleteInstance;
  void handleUpdateEndpoint;
  void handleDeleteEndpoint;
  void formatEndpointSummary;
  void formatEndpointUrl;

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
      onSecurityOpen={() => {
        setActiveView("security");
        if (authToken) {
          loadSecurityData().catch((err: unknown) =>
            setError(
              err instanceof Error
                ? err.message
                : "Failed to load security data",
            ),
          );
        }
      }}
      onToggleDarkMode={() => setDarkMode((value) => !value)}
      onViewChange={setActiveView}
    >
      {loading ? <div className="status">Loading registry data...</div> : null}

      {error && (
        <div className="error" role="alert">
          {error}
        </div>
      )}

      {testResult && <div className="success">{testResult}</div>}
      {authMessage && <div className="success">{authMessage}</div>}

      {activeView === "dashboard" ? (
        <DashboardView
          alertPolicies={alertPolicies}
          availability={availability}
          environments={environments}
          events={events}
          healthChecks={healthChecks}
          incidents={incidents}
          notificationChannels={notificationChannels}
          selectedEnvironmentId={selectedEnvironmentId}
          services={services}
        />
      ) : activeView === "services" ? (
        <section className="services-workflow">
          <PageHeader
            title="Services"
            context={`${pluralize(filteredServices.length, "service")} ${
              selectedEnvironmentId
                ? `in ${currentEnvironmentName}`
                : "across all environments"
            }`}
            description="Manage service catalog and runtime registrations."
            action={
              <button
                className="button-primary"
                type="button"
                onClick={() => setShowCreateService(true)}
              >
                + Create service
              </button>
            }
          />
          <div className="panel services-master">
            <div className="panel-heading">
              <h2>Services</h2>
              <span>
                {loading
                  ? "Loading"
                  : `${filteredServices.length} of ${services.length}`}
              </span>
            </div>
            <div className="service-filter-bar">
              <label className="service-search">
                <span>Search</span>
                <input
                  aria-label="Search services"
                  placeholder="Search services..."
                  value={serviceSearch}
                  onChange={(event) => setServiceSearch(event.target.value)}
                />
              </label>
              <div className="service-filter-row">
                <label>
                  <span>Health</span>
                  <select
                    value={serviceHealthFilter}
                    onChange={(event) =>
                      setServiceHealthFilter(event.target.value)
                    }
                  >
                    <option value="all">All health</option>
                    <option value="healthy">Healthy</option>
                    <option value="degraded">Degraded</option>
                  </select>
                </label>
                <label>
                  <span>Tags</span>
                  <input
                    aria-label="Filter by tag"
                    placeholder="Any tag"
                    value={serviceTagFilter}
                    onChange={(event) =>
                      setServiceTagFilter(event.target.value)
                    }
                  />
                </label>
              </div>
              {serviceSearch ||
              serviceHealthFilter !== "all" ||
              serviceTagFilter ? (
                <button
                  className="button-ghost clear-filters"
                  type="button"
                  onClick={() => {
                    setServiceSearch("");
                    setServiceHealthFilter("all");
                    setServiceTagFilter("");
                  }}
                >
                  Clear filters
                </button>
              ) : null}
            </div>
            {selectedBulkServiceIds.length > 0 ? (
              <div className="bulk-bar">
                <span>{selectedBulkServiceIds.length} selected</span>
                <button type="button" onClick={selectVisibleServices}>
                  Select visible
                </button>
                <button type="button" onClick={handleCopySelectedServiceIds}>
                  Copy IDs
                </button>
                <button
                  disabled={bulkActionRunning}
                  type="button"
                  onClick={handleDeleteSelectedServices}
                >
                  {bulkActionRunning ? "Deleting" : "Delete"}
                </button>
                <button
                  className="button-ghost"
                  type="button"
                  onClick={() => setSelectedBulkServiceIds([])}
                >
                  Clear
                </button>
              </div>
            ) : null}
            {loading ? (
              <LoadingRows rows={5} />
            ) : filteredServices.length === 0 ? (
              <EmptyState
                title="No matching services"
                description="No catalog services match the current search, health, and tag filters."
              />
            ) : (
              <div className="service-list">
                {visibleServices.map((service) => {
                  const serviceDeployments = deployments.filter(
                    (deployment) => deployment.serviceId === service.id,
                  );
                  const deploymentIds = new Set(
                    serviceDeployments.map((deployment) => deployment.id),
                  );
                  const serviceInstanceCount = instances.filter((instance) =>
                    deploymentIds.has(instance.deploymentId),
                  ).length;
                  const serviceStatus = serviceOperationalStatus(
                    service,
                    incidents,
                  );

                  return (
                    <div
                      className={
                        service.id === selectedServiceId
                          ? "service-row selected"
                          : "service-row"
                      }
                      key={service.id}
                    >
                      <label className="inline-check service-row-check">
                        <input
                          aria-label={`Select ${service.displayName || service.name}`}
                          checked={selectedBulkServiceIds.includes(service.id)}
                          type="checkbox"
                          onChange={() => toggleBulkService(service.id)}
                        />
                      </label>
                      <button
                        aria-pressed={service.id === selectedServiceId}
                        type="button"
                        onClick={() => setSelectedServiceId(service.id)}
                      >
                        <span className="service-row-title">
                          <strong>{service.displayName || service.name}</strong>
                          <StatusBadge status={serviceStatus} />
                        </span>
                        <span>{service.name}</span>
                        <span>
                          {pluralize(serviceInstanceCount, "instance")}
                        </span>
                      </button>
                    </div>
                  );
                })}
              </div>
            )}
            <div className="pagination-bar">
              <button
                disabled={servicePage <= 1}
                onClick={() => setServicePage((page) => page - 1)}
                type="button"
              >
                Previous
              </button>
              <span>
                Page {servicePage} of {servicePageCount}
              </span>
              <button
                disabled={servicePage >= servicePageCount}
                onClick={() => setServicePage((page) => page + 1)}
                type="button"
              >
                Next
              </button>
            </div>
          </div>

          <div className="panel service-workspace">
            {selectedService ? (
              <>
                <div className="service-workspace-header">
                  <div>
                    <div className="breadcrumb">
                      Services /{" "}
                      {selectedService.displayName || selectedService.name}
                    </div>
                    <div className="service-title-line">
                      <h2>
                        {selectedService.displayName || selectedService.name}
                      </h2>
                      <StatusBadge
                        status={serviceOperationalStatus(
                          selectedService,
                          incidents,
                        )}
                      />
                    </div>
                    {selectedService.description ? (
                      <p>{selectedService.description}</p>
                    ) : null}
                    <span>
                      {selectedService.name} · {selectedServiceRuntimeSummary}
                    </span>
                  </div>
                  <details className="resource-menu">
                    <summary aria-label="Service actions">...</summary>
                    <div className="resource-menu-items">
                      <button
                        type="button"
                        onClick={() => startEditService(selectedService)}
                      >
                        Edit service
                      </button>
                      <button
                        type="button"
                        onClick={() =>
                          navigator.clipboard
                            .writeText(selectedService.id)
                            .then(() => setTestResult("Service ID copied."))
                            .catch(() =>
                              setError("Failed to copy service ID."),
                            )
                        }
                      >
                        Copy service ID
                      </button>
                      <button
                        className="button-danger"
                        disabled={savingService}
                        type="button"
                        onClick={() => handleDeleteService(selectedService)}
                      >
                        Delete service
                      </button>
                    </div>
                  </details>
                </div>

                <div
                  className="service-tabs"
                  role="tablist"
                  aria-label="Service sections"
                >
                  {(
                    [
                      "overview",
                      "runtime",
                      "health",
                      "incidents",
                      "events",
                    ] as const
                  ).map((tab) => (
                    <button
                      aria-selected={serviceTab === tab}
                      className={serviceTab === tab ? "active" : ""}
                      key={tab}
                      onClick={() => setServiceTab(tab)}
                      role="tab"
                      type="button"
                    >
                      {tab[0].toUpperCase() + tab.slice(1)}
                    </button>
                  ))}
                </div>

                {serviceTab === "overview" ? (
                  <div className="service-tab-panel">
                    <div className="summary-strip">
                      <div>
                        <span>Runtime</span>
                        <strong>{selectedServiceRuntimeSummary}</strong>
                      </div>
                      <div>
                        <span>Catalog key</span>
                        <strong>{selectedService.name}</strong>
                      </div>
                      <div>
                        <span>Last activity</span>
                        <strong>{selectedServiceLastActivity}</strong>
                      </div>
                    </div>
                    <section className="overview-section">
                      <div className="tab-toolbar">
                        <h3>Availability</h3>
                      </div>
                      <div className="availability-grid">
                        <AvailabilityCard
                          label="24 hours"
                          summary={serviceAvailability.availability24h}
                        />
                        <AvailabilityCard
                          label="7 days"
                          summary={serviceAvailability.availability7d}
                        />
                        <AvailabilityCard
                          label="30 days"
                          summary={serviceAvailability.availability30d}
                        />
                      </div>
                    </section>
                    <div className="scoped-list">
                      <h3>Recent activity</h3>
                      {selectedServiceEvents.length === 0 ? (
                        <EmptyState
                          title="No recent activity"
                          description="No events have been recorded for this service in the selected scope."
                        />
                      ) : (
                        selectedServiceEvents.slice(0, 5).map((event) => (
                          <div className="activity-row" key={event.id}>
                            <time title={formatTimestamp(event.timestamp)}>
                              {formatActivityTimestamp(event.timestamp)}
                            </time>
                            <div>
                              <strong>{formatEventType(event.type)}</strong>
                              <span>{event.message}</span>
                            </div>
                            <small>{event.resourceType}</small>
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                ) : null}

                {serviceTab === "runtime" ? (
                  <div className="service-tab-panel">
                    <div className="tab-toolbar">
                      <div>
                        <h3>Runtime</h3>
                        <span>
                          {selectedEnvironmentId
                            ? currentEnvironmentName
                            : "Grouped by environment"}
                        </span>
                      </div>
                      {selectedServiceDeployments.length === 0 ? (
                        <button
                          type="button"
                          onClick={() => setShowAddRuntime(true)}
                        >
                          Add runtime
                        </button>
                      ) : null}
                    </div>
                    {selectedServiceDeployments.length === 0 ? (
                      <EmptyState
                        title="No runtime registered"
                        description={`${selectedService.displayName || selectedService.name} exists in the catalog but has no runtime instance in this scope.`}
                        action={
                          <button
                            type="button"
                            onClick={() => setShowAddRuntime(true)}
                          >
                            Add runtime
                          </button>
                        }
                      />
                    ) : (
                      selectedServiceDeployments.map((deployment) => {
                        const deploymentInstances =
                          selectedServiceInstances.filter(
                            (instance) =>
                              instance.deploymentId === deployment.id,
                          );
                        const deploymentIncident = incidents.some(
                          (incident) =>
                            incident.deploymentId === deployment.id &&
                            incident.state === "INCIDENT_STATE_OPEN",
                        );
                        return (
                          <section
                            className="runtime-environment"
                            key={deployment.id}
                          >
                            <div className="runtime-environment-heading">
                              <div>
                                <h3>
                                  {environmentName(
                                    environments,
                                    deployment.environmentId,
                                  )}
                                </h3>
                                <span>
                                  {pluralize(
                                    deploymentInstances.length,
                                    "instance",
                                  )}
                                </span>
                              </div>
                              <StatusBadge
                                status={
                                  deploymentIncident ? "degraded" : "healthy"
                                }
                              />
                              <button
                                type="button"
                                onClick={() => {
                                  setRegistrationForm((current) => ({
                                    ...current,
                                    environmentId: deployment.environmentId,
                                  }));
                                  setShowAddRuntime(true);
                                }}
                              >
                                + Add instance
                              </button>
                            </div>
                            {deploymentInstances.length === 0 ? (
                              <EmptyState
                                title="No instances"
                                description="This deployment has no runtime targets yet."
                                action={
                                  <button
                                    type="button"
                                    onClick={() => setShowAddRuntime(true)}
                                  >
                                    Add instance
                                  </button>
                                }
                              />
                            ) : (
                              deploymentInstances.map((instance) => {
                                const instanceEndpoints =
                                  selectedServiceEndpoints.filter(
                                    (endpoint) =>
                                      endpoint.instanceId === instance.id,
                                  );
                                const instanceChecks =
                                  selectedServiceHealthChecks.filter(
                                    (check) => check.instanceId === instance.id,
                                  );
                                const state = healthStateByInstanceId.get(
                                  instance.id,
                                );
                                return (
                                  <article
                                    className="runtime-instance"
                                    key={instance.id}
                                  >
                                    <div className="runtime-instance-header">
                                      <div>
                                        <strong>
                                          {instance.name}
                                        </strong>
                                        <span>
                                          {instance.address}:
                                          {instance.port ||
                                            primaryEndpointForInstance(
                                              instanceEndpoints,
                                              instance.id,
                                            )?.port ||
                                            "dynamic"}
                                        </span>
                                      </div>
                                      <StatusBadge
                                        status={
                                          state?.currentState
                                            ? formatHealthState(
                                                state.currentState,
                                              )
                                            : instance.enabled
                                              ? "enabled"
                                              : "disabled"
                                        }
                                      />
                                      <details className="resource-menu">
                                        <summary
                                          aria-label={`Instance actions for ${instance.name}`}
                                        >
                                          ...
                                        </summary>
                                        <div className="resource-menu-items">
                                          <button
                                            type="button"
                                            onClick={() =>
                                              startEditInstance(instance)
                                            }
                                          >
                                            Edit instance
                                          </button>
                                          <button
                                            type="button"
                                            onClick={() =>
                                              startAddEndpoint(instance)
                                            }
                                          >
                                            Add endpoint
                                          </button>
                                          <button
                                            className="button-danger"
                                            disabled={savingRuntimeEdit}
                                            type="button"
                                            onClick={() =>
                                              handleDeleteInstance(instance)
                                            }
                                          >
                                            Delete instance
                                          </button>
                                        </div>
                                      </details>
                                    </div>
                                    {editingInstanceId === instance.id ? (
                                      <form
                                        className="inline-edit-form runtime-edit-form"
                                        onSubmit={handleUpdateInstance}
                                      >
                                        <label>
                                          Address
                                          <input
                                            required
                                            value={instanceEditForm.address}
                                            onChange={(event) =>
                                              setInstanceEditForm(
                                                (current) => ({
                                                  ...current,
                                                  address: event.target.value,
                                                }),
                                              )
                                            }
                                          />
                                        </label>
                                        <label>
                                          Description
                                          <input
                                            value={
                                              instanceEditForm.description
                                            }
                                            onChange={(event) =>
                                              setInstanceEditForm(
                                                (current) => ({
                                                  ...current,
                                                  description:
                                                    event.target.value,
                                                }),
                                              )
                                            }
                                          />
                                        </label>
                                        <label className="checkbox-label">
                                          <input
                                            checked={instanceEditForm.enabled}
                                            type="checkbox"
                                            onChange={(event) =>
                                              setInstanceEditForm(
                                                (current) => ({
                                                  ...current,
                                                  enabled:
                                                    event.target.checked,
                                                }),
                                              )
                                            }
                                          />
                                          Enabled
                                        </label>
                                        <button
                                          disabled={savingRuntimeEdit}
                                          type="submit"
                                        >
                                          Save instance
                                        </button>
                                        <button
                                          type="button"
                                          onClick={() =>
                                            setEditingInstanceId("")
                                          }
                                        >
                                          Cancel
                                        </button>
                                      </form>
                                    ) : null}
                                    <div className="runtime-nested">
                                      <div className="runtime-subheading">
                                        <h4>Endpoints</h4>
                                        <button
                                          type="button"
                                          onClick={() =>
                                            startAddEndpoint(instance)
                                          }
                                        >
                                          + Add endpoint
                                        </button>
                                      </div>
                                      {addingEndpointInstanceId ===
                                      instance.id ? (
                                        <form
                                          className="inline-edit-form runtime-edit-form"
                                          onSubmit={handleUpdateEndpoint}
                                        >
                                          <label>
                                            Name
                                            <input
                                              required
                                              value={endpointEditForm.name}
                                              onChange={(event) =>
                                                setEndpointEditForm(
                                                  (current) => ({
                                                    ...current,
                                                    name: event.target.value,
                                                  }),
                                                )
                                              }
                                            />
                                          </label>
                                          <label>
                                            Protocol
                                            <select
                                              value={endpointEditForm.protocol}
                                              onChange={(event) =>
                                                setEndpointEditForm(
                                                  (current) => ({
                                                    ...current,
                                                    protocol:
                                                      event.target.value,
                                                  }),
                                                )
                                              }
                                            >
                                              <option value="PROTOCOL_HTTP">
                                                HTTP
                                              </option>
                                              <option value="PROTOCOL_HTTPS">
                                                HTTPS
                                              </option>
                                              <option value="PROTOCOL_GRPC">
                                                gRPC
                                              </option>
                                              <option value="PROTOCOL_TCP">
                                                TCP
                                              </option>
                                              <option value="PROTOCOL_UDP">
                                                UDP
                                              </option>
                                            </select>
                                          </label>
                                          <label>
                                            Port
                                            <input
                                              max="65535"
                                              min="1"
                                              required
                                              type="number"
                                              value={endpointEditForm.port}
                                              onChange={(event) =>
                                                setEndpointEditForm(
                                                  (current) => ({
                                                    ...current,
                                                    port: Number(
                                                      event.target.value,
                                                    ),
                                                  }),
                                                )
                                              }
                                            />
                                          </label>
                                          <label>
                                            Path
                                            <input
                                              value={endpointEditForm.path}
                                              onChange={(event) =>
                                                setEndpointEditForm(
                                                  (current) => ({
                                                    ...current,
                                                    path: event.target.value,
                                                  }),
                                                )
                                              }
                                            />
                                          </label>
                                          <button
                                            disabled={savingRuntimeEdit}
                                            type="submit"
                                          >
                                            Add endpoint
                                          </button>
                                          <button
                                            type="button"
                                            onClick={() =>
                                              setAddingEndpointInstanceId("")
                                            }
                                          >
                                            Cancel
                                          </button>
                                        </form>
                                      ) : null}
                                      {instanceEndpoints.length === 0 ? (
                                        <p>No endpoints attached.</p>
                                      ) : (
                                        instanceEndpoints.map((endpoint) => (
                                          <div
                                            className="runtime-child-row"
                                            key={endpoint.id}
                                          >
                                            <strong>
                                              {formatProtocol(
                                                endpoint.protocol,
                                              ).toUpperCase()}{" "}
                                              · :{endpoint.port}
                                              {endpoint.path || ""}
                                            </strong>
                                            <StatusBadge
                                              status={
                                                endpoint.enabled
                                                  ? "enabled"
                                                  : "disabled"
                                              }
                                            />
                                            <details className="resource-menu">
                                              <summary
                                                aria-label={`Endpoint actions for ${endpoint.name}`}
                                              >
                                                ...
                                              </summary>
                                              <div className="resource-menu-items">
                                                <button
                                                  type="button"
                                                  onClick={() =>
                                                    startEditEndpoint(endpoint)
                                                  }
                                                >
                                                  Edit endpoint
                                                </button>
                                                <button
                                                  type="button"
                                                  onClick={() =>
                                                    startConfigureHealth(
                                                      instance,
                                                      endpoint,
                                                    )
                                                  }
                                                >
                                                  Configure health
                                                </button>
                                                <button
                                                  className="button-danger"
                                                  disabled={savingRuntimeEdit}
                                                  type="button"
                                                  onClick={() =>
                                                    handleDeleteEndpoint(
                                                      endpoint,
                                                    )
                                                  }
                                                >
                                                  Delete endpoint
                                                </button>
                                              </div>
                                            </details>
                                            {editingEndpointId ===
                                            endpoint.id ? (
                                              <form
                                                className="inline-edit-form runtime-edit-form"
                                                onSubmit={
                                                  handleUpdateEndpoint
                                                }
                                              >
                                                <label>
                                                  Name
                                                  <input
                                                    required
                                                    value={
                                                      endpointEditForm.name
                                                    }
                                                    onChange={(event) =>
                                                      setEndpointEditForm(
                                                        (current) => ({
                                                          ...current,
                                                          name: event.target
                                                            .value,
                                                        }),
                                                      )
                                                    }
                                                  />
                                                </label>
                                                <label>
                                                  Protocol
                                                  <select
                                                    value={
                                                      endpointEditForm.protocol
                                                    }
                                                    onChange={(event) =>
                                                      setEndpointEditForm(
                                                        (current) => ({
                                                          ...current,
                                                          protocol:
                                                            event.target.value,
                                                        }),
                                                      )
                                                    }
                                                  >
                                                    <option value="PROTOCOL_HTTP">
                                                      HTTP
                                                    </option>
                                                    <option value="PROTOCOL_HTTPS">
                                                      HTTPS
                                                    </option>
                                                    <option value="PROTOCOL_GRPC">
                                                      gRPC
                                                    </option>
                                                    <option value="PROTOCOL_TCP">
                                                      TCP
                                                    </option>
                                                    <option value="PROTOCOL_UDP">
                                                      UDP
                                                    </option>
                                                  </select>
                                                </label>
                                                <label>
                                                  Port
                                                  <input
                                                    max="65535"
                                                    min="1"
                                                    required
                                                    type="number"
                                                    value={
                                                      endpointEditForm.port
                                                    }
                                                    onChange={(event) =>
                                                      setEndpointEditForm(
                                                        (current) => ({
                                                          ...current,
                                                          port: Number(
                                                            event.target.value,
                                                          ),
                                                        }),
                                                      )
                                                    }
                                                  />
                                                </label>
                                                <label>
                                                  Path
                                                  <input
                                                    value={
                                                      endpointEditForm.path
                                                    }
                                                    onChange={(event) =>
                                                      setEndpointEditForm(
                                                        (current) => ({
                                                          ...current,
                                                          path: event.target
                                                            .value,
                                                        }),
                                                      )
                                                    }
                                                  />
                                                </label>
                                                <label className="checkbox-label">
                                                  <input
                                                    checked={
                                                      endpointEditForm.enabled
                                                    }
                                                    type="checkbox"
                                                    onChange={(event) =>
                                                      setEndpointEditForm(
                                                        (current) => ({
                                                          ...current,
                                                          enabled:
                                                            event.target
                                                              .checked,
                                                        }),
                                                      )
                                                    }
                                                  />
                                                  Enabled
                                                </label>
                                                <button
                                                  disabled={savingRuntimeEdit}
                                                  type="submit"
                                                >
                                                  Save endpoint
                                                </button>
                                                <button
                                                  type="button"
                                                  onClick={() =>
                                                    setEditingEndpointId("")
                                                  }
                                                >
                                                  Cancel
                                                </button>
                                              </form>
                                            ) : null}
                                          </div>
                                        ))
                                      )}
                                      <h4>Health</h4>
                                      {instanceChecks.length === 0 ? (
                                        <div className="runtime-empty-action">
                                          <p>No health monitoring configured.</p>
                                          <button
                                            type="button"
                                            onClick={() =>
                                              startConfigureHealth(
                                                instance,
                                                primaryEndpointForInstance(
                                                  instanceEndpoints,
                                                  instance.id,
                                                ),
                                              )
                                            }
                                          >
                                            Configure health
                                          </button>
                                        </div>
                                      ) : (
                                        instanceChecks.map((check) => (
                                          <div
                                            className="runtime-child-row"
                                            key={check.id}
                                          >
                                            <strong>
                                              {formatCheckType(check.type)
                                                .toUpperCase()}{" "}
                                              {check.name} · every{" "}
                                              {check.intervalSeconds}s
                                            </strong>
                                            <button
                                              type="button"
                                              onClick={() =>
                                                startEditHealthCheck(check)
                                              }
                                            >
                                              Edit health check
                                            </button>
                                            <button
                                              type="button"
                                              onClick={() =>
                                                handleRunHealthCheck(check.id)
                                              }
                                            >
                                              Run check
                                            </button>
                                            <button
                                              className="button-danger"
                                              disabled={savingHealthCheck}
                                              type="button"
                                              onClick={() =>
                                                handleDeleteHealthCheck(check)
                                              }
                                            >
                                              Delete
                                            </button>
                                            {editingHealthCheckId ===
                                            check.id ? (
                                              <form
                                                className="inline-edit-form runtime-edit-form"
                                                onSubmit={
                                                  handleUpdateHealthCheck
                                                }
                                              >
                                                <label>
                                                  Interval seconds
                                                  <input
                                                    min="1"
                                                    type="number"
                                                    value={
                                                      healthEditForm.intervalSeconds
                                                    }
                                                    onChange={(event) =>
                                                      setHealthEditForm(
                                                        (current) => ({
                                                          ...current,
                                                          intervalSeconds:
                                                            Number(
                                                              event.target
                                                                .value,
                                                            ),
                                                        }),
                                                      )
                                                    }
                                                  />
                                                </label>
                                                <label>
                                                  Timeout seconds
                                                  <input
                                                    min="1"
                                                    type="number"
                                                    value={
                                                      healthEditForm.timeoutSeconds
                                                    }
                                                    onChange={(event) =>
                                                      setHealthEditForm(
                                                        (current) => ({
                                                          ...current,
                                                          timeoutSeconds:
                                                            Number(
                                                              event.target
                                                                .value,
                                                            ),
                                                        }),
                                                      )
                                                    }
                                                  />
                                                </label>
                                                <label className="checkbox-label">
                                                  <input
                                                    checked={
                                                      healthEditForm.enabled
                                                    }
                                                    type="checkbox"
                                                    onChange={(event) =>
                                                      setHealthEditForm(
                                                        (current) => ({
                                                          ...current,
                                                          enabled:
                                                            event.target
                                                              .checked,
                                                        }),
                                                      )
                                                    }
                                                  />
                                                  Enabled
                                                </label>
                                                <button
                                                  disabled={savingHealthCheck}
                                                  type="submit"
                                                >
                                                  Save health check
                                                </button>
                                                <button
                                                  type="button"
                                                  onClick={() =>
                                                    setEditingHealthCheckId("")
                                                  }
                                                >
                                                  Cancel
                                                </button>
                                              </form>
                                            ) : null}
                                          </div>
                                        ))
                                      )}
                                      {editingHealthCheckId === "new" &&
                                      healthForm.instanceId === instance.id ? (
                                        <form
                                          className="inline-edit-form runtime-edit-form"
                                          onSubmit={handleUpdateHealthCheck}
                                        >
                                          <label>
                                            Check name
                                            <input
                                              required
                                              value={healthForm.name}
                                              onChange={(event) =>
                                                setHealthForm((current) => ({
                                                  ...current,
                                                  name: event.target.value,
                                                }))
                                              }
                                            />
                                          </label>
                                          <label>
                                            Check type
                                            <select
                                              value={healthForm.type}
                                              onChange={(event) =>
                                                setHealthForm((current) => ({
                                                  ...current,
                                                  type: event.target.value,
                                                }))
                                              }
                                            >
                                              <option value="HEALTH_CHECK_TYPE_HTTP">
                                                HTTP
                                              </option>
                                              <option value="HEALTH_CHECK_TYPE_HTTPS">
                                                HTTPS
                                              </option>
                                              <option value="HEALTH_CHECK_TYPE_GRPC">
                                                gRPC
                                              </option>
                                              <option value="HEALTH_CHECK_TYPE_TCP">
                                                TCP
                                              </option>
                                              <option value="HEALTH_CHECK_TYPE_UDP">
                                                UDP
                                              </option>
                                            </select>
                                          </label>
                                          <label>
                                            Interval seconds
                                            <input
                                              min="1"
                                              type="number"
                                              value={
                                                healthForm.intervalSeconds
                                              }
                                              onChange={(event) =>
                                                setHealthForm((current) => ({
                                                  ...current,
                                                  intervalSeconds: Number(
                                                    event.target.value,
                                                  ),
                                                }))
                                              }
                                            />
                                          </label>
                                          <label>
                                            Timeout seconds
                                            <input
                                              min="1"
                                              type="number"
                                              value={healthForm.timeoutSeconds}
                                              onChange={(event) =>
                                                setHealthForm((current) => ({
                                                  ...current,
                                                  timeoutSeconds: Number(
                                                    event.target.value,
                                                  ),
                                                }))
                                              }
                                            />
                                          </label>
                                          <button
                                            disabled={savingHealthCheck}
                                            type="submit"
                                          >
                                            Configure health
                                          </button>
                                          <button
                                            type="button"
                                            onClick={() =>
                                              setEditingHealthCheckId("")
                                            }
                                          >
                                            Cancel
                                          </button>
                                        </form>
                                      ) : null}
                                    </div>
                                  </article>
                                );
                              })
                            )}
                          </section>
                        );
                      })
                    )}
                  </div>
                ) : null}

                {serviceTab === "health" ? (
                  <div className="service-tab-panel">
                    <div className="availability-grid">
                      <AvailabilityCard
                        label="24 hours"
                        summary={serviceAvailability.availability24h}
                      />
                      <AvailabilityCard
                        label="7 days"
                        summary={serviceAvailability.availability7d}
                      />
                      <AvailabilityCard
                        label="30 days"
                        summary={serviceAvailability.availability30d}
                      />
                    </div>
                    <div className="scoped-list">
                      <h3>Health checks</h3>
                      {selectedServiceHealthChecks.length === 0 ? (
                        <EmptyState
                          title="No health checks"
                          description="No monitoring configuration is attached to this service runtime yet."
                        />
                      ) : (
                        selectedServiceHealthChecks.map((check) => (
                          <div className="health-result-row" key={check.id}>
                            <strong>{check.name}</strong>
                            <span>{formatCheckType(check.type)}</span>
                            <span>{check.intervalSeconds}s</span>
                            <StatusBadge
                              status={check.enabled ? "enabled" : "disabled"}
                            />
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                ) : null}

                {serviceTab === "incidents" ? (
                  <div className="service-tab-panel scoped-list">
                    {filteredIncidents.filter(
                      (incident) => incident.serviceId === selectedService.id,
                    ).length === 0 ? (
                      <EmptyState
                        title="No service incidents"
                        description="No incidents match this service and the current incident filters."
                      />
                    ) : (
                      filteredIncidents
                        .filter(
                          (incident) =>
                            incident.serviceId === selectedService.id,
                        )
                        .map((incident) => (
                          <div className="incident-row" key={incident.id}>
                            <div>
                              <StatusBadge
                                status={formatIncidentState(incident.state)}
                              />
                              <span>
                                {incident.reason || "No reason recorded"}
                              </span>
                            </div>
                            <span>{incident.instanceId}</span>
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
                ) : null}

                {serviceTab === "events" ? (
                  <div className="service-tab-panel scoped-list">
                    {selectedServiceEvents.length === 0 ? (
                      <EmptyState
                        title="No service events"
                        description="No recent changes have been recorded for this service in the selected scope."
                      />
                    ) : (
                      selectedServiceEvents.map((event) => (
                        <div className="event-row timeline-row" key={event.id}>
                          <time title={formatTimestamp(event.timestamp)}>
                            {formatActivityTimestamp(event.timestamp)}
                          </time>
                          <strong>{formatEventType(event.type)}</strong>
                          <span>{event.message}</span>
                          <span>
                            {event.resourceType}:{event.resourceId}
                          </span>
                          <span>{event.actor}</span>
                        </div>
                      ))
                    )}
                  </div>
                ) : null}
              </>
            ) : (
              <EmptyState
                title="Select a service"
                description="Choose a service from the catalog to inspect runtime, health, incidents, and events in one workspace."
              />
            )}
          </div>

          {/*
            <div className="legacy-service-stack" aria-hidden="true">

            <div className="panel detail-panel">
              <div className="panel-heading">
                <h2>Service Detail</h2>
              </div>
              {selectedService ? (
                <dl className="detail-list">
                  <dt>ID</dt>
                  <dd>{selectedService.id}</dd>
                  <dt>Name</dt>
                  <dd>{selectedService.name}</dd>
                  <dt>Display name</dt>
                  <dd>{selectedService.displayName}</dd>
                  <dt>Description</dt>
                  <dd>{selectedService.description || "None"}</dd>
                  <dt>Tags</dt>
                  <dd>{formatMap(selectedService.tags)}</dd>
                  <dt>Metadata</dt>
                  <dd>{formatMap(selectedService.metadata)}</dd>
                  <dt>Status</dt>
                  <dd>
                    <StatusBadge
                      status={serviceOperationalStatus(
                        selectedService,
                        incidents,
                      )}
                    />
                  </dd>
                  <dt>Deployments</dt>
                  <dd>{selectedServiceDeployments.length}</dd>
                  <dt>Instances</dt>
                  <dd>{selectedServiceInstances.length}</dd>
                </dl>
              ) : (
                <EmptyState
                  title="Select a service"
                  description="Choose a service from the table to inspect its runtime topology, health checks, incidents, and events."
                />
              )}
            </div>

            <div className="panel service-wide-panel">
              <div className="panel-heading">
                <h2>Environment Overview</h2>
                <span>
                  {selectedService ? selectedService.name : "No service"}
                </span>
              </div>
              <div className="table">
                {!selectedService || selectedServiceDeployments.length === 0 ? (
                  <EmptyState
                    title="No runtime environments"
                    description="This service has not been deployed into the selected environment scope yet."
                  />
                ) : (
                  selectedServiceDeployments.map((deployment) => {
                    const deploymentInstances = instances.filter(
                      (instance) => instance.deploymentId === deployment.id,
                    );
                    const hasIncident = incidents.some(
                      (incident) =>
                        incident.deploymentId === deployment.id &&
                        incident.state === "INCIDENT_STATE_OPEN",
                    );
                    return (
                      <div
                        className="environment-overview-row"
                        key={deployment.id}
                      >
                        <strong>
                          {environmentName(
                            environments,
                            deployment.environmentId,
                          )}
                        </strong>
                        <span>
                          {pluralize(deploymentInstances.length, "instance")}
                        </span>
                        <StatusBadge status={hasIncident ? "degraded" : "healthy"} />
                        <span>{deployment.tags?.version ?? "n/a"}</span>
                      </div>
                    );
                  })
                )}
              </div>
            </div>

            <RuntimeRegistrationWizard
              environments={environments}
              form={registrationForm}
              registrationSuccess={registrationSuccess}
              saving={savingRegistration}
              selectedEnvironment={selectedEnvironment}
              selectedService={selectedService}
              selectedServiceId={selectedServiceId}
              services={services}
              step={registrationStep}
              onAddEndpoint={addRegistrationEndpoint}
              onBack={() => setRegistrationStep((step) => Math.max(step - 1, 0))}
              onRemoveEndpoint={removeRegistrationEndpoint}
              onSetPrimaryEndpoint={setPrimaryRegistrationEndpoint}
              onServiceChange={setSelectedServiceId}
              onStepChange={setRegistrationStep}
              onSubmit={handleRegisterInstance}
              onUpdateEndpoint={updateRegistrationEndpoint}
              setForm={setRegistrationForm}
            />

            <div className="panel service-wide-panel">
              <div className="panel-heading">
                <h2>Runtime Topology</h2>
                <span>
                  Service / Deployment / Instance / Endpoint / Health
                </span>
              </div>
              <RuntimeTopology
                deployments={selectedServiceDeployments}
                endpoints={selectedServiceEndpoints}
                environments={environments}
                healthChecks={selectedServiceHealthChecks}
                incidents={incidents}
                instances={selectedServiceInstances}
                serviceName={selectedService?.name ?? "This service"}
                onRegisterRuntime={focusRegistrationForm}
                onRunHealthCheck={handleRunHealthCheck}
              />
            </div>

            <div className="panel service-wide-panel">
              <div className="panel-heading">
                <h2>Availability</h2>
                <span>
                  {selectedService ? "Selected service" : "No service"}
                </span>
              </div>
              <div className="availability-grid">
                <AvailabilityCard
                  label="24h"
                  summary={serviceAvailability.availability24h}
                />
                <AvailabilityCard
                  label="7d"
                  summary={serviceAvailability.availability7d}
                />
                <AvailabilityCard
                  label="30d"
                  summary={serviceAvailability.availability30d}
                />
              </div>
            </div>

            <div className="panel">
              <div className="panel-heading">
                <h2>Instances</h2>
                <span>
                  {selectedEnvironment
                    ? `${selectedEnvironmentInstances.length} in ${selectedEnvironment.name} / ${selectedServiceInstances.length} total`
                    : `${selectedServiceInstances.length} total`}
                </span>
              </div>
              <div className="table">
                {selectedServiceInstances.length === 0 ? (
                  <div className="empty-action">
                    <strong>
                      {selectedEnvironmentDeployment
                        ? "No runtime instances registered."
                        : `${selectedService?.name ?? "This service"} has no runtime registration in ${selectedEnvironment?.name ?? "this environment"}.`}
                    </strong>
                    <p>
                      Alauda knows this service exists, but it does not yet know
                      where this service is running.
                    </p>
                    <button type="button" onClick={focusRegistrationForm}>
                      {selectedEnvironmentDeployment
                        ? "Register first instance"
                        : `Register runtime in ${selectedEnvironment?.name ?? "environment"}`}
                    </button>
                  </div>
                ) : (
                  selectedServiceInstances.map((instance) => (
                    <div className="instance-row" key={instance.id}>
                      <div>
                        <strong>{instance.name}</strong>
                        <span>{instance.id}</span>
                      </div>
                      <span>{instance.address}</span>
                      <span>
                        {formatEndpointSummary(
                          primaryEndpointForInstance(
                            selectedServiceEndpoints,
                            instance.id,
                          ),
                        )}
                      </span>
                      <StatusBadge
                        status={instance.enabled ? "enabled" : "disabled"}
                      />
                      <span>{formatTimestamp(instance.lastSeenAt)}</span>
                      <div className="row-actions">
                        <button
                          type="button"
                          onClick={() => startEditInstance(instance)}
                        >
                          Edit
                        </button>
                        <button
                          disabled={savingRuntimeEdit}
                          type="button"
                          onClick={() => handleDeleteInstance(instance)}
                        >
                          Delete
                        </button>
                      </div>
                      {editingInstanceId === instance.id ? (
                        <form
                          className="inline-edit-form"
                          onSubmit={handleUpdateInstance}
                        >
                          <label>
                            Address
                            <input
                              required
                              value={instanceEditForm.address}
                              onChange={(event) =>
                                setInstanceEditForm((current) => ({
                                  ...current,
                                  address: event.target.value,
                                }))
                              }
                            />
                          </label>
                          <label>
                            Description
                            <input
                              value={instanceEditForm.description}
                              onChange={(event) =>
                                setInstanceEditForm((current) => ({
                                  ...current,
                                  description: event.target.value,
                                }))
                              }
                            />
                          </label>
                          <label className="checkbox-label">
                            <input
                              checked={instanceEditForm.enabled}
                              type="checkbox"
                              onChange={(event) =>
                                setInstanceEditForm((current) => ({
                                  ...current,
                                  enabled: event.target.checked,
                                }))
                              }
                            />
                            Enabled
                          </label>
                          <button disabled={savingRuntimeEdit} type="submit">
                            Save instance
                          </button>
                          <button
                            type="button"
                            onClick={() => setEditingInstanceId("")}
                          >
                            Cancel
                          </button>
                        </form>
                      ) : null}
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="panel">
              <div className="panel-heading">
                <h2>Endpoints</h2>
                <span>{selectedServiceEndpoints.length} total</span>
              </div>
              <div className="table">
                {selectedServiceEndpoints.length === 0 ? (
                  <EmptyState
                    title="No endpoints"
                    description="Instances exist, but no protocol/path targets are attached to them yet."
                  />
                ) : (
                  selectedServiceEndpoints.map((endpoint) => {
                    const instance = selectedServiceInstanceById.get(
                      endpoint.instanceId,
                    );
                    return (
                      <div className="endpoint-row" key={endpoint.id}>
                        <div>
                          <strong>{endpoint.name}</strong>
                          <span>{endpoint.id}</span>
                        </div>
                        <span>{formatEndpointUrl(endpoint, instance)}</span>
                        <span>
                          {endpoint.primary ? "Primary " : ""}
                          {formatProtocol(endpoint.protocol)}
                        </span>
                        <StatusBadge
                          status={endpoint.enabled ? "enabled" : "disabled"}
                        />
                        <div className="row-actions">
                          <button
                            type="button"
                            onClick={() => startEditEndpoint(endpoint)}
                          >
                            Edit
                          </button>
                          <button
                            disabled={savingRuntimeEdit}
                            type="button"
                            onClick={() => handleDeleteEndpoint(endpoint)}
                          >
                            Delete
                          </button>
                        </div>
                        {editingEndpointId === endpoint.id ? (
                          <form
                            className="inline-edit-form"
                            onSubmit={handleUpdateEndpoint}
                          >
                            <label>
                              Name
                              <input
                                required
                                value={endpointEditForm.name}
                                onChange={(event) =>
                                  setEndpointEditForm((current) => ({
                                    ...current,
                                    name: event.target.value,
                                  }))
                                }
                              />
                            </label>
                            <label>
                              Protocol
                              <select
                                value={endpointEditForm.protocol}
                                onChange={(event) =>
                                  setEndpointEditForm((current) => ({
                                    ...current,
                                    protocol: event.target.value,
                                  }))
                                }
                              >
                                <option value="PROTOCOL_HTTP">HTTP</option>
                                <option value="PROTOCOL_HTTPS">HTTPS</option>
                                <option value="PROTOCOL_GRPC">gRPC</option>
                                <option value="PROTOCOL_TCP">TCP</option>
                                <option value="PROTOCOL_UDP">UDP</option>
                              </select>
                            </label>
                            <label>
                              Port
                              <input
                                max="65535"
                                min="1"
                                required
                                type="number"
                                value={endpointEditForm.port}
                                onChange={(event) =>
                                  setEndpointEditForm((current) => ({
                                    ...current,
                                    port: Number(event.target.value),
                                  }))
                                }
                              />
                            </label>
                            <label>
                              Path
                              <input
                                value={endpointEditForm.path}
                                onChange={(event) =>
                                  setEndpointEditForm((current) => ({
                                    ...current,
                                    path: event.target.value,
                                  }))
                                }
                              />
                            </label>
                            <label className="checkbox-label">
                              <input
                                checked={endpointEditForm.primary}
                                type="checkbox"
                                onChange={(event) =>
                                  setEndpointEditForm((current) => ({
                                    ...current,
                                    primary: event.target.checked,
                                  }))
                                }
                              />
                              Primary
                            </label>
                            <label className="checkbox-label">
                              <input
                                checked={endpointEditForm.enabled}
                                type="checkbox"
                                onChange={(event) =>
                                  setEndpointEditForm((current) => ({
                                    ...current,
                                    enabled: event.target.checked,
                                  }))
                                }
                              />
                              Enabled
                            </label>
                            <button disabled={savingRuntimeEdit} type="submit">
                              Save endpoint
                            </button>
                            <button
                              type="button"
                              onClick={() => setEditingEndpointId("")}
                            >
                              Cancel
                            </button>
                          </form>
                        ) : null}
                      </div>
                    );
                  })
                )}
              </div>
            </div>

            <div className="panel">
              <div className="panel-heading">
                <h2>Health Checks</h2>
                <span>{selectedServiceHealthChecks.length} total</span>
              </div>
              <div className="table">
                {selectedServiceHealthChecks.length === 0 ? (
                  <EmptyState
                    title="No health checks"
                    description="No monitoring configuration is attached to this service runtime yet."
                  />
                ) : (
                  selectedServiceHealthChecks.map((check) => (
                    <div className="health-result-row" key={check.id}>
                      <strong>{check.name}</strong>
                      <span>{formatCheckType(check.type)}</span>
                      <span>{check.intervalSeconds}s</span>
                      <StatusBadge
                        status={check.enabled ? "enabled" : "disabled"}
                      />
                    </div>
                  ))
                )}
              </div>
            </div>

            <div className="panel">
              <div className="panel-heading">
                <h2>Recent Events</h2>
                <span>{selectedServiceEvents.length} total</span>
              </div>
              <div className="table">
                {selectedServiceEvents.length === 0 ? (
                  <EmptyState
                    title="No service events"
                    description="No recent changes have been recorded for this service in the selected scope."
                  />
                ) : (
                  selectedServiceEvents.slice(0, 8).map((event) => (
                    <div className="timeline-row" key={event.id}>
                      <time title={formatTimestamp(event.timestamp)}>
                        {formatActivityTimestamp(event.timestamp)}
                      </time>
                      <strong>{formatEventType(event.type)}</strong>
                      <span>{event.message}</span>
                      <small>{event.resourceType}</small>
                    </div>
                  ))
                )}
              </div>
            </div>

            <form className="panel form-panel" onSubmit={handleCreateService}>
              <div className="panel-heading">
                <h2>Create Service</h2>
              </div>
              <label>
                Name
                <input
                  required
                  value={serviceForm.name}
                  onChange={(event) =>
                    setServiceForm((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                />
              </label>
              <label>
                Display name
                <input
                  required
                  value={serviceForm.displayName}
                  onChange={(event) =>
                    setServiceForm((current) => ({
                      ...current,
                      displayName: event.target.value,
                    }))
                  }
                />
              </label>
              <label>
                Description
                <textarea
                  value={serviceForm.description}
                  onChange={(event) =>
                    setServiceForm((current) => ({
                      ...current,
                      description: event.target.value,
                    }))
                  }
                />
              </label>
              <button disabled={savingService} type="submit">
                {savingService ? "Creating" : "Create service"}
              </button>
            </form>
            </div>
            */}

          {showCreateService ? (
            <div
              aria-labelledby="create-service-title"
              aria-modal="true"
              className="modal-backdrop"
              role="dialog"
            >
              <form
                className="modal form-panel drawer-form"
                onSubmit={handleCreateService}
              >
                <div className="modal-header">
                  <h2 id="create-service-title">Create service</h2>
                  <button
                    aria-label="Close"
                    className="icon-button"
                    type="button"
                    onClick={() => setShowCreateService(false)}
                  >
                    x
                  </button>
                </div>
                <label>
                  Name
                  <input
                    required
                    value={serviceForm.name}
                    onChange={(event) =>
                      setServiceForm((current) => ({
                        ...current,
                        name: event.target.value,
                      }))
                    }
                  />
                </label>
                <label>
                  Display name
                  <input
                    required
                    value={serviceForm.displayName}
                    onChange={(event) =>
                      setServiceForm((current) => ({
                        ...current,
                        displayName: event.target.value,
                      }))
                    }
                  />
                </label>
                <label>
                  Description
                  <textarea
                    value={serviceForm.description}
                    onChange={(event) =>
                      setServiceForm((current) => ({
                        ...current,
                        description: event.target.value,
                      }))
                    }
                  />
                </label>
                <div className="drawer-actions">
                  <button
                    type="button"
                    onClick={() => setShowCreateService(false)}
                  >
                    Cancel
                  </button>
                  <button disabled={savingService} type="submit">
                    {savingService ? "Creating" : "Create service"}
                  </button>
                </div>
              </form>
            </div>
          ) : null}

          {editingServiceId ? (
            <div
              aria-labelledby="edit-service-title"
              aria-modal="true"
              className="modal-backdrop"
              role="dialog"
            >
              <form
                className="modal form-panel drawer-form"
                onSubmit={handleUpdateService}
              >
                <div className="modal-header">
                  <div>
                    <h2 id="edit-service-title">Edit service</h2>
                    <span>{selectedService?.name}</span>
                  </div>
                  <button
                    aria-label="Close"
                    className="icon-button"
                    type="button"
                    onClick={() => setEditingServiceId("")}
                  >
                    x
                  </button>
                </div>
                <label>
                  Display name
                  <input
                    required
                    value={serviceEditForm.displayName}
                    onChange={(event) =>
                      setServiceEditForm((current) => ({
                        ...current,
                        displayName: event.target.value,
                      }))
                    }
                  />
                </label>
                <label>
                  Description
                  <textarea
                    value={serviceEditForm.description}
                    onChange={(event) =>
                      setServiceEditForm((current) => ({
                        ...current,
                        description: event.target.value,
                      }))
                    }
                  />
                </label>
                <div className="drawer-actions">
                  <button
                    type="button"
                    onClick={() => setEditingServiceId("")}
                  >
                    Cancel
                  </button>
                  <button disabled={savingService} type="submit">
                    {savingService ? "Saving" : "Save service"}
                  </button>
                </div>
              </form>
            </div>
          ) : null}

          {showAddRuntime && selectedService ? (
            <div
              aria-labelledby="add-instance-title"
              aria-modal="true"
              className="modal-backdrop"
              role="dialog"
            >
              <form
                className="modal form-panel drawer-form"
                onSubmit={handleRegisterInstance}
              >
                <div className="modal-header">
                  <div>
                    <h2 id="add-instance-title">
                      {selectedEnvironmentDeployment
                        ? `Add instance to ${selectedService.displayName || selectedService.name}`
                        : "Add runtime"}
                    </h2>
                    <span>
                      Service
                    </span>
                  </div>
                  <button
                    aria-label="Close"
                    className="icon-button"
                    type="button"
                    onClick={() => setShowAddRuntime(false)}
                  >
                    x
                  </button>
                </div>
                <div className="context-strip">
                  <span>Service</span>
                  <strong>{selectedService.displayName || selectedService.name}</strong>
                </div>
                <label>
                  Environment
                  <select
                    required
                    value={registrationForm.environmentId}
                    onChange={(event) =>
                      setRegistrationForm((current) => ({
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
                <div className="form-fields-grid">
                  <label>
                    Instance name
                    <input
                      required
                      data-registration-instance-name
                      value={registrationForm.instanceName}
                      onChange={(event) =>
                        setRegistrationForm((current) => ({
                          ...current,
                          instanceName: event.target.value,
                        }))
                      }
                    />
                  </label>
                  <label>
                    Address
                    <input
                      required
                      value={registrationForm.address}
                      onChange={(event) =>
                        setRegistrationForm((current) => ({
                          ...current,
                          address: event.target.value,
                        }))
                      }
                    />
                  </label>
                </div>
                <label>
                  Description
                  <input
                    value={registrationForm.description}
                    onChange={(event) =>
                      setRegistrationForm((current) => ({
                        ...current,
                        description: event.target.value,
                      }))
                    }
                  />
                </label>
                <section className="form-section">
                  <div className="form-section-heading">
                    <span>Endpoint</span>
                    <strong>Required</strong>
                  </div>
                  <p>An instance must have at least one endpoint.</p>
                  <div className="form-fields-grid">
                    <label>
                      Name
                      <input
                        required
                        value={registrationForm.endpoints[0]?.name ?? ""}
                        onChange={(event) =>
                          updateRegistrationEndpoint(0, {
                            name: event.target.value,
                          })
                        }
                      />
                    </label>
                    <label>
                      Protocol
                      <select
                        value={
                          registrationForm.endpoints[0]?.protocol ??
                          "PROTOCOL_HTTP"
                        }
                        onChange={(event) =>
                          updateRegistrationEndpoint(0, {
                            protocol: event.target.value,
                          })
                        }
                      >
                        <option value="PROTOCOL_HTTP">HTTP</option>
                        <option value="PROTOCOL_HTTPS">HTTPS</option>
                        <option value="PROTOCOL_GRPC">gRPC</option>
                        <option value="PROTOCOL_TCP">TCP</option>
                        <option value="PROTOCOL_UDP">UDP</option>
                      </select>
                    </label>
                    <label>
                      Port
                      <input
                        max="65535"
                        min="1"
                        required
                        type="number"
                        value={registrationForm.endpoints[0]?.port ?? 8080}
                        onChange={(event) =>
                          updateRegistrationEndpoint(0, {
                            port: Number(event.target.value),
                          })
                        }
                      />
                    </label>
                    <label>
                      Path
                      <input
                        value={registrationForm.endpoints[0]?.path ?? "/"}
                        onChange={(event) =>
                          updateRegistrationEndpoint(0, {
                            path: event.target.value,
                          })
                        }
                      />
                    </label>
                  </div>
                </section>
                <section className="form-section">
                  <button
                    aria-controls="health-monitoring-fields"
                    aria-expanded={showRuntimeHealth}
                    className="accordion-button"
                    type="button"
                    onClick={() => {
                      setShowRuntimeHealth((value) => {
                        const next = !value;
                        setRegistrationForm((current) => ({
                          ...current,
                          configureHealth: next,
                        }));
                        return next;
                      });
                    }}
                  >
                    <span
                      aria-hidden="true"
                      className={
                        showRuntimeHealth
                          ? "accordion-chevron expanded"
                          : "accordion-chevron"
                      }
                    >
                      &gt;
                    </span>
                    <span>Health monitoring</span>
                    <strong>Optional</strong>
                  </button>
                  {showRuntimeHealth ? (
                  <div className="form-fields-grid">
                    <label>
                      Check name
                      <input
                        value={registrationForm.healthName}
                        onChange={(event) =>
                          setRegistrationForm((current) => ({
                            ...current,
                            healthName: event.target.value,
                          }))
                        }
                      />
                    </label>
                    <label>
                      Check type
                      <select
                        value={registrationForm.healthType}
                        onChange={(event) =>
                          setRegistrationForm((current) => ({
                            ...current,
                            healthType: event.target.value,
                          }))
                        }
                      >
                        <option value="HEALTH_CHECK_TYPE_HTTP">HTTP</option>
                        <option value="HEALTH_CHECK_TYPE_HTTPS">HTTPS</option>
                        <option value="HEALTH_CHECK_TYPE_GRPC">gRPC</option>
                        <option value="HEALTH_CHECK_TYPE_TCP">TCP</option>
                        <option value="HEALTH_CHECK_TYPE_UDP">UDP</option>
                      </select>
                    </label>
                    <label>
                      Interval
                      <input
                        min="1"
                        type="number"
                        value={registrationForm.healthIntervalSeconds}
                        onChange={(event) =>
                          setRegistrationForm((current) => ({
                            ...current,
                            healthIntervalSeconds: Number(event.target.value),
                          }))
                        }
                      />
                    </label>
                    <label>
                      Timeout
                      <input
                        min="1"
                        type="number"
                        value={registrationForm.healthTimeoutSeconds}
                        onChange={(event) =>
                          setRegistrationForm((current) => ({
                            ...current,
                            healthTimeoutSeconds: Number(event.target.value),
                          }))
                        }
                      />
                    </label>
                  </div>
                  ) : null}
                </section>
                <div className="drawer-actions">
                  <button
                    type="button"
                    onClick={() => setShowAddRuntime(false)}
                  >
                    Cancel
                  </button>
                  <button disabled={savingRegistration} type="submit">
                    {savingRegistration
                      ? "Adding"
                      : selectedEnvironmentDeployment
                        ? "Add instance"
                        : "Add runtime"}
                  </button>
                </div>
              </form>
            </div>
          ) : null}
        </section>
      ) : activeView === "environments" ? (
        <section className="content-grid environments-grid">
          <PageHeader
            title="Environments"
            context={`${environments.length} environments`}
            description="Runtime scopes used to filter services, deployments, incidents, alerts, and events."
          />
          <div className="panel">
            <div className="panel-heading">
              <h2>Environments</h2>
              <span>
                {loading ? "Loading" : `${environments.length} total`}
              </span>
            </div>
            <div className="table">
              {environments.map((environment) => (
                <div className="table-row" key={environment.id}>
                  <strong>{environment.name}</strong>
                  <span>{environment.key}</span>
                  <span>{environment.tier || "untiered"}</span>
                  <StatusBadge
                    status={environment.enabled ? "enabled" : "disabled"}
                  />
                </div>
              ))}
            </div>
          </div>

          <form className="panel form-panel" onSubmit={handleCreateEnvironment}>
            <div className="panel-heading">
              <h2>Create Environment</h2>
            </div>
            <label>
              Key
              <input
                required
                value={environmentForm.key}
                onChange={(event) =>
                  setEnvironmentForm((current) => ({
                    ...current,
                    key: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Name
              <input
                required
                value={environmentForm.name}
                onChange={(event) =>
                  setEnvironmentForm((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Tier
              <input
                value={environmentForm.tier}
                onChange={(event) =>
                  setEnvironmentForm((current) => ({
                    ...current,
                    tier: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Description
              <textarea
                value={environmentForm.description}
                onChange={(event) =>
                  setEnvironmentForm((current) => ({
                    ...current,
                    description: event.target.value,
                  }))
                }
              />
            </label>
            <button disabled={savingEnvironment} type="submit">
              {savingEnvironment ? "Creating" : "Create environment"}
            </button>
          </form>
        </section>
      ) : activeView === "health" ? (
        <section className="content-grid">
          <PageHeader
            title="Health"
            context={`${filteredHealthInstances.length} instances · ${healthChecks.filter((check) => check.enabled).length} enabled checks`}
            description="Monitor unhealthy instances, recent check results, and manual execution from the current environment scope."
          />
          <div className="panel service-wide-panel">
            <div className="panel-heading">
              <h2>Global Health</h2>
              <span>{selectedEnvironmentId ? "Filtered" : "Global"}</span>
            </div>
            <div className="metrics-grid">
              <MetricCard
                label="Healthy"
                value={
                  healthStates.filter(
                    (state) => state.currentState === "HEALTH_STATE_HEALTHY",
                  ).length
                }
                detail={`${healthStates.length} states loaded`}
              />
              <MetricCard
                label="Unhealthy"
                value={
                  healthStates.filter(
                    (state) => state.currentState === "HEALTH_STATE_UNHEALTHY",
                  ).length
                }
                detail={`${incidents.filter((incident) => incident.state === "INCIDENT_STATE_OPEN").length} open incidents`}
              />
              <MetricCard
                label="Unknown"
                value={Math.max(instances.length - healthStates.length, 0)}
                detail={`${instances.length} instances`}
              />
              <MetricCard
                label="Checks"
                value={healthChecks.length}
                detail={`${healthChecks.filter((check) => check.enabled).length} enabled`}
              />
            </div>
          </div>

          <div className="panel service-wide-panel">
            <div className="panel-heading">
              <h2>Instance Health</h2>
              <span>{filteredHealthInstances.length} shown</span>
            </div>
            <div className="filter-bar">
              <label>
                Status
                <select
                  value={healthStatusFilter}
                  onChange={(event) =>
                    setHealthStatusFilter(event.target.value)
                  }
                >
                  <option value="all">All statuses</option>
                  <option value="healthy">Healthy</option>
                  <option value="unhealthy">Unhealthy</option>
                  <option value="unknown">Unknown</option>
                </select>
              </label>
            </div>
            <div className="table">
              {filteredHealthInstances.length === 0 ? (
                <EmptyState
                  title="No health targets"
                  description="No runtime instances match the selected health filter."
                />
              ) : (
                filteredHealthInstances.map((instance) => {
                  const state = healthStateByInstanceId.get(instance.id);
                  return (
                    <div className="instance-row" key={instance.id}>
                      <div>
                        <strong>{instance.name}</strong>
                        <span>{instance.id}</span>
                      </div>
                      <span>
                        {state?.currentState ? (
                          <StatusBadge
                            status={formatHealthState(state.currentState)}
                          />
                        ) : (
                          "unknown"
                        )}
                      </span>
                      <span>
                        {state
                          ? `${state.consecutiveSuccesses} ok / ${state.consecutiveFailures} fail`
                          : "No checks"}
                      </span>
                      <span>{formatTimestamp(state?.lastCheckTime)}</span>
                    </div>
                  );
                })
              )}
            </div>
          </div>

          <div className="panel">
            <div className="panel-heading">
              <h2>Health Checks</h2>
              <span>
                {loading ? "Loading" : `${healthChecks.length} total`}
              </span>
            </div>
            {healthChecks.length === 0 && !loading ? (
              <EmptyState
                title="No health checks"
                description="Create a health check or register runtime with monitoring enabled to start tracking state."
              />
            ) : (
              <div className="service-list">
                {healthChecks.map((check) => (
                  <button
                    className={
                      check.id === selectedHealthCheckId
                        ? "row selected"
                        : "row"
                    }
                    key={check.id}
                    onClick={() => handleSelectHealthCheck(check.id)}
                    type="button"
                  >
                    <strong>{check.name}</strong>
                    <span>
                      {formatCheckType(check.type)} / {check.instanceId}
                    </span>
                  </button>
                ))}
              </div>
            )}
          </div>

          <div className="panel detail-panel">
            <div className="panel-heading">
              <h2>Health Status</h2>
              <button
                disabled={!selectedHealthCheck || runningHealthCheck}
                onClick={() => handleRunHealthCheck()}
                type="button"
              >
                {runningHealthCheck ? "Running" : "Run check"}
              </button>
            </div>
            {selectedHealthCheck ? (
              <dl className="detail-list">
                <dt>ID</dt>
                <dd>{selectedHealthCheck.id}</dd>
                <dt>Type</dt>
                <dd>{formatCheckType(selectedHealthCheck.type)}</dd>
                <dt>Interval</dt>
                <dd>{selectedHealthCheck.intervalSeconds}s</dd>
                <dt>Timeout</dt>
                <dd>{selectedHealthCheck.timeoutSeconds}s</dd>
                <dt>State</dt>
                <dd>{latestState?.currentState ?? "Not loaded"}</dd>
                <dt>Counters</dt>
                <dd>
                  {latestState
                    ? `${latestState.consecutiveSuccesses} success / ${latestState.consecutiveFailures} failure`
                    : "Not loaded"}
                </dd>
              </dl>
            ) : (
              <EmptyState
                title="Select a health check"
                description="Pick a check to inspect its configuration, counters, and recent execution results."
              />
            )}
          </div>

          <div className="panel">
            <div className="panel-heading">
              <h2>Recent Results</h2>
              <span>{healthResults.length} shown</span>
            </div>
            <div className="table">
              {healthResults.length === 0 ? (
                <EmptyState
                  title="No health results"
                  description="Run this check manually or wait for scheduled execution to record a result."
                />
              ) : (
                healthResults.map((result) => (
                  <div className="health-result-row" key={result.id}>
                    <strong>
                      <StatusBadge
                        status={result.success ? "healthy" : "failed"}
                      />
                    </strong>
                    <span>
                      {result.statusCode || result.errorType || "n/a"}
                    </span>
                    <span>{result.latencyMs ?? 0}ms</span>
                    <span>{formatTimestamp(result.timestamp)}</span>
                  </div>
                ))
              )}
            </div>
          </div>

          <form className="panel form-panel" onSubmit={handleCreateHealthCheck}>
            <div className="panel-heading">
              <h2>Create Health Check</h2>
            </div>
            <label>
              Instance ID
              <input
                required
                value={healthForm.instanceId}
                onChange={(event) =>
                  setHealthForm((current) => ({
                    ...current,
                    instanceId: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Endpoint ID
              <input
                value={healthForm.endpointId}
                onChange={(event) =>
                  setHealthForm((current) => ({
                    ...current,
                    endpointId: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Name
              <input
                required
                value={healthForm.name}
                onChange={(event) =>
                  setHealthForm((current) => ({
                    ...current,
                    name: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Type
              <select
                value={healthForm.type}
                onChange={(event) =>
                  setHealthForm((current) => ({
                    ...current,
                    type: event.target.value,
                  }))
                }
              >
                <option value="HEALTH_CHECK_TYPE_HTTP">HTTP</option>
                <option value="HEALTH_CHECK_TYPE_HTTPS">HTTPS</option>
                <option value="HEALTH_CHECK_TYPE_TCP">TCP</option>
                <option value="HEALTH_CHECK_TYPE_GRPC">gRPC</option>
              </select>
            </label>
            <label>
              Path
              <input
                value={healthForm.path}
                onChange={(event) =>
                  setHealthForm((current) => ({
                    ...current,
                    path: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Expected status
              <input
                value={healthForm.expectedStatus}
                onChange={(event) =>
                  setHealthForm((current) => ({
                    ...current,
                    expectedStatus: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Interval seconds
              <input
                min="1"
                type="number"
                value={healthForm.intervalSeconds}
                onChange={(event) =>
                  setHealthForm((current) => ({
                    ...current,
                    intervalSeconds: Number(event.target.value),
                  }))
                }
              />
            </label>
            <label>
              Timeout seconds
              <input
                min="1"
                type="number"
                value={healthForm.timeoutSeconds}
                onChange={(event) =>
                  setHealthForm((current) => ({
                    ...current,
                    timeoutSeconds: Number(event.target.value),
                  }))
                }
              />
            </label>
            <button disabled={savingHealthCheck} type="submit">
              {savingHealthCheck ? "Creating" : "Create health check"}
            </button>
          </form>
        </section>
      ) : activeView === "incidents" ? (
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
                  <button
                    onClick={() => setSelectedIncidentId("")}
                    type="button"
                  >
                    Close
                  </button>
                </div>
                <dl className="detail-list">
                  <dt>ID</dt>
                  <dd>{selectedIncident.id}</dd>
                  <dt>State</dt>
                  <dd>
                    <StatusBadge
                      status={formatIncidentState(selectedIncident.state)}
                    />
                  </dd>
                  <dt>Reason</dt>
                  <dd>{selectedIncident.reason || "None"}</dd>
                  <dt>Impact</dt>
                  <dd>{selectedIncident.impactSummary || "None"}</dd>
                  <dt>Service</dt>
                  <dd>
                    <ResourceLink
                      onClick={() => {
                        setSelectedServiceId(selectedIncident.serviceId);
                        setSelectedIncidentId("");
                        setActiveView("services");
                      }}
                    >
                      {selectedIncident.serviceId}
                    </ResourceLink>
                  </dd>
                  <dt>Instance</dt>
                  <dd>{selectedIncident.instanceId}</dd>
                  <dt>Opened</dt>
                  <dd>{formatTimestamp(selectedIncident.openedAt)}</dd>
                  <dt>Resolved</dt>
                  <dd>{formatTimestamp(selectedIncident.resolvedAt)}</dd>
                  <dt>Duration</dt>
                  <dd>{formatDuration(selectedIncident.durationSeconds)}</dd>
                  <dt>Metadata</dt>
                  <dd>{formatMap(selectedIncident.metadata)}</dd>
                </dl>
              </div>
            </div>
          ) : null}
        </section>
      ) : activeView === "alerts" ? (
        <section className="content-grid alerts-grid">
          <PageHeader
            title="Alerts"
            context={`${alertPolicies.length} policies · ${notificationChannels.length} channels`}
            description="Keep notification channels separate from policies so routing and triggers stay clear."
          />
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
              Deployment ID
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
        </section>
      ) : activeView === "security" ? (
        <section className="content-grid alerts-grid">
          <PageHeader
            title="Security"
            context={authToken ? "Authenticated session" : "No active token"}
            description="Manage users, API tokens, and session access without exposing secrets in normal lists."
          />
          <form className="panel form-panel" onSubmit={handleLogin}>
            <div className="panel-heading">
              <h2>Session</h2>
              <span>{authToken ? "Token active" : "Not signed in"}</span>
            </div>
            <label>
              Username
              <input
                value={loginForm.username}
                onChange={(event) =>
                  setLoginForm((current) => ({
                    ...current,
                    username: event.target.value,
                  }))
                }
              />
            </label>
            <label>
              Password
              <input
                type="password"
                value={loginForm.password}
                onChange={(event) =>
                  setLoginForm((current) => ({
                    ...current,
                    password: event.target.value,
                  }))
                }
              />
            </label>
            <div className="button-row">
              <button type="submit">Sign in</button>
              <button type="button" onClick={handleLogout}>
                Sign out
              </button>
            </div>
          </form>

          <form className="panel form-panel" onSubmit={handleCreateUser}>
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
          </form>

          <form className="panel form-panel" onSubmit={handleCreateApiToken}>
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
          </form>
        </section>
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

function formatCheckType(value: string) {
  return value.replace("HEALTH_CHECK_TYPE_", "").toLowerCase();
}

function formatHealthState(value: string) {
  return value.replace("HEALTH_STATE_", "").toLowerCase();
}

function formatProtocol(value: string) {
  return value.replace("PROTOCOL_", "").toLowerCase();
}

function healthTypeForEndpoint(protocol: string) {
  switch (protocol) {
    case "PROTOCOL_HTTPS":
      return "HEALTH_CHECK_TYPE_HTTPS";
    case "PROTOCOL_GRPC":
      return "HEALTH_CHECK_TYPE_GRPC";
    case "PROTOCOL_TCP":
      return "HEALTH_CHECK_TYPE_TCP";
    case "PROTOCOL_UDP":
      return "HEALTH_CHECK_TYPE_UDP";
    default:
      return "HEALTH_CHECK_TYPE_HTTP";
  }
}

function environmentName(environments: Environment[], id: string) {
  return (
    environments.find((environment) => environment.id === id)?.name ??
    `Environment ${id}`
  );
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
    return `Deployment ${policy.deploymentId}`;
  }
  if (policy.environmentId) {
    return `Environment ${policy.environmentId}`;
  }
  return "No scope";
}

function serviceOperationalStatus(service: Service, incidents: Incident[]) {
  const hasOpenIncident = incidents.some(
    (incident) =>
      incident.serviceId === service.id &&
      incident.state === "INCIDENT_STATE_OPEN",
  );
  return hasOpenIncident ? "degraded" : "healthy";
}

export default App;
