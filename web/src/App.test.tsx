import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import axe from "axe-core";
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
      vi.fn(async (input: RequestInfo | URL) => {
        const url = input.toString();
        return new Response(JSON.stringify(mockResponse(url)), {
          headers: { "Content-Type": "application/json" },
          status: 200,
        });
      }),
    );
    vi.stubGlobal("EventSource", MockEventSource);
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
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
    fireEvent.click(screen.getByRole("button", { name: "Details" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Incident Detail")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Close" }));
    fireEvent.click(screen.getByRole("button", { name: "Events" }));
    expect(screen.getAllByText("service.created").length).toBeGreaterThan(0);
    expect(screen.getByText("Checkout service created")).toBeInTheDocument();
  });

  it("shows security users and API tokens", async () => {
    window.localStorage.setItem("registryToken", "sr_test");
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Security" }));

    await waitFor(() =>
      expect(screen.getAllByText("admin").length).toBeGreaterThan(0),
    );
    expect(screen.getByText("automation")).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Create token" }),
    ).toBeInTheDocument();
  });

  it("registers runtime through the composite API", async () => {
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));

    fireEvent.click(screen.getByRole("tab", { name: "Runtime" }));
    fireEvent.click(screen.getByRole("button", { name: "+ Add instance" }));
    fireEvent.change(screen.getByLabelText("Instance name"), {
      target: { value: "checkout-prod-02" },
    });
    fireEvent.change(screen.getByLabelText("Address"), {
      target: { value: "10.0.0.2" },
    });
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "http" },
    });
    const addInstanceButtons = screen.getAllByRole("button", {
      name: "Add instance",
    });
    fireEvent.click(addInstanceButtons[addInstanceButtons.length - 1]);

    await screen.findByText(
      "Runtime registered successfully: checkout-prod-02.",
    );

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
      name: "http",
      port: 8080,
      primary: true,
    });
  });

  it("scopes runtime registration to the selected service workspace", async () => {
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));

    expect(screen.getByRole("tab", { name: "Overview" })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.queryByText("Register Runtime")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Target" })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("tab", { name: "Runtime" }));
    expect(screen.getByText("Endpoints")).toBeInTheDocument();
    expect(screen.getAllByText("Health").length).toBeGreaterThan(0);

    expect(
      screen.queryByRole("button", { name: "Add instance" }),
    ).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "+ Add instance" }));
    expect(screen.getAllByText("Service").length).toBeGreaterThan(0);
    expect(screen.getAllByText("checkout").length).toBeGreaterThan(0);
    expect(screen.queryByLabelText("Service")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Environment")).toHaveValue("env-1");
    fireEvent.change(screen.getByLabelText("Instance name"), {
      target: { value: "checkout-prod-03" },
    });
    fireEvent.change(screen.getByLabelText("Address"), {
      target: { value: "10.0.0.3" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    fireEvent.click(screen.getByRole("button", { name: "+ Add instance" }));
    expect(screen.getByLabelText("Instance name")).toHaveValue(
      "checkout-prod-03",
    );
    expect(screen.getByLabelText("Address")).toHaveValue("10.0.0.3");
  });

  it("expands health monitoring locally without submitting or losing form state", async () => {
    const startingUrl = window.location.href;
    render(<App />);

    await screen.findByText("Checkout service created");
    fireEvent.click(screen.getByRole("button", { name: "Services" }));
    fireEvent.click(screen.getByRole("tab", { name: "Runtime" }));
    fireEvent.click(screen.getByRole("button", { name: "+ Add instance" }));

    fireEvent.change(screen.getByLabelText("Instance name"), {
      target: { value: "checkout-prod-04" },
    });
    fireEvent.change(screen.getByLabelText("Address"), {
      target: { value: "10.0.0.4" },
    });
    fireEvent.change(screen.getByLabelText("Name"), {
      target: { value: "public" },
    });

    const healthToggle = screen.getByRole("button", {
      name: /Health monitoring/i,
    });
    expect(healthToggle).toHaveAttribute("aria-expanded", "false");

    fireEvent.click(healthToggle);

    expect(healthToggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByLabelText("Check type")).toBeInTheDocument();
    expect(screen.getByLabelText("Interval")).toBeInTheDocument();
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByLabelText("Instance name")).toHaveValue(
      "checkout-prod-04",
    );
    expect(screen.getByLabelText("Address")).toHaveValue("10.0.0.4");
    expect(screen.getByLabelText("Name")).toHaveValue("public");
    expect(window.location.href).toBe(startingUrl);
    expect(
      vi.mocked(fetch).mock.calls.filter(([input]) =>
        input.toString().includes("RegisterRuntime"),
      ),
    ).toHaveLength(0);

    fireEvent.click(healthToggle);

    expect(healthToggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByLabelText("Check type")).not.toBeInTheDocument();
    expect(screen.getByLabelText("Instance name")).toHaveValue(
      "checkout-prod-04",
    );
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
