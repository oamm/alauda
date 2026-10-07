import { createContext, useContext, useId, useState, type HTMLAttributes, type ReactNode } from "react";
import { cn } from "../../lib/utils";

type TabsContextValue = {
  value: string;
  setValue: (value: string) => void;
  baseId: string;
};

const TabsContext = createContext<TabsContextValue | null>(null);

function useTabs() {
  const context = useContext(TabsContext);
  if (!context) throw new Error("Tabs components must be used inside Tabs.");
  return context;
}

type TabsProps = HTMLAttributes<HTMLDivElement> & {
  defaultValue?: string;
  value?: string;
  onValueChange?: (value: string) => void;
  children: ReactNode;
};

export function Tabs({ defaultValue = "", value: controlledValue, onValueChange, className, children, ...props }: TabsProps) {
  const [uncontrolledValue, setUncontrolledValue] = useState(defaultValue);
  const value = controlledValue ?? uncontrolledValue;
  const baseId = useId();
  const setValue = (nextValue: string) => {
    if (controlledValue === undefined) setUncontrolledValue(nextValue);
    onValueChange?.(nextValue);
  };

  return <TabsContext.Provider value={{ value, setValue, baseId }}><div className={cn(className)} {...props}>{children}</div></TabsContext.Provider>;
}

export function TabsList({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return <div role="tablist" className={cn("flex max-w-full items-center gap-1 overflow-x-auto", className)} {...props} />;
}

type TabsTriggerProps = HTMLAttributes<HTMLButtonElement> & { value: string; children: ReactNode };

export function TabsTrigger({ value, className, onClick, onKeyDown, children, ...props }: TabsTriggerProps) {
  const tabs = useTabs();
  const selected = tabs.value === value;

  function moveFocus(direction: 1 | -1, current: HTMLElement | null = document.activeElement as HTMLElement | null) {
    const triggers = current?.parentElement ? Array.from(current.parentElement.querySelectorAll<HTMLElement>('[role="tab"]')) : [];
    const index = triggers.indexOf(current as HTMLElement);
    if (index < 0 || !triggers.length) return;
    const next = triggers[(index + direction + triggers.length) % triggers.length];
    next.focus();
    next.click();
  }

  return <button
    id={`${tabs.baseId}-tab-${value}`}
    aria-controls={`${tabs.baseId}-content-${value}`}
    aria-selected={selected}
    role="tab"
    tabIndex={selected ? 0 : -1}
    className={cn("tab-trigger", selected && "active", className)}
    onClick={(event) => { tabs.setValue(value); onClick?.(event); }}
    onKeyDown={(event) => {
      if (event.key === "ArrowRight" || event.key === "ArrowDown") { event.preventDefault(); moveFocus(1, event.currentTarget); }
      if (event.key === "ArrowLeft" || event.key === "ArrowUp") { event.preventDefault(); moveFocus(-1, event.currentTarget); }
      if (event.key === "Home" || event.key === "End") {
        event.preventDefault();
        const triggers = Array.from((event.currentTarget.parentElement)?.querySelectorAll<HTMLElement>('[role="tab"]') ?? []);
        const next = event.key === "Home" ? triggers[0] : triggers[triggers.length - 1];
        next?.focus();
        next?.click();
      }
      onKeyDown?.(event);
    }}
    {...props}
  >{children}</button>;
}

type TabsContentProps = HTMLAttributes<HTMLDivElement> & { value: string; children: ReactNode };

export function TabsContent({ value, className, children, ...props }: TabsContentProps) {
  const tabs = useTabs();
  const selected = tabs.value === value;
  if (!selected) return null;
  return <div id={`${tabs.baseId}-content-${value}`} aria-labelledby={`${tabs.baseId}-tab-${value}`} role="tabpanel" className={cn("tab-content", className)} {...props}>{children}</div>;
}
