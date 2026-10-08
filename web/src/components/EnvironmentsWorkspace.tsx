import { useEffect, useState } from "react";
import {
  listEnvironmentTopology,
  type Environment,
  type ServiceDeployment,
  type ServiceInstance,
} from "../api";
import { environmentTier } from "../lib/environment-form";
import { EnvironmentForm } from "./EnvironmentForm";
import {
  Alert,
  ActionMenu,
  Button,
  Dialog,
  DialogBody,
  DialogContent,
  DialogHeader,
  DialogTitle,
  EmptyState,
  FilterBar,
  FormField,
  PageHeader,
  ResourceList,
  ResourceRow,
  SearchInput,
  Select,
  Skeleton,
  StatusBadge,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Workspace,
} from "./ui";

export function EnvironmentsWorkspace(props: {
  environments: Environment[];
  loading: boolean;
  error?: string;
  selectedEnvironmentId: string;
  onSaved: (environment: Environment, created: boolean) => Promise<void>;
  onViewServices: (environmentId: string) => void;
}) {
  const [editor, setEditor] = useState<Environment | "new">();
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [notice, setNotice] = useState("");
  const [counts, setCounts] = useState<{
    deployments: ServiceDeployment[];
    instances: ServiceInstance[];
  }>();
  const [countsError, setCountsError] = useState("");
  const [reload, setReload] = useState(0);
  useEffect(() => {
    let active = true;
    setCounts(undefined);
    setCountsError("");
    listEnvironmentTopology()
      .then((data) => {
        if (active) setCounts(data);
      })
      .catch((e) => {
        if (active)
          setCountsError(
            e instanceof Error ? e.message : "Resource counts could not load.",
          );
      });
    return () => {
      active = false;
    };
  }, [reload]);
  const rows = props.environments.filter(
    (e) =>
      (status === "all" || (status === "enabled") === e.enabled) &&
      [e.name, e.key, e.tier, e.description]
        .join(" ")
        .toLowerCase()
        .includes(search.toLowerCase().trim()),
  );
  const resourceCounts = (environment: Environment) => {
    if (!counts)
      return {
        services: countsError ? "Unavailable" : "Loading",
        instances: countsError ? "Unavailable" : "Loading",
      };
    const deployments = counts.deployments.filter(
      (d) => d.environmentId === environment.id,
    );
    const ids = new Set(deployments.map((d) => d.id));
    return {
      services: new Set(deployments.map((d) => d.serviceId)).size,
      instances: counts.instances.filter((i) => ids.has(i.deploymentId)).length,
    };
  };
  const actions = (environment: Environment) => (
    <ActionMenu
      label={`Actions for ${environment.name}`}
      items={[
        { label: "Edit environment", onSelect: () => setEditor(environment) },
        {
          label: "View services",
          onSelect: () => props.onViewServices(environment.id),
        },
      ]}
    />
  );
  return (
    <div className="grid min-w-0 gap-3">
      <PageHeader
        title="Environments"
        description="Manage the scopes used to organize services and operational data."
        action={
          props.environments.length ? (
            <Button variant="primary" onClick={() => setEditor("new")}>
              Create environment
            </Button>
          ) : undefined
        }
      />
      {props.error ? (
        <Alert tone="danger" title="Environments unavailable">
          {props.error}
        </Alert>
      ) : null}
      {notice ? (
        <p role="status" className="text-sm text-[var(--text-muted)]">
          {notice}
        </p>
      ) : null}
      <Workspace>
        {props.environments.length ? (
          <FilterBar compact>
            <FormField label="Search">
              <SearchInput
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search environments"
              />
            </FormField>
            <FormField label="Status">
              <Select
                value={status}
                onChange={(e) => setStatus(e.target.value)}
              >
                <option value="all">All statuses</option>
                <option value="enabled">Enabled</option>
                <option value="disabled">Disabled</option>
              </Select>
            </FormField>
          </FilterBar>
        ) : null}
        {countsError ? (
          <Alert tone="warning" title="Resource counts unavailable">
            {countsError}
            <Button
              size="sm"
              variant="ghost"
              onClick={() => setReload((r) => r + 1)}
            >
              Retry counts
            </Button>
          </Alert>
        ) : null}
        {props.loading ? (
          <Skeleton className="m-3 h-20" />
        ) : !rows.length ? (
          <EmptyState
            title={
              props.environments.length
                ? "No environments match"
                : "No environments yet"
            }
            description={
              props.environments.length
                ? "Try changing the search or status filter."
                : "Create an environment to organize services and operational data."
            }
            action={
              !props.environments.length ? (
                <Button variant="primary" onClick={() => setEditor("new")}>
                  Create environment
                </Button>
              ) : (
                <Button
                  variant="ghost"
                  onClick={() => {
                    setSearch("");
                    setStatus("all");
                  }}
                >
                  Reset filters
                </Button>
              )
            }
          />
        ) : (
          <>
            <div className="hidden md:block">
              <Table aria-label="Environments">
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-1/3">Name</TableHead>
                    <TableHead>Tier</TableHead>
                    <TableHead>Services</TableHead>
                    <TableHead>Instances</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead className="w-16">
                      <span className="sr-only">Actions</span>
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((environment) => {
                    const count = resourceCounts(environment);
                    return (
                      <TableRow
                        key={environment.id}
                        className={
                          environment.id === props.selectedEnvironmentId
                            ? "bg-[var(--surface-muted)]"
                            : ""
                        }
                      >
                        <TableCell>
                          <strong className="break-all font-medium">
                            {environment.name}
                          </strong>
                          <div className="break-all text-xs text-[var(--text-muted)]">
                            {environment.key}
                            {environment.description
                              ? ` · ${environment.description}`
                              : ""}
                          </div>
                        </TableCell>
                        <TableCell>
                          {environmentTier(environment.tier)}
                        </TableCell>
                        <TableCell>{count.services}</TableCell>
                        <TableCell>{count.instances}</TableCell>
                        <TableCell>
                          <StatusBadge
                            status={
                              environment.enabled ? "enabled" : "disabled"
                            }
                          />
                        </TableCell>
                        <TableCell>{actions(environment)}</TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </div>
            <div className="md:hidden">
              <ResourceList label="Environments">
                {rows.map((environment) => {
                  const count = resourceCounts(environment);
                  return (
                    <ResourceRow
                      key={environment.id}
                      title={
                        <strong className="break-all font-medium">
                          {environment.name}
                        </strong>
                      }
                      description={
                        <>
                          <span>
                            {environment.key} ·{" "}
                            {environmentTier(environment.tier)}
                          </span>
                          <span>
                            {count.services} services · {count.instances}{" "}
                            instances
                          </span>
                          {environment.description ? (
                            <span>{environment.description}</span>
                          ) : null}
                        </>
                      }
                      status={
                        <StatusBadge
                          status={environment.enabled ? "enabled" : "disabled"}
                        />
                      }
                      action={actions(environment)}
                    />
                  );
                })}
              </ResourceList>
            </div>
          </>
        )}
      </Workspace>
      <Dialog
        open={!!editor}
        onOpenChange={(open) => {
          if (!open) setEditor(undefined);
        }}
      >
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>
              {editor === "new" ? "Create environment" : "Edit environment"}
            </DialogTitle>
          </DialogHeader>
          <DialogBody>
            {editor ? (
              <EnvironmentForm
                key={editor === "new" ? "new" : editor.id}
                environment={editor === "new" ? undefined : editor}
                environments={props.environments}
                onCancel={() => setEditor(undefined)}
                onSaved={async (environment) => {
                  const created = editor === "new";
                  setEditor(undefined);
                  setNotice(
                    created ? "Environment created." : "Environment updated.",
                  );
                  try {
                    await props.onSaved(environment, created);
                    setReload((r) => r + 1);
                  } catch {
                    setNotice(
                      "Environment saved, but the list could not refresh. Refresh the page.",
                    );
                  }
                }}
              />
            ) : null}
          </DialogBody>
        </DialogContent>
      </Dialog>
    </div>
  );
}
