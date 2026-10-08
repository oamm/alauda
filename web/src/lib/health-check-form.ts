import { object, string, number, boolean, type infer as Infer } from "zod";
export const healthTypes = ["HTTP", "HTTPS", "TCP", "GRPC"];
export const allHealthTypes = [...healthTypes, "UDP", "HEARTBEAT"];
const healthCheckFields = object({
  name: string().trim().min(1, "Check name is required."),
  instanceId: string().min(1, "Select an instance."),
  endpointId: string(),
  type: string().refine(
    (value) => healthTypes.includes(value.replace("HEALTH_CHECK_TYPE_", "")),
    "Select a supported type.",
  ),
  enabled: boolean(),
  intervalSeconds: number().int().min(1),
  timeoutSeconds: number().int().min(1),
  failuresBeforeUnhealthy: number().int().min(1),
  successesBeforeHealthy: number().int().min(1),
  path: string(),
  expectedStatus: string(),
  description: string(),
});
export const healthCheckEditSchema = healthCheckFields.pick({
  enabled: true,
  intervalSeconds: true,
  timeoutSeconds: true,
  failuresBeforeUnhealthy: true,
  successesBeforeHealthy: true,
  description: true,
});
export const healthCheckSchema = healthCheckFields.superRefine((value, ctx) => {
  if (
    value.type.includes("HTTP") &&
    value.expectedStatus &&
    !value.expectedStatus.split(",").every((part) => {
      const bounds = part.trim().split("-").map(Number);
      return (
        bounds.length <= 2 &&
        bounds.every((n) => Number.isInteger(n) && n >= 100 && n <= 599) &&
        (bounds.length === 1 || bounds[0] <= bounds[1])
      );
    })
  )
    ctx.addIssue({
      code: "custom",
      path: ["expectedStatus"],
      message: "Use HTTP codes or ranges, such as 200-299,304.",
    });
});
export type HealthCheckValue = Infer<typeof healthCheckSchema>;
export function healthCheckDefaults(
  context: Partial<HealthCheckValue> = {},
): HealthCheckValue {
  return {
    name: "readiness",
    instanceId: "",
    endpointId: "",
    type: "HEALTH_CHECK_TYPE_HTTP",
    enabled: true,
    intervalSeconds: 10,
    timeoutSeconds: 3,
    failuresBeforeUnhealthy: 3,
    successesBeforeHealthy: 2,
    path: "/healthz",
    expectedStatus: "200-299",
    description: "",
    ...context,
  };
}
