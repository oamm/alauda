import { useEffect, useState } from "react";
import { ArrowRight, X } from "lucide-react";
import {
  queryHealthResults,
  type HealthResultRecord,
  type Service,
  type Environment,
  type ServiceInstance,
  type Endpoint,
  type HealthCheck,
  type ServiceDeployment,
} from "../api";
import { formatTimestamp } from "../utils/format";
import { allHealthTypes } from "../lib/health-check-form";
import {
  PageHeader,
  FilterBar,
  FormField,
  Input,
  SearchInput,
  Select,
  Button,
  IconButton,
  EmptyState,
  Alert,
  Skeleton,
  Table,
  TableHeader,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  Drawer,
  DrawerContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  DialogFooter,
  DefinitionList,
  StatusBadge,
  ResourceList,
  ResourceRow,
  Inline,
} from "./ui";

function localDateTime(value: string) {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000)
    .toISOString()
    .slice(0, 16);
}
function protocolName(protocol: number) {
  return ["", "HTTP", "HTTPS", "gRPC", "TCP", "UDP"][protocol] || "";
}
function typeName(type: string) {
  return type.replace("HEALTH_CHECK_TYPE_", "");
}
function duration(value: number | null) {
  return value == null ? "—" : `${value} ms`;
}
function relativeTime(value: string) {
  const delta = Math.round((Date.now() - Date.parse(value)) / 60000);
  return delta >= 0 && delta < 60 ? `${Math.max(1, delta)} min ago` : "";
}

type Props = {
  services: Service[];
  environments: Environment[];
  instances: ServiceInstance[];
  endpoints: Endpoint[];
  checks: HealthCheck[];
  deployments: ServiceDeployment[];
  onBack: () => void;
  onService?: (id: string, environmentId: string) => void;
  onInstance?: (id: string, serviceId: string, environmentId: string) => void;
  onCheck?: (id: string) => void;
};

