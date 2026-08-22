import { AvailabilitySummary } from "../api";
import { formatAvailability, formatDuration } from "../utils/format";

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
  return (
    <div className="availability-card">
      <span>{label}</span>
      <strong>{formatAvailability(summary?.availabilityPercent)}</strong>
      <small>
        {formatDuration(summary?.downtimeSeconds)} downtime /{" "}
        {summary?.incidentCount ?? 0} incidents
      </small>
    </div>
  );
}
