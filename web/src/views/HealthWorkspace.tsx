import { Dispatch, FormEvent, SetStateAction } from "react";

import {
  HealthCheck,
  HealthResult,
  HealthStateView,
  Incident,
  ServiceInstance,
} from "../api";
import {
  ActionGroup,
  Button,
  EmptyState,
  PageHeader,
  StatusBadge,
  Tabs,
} from "../components/OperationsUI";
import { MetricCard } from "../components/Cards";
import { formatTimestamp } from "../utils/format";

export type HealthFormValues = {
  instanceId: string;
  endpointId: string;
  name: string;
  type: string;
  intervalSeconds: number;
  timeoutSeconds: number;
  failuresBeforeUnhealthy: number;
  successesBeforeHealthy: number;
  description: string;
  path: string;
  expectedStatus: string;
};

type HealthSection = "overview" | "checks" | "results";

type HealthWorkspaceProps = {
  filteredHealthInstances: ServiceInstance[];
  healthStateByInstanceId: Map<string, HealthStateView>;
  healthStatusFilter: string;
  setHealthStatusFilter: (value: string) => void;
  healthStates: HealthStateView[];
  instances: ServiceInstance[];
  incidents: Incident[];
  healthChecks: HealthCheck[];
  selectedEnvironmentId: string;
  loading: boolean;
  selectedHealthCheck?: HealthCheck;
  selectedHealthCheckId: string;
  onSelectHealthCheck: (id: string) => Promise<void>;
  latestState: HealthStateView | null;
  runningHealthCheck: boolean;
  onRunHealthCheck: () => void;
  onEditHealthCheck: (check: HealthCheck) => void;
  healthResults: HealthResult[];
  healthSection: HealthSection;
  setHealthSection: Dispatch<SetStateAction<HealthSection>>;
  showCreateHealth: boolean;
  setShowCreateHealth: (show: boolean) => void;
  healthForm: HealthFormValues;
  setHealthForm: Dispatch<SetStateAction<HealthFormValues>>;
  savingHealthCheck: boolean;
  onCreateHealthCheck: (event: FormEvent<HTMLFormElement>) => void;
  formatCheckType: (value: string) => string;
  formatHealthState: (value: string) => string;
};

