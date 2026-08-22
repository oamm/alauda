import { ReactNode } from "react";

type StatusTone =
  | "healthy"
  | "degraded"
  | "unhealthy"
  | "unknown"
  | "open"
  | "resolved"
  | "enabled"
  | "disabled"
  | "warning"
  | "critical"
  | "neutral";

const statusLabels: Record<string, string> = {
  healthy: "Healthy",
  degraded: "Degraded",
  unhealthy: "Unhealthy",
  unknown: "Unknown",
  open: "Open",
  acknowledged: "Acknowledged",
  resolved: "Resolved",
  enabled: "Enabled",
  disabled: "Disabled",
  success: "Success",
  warning: "Warning",
  critical: "Critical",
  failed: "Failed",
  neutral: "Unknown",
};

const toneAliases: Record<string, StatusTone> = {
  acknowledged: "warning",
  failed: "critical",
  success: "healthy",
};

export function StatusBadge({
  status,
  label,
}: {
  status: string;
  label?: string;
}) {
  const normalized = normalizeStatus(status);
  const tone = toneAliases[normalized] ?? normalized;

  return (
    <span className={`status-badge status-badge-${tone}`}>
      <span aria-hidden="true" />
      {label ?? statusLabels[normalized] ?? toTitleCase(normalized)}
    </span>
  );
}

export function PageHeader({
  title,
  context,
  description,
  action,
}: {
  title: string;
  context?: string;
  description?: string;
  action?: ReactNode;
}) {
  return (
    <div className="page-header">
      <div>
        <div className="breadcrumb">Alauda / {title}</div>
        <h1>{title}</h1>
        {context ? <p>{context}</p> : null}
        {description ? <small>{description}</small> : null}
      </div>
      {action ? <div className="page-actions">{action}</div> : null}
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="empty-state">
      <strong>{title}</strong>
      <p>{description}</p>
      {action}
    </div>
  );
}

export function LoadingRows({ rows = 4 }: { rows?: number }) {
  return (
    <div className="loading-rows" aria-label="Loading rows">
      {Array.from({ length: rows }).map((_, index) => (
        <span key={index} />
      ))}
    </div>
  );
}

export function ResourceLink({
  children,
  onClick,
}: {
  children: ReactNode;
  onClick?: () => void;
}) {
  if (!onClick) {
    return <span className="resource-link static">{children}</span>;
  }

  return (
    <button className="resource-link" type="button" onClick={onClick}>
      {children}
    </button>
  );
}

export function normalizeStatus(value: string) {
  const normalized = value
    .replace("HEALTH_STATE_", "")
    .replace("INCIDENT_STATE_", "")
    .toLowerCase();
  return normalized || "unknown";
}

function toTitleCase(value: string) {
  return value.slice(0, 1).toUpperCase() + value.slice(1);
}
