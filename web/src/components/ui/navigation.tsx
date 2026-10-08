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
type PaginationProps =
  | {
      page: number;
      pageCount: number;
      onPageChange: (page: number) => void;
      mode?: "numbered";
    }
  | {
      mode: "cursor";
      range: string;
      page?: number;
      canPrevious: boolean;
      canNext: boolean;
      disabled?: boolean;
      onPrevious: () => void;
      onNext: () => void;
      pageSizeControl?: React.ReactNode;
      pageCount?: never;
    };

export function Pagination(props: PaginationProps) {
  const pagination =
    props.mode === "cursor"
      ? {
          label: props.range,
          page: props.page,
          pageSizeControl: props.pageSizeControl,
          previousDisabled: props.disabled || !props.canPrevious,
          nextDisabled: props.disabled || !props.canNext,
          previous: props.onPrevious,
          next: props.onNext,
        }
      : {
          label: `Page ${props.page} of ${props.pageCount}`,
          pageSizeControl: undefined,
          previousDisabled: props.page <= 1,
          nextDisabled: props.page >= props.pageCount,
          previous: () => props.onPageChange(props.page - 1),
          next: () => props.onPageChange(props.page + 1),
        };
  return (
    <nav
      aria-label="Pagination"
      className="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--border)] px-4 py-2 text-xs text-[var(--text-muted)]"
    >
      <span>
        {pagination.label}
        {pagination.page ? (
          <span className="sr-only">Page {pagination.page}</span>
        ) : null}
      </span>
      <div className="flex flex-wrap items-center justify-end gap-3">
        {pagination.pageSizeControl}
        <div className="flex gap-2">
          <Button
            size="sm"
            disabled={pagination.previousDisabled}
            onClick={pagination.previous}
          >
            Previous
          </Button>
          <Button
            size="sm"
            disabled={pagination.nextDisabled}
            onClick={pagination.next}
          >
            Next
          </Button>
        </div>
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
