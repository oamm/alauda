import { useState } from "react";
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import axe from "axe-core";
import { EnvironmentForm } from "./EnvironmentForm";
import { EnvironmentsWorkspace } from "./EnvironmentsWorkspace";
import { HealthWorkspace } from "../views/HealthWorkspace";
import { environmentTier } from "../lib/environment-form";
import { healthOverviewRows } from "../lib/health-overview";
import {
  listEnvironments,
  listEnvironmentTopology,
  type Environment,
  type Service,
  type ServiceDeployment,
  type ServiceInstance,
  type HealthCheck,
  type HealthStateView,
} from "../api";

const environments = [
  {
    id: "env1",
    name: "Development",
    key: "dev",
    tier: "0",
    description: "Local testing",
    enabled: true,
    tags: { owner: "operations" },
  },
  {
    id: "env2",
    name: "Staging",
    key: "stg",
    tier: "",
    description: "",
    enabled: false,
    tags: {},
  },
] as Environment[];
const deployments = [
  { id: "d1", serviceId: "s1", environmentId: "env1" },
  { id: "d2", serviceId: "s2", environmentId: "env2" },
] as ServiceDeployment[];
const instances = [
  {
    id: "i1",
    name: "node-one",
    address: "localhost",
    port: 8080,
    deploymentId: "d1",
    enabled: true,
  },
  {
    id: "i2",
    name: "node-two",
    address: "host-two",
    port: 80,
    deploymentId: "d2",
    enabled: true,
  },
  {
    id: "i3",
    name: "node-three",
    address: "host-three",
    port: 80,
    deploymentId: "d1",
    enabled: true,
  },
  {
    id: "i4",
    name: "node-four",
    address: "host-four",
    port: 80,
    deploymentId: "d1",
    enabled: true,
  },
] as ServiceInstance[];
const checks = [
  {
    id: "c1",
    name: "readiness",
    instanceId: "i1",
    type: "HEALTH_CHECK_TYPE_HTTP",
    enabled: true,
    intervalSeconds: 10,
    timeoutSeconds: 5,
  },
  {
    id: "c2",
    name: "liveness",
    instanceId: "i2",
    type: "HEALTH_CHECK_TYPE_HTTP",
    enabled: true,
    intervalSeconds: 10,
    timeoutSeconds: 5,
  },
  {
    id: "c4",
    name: "disabled",
    instanceId: "i4",
    type: "HEALTH_CHECK_TYPE_HTTP",
    enabled: false,
    intervalSeconds: 10,
    timeoutSeconds: 5,
  },
] as HealthCheck[];
const states = [
  { instanceId: "i1", currentState: "HEALTH_STATE_HEALTHY" },
  { instanceId: "i2", currentState: "HEALTH_STATE_UNHEALTHY" },
] as HealthStateView[];
const response = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
const saved = vi.fn(async () => {});
const viewServices = vi.fn();
beforeEach(() => {
  vi.clearAllMocks();
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = input.toString();
      const body = JSON.parse(String(init?.body || "{}"));
      if (path.includes("ListDeployments")) return response({ deployments });
      if (path.includes("ListInstances")) return response({ instances });
      return response({ environment: { ...environments[0], ...body } });
    }),
  );
});
afterEach(() => vi.unstubAllGlobals());
function environmentList(rows = environments) {
  return render(
    <EnvironmentsWorkspace
      environments={rows}
      loading={false}
      selectedEnvironmentId="env1"
      onSaved={saved}
      onViewServices={viewServices}
    />,
  );
}
function menu(label: string) {
  const trigger = screen.getAllByRole("button", { name: label })[0];
  fireEvent.keyDown(trigger, { key: "Enter" });
  return trigger;
}

