import { spawn, spawnSync } from "node:child_process";
import { mkdir } from "node:fs/promises";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const webRoot = path.join(root, "web");
const outputDir = path.join(root, "docs", "assets", "screenshots");
const port = Number(process.env.ALAUDA_SCREENSHOT_PORT || 4173);
const baseUrl = `http://127.0.0.1:${port}`;
const requirePlaywright = createRequire(
  path.join(process.env.PLAYWRIGHT_PACKAGE_ROOT || root, "package.json"),
);
const { chromium } = requirePlaywright("playwright");

const now = "2026-10-09T15:30:00Z";

const environments = [
  {
    id: "env-prod",
    key: "prod",
    name: "Production",
    description: "Customer-facing production environment",
    enabled: true,
    tier: "production",
    tags: { region: "us-west", owner: "platform" },
  },
  {
    id: "env-stage",
    key: "stage",
    name: "Staging",
    description: "Pre-production release validation",
    enabled: true,
    tier: "staging",
    tags: { region: "us-west", owner: "release" },
  },
];

const services = [
  {
    id: "svc-checkout",
    name: "checkout",
    displayName: "Checkout API",
    description: "Coordinates carts, payment authorization, and order capture.",
    tags: { team: "payments", tier: "critical" },
    metadata: { owner: "payments-oncall", runtime: "go" },
  },
  {
    id: "svc-identity",
    name: "identity",
    displayName: "Identity Service",
    description: "Authentication, sessions, and account policy enforcement.",
    tags: { team: "platform", tier: "critical" },
    metadata: { owner: "platform-identity", runtime: "go" },
  },
  {
    id: "svc-catalog",
    name: "catalog",
    displayName: "Product Catalog",
    description: "Searchable product metadata and merchandising feeds.",
    tags: { team: "commerce", tier: "standard" },
    metadata: { owner: "commerce", runtime: "node" },
  },
  {
    id: "svc-notify",
    name: "notifications",
    displayName: "Notifications",
    description: "Email, webhook, and incident notification delivery.",
    tags: { team: "ops", tier: "standard" },
    metadata: { owner: "operations", runtime: "go" },
  },
];

const deployments = [
  {
    id: "dep-checkout-prod",
    serviceId: "svc-checkout",
    environmentId: "env-prod",
    healthEnabled: true,
    alertsEnabled: true,
    alertCooldownMinutes: 10,
    tags: { version: "2026.10.3" },
  },
  {
    id: "dep-identity-prod",
    serviceId: "svc-identity",
    environmentId: "env-prod",
    healthEnabled: true,
    alertsEnabled: true,
    alertCooldownMinutes: 15,
    tags: { version: "2026.10.1" },
  },
  {
    id: "dep-catalog-prod",
    serviceId: "svc-catalog",
    environmentId: "env-prod",
    healthEnabled: true,
    alertsEnabled: true,
    alertCooldownMinutes: 20,
    tags: { version: "2026.9.8" },
  },
  {
    id: "dep-notify-prod",
    serviceId: "svc-notify",
    environmentId: "env-prod",
    healthEnabled: true,
    alertsEnabled: false,
    alertCooldownMinutes: 30,
    tags: { version: "2026.9.4" },
  },
];

