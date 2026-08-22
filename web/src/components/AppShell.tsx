import { ReactNode } from "react";

import { Environment } from "../api";
import { ActiveView } from "../types";

type AppShellProps = {
  activeView: ActiveView;
  children: ReactNode;
  currentEnvironmentName: string;
  darkMode: boolean;
  environments: Environment[];
  degradedServiceCount: number;
  openIncidentCount: number;
  selectedEnvironmentId: string;
  onEnvironmentChange: (environmentId: string) => void;
  onRefresh: () => void;
  onSecurityOpen: () => void;
  onToggleDarkMode: () => void;
  onViewChange: (view: ActiveView) => void;
};

const navItems: Array<{ icon: string; label: string; view: ActiveView }> = [
  { icon: "D", label: "Dashboard", view: "dashboard" },
  { icon: "S", label: "Services", view: "services" },
  { icon: "E", label: "Environments", view: "environments" },
  { icon: "H", label: "Health", view: "health" },
  { icon: "I", label: "Incidents", view: "incidents" },
  { icon: "A", label: "Alerts", view: "alerts" },
  { icon: "T", label: "Events", view: "events" },
  { icon: "U", label: "Security", view: "security" },
];

export function AppShell({
  activeView,
  children,
  currentEnvironmentName,
  darkMode,
  environments,
  degradedServiceCount,
  openIncidentCount,
  selectedEnvironmentId,
  onEnvironmentChange,
  onRefresh,
  onSecurityOpen,
  onToggleDarkMode,
  onViewChange,
}: AppShellProps) {
  return (
    <div
      className={
        darkMode
          ? "app-shell dark bg-ink-950 text-slate-100"
          : "app-shell bg-slate-100 text-slate-950"
      }
    >
      <aside className="sidebar">
        <div className="brand-lockup">
          <div className="brand-mark" aria-hidden="true">
            A
          </div>
          <div>
            <h1>Alauda</h1>
            <p>Service Registry</p>
          </div>
        </div>
        <nav aria-label="Primary">
          {navItems.map((item) => (
            <button
              className={activeView === item.view ? "active" : ""}
              key={item.view}
              onClick={() => {
                if (item.view === "security") {
                  onSecurityOpen();
                  return;
                }
                onViewChange(item.view);
              }}
              type="button"
            >
              <span className="nav-icon" aria-hidden="true">
                {item.icon}
              </span>
              <span>{item.label}</span>
              {item.view === "incidents" && openIncidentCount > 0 ? (
                <strong className="nav-badge" aria-hidden="true">
                  {openIncidentCount}
                </strong>
              ) : null}
            </button>
          ))}
        </nav>
        <div className="sidebar-status">
          <span>System status</span>
          <strong>{openIncidentCount === 0 ? "Healthy" : "Attention needed"}</strong>
          <small>
            {openIncidentCount === 0
              ? "All monitored services are operating normally."
              : `${degradedServiceCount} degraded service${degradedServiceCount === 1 ? "" : "s"} / ${openIncidentCount} open incident${openIncidentCount === 1 ? "" : "s"}.`}
          </small>
        </div>
      </aside>

      <main className="workspace">
        <header className="toolbar">
          <div className="toolbar-copy">
            <span>Registry command center</span>
            <h2>{currentEnvironmentName}</h2>
          </div>
          <div className="toolbar-control">
            <span>Environment</span>
            <select
              aria-label="Environment selector"
              value={selectedEnvironmentId}
              onChange={(event) => onEnvironmentChange(event.target.value)}
            >
              <option value="">All environments</option>
              {environments.map((environment) => (
                <option key={environment.id} value={environment.id}>
                  {environment.name}
                </option>
              ))}
            </select>
          </div>
          <div className="toolbar-actions">
            <button type="button" onClick={onRefresh}>
              Refresh
            </button>
            <button type="button" onClick={onToggleDarkMode}>
              {darkMode ? "Light mode" : "Dark mode"}
            </button>
          </div>
        </header>
        {children}
      </main>
    </div>
  );
}
