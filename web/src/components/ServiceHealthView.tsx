import { useEffect, useState } from "react";
import {
  HealthCheck,
  HealthResult,
  HealthStateView,
  ServiceInstance,
  Endpoint,
  AvailabilitySummary,
  listHealthResults,
} from "../api";
import { formatTimestamp, pluralize } from "../utils/format";
import {
  Alert,
  Button,
  DefinitionList,
  DialogBody,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Drawer,
  DrawerContent,
  EmptyState,
  Inline,
  ResourceList,
  ResourceRow,
  Section,
  Skeleton,
  Stat,
  StatGroup,
  StatusBadge,
} from "./ui";

type Props = {
  checks: HealthCheck[];
  states: HealthStateView[];
  instances: ServiceInstance[];
  endpoints: Endpoint[];
  availability: {
    availability24h?: AvailabilitySummary;
    availability7d?: AvailabilitySummary;
    availability30d?: AvailabilitySummary;
  };
  error: string;
  saving: boolean;
  onRun: (id: string) => Promise<void>;
  onConfigure: (instance: ServiceInstance) => void;
  onEdit: (check: HealthCheck) => void;
  onDelete: (check: HealthCheck) => void;
  onInstances: () => void;
  onResults: (checkId?: string) => void;
  onChecks?: () => void;
  showAvailability?: boolean;
  status: string;
};

function checkType(check: HealthCheck) {
  return check.type.replace("HEALTH_CHECK_TYPE_", "");
}
function resultObservation(result: HealthResult) {
  return (
    result.errorMessage ||
    result.errorType ||
    (typeof result.latencyMs === "number"
      ? `${result.latencyMs} ms`
      : "Duration not recorded")
  );
}

