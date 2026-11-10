import { Command, GitBranch } from "lucide-react";
import { Kbd } from "../../components/ui/Kbd";
import type { StatusResult } from "../../lib/gitcontrol";
import { hasWorktreeChange, isStaged } from "../../lib/format";

export function StatusBar({
  status,
  repo,
  onOpenPalette,
}: {
  status: StatusResult | null;
  repo: string;
  onOpenPalette: () => void;
}) {
  const files = status?.files.filter((file) => !file.ignored) ?? [];
  const staged = files.filter(isStaged).length;
  const changes = files.filter(
    (file) => hasWorktreeChange(file) || file.untracked || file.unmerged,
  ).length;

  return (
    <footer className="flex h-6 shrink-0 items-center gap-3 border-t border-line bg-surface px-3 text-[11px] text-muted">
      <span className="flex items-center gap-1.5 text-success">
        <GitBranch size={11} />
        {status?.branch ?? "—"}
      </span>
      <span>{staged} staged</span>
      <span>{changes} alterações</span>
      <span className="min-w-0 truncate font-mono text-muted/70">{repo}</span>

      <button
        type="button"
        onClick={onOpenPalette}
        className="ml-auto flex items-center gap-1 rounded px-1.5 py-0.5 transition hover:bg-surface-2 hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
      >
        <Command size={11} />
        <span>atalhos</span>
        <Kbd>Ctrl K</Kbd>
      </button>
      <span className="hidden items-center gap-1 sm:flex">
        buscar <Kbd>/</Kbd>
      </span>
    </footer>
  );
}
