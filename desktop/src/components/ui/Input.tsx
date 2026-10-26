import type { ComponentProps } from "react";
import { cn } from "../../lib/cn";

export function Input({ className, ...props }: ComponentProps<"input">) {
  return (
    <input
      className={cn(
        "h-8 min-w-0 rounded-md border border-line bg-bg px-2.5 font-mono text-[12px] text-ink outline-none transition placeholder:font-sans placeholder:text-muted focus:border-accent focus:ring-2 focus:ring-accent/20",
        className,
      )}
      {...props}
    />
  );
}
