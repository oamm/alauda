import { useState } from "react";
import { createEnvironment, updateEnvironment, type Environment } from "../api";
import {
  environmentDefaults,
  environmentSchema,
  type EnvironmentValue,
} from "../lib/environment-form";
import {
  ActionGroup,
  Alert,
  Button,
  FormField,
  Input,
  Textarea,
  Switch,
} from "./ui";

export function EnvironmentForm({
  environment,
  environments,
  onSaved,
  onCancel,
}: {
  environment?: Environment;
  environments: Environment[];
  onSaved: (environment: Environment) => void | Promise<void>;
  onCancel: () => void;
}) {
  const [value, setValue] = useState(() => environmentDefaults(environment));
  const [errors, setErrors] = useState<
    Partial<Record<keyof EnvironmentValue, string>>
  >({});
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  function change(patch: Partial<EnvironmentValue>) {
    setValue((current) => ({ ...current, ...patch }));
    setErrors({});
    setError("");
  }
  async function submit(event: React.FormEvent) {
    event.preventDefault();
    if (saving || saved) return;
    const validated = environmentSchema.safeParse(value);
    const next: typeof errors = {};
    if (!validated.success)
      for (const issue of validated.error.issues)
        next[issue.path[0] as keyof EnvironmentValue] = issue.message;
    if (!environment && environments.some((e) => e.key === value.key.trim()))
      next.key = "An environment with this key already exists.";
    if (Object.keys(next).length) {
      setErrors(next);
      return;
    }
    setSaving(true);
    setError("");
    try {
      const fields = validated.success ? validated.data : value;
      const result = environment
        ? await updateEnvironment({
            id: environment.id,
            name: fields.name,
            tier: fields.tier,
            description: fields.description,
            enabled: fields.enabled,
            tags: environment.tags || {},
          })
        : await createEnvironment({
            name: fields.name,
            key: fields.key,
            tier: fields.tier,
            description: fields.description,
          });
      setSaved(true);
      try {
        await onSaved(result);
      } catch {
        setError(
          "Environment saved, but the view could not refresh. Close the form and refresh the page.",
        );
      }
    } catch (e) {
      let message =
        e instanceof Error ? e.message : "Could not save the environment.";
      try {
        const parsed = JSON.parse(message);
        message = parsed.message || parsed.error || message;
      } catch {}
      if (message.includes("UNIQUE") || message.includes("already_exists"))
        setErrors({ key: "An environment with this key already exists." });
      else setError(message);
    } finally {
      setSaving(false);
    }
  }
  return (
    <form className="grid min-w-0 gap-3" onSubmit={submit} noValidate>
      {error ? (
        <Alert
          tone="danger"
          title={saved ? "Environment saved" : "Could not save environment"}
        >
          {error}
        </Alert>
      ) : null}
      <div className="grid min-w-0 gap-3 sm:grid-cols-2">
        <FormField label="Name" error={errors.name}>
          <Input
            value={value.name}
            disabled={saving || saved}
            onChange={(e) => change({ name: e.target.value })}
          />
        </FormField>
        <FormField
          label="Key"
          error={errors.key}
          hint={
            environment
              ? "Stable identifier; cannot be changed."
              : "Unique identifier for this environment."
          }
        >
          <Input
            value={value.key}
            readOnly={!!environment}
            disabled={saving || saved}
            onChange={(e) => change({ key: e.target.value })}
          />
        </FormField>
        <FormField
          label="Tier"
          hint="Optional classification or level."
          error={errors.tier}
        >
          <Input
            value={value.tier}
            disabled={saving || saved}
            onChange={(e) => change({ tier: e.target.value })}
          />
        </FormField>
        {environment ? (
          <FormField label="Enabled">
            <Switch
              checked={value.enabled}
              disabled={saving || saved}
              onCheckedChange={(enabled) => change({ enabled })}
            />
          </FormField>
        ) : (
          <p className="self-center text-xs text-[var(--text-muted)]">
            New environments are enabled.
          </p>
        )}
        <FormField label="Description" className="sm:col-span-2">
          <Textarea
            value={value.description}
            disabled={saving || saved}
            onChange={(e) => change({ description: e.target.value })}
          />
        </FormField>
      </div>
      <ActionGroup>
        <Button variant="ghost" disabled={saving} onClick={onCancel}>
          Cancel
        </Button>
        <Button
          variant="primary"
          type="submit"
          disabled={saving || saved}
          loading={saving}
        >
          {environment ? "Save changes" : "Create environment"}
        </Button>
      </ActionGroup>
    </form>
  );
}
