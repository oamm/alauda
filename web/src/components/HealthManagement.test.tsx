import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import axe from "axe-core";
import { HealthResultsPage } from "./HealthResultsPage";
import { HealthCheckForm } from "./HealthCheckForm";
import type {
  HealthCheck,
  HealthResultRecord,
  ServiceInstance,
  ServiceDeployment,
  Service,
  Environment,
  Endpoint,
} from "../api";
import {
  healthCheckDefaults,
  healthCheckSchema,
} from "../lib/health-check-form";

const instance = {
  id: "i1",
  name: "node-01",
  deploymentId: "d1",
  address: "localhost",
  port: 8080,
  enabled: true,
} as ServiceInstance;
const endpoint = {
  id: "e1",
  name: "default",
  instanceId: "i1",
  protocol: "PROTOCOL_HTTP",
  port: 8080,
  path: "/",
  enabled: true,
  primary: true,
} as Endpoint;
const check = {
  id: "c1",
  ...healthCheckDefaults({ instanceId: "i1", endpointId: "e1" }),
  metadata: { path: "/healthz", expectedStatus: "200-299" },
} as HealthCheck;
const context = {
  instances: [instance],
  endpoints: [endpoint],
  checks: [check],
  deployments: [
    { id: "d1", serviceId: "s1", environmentId: "env1" } as ServiceDeployment,
  ],
  services: [{ id: "s1", name: "lynx", displayName: "Lynx" } as Service],
  environments: [{ id: "env1", name: "Production" } as Environment],
};
const result: HealthResultRecord = {
  id: "r1",
  timestamp: "2026-10-07T15:04:00Z",
  serviceId: "s1",
  service: "Lynx",
  environmentId: "env1",
  environment: "Production",
  instanceId: "i1",
  instance: "node-01",
  endpointId: "e1",
  endpoint: "default",
  protocol: 1,
  port: 8080,
  address: "localhost",
  checkId: "c1",
  check: "readiness",
  type: "HEALTH_CHECK_TYPE_HTTP",
  path: "/healthz",
  expectedStatus: "200-299",
  success: false,
  latencyMs: null,
  statusCode: null,
  errorType: "invalid_target",
  errorMessage: "Port must be greater than 0",
};
const response = (value: unknown, status = 200) =>
  new Response(JSON.stringify(value), {
    status,
    headers: { "Content-Type": "application/json" },
  });

beforeEach(() => {
  window.history.replaceState({}, "", "/health/results");
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) =>
      input.toString().includes("/api/v1/health/results")
        ? response({
            results: [result],
            nextPageToken: new URL(
              input.toString(),
              window.location.origin,
            ).searchParams.has("pageToken")
              ? ""
              : "25",
          })
        : response({
            healthCheck: {
              ...check,
              ...JSON.parse(String(init?.body || "{}")),
            },
          }),
    ),
  );
});
afterEach(() => {
  vi.unstubAllGlobals();
  window.history.replaceState({}, "", "/dashboard");
});

