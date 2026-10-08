import { useEffect, useMemo, useState } from "react";
import type {
  HealthCheck,
  HealthCheckRecord,
  HealthStateView,
  ServiceInstance,
  Service,
  ServiceDeployment,
  Environment,
  Endpoint,
} from "../api";
import { listHealthResults, queryHealthChecks } from "../api";
import { healthOverviewRows } from "../lib/health-overview";
import { formatTimestamp, pluralize } from "../utils/format";
import { HealthCheckForm } from "../components/HealthCheckForm";
import {
  ActionMenu,
  Alert,
  Button,
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogBody,
  Select,
  SearchInput,
  FormField,
  FilterBar,
  PageHeader,
  StatGroup,
  Stat,
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
  ResourceList,
  ResourceRow,
  StatusBadge,
  EmptyState,
  Pagination,
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
  Workspace,
  Skeleton,
  Drawer,
  DrawerContent,
  DefinitionList,
  Section,
  Inline,
} from "../components/ui";

type Props = {
  instances: ServiceInstance[];
  healthStates: HealthStateView[];
  healthChecks: HealthCheck[];
  endpoints: Endpoint[];
  services: Service[];
  environments: Environment[];
  deployments: ServiceDeployment[];
  selectedEnvironmentId: string;
  error: string;
  loading?: boolean;
  healthStatusFilter: string;
  setHealthStatusFilter: (value: string) => void;
  onRun: (id: string) => Promise<void>;
  onDelete: (check: HealthCheck) => void;
  onSaved: () => Promise<void>;
  onResults: (checkId?: string) => void;
  onServiceHealth: (serviceId: string, environmentId: string) => void;
  onInstanceResults: (
    instanceId: string,
    serviceId: string,
    environmentId: string,
  ) => void;
};
function typeName(value: string) {
  return value.replace("HEALTH_CHECK_TYPE_", "");
}
function resultLabel(value: string) {
  return value === "healthy"
    ? "Healthy"
    : value === "unhealthy"
      ? "Unhealthy"
      : "Unknown";
}
function targetLabel(item: HealthCheckRecord) {
  return `${item.service} · ${item.environment} · ${item.instance} · ${item.endpoint} :${item.port}`;
}

