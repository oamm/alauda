import type {
  Environment,
  HealthCheck,
  HealthStateView,
  Service,
  ServiceDeployment,
  ServiceInstance,
} from "../api";

export function healthOverviewRows(data: {
  instances: ServiceInstance[];
  deployments: ServiceDeployment[];
  services: Service[];
  environments: Environment[];
  healthChecks: HealthCheck[];
  healthStates: HealthStateView[];
}) {
  const deployments = new Map(data.deployments.map((d) => [d.id, d]));
  const services = new Map(data.services.map((s) => [s.id, s]));
  const environments = new Map(data.environments.map((e) => [e.id, e]));
  const states = new Map(data.healthStates.map((s) => [s.instanceId, s]));
  const checksByInstance = new Map<string, HealthCheck[]>();
  for (const check of data.healthChecks)
    checksByInstance.set(check.instanceId, [
      ...(checksByInstance.get(check.instanceId) || []),
      check,
    ]);
  return data.instances.map((instance) => {
    const deployment = deployments.get(instance.deploymentId);
    const service = services.get(deployment?.serviceId || "");
    const environment = environments.get(deployment?.environmentId || "");
    const checks = checksByInstance.get(instance.id) || [];
    const enabledChecks = checks.filter((c) => c.enabled);
    const state = states.get(instance.id);
    const actual = state?.currentState
      ?.replace("HEALTH_STATE_", "")
      .toLowerCase();
    const status =
      actual === "healthy" || actual === "unhealthy" ? actual : "unknown";
    const reason = !instance.enabled
      ? "Instance disabled"
      : !checks.length
        ? "No health check configured"
        : !enabledChecks.length
          ? "Monitoring disabled"
          : status === "unknown"
            ? state?.lastCheckTime
              ? "Awaiting health transition"
              : "Current state unavailable"
            : "Monitoring enabled";
    return {
      instance,
      service,
      environment,
      checks,
      enabledChecks,
      status,
      reason,
    };
  });
}
