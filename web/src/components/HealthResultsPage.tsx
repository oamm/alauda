import { useEffect, useState } from "react";
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
  Select,
  Button,
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

export function HealthResultsPage(props: {
  services: Service[];
  environments: Environment[];
  instances: ServiceInstance[];
  endpoints: Endpoint[];
  checks: HealthCheck[];
  deployments: ServiceDeployment[];
  onBack: () => void;
}) {
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
  function go(next: URLSearchParams) {
    window.history.pushState(
      {},
      "",
      `/health/results${next.size ? `?${next}` : ""}`,
    );
    setSearch(window.location.search);
  }
  useEffect(() => {
    const controller = new AbortController();
    setLoading(true);
    setError("");
    setData({ results: [], nextPageToken: "" });
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
  const instances = props.instances.filter((i) => {
    const d = props.deployments.find((d) => d.id === i.deploymentId);
    return (
      (!get("serviceId") || d?.serviceId === get("serviceId")) &&
      (!get("environmentId") || d?.environmentId === get("environmentId"))
    );
  });
  const endpoints = props.endpoints.filter(
    (e) =>
      instances.some((i) => i.id === e.instanceId) &&
      (!get("instanceId") || e.instanceId === get("instanceId")),
  );
  const checks = props.checks.filter(
    (c) =>
      instances.some((i) => i.id === c.instanceId) &&
      (!get("instanceId") || c.instanceId === get("instanceId")) &&
      (!get("endpointId") || c.endpointId === get("endpointId")),
  );
  function select(
    key: string,
    label: string,
    items: { id: string; name: string }[],
  ) {
    return (
      <FormField label={label}>
        <Select value={get(key)} onChange={(e) => update(key, e.target.value)}>
          <option value="">All</option>
          {items.map((i) => (
            <option key={i.id} value={i.id}>
              {i.name}
            </option>
          ))}
        </Select>
      </FormField>
    );
  }
  const columns = [
    "Timestamp",
    "Service / Environment",
    "Instance / Address",
    "Endpoint / Port",
    "Type",
    "Check",
    "Result",
    "Duration",
    "Details",
  ];
  const token = Number(get("pageToken") || 0);
  const size = Number(get("pageSize") || 25);
  return (
    <div className="grid min-w-0 gap-4">
      <PageHeader
        title="Health results"
        action={
          <Button variant="ghost" onClick={props.onBack}>
            Back to Health
          </Button>
        }
      />
      <FilterBar
        collapseAfter={4}
        activeAdvanced={
          [
            "instanceId",
            "endpointId",
            "checkId",
            "port",
            "type",
            "status",
            "sort",
          ].filter((key) => get(key)).length
        }
      >
        {(
          [
            ["from", "From"],
            ["to", "To"],
          ] as const
        ).map(([key, label]) => (
          <FormField key={key} label={label}>
            <Input
              type="datetime-local"
              value={localDateTime(get(key))}
              onChange={(e) => update(key, e.target.value)}
            />
          </FormField>
        ))}
        {select(
          "serviceId",
          "Service",
          props.services.map((s) => ({
            id: s.id,
            name: s.displayName || s.name,
          })),
        )}
        {select("environmentId", "Environment", props.environments)}
        {select("instanceId", "Instance", instances)}
        {select("endpointId", "Endpoint", endpoints)}
        {select("checkId", "Health check", checks)}
        <FormField label="Port">
          <Input
            type="number"
            min={1}
            max={65535}
            step={1}
            value={get("port")}
            onChange={(e) => update("port", e.target.value)}
          />
        </FormField>
        {select(
          "type",
          "Check type",
          allHealthTypes.map((t) => ({
            id: `HEALTH_CHECK_TYPE_${t}`,
            name: t,
          })),
        )}
        {select("status", "Result", [
          { id: "healthy", name: "Healthy" },
          { id: "unhealthy", name: "Unhealthy" },
        ])}
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
            {[25, 50, 100].map((n) => (
              <option key={n} value={n}>
                {n}
              </option>
            ))}
          </Select>
        </FormField>
        <Button variant="ghost" onClick={() => go(new URLSearchParams())}>
          Reset filters
        </Button>
      </FilterBar>
      {loading ? (
        <Skeleton className="h-20" />
      ) : error ? (
        <Alert tone="danger" title="Health results unavailable">
          {error}
          <Button onClick={() => setRetry((v) => v + 1)}>Retry</Button>
        </Alert>
      ) : !data.results.length ? (
        <EmptyState
          title="No health results found"
          description={
            params.size
              ? "Try changing the selected filters or date range."
              : "No health checks have produced results yet. Run a check or wait for scheduled execution."
          }
        />
      ) : (
        <>
          <div className="hidden min-w-0 md:block">
            <Table aria-label="Health result history">
              <TableHeader>
                <TableRow>
                  {columns.map((c) => (
                    <TableHead key={c}>{c}</TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.results.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>{formatTimestamp(r.timestamp)}</TableCell>
                    <TableCell>
                      {r.service}
                      <div className="text-xs text-[var(--text-muted)]">
                        {r.environment}
                      </div>
                    </TableCell>
                    <TableCell>
                      {r.instance}
                      <div className="break-all text-xs text-[var(--text-muted)]">
                        {r.address}
                      </div>
                    </TableCell>
                    <TableCell>
                      {r.endpoint || "Instance address"}
                      <div className="text-xs">
                        {protocolName(r.protocol)} :{r.port}
                      </div>
                    </TableCell>
                    <TableCell>
                      {r.type.replace("HEALTH_CHECK_TYPE_", "")}
                    </TableCell>
                    <TableCell>{r.check}</TableCell>
                    <TableCell>
                      <StatusBadge
                        status={r.success ? "healthy" : "unhealthy"}
                      />
                    </TableCell>
                    <TableCell>
                      {r.latencyMs == null
                        ? "Not recorded"
                        : `${r.latencyMs} ms`}
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="ghost"
                        onClick={() => setSelection(r)}
                        aria-label={`Details for ${r.check} at ${r.timestamp}`}
                      >
                        Details
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          <div className="md:hidden">
            <ResourceList label="Health result history">
              {data.results.map((r) => (
                <ResourceRow
                  key={r.id}
                  title={
                    <Button variant="link" onClick={() => setSelection(r)}>
                      {r.check}
                    </Button>
                  }
                  description={
                    <>
                      <span>{formatTimestamp(r.timestamp)}</span>
                      <span>
                        {r.service} · {r.environment} · {r.instance} ·{" "}
                        {r.endpoint || "Instance address"} :{r.port}
                      </span>
                      <span>
                        {r.latencyMs == null
                          ? "Duration not recorded"
                          : `${r.latencyMs} ms`}
                      </span>
                    </>
                  }
                  status={
                    <StatusBadge status={r.success ? "healthy" : "unhealthy"} />
                  }
                />
              ))}
            </ResourceList>
          </div>
        </>
      )}
      <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
        <span>Page {Math.floor(token / size) + 1}</span>
        <div className="flex gap-2">
          <Button
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
            disabled={loading || !data.nextPageToken}
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
      <Drawer
        open={!!selection}
        onOpenChange={(open) => {
          if (!open) setSelection(undefined);
        }}
      >
        <DrawerContent size="lg">
          <DialogHeader>
            <DialogTitle>{selection?.check || "Result details"}</DialogTitle>
          </DialogHeader>
          <DialogBody>
            {selection ? (
              <DefinitionList>
                <dt>Time</dt>
                <dd>{formatTimestamp(selection.timestamp)}</dd>
                <dt>Result</dt>
                <dd>
                  <StatusBadge
                    status={selection.success ? "healthy" : "unhealthy"}
                  />
                </dd>
                <dt>Reason</dt>
                <dd className="break-words">
                  {selection.errorMessage ||
                    selection.errorType ||
                    "No failure reported"}
                </dd>
                {(
                  [
                    ["Service", selection.service],
                    ["Environment", selection.environment],
                    ["Instance", selection.instance],
                    ["Endpoint", selection.endpoint || "Instance address"],
                    ["Address", `${selection.address}:${selection.port}`],
                    ["Type", selection.type.replace("HEALTH_CHECK_TYPE_", "")],
                    [
                      "Protocol",
                      protocolName(selection.protocol) || "Instance address",
                    ],
                    ["Path", selection.path || "Not specified"],
                    [
                      "Expected status",
                      selection.expectedStatus || "Not specified",
                    ],
                    ["Actual status", selection.statusCode ?? "Not recorded"],
                    [
                      "Duration",
                      selection.latencyMs == null
                        ? "Not recorded"
                        : `${selection.latencyMs} ms`,
                    ],
                  ] as const
                ).map(([label, value]) => (
                  <div className="contents" key={label}>
                    <dt>{label}</dt>
                    <dd className="break-all">{value}</dd>
                  </div>
                ))}
              </DefinitionList>
            ) : null}
          </DialogBody>
          <DialogFooter>
            <Button onClick={() => setSelection(undefined)}>Done</Button>
          </DialogFooter>
        </DrawerContent>
      </Drawer>
    </div>
  );
}
