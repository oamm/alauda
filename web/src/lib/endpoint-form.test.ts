import { describe, expect, it } from "vitest";
import {
  newEndpoint,
  validateEndpoint,
  validateEndpointCollection,
} from "./endpoint-form";

describe("shared endpoint semantics", () => {
  it("suggests deterministic instance-local names with editable defaults", () => {
    expect(newEndpoint()).toMatchObject({
      name: "default",
      primary: true,
      enabled: true,
    });
    expect(newEndpoint(["default"])).toMatchObject({
      name: "default-2",
      primary: false,
      enabled: true,
    });
    expect(newEndpoint(["default", "default-2"])).toMatchObject({
      name: "default-3",
    });
    expect(newEndpoint(["default", "default-3"])).toMatchObject({
      name: "default-2",
    });
    expect(newEndpoint([], { name: "admin" }).name).toBe("admin");
  });
  it("validates names only against the supplied instance scope", () => {
    expect(validateEndpoint(newEndpoint(), [])).toEqual({});
    expect(validateEndpoint(newEndpoint(), ["default"]).name).toContain(
      '"default"',
    );
    expect(
      validateEndpoint(newEndpoint([], { name: " default " }), ["default"])
        .name,
    ).toBeDefined();
    expect(
      validateEndpoint(newEndpoint([], { name: "Default" }), ["default"]),
    ).toEqual({});
  });
  it("shares required name, protocol, integer port and boolean validation", () => {
    expect(validateEndpoint(newEndpoint([], { name: " " })).name).toBeDefined();
    expect(
      validateEndpoint(newEndpoint([], { protocol: "PROTOCOL_UNSPECIFIED" }))
        .protocol,
    ).toBeDefined();
    for (const port of [0, 65536, 12.5, NaN])
      expect(validateEndpoint(newEndpoint([], { port })).port).toBeDefined();
    expect(
      validateEndpoint(
        newEndpoint([], { enabled: "false" as unknown as boolean }),
      ).enabled,
    ).toBeDefined();
    expect(
      validateEndpoint(
        newEndpoint([], {
          protocol: "PROTOCOL_TCP",
          path: "",
          enabled: false,
          primary: false,
        }),
      ),
    ).toEqual({});
  });
  it("validates draft names and at-most-one primary without inventing address uniqueness", () => {
    expect(
      validateEndpointCollection([newEndpoint(), newEndpoint(["default"])]),
    ).toEqual([{}, {}]);
    expect(
      validateEndpointCollection([newEndpoint(), newEndpoint([])]).every(
        (error) => error.name && error.primary,
      ),
    ).toBe(true);
    expect(validateEndpointCollection([])).toEqual([]);
  });
});
