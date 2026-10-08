export function mockResponse(path: string, init?: RequestInit) {
  if (path.includes("/api/v1/health/results")) return { results: [], nextPageToken: "" };
  if (
    path.includes("CreateHealthCheck") ||
    path.includes("UpdateHealthCheck")
  ) {
    return {
      healthCheck: {
        id: "hc-1",
        instanceId: "inst-1",
        name: "HTTP health",
        type: "HEALTH_CHECK_TYPE_HTTP",
        enabled: true,
        intervalSeconds: 10,
        timeoutSeconds: 3,
        failuresBeforeUnhealthy: 3,
        successesBeforeHealthy: 2,
        ...JSON.parse(init?.body?.toString() ?? "{}"),
      },
    };
  }
  if (path.includes("CreateEndpoint") || path.includes("UpdateEndpoint")) {
    return {
      endpoint: {
        id: path.includes("CreateEndpoint") ? "end-2" : "end-1",
        instanceId: "inst-1",
        name: "http",
        protocol: "PROTOCOL_HTTP",
        port: 8080,
        path: "/",
        enabled: true,
        primary: false,
        ...JSON.parse(init?.body?.toString() ?? "{}"),
      },
    };
  }
  if (path.includes("UpdateInstance")) {
    return {
      instance: {
        id: "inst-1",
        deploymentId: "dep-1",
        name: "checkout-a",
        address: "10.0.0.1",
        port: 8080,
        enabled: true,
        ...JSON.parse(init?.body?.toString() ?? "{}"),
      },
    };
  }
  if (path.includes("/api/v1/auth/me")) {
    return {
      user: {
        id: "user-1",
        username: "admin",
        email: "admin@example.test",
        displayName: "Admin",
        role: "Administrator",
        enabled: true,
        mustChangePassword: false,
      },
      scopes: ["read", "write", "admin"],
      mustChangePassword: false,
    };
  }
  if (path.includes("/api/v1/auth/users")) {
    return {
      users: [
        {
          id: "user-1",
          username: "admin",
          email: "admin@example.test",
          displayName: "Admin",
          role: "Administrator",
          enabled: true,
        },
      ],
      user: {
        id: "user-1",
        username: "admin",
        email: "admin@example.test",
        displayName: "Admin",
        role: "Administrator",
        enabled: true,
      },
    };
  }
  if (path.includes("/api/v1/auth/tokens")) {
    return {
      tokens: [
        {
          id: "tok-1",
          userId: "user-1",
          name: "automation",
          scopes: ["read", "write"],
          enabled: true,
          createdBy: "user-1",
        },
      ],
      token: {
        id: "tok-2",
        userId: "user-1",
        name: "ci",
        scopes: ["read"],
        enabled: true,
        createdBy: "user-1",
      },
      secret: "sr_test",
    };
  }
  if (path.includes("/api/v1/auth/login")) {
    return {
      user: {
        id: "user-1",
        username: "admin",
        email: "admin@example.test",
        displayName: "Admin",
        role: "Administrator",
        enabled: true,
      },
      token: "sr_test",
    };
  }
  if (path.includes("ListEnvironments")) {
    return {
      environments: [
        {
          id: "env-1",
          key: "prod",
          name: "Production",
          enabled: true,
          tier: "prod",
        },
      ],
    };
  }
  if (path.includes("ListServices")) {
    return {
      services: [
        {
          id: "svc-1",
          name: "checkout",
          displayName: "Checkout",
          description: "Checkout API",
          tags: { team: "payments" },
          metadata: { owner: "ops" },
        },
      ],
    };
  }
  if (path.includes("ListHealthChecks")) {
    return {
      healthChecks: [
        {
          id: "hc-1",
          instanceId: "inst-1",
          name: "HTTP health",
          type: "HEALTH_CHECK_TYPE_HTTP",
          enabled: true,
          intervalSeconds: 10,
          timeoutSeconds: 3,
          failuresBeforeUnhealthy: 3,
          successesBeforeHealthy: 2,
        },
      ],
    };
  }
  if (path.includes("ListIncidents")) {
    return {
      incidents: [
        {
          id: "inc-1",
          instanceId: "inst-1",
          deploymentId: "dep-1",
          environmentId: "env-1",
          serviceId: "svc-1",
          state: "INCIDENT_STATE_OPEN",
          openedAt: "2026-08-12T12:00:00Z",
          reason: "health check failed",
        },
      ],
    };
  }
  if (path.includes("ListEvents")) {
    return {
      events: [
        {
          id: "evt-1",
          type: "service.created",
          timestamp: "2026-08-12T12:00:00Z",
          resourceType: "service",
          resourceId: "svc-1",
          environmentId: "env-1",
          serviceId: "svc-1",
          actor: "test",
          message: "Checkout service created",
        },
      ],
    };
  }
  if (path.includes("GetAvailability")) {
    return {
      availability24h: {
        windowHours: 24,
        availabilityPercent: 99.9,
        downtimeSeconds: 86,
        incidentCount: 1,
      },
      availability7d: {
        windowHours: 168,
        availabilityPercent: 99.95,
        downtimeSeconds: 302,
        incidentCount: 1,
      },
      availability30d: {
        windowHours: 720,
        availabilityPercent: 99.98,
        downtimeSeconds: 518,
        incidentCount: 1,
      },
    };
  }
  if (path.includes("ListNotificationChannels")) {
    return {
      channels: [
        {
          id: "chan-1",
          type: "webhook",
          name: "Ops Webhook",
          enabled: true,
          description: "Ops alerts",
        },
      ],
    };
  }
  if (path.includes("ListAlertPolicies")) {
    return {
      policies: [
        {
          id: "pol-1",
          environmentId: "env-1",
          enabled: true,
          notifyOn: ["unhealthy", "recovered"],
          cooldownMinutes: 15,
          sendRecoveryNotification: true,
          channelIds: ["chan-1"],
        },
      ],
    };
  }
  if (path.includes("ListDeployments")) {
    return {
      deployments: [
        {
          id: "dep-1",
          serviceId: "svc-1",
          environmentId: "env-1",
          healthEnabled: true,
          alertsEnabled: true,
          alertCooldownMinutes: 15,
        },
      ],
    };
  }
  if (path.includes("ListInstances")) {
    return {
      instances: [
        {
          id: "inst-1",
          deploymentId: "dep-1",
          name: "checkout-a",
          address: "10.0.0.1",
          port: 8080,
          enabled: true,
        },
      ],
    };
  }
  if (path.includes("ListEndpoints")) {
    return {
      endpoints: [
        {
          id: "end-1",
          instanceId: "inst-1",
          name: "healthz",
          protocol: "PROTOCOL_HTTP",
          port: 8080,
          path: "/healthz",
          enabled: true,
          primary: true,
        },
      ],
    };
  }
  if (path.includes("RegisterRuntime")) {
    return {
      deployment: {
        id: "dep-1",
        serviceId: "svc-1",
        environmentId: "env-1",
        healthEnabled: true,
        alertsEnabled: true,
        alertCooldownMinutes: 15,
      },
      instance: {
        id: "inst-2",
        deploymentId: "dep-1",
        name: "checkout-prod-02",
        address: "10.0.0.2",
        port: 0,
        enabled: true,
      },
      endpoints: [
        {
          id: "end-2",
          instanceId: "inst-2",
          name: "http",
          protocol: "PROTOCOL_HTTP",
          port: 8080,
          path: "/",
          enabled: true,
          primary: true,
        },
      ],
    };
  }
  if (path.includes("GetInstanceHealthState")) {
    return {
      state: {
        instanceId: "inst-1",
        currentState: "HEALTH_STATE_UNHEALTHY",
        consecutiveSuccesses: 0,
        consecutiveFailures: 2,
        lastCheckTime: "2026-08-12T12:00:00Z",
      },
    };
  }
  return {};
}
