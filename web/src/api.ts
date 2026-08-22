const apiBaseUrl = import.meta.env.VITE_API_BASE_URL ?? "";
let bearerToken = "";

export function setBearerToken(token: string) {
  bearerToken = token;
}

export type Environment = {
  id: string;
  key: string;
  name: string;
  description?: string;
  enabled: boolean;
  tier?: string;
  tags?: Record<string, string>;
};

export type Service = {
  id: string;
  name: string;
  displayName: string;
  description?: string;
  tags?: Record<string, string>;
  metadata?: Record<string, string>;
};

export type ServiceDeployment = {
  id: string;
  serviceId: string;
  environmentId: string;
  healthEnabled: boolean;
  alertsEnabled: boolean;
  alertCooldownMinutes: number;
  tags?: Record<string, string>;
  metadata?: Record<string, string>;
  createdAt?: string;
  updatedAt?: string;
};

export type ServiceInstance = {
  id: string;
  deploymentId: string;
  name: string;
  address: string;
  port: number;
  description?: string;
  enabled: boolean;
  tags?: Record<string, string>;
  metadata?: Record<string, string>;
  createdAt?: string;
  updatedAt?: string;
  lastSeenAt?: string;
};

export type Endpoint = {
  id: string;
  instanceId: string;
  name: string;
  protocol: string;
  port: number;
  path: string;
  enabled: boolean;
  primary: boolean;
  tags?: Record<string, string>;
  metadata?: Record<string, string>;
};

export type HealthCheck = {
  id: string;
  instanceId: string;
  endpointId?: string;
  name: string;
  type: string;
  enabled: boolean;
  intervalSeconds: number;
  timeoutSeconds: number;
  failuresBeforeUnhealthy: number;
  successesBeforeHealthy: number;
  description?: string;
  metadata?: Record<string, string>;
};

export type HealthResult = {
  id: string;
  healthCheckId: string;
  instanceId: string;
  timestamp?: string;
  success: boolean;
  latencyMs?: number;
  statusCode?: number;
  errorType?: string;
  errorMessage?: string;
};

export type HealthStateView = {
  instanceId: string;
  currentState: string;
  consecutiveSuccesses: number;
  consecutiveFailures: number;
  lastCheckTime?: string;
};

export type Incident = {
  id: string;
  instanceId: string;
  deploymentId: string;
  environmentId: string;
  serviceId: string;
  state: string;
  openedAt?: string;
  resolvedAt?: string;
  durationSeconds?: number;
  reason: string;
  impactSummary?: string;
  metadata?: Record<string, string>;
};

export type AvailabilitySummary = {
  environmentId?: string;
  serviceId?: string;
  deploymentId?: string;
  instanceId?: string;
  windowHours: number;
  windowStart?: string;
  windowEnd?: string;
  availabilityPercent: number;
  downtimeSeconds: number;
  incidentCount: number;
};

export type EventRecord = {
  id: string;
  type: string;
  timestamp?: string;
  resourceType: string;
  resourceId: string;
  environmentId?: string;
  serviceId?: string;
  deploymentId?: string;
  instanceId?: string;
  actor: string;
  message: string;
  metadata?: Record<string, string>;
};

export type NotificationChannel = {
  id: string;
  type: string;
  name: string;
  enabled: boolean;
  description?: string;
  configuration?: Record<string, string>;
  retryPolicy?: Record<string, string>;
  tags?: Record<string, string>;
  createdAt?: string;
  updatedAt?: string;
};

export type AlertPolicy = {
  id: string;
  deploymentId?: string;
  environmentId?: string;
  enabled: boolean;
  notifyOn?: string[];
  cooldownMinutes: number;
  sendRecoveryNotification: boolean;
  filters?: Record<string, string>;
  channelIds?: string[];
  createdAt?: string;
  updatedAt?: string;
};