const instances = [
  {
    id: "inst-checkout-a",
    deploymentId: "dep-checkout-prod",
    name: "checkout-prod-a",
    address: "10.44.12.18",
    port: 8080,
    description: "Primary checkout API runtime",
    enabled: true,
    tags: { zone: "az-a" },
    lastSeenAt: now,
  },
  {
    id: "inst-checkout-b",
    deploymentId: "dep-checkout-prod",
    name: "checkout-prod-b",
    address: "10.44.14.23",
    port: 8080,
    description: "Secondary checkout API runtime",
    enabled: true,
    tags: { zone: "az-b" },
    lastSeenAt: "2026-10-09T15:28:00Z",
  },
  {
    id: "inst-identity-a",
    deploymentId: "dep-identity-prod",
    name: "identity-prod-a",
    address: "10.44.8.11",
    port: 8443,
    enabled: true,
    tags: { zone: "az-a" },
    lastSeenAt: now,
  },
  {
    id: "inst-catalog-a",
    deploymentId: "dep-catalog-prod",
    name: "catalog-prod-a",
    address: "10.44.16.7",
    port: 7000,
    enabled: true,
    tags: { zone: "az-c" },
    lastSeenAt: "2026-10-09T15:26:00Z",
  },
  {
    id: "inst-notify-a",
    deploymentId: "dep-notify-prod",
    name: "notifications-prod-a",
    address: "10.44.20.9",
    port: 8085,
    enabled: true,
    tags: { zone: "az-b" },
    lastSeenAt: "2026-10-09T15:22:00Z",
  },
];

const endpoints = [
  endpoint("end-checkout-http-a", "inst-checkout-a", "http", "HTTP", 8080, "/healthz", true),
  endpoint("end-checkout-grpc-a", "inst-checkout-a", "grpc", "GRPC", 9090, "", false),
  endpoint("end-checkout-http-b", "inst-checkout-b", "http", "HTTP", 8080, "/healthz", true),
  endpoint("end-identity-https", "inst-identity-a", "https", "HTTPS", 8443, "/ready", true),
  endpoint("end-catalog-http", "inst-catalog-a", "http", "HTTP", 7000, "/status", true),
  endpoint("end-notify-http", "inst-notify-a", "http", "HTTP", 8085, "/healthz", true),
];

const healthChecks = [
  check("hc-checkout-a", "inst-checkout-a", "end-checkout-http-a", "Checkout readiness", "HEALTH_CHECK_TYPE_HTTP", true),
  check("hc-checkout-b", "inst-checkout-b", "end-checkout-http-b", "Checkout readiness", "HEALTH_CHECK_TYPE_HTTP", true),
  check("hc-identity", "inst-identity-a", "end-identity-https", "Identity TLS readiness", "HEALTH_CHECK_TYPE_HTTP", true),
  check("hc-catalog", "inst-catalog-a", "end-catalog-http", "Catalog search readiness", "HEALTH_CHECK_TYPE_HTTP", true),
  check("hc-notify", "inst-notify-a", "end-notify-http", "Notification queue readiness", "HEALTH_CHECK_TYPE_HTTP", false),
];

const incidents = [
  {
    id: "inc-checkout-latency",
    instanceId: "inst-checkout-b",
    deploymentId: "dep-checkout-prod",
    environmentId: "env-prod",
    serviceId: "svc-checkout",
    state: "INCIDENT_STATE_OPEN",
    openedAt: "2026-10-09T14:58:00Z",
    reason: "HTTP readiness latency exceeded threshold",
    impactSummary: "Checkout remains available with one degraded instance.",
    metadata: { health_check_id: "hc-checkout-b", severity: "warning" },
  },
  {
    id: "inc-catalog-recovery",
    instanceId: "inst-catalog-a",
    deploymentId: "dep-catalog-prod",
    environmentId: "env-prod",
    serviceId: "svc-catalog",
    state: "INCIDENT_STATE_RESOLVED",
    openedAt: "2026-10-08T18:12:00Z",
    resolvedAt: "2026-10-08T18:19:00Z",
    durationSeconds: 420,
    reason: "Catalog search readiness returned 503",
    impactSummary: "Search failover served cached catalog results.",
    resolutionMethod: "VerifiedRecovery",
  },
];

