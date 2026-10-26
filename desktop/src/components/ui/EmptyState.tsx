import type { ReactNode } from "react";
import { Button } from "./Button";

export function EmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 px-6 py-8 text-center">
      {icon && <div className="text-muted/70">{icon}</div>}
      <p className="text-[13px] font-medium text-ink">{title}</p>
      {description && (
        <p className="max-w-[26ch] text-[12px] leading-relaxed text-muted">
          {description}
        </p>
      )}
      {action && (
        <Button className="mt-1" onClick={action.onClick}>
          {action.label}
        </Button>
      )}
    </div>
  );
}
