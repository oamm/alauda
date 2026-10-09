import type { ReactNode } from "react";
import { useId } from "react";
import { cn } from "../../lib/utils";

export function CatalogRow({
  name,
  metadata,
  status,
  selected,
  onSelect,
  selection,
}: {
  name: string;
  metadata: ReactNode;
  status: ReactNode;
  selected: boolean;
  onSelect: () => void;
  selection?: ReactNode;
}) {
  const id = useId();
  return (
    <li className={cn("alauda-catalog-row", selected && "is-selected")}>
      {selection}
      <button
        type="button"
        className="alauda-catalog-link"
        aria-pressed={selected}
        aria-label={name}
        aria-describedby={`${id}-status ${id}-meta`}
        onClick={onSelect}
      >
        <span className="alauda-catalog-identity">
          <strong title={name}>{name}</strong>
          <span id={`${id}-status`}>{status}</span>
        </span>
        <span id={`${id}-meta`} className="alauda-catalog-meta">
          {metadata}
        </span>
      </button>
    </li>
  );
}