const events = [
  event("evt-1", "incident.opened", "incident", "inc-checkout-latency", "svc-checkout", "checkout-prod-b opened an availability incident", "registry-health", "2026-10-09T14:58:10Z"),
  event("evt-2", "health.failed", "health_check", "hc-checkout-b", "svc-checkout", "Checkout readiness failed after 2.4s", "health-monitor", "2026-10-09T14:57:40Z"),
  event("evt-3", "service.updated", "service", "svc-identity", "svc-identity", "Identity Service metadata updated", "admin@example.test", "2026-10-09T13:20:00Z"),
  event("evt-4", "instance.registered", "instance", "inst-catalog-a", "svc-catalog", "catalog-prod-a registered endpoint http :7000/status", "catalog-prod-a", "2026-10-09T12:48:00Z"),
  event("evt-5", "incident.resolved", "incident", "inc-catalog-recovery", "svc-catalog", "Catalog recovery verified from health evidence", "registry-health", "2026-10-08T18:19:00Z"),
];

const channels = [
  {
    id: "chan-ops",
    type: "webhook",
    name: "Ops Webhook",
    enabled: true,
    description: "Primary incident routing for platform operations.",
  },
  {
    id: "chan-payments",
    type: "email",
    name: "Payments On-call",
    enabled: true,
    description: "Checkout and payment service escalation list.",
  },
];

const policies = [
  {
    id: "pol-checkout",
    deploymentId: "dep-checkout-prod",
    environmentId: "env-prod",
    enabled: true,
    notifyOn: ["unhealthy", "recovered"],
    cooldownMinutes: 10,
    sendRecoveryNotification: true,
    channelIds: ["chan-ops", "chan-payments"],
  },
  {
    id: "pol-platform",
    environmentId: "env-prod",
    enabled: true,
    notifyOn: ["incident_opened", "incident_resolved"],
    cooldownMinutes: 15,
    sendRecoveryNotification: true,
    channelIds: ["chan-ops"],
  },
];

function endpoint(id, instanceId, name, kind, port, pathName, primary) {
  const instance = instances.find((item) => item.id === instanceId);
  return {
    id,
    instanceId,
    name,
    kind,
    port,
    path: pathName,
    enabled: true,
    primary,
    address:
      kind === "HTTPS"
        ? `https://${instance.address}:${port}${pathName}`
        : kind === "HTTP"
          ? `http://${instance.address}:${port}${pathName}`
          : `${instance.address}:${port}`,
  };
}

function check(id, instanceId, endpointId, name, type, enabled) {
  return {
    id,
    instanceId,
    endpointId,
    name,
    type,
    enabled,
    intervalSeconds: 30,
    timeoutSeconds: 5,
    failuresBeforeUnhealthy: 2,
    successesBeforeHealthy: 2,
    description: `${name} monitor`,
  };
}

function event(id, type, resourceType, resourceId, serviceId, message, actor, timestamp) {
  return {
    id,
    type,
    timestamp,
    resourceType,
    resourceId,
    environmentId: "env-prod",
    serviceId,
    actor,
    message,
  };
}

function healthStatus() {
  return {
    services: {
      "svc-checkout": "Degraded",
      "svc-identity": "Healthy",
      "svc-catalog": "Healthy",
      "svc-notify": "Unknown",
    },
    instances: {
      "inst-checkout-a": "Healthy",
      "inst-checkout-b": "Degraded",
      "inst-identity-a": "Healthy",
      "inst-catalog-a": "Healthy",
      "inst-notify-a": "Unknown",
    },
    monitored: {
      "inst-checkout-a": true,
      "inst-checkout-b": true,
      "inst-identity-a": true,
      "inst-catalog-a": true,
      "inst-notify-a": false,
    },
  };
}

