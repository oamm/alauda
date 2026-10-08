import { cn } from "../../lib/utils";
import { Children, useId, useState } from "react";
import { SlidersHorizontal } from "lucide-react";
import { Button } from "./button";

export function Workspace({
  className,
  ...props
}: React.HTMLAttributes<HTMLElement>) {
  return (
    <section {...props} className={cn("alauda-workspace-surface", className)} />
  );
}

export function ResourceList({
  children,
  label,
}: {
  children: React.ReactNode;
  label: string;
}) {
  return (
    <ul aria-label={label} className="alauda-resource-list">
      {children}
    </ul>
  );
}
export function ResourceRow({
  title,
  description,
  status,
  action,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  status?: React.ReactNode;
  action?: React.ReactNode;
}) {
  return (
    <li className="alauda-resource-row">
      <div className="alauda-resource-content">
        {title}
        {description ? (
          <div className="alauda-resource-description">{description}</div>
        ) : null}
      </div>
      <div className="alauda-resource-actions">
        {status}
        {action}
      </div>
    </li>
  );
}

export function Section({
  title,
  description,
  action,
  children,
  divider = true,
  className,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
  children: React.ReactNode;
  divider?: boolean;
  className?: string;
}) {
  const id = useId();
  return (
    <section
      aria-labelledby={id}
      className={cn(
        "alauda-section",
        divider && "alauda-section-divided",
        className,
      )}
    >
      <header className="alauda-section-header">
        <div>
          <h3 id={id}>{title}</h3>
          {description ? <p>{description}</p> : null}
        </div>
        {action}
      </header>
      <div className="alauda-section-content">{children}</div>
    </section>
  );
}

export function StatGroup({
  children,
  label = "Summary",
  columns = 4,
}: {
  children: React.ReactNode;
  label?: string;
  columns?: 2 | 3 | 4 | 5;
}) {
  return (
    <div
      role="group"
      aria-label={label}
      data-columns={columns}
      className="alauda-stat-group"
    >
      {children}
    </div>
  );
}

export function ActivityList({
  items,
}: {
  items: Array<{ id: string; time: string; event: string; context: string }>;
}) {
  return (
    <ul className="alauda-activity-list">
      {items.map((item) => (
        <li key={item.id}>
          <span className="alauda-activity-time">{item.time}</span>
          <strong>{item.event}</strong>
          <span className="alauda-activity-context">{item.context}</span>
        </li>
      ))}
    </ul>
  );
}

export function Card({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <section
      className={cn(
        "rounded-[var(--radius-panel)] border border-[var(--border)] bg-[var(--surface)] shadow-[var(--shadow-panel)]",
        className,
      )}
      {...props}
    />
  );
}
export function PageHeader({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--border)] pb-3">
      <div>
        <h1 className="m-0 text-xl font-semibold text-[var(--text)]">
          {title}
        </h1>
        {description ? (
          <p className="mt-1 max-w-2xl text-xs text-[var(--text-muted)]">
            {description}
          </p>
        ) : null}
      </div>
      {action ? <div className="flex flex-wrap gap-2">{action}</div> : null}
    </div>
  );
}
export function SectionHeader({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--border)] px-4 py-2.5">
      <div>
        <h2 className="m-0 text-sm font-semibold text-[var(--text)]">
          {title}
        </h2>
        {description ? (
          <p className="mt-0.5 text-xs text-[var(--text-muted)]">
            {description}
          </p>
        ) : null}
      </div>
      {action}
    </div>
  );
}
export function FilterBar({
  children,
  collapseAfter,
  activeAdvanced = 0,
  compact = false,
}: {
  children: React.ReactNode;
  collapseAfter?: number;
  activeAdvanced?: number;
  compact?: boolean;
}) {
  const id = useId();
  const [expanded, setExpanded] = useState(false);
  const controls = Children.toArray(children);
  return (
    <div
      className="alauda-filter-bar"
      data-density={compact ? "compact" : undefined}
    >
      {collapseAfter ? controls.slice(0, collapseAfter) : children}
      {collapseAfter ? (
        <>
          <Button
            className="md:hidden"
            variant="ghost"
            aria-controls={id}
            aria-expanded={expanded}
            onClick={() => setExpanded(!expanded)}
          >
            <SlidersHorizontal size={14} /> More filters
            {activeAdvanced ? ` (${activeAdvanced})` : ""}
          </Button>
          <div
            id={id}
            className="alauda-filter-advanced"
            data-expanded={expanded}
          >
            {controls.slice(collapseAfter)}
          </div>
        </>
      ) : null}
    </div>
  );
}
export function ActionGroup({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-wrap items-center justify-end gap-2">
      {children}
    </div>
  );
}
export function Stat({
  label,
  value,
  detail,
}: {
  label: string;
  value: React.ReactNode;
  detail?: string;
}) {
  return (
    <dl className="alauda-stat">
      <dt>{label}</dt>
      <dd>{value}</dd>
      {detail ? <dd className="alauda-stat-detail">{detail}</dd> : null}
    </dl>
  );
}
export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="alauda-empty-state grid justify-items-start gap-1.5 px-5 py-7 text-left">
      <h3 className="m-0 text-sm font-semibold text-[var(--text)]">{title}</h3>
      <p className="m-0 max-w-md text-sm text-[var(--text-muted)]">
        {description}
      </p>
      {action ? <div className="pt-1.5">{action}</div> : null}
    </div>
  );
}
export function Skeleton({ className }: { className?: string }) {
  return (
    <span
      aria-hidden="true"
      className={cn(
        "block animate-pulse rounded bg-[var(--surface-muted)]",
        className,
      )}
    />
  );
}
