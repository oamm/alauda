import {
  createContext,
  forwardRef,
  useContext,
  useId,
  useState,
  type InputHTMLAttributes,
  type TextareaHTMLAttributes,
  type ReactNode,
} from "react";
import { Eye, EyeOff, Search } from "lucide-react";
import { cn } from "../../lib/utils";
import { IconButton } from "./button";

type FieldContextValue = {
  id: string;
  description?: string;
  invalid?: boolean;
};
const FieldContext = createContext<FieldContextValue | null>(null);

export function useFieldControl(props: {
  id?: string;
  "aria-describedby"?: string;
  "aria-label"?: string;
  "aria-labelledby"?: string;
  "aria-invalid"?: boolean | "true" | "false" | "grammar" | "spelling";
}) {
  const field = useContext(FieldContext);
  return {
    id: props.id ?? field?.id,
    "aria-labelledby":
      props["aria-labelledby"] ??
      (props["aria-label"]
        ? undefined
        : field
          ? `${field.id}-label`
          : undefined),
    "aria-describedby":
      [props["aria-describedby"], field?.description]
        .filter(Boolean)
        .join(" ") || undefined,
    "aria-invalid": props["aria-invalid"] ?? field?.invalid,
  };
}

export const Input = forwardRef<
  HTMLInputElement,
  InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => {
  const field = useFieldControl(props);
  return (
    <input
      {...props}
      {...field}
      ref={ref}
      className={cn("alauda-control", className)}
    />
  );
});
Input.displayName = "Input";

export const Textarea = forwardRef<
  HTMLTextAreaElement,
  TextareaHTMLAttributes<HTMLTextAreaElement>
>(({ className, ...props }, ref) => {
  const field = useFieldControl(props);
  return (
    <textarea
      {...props}
      {...field}
      ref={ref}
      className={cn("alauda-control alauda-textarea", className)}
    />
  );
});
Textarea.displayName = "Textarea";

export const SearchInput = forwardRef<
  HTMLInputElement,
  InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => (
  <div className="alauda-control-adornment">
    <Search size={15} aria-hidden="true" />
    <Input
      {...props}
      ref={ref}
      type="search"
      className={cn("alauda-search", className)}
    />
  </div>
));
SearchInput.displayName = "SearchInput";

export const PasswordInput = forwardRef<
  HTMLInputElement,
  InputHTMLAttributes<HTMLInputElement>
>(({ className, ...props }, ref) => {
  const [visible, setVisible] = useState(false);
  return (
    <div className="alauda-control-adornment">
      <Input
        {...props}
        ref={ref}
        type={visible ? "text" : "password"}
        className={cn("alauda-password", className)}
      />
      <IconButton
        type="button"
        variant="ghost"
        label={visible ? "Hide password" : "Show password"}
        onClick={() => setVisible(!visible)}
      >
        {visible ? <EyeOff size={15} /> : <Eye size={15} />}
      </IconButton>
    </div>
  );
});
PasswordInput.displayName = "PasswordInput";

export function FieldDescription({
  children,
  id,
}: {
  children: ReactNode;
  id?: string;
}) {
  return (
    <span id={id} className="alauda-field-description">
      {children}
    </span>
  );
}
export function FieldError({
  children,
  id,
}: {
  children: ReactNode;
  id?: string;
}) {
  return (
    <span id={id} role="alert" className="alauda-field-error">
      {children}
    </span>
  );
}

export function FormField({
  label,
  hint,
  error,
  children,
  id,
  className,
}: {
  label: string;
  hint?: string;
  error?: string;
  children: ReactNode;
  id?: string;
  className?: string;
}) {
  const generatedId = useId();
  const controlId = id ?? generatedId;
  const descriptionId = `${controlId}-description`;
  return (
    <FieldContext.Provider
      value={{
        id: controlId,
        description: hint || error ? descriptionId : undefined,
        invalid: error ? true : undefined,
      }}
    >
      <label className={cn("alauda-field", className)}>
        <span id={`${controlId}-label`}>{label}</span>
        {children}
        {error ? (
          <FieldError id={descriptionId}>{error}</FieldError>
        ) : hint ? (
          <FieldDescription id={descriptionId}>{hint}</FieldDescription>
        ) : null}
      </label>
    </FieldContext.Provider>
  );
}

export function FormSection({
  title,
  description,
  children,
  className,
}: {
  title: string;
  description?: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <fieldset className={cn("alauda-form-section", className)}>
      <legend>{title}</legend>
      {description ? <FieldDescription>{description}</FieldDescription> : null}
      <div className="alauda-form-fields">{children}</div>
    </fieldset>
  );
}