function availability() {
  return {
    availability24h: {
      environmentId: "env-prod",
      windowHours: 24,
      windowStart: "2026-10-08T15:30:00Z",
      windowEnd: now,
      availabilityPercent: 99.92,
      downtimeSeconds: 69,
      incidentCount: 1,
    },
    availability7d: {
      environmentId: "env-prod",
      windowHours: 168,
      windowStart: "2026-10-02T15:30:00Z",
      windowEnd: now,
      availabilityPercent: 99.96,
      downtimeSeconds: 242,
      incidentCount: 2,
    },
    availability30d: {
      environmentId: "env-prod",
      windowHours: 720,
      windowStart: "2026-09-09T15:30:00Z",
      windowEnd: now,
      availabilityPercent: 99.98,
      downtimeSeconds: 518,
      incidentCount: 4,
    },
  };
}

function healthCheckRecords() {
  return healthChecks.map((item) => {
    const instance = instances.find((candidate) => candidate.id === item.instanceId);
    const deployment = deployments.find((candidate) => candidate.id === instance.deploymentId);
    const service = services.find((candidate) => candidate.id === deployment.serviceId);
    const environment = environments.find((candidate) => candidate.id === deployment.environmentId);
    const targetEndpoint = endpoints.find((candidate) => candidate.id === item.endpointId);
    const status = item.id === "hc-checkout-b" ? "unhealthy" : item.enabled ? "healthy" : "unknown";
    return {
      ...item,
      serviceId: service.id,
      service: service.displayName,
      environmentId: environment.id,
      environment: environment.name,
      instance: instance.name,
      address: instance.address,
      endpoint: targetEndpoint.name,
      kind: targetEndpoint.kind,
      port: targetEndpoint.port,
      latestStatus: status,
      latestResultAt: status === "unknown" ? undefined : "2026-10-09T15:27:00Z",
      latestDurationMs: status === "unhealthy" ? 2400 : status === "healthy" ? 42 : null,
      latestStatusCode: status === "unhealthy" ? 503 : status === "healthy" ? 200 : null,
      latestError: status === "unhealthy" ? "readiness latency threshold exceeded" : "",
      path: targetEndpoint.path,
      expectedStatus: "200",
    };
  });
}

function healthResults() {
  const rows = [
    ["hr-1", "hc-checkout-b", "inst-checkout-b", false, 2400, 503, "timeout", "readiness latency threshold exceeded", "2026-10-09T15:27:00Z"],
    ["hr-2", "hc-checkout-a", "inst-checkout-a", true, 38, 200, "", "", "2026-10-09T15:26:40Z"],
    ["hr-3", "hc-identity", "inst-identity-a", true, 31, 200, "", "", "2026-10-09T15:26:20Z"],
    ["hr-4", "hc-catalog", "inst-catalog-a", true, 44, 200, "", "", "2026-10-09T15:25:50Z"],
    ["hr-5", "hc-checkout-b", "inst-checkout-b", false, 2180, 503, "http_status", "readiness returned 503", "2026-10-09T15:24:00Z"],
    ["hr-6", "hc-checkout-a", "inst-checkout-a", true, 41, 200, "", "", "2026-10-09T15:23:20Z"],
  ];
  return rows.map(([id, checkId, instanceId, success, latencyMs, statusCode, errorType, errorMessage, timestamp]) => {
    const checkRecord = healthChecks.find((item) => item.id === checkId);
    const instance = instances.find((item) => item.id === instanceId);
    const deployment = deployments.find((item) => item.id === instance.deploymentId);
    const service = services.find((item) => item.id === deployment.serviceId);
    const environment = environments.find((item) => item.id === deployment.environmentId);
    const targetEndpoint = endpoints.find((item) => item.id === checkRecord.endpointId);
    return {
      id,
      timestamp,
      serviceId: service.id,
      service: service.displayName,
      environmentId: environment.id,
      environment: environment.name,
      instanceId: instance.id,
      instance: instance.name,
      endpointId: targetEndpoint.id,
      endpoint: targetEndpoint.name,
      kind: targetEndpoint.kind,
      port: targetEndpoint.port,
      address: instance.address,
      checkId,
      check: checkRecord.name,
      type: checkRecord.type,
      path: targetEndpoint.path,
      expectedStatus: "200",
      success,
      latencyMs,
      statusCode,
      errorType,
      errorMessage,
    };
  });
}