describe("Environments management", () => {
  it("disables submission while saving and prevents duplicate mutations", async () => {
    let complete!: (value: Response) => void;
    vi.stubGlobal(
      "fetch",
      vi.fn(
        () =>
          new Promise<Response>((resolve) => {
            complete = resolve;
          }),
      ),
    );
    render(
      <EnvironmentForm environments={[]} onSaved={saved} onCancel={() => {}} />,
    );
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "Production" },
    });
    fireEvent.change(screen.getByLabelText("Key"), {
      target: { value: "prod" },
    });
    const button = screen.getByRole("button", { name: "Create environment" });
    fireEvent.click(button);
    expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(fetch).toHaveBeenCalledTimes(1);
    complete(response({ environment: environments[0] }));
    await waitFor(() => expect(saved).toHaveBeenCalled());
    expect(button).toBeDisabled();
  });
  it("maps concurrent server key collisions to a field error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        response(
          {
            code: "internal",
            message: "UNIQUE constraint failed: environments.key",
          },
          500,
        ),
      ),
    );
    render(
      <EnvironmentForm environments={[]} onSaved={saved} onCancel={() => {}} />,
    );
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "Production" },
    });
    fireEvent.change(screen.getByLabelText("Key"), {
      target: { value: "prod" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create environment" }));
    expect(
      await screen.findByText("An environment with this key already exists."),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Key")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    expect(saved).not.toHaveBeenCalled();
  });
  it("returns keyboard focus to the menu trigger after closing an edit dialog", async () => {
    environmentList();
    const trigger = menu("Actions for Development");
    fireEvent.click(
      await screen.findByRole("menuitem", { name: "Edit environment" }),
    );
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByLabelText("Name")).toHaveFocus();
    fireEvent.keyDown(dialog, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    await waitFor(() => expect(trigger).toHaveFocus());
  });
  it("shows a list, meaningful tiers, status and counts across environments, not an inline form", async () => {
    environmentList();
    expect(
      screen.queryByRole("textbox", { name: "Name" }),
    ).not.toBeInTheDocument();
    const table = screen.getByRole("table", { name: "Environments" });
    expect(within(table).getByText("Tier 0")).toBeInTheDocument();
    expect(within(table).getByText("Untiered")).toBeInTheDocument();
    expect(within(table).getByText("Disabled")).toBeInTheDocument();
    await waitFor(() =>
      expect(within(table).queryByText("Loading")).not.toBeInTheDocument(),
    );
    const row = within(table).getByText("Staging").closest("tr")!;
    expect(within(row).getAllByText("1")).toHaveLength(2);
    for (const [, init] of vi.mocked(fetch).mock.calls)
      expect(JSON.parse(String(init?.body))).not.toHaveProperty(
        "environmentId",
      );
  });
  it("opens a labeled create dialog, validates required fields, and creates with actual API defaults", async () => {
    environmentList();
    fireEvent.click(screen.getByRole("button", { name: "Create environment" }));
    const dialog = screen.getByRole("dialog");
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Create environment" }),
    );
    expect(within(dialog).getByLabelText("Name")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    fireEvent.change(within(dialog).getByLabelText("Name"), {
      target: { value: "Production" },
    });
    fireEvent.change(within(dialog).getByLabelText("Key"), {
      target: { value: "prod" },
    });
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Create environment" }),
    );
    await waitFor(() =>
      expect(saved).toHaveBeenCalledWith(
        expect.objectContaining({ key: "prod" }),
        true,
      ),
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(screen.getByText("Environment created.")).toBeInTheDocument();
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) =>
        url.toString().includes("CreateEnvironment"),
      )!;
    expect(JSON.parse(String(call[1]?.body))).not.toHaveProperty("enabled");
  });
  it("rejects duplicate keys locally without mutation", async () => {
    render(
      <EnvironmentForm
        environments={environments}
        onSaved={saved}
        onCancel={() => {}}
      />,
    );
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "Duplicate" },
    });
    fireEvent.change(screen.getByLabelText("Key"), {
      target: { value: "dev" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create environment" }));
    expect(
      screen.getByText("An environment with this key already exists."),
    ).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalled();
  });
  it("edits through the same dialog, keeps key immutable and preserves tags", async () => {
    environmentList();
    menu("Actions for Development");
    fireEvent.click(
      await screen.findByRole("menuitem", { name: "Edit environment" }),
    );
    expect(await screen.findByLabelText("Key")).toHaveAttribute("readonly");
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "Development updated" },
    });
    fireEvent.click(screen.getByRole("switch", { name: "Enabled" }));
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    await waitFor(() =>
      expect(saved).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "Development updated",
          enabled: false,
        }),
        false,
      ),
    );
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) =>
        url.toString().includes("UpdateEnvironment"),
      )!;
    expect(JSON.parse(String(call[1]?.body))).toEqual(
      expect.objectContaining({
        tags: { owner: "operations" },
        enabled: false,
      }),
    );
    expect(JSON.parse(String(call[1]?.body))).not.toHaveProperty("key");
  });
  it("opens menus with a keyboard and navigates to canonical service scope", async () => {
    environmentList();
    menu("Actions for Staging");
    fireEvent.click(
      await screen.findByRole("menuitem", { name: "View services" }),
    );
    await waitFor(() => expect(viewServices).toHaveBeenCalledWith("env2"));
  });
  it("filters by search and enabled state, and provides a focused empty state", () => {
    environmentList();
    fireEvent.change(screen.getByLabelText("Status"), {
      target: { value: "disabled" },
    });
    const table = screen.getByRole("table", { name: "Environments" });
    expect(within(table).queryByText("Development")).not.toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Search"), {
      target: { value: "missing" },
    });
    expect(screen.getByText("No environments match")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reset filters" }));
    expect(screen.getByRole("table")).toBeInTheDocument();
  });
  it("uses a single create action when no environments exist", () => {
    environmentList([]);
    expect(screen.getByText("No environments yet")).toBeInTheDocument();
    expect(
      screen.getAllByRole("button", { name: "Create environment" }),
    ).toHaveLength(1);
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });
  it("shows unavailable counts rather than incorrect zeroes when topology fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => response({ message: "Counts failed" }, 500)),
    );
    environmentList();
    expect(
      await screen.findByText("Resource counts unavailable"),
    ).toBeInTheDocument();
    expect(
      within(screen.getByRole("table")).getAllByText("Unavailable"),
    ).toHaveLength(4);
  });
  it("associates labels and validation errors accessibly", async () => {
    const { container } = render(
      <EnvironmentForm environments={[]} onSaved={saved} onCancel={() => {}} />,
    );
    fireEvent.click(screen.getByRole("button", { name: "Create environment" }));
    const report = await axe.run(container, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
    });
    expect(report.violations).toEqual([]);
  });
  it("keeps free-form tier classifications and distinguishes numeric levels", () => {
    expect(environmentTier("Production")).toBe("Production");
    expect(environmentTier("2")).toBe("Tier 2");
    expect(environmentTier(" ")).toBe("Untiered");
  });
  it("loads every environment and topology page with existing server pagination", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const body = JSON.parse(String(init?.body));
        const second = body.pagination.pageToken === "next";
        const field = input.toString().includes("ListEnvironments")
          ? "environments"
          : input.toString().includes("ListDeployments")
            ? "deployments"
            : "instances";
        return response({
          [field]: [{ id: second ? "second" : "first" }],
          pagination: { nextPageToken: second ? "" : "next" },
        });
      }),
    );
    expect(await listEnvironments()).toHaveLength(2);
    const topology = await listEnvironmentTopology();
    expect(topology.instances).toHaveLength(2);
    expect(topology.deployments).toHaveLength(2);
    expect(vi.mocked(fetch).mock.calls).toHaveLength(6);
  });
});

