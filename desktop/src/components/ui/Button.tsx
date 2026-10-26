import type { ButtonHTMLAttributes } from "react";
import { cn } from "../../lib/cn";

type Variant = "primary" | "secondary" | "ghost" | "danger";
type Size = "sm" | "md";

const variants: Record<Variant, string> = {
  primary:
    "border-transparent bg-accent text-black hover:bg-accent/90 active:bg-accent/80",
  secondary:
    "border-line bg-surface-2 text-ink hover:border-accent/60 hover:text-accent",
  ghost:
    "border-transparent bg-transparent text-muted hover:bg-surface-2 hover:text-ink",
  danger:
    "border-danger/40 bg-danger/10 text-danger hover:border-danger/70 hover:bg-danger/20",
};

const sizes: Record<Size, string> = {
  sm: "h-7 gap-1.5 px-2.5 text-[12px]",
  md: "h-8 gap-2 px-3 text-[13px]",
};

export function Button({
  variant = "secondary",
  size = "sm",
  className,
  type = "button",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: Variant;
  size?: Size;
}) {
  return (
    <button
      type={type}
      className={cn(
        "inline-flex items-center justify-center rounded-md border font-medium transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 disabled:pointer-events-none disabled:opacity-40",
        variants[variant],
        sizes[size],
        className,
      )}
      {...props}
    />
  );
}
