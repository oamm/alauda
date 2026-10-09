import { useState } from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { EndpointEditor } from "./EndpointEditor";
import {
  newEndpoint,
  validateEndpoint,
  type EndpointFormValue,
} from "../lib/endpoint-form";
import { CatalogRow, StatusBadge, ResourceList } from "./ui";

function Editor({ protocol = "PROTOCOL_HTTP" }: { protocol?: string }) {
  const [value, setValue] = useState<EndpointFormValue>(
    newEndpoint([], { protocol }),
  );
  return (
    <>
      <EndpointEditor value={value} onChange={setValue} />
      <output aria-label="Draft">{JSON.stringify(value)}</output>
    </>
  );
}
describe("Endpoint protocol capabilities", () => {
  for (const protocol of ["PROTOCOL_HTTP", "PROTOCOL_HTTPS"])
    it(`${protocol} shows and preserves Path`, () => {
      render(<Editor protocol={protocol} />);
      fireEvent.change(screen.getByLabelText("Path"), {
        target: { value: "/metrics" },
      });
      expect(screen.getByLabelText("Draft")).toHaveTextContent(
        '"path":"/metrics"',
      );
    });
  for (const protocol of ["PROTOCOL_TCP", "PROTOCOL_UDP", "PROTOCOL_GRPC"])
    it(`${protocol} hides Path and clears stale state on selection`, () => {
      render(<Editor />);
      fireEvent.change(screen.getByLabelText("Path"), {
        target: { value: "/api" },
      });
      fireEvent.change(screen.getByLabelText("Protocol"), {
        target: { value: protocol },
      });
      expect(screen.queryByLabelText("Path")).not.toBeInTheDocument();
      expect(screen.getByLabelText("Draft")).toHaveTextContent('"path":""');
      fireEvent.change(screen.getByLabelText("Protocol"), {
        target: { value: "PROTOCOL_HTTP" },
      });
      expect(screen.getByLabelText("Path")).toHaveValue("");
    });
  it("rejects incompatible paths in shared validation rather than silently submitting", () => {
    for (const protocol of ["PROTOCOL_TCP", "PROTOCOL_UDP", "PROTOCOL_GRPC"])
      expect(
        validateEndpoint({ ...newEndpoint(), protocol, path: "/bad" }).path,
      ).toBeDefined();
    for (const path of [
      "//host",
      "/api?query=1",
      "/api#fragment",
      "/bad\\path",
    ])
      expect(validateEndpoint({ ...newEndpoint(), path }).path).toBeDefined();
  });
  it("normalizes legacy non-path values when loading editable defaults", () => {
    expect(
      newEndpoint([], { protocol: "PROTOCOL_TCP", path: "/legacy" }).path,
    ).toBe("");
  });
});
describe("Compact catalog rows", () => {
  it("presents one full accessible identity, count, status and selected state", () => {
    const name = "ExternalServices.Betterstack.Ingestion";
    render(
      <ResourceList label="Catalog">
        <CatalogRow
          name={name}
          metadata="2 instances"
          status={<StatusBadge status="unknown" />}
          selected
          onSelect={() => {}}
        />
      </ResourceList>,
    );
    expect(screen.getAllByText(name)).toHaveLength(1);
    expect(screen.getByTitle(name)).toBeInTheDocument();
    expect(screen.getByRole("button", { name })).toHaveAttribute(
      "aria-pressed",
      "true",
    );
    expect(screen.getByText("2 instances")).toBeInTheDocument();
  });
});
