import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import axe from "axe-core";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import App from "./App";
import { mockResponse } from "./test/fixtures";

class MockEventSource {
  onerror: (() => void) | null = null;

  constructor(public url: string) {}

  addEventListener() {}

  close() {}
}

describe("App", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
        const url = input.toString();
        return new Response(JSON.stringify(mockResponse(url, init)), {
          headers: { "Content-Type": "application/json" },
          status: 200,
        });
      }),
    );
    vi.stubGlobal("EventSource", MockEventSource);
  });

  afterEach(() => {
    window.localStorage.clear();
    window.history.replaceState({}, "", "/dashboard");
    vi.unstubAllGlobals();
  });

  it("renders only the dedicated login boundary without a session", async () => {
    vi.mocked(fetch).mockImplementation(async (input: RequestInfo | URL) => {
      if (input.toString().includes("/api/v1/auth/me")) {
        return new Response(
          JSON.stringify({ error: "invalid authentication" }),
          { status: 401 },
        );
      }
      return new Response(JSON.stringify(mockResponse(input.toString())), {
        headers: { "Content-Type": "application/json" },
        status: 200,
      });
    });

    render(<App />);

    expect(
      await screen.findByRole("heading", { name: "Sign in to Alauda" }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole("navigation", { name: "Primary" }),
    ).not.toBeInTheDocument();
    expect(screen.queryByText("System status")).not.toBeInTheDocument();
  });

  it("renders operational dashboard data", async () => {
    render(<App />);

    await waitFor(() =>
      expect(screen.getByText("Checkout service created")).toBeInTheDocument(),
    );

    expect(screen.getByText("Open incidents")).toBeInTheDocument();
    expect(screen.getByText("Alert policies")).toBeInTheDocument();
    expect(screen.getByText("Checkout service created")).toBeInTheDocument();
  });

  it("toggles dark mode", async () => {
    const { container } = render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Dark mode" }));

    expect(container.firstElementChild).toHaveClass("dark");
    expect(
      screen.getByRole("button", { name: "Light mode" }),
    ).toBeInTheDocument();
  });

  it("navigates incident and event workflows with mocked registry data", async () => {
    render(<App />);

    await screen.findByText("Checkout service created");

    fireEvent.click(screen.getByRole("button", { name: "Incidents" }));
    expect(screen.getByText("health check failed")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("State"), {
      target: { value: "open" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Verify recovery" }));
    expect(
      await screen.findByText(
        "Recovery verified. The Incident has been resolved.",
      ),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Details" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Incident Detail")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Close dialog" }));
    fireEvent.click(screen.getByRole("button", { name: "Events" }));
    expect(screen.getAllByText("service.created").length).toBeGreaterThan(0);
    expect(screen.getByText("Checkout service created")).toBeInTheDocument();
  });

  it("shows security users and API tokens", async () => {
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Security" }));

    await waitFor(() =>
      expect(screen.getAllByText("admin").length).toBeGreaterThan(0),
    );
    expect(screen.queryByText("Sign in to Alauda")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Users" }));
    expect(screen.getByText("automation")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "API tokens" }));
    expect(
      screen.getByRole("button", { name: "Create API token" }),
    ).toBeInTheDocument();
  });

  it("uses the shared tab interaction contract", async () => {
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));

    const overviewTab = screen.getByRole("tab", { name: "Overview" });
    expect(overviewTab).toHaveAttribute("aria-selected", "true");
    fireEvent.keyDown(overviewTab, { key: "ArrowRight" });
    expect(screen.getByRole("tab", { name: "Instances" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("adds an instance through the composite API", async () => {
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));

    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(screen.getByRole("button", { name: "Add instance" }));
    expect(
      screen.getByRole("button", { name: "Close dialog" }),
    ).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Instance name"), {
      target: { value: "checkout-prod-02" },
    });
    fireEvent.change(screen.getByLabelText("Address / hostname"), {
      target: { value: "10.0.0.2" },
    });
    fireEvent.change(screen.getByLabelText("Path"), {
      target: { value: "/" },
    });
    const addInstanceButtons = screen.getAllByRole("button", {
      name: "Add instance",
    });
    fireEvent.click(addInstanceButtons[addInstanceButtons.length - 1]);

    await screen.findByText("Instance added successfully: checkout-prod-02.");

    const registerCall = vi
      .mocked(fetch)
      .mock.calls.find(([input]) =>
        input.toString().includes("RegisterRuntime"),
      );
    expect(registerCall).toBeDefined();
    const body = JSON.parse(registerCall?.[1]?.body?.toString() ?? "{}");
    expect(body.serviceId).toBe("svc-1");
    expect(body.environmentId).toBe("env-1");
    expect(body.deploymentId).toBeUndefined();
    expect(body.instance).toMatchObject({
      name: "checkout-prod-02",
      address: "10.0.0.2",
    });
    expect(body.endpoints[0]).toMatchObject({
      name: "default",
      port: 8080,
      primary: true,
    });
  });

  it("uses the shared endpoint fields for named, disabled registration drafts", async () => {
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(screen.getByRole("button", { name: "Add instance" }));
    const dialog = screen.getByRole("dialog", { name: "Add instance" });
    expect(within(dialog).getByLabelText("Name")).toHaveValue("default");
    expect(
      within(dialog).getByRole("switch", { name: "Enabled" }),
    ).toBeChecked();
    fireEvent.change(within(dialog).getByLabelText("Instance name"), {
      target: { value: "new-instance" },
    });
    fireEvent.change(within(dialog).getByLabelText("Address / hostname"), {
      target: { value: "host.example" },
    });
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Add endpoint" }),
    );
    expect(within(dialog).getByLabelText("Endpoint 2 name")).toHaveValue(
      "default-2",
    );
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Add endpoint" }),
    );
    expect(within(dialog).getByLabelText("Endpoint 3 name")).toHaveValue(
      "default-3",
    );
    expect(
      (
        await axe.run(dialog, {
          runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
        })
      ).violations,
    ).toEqual([]);
    fireEvent.click(
      within(dialog).getAllByRole("button", { name: "Remove endpoint" })[2],
    );
    fireEvent.change(within(dialog).getByLabelText("Endpoint 2 name"), {
      target: { value: "default" },
    });
    expect(within(dialog).getByLabelText("Endpoint 2 name")).toHaveAttribute(
      "aria-invalid",
      "true",
    );
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Add instance" }),
    );
    expect(
      vi
        .mocked(fetch)
        .mock.calls.filter(([url]) =>
          url.toString().includes("RegisterRuntime"),
        ),
    ).toHaveLength(0);
    fireEvent.change(within(dialog).getByLabelText("Endpoint 2 name"), {
      target: { value: "metrics" },
    });
    fireEvent.click(
      within(dialog).getByRole("switch", { name: "Endpoint 2 primary" }),
    );
    expect(
      within(dialog).getByRole("switch", { name: "Endpoint 1 primary" }),
    ).not.toBeChecked();
    fireEvent.click(
      within(dialog).getByRole("switch", { name: "Endpoint 2 enabled" }),
    );
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Add instance" }),
    );
    await screen.findByText("Instance added");
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("RegisterRuntime"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}").endpoints).toEqual([
      {
        name: "default",
        protocol: "PROTOCOL_HTTP",
        port: 8080,
        path: "/",
        enabled: true,
        primary: false,
      },
      {
        name: "metrics",
        protocol: "PROTOCOL_HTTP",
        port: 8080,
        path: "/",
        enabled: false,
        primary: true,
      },
    ]);
  });

  it("allows endpoint-free registration and restores first-draft defaults", async () => {
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(screen.getByRole("button", { name: "Add instance" }));
    const dialog = screen.getByRole("dialog", { name: "Add instance" });
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Remove endpoint" }),
    );
    expect(within(dialog).queryByLabelText("Name")).not.toBeInTheDocument();
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Add endpoint" }),
    );
    expect(within(dialog).getByLabelText("Name")).toHaveValue("default");
    expect(
      within(dialog).getByRole("switch", { name: "Primary" }),
    ).toBeChecked();
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Remove endpoint" }),
    );
    fireEvent.change(within(dialog).getByLabelText("Instance name"), {
      target: { value: "no-endpoints" },
    });
    fireEvent.change(within(dialog).getByLabelText("Address / hostname"), {
      target: { value: "host.example" },
    });
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Add instance" }),
    );
    await screen.findByText("Instance added");
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("RegisterRuntime"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}").endpoints).toEqual(
      [],
    );
  });

  it("scopes instance creation to the selected service workspace", async () => {
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));

    expect(screen.getByRole("tab", { name: "Overview" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.queryByText("Add instance")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Target" }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    expect(screen.getByText("Endpoints")).toBeInTheDocument();
    expect(screen.getAllByText("Health").length).toBeGreaterThan(0);

    expect(
      screen.getByRole("button", { name: "Add instance" }),
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Add instance" }));
    expect(
      screen.getByRole("dialog", { name: "Add instance" }),
    ).toHaveAccessibleDescription(/Checkout/);
    expect(screen.queryByLabelText("Service")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Environment")).toHaveValue("env-1");
    fireEvent.change(screen.getByLabelText("Instance name"), {
      target: { value: "checkout-prod-03" },
    });
    fireEvent.change(screen.getByLabelText("Address / hostname"), {
      target: { value: "10.0.0.3" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "Add instance" }));
    expect(screen.getByLabelText("Instance name")).toHaveValue(
      "checkout-prod-03",
    );
    expect(screen.getByLabelText("Address / hostname")).toHaveValue("10.0.0.3");
  });

  it("configures optional health monitoring after adding an instance", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("CreateHealthCheck")
        ? new Response("Monitoring unavailable", { status: 500 })
        : original(input, init),
    );
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(screen.getByRole("button", { name: "Add instance" }));

    fireEvent.change(screen.getByLabelText("Instance name"), {
      target: { value: "checkout-prod-04" },
    });
    fireEvent.change(screen.getByLabelText("Address / hostname"), {
      target: { value: "10.0.0.4" },
    });
    fireEvent.change(screen.getByLabelText("Path"), {
      target: { value: "/public" },
    });

    expect(screen.queryByText("Health monitoring")).not.toBeInTheDocument();
    const addInstanceButtons = screen.getAllByRole("button", {
      name: "Add instance",
    });
    fireEvent.click(addInstanceButtons[addInstanceButtons.length - 1]);
    await screen.findByRole("heading", { name: "Instance added" });
    expect(
      screen.getByRole("dialog", { name: "Instance added" }),
    ).toHaveAccessibleDescription(/checkout-prod-02.*10.0.0.2/);
    fireEvent.click(screen.getByRole("button", { name: "Configure health" }));
    expect(screen.getByLabelText("Type")).toBeInTheDocument();
    expect(screen.getByText("Interval (seconds)")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(window.location.pathname).toBe("/services/svc-1/instances");
    fireEvent.click(
      screen.getByRole("button", { name: "Create health check" }),
    );
    await screen.findByText(/Health monitoring could not be configured/);
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Retry" }));
    await waitFor(() =>
      expect(
        vi
          .mocked(fetch)
          .mock.calls.filter(([url]) =>
            url.toString().includes("CreateHealthCheck"),
          ),
      ).toHaveLength(2),
    );
    expect(
      vi
        .mocked(fetch)
        .mock.calls.filter(([input]) =>
          input.toString().includes("RegisterRuntime"),
        ),
    ).toHaveLength(1);
  });

  it("keeps instance creation out of unified Health and removes Availability navigation", async () => {
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    expect(
      screen.queryByRole("tab", { name: "Availability" }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("tab", { name: "Health" }));

    expect(
      screen.queryByRole("button", { name: "Add instance" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("list", { name: "Health checks" }),
    ).toHaveTextContent("checkout-a");
  });

  it("discloses instance endpoints and monitoring only in the detail dialog", async () => {
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    expect(
      screen.getByRole("table", { name: "Instances in Production" }),
    ).toHaveTextContent("checkout-a");
    expect(screen.queryByText("/healthz")).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Add endpoint" }),
    ).not.toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    const dialog = screen.getByRole("dialog", { name: "checkout-a" });
    expect(
      within(dialog).getByRole("table", { name: "Instance endpoints" }),
    ).toHaveTextContent("/healthz");
    expect(
      within(dialog).getByRole("cell", { name: "Primary" }),
    ).toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("tab", { name: "Monitoring" }));
    expect(
      within(dialog).getByText(/1 health check configured/),
    ).toBeInTheDocument();
    expect(
      within(dialog).getByRole("button", { name: "View service health" }),
    ).toBeInTheDocument();
    expect(
      within(dialog).queryByRole("button", { name: "Run check" }),
    ).not.toBeInTheDocument();
    fireEvent.click(within(dialog).getByRole("button", { name: "Done" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("creates an endpoint for the selected instance through the focused editor", async () => {
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Add endpoint" }));
    const editor = screen.getByRole("dialog", { name: "Add endpoint" });
    expect(within(editor).getByLabelText("Name")).toHaveValue("default");
    expect(
      within(editor).getByRole("switch", { name: "Primary" }),
    ).not.toBeChecked();
    fireEvent.change(within(editor).getByLabelText("Name"), {
      target: { value: "admin" },
    });
    fireEvent.click(within(editor).getByRole("switch", { name: "Enabled" }));
    fireEvent.change(within(editor).getByLabelText("Port"), {
      target: { value: "9090" },
    });
    fireEvent.click(within(editor).getByRole("button", { name: "Save" }));
    await screen.findByRole("dialog", { name: "checkout-a" });
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("CreateEndpoint"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}")).toMatchObject({
      instanceId: "inst-1",
      name: "admin",
      port: 9090,
      primary: false,
      enabled: false,
    });
    expect(
      vi
        .mocked(fetch)
        .mock.calls.filter(([url]) =>
          url.toString().includes("RegisterRuntime"),
        ),
    ).toHaveLength(0);
  });

  it("edits instance enabled state without exposing forms in the list", async () => {
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Edit instance" }));
    const editor = screen.getByRole("dialog", { name: "checkout-a" });
    fireEvent.click(within(editor).getByRole("switch", { name: "Enabled" }));
    fireEvent.click(
      within(editor).getByRole("button", { name: "Save instance" }),
    );
    await waitFor(() =>
      expect(
        within(editor).queryByRole("button", { name: "Save instance" }),
      ).not.toBeInTheDocument(),
    );
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("UpdateInstance"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}")).toMatchObject({
      id: "inst-1",
      address: "10.0.0.1",
      port: 8080,
      enabled: false,
    });
  });

  it("returns from the endpoint editor with Escape and restores instance-list focus", async () => {
    const user = userEvent.setup();
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    const trigger = screen.getByRole("button", {
      name: "View details for checkout-a",
    });
    await user.click(trigger);
    await user.click(screen.getByRole("button", { name: "Add endpoint" }));
    expect(screen.getByRole("textbox", { name: "Name" })).toHaveFocus();
    await user.keyboard("{Escape}");
    expect(
      screen.getByRole("dialog", { name: "checkout-a" }),
    ).toBeInTheDocument();
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await waitFor(() => expect(trigger).toHaveFocus());
  });

  it("edits primary endpoint state without changing endpoint ownership", async () => {
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Edit endpoint healthz" }),
    );
    const editor = screen.getByRole("dialog", { name: "Edit endpoint" });
    expect(within(editor).getByLabelText("Name")).toHaveValue("healthz");
    fireEvent.change(within(editor).getByLabelText("Name"), {
      target: { value: "readiness" },
    });
    expect(
      within(editor).getByRole("switch", { name: "Primary" }),
    ).toBeChecked();
    fireEvent.click(within(editor).getByRole("switch", { name: "Enabled" }));
    fireEvent.click(within(editor).getByRole("button", { name: "Save" }));
    await screen.findByRole("dialog", { name: "checkout-a" });
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("UpdateEndpoint"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}")).toMatchObject({
      id: "end-1",
      name: "readiness",
      primary: true,
      enabled: false,
      port: 8080,
      path: "/healthz",
    });
  });

  it("suggests an unused endpoint name and rejects duplicates within the instance", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("ListEndpoints")
        ? new Response(
            JSON.stringify({
              endpoints: [
                {
                  id: "end-1",
                  instanceId: "inst-1",
                  name: "default",
                  protocol: "PROTOCOL_HTTP",
                  port: 8080,
                  path: "/",
                  primary: true,
                  enabled: true,
                },
              ],
            }),
            { status: 200 },
          )
        : original(input, init),
    );
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Add endpoint" }));
    const editor = screen.getByRole("dialog", { name: "Add endpoint" });
    const name = within(editor).getByRole("textbox", { name: "Name" });
    expect(name).toHaveValue("default-2");
    fireEvent.change(name, { target: { value: "default" } });
    fireEvent.click(within(editor).getByRole("button", { name: "Save" }));
    expect(name).toHaveAttribute("aria-invalid", "true");
    expect(
      vi
        .mocked(fetch)
        .mock.calls.filter(([url]) =>
          url.toString().includes("CreateEndpoint"),
        ),
    ).toHaveLength(0);
    fireEvent.change(name, { target: { value: "default-2" } });
    fireEvent.click(within(editor).getByRole("button", { name: "Save" }));
    await screen.findByRole("dialog", { name: "checkout-a" });
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("CreateEndpoint"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}")).toMatchObject({
      name: "default-2",
      instanceId: "inst-1",
    });
  });

  it("rejects duplicate endpoint renames while excluding the current endpoint", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("ListEndpoints")
        ? new Response(
            JSON.stringify({
              endpoints: [
                {
                  id: "end-1",
                  instanceId: "inst-1",
                  name: "default",
                  protocol: "PROTOCOL_HTTP",
                  port: 8080,
                  path: "/",
                  enabled: true,
                  primary: true,
                },
                {
                  id: "end-2",
                  instanceId: "inst-1",
                  name: "metrics",
                  protocol: "PROTOCOL_HTTP",
                  port: 8081,
                  path: "/metrics",
                  enabled: true,
                  primary: false,
                },
              ],
            }),
            { status: 200 },
          )
        : original(input, init),
    );
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Edit endpoint default" }),
    );
    const editor = screen.getByRole("dialog", { name: "Edit endpoint" });
    const name = within(editor).getByLabelText("Name");
    expect(name).toHaveValue("default");
    expect(name).not.toHaveAttribute("aria-invalid", "true");
    fireEvent.change(name, { target: { value: "metrics" } });
    fireEvent.click(within(editor).getByRole("button", { name: "Save" }));
    expect(name).toHaveAttribute("aria-invalid", "true");
    expect(
      vi
        .mocked(fetch)
        .mock.calls.filter(([url]) =>
          url.toString().includes("UpdateEndpoint"),
        ),
    ).toHaveLength(0);
    fireEvent.change(name, { target: { value: "admin" } });
    fireEvent.click(within(editor).getByRole("button", { name: "Save" }));
    await screen.findByRole("dialog", { name: "checkout-a" });
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("UpdateEndpoint"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}")).toMatchObject({
      id: "end-1",
      name: "admin",
    });
  });

  it("explains backend endpoint-name conflicts without discarding the form", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("CreateEndpoint")
        ? new Response(
            JSON.stringify({
              code: "already_exists",
              message:
                "failed to create endpoint: UNIQUE constraint failed: endpoints.instance_id, endpoints.name",
            }),
            { status: 409 },
          )
        : original(input, init),
    );
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Add endpoint" }));
    const editor = screen.getByRole("dialog", { name: "Add endpoint" });
    fireEvent.click(within(editor).getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(within(editor).getByRole("alert")).toHaveTextContent(
        "Choose a different name",
      ),
    );
    expect(within(editor).getByRole("textbox", { name: "Name" })).toHaveValue(
      "default",
    );
    expect(editor).not.toHaveTextContent("UNIQUE constraint");
  });

  it("keeps a failed endpoint save in the focused editor for retry", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("CreateEndpoint")
        ? new Response("Endpoint unavailable", { status: 500 })
        : original(input, init),
    );
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Add endpoint" }));
    const editor = screen.getByRole("dialog", { name: "Add endpoint" });
    fireEvent.change(within(editor).getByLabelText("Port"), {
      target: { value: "9090" },
    });
    fireEvent.click(within(editor).getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(within(editor).getByRole("alert")).toHaveTextContent(
        "Endpoint unavailable",
      ),
    );
    expect(within(editor).getByLabelText("Port")).toHaveValue(9090);
    expect(
      within(editor).getByRole("button", { name: "Save" }),
    ).not.toBeDisabled();
  });

  it("offers one empty-endpoint action and hands monitoring configuration to Health", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("ListEndpoints") ||
      input.toString().includes("ListHealthChecks")
        ? new Response("{}", { status: 200 })
        : original(input, init),
    );
    render(<App />);
    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Instances" }));
    fireEvent.click(
      screen.getByRole("button", { name: "View details for checkout-a" }),
    );
    const dialog = screen.getByRole("dialog", { name: "checkout-a" });
    expect(within(dialog).getByText("No endpoints yet")).toBeInTheDocument();
    expect(
      within(dialog).getAllByRole("button", { name: "Add endpoint" }),
    ).toHaveLength(1);
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Add endpoint" }),
    );
    const endpointEditor = screen.getByRole("dialog", { name: "Add endpoint" });
    expect(within(endpointEditor).getByLabelText("Name")).toHaveValue(
      "default",
    );
    expect(
      within(endpointEditor).getByRole("switch", { name: "Primary" }),
    ).toBeChecked();
    expect(
      within(endpointEditor).getByRole("switch", { name: "Enabled" }),
    ).toBeChecked();
    fireEvent.click(
      within(endpointEditor).getByRole("button", { name: "Cancel" }),
    );
    fireEvent.click(within(dialog).getByRole("tab", { name: "Monitoring" }));
    expect(
      within(dialog).getByText("No monitoring configured"),
    ).toBeInTheDocument();
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Configure health" }),
    );
    expect(
      screen.queryByRole("dialog", { name: "checkout-a" }),
    ).not.toBeInTheDocument();
    expect(
      screen.getByRole("dialog", { name: "Configure health" }),
    ).toBeInTheDocument();
    expect(window.location.pathname).toBe("/services/svc-1/health");
  });

  it("maps old Availability URLs to Health and preserves historical window values", async () => {
    window.history.replaceState({}, "", "/services/svc-1/availability");
    render(<App />);
    await screen.findByRole("group", { name: "Current health" });
    await waitFor(() =>
      expect(window.location.pathname).toBe("/services/svc-1/health"),
    );
    expect(screen.getByRole("tab", { name: "Health" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(
      screen.queryByRole("tab", { name: "Availability" }),
    ).not.toBeInTheDocument();
    const metrics = screen.getByRole("group", { name: "Availability windows" });
    await waitFor(() => expect(metrics).toHaveTextContent("99.90%"));
    expect(metrics).toHaveTextContent("99.95%");
    expect(metrics).toHaveTextContent("99.98%");
    expect(
      screen.getByRole("group", { name: "Current health" }),
    ).toHaveTextContent("Monitored1");
  });

  it("loads scoped results and opens check details without exposing technical IDs", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("ListHealthResults")
        ? new Response(
            JSON.stringify({
              results: [
                {
                  id: "result-1",
                  instanceId: "inst-1",
                  healthCheckId: "hc-1",
                  timestamp: "2026-08-22T15:04:00Z",
                  success: true,
                  latencyMs: 124,
                },
                {
                  id: "other-result",
                  instanceId: "other",
                  healthCheckId: "other-check",
                  success: false,
                  errorMessage: "Other scope",
                },
              ],
            }),
            { status: 200 },
          )
        : original(input, init),
    );
    window.history.replaceState({}, "", "/services/svc-1/health");
    render(<App />);
    await screen.findByText(/Healthy · 124 ms/);
    expect(
      screen.getByRole("list", { name: "Health results" }),
    ).toHaveTextContent("124 ms");
    expect(screen.queryByText("Other scope")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "HTTP health" }));
    const drawer = screen.getByRole("dialog", { name: "HTTP health" });
    expect(drawer).toHaveAccessibleDescription(/checkout-a/);
    expect(drawer).toHaveTextContent("Configuration");
    expect(drawer).toHaveTextContent("Recent executions");
    expect(drawer).not.toHaveTextContent("inst-1");
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("ListHealthResults"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}")).toMatchObject({
      healthCheckId: "hc-1",
    });
  });

  it("keeps monitoring/no-results empty states concise and creates monitoring in scope", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("ListHealthChecks") ||
      input.toString().includes("GetAvailability")
        ? new Response("{}", { status: 200 })
        : original(input, init),
    );
    window.history.replaceState({}, "", "/services/svc-1/health");
    render(<App />);
    await screen.findByText("No health checks configured");
    expect(screen.queryByText("No health results yet")).not.toBeInTheDocument();
    expect(
      screen.getByText(
        "Availability data will appear after health results are collected.",
      ),
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", { name: "Configure monitoring" }),
    );
    const dialog = screen.getByRole("dialog", { name: "Configure health" });
    expect(
      within(dialog).getByRole("combobox", { name: "Instance" }),
    ).toHaveValue("inst-1");
    fireEvent.change(within(dialog).getByLabelText("Check name"), {
      target: { value: "readiness" },
    });
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Create health check" }),
    );
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Configure health" }),
      ).not.toBeInTheDocument(),
    );
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("CreateHealthCheck"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}")).toMatchObject({
      instanceId: "inst-1",
      name: "readiness",
    });
  });

  it("runs checks, refreshes results and uses the returned current state", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("RunHealthCheck")
        ? new Response(
            JSON.stringify({
              state: {
                instanceId: "inst-1",
                currentState: "HEALTH_STATE_HEALTHY",
                consecutiveSuccesses: 1,
                consecutiveFailures: 0,
              },
            }),
            { status: 200 },
          )
        : original(input, init),
    );
    window.history.replaceState({}, "", "/services/svc-1/health");
    render(<App />);
    await screen.findByText("No health results yet");
    fireEvent.click(screen.getByRole("button", { name: "Run check" }));
    await waitFor(() =>
      expect(
        screen.getByRole("group", { name: "Current health" }),
      ).toHaveTextContent("Healthy1"),
    );
    expect(
      vi
        .mocked(fetch)
        .mock.calls.filter(([url]) =>
          url.toString().includes("RunHealthCheck"),
        ),
    ).toHaveLength(1);
  });

  it("edits monitoring with the actual update-form values", async () => {
    window.history.replaceState({}, "", "/services/svc-1/health");
    render(<App />);
    await screen.findByRole("button", { name: "HTTP health" });
    fireEvent.click(screen.getByRole("button", { name: "HTTP health" }));
    fireEvent.click(screen.getByRole("button", { name: "Edit check" }));
    const dialog = screen.getByRole("dialog", { name: "Edit health check" });
    fireEvent.change(within(dialog).getByLabelText("Interval (seconds)"), {
      target: { value: "30" },
    });
    fireEvent.click(within(dialog).getByRole("switch", { name: "Enabled" }));
    fireEvent.click(
      within(dialog).getByRole("button", { name: "Save changes" }),
    );
    await waitFor(() =>
      expect(
        screen.queryByRole("dialog", { name: "Edit health check" }),
      ).not.toBeInTheDocument(),
    );
    const call = vi
      .mocked(fetch)
      .mock.calls.find(([url]) => url.toString().includes("UpdateHealthCheck"));
    expect(JSON.parse(call?.[1]?.body?.toString() ?? "{}")).toMatchObject({
      id: "hc-1",
      intervalSeconds: 30,
      enabled: false,
    });
  });

  it("preserves environment-scoped availability requests and excludes out-of-scope checks", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("ListHealthChecks")
        ? new Response(
            JSON.stringify({
              healthChecks: [
                {
                  id: "other-check",
                  instanceId: "other-instance",
                  name: "Other check",
                  enabled: true,
                },
              ],
            }),
            { status: 200 },
          )
        : original(input, init),
    );
    window.history.replaceState({}, "", "/services/svc-1/health");
    render(<App />);
    await screen.findByText("No health checks configured");
    fireEvent.change(
      screen.getByRole("combobox", { name: "Environment selector" }),
      { target: { value: "env-1" } },
    );
    await waitFor(() =>
      expect(
        vi
          .mocked(fetch)
          .mock.calls.some(
            ([url, init]) =>
              url.toString().includes("GetAvailability") &&
              JSON.parse(init?.body?.toString() ?? "{}").environmentId ===
                "env-1" &&
              JSON.parse(init?.body?.toString() ?? "{}").serviceId === "svc-1",
          ),
      ).toBe(true),
    );
    expect(
      screen.queryByRole("button", { name: "Other check" }),
    ).not.toBeInTheDocument();
  });

  it("bounds recent health results and opens the available result history", async () => {
    const original = vi.mocked(fetch).getMockImplementation()!;
    vi.mocked(fetch).mockImplementation(async (input, init) =>
      input.toString().includes("ListHealthResults")
        ? new Response(
            JSON.stringify({
              results: Array.from({ length: 6 }, (_, index) => ({
                id: `result-${index}`,
                instanceId: "inst-1",
                healthCheckId: "hc-1",
                timestamp: `2026-08-22T15:0${index}:00Z`,
                success: true,
                latencyMs: 100 + index,
              })),
            }),
            { status: 200 },
          )
        : original(input, init),
    );
    window.history.replaceState({}, "", "/services/svc-1/health");
    render(<App />);
    await screen.findByRole("button", { name: "View all results" });
    expect(
      within(screen.getByRole("list", { name: "Health results" })).getAllByRole(
        "listitem",
      ),
    ).toHaveLength(5);
    fireEvent.click(screen.getByRole("button", { name: "View all results" }));
    await screen.findByRole("heading", { name: "Health results" });
    expect(window.location.pathname).toBe("/health/results");
    expect(new URLSearchParams(window.location.search).get("serviceId")).toBe(
      "svc-1",
    );
    await screen.findByText("No health results found");
    expect(
      vi
        .mocked(fetch)
        .mock.calls.some(([url]) =>
          url.toString().includes("/api/v1/health/results?serviceId=svc-1"),
        ),
    ).toBe(true);
  });

  it("provides an accessible health-check drawer with Escape focus restoration", async () => {
    const user = userEvent.setup();
    window.history.replaceState({}, "", "/services/svc-1/health");
    render(<App />);
    const trigger = await screen.findByRole("button", { name: "HTTP health" });
    await user.click(trigger);
    const drawer = screen.getByRole("dialog", { name: "HTTP health" });
    const results = await axe.run(drawer, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
    });
    expect(results.violations).toEqual([]);
    await user.keyboard("{Escape}");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    await waitFor(() => expect(trigger).toHaveFocus());
  });

  it("has no WCAG 2 A/AA axe violations on the dashboard", async () => {
    const { container } = render(<App />);

    await screen.findByText("Checkout service created");
    const results = await axe.run(container, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
    });

    expect(results.violations).toEqual([]);
  });
});