export function HealthResultsPage(props: Props) {
  const [search, setSearch] = useState(window.location.search);
  const [data, setData] = useState<{
    results: HealthResultRecord[];
    nextPageToken: string;
  }>({ results: [], nextPageToken: "" });
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);
  const [selection, setSelection] = useState<HealthResultRecord>();
  const [retry, setRetry] = useState(0);
  const params = new URLSearchParams(search);
  const get = (key: string) => params.get(key) || "";
  useEffect(() => {
    const pop = () => setSearch(window.location.search);
    window.addEventListener("popstate", pop);
    return () => window.removeEventListener("popstate", pop);
  }, []);
  function go(next: URLSearchParams) {
    window.history.pushState(
      {},
      "",
      `/health/results${next.size ? `?${next}` : ""}`,
    );
    setSearch(window.location.search);
  }
  function update(key: string, value: string) {
    if ((key === "from" || key === "to") && value) {
      const date = new Date(value);
      if (Number.isNaN(date.getTime())) return;
      value = date.toISOString();
    }
    const next = new URLSearchParams(search);
    if (value) next.set(key, value);
    else next.delete(key);
    next.delete("pageToken");
    if (key === "serviceId" || key === "environmentId") {
      next.delete("instanceId");
      next.delete("endpointId");
      next.delete("checkId");
    }
    if (key === "instanceId") {
      next.delete("endpointId");
      next.delete("checkId");
    }
    if (key === "endpointId") next.delete("checkId");
    go(next);
  }
  function clearAll() {
    go(new URLSearchParams());
  }
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError("");
    const query = new URLSearchParams(search);
    for (const key of ["from", "to"]) {
      const value = query.get(key);
      if (value) {
        const date = new Date(value);
        if (Number.isNaN(date.getTime())) {
          setError("Enter valid dates.");
          setLoading(false);
          return;
        }
        query.set(key, date.toISOString());
      }
    }
    if (
      query.get("from") &&
      query.get("to") &&
      query.get("from")! > query.get("to")!
    ) {
      setError("From must not be after To.");
      setLoading(false);
      return;
    }
    queryHealthResults(query, controller.signal)
      .then((value) => {
        if (!controller.signal.aborted) setData(value);
      })
      .catch((e) => {
        if (!controller.signal.aborted)
          setError(e instanceof Error ? e.message : "Unable to load results");
      })
      .finally(() => {
        if (!controller.signal.aborted) setLoading(false);
      });
    return () => controller.abort();
  }, [search, retry]);
  const scopedInstances = props.instances.filter((instance) => {
    const deployment = props.deployments.find(
      (value) => value.id === instance.deploymentId,
    );
    return (
      (!get("serviceId") || deployment?.serviceId === get("serviceId")) &&
      (!get("environmentId") ||
        deployment?.environmentId === get("environmentId"))
    );
  });
  const scopedEndpoints = props.endpoints.filter(
    (endpoint) =>
      scopedInstances.some((instance) => instance.id === endpoint.instanceId) &&
      (!get("instanceId") || endpoint.instanceId === get("instanceId")),
  );
  const scopedChecks = props.checks.filter(
    (check) =>
      scopedInstances.some((instance) => instance.id === check.instanceId) &&
      (!get("instanceId") || check.instanceId === get("instanceId")) &&
      (!get("endpointId") || check.endpointId === get("endpointId")),
  );
  const advancedKeys = [
    "environmentId",
    "instanceId",
    "endpointId",
    "port",
    "type",
    "checkId",
    "sort",
    "pageSize",
  ].filter((key) => get(key));
  const token = Number(get("pageToken") || 0);
  const size = Number(get("pageSize") || 25);
  const first = data.results.length ? token + 1 : 0;
  const last = token + data.results.length;
  const hasNext = !!data.nextPageToken;
  function select(
    key: string,
    label: string,
    items: { id: string; name: string }[],
  ) {
    return (
      <FormField label={label}>
        <Select value={get(key)} onChange={(e) => update(key, e.target.value)}>
          <option value="">All</option>
          {items.map((item) => (
            <option key={item.id} value={item.id}>
              {item.name}
            </option>
          ))}
        </Select>
      </FormField>
    );
  }
  const chips: Array<[string, string, string]> = [];
  const chip = (key: string, label: string, value: string) => {
    if (value) chips.push([key, label, value]);
  };
  chip(
    "serviceId",
    "Service",
    props.services.find((item) => item.id === get("serviceId"))?.displayName ||
      props.services.find((item) => item.id === get("serviceId"))?.name ||
      "Service",
  );
  chip(
    "environmentId",
    "Environment",
    props.environments.find((item) => item.id === get("environmentId"))?.name ||
      "Environment",
  );
  chip(
    "instanceId",
    "Instance",
    props.instances.find((item) => item.id === get("instanceId"))?.name ||
      "Instance",
  );
  chip(
    "endpointId",
    "Endpoint",
    props.endpoints.find((item) => item.id === get("endpointId"))?.name ||
      "Endpoint",
  );
  chip("port", "Port", get("port"));
  chip("type", "Type", typeName(get("type")));
  chip(
    "checkId",
    "Health check",
    props.checks.find((item) => item.id === get("checkId"))?.name ||
      "Health check",
  );
  chip("status", "Result", get("status"));
  chip("search", "Search", get("search"));
  return (
    <div className="grid min-w-0 gap-3">
      <PageHeader
        title="Health results"
        description="Search and inspect health-check executions across services."
        action={
          <Button variant="ghost" onClick={props.onBack}>
            Back to Health
          </Button>
        }
      />
      <FilterBar
        collapseAfter={5}
        persistentDisclosure
        compact
        activeAdvanced={advancedKeys.length}
      >
        <FormField label="From">
          <Input
            type="datetime-local"
            value={localDateTime(get("from"))}
            onChange={(e) => update("from", e.target.value)}
          />
        </FormField>
        <FormField label="To">
          <Input
            type="datetime-local"
            value={localDateTime(get("to"))}
            onChange={(e) => update("to", e.target.value)}
          />
        </FormField>
        {select(
          "serviceId",
          "Service",
          props.services.map((s) => ({
            id: s.id,
            name: s.displayName || s.name,
          })),
        )}
        {select("status", "Result", [
          { id: "healthy", name: "Healthy" },
          { id: "unhealthy", name: "Unhealthy" },
        ])}
        <FormField label="Search">
          <SearchInput
            value={get("search")}
            placeholder="Search targets, checks, failures"
            onChange={(e) => update("search", e.target.value)}
          />
        </FormField>
        {select("environmentId", "Environment", props.environments)}
        {select("instanceId", "Instance", scopedInstances)}
        {select("endpointId", "Endpoint", scopedEndpoints)}
        <FormField label="Port">
          <Input
            type="number"
            min={1}
            max={65535}
            value={get("port")}
            onChange={(e) => update("port", e.target.value)}
          />
        </FormField>
        {select(
          "type",
          "Check type",
          allHealthTypes.map((type) => ({
            id: `HEALTH_CHECK_TYPE_${type}`,
            name: type,
          })),
        )}
        {select("checkId", "Health check", scopedChecks)}
        <FormField label="Order">
          <Select
            value={get("sort") || "newest"}
            onChange={(e) => update("sort", e.target.value)}
          >
            <option value="newest">Newest first</option>
            <option value="oldest">Oldest first</option>
          </Select>
        </FormField>
        <FormField label="Page size">
          <Select
            value={String(size)}
            onChange={(e) => update("pageSize", e.target.value)}
          >
            <option value="25">25</option>
            <option value="50">50</option>
            <option value="100">100</option>
          </Select>
        </FormField>
        <Button variant="ghost" aria-label="Reset filters" onClick={clearAll}>
          Clear all
        </Button>
      </FilterBar>
      {chips.length ? (
        <div
          className="flex flex-wrap items-center gap-1.5 text-xs"
          aria-label="Active filters"
        >
          <span className="mr-1 text-[var(--text-muted)]">Filtered by</span>
          {chips.map(([key, label, value]) => (
            <span
              className="inline-flex items-center gap-1 rounded-[var(--radius-control)] border border-[var(--border)] bg-[var(--surface-muted)] px-2 py-1"
              key={key}
            >
              {label}: {value}
              <button
                type="button"
                aria-label={`Remove ${label} filter`}
                onClick={() => update(key, "")}
              >
                <X size={12} />
              </button>
            </span>
          ))}
          <Button size="sm" variant="link" onClick={clearAll}>
            Clear all
          </Button>
        </div>
      ) : null}
      {loading ? (
        <>
          <TableSkeleton />
          <div className="sr-only" role="status">
            Loading health results
          </div>
        </>
      ) : error ? (
        <Alert tone="danger" title={error || "Unable to load health results"}>
          <Button size="sm" onClick={() => setRetry((value) => value + 1)}>
            Retry
          </Button>
        </Alert>
      ) : !data.results.length ? (
        <EmptyState
          title="No health results found"
          description={
            params.size
              ? "Try changing the selected filters or date range."
              : "No health checks have produced results yet. Run a Health Check or wait for scheduled execution."
          }
        />
      ) : (
        <>
          <div className="hidden min-w-0 md:block">
            <Table aria-label="Health result history">
              <TableHeader>
                <TableRow>
                  <TableHead>Timestamp</TableHead>
                  <TableHead>Target</TableHead>
                  <TableHead>Check</TableHead>
                  <TableHead>Result</TableHead>
                  <TableHead>Duration</TableHead>
                  <TableHead>
                    <span className="sr-only">Open details</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.results.map((result) => (
                  <TableRow
                    key={result.id}
                    className="cursor-pointer"
                    onClick={() => setSelection(result)}
                  >
                    <TableCell>
                      <div>{formatTimestamp(result.timestamp)}</div>
                      {relativeTime(result.timestamp) ? (
                        <div className="text-xs text-[var(--text-muted)]">
                          {relativeTime(result.timestamp)}
                        </div>
                      ) : null}
                    </TableCell>
                    <TableCell>
                      <div>
                        {result.service} · {result.environment}
                      </div>
                      <div>{result.instance}</div>
                      <div className="text-xs text-[var(--text-muted)]">
                        {result.endpoint || "Instance address"} ·{" "}
                        {protocolName(result.protocol)} :{result.port}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div>{result.check}</div>
                      <div className="text-xs text-[var(--text-muted)]">
                        {typeName(result.type)}
                      </div>
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        status={result.success ? "healthy" : "unhealthy"}
                      />
                    </TableCell>
                    <TableCell>
                      <span
                        className={
                          result.latencyMs == null
                            ? "text-[var(--text-muted)]"
                            : undefined
                        }
                        title={result.latencyMs == null ? "Duration not recorded" : undefined}
                      >
                        {duration(result.latencyMs)}
                        {result.latencyMs == null ? (
                          <span className="sr-only">Not recorded</span>
                        ) : null}
                      </span>
                    </TableCell>
                    <TableCell>
                  <IconButton
                      label={`Details for ${result.check}`}
                        variant="ghost"
                        onClick={(e) => {
                          e.stopPropagation();
                          setSelection(result);
                        }}
                      >
                        <ArrowRight size={15} />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          <div className="md:hidden">
            <ResourceList label="Health result history">
              {data.results.map((result) => (
                <ResourceRow
                  key={result.id}
                  title={
                    <Button variant="link" onClick={() => setSelection(result)}>
                      {result.check}
                    </Button>
                  }
                  description={
                    <>
                      <span>{formatTimestamp(result.timestamp)}</span>
                      <span>
                        {result.service} · {result.environment} ·{" "}
                        {result.instance}
                      </span>
                      <span>
                        {result.endpoint || "Instance address"} ·{" "}
                        {protocolName(result.protocol)} :{result.port}
                      </span>
                        <span
                          className={
                            result.latencyMs == null
                              ? "text-[var(--text-muted)]"
                              : undefined
                          }
                          title={result.latencyMs == null ? "Duration not recorded" : undefined}
                        >
                          {duration(result.latencyMs)}
                          {result.latencyMs == null ? <span className="sr-only">Not recorded</span> : null}
                        </span>
                    </>
                  }
                  status={
                    <StatusBadge
                      status={result.success ? "healthy" : "unhealthy"}
                    />
                  }
                  action={
                    <IconButton
                    label={`Details for ${result.check}`}
                      variant="ghost"
                      onClick={() => setSelection(result)}
                    >
                      <ArrowRight size={15} />
                    </IconButton>
                  }
                />
              ))}
            </ResourceList>
          </div>
        </>
      )}
    <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--border)] pt-2 text-xs text-[var(--text-muted)]">
      <span>
        {first}–{last}
        {hasNext ? "+" : ""}
        {token ? <span className="sr-only">Page {Math.floor(token / size) + 1}</span> : null}
      </span>
        <div className="flex gap-2">
          <Button
            size="sm"
            disabled={loading || !token}
            onClick={() => {
              const next = new URLSearchParams(search);
              const previous = Math.max(0, token - size);
              if (previous) next.set("pageToken", String(previous));
              else next.delete("pageToken");
              go(next);
            }}
          >
            Previous
          </Button>
          <Button
            size="sm"
            disabled={loading || !hasNext}
            onClick={() => {
              const next = new URLSearchParams(search);
              next.set("pageToken", data.nextPageToken);
              go(next);
            }}
          >
            Next
          </Button>
        </div>
      </div>
      <ResultDetail
        result={selection}
        onClose={() => setSelection(undefined)}
        onService={props.onService}
        onInstance={props.onInstance}
        onCheck={props.onCheck}
      />
    </div>
  );
}

