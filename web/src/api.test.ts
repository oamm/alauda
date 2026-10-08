import { afterEach, describe, expect, it, vi } from "vitest";
import {
  APIError,
  credentialCapabilities,
  listEnvironments,
  listServices,
  registerServiceInstance,
  setAuthenticationFailureHandler,
  setBearerToken,
} from "./api";

const response = (data: unknown, status = 200) =>
  new Response(JSON.stringify(data), {
    status,
    headers: {
      "Content-Type":
        status >= 400 ? "application/problem+json" : "application/json",
    },
  });

afterEach(() => {
  setBearerToken("");
  setAuthenticationFailureHandler(undefined);
  vi.unstubAllGlobals();
});

describe("Public browser contract", () => {
  it("registers by keys in one request and consumes canonical addresses", async () => {
    const fetch = vi.fn(async () =>
      response({
        service: "Authentication.Grpc",
        environment: "stg",
        instance: { id: "i1", name: "auth-01", address: "::1", enabled: true },
        endpoints: [
          {
            id: "e1",
            name: "default",
            protocol: "http",
            port: 81,
            path: "/",
            enabled: true,
            primary: true,
            address: "http://[::1]:81/",
          },
        ],
      }),
    );
    vi.stubGlobal("fetch", fetch);
    const input = {
      service: "Authentication.Grpc",
      environment: "stg",
      instance: {
        name: "auth-01",
        address: "::1",
        description: "",
        enabled: true,
      },
      endpoints: [
        {
          name: "default",
          protocol: "PROTOCOL_HTTP",
          port: 81,
          path: "/",
          enabled: true,
          primary: false,
        },
      ],
    };
    const first = await registerServiceInstance(input);
    const second = await registerServiceInstance(input);
    expect(fetch).toHaveBeenCalledTimes(2);
    expect(fetch.mock.calls[0]).toMatchObject([
      "/api/v1/services/Authentication.Grpc/instances",
      { method: "POST", credentials: "include" },
    ]);
    const payload = JSON.parse(
      String((fetch.mock.calls[0] as unknown as [string, RequestInit])[1].body),
    );
    expect(payload).not.toHaveProperty("serviceId");
    expect(payload).not.toHaveProperty("environmentId");
    expect(payload).not.toHaveProperty("deploymentId");
    expect(payload.environment).toBe("stg");
    expect(payload.endpoints[0]).toMatchObject({
      protocol: "http",
      primary: false,
    });
    expect(first.instance.id).toBe(second.instance.id);
    expect(first.endpoints[0]).toMatchObject({
      instanceId: "i1",
      address: "http://[::1]:81/",
      protocol: "PROTOCOL_HTTP",
    });
  });

  it("loads all public catalog pages using keys and top-level page tokens", async () => {
    const fetch = vi.fn(async (input: string) => {
      const url = new URL(input, "http://localhost");
      const field = url.pathname.endsWith("environments")
        ? "environments"
        : "services";
      const next = url.searchParams.get("pageToken");
      return response({
        [field]: [{ id: next ? "second" : "first" }],
        nextPageToken: next ? "" : "100",
      });
    });
    vi.stubGlobal("fetch", fetch);
    expect(await listEnvironments()).toHaveLength(2);
    expect(await listServices("stg")).toHaveLength(2);
    expect(fetch).toHaveBeenCalledTimes(4);
    for (const [url] of fetch.mock.calls.slice(2)) {
      expect(
        new URL(url, "http://localhost").searchParams.get("environment"),
      ).toBe("stg");
      expect(url).not.toContain("environmentId");
    }
  });

  it("renders Problem Details fields without raw JSON while preserving error codes", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        response(
          {
            code: "validation_failed",
            detail: "Registration validation failed.",
            errors: {
              "endpoints[0].port": ["Port must be between 1 and 65535."],
            },
          },
          400,
        ),
      ),
    );
    const promise = listServices();
    await expect(promise).rejects.toBeInstanceOf(APIError);
    await expect(promise).rejects.toMatchObject({
      status: 400,
      code: "validation_failed",
      message:
        "Registration validation failed. endpoints[0].port: Port must be between 1 and 65535.",
    });
  });

  it("preserves the session-expiration callback on public requests", async () => {
    const expired = vi.fn();
    setAuthenticationFailureHandler(expired);
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        response(
          { code: "authentication_required", detail: "Sign in again." },
          401,
        ),
      ),
    );
    await expect(listServices()).rejects.toMatchObject({
      status: 401,
      code: "authentication_required",
      message: "Sign in again.",
    });
    expect(expired).toHaveBeenCalledOnce();
  });

  it("offers granular capabilities instead of legacy broad scopes", () => {
    expect(credentialCapabilities).toContain("discovery.read");
    expect(credentialCapabilities).toContain("health.execute");
    expect(credentialCapabilities).not.toContain("read");
    expect(credentialCapabilities).not.toContain("write");
  });
});