describe("Health result history", () => {
  it("offers accessible progressive disclosure of advanced mobile filters", async () => {
    render(<HealthResultsPage {...context} onBack={vi.fn()} />);
    await screen.findByRole("table", { name: "Health result history" });
    const toggle = screen.getByRole("button", { name: "More filters" });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    fireEvent.click(toggle);
    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(
      document.getElementById(toggle.getAttribute("aria-controls")!),
    ).toHaveAttribute("data-expanded", "true");
  });
  it("queries every filter on the server and keeps dates/shareable state in the URL", async () => {
    render(<HealthResultsPage {...context} onBack={vi.fn()} />);
    await screen.findByRole("table", { name: "Health result history" });
    for (const [label, value] of [
      ["Service", "s1"],
      ["Environment", "env1"],
      ["Instance", "i1"],
      ["Endpoint", "e1"],
      ["Health check", "c1"],
      ["Check type", "HEALTH_CHECK_TYPE_HTTP"],
      ["Result", "unhealthy"],
      ["Order", "oldest"],
    ])
      fireEvent.change(screen.getByLabelText(label), { target: { value } });
    fireEvent.change(screen.getByLabelText("Port"), {
      target: { value: "8080" },
    });
    fireEvent.change(screen.getByLabelText("From"), {
      target: { value: "2026-10-01T10:00" },
    });
    fireEvent.change(screen.getByLabelText("To"), {
      target: { value: "2026-10-08T10:00" },
    });
    await waitFor(() => {
      const url = new URL(
        vi
          .mocked(fetch)
          .mock.calls[vi.mocked(fetch).mock.calls.length - 1][0].toString(),
        window.location.origin,
      );
      expect(Object.fromEntries(url.searchParams)).toMatchObject({
        serviceId: "s1",
        environmentId: "env1",
        instanceId: "i1",
        endpointId: "e1",
        checkId: "c1",
        port: "8080",
        status: "unhealthy",
        type: "HEALTH_CHECK_TYPE_HTTP",
        sort: "oldest",
        from: new Date("2026-10-01T10:00").toISOString(),
        to: new Date("2026-10-08T10:00").toISOString(),
      });
    });
    expect(new URLSearchParams(window.location.search).get("from")).toMatch(
      /Z$/,
    );
    fireEvent.click(screen.getByRole("button", { name: "Clear all" }));
    expect(window.location.search).toBe("");
    await screen.findByRole("table", { name: "Health result history" });
  });
  it("restores filters on entry and browser navigation, uses bounded server pages", async () => {
    window.history.replaceState(
      {},
      "",
      "/health/results?serviceId=s1&port=8080&sort=oldest&pageSize=25",
    );
    render(<HealthResultsPage {...context} onBack={vi.fn()} />);
    await screen.findByRole("table", { name: "Health result history" });
    expect(screen.getByLabelText("Port")).toHaveValue(8080);
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    await waitFor(() => expect(screen.getByText("Page 2")).toBeInTheDocument());
    expect(new URLSearchParams(window.location.search).get("pageToken")).toBe(
      "25",
    );
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "Next" })).toBeDisabled(),
    );
    fireEvent.change(screen.getByLabelText("Rows per page"), {
      target: { value: "50" },
    });
    expect(window.location.search).not.toContain("pageToken");
    window.history.replaceState({}, "", "/health/results?status=healthy");
    fireEvent(window, new PopStateEvent("popstate"));
    expect(screen.getByLabelText("Result")).toHaveValue("healthy");
  });
  it("provides structured result details and an accessible compact filter/table surface", async () => {
    const { container } = render(
      <HealthResultsPage {...context} onBack={vi.fn()} />,
    );
    const table = await screen.findByRole("table", {
      name: "Health result history",
    });
    expect(table).toHaveTextContent("Production");
    expect(table).toHaveTextContent("Not recorded");
    fireEvent.click(
      within(table).getByRole("button", {
        name: "View health result details",
      }),
    );
    const drawer = screen.getByRole("dialog", { name: "readiness" });
    expect(drawer).toHaveTextContent("Reason");
    expect(drawer).toHaveTextContent("Port must be greater than 0");
    expect(
      (
        await axe.run(drawer, {
          runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
        })
      ).violations,
    ).toEqual([]);
    fireEvent.keyDown(drawer, { key: "Escape" });
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument(),
    );
    expect(container.querySelectorAll("h1")).toHaveLength(1);
  });
  it("handles no matches, invalid dates and API failure without an empty-page crash", async () => {
    vi.mocked(fetch).mockImplementation(async () =>
      response({ results: [], nextPageToken: "" }),
    );
    render(<HealthResultsPage {...context} onBack={vi.fn()} />);
    await screen.findByText(/No health checks have produced results yet/);
    fireEvent.change(screen.getByLabelText("Port"), {
      target: { value: "9999" },
    });
    await screen.findByText("Try changing the selected filters or date range.");
    window.history.replaceState({}, "", "/health/results?from=bad");
    fireEvent(window, new PopStateEvent("popstate"));
    await screen.findByText("Enter valid dates.");
    vi.mocked(fetch).mockResolvedValue(response({}));
    window.history.replaceState({}, "", "/health/results");
    fireEvent(window, new PopStateEvent("popstate"));
    await screen.findByText(
      "Invalid health results response. Refresh or try again later.",
    );
  });
});