export type UserAccount = {
  id: string;
  username: string;
  email: string;
  displayName: string;
  role: string;
  enabled: boolean;
  createdAt?: string;
  updatedAt?: string;
  lastLoginAt?: string;
};

export type ApiToken = {
  id: string;
  userId: string;
  name: string;
  scopes: string[];
  environmentIds?: string[];
  expiresAt?: string;
  lastUsedAt?: string;
  enabled: boolean;
  createdAt?: string;
  createdBy: string;
};

type ListEnvironmentsResponse = {
  environments?: Environment[];
};

type ListServicesResponse = {
  services?: Service[];
};

type CreateEnvironmentResponse = {
  environment?: Environment;
};

type CreateServiceResponse = {
  service?: Service;
};

type DeleteServiceResponse = Record<string, never>;

type CreateDeploymentResponse = {
  deployment?: ServiceDeployment;
};

type ListDeploymentsResponse = {
  deployments?: ServiceDeployment[];
};

type CreateInstanceResponse = {
  instance?: ServiceInstance;
};

type UpdateInstanceResponse = {
  instance?: ServiceInstance;
};

type DeleteInstanceResponse = Record<string, never>;

type RegisterRuntimeResponse = {
  deployment?: ServiceDeployment;
  instance?: ServiceInstance;
  endpoints?: Endpoint[];
};

type ListInstancesResponse = {
  instances?: ServiceInstance[];
};

type CreateEndpointResponse = {
  endpoint?: Endpoint;
};

type UpdateEndpointResponse = {
  endpoint?: Endpoint;
};

type DeleteEndpointResponse = Record<string, never>;

type ListEndpointsResponse = {
  endpoints?: Endpoint[];
};

type ListHealthChecksResponse = {
  healthChecks?: HealthCheck[];
};

type CreateHealthCheckResponse = {
  healthCheck?: HealthCheck;
};

type RunHealthCheckResponse = {
  result?: HealthResult;
  state?: HealthStateView;
};

type GetInstanceHealthStateResponse = {
  state?: HealthStateView;
};

type ListHealthResultsResponse = {
  results?: HealthResult[];
};

type ListIncidentsResponse = {
  incidents?: Incident[];
};

type ResolveIncidentResponse = {
  incident?: Incident;
};

type GetAvailabilityResponse = {
  availability24h?: AvailabilitySummary;
  availability7d?: AvailabilitySummary;
  availability30d?: AvailabilitySummary;
};

type ListEventsResponse = {
  events?: EventRecord[];
};

type ListNotificationChannelsResponse = {
  channels?: NotificationChannel[];
};

type CreateNotificationChannelResponse = {
  channel?: NotificationChannel;
};

type UpdateNotificationChannelResponse = {
  channel?: NotificationChannel;
};

type ListAlertPoliciesResponse = {
  policies?: AlertPolicy[];
};

type CreateAlertPolicyResponse = {
  policy?: AlertPolicy;
};

type UpdateAlertPolicyResponse = {
  policy?: AlertPolicy;
};

type LoginResponse = {
  user?: UserAccount;
  token?: string;
  expiresAt?: string;
};

type ListUsersResponse = {
  users?: UserAccount[];
};

type CreateUserResponse = {
  user?: UserAccount;
};

type ListApiTokensResponse = {
  tokens?: ApiToken[];
};

type CreateApiTokenResponse = {
  token?: ApiToken;
  secret?: string;
};

function authHeaders(): Record<string, string> {
  return bearerToken ? { Authorization: `Bearer ${bearerToken}` } : {};
}

