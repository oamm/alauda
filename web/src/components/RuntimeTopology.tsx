import {
  Endpoint,
  Environment,
  HealthCheck,
  Incident,
  ServiceDeployment,
  ServiceInstance,
} from "../api";
import { formatTimestamp } from "../utils/format";
import { EmptyState, StatusBadge } from "./OperationsUI";

type RuntimeTopologyProps = {
  deployments: ServiceDeployment[];
  endpoints: Endpoint[];
  environments: Environment[];
  healthChecks: HealthCheck[];
  incidents: Incident[];
  instances: ServiceInstance[];
  serviceName: string;
  onRegisterRuntime: () => void;
  onRunHealthCheck: (id: string) => void;
};

export function RuntimeTopology({
  deployments,
  endpoints,
  environments,
  healthChecks,
  incidents,
  instances,
  serviceName,
  onRegisterRuntime,
  onRunHealthCheck,
}: RuntimeTopologyProps) {
  if (deployments.length === 0) {
    return (
      <EmptyState
        title="No runtime topology"
        description={`${serviceName} has catalog metadata, but no deployment, instance, endpoint, or health relationship is registered in the current scope.`}
        action={
          <button type="button" onClick={onRegisterRuntime}>
            Register runtime
          </button>
        }
      />
    );
  }

  return (
    <div className="topology">
      {deployments.map((deployment) => {
        const deploymentInstances = instances.filter(
          (instance) => instance.deploymentId === deployment.id,
        );
        const openIncidents = incidents.filter(
          (incident) =>
            incident.deploymentId === deployment.id &&
            incident.state === "INCIDENT_STATE_OPEN",
        );

        return (
          <section className="topology-deployment" key={deployment.id}>
            <div className="topology-node topology-root">
              <div>
                <span>Deployment</span>
                <strong>{environmentName(environments, deployment.environmentId)}</strong>
              </div>
              <StatusBadge
                status={openIncidents.length > 0 ? "degraded" : "healthy"}
              />
            </div>
            <div className="topology-children">
              {deploymentInstances.length === 0 ? (
                <EmptyState
                  title="No instances"
                  description="This deployment exists, but no concrete runtime targets are registered."
                  action={
                    <button type="button" onClick={onRegisterRuntime}>
                      Add instance
                    </button>
                  }
                />
              ) : (
                deploymentInstances.map((instance) => {
                  const instanceEndpoints = endpoints.filter(
                    (endpoint) => endpoint.instanceId === instance.id,
                  );
                  const instanceChecks = healthChecks.filter(
                    (check) => check.instanceId === instance.id,
                  );
                  const instanceIncidents = incidents.filter(
                    (incident) =>
                      incident.instanceId === instance.id &&
                      incident.state === "INCIDENT_STATE_OPEN",
                  );

                  return (
                    <div className="topology-instance" key={instance.id}>
                      <div className="topology-node">
                        <div>
                          <span>Instance</span>
                          <strong>{instance.name}</strong>
                          <small>
                            {instance.address}:{instance.port || "dynamic"} ·{" "}
                            {formatTimestamp(instance.lastSeenAt)}
                          </small>
                        </div>
                        <StatusBadge
                          status={
                            instanceIncidents.length > 0
                              ? "unhealthy"
                              : instance.enabled
                                ? "enabled"
                                : "disabled"
                          }
                        />
                      </div>
                      <div className="topology-leaves">
                        {instanceEndpoints.map((endpoint) => (
                          <div className="topology-leaf" key={endpoint.id}>
                            <span>Endpoint</span>
                            <strong>
                              {endpoint.name} · {formatProtocol(endpoint.protocol)}{" "}
                              :{endpoint.port}
                              {endpoint.path || ""}
                            </strong>
                            <StatusBadge
                              status={endpoint.enabled ? "enabled" : "disabled"}
                              label={endpoint.primary ? "Primary" : undefined}
                            />
                          </div>
                        ))}
                        {instanceChecks.map((check) => (
                          <div className="topology-leaf" key={check.id}>
                            <span>Health check</span>
                            <strong>
                              {check.name} · {formatCheckType(check.type)} every{" "}
                              {check.intervalSeconds}s
                            </strong>
                            <button
                              type="button"
                              onClick={() => onRunHealthCheck(check.id)}
                            >
                              Run check
                            </button>
                          </div>
                        ))}
                        {instanceEndpoints.length === 0 &&
                        instanceChecks.length === 0 ? (
                          <p className="topology-empty">
                            No endpoints or health checks attached.
                          </p>
                        ) : null}
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </section>
        );
      })}
    </div>
  );
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

function formatCheckType(value: string) {
  return value.replace("HEALTH_CHECK_TYPE_", "").toLowerCase();
}
