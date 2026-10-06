import { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger" | "icon";
type ButtonSize = "sm" | "md";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  size?: ButtonSize;
  loading?: boolean;
};

export function Button({
  variant = "secondary",
  size = "md",
  loading = false,
  className = "",
  disabled,
  children,
  ...props
}: ButtonProps) {
  return (
    <button
      {...props}
      aria-busy={loading || undefined}
      className={`ui-button ui-button-${variant} ui-button-${size} ${className}`.trim()}
      disabled={disabled || loading}
    >
      {children}
    </button>
  );
}

export function IconButton({
  label,
  size = "md",
  className = "",
  children,
  ...props
}: Omit<ButtonProps, "variant" | "children"> & {
  label: string;
  children: ReactNode;
}) {
  return (
    <Button
      {...props}
      aria-label={label}
      className={`ui-icon-button ${className}`.trim()}
      size={size}
      variant="icon"
    >
      {children}
    </Button>
  );
}

export function ActionGroup({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={`action-group ${className}`.trim()}>{children}</div>;
}

export function Metadata({ children }: { children: ReactNode }) {
  return <span className="metadata-cluster">{children}</span>;
}

export function Tabs({
  ariaLabel,
  items,
  value,
  onChange,
}: {
  ariaLabel: string;
  items: Array<{ value: string; label: string }>;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="ui-tabs" role="tablist" aria-label={ariaLabel}>
      {items.map((item) => (
        <button
          aria-selected={value === item.value}
          className={value === item.value ? "active" : ""}
          key={item.value}
          onClick={() => onChange(item.value)}
          onKeyDown={(event) => {
            if (event.key !== "ArrowRight" && event.key !== "ArrowLeft") {
              return;
            }
            event.preventDefault();
            const tabs = Array.from(
              event.currentTarget.parentElement?.querySelectorAll<HTMLButtonElement>(
                '[role="tab"]',
              ) ?? [],
            );
            const currentIndex = tabs.indexOf(event.currentTarget);
            const nextIndex =
              (currentIndex + (event.key === "ArrowRight" ? 1 : -1) + tabs.length) %
              tabs.length;
            tabs[nextIndex]?.focus();
            tabs[nextIndex]?.click();
          }}
          role="tab"
          tabIndex={value === item.value ? 0 : -1}
          type="button"
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}

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
  href = "#",
}: {
  children: ReactNode;
  onClick?: () => void;
  href?: string;
}) {
  if (!onClick) {
    return <span className="resource-link static">{children}</span>;
  }

  return (
    <a
      className="resource-link"
      href={href}
      onClick={(event) => {
        event.preventDefault();
        onClick();
      }}
    >
      {children}
    </a>
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