async function connectRequest<TResponse>(
  path: string,
  body: Record<string, unknown>,
): Promise<TResponse> {
  const response = await fetch(`${apiBaseUrl}${path}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...authHeaders(),
    },
    body: JSON.stringify(body),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || `Request failed with status ${response.status}`);
  }

  return response.json() as Promise<TResponse>;
}

export async function listEnvironments(): Promise<Environment[]> {
  const response = await connectRequest<ListEnvironmentsResponse>(
    "/registry.v1.EnvironmentService/ListEnvironments",
    {
      includeDisabled: true,
      pagination: { pageSize: 100 },
    },
  );
  return response.environments ?? [];
}

export async function createEnvironment(input: {
  key: string;
  name: string;
  tier: string;
  description: string;
}): Promise<Environment> {
  const response = await connectRequest<CreateEnvironmentResponse>(
    "/registry.v1.EnvironmentService/CreateEnvironment",
    input,
  );
  if (!response.environment) {
    throw new Error("CreateEnvironment returned no environment");
  }
  return response.environment;
}

export async function listServices(environmentId?: string): Promise<Service[]> {
  const response = await connectRequest<ListServicesResponse>(
    "/registry.v1.CatalogService/ListServices",
    {
      environmentId: environmentId || undefined,
      pagination: { pageSize: 100 },
    },
  );
  return response.services ?? [];
}

export async function createService(input: {
  name: string;
  displayName: string;
  description: string;
}): Promise<Service> {
  const response = await connectRequest<CreateServiceResponse>(
    "/registry.v1.CatalogService/CreateService",
    input,
  );
  if (!response.service) {
    throw new Error("CreateService returned no service");
  }
  return response.service;
}

export async function deleteService(id: string): Promise<void> {
  await connectRequest<DeleteServiceResponse>(
    "/registry.v1.CatalogService/DeleteService",
    { id },
  );
}

export async function listDeployments(input: {
  serviceId?: string;
  environmentId?: string;
}): Promise<ServiceDeployment[]> {
  const response = await connectRequest<ListDeploymentsResponse>(
    "/registry.v1.DeploymentService/ListDeployments",
    {
      ...input,
      pagination: { pageSize: 100 },
    },
  );
  return response.deployments ?? [];
}

export async function createDeployment(input: {
  serviceId: string;
  environmentId: string;
  healthEnabled: boolean;
  alertsEnabled: boolean;
  alertCooldownMinutes: number;
}): Promise<ServiceDeployment> {
  const response = await connectRequest<CreateDeploymentResponse>(
    "/registry.v1.DeploymentService/CreateDeployment",
    input,
  );
  if (!response.deployment) {
    throw new Error("CreateDeployment returned no deployment");
  }
  return response.deployment;
}

export async function listInstances(
  deploymentId?: string,
): Promise<ServiceInstance[]> {
  const response = await connectRequest<ListInstancesResponse>(
    "/registry.v1.InstanceService/ListInstances",
    {
      deploymentId: deploymentId || undefined,
      pagination: { pageSize: 100 },
    },
  );
  return response.instances ?? [];
}

export async function createInstance(input: {
  deploymentId: string;
  name: string;
  address: string;
  port: number;
  description: string;
  enabled: boolean;
}): Promise<ServiceInstance> {
  const response = await connectRequest<CreateInstanceResponse>(
    "/registry.v1.InstanceService/CreateInstance",
    input,
  );
  if (!response.instance) {
    throw new Error("CreateInstance returned no instance");
  }
  return response.instance;
}

export async function updateInstance(input: {
  id: string;
  address: string;
  port: number;
  description: string;
  enabled: boolean;
  tags?: Record<string, string>;
  metadata?: Record<string, string>;
}): Promise<ServiceInstance> {
  const response = await connectRequest<UpdateInstanceResponse>(
    "/registry.v1.InstanceService/UpdateInstance",
    {
      ...input,
      tags: input.tags ?? {},
      metadata: input.metadata ?? {},
    },
  );
  if (!response.instance) {
    throw new Error("UpdateInstance returned no instance");
  }
  return response.instance;
}

export async function deleteInstance(id: string): Promise<void> {
  await connectRequest<DeleteInstanceResponse>(
    "/registry.v1.InstanceService/DeleteInstance",
    { id },
  );
}

export async function registerRuntime(input: {
  serviceId: string;
  environmentId: string;
  instance: {
    name: string;
    address: string;
    description: string;
    enabled: boolean;
  };
  endpoints: {
    name: string;
    protocol: string;
    port: number;
    path: string;
    enabled: boolean;
    primary: boolean;
  }[];
}): Promise<RegisterRuntimeResponse> {
  const response = await connectRequest<RegisterRuntimeResponse>(
    "/registry.v1.InstanceService/RegisterRuntime",
    input,
  );
  if (!response.deployment || !response.instance) {
    throw new Error("RegisterRuntime returned no runtime registration");
  }
  return response;
}

export async function listEndpoints(instanceId?: string): Promise<Endpoint[]> {
  const response = await connectRequest<ListEndpointsResponse>(
    "/registry.v1.EndpointService/ListEndpoints",
    {
      instanceId: instanceId || undefined,
      pagination: { pageSize: 100 },
    },
  );
  return response.endpoints ?? [];
}

export async function createEndpoint(input: {
  instanceId: string;
  name: string;
  protocol: string;
  port: number;
  path: string;
  enabled: boolean;
  primary?: boolean;
}): Promise<Endpoint> {
  const response = await connectRequest<CreateEndpointResponse>(
    "/registry.v1.EndpointService/CreateEndpoint",
    input,
  );
  if (!response.endpoint) {
    throw new Error("CreateEndpoint returned no endpoint");
  }
  return response.endpoint;
}

export async function updateEndpoint(input: {
  id: string;
  name: string;
  protocol: string;
  port: number;
  path: string;
  enabled: boolean;
  primary?: boolean;
  tags?: Record<string, string>;
  metadata?: Record<string, string>;
}): Promise<Endpoint> {
  const response = await connectRequest<UpdateEndpointResponse>(
    "/registry.v1.EndpointService/UpdateEndpoint",
    {
      ...input,
      tags: input.tags ?? {},
      metadata: input.metadata ?? {},
    },
  );
  if (!response.endpoint) {
    throw new Error("UpdateEndpoint returned no endpoint");
  }
  return response.endpoint;
}

export async function deleteEndpoint(id: string): Promise<void> {
  await connectRequest<DeleteEndpointResponse>(
    "/registry.v1.EndpointService/DeleteEndpoint",
    { id },
  );
}

export async function listHealthChecks(
  instanceId?: string,
): Promise<HealthCheck[]> {
  const response = await connectRequest<ListHealthChecksResponse>(
    "/registry.v1.HealthService/ListHealthChecks",
    {
      instanceId: instanceId || undefined,
      includeDisabled: true,
      pagination: { pageSize: 100 },
    },
  );
  return response.healthChecks ?? [];
}

export async function createHealthCheck(input: {
  instanceId: string;
  endpointId: string;
  name: string;
  type: string;
  enabled: boolean;
  intervalSeconds: number;
  timeoutSeconds: number;
  failuresBeforeUnhealthy: number;
  successesBeforeHealthy: number;
  description: string;
  metadata: Record<string, string>;
}): Promise<HealthCheck> {
  const response = await connectRequest<CreateHealthCheckResponse>(
    "/registry.v1.HealthService/CreateHealthCheck",
    {
      ...input,
      endpointId: input.endpointId || undefined,
    },
  );
  if (!response.healthCheck) {
    throw new Error("CreateHealthCheck returned no health check");
  }
  return response.healthCheck;
}

export async function runHealthCheck(
  id: string,
): Promise<RunHealthCheckResponse> {
  return connectRequest<RunHealthCheckResponse>(
    "/registry.v1.HealthService/RunHealthCheck",
    { id },
  );
}

export async function getInstanceHealthState(
  instanceId: string,
): Promise<HealthStateView | null> {
  const response = await connectRequest<GetInstanceHealthStateResponse>(
    "/registry.v1.HealthService/GetInstanceHealthState",
    { instanceId },
  );
  return response.state ?? null;
}

export async function listHealthResults(
  healthCheckId?: string,
): Promise<HealthResult[]> {
  const response = await connectRequest<ListHealthResultsResponse>(
    "/registry.v1.HealthService/ListHealthResults",
    {
      healthCheckId: healthCheckId || undefined,
      pagination: { pageSize: 20 },
    },
  );
  return response.results ?? [];
}

export async function listIncidents(input: {
  environmentId?: string;
  serviceId?: string;
  deploymentId?: string;
  instanceId?: string;
  state?: string;
}): Promise<Incident[]> {
  const response = await connectRequest<ListIncidentsResponse>(
    "/registry.v1.IncidentService/ListIncidents",
    {
      ...input,
      pagination: { pageSize: 100 },
    },
  );
  return response.incidents ?? [];
}

export async function resolveIncident(id: string): Promise<Incident> {
  const response = await connectRequest<ResolveIncidentResponse>(
    "/registry.v1.IncidentService/ResolveIncident",
    { id, reason: "resolved from UI" },
  );
  if (!response.incident) {
    throw new Error("ResolveIncident returned no incident");
  }
  return response.incident;
}

export async function getAvailability(input: {
  environmentId?: string;
  serviceId?: string;
  deploymentId?: string;
  instanceId?: string;
}): Promise<GetAvailabilityResponse> {
  return connectRequest<GetAvailabilityResponse>(
    "/registry.v1.HealthService/GetAvailability",
    input,
  );
}

export async function listEvents(input: {
  environmentId?: string;
  serviceId?: string;
  deploymentId?: string;
  instanceId?: string;
  type?: string;
}): Promise<EventRecord[]> {
  const response = await connectRequest<ListEventsResponse>(
    "/registry.v1.EventService/ListEvents",
    {
      ...input,
      pagination: { pageSize: 100 },
    },
  );
  return response.events ?? [];
}

export function eventStreamUrl(input: {
  environmentId?: string;
  serviceId?: string;
  deploymentId?: string;
  instanceId?: string;
}): string {
  const params = new URLSearchParams();
  Object.entries(input).forEach(([key, value]) => {
    if (value) {
      params.set(key, value);
    }
  });
  const query = params.toString();
  return `${apiBaseUrl}/api/v1/events/watch${query ? `?${query}` : ""}`;
}

export async function listNotificationChannels(): Promise<
  NotificationChannel[]
> {
  const response = await connectRequest<ListNotificationChannelsResponse>(
    "/registry.v1.AlertService/ListNotificationChannels",
    {
      pagination: { pageSize: 100 },
    },
  );
  return response.channels ?? [];
}

export async function createNotificationChannel(input: {
  type: string;
  name: string;
  enabled: boolean;
  description: string;
  configuration: Record<string, string>;
  retryPolicy: Record<string, string>;
  tags: Record<string, string>;
}): Promise<NotificationChannel> {
  const response = await connectRequest<CreateNotificationChannelResponse>(
    "/registry.v1.AlertService/CreateNotificationChannel",
    input,
  );
  if (!response.channel) {
    throw new Error("CreateNotificationChannel returned no channel");
  }
  return response.channel;
}

export async function updateNotificationChannel(input: {
  id: string;
  name: string;
  enabled: boolean;
  description: string;
  configuration: Record<string, string>;
  retryPolicy: Record<string, string>;
  tags: Record<string, string>;
}): Promise<NotificationChannel> {
  const response = await connectRequest<UpdateNotificationChannelResponse>(
    "/registry.v1.AlertService/UpdateNotificationChannel",
    input,
  );
  if (!response.channel) {
    throw new Error("UpdateNotificationChannel returned no channel");
  }
  return response.channel;
}

export async function listAlertPolicies(input: {
  environmentId?: string;
  deploymentId?: string;
}): Promise<AlertPolicy[]> {
  const response = await connectRequest<ListAlertPoliciesResponse>(
    "/registry.v1.AlertService/ListAlertPolicies",
    {
      ...input,
      pagination: { pageSize: 100 },
    },
  );
  return response.policies ?? [];
}

export async function createAlertPolicy(input: {
  deploymentId: string;
  environmentId: string;
  enabled: boolean;
  notifyOn: string[];
  cooldownMinutes: number;
  sendRecoveryNotification: boolean;
  filters: Record<string, string>;
  channelIds: string[];
}): Promise<AlertPolicy> {
  const response = await connectRequest<CreateAlertPolicyResponse>(
    "/registry.v1.AlertService/CreateAlertPolicy",
    {
      ...input,
      deploymentId: input.deploymentId || undefined,
      environmentId: input.environmentId || undefined,
    },
  );
  if (!response.policy) {
    throw new Error("CreateAlertPolicy returned no policy");
  }
  return response.policy;
}

export async function updateAlertPolicy(input: {
  id: string;
  enabled: boolean;
  notifyOn: string[];
  cooldownMinutes: number;
  sendRecoveryNotification: boolean;
  filters: Record<string, string>;
  channelIds: string[];
}): Promise<AlertPolicy> {
  const response = await connectRequest<UpdateAlertPolicyResponse>(
    "/registry.v1.AlertService/UpdateAlertPolicy",
    input,
  );
  if (!response.policy) {
    throw new Error("UpdateAlertPolicy returned no policy");
  }
  return response.policy;
}

export async function testNotificationChannel(
  channelId: string,
): Promise<string> {
  const response = await fetch(
    `${apiBaseUrl}/api/v1/alerts/test/${channelId}`,
    {
      method: "POST",
      headers: authHeaders(),
    },
  );
  const text = await response.text();
  if (!response.ok) {
    throw new Error(text || `Request failed with status ${response.status}`);
  }
  return text;
}

export async function login(input: {
  username: string;
  password: string;
}): Promise<LoginResponse> {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    throw new Error((await response.text()) || "Login failed");
  }
  return response.json() as Promise<LoginResponse>;
}

export async function listUsers(): Promise<UserAccount[]> {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/users`, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    throw new Error((await response.text()) || "List users failed");
  }
  const payload = (await response.json()) as ListUsersResponse;
  return payload.users ?? [];
}

