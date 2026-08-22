import { AvailabilitySummary } from "../api";
import {
  availabilitySeverity,
  formatAvailability,
  formatDuration,
  pluralize,
} from "../utils/format";

export function MetricCard({
  label,
  value,
  detail,
}: {
  label: string;
  value: number;
  detail: string;
}) {
  return (
    <div className="metric-card">
      <span>{label}</span>
      <strong>{value}</strong>
      <small>{detail}</small>
    </div>
  );
}

export function AvailabilityCard({
  label,
  summary,
}: {
  label: string;
  summary?: AvailabilitySummary;
}) {
  const severity = availabilitySeverity(summary?.availabilityPercent);
  const severityLabel =
    severity === "critical"
      ? "Critical availability"
      : severity === "degraded"
        ? "Degraded availability"
        : severity === "healthy"
          ? "Healthy availability"
          : "Availability unknown";

  return (
    <div className={`availability-card availability-card-${severity}`}>
      <span>{label}</span>
      <strong>{formatAvailability(summary?.availabilityPercent)}</strong>
      <em>{severityLabel}</em>
      <div className="availability-bar" aria-hidden="true">
        <span
          style={{
            width: `${Math.max(
              0,
              Math.min(100, summary?.availabilityPercent ?? 0),
            )}%`,
          }}
        />
      </div>
      <small>
        {formatDuration(summary?.downtimeSeconds)} downtime ·{" "}
        {pluralize(summary?.incidentCount ?? 0, "incident")}
      </small>
    </div>
  );
}
