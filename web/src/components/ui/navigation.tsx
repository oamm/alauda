import { ChevronRight } from "lucide-react";
import { cn } from "../../lib/utils";
import { Button } from "./button";

export function Breadcrumb({ items }: { items: string[] }) {
  return (
    <nav
      aria-label="Breadcrumb"
      className="flex flex-wrap items-center gap-1 text-xs text-[var(--text-muted)]"
    >
      {items.map((item, index) => (
        <span
          className="inline-flex items-center gap-1"
          key={`${item}-${index}`}
        >
          {index ? <ChevronRight size={12} aria-hidden="true" /> : null}
          <span
            className={cn(index === items.length - 1 && "text-[var(--text)]")}
          >
            {item}
          </span>
        </span>
      ))}
    </nav>
  );
}
export function Pagination({
  page,
  pageCount,
  onPageChange,
}: {
  page: number;
  pageCount: number;
  onPageChange: (page: number) => void;
}) {
  return (
    <nav
      aria-label="Pagination"
      className="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--border)] px-4 py-2 text-xs text-[var(--text-muted)]"
    >
      <span>
        Page {page} of {pageCount}
      </span>
      <div className="flex gap-2">
        <Button
          size="sm"
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          Previous
        </Button>
        <Button
          size="sm"
          disabled={page >= pageCount}
          onClick={() => onPageChange(page + 1)}
        >
          Next
        </Button>
      </div>
    </nav>
  );
}
export function Stack({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("grid gap-4", className)} {...props} />;
}
export function Inline({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("flex flex-wrap items-center gap-2", className)}
      {...props}
    />
  );
}
export function DefinitionList({ children }: { children: React.ReactNode }) {
  return (
    <dl className="grid grid-cols-[minmax(100px,0.4fr)_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm">
      {children}
    </dl>
  );
}
