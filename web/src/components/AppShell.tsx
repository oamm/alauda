import { ReactNode } from "react";
import {
  Activity,
  AlertTriangle,
  Bell,
  Boxes,
  Building2,
  Clock3,
  LayoutDashboard,
  LogOut,
  Moon,
  RefreshCw,
  Shield,
  Sun,
} from "lucide-react";

import { Environment } from "../api";
import { ActiveView } from "../types";
import { Button, Select } from "./ui";

type AppShellProps = {
  canAdmin: boolean;
  activeView: ActiveView;
  children: ReactNode;
  currentEnvironmentName: string;
  darkMode: boolean;
  environments: Environment[];
  openIncidentCount: number;
  selectedEnvironmentId: string;
  onEnvironmentChange: (environmentId: string) => void;
  onRefresh: () => void;
  onLogout: () => void;
  onSecurityOpen: () => void;
  onToggleDarkMode: () => void;
  onViewChange: (view: ActiveView) => void;
};

const navItems: Array<{
  icon: typeof LayoutDashboard;
  label: string;
  view: ActiveView;
}> = [
  { icon: LayoutDashboard, label: "Dashboard", view: "dashboard" },
  { icon: Boxes, label: "Services", view: "services" },
  { icon: Building2, label: "Environments", view: "environments" },
  { icon: Activity, label: "Health", view: "health" },
  { icon: AlertTriangle, label: "Incidents", view: "incidents" },
  { icon: Bell, label: "Alerts", view: "alerts" },
  { icon: Clock3, label: "Events", view: "events" },
  { icon: Shield, label: "Security", view: "security" },
];

export function AppShell({
  activeView,
  canAdmin,
  children,
  currentEnvironmentName,
  darkMode,
  environments,
  openIncidentCount,
  selectedEnvironmentId,
  onEnvironmentChange,
  onRefresh,
  onLogout,
  onSecurityOpen,
  onToggleDarkMode,
  onViewChange,
}: AppShellProps) {
  return (
    <div className={darkMode ? "alauda-shell dark" : "alauda-shell"}>
      <aside className="alauda-sidebar">
        <div className="alauda-brand">
          <div className="alauda-brand-mark" aria-hidden="true">
            A
          </div>
          <div>
            <strong>Alauda</strong>
            <span>Service Registry</span>
          </div>
        </div>
        <nav aria-label="Primary navigation" className="alauda-nav">
          <span className="alauda-nav-label">Workspace</span>
          {navItems
            .filter(
              ({ view }) =>
                canAdmin || (view !== "alerts" && view !== "security"),
            )
            .map(({ icon: Icon, label, view }) => (
              <button
                aria-current={activeView === view ? "page" : undefined}
                className={
                  activeView === view
                    ? "alauda-nav-item active"
                    : "alauda-nav-item"
                }
                key={view}
                onClick={() =>
                  view === "security" ? onSecurityOpen() : onViewChange(view)
                }
                type="button"
              >
                <Icon aria-hidden="true" size={16} />
                <span>{label}</span>
                {view === "incidents" && openIncidentCount > 0 ? (
                  <strong aria-hidden="true">{openIncidentCount}</strong>
                ) : null}
              </button>
            ))}
        </nav>
        <div className="alauda-system-status">
          <div className="alauda-status-heading">
            <span>Incidents</span>
          </div>
          <p>{openIncidentCount} open incident{openIncidentCount === 1 ? "" : "s"}</p>
        </div>
        <Button
          className="alauda-logout"
          onClick={onLogout}
          type="button"
          variant="ghost"
        >
          <LogOut size={16} />
          Log out
        </Button>
      </aside>
      <main className="alauda-workspace">
        <header className="alauda-topbar">
          <div>
            <p className="alauda-context-label">{currentEnvironmentName}</p>
          </div>
          <div className="alauda-topbar-actions">
            <div className="alauda-environment-field">
              <span>Environment</span>
              <Select
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
              </Select>
            </div>
            <Button
              aria-label="Refresh data"
              onClick={onRefresh}
              size="sm"
              title="Refresh data"
              type="button"
              variant="ghost"
            >
              <RefreshCw size={16} />
            </Button>
            <Button
              aria-label={darkMode ? "Light mode" : "Dark mode"}
              onClick={onToggleDarkMode}
              size="sm"
              title={darkMode ? "Switch to light mode" : "Switch to dark mode"}
              type="button"
              variant="ghost"
            >
              {darkMode ? <Sun size={16} /> : <Moon size={16} />}
            </Button>
          </div>
        </header>
        <div className="alauda-content">{children}</div>
      </main>
    </div>
  );
}
