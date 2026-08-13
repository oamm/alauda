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

  it("has no WCAG 2 A/AA axe violations on the dashboard", async () => {
    const { container } = render(<App />);

    await screen.findByText("Checkout service created");
    const results = await axe.run(container, {
      runOnly: { type: "tag", values: ["wcag2a", "wcag2aa"] },
    });

    expect(results.violations).toEqual([]);
  });
});