function TableSkeleton() {
  return (
    <div className="grid gap-px rounded-[var(--radius-panel)] border border-[var(--border)] p-2">
      {[1, 2, 3, 4, 5].map((row) => (
        <Skeleton key={row} className="h-14" />
      ))}
    </div>
  );
}

function ResultDetail(props: {
  result?: HealthResultRecord;
  onClose: () => void;
  onService?: (id: string, environmentId: string) => void;
  onInstance?: (id: string, serviceId: string, environmentId: string) => void;
  onCheck?: (id: string) => void;
}) {
  const result = props.result;
  if (!result) return null;
  return (
    <Drawer open onOpenChange={(open) => !open && props.onClose()}>
      <DrawerContent size="lg">
        <DialogHeader>
          <div className="flex items-start justify-between gap-3">
            <div>
              <DialogTitle>{result.check}</DialogTitle>
              <p className="mt-1 text-xs text-[var(--text-muted)]">
                {result.service} · {result.environment} · {result.instance}
              </p>
            </div>
            <StatusBadge status={result.success ? "healthy" : "unhealthy"} />
          </div>
        </DialogHeader>
        <DialogBody>
          <div className="grid gap-4">
            <section>
              <h3 className="m-0 text-sm font-semibold">Execution</h3>
              <DefinitionList>
                <dt>Timestamp</dt>
                <dd>{formatTimestamp(result.timestamp)}</dd>
                <dt>Result</dt>
                <dd>
                  <StatusBadge
                    status={result.success ? "healthy" : "unhealthy"}
                  />
                </dd>
                <dt>Duration</dt>
                <dd
                  className={
                    result.latencyMs == null
                      ? "text-[var(--text-muted)]"
                      : undefined
                  }
                >
                  {duration(result.latencyMs)}
                </dd>
              </DefinitionList>
            </section>
            <section>
              <h3 className="m-0 text-sm font-semibold">Target</h3>
              <DefinitionList>
                <dt>Service</dt>
                <dd>{result.service}</dd>
                <dt>Environment</dt>
                <dd>{result.environment}</dd>
                <dt>Instance</dt>
                <dd>{result.instance}</dd>
                <dt>Endpoint</dt>
                <dd>{result.endpoint || "Instance address"}</dd>
                <dt>Address</dt>
                <dd>{result.address}</dd>
                <dt>Protocol / port</dt>
                <dd>
                  {protocolName(result.protocol) || "Instance address"} :
                  {result.port}
                </dd>
                <dt>Path</dt>
                <dd>{result.path || "Not specified"}</dd>
              </DefinitionList>
            </section>
            <section>
              <h3 className="m-0 text-sm font-semibold">Health Check</h3>
              <DefinitionList>
                <dt>Name</dt>
                <dd>{result.check}</dd>
                <dt>Type</dt>
                <dd>{typeName(result.type)}</dd>
                <dt>Expected status</dt>
                <dd>{result.expectedStatus || "Not specified"}</dd>
              </DefinitionList>
            </section>
            {!result.success ? (
              <section>
                <h3 className="m-0 text-sm font-semibold">Failure</h3>
                <DefinitionList>
                  <dt>Reason</dt>
                  <dd>{result.errorType || "Unhealthy result"}</dd>
                  <dt>Diagnostic</dt>
                  <dd className="break-words">
                    {result.errorMessage || "No diagnostic message recorded."}
                  </dd>
                </DefinitionList>
              </section>
            ) : null}
          </div>
        </DialogBody>
        <DialogFooter>
          <Inline>
            {props.onCheck ? (
              <Button
                size="sm"
                variant="secondary"
                onClick={() => props.onCheck?.(result.checkId)}
              >
                View Health Check
              </Button>
            ) : null}
            {props.onInstance ? (
              <Button
                size="sm"
                variant="secondary"
                onClick={() =>
                  props.onInstance?.(
                    result.instanceId,
                    result.serviceId,
                    result.environmentId,
                  )
                }
              >
                View Instance
              </Button>
            ) : null}
            {props.onService ? (
              <Button
                size="sm"
                variant="secondary"
                onClick={() =>
                  props.onService?.(result.serviceId, result.environmentId)
                }
              >
                View Service
              </Button>
            ) : null}
            <Button size="sm" onClick={props.onClose}>
              Close
            </Button>
          </Inline>
        </DialogFooter>
      </DrawerContent>
    </Drawer>
  );
}
