import { useState } from "react";
import {
  createHealthCheck,
  updateHealthCheck,
  type HealthCheck,
  type ServiceInstance,
  type Endpoint,
  type Service,
  type Environment,
  type ServiceDeployment,
} from "../api";
import {
  healthCheckDefaults,
  healthCheckSchema,
  healthCheckEditSchema,
  healthTypes,
  type HealthCheckValue,
} from "../lib/health-check-form";
import {
  ActionGroup,
  Alert,
  Button,
  FormField,
  Input,
  Select,
  Switch,
  Textarea,
} from "./ui";

export type HealthCheckFormProps = {
  instances: ServiceInstance[];
  endpoints: Endpoint[];
  services?: Service[];
  environments?: Environment[];
  deployments?: ServiceDeployment[];
  serviceId?: string;
  environmentId?: string;
  instanceId?: string;
  initialInstanceId?: string;
  endpointId?: string;
  check?: HealthCheck;
  onSaved: (check: HealthCheck) => void | Promise<void>;
  onCancel: () => void;
};
export function HealthCheckForm(props: HealthCheckFormProps) {
  const [value, setValue] = useState(() =>
    healthCheckDefaults(
      props.check
        ? {
            ...props.check,
            path: props.check.metadata?.path || "",
            expectedStatus: props.check.metadata?.expectedStatus || "",
          }
        : {
            instanceId: props.instanceId || props.initialInstanceId || "",
            endpointId: props.endpointId || "",
          },
    ),
  );
  const [service, setService] = useState(props.serviceId || "");
  const [environment, setEnvironment] = useState(props.environmentId || "");
  const [errors, setErrors] = useState<
    Partial<Record<keyof HealthCheckValue, string>>
  >({});
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const editing = !!props.check;
  const immutable = editing;
  const targets = props.instances.filter((instance) => {
    const d = props.deployments?.find((d) => d.id === instance.deploymentId);
    return (
      (!service || !props.deployments || d?.serviceId === service) &&
      (!environment || !props.deployments || d?.environmentId === environment)
    );
  });
  const endpoints = props.endpoints.filter(
    (endpoint) => endpoint.instanceId === value.instanceId,
  );
  function change(patch: Partial<HealthCheckValue>) {
    setValue((current) => ({ ...current, ...patch }));
    setErrors({});
    setError("");
  }
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (saving || saved) return;
    const validation = (
      editing ? healthCheckEditSchema : healthCheckSchema
    ).safeParse(value);
    const next: typeof errors = {};
    if (!validation.success)
      for (const issue of validation.error.issues)
        next[issue.path[0] as keyof HealthCheckValue] = issue.message;
    if (
      !editing &&
      !targets.some((instance) => instance.id === value.instanceId)
    )
      next.instanceId = "Select an instance in this scope.";
    if (
      !editing &&
      value.endpointId &&
      !endpoints.some((endpoint) => endpoint.id === value.endpointId)
    )
      next.endpointId = "Select an endpoint belonging to this instance.";
    if (Object.keys(next).length) {
      setErrors(next);
      return;
    }
    setSaving(true);
    setError("");
    try {
      const check = editing
        ? await updateHealthCheck({
            id: props.check!.id,
            enabled: value.enabled,
            intervalSeconds: value.intervalSeconds,
            timeoutSeconds: value.timeoutSeconds,
            failuresBeforeUnhealthy: value.failuresBeforeUnhealthy,
            successesBeforeHealthy: value.successesBeforeHealthy,
            description: value.description,
          })
        : await createHealthCheck({
            ...value,
            name: value.name.trim(),
            metadata: {
              path: value.path,
              expectedStatus: value.expectedStatus,
            },
          });
      setSaved(true);
      try {
        await props.onSaved(check);
      } catch {
        setError(
          "Health check saved, but the view could not refresh. Close this form and refresh the page.",
        );
      }
    } catch (e) {
      let message =
        e instanceof Error ? e.message : "Could not save health monitoring.";
      try {
        const parsed = JSON.parse(message);
        message =
          parsed.code === "already_exists"
            ? "A check with this name already exists on this instance."
            : parsed.message || message;
      } catch {}
      setError(message);
    } finally {
      setSaving(false);
    }
  }
  return (
    <form className="grid min-w-0 gap-3" onSubmit={submit}>
      {error ? (
        <Alert
          tone="danger"
          title={
            saved
              ? "Health check saved"
              : "Health monitoring could not be configured."
          }
        >
          {error}
          <p>The instance and its endpoints already exist.</p>
          {!saved ? (
            <Button size="sm" type="submit" disabled={saving}>
              Retry
            </Button>
          ) : null}
        </Alert>
      ) : null}
      <div className="grid min-w-0 gap-3 sm:grid-cols-2">
        {!editing && props.services && !props.serviceId ? (
          <FormField label="Service">
            <Select
              value={service}
              disabled={saving || saved}
              onChange={(e) => {
                setService(e.target.value);
                change({ instanceId: "", endpointId: "" });
              }}
            >
              <option value="">All services</option>
              {props.services.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.displayName || s.name}
                </option>
              ))}
            </Select>
          </FormField>
        ) : null}
        {!editing && props.environments && !props.environmentId ? (
          <FormField label="Environment">
            <Select
              value={environment}
              disabled={saving || saved}
              onChange={(e) => {
                setEnvironment(e.target.value);
                change({ instanceId: "", endpointId: "" });
              }}
            >
              <option value="">All environments</option>
              {props.environments.map((e) => (
                <option key={e.id} value={e.id}>
                  {e.name}
                </option>
              ))}
            </Select>
          </FormField>
        ) : null}
        {!props.instanceId && !editing ? (
          <FormField label="Instance" error={errors.instanceId}>
            <Select
              value={value.instanceId}
              disabled={saving || saved}
              onChange={(e) =>
                change({ instanceId: e.target.value, endpointId: "" })
              }
            >
              <option value="">Select instance</option>
              {targets.map((i) => (
                <option key={i.id} value={i.id}>
                  {i.name} · {i.address}
                </option>
              ))}
            </Select>
          </FormField>
        ) : (
          <p className="text-xs text-[var(--text-muted)] sm:col-span-2">
            {props.instances.find((i) => i.id === value.instanceId)?.name ||
              "Instance"}
          </p>
        )}
        <FormField label="Endpoint" error={errors.endpointId}>
          <Select
            disabled={immutable || saving}
            value={value.endpointId}
            onChange={(e) => change({ endpointId: e.target.value })}
          >
            <option value="">Instance address</option>
            {endpoints.map((e) => (
              <option key={e.id} value={e.id}>
                {e.name} · :{e.port}
              </option>
            ))}
          </Select>
        </FormField>
        <FormField label="Check name" error={errors.name}>
          <Input
            readOnly={immutable}
            value={value.name}
            disabled={saving}
            onChange={(e) => change({ name: e.target.value })}
          />
        </FormField>
        <FormField label="Type" error={errors.type}>
          <Select
            disabled={immutable || saving}
            value={value.type}
            onChange={(e) => change({ type: e.target.value })}
          >
            {healthTypes.map((t) => (
              <option key={t} value={`HEALTH_CHECK_TYPE_${t}`}>
                {t === "GRPC" ? "gRPC" : t}
              </option>
            ))}
            {editing &&
            !healthTypes.includes(
              value.type.replace("HEALTH_CHECK_TYPE_", ""),
            ) ? (
              <option value={value.type}>
                {value.type.replace("HEALTH_CHECK_TYPE_", "")}
              </option>
            ) : null}
          </Select>
        </FormField>
        <FormField label="Path" error={errors.path}>
          <Input
            readOnly={immutable}
            disabled={saving}
            value={value.path}
            onChange={(e) => change({ path: e.target.value })}
          />
        </FormField>
        <FormField label="Expected status" error={errors.expectedStatus}>
          <Input
            readOnly={immutable}
            disabled={saving || !value.type.includes("HTTP")}
            value={value.expectedStatus}
            onChange={(e) => change({ expectedStatus: e.target.value })}
          />
        </FormField>
        {(
          [
            ["intervalSeconds", "Interval (seconds)"],
            ["timeoutSeconds", "Timeout (seconds)"],
            ["failuresBeforeUnhealthy", "Failure threshold"],
            ["successesBeforeHealthy", "Recovery threshold"],
          ] as const
        ).map(([key, label]) => (
          <FormField key={key} label={label} error={errors[key]}>
            <Input
              disabled={saving}
              type="number"
              min={1}
              step={1}
              value={value[key]}
              onChange={(e) => change({ [key]: Number(e.target.value) })}
            />
          </FormField>
        ))}
        <FormField label="Enabled">
          <Switch
            disabled={saving}
            checked={value.enabled}
            onCheckedChange={(enabled) => change({ enabled })}
          />
        </FormField>
        <FormField label="Description" className="sm:col-span-2">
          <Textarea
            disabled={saving}
            value={value.description}
            onChange={(e) => change({ description: e.target.value })}
          />
        </FormField>
      </div>
      <ActionGroup>
        <Button variant="ghost" disabled={saving} onClick={props.onCancel}>
          Cancel
        </Button>
        <Button
          type="submit"
          variant="primary"
          disabled={saving}
          loading={saving}
        >
          {editing ? "Save changes" : "Create health check"}
        </Button>
      </ActionGroup>
    </form>
  );
}
