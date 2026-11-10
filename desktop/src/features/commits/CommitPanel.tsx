import { Copy, GitCommitHorizontal } from "lucide-react";
import { Button } from "../../components/ui/Button";
import { EmptyState } from "../../components/ui/EmptyState";
import { IconButton } from "../../components/ui/IconButton";
import { Panel } from "../../components/ui/Panel";
import { Skeleton } from "../../components/ui/Skeleton";
import { cn } from "../../lib/cn";
import type { Commit } from "../../lib/gitcontrol";
import {
  formatAbsoluteTime,
  formatRelativeTime,
  initials,
  shortHash,
} from "../../lib/format";

export function CommitPanel({
  commits,
  graph,
  selected,
  onSelect,
  onCherryPick,
  onCopyHash,
  busy,
  loading,
  query,
}: {
  commits: Commit[];
  graph: Record<string, string>;
  selected: Commit | null;
  onSelect: (commit: Commit) => void;
  onCherryPick: () => void;
  onCopyHash: (hash: string) => void;
  busy: boolean;
  loading: boolean;
  query: string;
}) {
  return (
    <Panel
      title="Commits"
      count={commits.length}
      bodyClassName="overflow-auto py-1"
      actions={
        selected && (
          <Button onClick={onCherryPick} disabled={busy}>
            <GitCommitHorizontal size={13} />
            Cherry-pick
          </Button>
        )
      }
    >
      {loading ? (
        <Skeleton rows={8} />
      ) : commits.length === 0 ? (
        <EmptyState
          icon={<GitCommitHorizontal size={22} />}
          title={query ? "Nenhum commit encontrado" : "Nenhum commit"}
          description={
            query
              ? "Tente outro termo de busca."
              : "Os commits deste repositório aparecerão aqui."
          }
        />
      ) : (
        commits.map((commit) => {
          const isSelected = selected?.hash === commit.hash;
          return (
            <div
              key={commit.hash}
              className={cn(
                "group flex items-center border-l-2 pr-2 transition",
                isSelected
                  ? "border-accent bg-accent/10"
                  : "border-transparent hover:bg-surface-2",
              )}
            >
              <button
                type="button"
                aria-pressed={isSelected}
                onClick={() => onSelect(commit)}
                className="flex min-w-0 flex-1 items-start gap-2 px-3 py-1.5 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/60"
              >
                <span className="mt-0.5 shrink-0 font-mono text-[11px] text-accent/80">
                  {graph[commit.hash] ?? "*"}
                </span>
                <span className="mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-surface-2 text-[9px] font-semibold text-muted">
                  {initials(commit.author)}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="flex items-baseline gap-2">
                    <span className="shrink-0 font-mono text-[11px] text-muted">
                      {shortHash(commit.hash)}
                    </span>
                    <span className="truncate text-[13px] text-ink/90">
                      {commit.subject}
                    </span>
                  </span>
                  <span className="mt-0.5 block truncate text-[10.5px] text-muted">
                    {commit.author} ·{" "}
                    <span title={formatAbsoluteTime(commit.time)}>
                      {formatRelativeTime(commit.time)}
                    </span>
                  </span>
                </span>
              </button>
              <IconButton
                label={`Copiar hash ${shortHash(commit.hash)}`}
                icon={<Copy size={13} />}
                className="opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                onClick={() => onCopyHash(commit.hash)}
              />
            </div>
          );
        })
      )}
    </Panel>
  );
}
