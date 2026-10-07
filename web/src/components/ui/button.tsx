import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { forwardRef, type ButtonHTMLAttributes } from "react";

import { cn } from "../../lib/utils";

const buttonVariants = cva(
  "inline-flex min-h-9 items-center justify-center gap-2 whitespace-nowrap rounded-[var(--radius-control)] border text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--focus-ring)] disabled:pointer-events-none disabled:opacity-50",
  {
    variants: {
      variant: {
        primary:
          "border-transparent bg-[var(--accent)] text-white hover:bg-[var(--accent-strong)]",
        secondary:
          "border-[var(--border-strong)] bg-[var(--surface)] text-[var(--text)] hover:bg-[var(--surface-muted)]",
        ghost:
          "border-transparent bg-transparent text-[var(--text-muted)] hover:bg-[var(--surface-muted)] hover:text-[var(--text)]",
        danger:
          "border-[var(--danger-border)] bg-[var(--danger-bg)] text-[var(--danger-text)] hover:bg-[var(--danger-bg-strong)]",
        link: "min-h-0 border-0 bg-transparent px-0 text-[var(--accent-strong)] underline-offset-4 hover:underline",
      },
      size: {
        sm: "h-8 px-2.5 text-xs",
        md: "h-9 px-3.5",
      },
    },
    defaultVariants: { variant: "secondary", size: "md" },
  },
);

export interface ButtonProps
  extends
    ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      asChild = false,
      className,
      loading = false,
      children,
      disabled,
      variant = "secondary",
      size = "md",
      type = "button",
      ...props
    },
    ref,
  ) => {
    const Comp = asChild ? Slot : "button";
    return (
      <Comp
        ref={ref}
        aria-busy={loading || undefined}
        className={cn(
          buttonVariants({ variant, size }),
          "alauda-button",
          className,
        )}
        data-variant={variant}
        data-size={size}
        type={type}
        disabled={disabled || loading}
        {...props}
      >
        {loading ? "Working..." : children}
      </Comp>
    );
  },
);
Button.displayName = "Button";

export function IconButton({
  label,
  className,
  size = "sm",
  ...props
}: ButtonProps & { label: string }) {
  return (
    <Button
      {...props}
      aria-label={label}
      title={props.title ?? label}
      className={cn("alauda-icon-button", className)}
      size={size}
    />
  );
}

export { buttonVariants };
