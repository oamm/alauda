import { AlertCircle, CheckCircle2, Info, TriangleAlert } from "lucide-react";
import { cn } from "../../lib/utils";

const icons = {
  info: Info,
  success: CheckCircle2,
  warning: TriangleAlert,
  danger: AlertCircle,
};
export function Alert({
  tone = "info",
  title,
  children,
  className,
}: {
  tone?: keyof typeof icons;
  title: string;
  children?: React.ReactNode;
  className?: string;
}) {
  const Icon = icons[tone];
  return (
    <div
      role={tone === "danger" ? "alert" : "status"}
      className={cn(
        "flex gap-3 rounded-[var(--radius-control)] border p-3 text-sm",
        tone === "success" &&
          "border-[var(--success-border)] bg-[var(--success-bg)] text-[var(--success-text)]",
        tone === "warning" &&
          "border-[var(--warning-border)] bg-[var(--warning-bg)] text-[var(--warning-text)]",
        tone === "danger" &&
          "border-[var(--danger-border)] bg-[var(--danger-bg)] text-[var(--danger-text)]",
        tone === "info" &&
          "border-[var(--border)] bg-[var(--surface-muted)] text-[var(--text)]",
        className,
      )}
    >
      <Icon size={16} className="mt-0.5 shrink-0" />
      <div>
        <strong className="font-medium">{title}</strong>
        {children ? (
          <div className="mt-1 text-xs opacity-80">{children}</div>
        ) : null}
      </div>
    </div>
  );
}