function filterByDeployment(deploymentId) {
  return instances.filter((item) => !deploymentId || item.deploymentId === deploymentId);
}

function requestBody(request) {
  try {
    return request.postDataJSON();
  } catch {
    return {};
  }
}

async function mockResponse(route) {
  const request = route.request();
  const url = new URL(request.url());
  const pathname = url.pathname;
  let body;

  if (pathname === "/api/v1/auth/me") {
    body = {
      user: {
        id: "user-admin",
        username: "admin",
        email: "admin@example.test",
        displayName: "Alauda Admin",
        role: "Administrator",
        enabled: true,
        mustChangePassword: false,
      },
      scopes: ["read", "write", "admin"],
      mustChangePassword: false,
    };
  } else if (pathname === "/api/v1/environments") {
    body = { environments, nextPageToken: "" };
  } else if (pathname === "/api/v1/services") {
    body = { services, nextPageToken: "" };
  } else if (pathname === "/api/v1/health/status") {
    body = healthStatus();
  } else if (pathname === "/api/v1/health/checks") {
    body = { items: healthCheckRecords(), total: healthChecks.length, page: 1, pageSize: 50 };
  } else if (pathname === "/api/v1/health/results") {
    body = { results: healthResults(), nextPageToken: "" };
  } else if (pathname === "/api/v1/events/watch") {
    return route.fulfill({
      status: 204,
      headers: { "content-type": "text/event-stream" },
      body: "",
    });
  } else if (pathname.includes("/ListDeployments")) {
    body = { deployments, pagination: { nextPageToken: "" } };
  } else if (pathname.includes("/ListInstances")) {
    const payload = requestBody(request);
    body = { instances: filterByDeployment(payload.deploymentId), pagination: { nextPageToken: "" } };
  } else if (pathname.includes("/ListEndpoints")) {
    const payload = requestBody(request);
    body = {
      endpoints: endpoints.filter((item) => !payload.instanceId || item.instanceId === payload.instanceId),
      pagination: { nextPageToken: "" },
    };
  } else if (pathname.includes("/ListHealthChecks")) {
    const payload = requestBody(request);
    body = {
      healthChecks: healthChecks.filter((item) => !payload.instanceId || item.instanceId === payload.instanceId),
      pagination: { nextPageToken: "" },
    };
  } else if (pathname.includes("/ListHealthResults")) {
    body = { results: healthResults(), pagination: { nextPageToken: "" } };
  } else if (pathname.includes("/GetInstanceHealthState")) {
    const payload = requestBody(request);
    const state = healthStatus().instances[payload.instanceId] || "Unknown";
    body = {
      state: {
        instanceId: payload.instanceId,
        currentState: `HEALTH_STATE_${state.toUpperCase()}`,
        monitored: healthStatus().monitored[payload.instanceId] || false,
        consecutiveSuccesses: state === "Healthy" ? 5 : 0,
        consecutiveFailures: state === "Degraded" ? 2 : 0,
        lastCheckTime: now,
      },
    };
  } else if (pathname.includes("/GetAvailability")) {
    body = availability();
  } else if (pathname.includes("/ListIncidents")) {
    body = { incidents, pagination: { nextPageToken: "" } };
  } else if (pathname.includes("/ListEvents")) {
    body = { events, pagination: { nextPageToken: "" } };
  } else if (pathname.includes("/ListNotificationChannels")) {
    body = { channels, pagination: { nextPageToken: "" } };
  } else if (pathname.includes("/ListAlertPolicies")) {
    body = { policies, pagination: { nextPageToken: "" } };
  } else if (pathname === "/api/v1/auth/logout") {
    body = {};
  } else {
    body = {};
  }

  return route.fulfill({
    status: 200,
    headers: {
      "content-type": "application/json",
      "cache-control": "no-store",
    },
    body: JSON.stringify(body),
  });
}

