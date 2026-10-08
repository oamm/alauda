import { object, string, boolean, type infer as Infer } from "zod";
import type { Environment } from "../api";

export const environmentSchema = object({
  name: string().trim().min(1, "Name is required."),
  key: string().trim().min(1, "Key is required."),
  tier: string().trim(),
  description: string(),
  enabled: boolean(),
});
export type EnvironmentValue = Infer<typeof environmentSchema>;
export function environmentDefaults(
  environment?: Environment,
): EnvironmentValue {
  return {
    name: environment?.name || "",
    key: environment?.key || "",
    tier: environment?.tier || "",
    description: environment?.description || "",
    enabled: environment?.enabled ?? true,
  };
}
export function environmentTier(tier?: string) {
  const value = tier?.trim();
  return !value ? "Untiered" : /^\d+$/.test(value) ? `Tier ${value}` : value;
}