export async function createUser(input: {
  username: string;
  email: string;
  displayName: string;
  password: string;
  role: string;
}): Promise<UserAccount> {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/users`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    throw new Error((await response.text()) || "Create user failed");
  }
  const payload = (await response.json()) as CreateUserResponse;
  if (!payload.user) {
    throw new Error("Create user returned no user");
  }
  return payload.user;
}

export async function listApiTokens(userId: string): Promise<ApiToken[]> {
  const query = userId ? `?userId=${encodeURIComponent(userId)}` : "";
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/tokens${query}`, {
    headers: authHeaders(),
  });
  if (!response.ok) {
    throw new Error((await response.text()) || "List tokens failed");
  }
  const payload = (await response.json()) as ListApiTokensResponse;
  return payload.tokens ?? [];
}

export async function createApiToken(input: {
  userId: string;
  name: string;
  scopes: string[];
  expiresAt?: string;
}): Promise<CreateApiTokenResponse> {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/tokens`, {
    method: "POST",
    headers: { "Content-Type": "application/json", ...authHeaders() },
    body: JSON.stringify(input),
  });
  if (!response.ok) {
    throw new Error((await response.text()) || "Create token failed");
  }
  return response.json() as Promise<CreateApiTokenResponse>;
}

export async function revokeApiToken(id: string): Promise<void> {
  const response = await fetch(`${apiBaseUrl}/api/v1/auth/tokens/${id}`, {
    method: "DELETE",
    headers: authHeaders(),
  });
  if (!response.ok) {
    throw new Error((await response.text()) || "Revoke token failed");
  }
}
