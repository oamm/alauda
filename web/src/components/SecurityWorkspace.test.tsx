import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { SecurityWorkspace } from "./SecurityWorkspace";

const api = vi.hoisted(() => ({
  listUsers: vi.fn(), listApiTokens: vi.fn(), listApplicationKeys: vi.fn(), listSessions: vi.fn(),
  createUser: vi.fn(), createApiToken: vi.fn(), createApplicationKey: vi.fn(),
  revokeApiToken: vi.fn(), revokeApplicationKey: vi.fn(), revokeSession: vi.fn(),
}));
vi.mock("../api", () => ({
  ...api,
  credentialCapabilities: ["discovery.read", "registry.read", "registry.write", "admin"],
}));

const environments = [
  { id: "env-stg", key: "stg", name: "Staging", enabled: true },
  { id: "env-prod", key: "prod", name: "Production", enabled: true },
];

beforeEach(() => {
  vi.clearAllMocks();
  api.listUsers.mockResolvedValue([{ id: "user-1", username: "root", displayName: "Root Administrator",
    email: "root@example.local", role: "Administrator", enabled: true, createdAt: "2026-10-01T00:00:00Z" }]);
  api.listApiTokens.mockResolvedValue([{ id: "token-1", userId: "user-1", name: "automation",
    scopes: ["registry.read"], createdBy: "user-1", enabled: true }]);
  api.listApplicationKeys.mockResolvedValue([{ id: "key-1", name: "service-client",
    scopes: ["discovery.read"], environmentIds: ["env-stg"], createdBy: "user-1", enabled: true }]);
  api.listSessions.mockResolvedValue([{ id: "session-1", userId: "user-1",
    createdAt: "2026-10-01T00:00:00Z", lastActivityAt: "2026-10-02T00:00:00Z",
    expiresAt: "2026-11-01T00:00:00Z", clientIp: "127.0.0.1", userAgent: "Browser" }]);
  api.createUser.mockResolvedValue({ id: "user-2", username: "operator" });
  api.createApiToken.mockResolvedValue({ secret: "one-time-token-secret" });
  api.createApplicationKey.mockResolvedValue({ secret: "one-time-key-secret" });
  api.revokeSession.mockResolvedValue(undefined);
  api.revokeApiToken.mockResolvedValue(undefined);
  api.revokeApplicationKey.mockResolvedValue(undefined);
});

