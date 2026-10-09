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

function Editor({ kind = "HTTP" }: { kind?: string }) {
  const [value, setValue] = useState<EndpointFormValue>(
    newEndpoint([], { kind }),
  );
  return (
    <>
      <EndpointEditor value={value} onChange={setValue} />
      <output aria-label="Draft">{JSON.stringify(value)}</output>
    </>
  );
}
describe("kind capabilities", () => {
  for (const kind of ["HTTP", "HTTPS"])
    it(`${kind} shows and preserves Path`, () => {
      render(<Editor kind={kind} />);
      fireEvent.change(screen.getByLabelText("Path"), {
        target: { value: "/metrics" },
      });
      expect(screen.getByLabelText("Draft")).toHaveTextContent(
        '"path":"/metrics"',
      );
    });
  for (const kind of [
    "TCP",
    "UDP",
    "GRPC",
    "POSTGRES",
    "REDIS",
    "CUSTOM",
  ])
    it(`${kind} hides Path and clears stale state on selection`, () => {
      render(<Editor />);
      fireEvent.change(screen.getByLabelText("Path"), {
        target: { value: "/api" },
      });
      fireEvent.change(screen.getByLabelText("Endpoint type"), {
        target: { value: kind },
      });
      expect(screen.queryByLabelText("Path")).not.toBeInTheDocument();
      expect(screen.getByLabelText("Draft")).toHaveTextContent('"path":""');
      fireEvent.change(screen.getByLabelText("Endpoint type"), {
        target: { value: "HTTP" },
      });
      expect(screen.getByLabelText("Path")).toHaveValue("");
    });
  it("rejects incompatible paths in shared validation rather than silently submitting", () => {
    for (const kind of [
      "TCP",
      "UDP",
      "GRPC",
      "POSTGRES",
      "REDIS",
      "CUSTOM",
    ])
      expect(
        validateEndpoint({ ...newEndpoint(), kind, path: "/bad" }).path,
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
      newEndpoint([], { kind: "TCP", path: "/legacy" }).path,
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
