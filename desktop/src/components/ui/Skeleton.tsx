import { cn } from "../../lib/cn";

export function Skeleton({ rows = 6, className }: { rows?: number; className?: string }) {
  return (
    <div className={cn("space-y-2 p-3", className)} aria-hidden="true">
      {Array.from({ length: rows }).map((_, index) => (
        <div
          key={index}
          className="h-4 animate-pulse rounded bg-surface-2"
          style={{ width: `${55 + ((index * 17) % 40)}%` }}
        />
      ))}
    </div>
  );
}
