import { object, string, number, boolean, type infer as Infer } from "zod";

export const endpointProtocols = [
  ["PROTOCOL_HTTP", "HTTP"],
  ["PROTOCOL_HTTPS", "HTTPS"],
  ["PROTOCOL_GRPC", "gRPC"],
  ["PROTOCOL_TCP", "TCP"],
  ["PROTOCOL_UDP", "UDP"],
] as const;

export const endpointCapabilities: Record<string, { supportsPath: boolean }> =
  Object.fromEntries(
    endpointProtocols.map(([protocol]) => [
      protocol,
      {
        supportsPath:
          protocol === "PROTOCOL_HTTP" || protocol === "PROTOCOL_HTTPS",
      },
    ]),
  );
export function supportsEndpointPath(protocol: string) {
  return endpointCapabilities[protocol]?.supportsPath === true;
}
export function endpointPath(protocol: string, path = "") {
  return supportsEndpointPath(protocol) ? path : "";
}

export const endpointSchema = object({
  name: string().trim().min(1, "Endpoint name is required."),
  protocol: string().refine(
    (value) => endpointProtocols.some(([protocol]) => protocol === value),
    "Choose a protocol.",
  ),
  port: number()
    .int()
    .min(1, "Port must be between 1 and 65535.")
    .max(65535, "Port must be between 1 and 65535."),
  path: string(),
  primary: boolean(),
  enabled: boolean(),
}).superRefine((value, context) => {
  if (!supportsEndpointPath(value.protocol) && value.path)
    context.addIssue({
      code: "custom",
      path: ["path"],
      message: "Path is only supported for HTTP and HTTPS endpoints.",
    });
  else if (
    /[?#\\\x00-\x1f\x7f]/.test(value.path) ||
    value.path.startsWith("//")
  )
    context.addIssue({
      code: "custom",
      path: ["path"],
      message: "Use a path without query, fragment or authority.",
    });
});
export type EndpointFormValue = Infer<typeof endpointSchema>;
export type EndpointErrors = Partial<Record<keyof EndpointFormValue, string>>;

export function validateEndpoint(
  value: EndpointFormValue,
  existingNames: string[] = [],
): EndpointErrors {
  const result = endpointSchema.safeParse(value);
  const errors: EndpointErrors = {};
  if (!result.success)
    for (const issue of result.error.issues) {
      errors[issue.path[0] as keyof EndpointFormValue] = issue.message;
    }
  if (value.name.trim() && existingNames.includes(value.name.trim())) {
    errors.name = `An endpoint named "${value.name.trim()}" already exists in this instance.`;
  }
  return errors;
}

export function newEndpoint(
  existingNames: string[] = [],
  overrides: Partial<EndpointFormValue> = {},
): EndpointFormValue {
  let name = "default";
  for (let suffix = 2; existingNames.includes(name); suffix++)
    name = `default-${suffix}`;
  const value = {
    name,
    protocol: "PROTOCOL_HTTP",
    port: 8080,
    path: "/",
    primary: existingNames.length === 0,
    enabled: true,
    ...overrides,
  };
  return { ...value, path: endpointPath(value.protocol, value.path) };
}

export function validateEndpointCollection(values: EndpointFormValue[]) {
  const errors = values.map((value, index) =>
    validateEndpoint(
      value,
      values
        .filter((_, other) => other !== index)
        .map((endpoint) => endpoint.name.trim()),
    ),
  );
  if (values.filter((value) => value.primary).length > 1) {
    errors.forEach((error, index) => {
      if (values[index].primary)
        error.primary = "Only one endpoint can be primary.";
    });
  }
  return errors;
}
