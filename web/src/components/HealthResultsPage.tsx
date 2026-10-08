import { useEffect, useState } from "react";
import { ArrowLeft, ArrowRight } from "lucide-react";
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
  FilterChip,
  ResourceList,
  ResourceRow,
  Inline,
  Pagination,
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
      get("serviceId"),
  );
  chip(
    "environmentId",
    "Environment",
    props.environments.find((item) => item.id === get("environmentId"))?.name ||
      get("environmentId"),
  );
  chip(
    "instanceId",
    "Instance",
    props.instances.find((item) => item.id === get("instanceId"))?.name ||
      get("instanceId"),
  );
  chip(
    "endpointId",
    "Endpoint",
    props.endpoints.find((item) => item.id === get("endpointId"))?.name ||
      get("endpointId"),
  );
  chip("port", "Port", get("port"));
  chip("type", "Type", typeName(get("type")));
  chip(
    "checkId",
    "Health check",
    props.checks.find((item) => item.id === get("checkId"))?.name ||
      get("checkId"),
  );
  chip("status", "Result", get("status"));
  chip("search", "Search", get("search"));
  for (const [key, label] of [
    ["from", "From"],
    ["to", "To"],
  ]) {
    const value = get(key);
    if (!value) continue;
    const date = new Date(value);
    chip(
      key,
      label,
      Number.isNaN(date.getTime())
        ? value
        : date.toLocaleString([], { dateStyle: "medium", timeStyle: "short" }),
    );
  }
  return (
    <div className="grid min-w-0 gap-3">
      <PageHeader
        title="Health results"
        description="Search and inspect health-check executions across services."
        action={
          <Button variant="ghost" onClick={props.onBack}>
            <ArrowLeft size={14} aria-hidden="true" />
            Back to Health
          </Button>
        }
      />
      <div className="health-results-surface">
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
        </FilterBar>
        {chips.length ? (
          <div
            className="flex flex-wrap items-center gap-2 text-xs"
            aria-label="Active filters"
          >
            {chips.map(([key, label, value]) => (
              <FilterChip
                key={key}
                label={label}
                value={value}
                onRemove={() => update(key, "")}
              />
            ))}
            <Button size="sm" variant="ghost" onClick={clearAll}>
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
              <Table
                aria-label="Health result history"
                className="health-results-table"
              >
                <colgroup>
                  <col className="health-col-timestamp" />
                  <col className="health-col-target" />
                  <col className="health-col-check" />
                  <col className="health-col-result" />
                  <col className="health-col-duration" />
                  <col className="health-col-action" />
                </colgroup>
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
                    <TableRow key={result.id}>
                      <TableCell>
                        <div className="font-medium">
                          {formatTimestamp(result.timestamp)}
                        </div>
                        {relativeTime(result.timestamp) ? (
                          <div className="text-xs text-[var(--text-muted)]">
                            {relativeTime(result.timestamp)}
                          </div>
                        ) : null}
                      </TableCell>
                      <TableCell>
                        <TargetSummary result={result} />
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
                          title={
                            result.latencyMs == null
                              ? "Duration not recorded"
                              : undefined
                          }
                        >
                          {duration(result.latencyMs)}
                          {result.latencyMs == null ? (
                            <span className="sr-only">Not recorded</span>
                          ) : null}
                        </span>
                      </TableCell>
                      <TableCell>
                        <IconButton
                          label="View health result details"
                          title="View details"
                          variant="ghost"
                          onClick={() => setSelection(result)}
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
                      <Button
                        variant="link"
                        onClick={() => setSelection(result)}
                      >
                        {result.check}
                      </Button>
                    }
                    description={
                      <>
                        <span className="font-medium">
                          {formatTimestamp(result.timestamp)}
                        </span>
                        {relativeTime(result.timestamp) ? (
                          <span className="text-[var(--text-muted)]">
                            {relativeTime(result.timestamp)}
                          </span>
                        ) : null}
                        <TargetSummary result={result} />
                        <span>
                          {result.check} · {typeName(result.type)}
                        </span>
                        <span>
                          <StatusBadge
                            status={result.success ? "healthy" : "unhealthy"}
                          />
                        </span>
                        <span
                          className={
                            result.latencyMs == null
                              ? "text-[var(--text-muted)]"
                              : undefined
                          }
                          title={
                            result.latencyMs == null
                              ? "Duration not recorded"
                              : undefined
                          }
                        >
                          {duration(result.latencyMs)}
                          {result.latencyMs == null ? (
                            <span className="sr-only">Not recorded</span>
                          ) : null}
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
                        label="View health result details"
                        title="View details"
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
        <Pagination
          mode="cursor"
          page={Math.floor(token / size) + 1}
          range={`${first}–${last}${hasNext ? "+" : ""}`}
          canPrevious={!!token}
          canNext={hasNext}
          disabled={loading}
          pageSizeControl={
            <label className="flex items-center gap-2 whitespace-nowrap text-xs text-[var(--text-muted)]">
              Rows per page
              <Select
                aria-label="Rows per page"
                value={String(size)}
                onChange={(e) => update("pageSize", e.target.value)}
                className="w-[76px]"
              >
                <option value="25">25</option>
                <option value="50">50</option>
                <option value="100">100</option>
              </Select>
            </label>
          }
          onPrevious={() => {
            const next = new URLSearchParams(search);
            const previous = Math.max(0, token - size);
            if (previous) next.set("pageToken", String(previous));
            else next.delete("pageToken");
            go(next);
          }}
          onNext={() => {
            const next = new URLSearchParams(search);
            next.set("pageToken", data.nextPageToken);
            go(next);
          }}
        />
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

function TargetSummary({ result }: { result: HealthResultRecord }) {
  const endpointTarget = !!result.endpointId && !!result.endpoint;
  const protocol = protocolName(result.protocol);
  const port = result.port > 0 ? result.port : null;
  return (
    <div className="health-target">
      <div className="health-target-scope">
        {result.service} · {result.environment}
      </div>
      <div className="health-target-instance">{result.instance}</div>
      <div className="health-target-detail">
        {endpointTarget
          ? `${result.endpoint}${protocol ? ` · ${protocol}` : ""}${port ? ` :${port}` : ""}`
          : result.address || "Instance-level check"}
        {!endpointTarget && result.address ? " · Instance target" : null}
        {!endpointTarget && !result.address ? "Instance-level check" : null}
      </div>
    </div>
  );
}

function TableSkeleton() {
  return (
    <div
      className="health-results-skeleton"
      aria-label="Loading health results"
    >
      <div className="health-results-skeleton-head">
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-4 w-32" />
      </div>
      {[1, 2, 3, 4, 5].map((row) => (
        <div className="health-results-skeleton-row" key={row}>
          <Skeleton className="h-4 w-28" />
          <Skeleton className="h-4 w-40" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-4 w-16" />
        </div>
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
                <dd>{result.endpoint || "Instance-level check"}</dd>
                <dt>Address</dt>
                <dd>{result.address}</dd>
                <dt>Protocol / port</dt>
                <dd>
                  {result.endpointId && result.endpoint
                    ? `${protocolName(result.protocol)}${result.port > 0 ? ` :${result.port}` : ""}`
                    : "Instance target"}
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
