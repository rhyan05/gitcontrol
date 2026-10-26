import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { open as openDialog } from "@tauri-apps/plugin-dialog";
import {
  Check,
  GitCommitHorizontal,
  GitMerge,
  Minus,
  Plus,
  RefreshCw,
  Search,
} from "lucide-react";
import { Logo } from "./components/Logo";
import { AppHeader } from "./features/repo/AppHeader";
import { StatusBar } from "./features/repo/StatusBar";
import { BranchPanel } from "./features/branches/BranchPanel";
import { CommitPanel } from "./features/commits/CommitPanel";
import { StatusPanel } from "./features/status/StatusPanel";
import { DiffPanel } from "./features/diff/DiffPanel";
import {
  CommandPalette,
  type Command,
} from "./features/command/CommandPalette";
import { ConfirmDialog } from "./components/ui/ConfirmDialog";
import { useToast } from "./components/ui/toast-context";
import { useRepo } from "./hooks/useRepo";
import "./App.css";
import {
  gg,
  type Branch,
  type Commit,
  type ShowResult,
} from "./lib/gg";
import {
  filterBranches,
  filterCommits,
  filterFiles,
  groupStatus,
  visibleFiles,
} from "./lib/selectors";
import { shortHash } from "./lib/format";
import { cn } from "./lib/cn";

type Pending = {
  action: "checkout" | "merge" | "cherry-pick";
  target: string;
  title: string;
  description: string;
  confirmLabel: string;
};

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function loadRecentRepos(): string[] {
  try {
    const parsed: unknown = JSON.parse(
      localStorage.getItem("gg.recentRepos") ?? "[]",
    );
    return Array.isArray(parsed)
      ? parsed.filter((value): value is string => typeof value === "string")
      : [];
  } catch {
    return [];
  }
}

function initialDiffHeight(): number {
  const stored = Number(localStorage.getItem("gg.diffHeight"));
  return Number.isFinite(stored) && stored >= 120 ? stored : 240;
}