const healthProps = {
  instances,
  healthStates: states,
  healthChecks: checks,
  endpoints: [],
  services: [
    { id: "s1", name: "lynx", displayName: "Lynx" },
    { id: "s2", name: "other", displayName: "Other" },
  ] as Service[],
  environments,
  deployments,
  selectedEnvironmentId: "",
  error: "",
  onRun: vi.fn(async () => {}),
  onDelete: vi.fn(),
  onSaved: saved,
  onResults: vi.fn(),
  onServiceHealth: vi.fn(),
  onInstanceResults: vi.fn(),
};
function Health() {
  const [status, setStatus] = useState("all");
  return (
    <HealthWorkspace
      {...healthProps}
      healthStatusFilter={status}
      setHealthStatusFilter={setStatus}
    />
  );
}
describe("Global operational health", () => {
  it("paginates larger loaded catalogs and resets pagination for filtering", () => {
    const many = Array.from({ length: 26 }, (_, index) => ({
      ...instances[0],
      id: `node-${index}`,
      name: `node-${index}`,
    }));
    render(
      <HealthWorkspace
        {...healthProps}
        instances={many}
        healthStatusFilter="all"
        setHealthStatusFilter={() => {}}
      />,
    );
    expect(within(screen.getByRole("table")).getAllByRole("row")).toHaveLength(
      26,
    );
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(within(screen.getByRole("table")).getAllByRole("row")).toHaveLength(
      2,
    );
    fireEvent.change(screen.getByLabelText("Search"), {
      target: { value: "node-0" },
    });
    expect(
      within(screen.getByRole("table")).getByRole("button", { name: "node-0" }),
    ).toBeInTheDocument();
  });
  it("keeps overview controls, context rows and menus accessible", async () => {
    const { container } = render(<Health />);
    const report = await axe.run(container, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
    });
    expect(report.violations).toEqual([]);
  });
  it("shows five summary metrics and contextual rows without duplicate Health headings", () => {
    render(<Health />);
    expect(screen.getAllByRole("heading", { name: "Health" })).toHaveLength(1);
    const summary = screen.getByRole("group", { name: "Global health" });
    expect(within(summary).getByText("Checks")).toBeInTheDocument();
    const table = screen.getByRole("table");
    expect(within(table).getAllByText("Lynx")).toHaveLength(3);
    expect(within(table).getByText("Staging")).toBeInTheDocument();
    expect(
      within(table).getByText("No health check configured"),
    ).toBeInTheDocument();
    expect(within(table).getByText("Monitoring disabled")).toBeInTheDocument();
  });
  it("combines status, service, environment and search filters", () => {
    render(<Health />);
    fireEvent.change(screen.getByLabelText("Status"), {
      target: { value: "unhealthy" },
    });
    expect(within(screen.getByRole("table")).getAllByRole("row")).toHaveLength(
      2,
    );
    fireEvent.change(screen.getByLabelText("Service"), {
      target: { value: "s2" },
    });
    fireEvent.change(screen.getByLabelText("Environment"), {
      target: { value: "env2" },
    });
    fireEvent.change(screen.getByLabelText("Search"), {
      target: { value: "host-two" },
    });
    expect(
      within(screen.getByRole("table")).getByRole("button", {
        name: "node-two",
      }),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Search"), {
      target: { value: "missing" },
    });
    expect(screen.getByText("No health targets")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Reset filters" }));
    expect(screen.getByRole("table")).toBeInTheDocument();
  });
  it("links instance rows to existing service health and scopes result navigation", async () => {
    render(<Health />);
    fireEvent.click(
      within(screen.getByRole("table")).getByRole("button", {
        name: "node-one",
      }),
    );
    expect(healthProps.onServiceHealth).toHaveBeenCalledWith("s1", "env1");
    menu("Actions for node-one");
    fireEvent.click(
      await screen.findByRole("menuitem", { name: "View results" }),
    );
    await waitFor(() =>
      expect(healthProps.onInstanceResults).toHaveBeenCalledWith(
        "i1",
        "s1",
        "env1",
      ),
    );
  });
  it("runs the one enabled check from the row menu", async () => {
    render(<Health />);
    menu("Actions for node-one");
    fireEvent.click(await screen.findByRole("menuitem", { name: "Run check" }));
    await waitFor(() => expect(healthProps.onRun).toHaveBeenCalledWith("c1"));
  });
  it("uses the canonical scoped form for instance monitoring setup", async () => {
    render(<Health />);
    menu("Actions for node-three");
    fireEvent.click(
      await screen.findByRole("menuitem", { name: "Configure monitoring" }),
    );
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByLabelText("Check name")).toBeInTheDocument();
    expect(
      within(dialog).queryByRole("combobox", { name: "Service" }),
    ).not.toBeInTheDocument();
    expect(
      within(dialog).queryByRole("combobox", { name: "Instance" }),
    ).not.toBeInTheDocument();
  });
  it("retains Checks and canonical Results navigation", () => {
    render(<Health />);
    fireEvent.click(screen.getByRole("button", { name: "Results" }));
    expect(healthProps.onResults).toHaveBeenCalled();
    fireEvent.click(screen.getByRole("tab", { name: "Checks" }));
    expect(
      screen.getByRole("button", { name: "Create health check" }),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Create health check" }),
    );
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });
  it("reports unavailable state without claiming checks never executed", () => {
    const rows = healthOverviewRows({ ...healthProps, healthStates: [] });
    expect(rows[0].reason).toBe("Current state unavailable");
    expect(rows[0].status).toBe("unknown");
  });
  it("shows an empty operational state without an empty table", () => {
    render(
      <HealthWorkspace
        {...healthProps}
        instances={[]}
        healthStatusFilter="all"
        setHealthStatusFilter={() => {}}
      />,
    );
    expect(screen.getByText("No health targets")).toBeInTheDocument();
    expect(screen.queryByRole("table")).not.toBeInTheDocument();
  });
});
