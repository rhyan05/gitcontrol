import { FolderOpen, RefreshCw, Search } from "lucide-react";
import type { RefObject } from "react";
import { Logo } from "../../components/Logo";
import { Badge } from "../../components/ui/Badge";
import { IconButton } from "../../components/ui/IconButton";
import { Input } from "../../components/ui/Input";
import { Kbd } from "../../components/ui/Kbd";
import type { StatusResult } from "../../lib/gg";
import { cn } from "../../lib/cn";

export function AppHeader({
  repo,
  onRepoChange,
  recentRepos,
  onPickRepo,
  status,
  loading,
  refreshing,
  onRefresh,
  search,
  onSearchChange,
  searchRef,
}: {
  repo: string;
  onRepoChange: (repo: string) => void;
  recentRepos: string[];
  onPickRepo: () => void;
  status: StatusResult | null;
  loading: boolean;
  refreshing: boolean;
  onRefresh: () => void;
  search: string;
  onSearchChange: (value: string) => void;
  searchRef: RefObject<HTMLInputElement | null>;
}) {
  return (
    <header className="flex h-12 shrink-0 items-center gap-3 border-b border-line bg-surface px-3">
      <div className="flex shrink-0 items-center gap-2">
        <Logo className="h-8 w-8" />
        <div className="leading-none">
          <p className="text-[14px] font-semibold tracking-tight text-ink">gg</p>
          <p className="mt-0.5 text-[11px] text-muted">gerenciador git</p>
        </div>
      </div>

      <div className="mx-1 h-6 w-px bg-line" />

      <div className="flex min-w-0 flex-1 items-center gap-2">
        <Input
          value={repo}
          onChange={(event) => onRepoChange(event.currentTarget.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") onRefresh();
          }}
          spellCheck={false}
          aria-label="Caminho do repositório"
          list="gg-recent-repos"
          className="w-full max-w-[380px] bg-surface-2"
        />
        <datalist id="gg-recent-repos">
          {recentRepos.map((recent) => (
            <option key={recent} value={recent} />
          ))}
        </datalist>
        <IconButton
          label="Escolher pasta do repositório"
          icon={<FolderOpen size={15} />}
          onClick={onPickRepo}
        />
        <IconButton
          label="Atualizar repositório"
          icon={
            <RefreshCw
              size={15}
              className={cn((loading || refreshing) && "animate-spin")}
            />
          }
          onClick={onRefresh}
          disabled={loading}
        />
      </div>

      {status && (
        <div className="flex shrink-0 items-center gap-2">
          <Badge tone="success">{status.branch || "HEAD"}</Badge>
          {status.upstream && (
            <span className="hidden font-mono text-[11px] text-muted xl:inline">
              → {status.upstream}
            </span>
          )}
          {(status.ahead ?? 0) > 0 && (
            <Badge tone="accent">↑{status.ahead}</Badge>
          )}
          {(status.behind ?? 0) > 0 && (
            <Badge tone="warning">↓{status.behind}</Badge>
          )}
        </div>
      )}

      <div className="relative hidden w-[220px] shrink-0 md:block">
        <Search
          size={13}
          className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-muted"
        />
        <Input
          ref={searchRef}
          value={search}
          onChange={(event) => onSearchChange(event.currentTarget.value)}
          placeholder="Buscar"
          aria-label="Filtrar branches, commits e arquivos"
          className="w-full pl-7 pr-8"
        />
        <span className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2">
          <Kbd>/</Kbd>
        </span>
      </div>
    </header>
  );
}