export function HealthWorkspace({
  filteredHealthInstances,
  healthStateByInstanceId,
  healthStatusFilter,
  setHealthStatusFilter,
  healthStates,
  instances,
  incidents,
  healthChecks,
  selectedEnvironmentId,
  loading,
  selectedHealthCheck,
  selectedHealthCheckId,
  onSelectHealthCheck,
  latestState,
  runningHealthCheck,
  onRunHealthCheck,
  onEditHealthCheck,
  healthResults,
  healthSection,
  setHealthSection,
  showCreateHealth,
  setShowCreateHealth,
  healthForm,
  setHealthForm,
  savingHealthCheck,
  onCreateHealthCheck,
  formatCheckType,
  formatHealthState,
}: HealthWorkspaceProps) {
  return (
    <section className="content-grid health-workspace">
      <PageHeader
        title="Health"
        context={`${filteredHealthInstances.length} instances · ${healthChecks.filter((check) => check.enabled).length} enabled checks`}
        description="Start with system health, then open a check when you need configuration or execution history."
      />
      <Tabs
        ariaLabel="Health views"
        items={["overview", "checks", "results"].map((section) => ({
          value: section,
          label: section[0].toUpperCase() + section.slice(1),
        }))}
        value={healthSection}
        onChange={(value) => setHealthSection(value as HealthSection)}
      />

      {healthSection === "overview" ? (
        <>
          <div className="panel service-wide-panel">
            <div className="panel-heading">
              <h2>Global status</h2>
              <span>{selectedEnvironmentId ? "Filtered" : "All environments"}</span>
            </div>
            <div className="metrics-grid">
              <MetricCard label="Healthy" value={healthStates.filter((state) => state.currentState === "HEALTH_STATE_HEALTHY").length} detail={`${healthStates.length} states loaded`} />
              <MetricCard label="Unhealthy" value={healthStates.filter((state) => state.currentState === "HEALTH_STATE_UNHEALTHY").length} detail={`${incidents.filter((incident) => incident.state === "INCIDENT_STATE_OPEN").length} open incidents`} />
              <MetricCard label="Unknown" value={Math.max(instances.length - healthStates.length, 0)} detail={`${instances.length} instances`} />
              <MetricCard label="Checks" value={healthChecks.length} detail={`${healthChecks.filter((check) => check.enabled).length} enabled`} />
            </div>
          </div>
          <div className="panel service-wide-panel">
            <div className="panel-heading"><h2>Instance health</h2><span>{filteredHealthInstances.length} shown</span></div>
            <div className="filter-bar"><label>Status<select value={healthStatusFilter} onChange={(event) => setHealthStatusFilter(event.target.value)}><option value="all">All statuses</option><option value="healthy">Healthy</option><option value="unhealthy">Unhealthy</option><option value="unknown">Unknown</option></select></label></div>
            <div className="table">
              {filteredHealthInstances.length === 0 ? <EmptyState title="No health targets" description="No instances match the selected health filter." /> : filteredHealthInstances.map((instance) => {
                const state = healthStateByInstanceId.get(instance.id);
                return <div className="instance-row" key={instance.id}><div><strong>{instance.name}</strong><span>{instance.address}</span></div><span>{state?.currentState ? <StatusBadge status={formatHealthState(state.currentState)} /> : "unknown"}</span><span>{state ? `${state.consecutiveSuccesses} ok / ${state.consecutiveFailures} fail` : "No checks"}</span><span>{formatTimestamp(state?.lastCheckTime)}</span></div>;
              })}
            </div>
          </div>
          <div className="panel"><div className="panel-heading"><h2>Checks requiring attention</h2><Button size="sm" variant="ghost" type="button" onClick={() => setHealthSection("checks")}>View all checks</Button></div><div className="service-list">{healthChecks.filter((check) => !check.enabled).slice(0, 5).map((check) => <button className="row" key={check.id} type="button" onClick={() => { void onSelectHealthCheck(check.id); setHealthSection("checks"); }}><strong>{check.name}</strong><span>Disabled</span></button>)}{healthChecks.every((check) => check.enabled) ? <EmptyState title="No checks require attention" description="All configured health checks are enabled." /> : null}</div></div>
        </>
      ) : healthSection === "checks" ? (
        <>
          <div className="panel">
            <div className="panel-heading"><div><h2>Health checks</h2><span>{loading ? "Loading" : `${healthChecks.length} total`}</span></div><Button variant="primary" type="button" onClick={() => setShowCreateHealth(true)}>+ Create health check</Button></div>
            {healthChecks.length === 0 && !loading ? <EmptyState title="No health checks" description="Create a health check or register a service with monitoring enabled to start tracking state." /> : <div className="service-list">{healthChecks.map((check) => <button className={check.id === selectedHealthCheckId ? "row selected" : "row"} key={check.id} type="button" onClick={() => { void onSelectHealthCheck(check.id); }}><strong>{check.name}</strong><span>{formatCheckType(check.type)}</span><StatusBadge status={check.enabled ? "enabled" : "disabled"} /></button>)}</div>}
          </div>
          {selectedHealthCheck ? <div className="panel detail-panel"><div className="panel-heading"><div><span className="breadcrumb">Health / {selectedHealthCheck.name}</span><h2>{selectedHealthCheck.name}</h2></div><ActionGroup><Button disabled={runningHealthCheck} loading={runningHealthCheck} variant="primary" onClick={onRunHealthCheck} type="button">Run check</Button><Button type="button" onClick={() => onEditHealthCheck(selectedHealthCheck)}>Edit</Button></ActionGroup></div><StatusBadge status={latestState?.currentState ? formatHealthState(latestState.currentState) : "unknown"} /><dl className="detail-list"><dt>ID</dt><dd>{selectedHealthCheck.id}</dd><dt>Type</dt><dd>{formatCheckType(selectedHealthCheck.type)}</dd><dt>Interval</dt><dd>{selectedHealthCheck.intervalSeconds}s</dd><dt>Timeout</dt><dd>{selectedHealthCheck.timeoutSeconds}s</dd><dt>Counters</dt><dd>{latestState ? `${latestState.consecutiveSuccesses} success / ${latestState.consecutiveFailures} failure` : "Run the check to load current counters"}</dd></dl></div> : <EmptyState title="Select a health check" description="Pick a check to inspect its configuration and recent executions." />}
        </>
      ) : (
        <div className="panel"><div className="panel-heading"><div><h2>Recent results</h2><span>{selectedHealthCheck ? selectedHealthCheck.name : "Select a check from Checks"}</span></div></div><div className="table">{healthResults.length === 0 ? <EmptyState title="No health results" description="Run a check manually or wait for scheduled execution to record a result." /> : healthResults.map((result) => <div className="health-result-row" key={result.id}><strong><StatusBadge status={result.success ? "healthy" : "failed"} /></strong><span>{result.statusCode || result.errorType || "n/a"}</span><span>{result.latencyMs ?? 0}ms</span><span>{formatTimestamp(result.timestamp)}</span></div>)}</div></div>
      )}

      {showCreateHealth ? <div className="dialog-backdrop"><form className="panel form-panel dialog-panel" onSubmit={onCreateHealthCheck}><div className="panel-heading"><h2>Create health check</h2><button type="button" onClick={() => setShowCreateHealth(false)}>Cancel</button></div><label>Instance reference<input required value={healthForm.instanceId} onChange={(event) => setHealthForm((current) => ({ ...current, instanceId: event.target.value }))} /></label><label>Endpoint reference<input value={healthForm.endpointId} onChange={(event) => setHealthForm((current) => ({ ...current, endpointId: event.target.value }))} /></label><label>Name<input required value={healthForm.name} onChange={(event) => setHealthForm((current) => ({ ...current, name: event.target.value }))} /></label><label>Type<select value={healthForm.type} onChange={(event) => setHealthForm((current) => ({ ...current, type: event.target.value }))}><option value="HEALTH_CHECK_TYPE_HTTP">HTTP</option><option value="HEALTH_CHECK_TYPE_HTTPS">HTTPS</option><option value="HEALTH_CHECK_TYPE_TCP">TCP</option><option value="HEALTH_CHECK_TYPE_GRPC">gRPC</option></select></label><label>Path<input value={healthForm.path} onChange={(event) => setHealthForm((current) => ({ ...current, path: event.target.value }))} /></label><label>Expected status<input value={healthForm.expectedStatus} onChange={(event) => setHealthForm((current) => ({ ...current, expectedStatus: event.target.value }))} /></label><div className="form-fields-grid"><label>Interval seconds<input min="1" type="number" value={healthForm.intervalSeconds} onChange={(event) => setHealthForm((current) => ({ ...current, intervalSeconds: Number(event.target.value) }))} /></label><label>Timeout seconds<input min="1" type="number" value={healthForm.timeoutSeconds} onChange={(event) => setHealthForm((current) => ({ ...current, timeoutSeconds: Number(event.target.value) }))} /></label></div><button className="button-primary" disabled={savingHealthCheck} type="submit">{savingHealthCheck ? "Creating" : "Create health check"}</button></form></div> : null}
    </section>
  );
}