export function ServiceHealthView(props: Props) {
  const [selection, setSelection] = useState("");

  const [results, setResults] = useState<HealthResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [resultsError, setResultsError] = useState("");
  const [refresh, setRefresh] = useState(0);
  const [running, setRunning] = useState("");
  const checkIds = JSON.stringify(props.checks.map((check) => check.id).sort());
  useEffect(() => {
    let active = true;
    const ids = JSON.parse(checkIds) as string[];
    setResults([]);
    setResultsError("");
    setLoading(ids.length > 0);
    Promise.all(ids.map((id) => listHealthResults(id)))
      .then((pages) => {
        if (!active) return;
        const allowed = new Set(ids);
        const unique = new Map(
          pages
            .flat()
            .filter((result) => allowed.has(result.healthCheckId))
            .map((result) => [result.id, result]),
        );
        setResults(
          [...unique.values()].sort(
            (a, b) =>
              (Date.parse(b.timestamp ?? "") || 0) -
              (Date.parse(a.timestamp ?? "") || 0),
          ),
        );
      })
      .catch((error) => {
        if (active)
          setResultsError(
            error instanceof Error
              ? error.message
              : "Unable to load health results",
          );
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [checkIds, refresh]);

  const selected = props.checks.find((check) => check.id === selection);
  const stateFor = (instance: ServiceInstance) =>
    props.states.find((state) => state.instanceId === instance.id)
      ?.currentState;
  const healthy = props.instances.filter(
    (instance) => stateFor(instance) === "HEALTH_STATE_HEALTHY",
  ).length;
  const unhealthy = props.instances.filter(
    (instance) => stateFor(instance) === "HEALTH_STATE_UNHEALTHY",
  ).length;
  const degraded = props.instances.filter(
    (instance) => stateFor(instance) === "HEALTH_STATE_DEGRADED",
  ).length;
  const disabled = props.instances.filter(
    (instance) => stateFor(instance) === "HEALTH_STATE_DISABLED",
  ).length;
  const unknown = props.instances.length - healthy - unhealthy - degraded - disabled;
  const overall = props.status;
  const monitored = props.instances.filter((instance) =>
    props.states.some((state) => state.instanceId === instance.id && state.monitored),
  ).length;
  const windows: Array<[string, AvailabilitySummary | undefined]> = [
    ["24 hours", props.availability.availability24h],
    ["7 days", props.availability.availability7d],
    ["30 days", props.availability.availability30d],
  ];
  function target(check: HealthCheck) {
    const instance = props.instances.find(
      (instance) => instance.id === check.instanceId,
    );
    const endpoint = props.endpoints.find(
      (endpoint) => endpoint.id === check.endpointId,
    );
    return instance
      ? `${instance.name} · ${instance.address}${endpoint ? ` · ${endpoint.protocol.replace("PROTOCOL_", "")} :${endpoint.port}${endpoint.path || ""}` : instance.port ? `:${instance.port}` : ""}`
      : "Target not available in this scope";
  }
  async function run(check: HealthCheck) {
    setRunning(check.id);
    try {
      await props.onRun(check.id);
      setRefresh((value) => value + 1);
    } finally {
      setRunning("");
    }
  }
  function historyContent(items: HealthResult[]) {
    return (
      <ResourceList label="Health results">
        {items.map((result) => {
          const check = props.checks.find(
            (check) => check.id === result.healthCheckId,
          );
          return (
            <ResourceRow
              key={result.id}
              title={formatTimestamp(result.timestamp)}
              description={
                <>
                  <span>
                    {check?.name || "Health check"} ·{" "}
                    {props.instances.find(
                      (instance) => instance.id === result.instanceId,
                    )?.name || "Instance unavailable"}
                  </span>
                  <span>{resultObservation(result)}</span>
                </>
              }
              status={
                <StatusBadge
                  status={result.success ? "healthy" : "unhealthy"}
                />
              }
            />
          );
        })}
      </ResourceList>
    );
  }
  return (
    <div className="service-section-new">
      <Section
        title="Current status"
        divider={false}
        action={<StatusBadge status={overall} />}
      >
        <StatGroup label="Current health">
          <Stat label="Instances" value={props.instances.length} />
          <Stat label="Monitored" value={monitored} />
          <Stat label="Healthy" value={healthy} />
          <Stat label="Unhealthy" value={unhealthy} />
          <Stat label="Unknown" value={unknown} />
          {degraded > 0 ? <Stat label="Degraded" value={degraded} /> : null}
          {disabled > 0 ? <Stat label="Disabled" value={disabled} /> : null}
        </StatGroup>
      </Section>
      {props.showAvailability !== false ? (
        <Section title="Availability">
          <StatGroup label="Availability windows" columns={3}>
            {windows.map(([label, summary]) => (
              <Stat
                key={label}
                label={label}
                value={
                  typeof summary?.availabilityPercent === "number"
                    ? `${summary.availabilityPercent.toFixed(2)}%`
                    : "No data"
                }
                detail={
                  summary &&
                  typeof summary.incidentCount === "number" &&
                  typeof summary.downtimeSeconds === "number"
                    ? `${pluralize(summary.incidentCount, "incident")} · ${Math.round(summary.downtimeSeconds / 60)}m downtime`
                    : undefined
                }
              />
            ))}
          </StatGroup>
          {!windows.some(
            ([, summary]) => typeof summary?.availabilityPercent === "number",
          ) ? (
            <p className="mt-2 text-xs text-[var(--text-muted)]">
              Availability data will appear after health results are collected.
            </p>
          ) : null}
        </Section>
      ) : null}
      <Section
        title="Health checks"
        action={
          <Inline>
            {props.checks.length > 5 && props.onChecks ? (
              <Button size="sm" variant="link" onClick={props.onChecks}>
                View all {props.checks.length} checks
              </Button>
            ) : null}
            {props.instances[0] ? (
              <Button
                size="sm"
                onClick={() => props.onConfigure(props.instances[0])}
              >
                Create health check
              </Button>
            ) : undefined}
          </Inline>
        }
      >
        {props.checks.length ? (
          <ResourceList label="Health checks">
            {props.checks.slice(0, 5).map((check) => {
              const latest = results.find(
                (result) => result.healthCheckId === check.id,
              );
              return (
                <ResourceRow
                  key={check.id}
                  title={
                    <Button
                      variant="link"
                      onClick={() => setSelection(check.id)}
                    >
                      {check.name}
                    </Button>
                  }
                  description={
                    <>
                      <span>
                        {checkType(check)} · every {check.intervalSeconds}s
                      </span>
                      <span>{target(check)}</span>
                      {latest ? (
                        <span>
                          Last result:{" "}
                          {latest.success ? "Healthy" : "Unhealthy"} ·{" "}
                          {resultObservation(latest)}
                        </span>
                      ) : null}
                    </>
                  }
                  status={
                    <StatusBadge
                      status={check.enabled ? "enabled" : "disabled"}
                    />
                  }
                  action={
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={!!running}
                      onClick={() => {
                        void run(check);
                      }}
                    >
                      {running === check.id ? "Running" : "Run check"}
                    </Button>
                  }
                />
              );
            })}
            {props.checks.length > 5 && props.onChecks ? (
              <div className="px-4 py-2 text-xs text-[var(--text-muted)]">
                Showing 5 of {props.checks.length} checks.
              </div>
            ) : null}
          </ResourceList>
        ) : (
          <EmptyState
            title="No health checks configured"
            description="Configure monitoring to track this service's health and availability."
            action={
              props.instances[0] ? (
                <Button onClick={() => props.onConfigure(props.instances[0])}>
                  Configure monitoring
                </Button>
              ) : (
                <Button onClick={props.onInstances}>Go to Instances</Button>
              )
            }
          />
        )}
      </Section>
      {props.checks.length > 0 ? (
        <Section
          title="Recent results"
          action={
            <Button size="sm" variant="link" onClick={() => props.onResults()}>
              View all results
            </Button>
          }
        >
          {loading ? (
            <Skeleton className="h-12" />
          ) : resultsError ? (
            <Alert tone="danger" title="Health results unavailable">
              <span>{resultsError}</span>
              <Button
                size="sm"
                onClick={() => setRefresh((value) => value + 1)}
              >
                Retry
              </Button>
            </Alert>
          ) : results.length ? (
            historyContent(results.slice(0, 5))
          ) : (
            <EmptyState
              title="No health results yet"
              description="Run a check manually or wait for the next scheduled execution."
            />
          )}
        </Section>
      ) : null}
      <Drawer
        open={!!selected}
        onOpenChange={(open) => {
          if (!open) {
            setSelection("");
          }
        }}
      >
        <DrawerContent size="lg">
          <DialogHeader>
            <DialogTitle>{selected?.name || "Result history"}</DialogTitle>
            <DialogDescription>
              {selected
                ? target(selected)
                : "Latest recorded executions in the current service scope."}
            </DialogDescription>
          </DialogHeader>
          <DialogBody className="max-h-none">
            {props.error ? <Alert tone="danger" title={props.error} /> : null}
            {selected ? (
              <div className="grid gap-4">
                <Section title="Configuration" divider={false}>
                  <DefinitionList>
                    <dt>Type</dt>
                    <dd>{checkType(selected)}</dd>
                    <dt>State</dt>
                    <dd>
                      <StatusBadge
                        status={selected.enabled ? "enabled" : "disabled"}
                      />
                    </dd>
                    <dt>Path</dt>
                    <dd>{selected.metadata?.path || "Not specified"}</dd>
                    <dt>Expected status</dt>
                    <dd>
                      {selected.metadata?.expectedStatus || "Not specified"}
                    </dd>
                    <dt>Interval</dt>
                    <dd>{selected.intervalSeconds}s</dd>
                    <dt>Timeout</dt>
                    <dd>{selected.timeoutSeconds}s</dd>
                    <dt>Failure threshold</dt>
                    <dd>{selected.failuresBeforeUnhealthy}</dd>
                    <dt>Recovery threshold</dt>
                    <dd>{selected.successesBeforeHealthy}</dd>
                  </DefinitionList>
                </Section>
                <Section title="Current state">
                  <DefinitionList>
                    <dt>Status</dt>
                    <dd>
                      <StatusBadge
                        status={
                          props.states.find(
                            (state) => state.instanceId === selected.instanceId,
                          )?.currentState || "unknown"
                        }
                      />
                    </dd>
                    <dt>Consecutive failures</dt>
                    <dd>
                      {props.states.find(
                        (state) => state.instanceId === selected.instanceId,
                      )?.consecutiveFailures ?? "Not available"}
                    </dd>
                    <dt>Consecutive successes</dt>
                    <dd>
                      {props.states.find(
                        (state) => state.instanceId === selected.instanceId,
                      )?.consecutiveSuccesses ?? "Not available"}
                    </dd>
                  </DefinitionList>
                </Section>
                <Section
                  title="Recent executions"
                  action={
                    <Button
                      size="sm"
                      variant="link"
                      onClick={() => props.onResults(selected.id)}
                    >
                      View all results
                    </Button>
                  }
                >
                  {results.filter(
                    (result) => result.healthCheckId === selected.id,
                  ).length ? (
                    historyContent(
                      results
                        .filter(
                          (result) => result.healthCheckId === selected.id,
                        )
                        .slice(0, 3),
                    )
                  ) : (
                    <p className="text-xs text-[var(--text-muted)]">
                      No executions recorded.
                    </p>
                  )}
                </Section>
              </div>
            ) : null}
          </DialogBody>
          <DialogFooter>
            {selected ? (
              <Inline>
                <Button
                  disabled={!!running}
                  onClick={() => {
                    void run(selected);
                  }}
                >
                  Run check
                </Button>
                <Button
                  onClick={() => {
                    setSelection("");
                    props.onEdit(selected);
                  }}
                >
                  Edit check
                </Button>
                <Button
                  variant="danger"
                  disabled={props.saving}
                  onClick={() => props.onDelete(selected)}
                >
                  Delete check
                </Button>
              </Inline>
            ) : null}
            <Button
              onClick={() => {
                setSelection("");
              }}
            >
              Done
            </Button>
          </DialogFooter>
        </DrawerContent>
      </Drawer>
    </div>
  );
}
