import * as DialogPrimitive from "@radix-ui/react-dialog";
import { X } from "lucide-react";
import { cn } from "../../lib/utils";
import { useRef } from "react";

export const Dialog = DialogPrimitive.Root;
export const DialogTrigger = DialogPrimitive.Trigger;
export const DialogClose = DialogPrimitive.Close;
export const DialogTitle = DialogPrimitive.Title;
export const DialogDescription = DialogPrimitive.Description;

export function DialogContent({
  className,
  children,
  size = "md",
  placement = "center",
  onOpenAutoFocus,
  onCloseAutoFocus,
  ...props
}: React.ComponentPropsWithoutRef<typeof DialogPrimitive.Content> & {
  size?: "sm" | "md" | "lg";
  placement?: "center" | "right";
}) {
  const opener = useRef<HTMLElement | null>(null);
  // Capture the external opener before nested focus scopes or autoFocus fields move focus.
  const activeElement = document.activeElement;
  if (
    activeElement instanceof HTMLElement &&
    activeElement !== document.body &&
    !activeElement.closest('[role="dialog"], [role="menu"]')
  )
    opener.current = activeElement;
  return (
    <DialogPrimitive.Portal>
      <DialogPrimitive.Overlay className="fixed inset-0 z-40 bg-black/35" />
      <DialogPrimitive.Content
        onOpenAutoFocus={(event) => {
          onOpenAutoFocus?.(event);
        }}
        onCloseAutoFocus={(event) => {
          onCloseAutoFocus?.(event);
          if (
            !event.defaultPrevented &&
            opener.current?.isConnected &&
            opener.current !== document.body
          ) {
            event.preventDefault();
            opener.current.focus();
          }
        }}
        className={cn(
          "fixed z-50 grid w-[calc(100%-32px)] gap-0 overflow-hidden border border-[var(--border)] bg-[var(--surface)] text-[var(--text)] shadow-[var(--shadow-dialog)] focus:outline-none",
          placement === "center"
            ? "left-1/2 top-1/2 max-h-[calc(100vh-48px)] -translate-x-1/2 -translate-y-1/2 rounded-[var(--radius-panel)]"
            : "right-0 top-0 h-[100dvh] max-h-[100dvh] grid-rows-[auto_minmax(0,1fr)_auto]",
          {
            "max-w-sm": size === "sm",
            "max-w-2xl": size === "lg",
            "max-w-lg": size === "md",
          },
          className,
        )}
        {...props}
      >
        {children}
        <DialogPrimitive.Close
          className="absolute right-3 top-3 inline-flex h-8 w-8 items-center justify-center rounded-md text-[var(--text-muted)] hover:bg-[var(--surface-muted)] hover:text-[var(--text)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)]"
          aria-label="Close dialog"
        >
          <X size={16} />
        </DialogPrimitive.Close>
      </DialogPrimitive.Content>
    </DialogPrimitive.Portal>
  );
}

export function DialogHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "min-w-0 break-words border-b border-[var(--border)] px-5 py-3 pr-12",
        className,
      )}
      {...props}
    />
  );
}
export function DialogBody({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "max-h-[min(70vh,640px)] overflow-y-auto px-5 py-4",
        className,
      )}
      {...props}
    />
  );
}
export function DialogFooter({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "flex flex-wrap justify-end gap-2 border-t border-[var(--border)] px-5 py-3",
        className,
      )}
      {...props}
    />
  );
}
