import { useEffect, useState } from "react";
import type {
  HealthCheck,
  HealthStateView,
  ServiceInstance,
  Service,
  ServiceDeployment,
  Environment,
  Endpoint,
} from "../api";
import { healthOverviewRows } from "../lib/health-overview";
import { pluralize } from "../utils/format";
import { HealthCheckForm } from "../components/HealthCheckForm";
import { ServiceHealthView } from "../components/ServiceHealthView";
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
} from "../components/ui";

export function HealthWorkspace(props: {
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
}) {
  const [tab, setTab] = useState("overview");
  const [editor, setEditor] = useState<HealthCheck | "new">();
  const [editorInstance, setEditorInstance] = useState<string>();
  const [service, setService] = useState("");
  const [environment, setEnvironment] = useState("");
  const [search, setSearch] = useState("");
  const [page, setPage] = useState(1);
  const [running, setRunning] = useState("");
  const [runError, setRunError] = useState("");
  useEffect(() => {
    setService("");
    setEnvironment("");
    setPage(1);
  }, [props.selectedEnvironmentId]);
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
  const pageCount = Math.max(1, Math.ceil(targets.length / 25));
  const currentPage = Math.min(page, pageCount);
  const visible = targets.slice((currentPage - 1) * 25, currentPage * 25);
  const checks = allRows.flatMap((row) => row.checks);
  const configure = (instanceId?: string) => {
    setEditorInstance(instanceId);
    setEditor("new");
  };
  async function run(check: HealthCheck) {
    setRunning(check.id);
    setRunError("");
    try {
      await props.onRun(check.id);
    } catch (e) {
      setRunError(e instanceof Error ? e.message : "Check could not run.");
    } finally {
      setRunning("");
    }
  }
  const actions = (row: (typeof allRows)[number]) => (
    <ActionMenu
      label={`Actions for ${row.instance.name}`}
      items={[
        {
          label: "View service health",
          disabled: !row.service,
          onSelect: () => {
            if (row.service)
              props.onServiceHealth(row.service.id, row.environment?.id || "");
          },
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
                  onSelect: () => void run(row.enabledChecks[0]),
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
  const instanceLink = (row: (typeof allRows)[number]) => (
    <Button
      variant="link"
      className="max-w-full whitespace-normal break-all text-left"
      disabled={!row.service}
      onClick={() => {
        if (row.service)
          props.onServiceHealth(row.service.id, row.environment?.id || "");
      }}
    >
      {row.instance.name}
    </Button>
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
      <Tabs value={tab} onValueChange={setTab}>
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
                  {props.environments
                    .filter(
                      (e) =>
                        !props.selectedEnvironmentId ||
                        e.id === props.selectedEnvironmentId,
                    )
                    .map((e) => (
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
            ) : !visible.length ? (
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
                        <TableHead className="w-1/4">
                          Instance / Address
                        </TableHead>
                        <TableHead>Service</TableHead>
                        <TableHead>Environment</TableHead>
                        <TableHead>Monitoring</TableHead>
                        <TableHead>Status</TableHead>
                        <TableHead className="w-16">
                          <span className="sr-only">Actions</span>
                        </TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {visible.map((row) => (
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
                          <TableCell>{actions(row)}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </div>
                <div className="lg:hidden">
                  <ResourceList label="Instance health">
                    {visible.map((row) => (
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
                        action={actions(row)}
                      />
                    ))}
                  </ResourceList>
                </div>
              </>
            )}
            {targets.length > 25 ? (
              <Pagination
                page={currentPage}
                pageCount={pageCount}
                onPageChange={setPage}
              />
            ) : null}
          </Workspace>
        </TabsContent>
        <TabsContent value="checks">
          <Workspace>
            <ServiceHealthView
              checks={checks}
              states={props.healthStates}
              instances={props.instances}
              endpoints={props.endpoints}
              availability={{}}
              error={props.error}
              saving={false}
              onRun={props.onRun}
              onEdit={(check) => {
                setEditorInstance(undefined);
                setEditor(check);
              }}
              onDelete={props.onDelete}
              onConfigure={() => configure()}
              onInstances={() => setTab("overview")}
              onResults={props.onResults}
              showAvailability={false}
            />
          </Workspace>
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
