import { Trash2 } from "lucide-react";
import { FormField, IconButton, Input, Select, Switch } from "./ui";
import {
  endpointProtocols,
  supportsEndpointPath,
  endpointPath,
  validateEndpoint,
  type EndpointFormValue,
  type EndpointErrors,
} from "../lib/endpoint-form";

export function EndpointEditor({
  value,
  onChange,
  existingNames = [],
  errors = {},
  labelPrefix = "",
  autoFocus = false,
  disabled = false,
  primaryLocked = false,
  onRemove,
}: {
  value: EndpointFormValue;
  onChange: (value: EndpointFormValue) => void;
  existingNames?: string[];
  errors?: EndpointErrors;
  labelPrefix?: string;
  autoFocus?: boolean;
  disabled?: boolean;
  primaryLocked?: boolean;
  onRemove?: () => void;
}) {
  const validation = { ...validateEndpoint(value, existingNames), ...errors };
  const label = (field: string) =>
    labelPrefix ? `${labelPrefix} ${field.toLowerCase()}` : field;
  const change = (patch: Partial<EndpointFormValue>) =>
    onChange({ ...value, ...patch });
  return (
    <div className="grid min-w-0 gap-3 sm:grid-cols-2">
      <FormField
        label="Name"
        hint="Unique within this instance."
        error={validation.name}
      >
        <Input
          required
          autoFocus={autoFocus}
          disabled={disabled}
          aria-label={label("Name")}
          value={value.name}
          onChange={(event) => change({ name: event.target.value })}
        />
      </FormField>
      <FormField label="Protocol" error={validation.protocol}>
        <Select
          disabled={disabled}
          aria-label={label("Protocol")}
          value={value.protocol}
          onChange={(event) =>
            change({
              protocol: event.target.value,
              path: endpointPath(event.target.value, value.path),
            })
          }
        >
          {endpointProtocols.map(([protocol, text]) => (
            <option key={protocol} value={protocol}>
              {text}
            </option>
          ))}
        </Select>
      </FormField>
      <FormField
        label="Port"
        error={validation.port}
        className={
          supportsEndpointPath(value.protocol) ? undefined : "sm:col-span-2"
        }
      >
        <Input
          required
          disabled={disabled}
          aria-label={label("Port")}
          type="number"
          min={1}
          max={65535}
          step={1}
          value={value.port || ""}
          onChange={(event) => change({ port: Number(event.target.value) })}
        />
      </FormField>
      {supportsEndpointPath(value.protocol) ? (
        <FormField label="Path" error={validation.path}>
          <Input
            disabled={disabled}
            aria-label={label("Path")}
            value={value.path}
            onChange={(event) => change({ path: event.target.value })}
          />
        </FormField>
      ) : null}
      <FormField
        label="Primary"
        error={validation.primary}
        hint={
          primaryLocked
            ? "A single registration endpoint is automatically primary."
            : undefined
        }
      >
        <Switch
          disabled={disabled || primaryLocked}
          aria-label={label("Primary")}
          checked={value.primary}
          onCheckedChange={(primary) => change({ primary })}
        />
      </FormField>
      <FormField label="Enabled" error={validation.enabled}>
        <Switch
          disabled={disabled}
          aria-label={label("Enabled")}
          checked={value.enabled}
          onCheckedChange={(enabled) => change({ enabled })}
        />
      </FormField>
      {onRemove ? (
        <div className="flex justify-end sm:col-span-2">
          <IconButton
            type="button"
            variant="ghost"
            disabled={disabled}
            label="Remove endpoint"
            onClick={onRemove}
          >
            <Trash2 size={14} />
          </IconButton>
        </div>
      ) : null}
    </div>
  );
}
