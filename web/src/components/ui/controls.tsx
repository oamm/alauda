import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import * as SwitchPrimitive from "@radix-ui/react-switch";
import * as Popover from "@radix-ui/react-popover";
import { Check, ChevronDown } from "lucide-react";
import { forwardRef, useId, type SelectHTMLAttributes } from "react";
import { cn } from "../../lib/utils";
import { useFieldControl } from "./form";

export function Checkbox({
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof CheckboxPrimitive.Root>) {
  const field = useFieldControl(props);
  return (
    <CheckboxPrimitive.Root
      {...props}
      {...field}
      className={cn("alauda-checkbox", className)}
    >
      <CheckboxPrimitive.Indicator>
        <Check size={12} aria-hidden="true" />
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}

export function Switch({
  className,
  ...props
}: React.ComponentPropsWithoutRef<typeof SwitchPrimitive.Root>) {
  const field = useFieldControl(props);
  return (
    <SwitchPrimitive.Root
      {...props}
      {...field}
      className={cn("alauda-switch", className)}
    >
      <SwitchPrimitive.Thumb className="alauda-switch-thumb" />
    </SwitchPrimitive.Root>
  );
}

export const Select = forwardRef<
  HTMLSelectElement,
  SelectHTMLAttributes<HTMLSelectElement>
>(({ className, ...props }, ref) => {
  const field = useFieldControl(props);
  return (
    <div className="alauda-select-wrap">
      <select
        {...props}
        {...field}
        ref={ref}
        className={cn("alauda-control alauda-select", className)}
      />
      <ChevronDown size={14} aria-hidden="true" />
    </div>
  );
});
Select.displayName = "Select";

export function RadioGroup({
  options,
  value,
  onValueChange,
  name,
  disabled,
  ...props
}: {
  options: Array<{ value: string; label: string }>;
  value: string;
  onValueChange: (value: string) => void;
  name?: string;
  disabled?: boolean;
  id?: string;
  "aria-label"?: string;
}) {
  const generatedName = useId();
  const field = useFieldControl(props);
  return (
    <div
      {...props}
      {...field}
      role="radiogroup"
      className="alauda-choice-group"
    >
      {options.map((option) => (
        <label key={option.value} className="alauda-choice">
          <input
            className="alauda-radio"
            type="radio"
            name={name ?? generatedName}
            value={option.value}
            checked={value === option.value}
            disabled={disabled}
            onChange={() => onValueChange(option.value)}
          />
          {option.label}
        </label>
      ))}
    </div>
  );
}

export function MultiSelect({
  options,
  value,
  onValueChange,
  placeholder = "Select options",
  disabled,
  ...props
}: {
  options: Array<{ value: string; label: string }>;
  value: string[];
  onValueChange: (value: string[]) => void;
  placeholder?: string;
  disabled?: boolean;
  id?: string;
  "aria-label"?: string;
}) {
  const field = useFieldControl(props);
  const optionsId = useId();
  return (
    <Popover.Root>
      <Popover.Trigger
        {...props}
        {...field}
        disabled={disabled}
        type="button"
        className="alauda-control alauda-multiselect"
      >
        <span>
          {value.length
            ? options
                .filter((option) => value.includes(option.value))
                .map((option) => option.label)
                .join(", ")
            : placeholder}
        </span>
        <ChevronDown size={14} aria-hidden="true" />
      </Popover.Trigger>
      <Popover.Portal>
        <Popover.Content
          align="start"
          sideOffset={4}
          className="alauda-options"
        >
          <div role="group" aria-label={props["aria-label"] ?? "Options"}>
            {options.map((option, index) => (
              <label className="alauda-choice" key={option.value}>
                <Checkbox
                  id={`${optionsId}-${index}`}
                  aria-labelledby={`${optionsId}-${index}-label`}
                  checked={value.includes(option.value)}
                  onCheckedChange={(checked) =>
                    onValueChange(
                      checked === true
                        ? [...value, option.value]
                        : value.filter((item) => item !== option.value),
                    )
                  }
                />
                <span id={`${optionsId}-${index}-label`}>{option.label}</span>
              </label>
            ))}
          </div>
        </Popover.Content>
      </Popover.Portal>
    </Popover.Root>
  );
}
