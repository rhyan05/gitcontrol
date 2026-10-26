import type { ReactNode } from "react";
import { cn } from "../../lib/cn";

export function Panel({
  title,
  count,
  actions,
  children,
  footer,
  className,
  bodyClassName,
}: {
  title: string;
  count?: number;
  actions?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
  className?: string;
  bodyClassName?: string;
}) {
  return (
    <section className={cn("flex min-h-0 min-w-0 flex-col", className)}>
      <header className="flex h-9 shrink-0 items-center gap-2 border-b border-line px-3">
        <h2 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted">
          {title}
        </h2>
        {count !== undefined && (
          <span className="text-[11px] tabular-nums text-muted">{count}</span>
        )}
        <div className="ml-auto flex items-center gap-1">{actions}</div>
      </header>
      <div className={cn("min-h-0 flex-1", bodyClassName)}>{children}</div>
      {footer}
    </section>
  );
}