async function waitForServer() {
  const deadline = Date.now() + 30_000;
  while (Date.now() < deadline) {
    try {
      const response = await fetch(baseUrl);
      if (response.ok) return;
    } catch {
      await new Promise((resolve) => setTimeout(resolve, 250));
    }
  }
  throw new Error(`Timed out waiting for Vite at ${baseUrl}`);
}

async function waitForApp(page) {
  await page.waitForSelector("text=Loading authentication...", { state: "detached", timeout: 20_000 });
  await page.waitForSelector("text=Loading registry data...", { state: "detached", timeout: 20_000 });
  await page.locator(".alauda-shell").waitFor({ state: "visible", timeout: 20_000 });
  await page.evaluate(() => document.fonts?.ready);
  await page.waitForTimeout(600);
}

async function capture(page, urlPath, fileName, prepare) {
  await page.goto(`${baseUrl}${urlPath}`, { waitUntil: "domcontentloaded" });
  await waitForApp(page);
  if (prepare) {
    await prepare(page);
    await page.waitForTimeout(400);
  }
  await page.screenshot({
    path: path.join(outputDir, fileName),
    fullPage: false,
    animations: "disabled",
  });
}

await mkdir(outputDir, { recursive: true });

const npmCommand = process.platform === "win32" ? "npm.cmd" : "npm";
const serverCommand = process.platform === "win32" ? "cmd.exe" : npmCommand;
const serverArgs =
  process.platform === "win32"
    ? [
        "/d",
        "/s",
        "/c",
        `${npmCommand} run dev -- --host 127.0.0.1 --port ${port} --strictPort`,
      ]
    : [
        "run",
        "dev",
        "--",
        "--host",
        "127.0.0.1",
        "--port",
        String(port),
        "--strictPort",
      ];
const server = spawn(
  serverCommand,
  serverArgs,
  {
    cwd: webRoot,
    env: { ...process.env, BROWSER: "none" },
    stdio: ["ignore", "pipe", "pipe"],
  },
);

server.stdout.on("data", (chunk) => process.stdout.write(chunk));
server.stderr.on("data", (chunk) => process.stderr.write(chunk));

try {
  await waitForServer();
  const browser = await chromium.launch();
  try {
    const page = await browser.newPage({
      viewport: { width: 1440, height: 980 },
      deviceScaleFactor: 1,
      locale: "en-US",
      timezoneId: "America/Phoenix",
    });
    await page.route("**/*", (route) => {
      const pathname = new URL(route.request().url()).pathname;
      if (
        pathname.startsWith("/api/") ||
        pathname.startsWith("/registry.") ||
        pathname === "/openapi.json"
      ) {
        return mockResponse(route);
      }
      return route.continue();
    });
    await page.addStyleTag({
      content: `
        *, *::before, *::after {
          caret-color: transparent !important;
        }
      `,
    }).catch(() => undefined);

    await capture(page, "/dashboard", "alauda-dashboard.png");
    await capture(page, "/services/svc-checkout/instances", "alauda-service-inventory.png");
    await capture(page, "/health/results", "alauda-health-results.png", async (healthPage) => {
      await healthPage.locator('input[type="datetime-local"]').nth(0).fill("2026-10-09T08:00");
      await healthPage.locator('input[type="datetime-local"]').nth(1).fill("2026-10-09T09:00");
      await healthPage.evaluate(() => {
        if (document.activeElement instanceof HTMLElement) {
          document.activeElement.blur();
        }
      });
    });
  } finally {
    await browser.close();
  }
} finally {
  if (process.platform === "win32") {
    spawnSync("taskkill", ["/pid", String(server.pid), "/t", "/f"], {
      stdio: "ignore",
    });
  } else {
    server.kill();
  }
}

console.log(`Saved screenshots to ${outputDir}`);
