import { cn } from "../../lib/utils";

export function Badge({ children, className }: { children: React.ReactNode; className?: string }) {
  return <span className={cn("inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium", className)}>{children}</span>;
}

const labels: Record<string, string> = {
  healthy: "Healthy", degraded: "Degraded", unhealthy: "Unhealthy", unknown: "Unknown",
  enabled: "Enabled", disabled: "Disabled", active: "Active", inactive: "Inactive",
  expired: "Expired", revoked: "Revoked", open: "Open", resolved: "Resolved",
};

const tones: Record<string, string> = {
  healthy: "border-[var(--success-border)] bg-[var(--success-bg)] text-[var(--success-text)]",
  enabled: "border-[var(--success-border)] bg-[var(--success-bg)] text-[var(--success-text)]",
  active: "border-[var(--success-border)] bg-[var(--success-bg)] text-[var(--success-text)]",
  resolved: "border-[var(--success-border)] bg-[var(--success-bg)] text-[var(--success-text)]",
  degraded: "border-[var(--warning-border)] bg-[var(--warning-bg)] text-[var(--warning-text)]",
  open: "border-[var(--warning-border)] bg-[var(--warning-bg)] text-[var(--warning-text)]",
  unhealthy: "border-[var(--danger-border)] bg-[var(--danger-bg)] text-[var(--danger-text)]",
  revoked: "border-[var(--danger-border)] bg-[var(--danger-bg)] text-[var(--danger-text)]",
  expired: "border-[var(--danger-border)] bg-[var(--danger-bg)] text-[var(--danger-text)]",
  disabled: "border-[var(--border)] bg-[var(--surface-muted)] text-[var(--text-muted)]",
  inactive: "border-[var(--border)] bg-[var(--surface-muted)] text-[var(--text-muted)]",
  unknown: "border-[var(--border)] bg-[var(--surface-muted)] text-[var(--text-muted)]",
};

export function StatusBadge({ status, label }: { status: string; label?: string }) {
  const value = status.replace(/^HEALTH_STATE_|^INCIDENT_STATE_/, "").toLowerCase();
  return <Badge className={cn(tones[value] ?? tones.unknown)}><span aria-hidden="true" className="mr-1.5 h-1.5 w-1.5 rounded-full bg-current" />{label ?? labels[value] ?? value}</Badge>;
}