function App() {
  const toast = useToast();
  const [repo, setRepo] = useState(
    () => localStorage.getItem("gg.repo") ?? "..",
  );
  const [recentRepos, setRecentRepos] = useState(loadRecentRepos);
  const [search, setSearch] = useState("");
  const [selectedBranch, setSelectedBranch] = useState<Branch | null>(null);
  const [selectedCommit, setSelectedCommit] = useState<Commit | null>(null);
  const [diff, setDiff] = useState("");
  const [diffLoading, setDiffLoading] = useState(false);
  const [diffError, setDiffError] = useState("");
  const [copiedDiff, setCopiedDiff] = useState(false);
  const [pending, setPending] = useState<Pending | null>(null);
  const [commitMessage, setCommitMessage] = useState("");
  const [busy, setBusy] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [manualRefreshing, setManualRefreshing] = useState(false);
  const [diffHeight, setDiffHeight] = useState(initialDiffHeight);
  const searchRef = useRef<HTMLInputElement>(null);
  const diffRequest = useRef(0);

  const { status, branches, commits, loading, error, refresh } = useRepo(repo);

  const rememberRepo = useCallback((value: string) => {
    const trimmed = value.trim();
    if (!trimmed) return;

    setRecentRepos((current) => {
      const next = [trimmed, ...current.filter((item) => item !== trimmed)].slice(
        0,
        8,
      );
      localStorage.setItem("gg.recentRepos", JSON.stringify(next));
      return next;
    });
  }, []);

  const refreshRepo = useCallback(() => {
    rememberRepo(repo);
    setManualRefreshing(true);
    void refresh(true).finally(() => setManualRefreshing(false));
  }, [rememberRepo, refresh, repo]);

  const changeRepo = useCallback((next: string) => {
    diffRequest.current++;
    setRepo(next);
    setSelectedBranch(null);
    setSelectedCommit(null);
    setDiff("");
    setDiffError("");
    setDiffLoading(false);
    setSearch("");
    setCommitMessage("");
  }, []);

  const focusSearch = useCallback(() => {
    searchRef.current?.focus();
  }, []);

  const selectCommit = useCallback(
    (commit: Commit) => {
      const request = ++diffRequest.current;
      setSelectedCommit(commit);
      setDiff("");
      setDiffError("");
      setDiffLoading(true);

      void gg<ShowResult>(repo, "show", commit.hash)
        .then((result) => {
          if (request === diffRequest.current) setDiff(result.diff);
        })
        .catch((err: unknown) => {
          if (request === diffRequest.current) {
            setDiffError(errorMessage(err));
          }
        })
        .finally(() => {
          if (request === diffRequest.current) setDiffLoading(false);
        });
    },
    [repo],
  );

  useEffect(() => {
    localStorage.setItem("gg.repo", repo);
  }, [repo]);

  useEffect(() => {
    localStorage.setItem("gg.diffHeight", String(diffHeight));
  }, [diffHeight]);

  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      const target = event.target as HTMLElement | null;
      const typing =
        target?.tagName === "INPUT" ||
        target?.tagName === "TEXTAREA" ||
        target?.isContentEditable === true;

      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setPaletteOpen(true);
        return;
      }

      if (typing) return;

      if (event.key === "/") {
        event.preventDefault();
        focusSearch();
      } else if (event.key === "r") {
        event.preventDefault();
        refreshRepo();
      }
    }

    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [focusSearch, refreshRepo]);

  const runAction = useCallback(
    async (
      args: string[],
      success: string,
      resetSelection = true,
    ): Promise<boolean> => {
      setBusy(true);
      try {
        await gg<null>(repo, ...args);
        await refresh(true);

        if (resetSelection) {
          diffRequest.current++;
          setSelectedBranch(null);
          setSelectedCommit(null);
          setDiff("");
          setDiffError("");
          setDiffLoading(false);
        }
        toast.success(success);
        return true;
      } catch (err) {
        toast.error(errorMessage(err));
        return false;
      } finally {
        setBusy(false);
      }
    },
    [repo, refresh, toast],
  );

  const stage = useCallback(
    (paths: string[]) => {
      if (paths.length === 0) return;
      const label =
        paths.length === 1 ? paths[0] : `${paths.length} arquivos`;
      void runAction(["stage", ...paths], `stage: ${label}`, false);
    },
    [runAction],
  );

  const unstage = useCallback(
    (paths: string[]) => {
      if (paths.length === 0) return;
      const label =
        paths.length === 1 ? paths[0] : `${paths.length} arquivos`;
      void runAction(["unstage", ...paths], `unstage: ${label}`, false);
    },
    [runAction],
  );

  async function commit() {
    const message = commitMessage.trim();
    if (!message) return;

    const done = await runAction(["commit", message], `commit: ${message}`);
    if (done) setCommitMessage("");
  }

  async function executePending() {
    if (!pending) return;

    const done = await runAction(
      [pending.action, pending.target],
      `${pending.action}: ${pending.target}`,
    );
    if (done) setPending(null);
  }

  async function pickRepo() {
    try {
      const picked = await openDialog({
        directory: true,
        multiple: false,
        title: "Escolher repositório",
      });
      if (typeof picked === "string") {
        rememberRepo(picked);
        changeRepo(picked);
      }
    } catch (err) {
      toast.error(errorMessage(err));
    }
  }

  async function copy(text: string, label: string) {
    try {
      await navigator.clipboard.writeText(text);
      toast.success(label);
    } catch {
      toast.error("Não foi possível copiar para a área de transferência");
    }
  }

  async function copyDiff() {
    if (!diff) return;
    await copy(diff, "Diff copiado");
    setCopiedDiff(true);
    window.setTimeout(() => setCopiedDiff(false), 1500);
  }

  function startResize(event: ReactPointerEvent<HTMLDivElement>) {
    event.preventDefault();
    const startY = event.clientY;
    const startHeight = diffHeight;

    const onMove = (moveEvent: PointerEvent) => {
      const max = window.innerHeight * 0.6;
      const next = Math.min(
        Math.max(startHeight - (moveEvent.clientY - startY), 120),
        max,
      );
      setDiffHeight(Math.round(next));
    };

    const onUp = () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp, { once: true });
  }

  function resizeBy(delta: number) {
    setDiffHeight((current) =>
      Math.min(Math.max(current + delta, 120), window.innerHeight * 0.6),
    );
  }

  const filteredBranches = filterBranches(branches, search);
  const filteredCommits = filterCommits(commits, search);
  const files = visibleFiles(status?.files ?? []);
  const filteredFiles = filterFiles(files, search);
  const groups = groupStatus(filteredFiles);
  const totalFiles = filteredFiles.length;

  const stageAllPaths = Array.from(
    new Set([
      ...groups.conflicts.map((file) => file.path),
      ...groups.changes.map((file) => file.path),
      ...groups.untracked.map((file) => file.path),
    ]),
  );
  const unstageAllPaths = groups.staged.map((file) => file.path);

  const commands: Command[] = [
    {
      id: "refresh",
      label: "Atualizar repositório",
      hint: "r",
      icon: <RefreshCw size={14} />,
      run: refreshRepo,
    },
    {
      id: "search",
      label: "Buscar branches, commits e arquivos",
      hint: "/",
      icon: <Search size={14} />,
      run: focusSearch,
    },
  ];

  if (selectedBranch && !selectedBranch.head) {
    commands.push(
      {
        id: "checkout",
        label: `Checkout ${selectedBranch.name}`,
        icon: <Check size={14} />,
        run: () =>
          setPending({
            action: "checkout",
            target: selectedBranch.name,
            title: `Checkout em ${selectedBranch.name}?`,
            description: selectedBranch.remote
              ? "Uma branch local de tracking será criada a partir da remota."
              : "A branch atual do repositório será trocada.",
            confirmLabel: "Fazer checkout",
          }),
      },
      {
        id: "merge",
        label: `Merge ${selectedBranch.name} em ${status?.branch ?? "HEAD"}`,
        icon: <GitMerge size={14} />,
        run: () =>
          setPending({
            action: "merge",
            target: selectedBranch.name,
            title: `Merge de ${selectedBranch.name}?`,
            description: `A branch será incorporada em ${status?.branch ?? "HEAD"}. Conflitos precisam ser resolvidos no terminal.`,
            confirmLabel: "Fazer merge",
          }),
      },
    );
  }

  if (selectedCommit) {
    commands.push({
      id: "cherry-pick",
      label: `Cherry-pick ${shortHash(selectedCommit.hash)}`,
      icon: <GitCommitHorizontal size={14} />,
      run: () =>
        setPending({
          action: "cherry-pick",
          target: selectedCommit.hash,
          title: `Cherry-pick ${shortHash(selectedCommit.hash)}?`,
          description: `O commit será aplicado em ${status?.branch ?? "HEAD"}.`,
          confirmLabel: "Aplicar commit",
        }),
    });
  }

  if (stageAllPaths.length > 0) {
    commands.push({
      id: "stage-all",
      label: "Stage em todas as alterações",
      icon: <Plus size={14} />,
      run: () => stage(stageAllPaths),
    });
  }
  if (unstageAllPaths.length > 0) {
    commands.push({
      id: "unstage-all",
      label: "Unstage de todos os arquivos",
      icon: <Minus size={14} />,
      run: () => unstage(unstageAllPaths),
    });
  }

  return (
    <div className="flex h-screen flex-col bg-bg text-ink">
      <AppHeader
        repo={repo}
        onRepoChange={changeRepo}
        recentRepos={recentRepos}
        onPickRepo={() => void pickRepo()}
        status={status}
        loading={loading}
        refreshing={manualRefreshing}
        onRefresh={refreshRepo}
        search={search}
        onSearchChange={setSearch}
        searchRef={searchRef}
      />

      {error && (
        <div className="mx-2 mt-2 flex shrink-0 items-center gap-2 rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 font-mono text-[12px] text-danger">
          <Logo className="h-5 w-5 shrink-0" />
          <span className="whitespace-pre-wrap">{error}</span>
        </div>
      )}

      <div className="flex min-h-0 flex-1 flex-col px-2 pt-2">
        <div className="flex min-h-0 flex-1 overflow-hidden rounded-lg border border-line bg-surface">
          <div className="grid min-h-0 flex-1 grid-cols-1 divide-y divide-line overflow-auto lg:grid-cols-[240px_minmax(0,1fr)_320px] lg:divide-x lg:divide-y-0 lg:overflow-hidden">
            <BranchPanel
              branches={filteredBranches}
              selected={selectedBranch}
              onSelect={setSelectedBranch}
              onCheckout={() => {
                if (!selectedBranch) return;
                setPending({
                  action: "checkout",
                  target: selectedBranch.name,
                  title: `Checkout em ${selectedBranch.name}?`,
                  description: selectedBranch.remote
                    ? "Uma branch local de tracking será criada a partir da remota."
                    : "A branch atual do repositório será trocada.",
                  confirmLabel: "Fazer checkout",
                });
              }}
              onMerge={() => {
                if (!selectedBranch) return;
                setPending({
                  action: "merge",
                  target: selectedBranch.name,
                  title: `Merge de ${selectedBranch.name}?`,
                  description: `A branch será incorporada em ${status?.branch ?? "HEAD"}. Conflitos precisam ser resolvidos no terminal.`,
                  confirmLabel: "Fazer merge",
                });
              }}
              busy={busy}
              loading={loading}
              query={search}
            />

            <CommitPanel
              commits={filteredCommits}
              selected={selectedCommit}
              onSelect={selectCommit}
              onCherryPick={() => {
                if (!selectedCommit) return;
                setPending({
                  action: "cherry-pick",
                  target: selectedCommit.hash,
                  title: `Cherry-pick ${shortHash(selectedCommit.hash)}?`,
                  description: `O commit será aplicado em ${status?.branch ?? "HEAD"}.`,
                  confirmLabel: "Aplicar commit",
                });
              }}
              onCopyHash={(hash) => void copy(hash, "Hash copiado")}
              busy={busy}
              loading={loading}
              query={search}
            />

            <StatusPanel
              groups={groups}
              total={totalFiles}
              query={search}
              loading={loading}
              busy={busy}
              commitMessage={commitMessage}
              onCommitMessageChange={setCommitMessage}
              onCommit={() => void commit()}
              onStage={stage}
              onUnstage={unstage}
            />
          </div>
        </div>

        <div
          role="separator"
          aria-orientation="horizontal"
          aria-label="Redimensionar painel de diff"
          tabIndex={0}
          onPointerDown={startResize}
          onKeyDown={(event) => {
            if (event.key === "ArrowUp") {
              event.preventDefault();
              resizeBy(24);
            } else if (event.key === "ArrowDown") {
              event.preventDefault();
              resizeBy(-24);
            }
          }}
          className="group flex h-3 shrink-0 cursor-row-resize items-center justify-center focus-visible:outline-none"
        >
          <span
            className={cn(
              "h-0.5 w-12 rounded-full bg-line transition group-hover:bg-accent/70",
            )}
          />
        </div>

        <div
          className="mb-2 min-h-0 shrink-0 overflow-hidden rounded-lg border border-line bg-surface"
          style={{ height: diffHeight }}
        >
          <DiffPanel
            selected={selectedCommit}
            diff={diff}
            loading={diffLoading}
            error={diffError}
            onCopy={() => void copyDiff()}
            copied={copiedDiff}
          />
        </div>
      </div>

      <StatusBar
        status={status}
        repo={repo}
        onOpenPalette={() => setPaletteOpen(true)}
      />

      <ConfirmDialog
        open={pending !== null}
        title={pending?.title ?? ""}
        description={pending?.description}
        confirmLabel={pending?.confirmLabel}
        busy={busy}
        onConfirm={() => void executePending()}
        onClose={() => {
          if (!busy) setPending(null);
        }}
      />

      <CommandPalette
        open={paletteOpen}
        commands={commands}
        onClose={() => setPaletteOpen(false)}
      />
    </div>
  );
}

export default App;
