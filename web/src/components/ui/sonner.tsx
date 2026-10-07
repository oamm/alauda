import { Toaster as Sonner } from "sonner";

export function Toaster() { return <Sonner closeButton position="bottom-right" toastOptions={{ className: "border-[var(--border)] bg-[var(--surface)] text-[var(--text)]" }} />; }