export function HealthWorkspace(props: Props) {
  const [tab, setTab] = useState(() =>
    new URLSearchParams(window.location.search).has("serviceId")
      ? "checks"
      : "overview",
  );
  const [editor, setEditor] = useState<HealthCheck | "new">();
  const [editorInstance, setEditorInstance] = useState<string>();
  const [service, setService] = useState("");
  const [environment, setEnvironment] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(25);
  const [resultFilter, setResultFilter] = useState("");
  const [enabledFilter, setEnabledFilter] = useState("");
  const [catalog, setCatalog] = useState<{
    items: HealthCheckRecord[];
    total: number;
    page: number;
    pageSize: number;
  }>({ items: [], total: 0, page: 1, pageSize: 25 });
  const [catalogLoading, setCatalogLoading] = useState(false);
  const [catalogError, setCatalogError] = useState("");
  const [selected, setSelected] = useState<HealthCheckRecord>();
  const [running, setRunning] = useState("");
  const [runError, setRunError] = useState("");
  const scopedService = useMemo(
    () => new URLSearchParams(window.location.search).get("serviceId") || "",
    [],
  );
  useEffect(() => {
    setService(scopedService);
    setPage(1);
  }, [scopedService, props.selectedEnvironmentId]);
  useEffect(() => {
    if (tab !== "checks") return;
    const controller = new AbortController();
    const params = new URLSearchParams();
    if (search.trim()) params.set("search", search.trim());
    if (service) params.set("serviceId", service);
    if (environment) params.set("environmentId", environment);
    if (enabledFilter) params.set("enabled", enabledFilter);
    if (resultFilter) params.set("latestStatus", resultFilter);
    params.set("page", String(page));
    params.set("pageSize", String(pageSize));
    setCatalogLoading(true);
    setCatalogError("");
    queryHealthChecks(params, controller.signal)
      .then((value) => {
        if (!controller.signal.aborted)
          setCatalog({ ...value, items: value.items ?? [] });
      })
      .catch((error) => {
        if (!controller.signal.aborted)
          setCatalogError(
            error instanceof Error
              ? error.message
              : "Health checks unavailable",
          );
      })
      .finally(() => {
        if (!controller.signal.aborted) setCatalogLoading(false);
      });
    return () => controller.abort();
  }, [
    tab,
    search,
    service,
    environment,
    enabledFilter,
    resultFilter,
    page,
    pageSize,
  ]);
  const allRows = healthOverviewRows(props);
  const scope = allRows.filter(
    (row) =>
      (!service || row.service?.id === service) &&
      (!environment || row.environment?.id === environment) &&
      [
        row.instance.name,
        row.instance.address,
        row.service?.displayName,
        row.service?.name,
        row.environment?.name,
      ]
        .join(" ")
        .toLowerCase()
        .includes(search.trim().toLowerCase()),
  );
  const targets = scope.filter(
    (row) =>
      props.healthStatusFilter === "all" ||
      row.status === props.healthStatusFilter,
  );
  const overviewPageCount = Math.max(1, Math.ceil(targets.length / 25));
  const visibleOverview = targets.slice((page - 1) * 25, page * 25);
  const configure = (instanceId?: string) => {
    setEditorInstance(instanceId);
    setEditor("new");
  };
  async function run(id: string) {
    setRunning(id);
    setRunError("");
    try {
      await props.onRun(id);
      setSelected(undefined);
    } catch (error) {
      setRunError(
        error instanceof Error ? error.message : "Check could not run.",
      );
    } finally {
      setRunning("");
    }
  }
  function openEdit(item: HealthCheckRecord) {
    const check = props.healthChecks.find((value) => value.id === item.id);
    if (check) {
      setSelected(undefined);
      setEditor(check);
    }
  }
  function openChecks() {
    setTab("checks");
    const next = new URLSearchParams(window.location.search);
    if (service) next.set("serviceId", service);
    else next.delete("serviceId");
    window.history.pushState(
      {},
      "",
      `/health/checks${next.size ? `?${next}` : ""}`,
    );
  }
  const instanceLink = (row: (typeof allRows)[number]) => (
    <Button
      variant="link"
      className="max-w-full whitespace-normal break-all text-left"
      disabled={!row.service}
      onClick={() =>
        row.service &&
        props.onServiceHealth(row.service.id, row.environment?.id || "")
      }
    >
      {row.instance.name}
    </Button>
  );
  const overviewActions = (row: (typeof allRows)[number]) => (
    <ActionMenu
      label={`Actions for ${row.instance.name}`}
      items={[
        {
          label: "View service health",
          disabled: !row.service,
          onSelect: () =>
            row.service &&
            props.onServiceHealth(row.service.id, row.environment?.id || ""),
        },
        {
          label: "View results",
          onSelect: () =>
            props.onInstanceResults(
              row.instance.id,
              row.service?.id || "",
              row.environment?.id || "",
            ),
        },
        ...(row.checks.length
          ? row.enabledChecks.length === 1
            ? [
                {
                  label: "Run check",
                  disabled: !!running || !row.instance.enabled,
                  onSelect: () => void run(row.enabledChecks[0].id),
                },
              ]
            : []
          : [
              {
                label: "Configure monitoring",
                onSelect: () => configure(row.instance.id),
              },
            ]),
      ]}
    />
  );
  return (
    <div className="grid min-w-0 gap-3">
      <PageHeader
        title="Health"
        description="Operational health and monitoring across services."
        action={
          tab === "overview" ? (
            <Button variant="primary" onClick={() => configure()}>
              Create health check
            </Button>
          ) : undefined
        }
      />
      <Tabs
        value={tab}
        onValueChange={(value) => {
          setTab(value);
          if (value === "checks") openChecks();
        }}
      >
        <div className="flex flex-wrap items-center gap-1">
          <TabsList aria-label="Health views">
            <TabsTrigger value="overview">Overview</TabsTrigger>
            <TabsTrigger value="checks">Checks</TabsTrigger>
          </TabsList>
          <Button variant="ghost" onClick={() => props.onResults()}>
            Results
          </Button>
        </div>
        <TabsContent value="overview">
          {props.error || runError ? (
            <Alert tone="danger" title="Health data unavailable">
              {props.error || runError}
            </Alert>
          ) : null}
          <Workspace>
            <div className="p-3">
              <StatGroup label="Global health" columns={5}>
                <Stat label="Instances" value={scope.length} />
                <Stat
                  label="Healthy"
                  value={scope.filter((row) => row.status === "healthy").length}
                />
                <Stat
                  label="Unhealthy"
                  value={
                    scope.filter((row) => row.status === "unhealthy").length
                  }
                />
                <Stat
                  label="Unknown"
                  value={scope.filter((row) => row.status === "unknown").length}
                />
                <Stat
                  label="Checks"
                  value={scope.reduce(
                    (total, row) => total + row.checks.length,
                    0,
                  )}
                />
              </StatGroup>
            </div>
            <FilterBar compact>
              <FormField label="Status">
                <Select
                  value={props.healthStatusFilter}
                  onChange={(e) => {
                    props.setHealthStatusFilter(e.target.value);
                    setPage(1);
                  }}
                >
                  {["all", "healthy", "unhealthy", "unknown"].map((s) => (
                    <option key={s} value={s}>
                      {s === "all"
                        ? "All statuses"
                        : s.charAt(0).toUpperCase() + s.slice(1)}
                    </option>
                  ))}
                </Select>
              </FormField>
              <FormField label="Service">
                <Select
                  value={service}
                  onChange={(e) => {
                    setService(e.target.value);
                    setPage(1);
                  }}
                >
                  <option value="">All services</option>
                  {props.services.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.displayName || s.name}
                    </option>
                  ))}
                </Select>
              </FormField>
              <FormField label="Environment">
                <Select
                  value={environment}
                  onChange={(e) => {
                    setEnvironment(e.target.value);
                    setPage(1);
                  }}
                >
                  <option value="">All environments in scope</option>
                  {props.environments.map((e) => (
                    <option key={e.id} value={e.id}>
                      {e.name}
                    </option>
                  ))}
                </Select>
              </FormField>
              <FormField label="Search">
                <SearchInput
                  value={search}
                  placeholder="Search instances"
                  onChange={(e) => {
                    setSearch(e.target.value);
                    setPage(1);
                  }}
                />
              </FormField>
            </FilterBar>
            {props.loading ? (
              <Skeleton className="m-3 h-20" />
            ) : !visibleOverview.length ? (
              <EmptyState
                title="No health targets"
                description={
                  allRows.length
                    ? "No instances match the selected filters."
                    : "No registered instances are available in this scope."
                }
                action={
                  allRows.length ? (
                    <Button
                      variant="ghost"
                      onClick={() => {
                        setService("");
                        setEnvironment("");
                        setSearch("");
                        props.setHealthStatusFilter("all");
                        setPage(1);
                      }}
                    >
                      Reset filters
                    </Button>
                  ) : undefined
                }
              />
            ) : (
              <>
                <div className="hidden lg:block">
                  <Table aria-label="Instance health">
                    <TableHeader>
                      <TableRow>
                        <TableHead>Instance / Address</TableHead>
                        <TableHead>Service</TableHead>
                        <TableHead>Environment</TableHead>
                        <TableHead>Monitoring</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead>
                          <span className="sr-only">Actions</span>
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {visibleOverview.map((row) => (
                        <TableRow key={row.instance.id}>
                          <TableCell>
                            {instanceLink(row)}
                            <div className="break-all text-xs text-[var(--text-muted)]">
                              {row.instance.address}
                            </div>
                          </TableCell>
                          <TableCell>
                            {row.service?.displayName ||
                              row.service?.name ||
                              "Service unavailable"}
                          </TableCell>
                          <TableCell>
                            {row.environment?.name || "Environment unavailable"}
                          </TableCell>
                          <TableCell>
                            <span>
                              {pluralize(row.checks.length, "health check")}
                            </span>
                            <div className="text-xs text-[var(--text-muted)]">
                              {row.reason}
                            </div>
                          </TableCell>
                          <TableCell>
                            <StatusBadge status={row.status} />
                          </TableCell>
                          <TableCell>{overviewActions(row)}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
                <div className="lg:hidden">
                  <ResourceList label="Instance health">
                    {visibleOverview.map((row) => (
                      <ResourceRow
                        key={row.instance.id}
                        title={instanceLink(row)}
                        description={
                          <>
                            <span>
                              {row.service?.displayName ||
                                row.service?.name ||
                                "Service unavailable"}{" "}
                              ·{" "}
                              {row.environment?.name ||
                                "Environment unavailable"}
                            </span>
                            <span className="break-all">
                              {row.instance.address}
                            </span>
                            <span>
                              {pluralize(row.checks.length, "health check")} ·{" "}
                              {row.reason}
                            </span>
                          </>
                        }
                        status={<StatusBadge status={row.status} />}
                        action={overviewActions(row)}
                      />
                    ))}
                  </ResourceList>
                </div>
              </>
            )}
            {targets.length > 25 ? (
              <Pagination
                page={Math.min(page, overviewPageCount)}
                pageCount={overviewPageCount}
                onPageChange={setPage}
              />
            ) : null}
          </Workspace>
        </TabsContent>
        <TabsContent value="checks">
          <CheckCatalog
            items={catalog.items}
            total={catalog.total}
            page={page}
            pageSize={pageSize}
            loading={catalogLoading}
            error={catalogError}
            search={search}
            service={service}
            environment={environment}
            enabled={enabledFilter}
            result={resultFilter}
            services={props.services}
            environments={props.environments}
            selected={selected}
            running={running}
            onSearch={(value) => {
              setSearch(value);
              setPage(1);
            }}
            onService={(value) => {
              setService(value);
              setPage(1);
            }}
            onEnvironment={(value) => {
              setEnvironment(value);
              setPage(1);
            }}
            onEnabled={(value) => {
              setEnabledFilter(value);
              setPage(1);
            }}
            onResult={(value) => {
              setResultFilter(value);
              setPage(1);
            }}
            onPage={setPage}
            onPageSize={(value) => {
              setPageSize(value);
              setPage(1);
            }}
            onSelect={setSelected}
            onRun={run}
            onEdit={openEdit}
            onDelete={props.onDelete}
            onCreate={() => configure()}
            onResults={props.onResults}
          />
        </TabsContent>
      </Tabs>
      <Dialog
        open={!!editor}
        onOpenChange={(open) => {
          if (!open) setEditor(undefined);
        }}
      >
        <DialogContent size="lg">
          <DialogHeader>
            <DialogTitle>
              {editor === "new" ? "Create health check" : "Edit health check"}
            </DialogTitle>
          </DialogHeader>
          <DialogBody>
            {editor ? (
              <HealthCheckForm
                key={
                  editor === "new"
                    ? `new-${editorInstance || "global"}`
                    : editor.id
                }
                instances={props.instances}
                endpoints={props.endpoints}
                services={props.services}
                environments={props.environments}
                deployments={props.deployments}
                serviceId={
                  editorInstance
                    ? allRows.find((row) => row.instance.id === editorInstance)
                        ?.service?.id
                    : service || undefined
                }
                instanceId={editorInstance}
                environmentId={
                  editorInstance
                    ? allRows.find((row) => row.instance.id === editorInstance)
                        ?.environment?.id
                    : environment || props.selectedEnvironmentId
                }
                check={editor === "new" ? undefined : editor}
                onSaved={async () => {
                  setEditor(undefined);
                  await props.onSaved();
                }}
                onCancel={() => setEditor(undefined)}
              />
            ) : null}
          </DialogBody>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function CheckCatalog(props: {
  items: HealthCheckRecord[];
  total: number;
  page: number;
  pageSize: number;
  loading: boolean;
  error: string;
  search: string;
  service: string;
  environment: string;
  enabled: string;
  result: string;
  services: Service[];
  environments: Environment[];
  selected?: HealthCheckRecord;
  running: string;
  onSearch: (value: string) => void;
  onService: (value: string) => void;
  onEnvironment: (value: string) => void;
  onEnabled: (value: string) => void;
  onResult: (value: string) => void;
  onPage: (value: number) => void;
  onPageSize: (value: number) => void;
  onSelect: (value?: HealthCheckRecord) => void;
  onRun: (id: string) => Promise<void>;
  onEdit: (item: HealthCheckRecord) => void;
  onDelete: (check: HealthCheck) => void;
  onCreate: () => void;
  onResults: (id?: string) => void;
}) {
  const pageCount = Math.max(1, Math.ceil(props.total / props.pageSize));
  const first = props.total ? (props.page - 1) * props.pageSize + 1 : 0;
  const last = Math.min(props.total, props.page * props.pageSize);
  const itemActions = (item: HealthCheckRecord) => (
    <ActionMenu
      label={`Actions for ${item.name}`}
      items={[
        { label: "View details", onSelect: () => props.onSelect(item) },
        {
          label: "Run check",
          disabled: !!props.running,
          onSelect: () => void props.onRun(item.id),
        },
        { label: "Edit", onSelect: () => props.onEdit(item) },
        {
          label: item.enabled ? "Disable" : "Enable",
          onSelect: () => props.onEdit(item),
        },
        { label: "Delete", onSelect: () => props.onDelete(item) },
      ]}
    />
  );
  return (
    <Workspace>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-[var(--border)] px-4 py-3">
        <div>
          <h2 className="m-0 text-sm font-semibold">
            Health checks{" "}
            <span className="font-normal text-[var(--text-muted)]">
              {props.total} checks
            </span>
          </h2>
          {props.service ? (
            <p className="m-0 mt-1 text-xs text-[var(--text-muted)]">
              Filtered by:{" "}
              {props.services.find((s) => s.id === props.service)
                ?.displayName || "Service"}
            </p>
          ) : null}
        </div>
        <Button variant="primary" size="sm" onClick={props.onCreate}>
          Create health check
        </Button>
      </div>
      <FilterBar
        collapseAfter={4}
        activeAdvanced={
          [props.environment, props.result].filter(Boolean).length
        }
      >
        <FormField label="Search">
          <SearchInput
            value={props.search}
            placeholder="Search checks, targets, address"
            onChange={(e) => props.onSearch(e.target.value)}
          />
        </FormField>
        <FormField label="Status">
          <Select
            value={props.enabled}
            onChange={(e) => props.onEnabled(e.target.value)}
          >
            <option value="">All statuses</option>
            <option value="true">Enabled</option>
            <option value="false">Disabled</option>
          </Select>
        </FormField>
        <FormField label="Result">
          <Select
            value={props.result}
            onChange={(e) => props.onResult(e.target.value)}
          >
            <option value="">All results</option>
            <option value="healthy">Healthy</option>
            <option value="unhealthy">Unhealthy</option>
            <option value="unknown">Unknown</option>
          </Select>
        </FormField>
        <FormField label="Service">
          <Select
            value={props.service}
            onChange={(e) => props.onService(e.target.value)}
          >
            <option value="">All services</option>
            {props.services.map((s) => (
              <option key={s.id} value={s.id}>
                {s.displayName || s.name}
              </option>
            ))}
          </Select>
        </FormField>
        <FormField label="Environment">
          <Select
            value={props.environment}
            onChange={(e) => props.onEnvironment(e.target.value)}
          >
            <option value="">All environments</option>
            {props.environments.map((e) => (
              <option key={e.id} value={e.id}>
                {e.name}
              </option>
            ))}
          </Select>
        </FormField>
      </FilterBar>
      {props.error ? (
        <Alert tone="danger" title="Health checks unavailable">
          {props.error}
        </Alert>
      ) : null}
      {props.loading ? (
        <div className="grid gap-px px-4 py-2">
          {[1, 2, 3, 4, 5].map((row) => (
            <Skeleton key={row} className="h-12" />
          ))}
        </div>
      ) : !props.items.length ? (
        <EmptyState
          title={
            props.total
              ? "No health checks found"
              : "No health checks configured"
          }
          description={
            props.total
              ? "Try changing the selected filters."
              : "Create a Health Check to monitor a Service Instance or Endpoint."
          }
          action={
            !props.total ? (
              <Button onClick={props.onCreate}>Create health check</Button>
            ) : undefined
          }
        />
      ) : (
        <>
          <div className="hidden min-w-0 lg:block">
            <Table aria-label="Health check catalog">
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Target</TableHead>
                  <TableHead>Interval</TableHead>
                  <TableHead>Last result</TableHead>
                  <TableHead>Last run</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>
                    <span className="sr-only">Actions</span>
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {props.items.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <Button
                        variant="link"
                        className="text-left"
                        onClick={() => props.onSelect(item)}
                      >
                        {item.name}
                      </Button>
                      <div className="text-xs text-[var(--text-muted)]">
                        {typeName(item.type)}
                      </div>
                    </TableCell>
                    <TableCell>{targetLabel(item)}</TableCell>
                    <TableCell>{item.intervalSeconds}s</TableCell>
                    <TableCell>
                      <StatusBadge
                        status={item.latestStatus}
                        label={resultLabel(item.latestStatus)}
                      />
                    </TableCell>
                    <TableCell>
                      {item.latestResultAt
                        ? formatTimestamp(item.latestResultAt)
                        : "Not run"}
                    </TableCell>
                    <TableCell>
                      <StatusBadge
                        status={item.enabled ? "enabled" : "disabled"}
                        label={item.enabled ? "Enabled" : "Disabled"}
                      />
                    </TableCell>
                    <TableCell>{itemActions(item)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          <div className="lg:hidden">
            <ResourceList label="Health check catalog">
              {props.items.map((item) => (
                <ResourceRow
                  key={item.id}
                  title={
                    <Button variant="link" onClick={() => props.onSelect(item)}>
                      {item.name}
                    </Button>
                  }
                  description={
                    <>
                      <span>{targetLabel(item)}</span>
                      <span>
                        {item.intervalSeconds}s ·{" "}
                        {item.latestResultAt
                          ? formatTimestamp(item.latestResultAt)
                          : "Not run"}
                      </span>
                      <span>Last result: {resultLabel(item.latestStatus)}</span>
                    </>
                  }
                  status={
                    <StatusBadge
                      status={item.enabled ? "enabled" : "disabled"}
                      label={item.enabled ? "Enabled" : "Disabled"}
                    />
                  }
                  action={itemActions(item)}
                />
              ))}
            </ResourceList>
          </div>
        </>
      )}
      <div className="flex flex-wrap items-center justify-between gap-3 border-t border-[var(--border)] px-4 py-2 text-xs text-[var(--text-muted)]">
        <span>
          {first}–{last} of {props.total}
        </span>
        <label className="flex items-center gap-2">
          Rows
          <Select
            value={String(props.pageSize)}
            onChange={(e) => props.onPageSize(Number(e.target.value))}
          >
            <option value="25">25</option>
            <option value="50">50</option>
            <option value="100">100</option>
          </Select>
        </label>
        <Pagination
          page={props.page}
          pageCount={pageCount}
          onPageChange={props.onPage}
        />
      </div>
      <CheckDetail
        item={props.selected}
        running={props.running}
        onClose={() => props.onSelect(undefined)}
        onRun={props.onRun}
        onEdit={props.onEdit}
        onDelete={props.onDelete}
        onResults={props.onResults}
      />
    </Workspace>
  );
}

function CheckDetail(props: {
  item?: HealthCheckRecord;
  running: string;
  onClose: () => void;
  onRun: (id: string) => Promise<void>;
  onEdit: (item: HealthCheckRecord) => void;
  onDelete: (check: HealthCheck) => void;
  onResults: (id?: string) => void;
}) {
  const item = props.item;
  const [recent, setRecent] = useState<
    Awaited<ReturnType<typeof listHealthResults>>
  >([]);
  const [recentLoading, setRecentLoading] = useState(false);
  useEffect(() => {
    if (!item) {
      setRecent([]);
      return;
    }
    let active = true;
    setRecentLoading(true);
    listHealthResults(item.id)
      .then((value) => {
        if (active) setRecent(value.slice(0, 5));
      })
      .catch(() => {
        if (active) setRecent([]);
      })
      .finally(() => {
        if (active) setRecentLoading(false);
      });
    return () => {
      active = false;
    };
  }, [item?.id]);
  if (!item) return null;
  return (
    <Drawer open onOpenChange={(open) => !open && props.onClose()}>
      <DrawerContent size="lg">
        <DialogHeader>
          <div className="flex items-start justify-between gap-3">
            <div>
              <DialogTitle>{item.name}</DialogTitle>
              <p className="mt-1 text-xs text-[var(--text-muted)]">
                {typeName(item.type)} · every {item.intervalSeconds}s
              </p>
              <p className="mt-1 text-xs text-[var(--text-muted)]">
                {targetLabel(item)}
              </p>
            </div>
            <StatusBadge
              status={item.enabled ? "enabled" : "disabled"}
              label={item.enabled ? "Enabled" : "Disabled"}
            />
          </div>
        </DialogHeader>
        <DialogBody>
          <div className="grid gap-4">
            <Section title="Latest result" divider={false}>
              <Inline>
                <StatusBadge
                  status={item.latestStatus}
                  label={resultLabel(item.latestStatus)}
                />
                {item.latestResultAt ? (
                  <span className="text-xs text-[var(--text-muted)]">
                    {formatTimestamp(item.latestResultAt)}
                  </span>
                ) : (
                  <span className="text-xs text-[var(--text-muted)]">
                    Not run
                  </span>
                )}
              </Inline>
              {item.latestError ? (
                <p className="mt-2 text-sm text-[var(--danger)]">
                  {item.latestError}
                </p>
              ) : null}
            </Section>
            <Section title="Target">
              <DefinitionList>
                <dt>Service</dt>
                <dd>{item.service}</dd>
                <dt>Environment</dt>
                <dd>{item.environment}</dd>
                <dt>Instance</dt>
                <dd>{item.instance}</dd>
                <dt>Address</dt>
                <dd>{item.address}</dd>
                <dt>Endpoint</dt>
                <dd>{item.endpoint}</dd>
                <dt>Protocol / port</dt>
                <dd>
                  {item.protocol || "Instance address"} :{item.port}
                </dd>
                <dt>Path</dt>
                <dd>{item.path || "Not specified"}</dd>
              </DefinitionList>
            </Section>
            <Section title="Configuration">
              <DefinitionList>
                <dt>Type</dt>
                <dd>{typeName(item.type)}</dd>
                <dt>Interval</dt>
                <dd>{item.intervalSeconds}s</dd>
                <dt>Timeout</dt>
                <dd>{item.timeoutSeconds}s</dd>
                <dt>Expected status</dt>
                <dd>{item.expectedStatus || "Not specified"}</dd>
              </DefinitionList>
            </Section>
            <Section
              title="Recent executions"
              action={
                <Button
                  size="sm"
                  variant="link"
                  onClick={() => props.onResults(item.id)}
                >
                  View all results
                </Button>
              }
            >
              {recentLoading ? (
                <Skeleton className="h-10" />
              ) : recent.length ? (
                <ResourceList label="Recent executions">
                  {recent.map((result) => (
                    <ResourceRow
                      key={result.id}
                      title={formatTimestamp(result.timestamp)}
                      description={
                        result.errorMessage ||
                        result.errorType ||
                        (result.latencyMs == null
                          ? "Duration not recorded"
                          : `${result.latencyMs} ms`)
                      }
                      status={
                        <StatusBadge
                          status={result.success ? "healthy" : "unhealthy"}
                        />
                      }
                    />
                  ))}
                </ResourceList>
              ) : (
                <p className="text-xs text-[var(--text-muted)]">
                  No executions recorded.
                </p>
              )}
            </Section>
          </div>
        </DialogBody>
        <div className="flex flex-wrap justify-end gap-2 border-t border-[var(--border)] p-4">
          <Button
            disabled={!!props.running}
            onClick={() => void props.onRun(item.id)}
          >
            Run check
          </Button>
          <Button variant="secondary" onClick={() => props.onEdit(item)}>
            Edit
          </Button>
          <Button variant="ghost" onClick={() => props.onEdit(item)}>
            {item.enabled ? "Disable" : "Enable"}
          </Button>
          <Button variant="ghost" onClick={() => props.onDelete(item)}>
            Delete
          </Button>
        </div>
      </DrawerContent>
    </Drawer>
  );
}