describe("Security workspace", () => {
  it("uses accessible tabs and canonical resource tables", async () => {
    render(<SecurityWorkspace environments={environments} />);
    expect(screen.getByRole("heading", { name: "Security" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Users" })).toHaveAttribute("aria-selected", "true");
    expect(await screen.findByRole("table", { name: "Users" })).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /root Root Administrator/ })).toBeInTheDocument();
    expect(screen.queryByText("—")).not.toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole("tab", { name: "Users" }), { key: "ArrowRight" });
    expect(screen.getByRole("tab", { name: "API tokens" })).toHaveAttribute("aria-selected", "true");
    expect(screen.getByRole("table", { name: "API tokens" })).toBeInTheDocument();
    expect(screen.queryByRole("table", { name: "Users" })).not.toBeInTheDocument();
  });

  it("creates users in a shared dialog and keeps password out of the list", async () => {
    render(<SecurityWorkspace environments={environments} />);
    await screen.findByRole("table", { name: "Users" });
    fireEvent.click(screen.getByRole("button", { name: "Create user" }));
    const dialog = screen.getByRole("dialog", { name: "Create user" });
    fireEvent.change(within(dialog).getByLabelText("Username"), { target: { value: "operator" } });
    fireEvent.change(within(dialog).getByLabelText("Display name"), { target: { value: "Operator" } });
    fireEvent.change(within(dialog).getByLabelText("Email"), { target: { value: "operator@example.local" } });
    fireEvent.change(within(dialog).getByLabelText("Password"), { target: { value: "private-password" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Create user" }));
    await waitFor(() => expect(api.createUser).toHaveBeenCalledWith(expect.objectContaining({ username: "operator", password: "private-password" })));
    await waitFor(() => expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
    expect(screen.queryByText("private-password")).not.toBeInTheDocument();
  });

  it("pages a large user list without growing the workspace indefinitely", async () => {
    api.listUsers.mockResolvedValue(Array.from({ length: 27 }, (_, index) => ({
      id: `user-${index}`, username: `operator-${index}`, displayName: `Operator ${index}`,
      email: `operator-${index}@example.local`, role: "Operator", enabled: true,
    })));
    render(<SecurityWorkspace environments={environments} />);
    const table = await screen.findByRole("table", { name: "Users" });
    expect(within(table).getByText("operator-0")).toBeInTheDocument();
    expect(within(table).queryByText("operator-26")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Next" }));
    expect(within(table).getByText("operator-26")).toBeInTheDocument();
    expect(within(table).queryByText("operator-0")).not.toBeInTheDocument();
  });

  it("distinguishes an empty filter result from an empty user catalog", async () => {
    render(<SecurityWorkspace environments={environments} />);
    await screen.findByRole("table", { name: "Users" });
    fireEvent.change(screen.getByRole("searchbox", { name: "Search" }), { target: { value: "absent" } });
    expect(screen.getByText("No matching users")).toBeInTheDocument();
    expect(screen.queryByText("No users")).not.toBeInTheDocument();
  });

  it("shows an API token only once inside its creation dialog", async () => {
    render(<SecurityWorkspace environments={environments} />);
    fireEvent.click(screen.getByRole("tab", { name: "API tokens" }));
    await screen.findByRole("table", { name: "API tokens" });
    expect(screen.queryByText("one-time-token-secret")).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Create API token" }));
    const dialog = screen.getByRole("dialog", { name: "Create API token" });
    fireEvent.change(within(dialog).getByLabelText("Name"), { target: { value: "ci" } });
    fireEvent.click(within(dialog).getByRole("button", { name: "Create API token" }));
    expect(await screen.findByRole("dialog", { name: "API token created" })).toHaveTextContent("one-time-token-secret");
    expect(api.createApiToken).toHaveBeenCalledWith(expect.objectContaining({ userId: "user-1", name: "ci" }));
    fireEvent.click(screen.getByRole("button", { name: "Done" }));
    expect(screen.queryByText("one-time-token-secret")).not.toBeInTheDocument();
  });

  it("keeps Application Key scope and Environment selection explicit", async () => {
    render(<SecurityWorkspace environments={environments} />);
    fireEvent.click(screen.getByRole("tab", { name: "Application keys" }));
    await screen.findByRole("table", { name: "Application keys" });
    expect(screen.getAllByText("stg").length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("button", { name: "Create application key" }));
    const dialog = screen.getByRole("dialog", { name: "Create application key" });
    expect(within(dialog).getByRole("checkbox", { name: "discovery.read" })).toBeChecked();
    expect(within(dialog).getByRole("checkbox", { name: "admin" })).not.toBeChecked();
    fireEvent.change(within(dialog).getByLabelText("Name"), { target: { value: "payments" } });
    fireEvent.click(within(dialog).getByRole("checkbox", { name: "Production" }));
    fireEvent.click(within(dialog).getByRole("button", { name: "Create application key" }));
    await waitFor(() => expect(api.createApplicationKey).toHaveBeenCalledWith(expect.objectContaining({
      name: "payments", scopes: ["discovery.read"], environmentIds: ["env-prod"],
    })));
    expect(await screen.findByRole("dialog", { name: "Application key created" })).toHaveTextContent("one-time-key-secret");
  });

  it("revokes a session through the shared action menu and confirmation dialog", async () => {
    render(<SecurityWorkspace environments={environments} />);
    fireEvent.click(screen.getByRole("tab", { name: "Sessions" }));
    await screen.findByRole("table", { name: "Sessions" });
    fireEvent.keyDown(screen.getAllByRole("button", { name: "Actions for root session" })[0], { key: "Enter" });
    fireEvent.click(await screen.findByRole("menuitem", { name: "Revoke" }));
    const dialog = await screen.findByRole("dialog", { name: "Revoke access" });
    fireEvent.click(within(dialog).getByRole("button", { name: "Revoke" }));
    await waitFor(() => expect(api.revokeSession).toHaveBeenCalledWith("session-1"));
  });
});
