import { object, string, number, boolean, type infer as Infer } from "zod";

export type EndpointKind =
  | "HTTP"
  | "HTTPS"
  | "GRPC"
  | "POSTGRES"
  | "REDIS"
  | "TCP"
  | "UDP"
  | "CUSTOM";

export const endpointKinds = [
  ["HTTP", "HTTP"],
  ["HTTPS", "HTTPS"],
  ["GRPC", "gRPC"],
  ["POSTGRES", "PostgreSQL"],
  ["REDIS", "Redis"],
  ["TCP", "TCP"],
  ["UDP", "UDP"],
  ["CUSTOM", "Custom"],
] as const;

type EndpointKindCapabilities = {
  supportsPath: boolean;
  displayName: string;
  valueFormat: "uri" | "hostPort";
};

export const endpointCapabilities: Record<
  EndpointKind,
  EndpointKindCapabilities
> = {
  HTTP: { supportsPath: true, displayName: "HTTP", valueFormat: "uri" },
  HTTPS: { supportsPath: true, displayName: "HTTPS", valueFormat: "uri" },
  GRPC: { supportsPath: false, displayName: "gRPC", valueFormat: "hostPort" },
  POSTGRES: {
    supportsPath: false,
    displayName: "PostgreSQL",
    valueFormat: "hostPort",
  },
  REDIS: { supportsPath: false, displayName: "Redis", valueFormat: "hostPort" },
  TCP: { supportsPath: false, displayName: "TCP", valueFormat: "hostPort" },
  UDP: { supportsPath: false, displayName: "UDP", valueFormat: "hostPort" },
  CUSTOM: {
    supportsPath: false,
    displayName: "Custom",
    valueFormat: "hostPort",
  },
};

export function normalizeEndpointKind(kind: string): EndpointKind {
  return kind.replace(/^ENDPOINT_KIND_/, "").toUpperCase() as EndpointKind;
}

export function getEndpointKindCapabilities(kind: string) {
  return endpointCapabilities[normalizeEndpointKind(kind)];
}

export function supportsEndpointPath(kind: string) {
  return getEndpointKindCapabilities(kind)?.supportsPath === true;
}
export function endpointPath(kind: string, path = "") {
  return supportsEndpointPath(kind) ? path : "";
}

export const endpointSchema = object({
  name: string().trim().min(1, "Endpoint name is required."),
  kind: string().refine(
    (value) => endpointKinds.some(([kind]) => kind === value),
    "Choose an endpoint type.",
  ),
  port: number()
    .int()
    .min(1, "Port must be between 1 and 65535.")
    .max(65535, "Port must be between 1 and 65535."),
  path: string(),
  primary: boolean(),
  enabled: boolean(),
}).superRefine((value, context) => {
  if (!supportsEndpointPath(value.kind) && value.path)
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
    kind: "HTTP",
    port: 8080,
    path: "/",
    primary: existingNames.length === 0,
    enabled: true,
    ...overrides,
  };
  return { ...value, path: endpointPath(value.kind, value.path) };
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