describe("Canonical health-check editor", () => {
  it("edits mutable settings on an existing unsupported-executor type without rewriting its configuration", async () => {
    const onSaved = vi.fn();
    render(
      <HealthCheckForm
        {...context}
        check={{ ...check, type: "HEALTH_CHECK_TYPE_HEARTBEAT" }}
        onSaved={onSaved}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.getByLabelText("Type")).toHaveValue(
      "HEALTH_CHECK_TYPE_HEARTBEAT",
    );
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    expect(
      JSON.parse(String(vi.mocked(fetch).mock.calls[0][1]?.body)),
    ).not.toHaveProperty("type");
  });
  it("keeps the shared form labeled and accessible in a scoped surface", async () => {
    const { container } = render(
      <HealthCheckForm
        {...context}
        serviceId="s1"
        environmentId="env1"
        instanceId="i1"
        onSaved={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.queryByLabelText("Service")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Environment")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Instance")).not.toBeInTheDocument();
    expect(
      (
        await axe.run(container, {
          runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
        })
      ).violations,
    ).toEqual([]);
  });
  it.each([false, true])(
    "uses the same defaults and create contract in global/scoped context (%s)",
    async (scoped) => {
      const onSaved = vi.fn();
      render(
        <HealthCheckForm
          {...context}
          serviceId={scoped ? "s1" : undefined}
          environmentId={scoped ? "env1" : undefined}
          initialInstanceId="i1"
          onSaved={onSaved}
          onCancel={vi.fn()}
        />,
      );
      expect(screen.getByLabelText("Check name")).toHaveValue("readiness");
      expect(screen.getByLabelText("Interval (seconds)")).toHaveValue(10);
      expect(screen.getByRole("switch", { name: "Enabled" })).toBeChecked();
      expect(screen.queryByLabelText("Service") !== null).toBe(!scoped);
      fireEvent.change(screen.getByLabelText("Endpoint"), {
        target: { value: "e1" },
      });
      fireEvent.click(
        screen.getByRole("button", { name: "Create health check" }),
      );
      await waitFor(() => expect(onSaved).toHaveBeenCalled());
      expect(
        JSON.parse(String(vi.mocked(fetch).mock.calls[0][1]?.body)),
      ).toMatchObject({
        instanceId: "i1",
        endpointId: "e1",
        name: "readiness",
        enabled: true,
        intervalSeconds: 10,
        timeoutSeconds: 3,
        metadata: { path: "/healthz", expectedStatus: "200-299" },
      });
    },
  );
  it("locks known instance context and keeps update-only fields editable", async () => {
    const onSaved = vi.fn();
    render(
      <HealthCheckForm
        {...context}
        check={check}
        instanceId="i1"
        onSaved={onSaved}
        onCancel={vi.fn()}
      />,
    );
    expect(screen.queryByLabelText("Instance")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Check name")).toHaveAttribute("readonly");
    expect(screen.getByLabelText("Endpoint")).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Interval (seconds)"), {
      target: { value: "30" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save changes" }));
    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    const payload = JSON.parse(String(vi.mocked(fetch).mock.calls[0][1]?.body));
    expect(payload).toMatchObject({ id: "c1", intervalSeconds: 30 });
    expect(payload).not.toHaveProperty("instanceId");
    expect(payload).not.toHaveProperty("metadata");
  });
  it("validates names/targets/statuses and shows backend failure with health-only retry", async () => {
    vi.mocked(fetch).mockImplementation(async () =>
      response({ code: "already_exists" }, 409),
    );
    render(
      <HealthCheckForm
        {...context}
        initialInstanceId="i1"
        onSaved={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    fireEvent.change(screen.getByLabelText("Check name"), {
      target: { value: "" },
    });
    fireEvent.submit(
      screen
        .getByRole("button", { name: "Create health check" })
        .closest("form")!,
    );
    expect(screen.getByText("Check name is required.")).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText("Check name"), {
      target: { value: "ready" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Create health check" }),
    );
    await screen.findByText(
      "A check with this name already exists on this instance.",
    );
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() => expect(fetch).toHaveBeenCalledTimes(2));
    expect(
      healthCheckSchema.safeParse(
        healthCheckDefaults({ instanceId: "i1", expectedStatus: "599-200" }),
      ).success,
    ).toBe(false);
  });
  it("does not recreate a check when the post-save refresh fails", async () => {
    render(
      <HealthCheckForm
        {...context}
        initialInstanceId="i1"
        onSaved={async () => {
          throw new Error("refresh");
        }}
        onCancel={vi.fn()}
      />,
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Create health check" }),
    );
    await screen.findByText(
      /Health check saved, but the view could not refresh/,
    );
    expect(
      screen.queryByRole("button", { name: "Retry" }),
    ).not.toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Create health check" }),
    );
    expect(fetch).toHaveBeenCalledTimes(1);
  });
  it("disables target selection while saving", async () => {
    let resolve!: (value: Response) => void;
    vi.mocked(fetch).mockImplementation(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    render(
      <HealthCheckForm
        {...context}
        initialInstanceId="i1"
        onSaved={vi.fn()}
        onCancel={vi.fn()}
      />,
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Create health check" }),
    );
    expect(screen.getByLabelText("Instance")).toBeDisabled();
    expect(screen.getByLabelText("Service")).toBeDisabled();
    await act(async () => resolve(response({ healthCheck: check })));
    expect(
      screen.getByRole("button", { name: "Create health check" }),
    ).not.toHaveAttribute("aria-busy", "true");
  });
});
