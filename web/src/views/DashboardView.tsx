import {
  AlertPolicy,
  AvailabilitySummary,
  Environment,
  EventRecord,
  HealthCheck,
  Incident,
  NotificationChannel,
  Service,
} from "../api";
import { AvailabilityCard, MetricCard } from "../components/Cards";
import { EmptyState, PageHeader, StatusBadge } from "../components/OperationsUI";
import { formatAvailability, formatTimestamp } from "../utils/format";

type DashboardViewProps = {
  alertPolicies: AlertPolicy[];
  availability: {
    availability24h?: AvailabilitySummary;
    availability7d?: AvailabilitySummary;
    availability30d?: AvailabilitySummary;
  };
  environments: Environment[];
  events: EventRecord[];
  healthChecks: HealthCheck[];
  incidents: Incident[];
  notificationChannels: NotificationChannel[];
  selectedEnvironmentId: string;
  services: Service[];
};

export function DashboardView({
  alertPolicies,
  availability,
  environments,
  events,
  healthChecks,
  incidents,
  notificationChannels,
  selectedEnvironmentId,
  services,
}: DashboardViewProps) {
  const openIncidents = incidents.filter(
    (incident) => incident.state === "INCIDENT_STATE_OPEN",
  );
  const enabledHealthCheckCount = healthChecks.filter(
    (check) => check.enabled,
  ).length;
  const degradedServiceCount = services.filter((service) =>
    openIncidents.some((incident) => incident.serviceId === service.id),
  ).length;
  const registryState =
    openIncidents.length === 0 ? "Registry healthy" : "Registry degraded";

  return (
    <section className="dashboard-grid">
      <PageHeader
        title="Dashboard"
        context={`${formatAvailability(availability.availability24h?.availabilityPercent)} availability · ${services.length} services · ${degradedServiceCount} degraded · ${openIncidents.length} open incidents`}
        description="Registry health, active breakage, and recent changes in the selected environment scope."
      />

      <div className="operations-summary">
        <div>
          <StatusBadge
            status={openIncidents.length === 0 ? "healthy" : "degraded"}
            label={registryState}
          />
          <strong>
            {formatAvailability(availability.availability24h?.availabilityPercent)}
          </strong>
          <span>
            24h availability · {healthChecks.length} checks ·{" "}
            {alertPolicies.length} alert policies
          </span>
        </div>
        <div className="summary-actions">
          <span>{selectedEnvironmentId ? "Environment scoped" : "Global scope"}</span>
          <StatusBadge
            status={openIncidents.length === 0 ? "healthy" : "open"}
            label={
              openIncidents.length === 0
                ? "No incident action"
                : `${openIncidents.length} needs triage`
            }
          />
        </div>
      </div>

      <div className="metrics-grid">
        <MetricCard
          label="Services"
          value={services.length}
          detail={`${environments.length} environments`}
        />
        <MetricCard
          label="Health checks"
          value={healthChecks.length}
          detail={`${enabledHealthCheckCount} enabled`}
        />
        <MetricCard
          label="Open incidents"
          value={openIncidents.length}
          detail={`${incidents.length} total`}
        />
        <MetricCard
          label="Alert policies"
          value={alertPolicies.length}
          detail={`${notificationChannels.length} channels`}
        />
      </div>

      <div className="panel">
        <div className="panel-heading">
          <h2>Availability</h2>
          <span>{selectedEnvironmentId ? "Filtered" : "Global"}</span>
        </div>
        <div className="availability-grid">
          <AvailabilityCard label="24h" summary={availability.availability24h} />
          <AvailabilityCard label="7d" summary={availability.availability7d} />
          <AvailabilityCard label="30d" summary={availability.availability30d} />
        </div>
      </div>

      <div className="dashboard-split">
        <div className="panel">
          <div className="panel-heading">
            <h2>Active Incidents</h2>
            <span>{openIncidents.length} open</span>
          </div>
          <div className="table">
            {openIncidents.length === 0 ? (
              <EmptyState
                title="No active incidents"
                description="All monitored services are operating normally in this scope."
              />
            ) : (
              openIncidents.slice(0, 6).map((incident) => (
                <div className="incident-triage-row" key={incident.id}>
                  <div>
                    <StatusBadge status="open" />
                    <strong>{incident.reason || "Open incident"}</strong>
                    <span>{incident.impactSummary || "Impact not recorded"}</span>
                  </div>
                  <span>{incident.serviceId || incident.instanceId}</span>
                  <span>{formatTimestamp(incident.openedAt)}</span>
                  <button type="button">Triage</button>
                </div>
              ))
            )}
          </div>
        </div>

        <div className="panel">
          <div className="panel-heading">
            <h2>Recent Events</h2>
            <span>{events.length} loaded</span>
          </div>
          <div className="table">
            {events.length === 0 ? (
              <EmptyState
                title="No recent activity"
                description="No registry events have been recorded for the selected scope."
              />
            ) : (
              events.slice(0, 8).map((event) => (
                <div className="timeline-row" key={event.id}>
                  <time>{formatTimestamp(event.timestamp)}</time>
                  <strong>{event.type}</strong>
                  <span>{event.message}</span>
                  <small>{event.resourceType}</small>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
