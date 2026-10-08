import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import { MoreHorizontal } from "lucide-react";
import { IconButton } from "./button";
import { useRef } from "react";

export function ActionMenu({
  label,
  items,
}: {
  label: string;
  items: Array<{ label: string; onSelect: () => void; disabled?: boolean }>;
}) {
  const trigger = useRef<HTMLButtonElement>(null);
  const pendingAction = useRef<(() => void) | null>(null);
  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <IconButton ref={trigger} label={label} variant="ghost">
          <MoreHorizontal size={16} />
        </IconButton>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          align="end"
          sideOffset={4}
          className="alauda-options"
          onCloseAutoFocus={(event) => {
            if (pendingAction.current) {
              event.preventDefault();
              trigger.current?.focus();
              const action = pendingAction.current;
              pendingAction.current = null;
              action();
            }
          }}
        >
          {items.map((item) => (
            <DropdownMenu.Item
              key={item.label}
              onSelect={() => {
                // Open subsequent surfaces after the menu releases its focus trap.
                pendingAction.current = item.onSelect;
              }}
              disabled={item.disabled}
              className="alauda-menu-item"
            >
              {item.label}
            </DropdownMenu.Item>
          ))}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
