import { Check, GitBranch, GitMerge } from "lucide-react";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { EmptyState } from "../../components/ui/EmptyState";
import { Panel } from "../../components/ui/Panel";
import { Skeleton } from "../../components/ui/Skeleton";
import { cn } from "../../lib/cn";
import type { Branch } from "../../lib/gitcontrol";

function branchKey(branch: Branch): string {
  return `${branch.remote ? "r" : "l"}:${branch.name}`;
}

export function BranchPanel({
  branches,
  selected,
  onSelect,
  onCheckout,
  onMerge,
  busy,
  loading,
  query,
}: {
  branches: Branch[];
  selected: Branch | null;
  onSelect: (branch: Branch) => void;
  onCheckout: () => void;
  onMerge: () => void;
  busy: boolean;
  loading: boolean;
  query: string;
}) {
  const locals = branches.filter((branch) => !branch.remote);
  const remotes = branches.filter((branch) => branch.remote);
  const selectedKey = selected ? branchKey(selected) : "";

  return (
    <Panel
      title="Branches"
      count={branches.length}
      bodyClassName="overflow-auto py-1"
    >
      {selected && (
        <div className="flex gap-2 border-b border-line bg-surface-2/60 px-3 py-1.5">
          <Button
            className="flex-1"
            onClick={onCheckout}
            disabled={busy || selected.head}
            title={
              selected.head
                ? "A branch atual já está em checkout"
                : `Checkout ${selected.name}`
            }
          >
            <Check size={13} />
            Checkout
          </Button>
          <Button
            className="flex-1"
            onClick={onMerge}
            disabled={busy || selected.head}
            title={`Merge ${selected.name} na branch atual`}
          >
            <GitMerge size={13} />
            Merge
          </Button>
        </div>
      )}
      {loading ? (
        <Skeleton rows={7} />
      ) : branches.length === 0 ? (
        <EmptyState
          icon={<GitBranch size={22} />}
          title={query ? "Nenhuma branch encontrada" : "Nenhuma branch"}
          description={
            query
              ? "Ajuste o filtro para ver outras branches."
              : "Este repositório ainda não tem branches."
          }
        />
      ) : (
        <>
          <BranchGroup
            label="Locais"
            branches={locals}
            selectedKey={selectedKey}
            onSelect={onSelect}
          />
          <BranchGroup
            label="Remotas"
            branches={remotes}
            selectedKey={selectedKey}
            onSelect={onSelect}
          />
        </>
      )}
    </Panel>
  );
}

function BranchGroup({
  label,
  branches,
  selectedKey,
  onSelect,
}: {
  label: string;
  branches: Branch[];
  selectedKey: string;
  onSelect: (branch: Branch) => void;
}) {
  if (branches.length === 0) return null;

  return (
    <div className="pb-1">
      <p className="px-3 pb-1 pt-2 text-[10px] font-semibold uppercase tracking-[0.16em] text-muted/70">
        {label}
      </p>
      {branches.map((branch) => {
        const selected = branchKey(branch) === selectedKey;
        return (
          <button
            key={branchKey(branch)}
            type="button"
            aria-pressed={selected}
            onClick={() => onSelect(branch)}
            className={cn(
              "flex w-full items-center gap-2 border-l-2 px-3 py-1.5 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-accent/60",
              selected
                ? "border-accent bg-accent/10"
                : "border-transparent hover:bg-surface-2",
            )}
          >
            <span
              className={cn(
                "h-1.5 w-1.5 shrink-0 rounded-full border",
                branch.head
                  ? "border-success bg-success"
                  : "border-muted/60",
              )}
            />
            <span className="min-w-0 flex-1 truncate font-mono text-[12.5px] text-ink/90">
              {branch.name}
            </span>
            {branch.head && <Badge tone="success">HEAD</Badge>}
            {branch.remote && <Badge tone="accent">remota</Badge>}
            {!branch.remote && branch.upstream && (
              <span className="hidden shrink-0 truncate font-mono text-[10px] text-muted lg:inline">
                → {branch.upstream}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
