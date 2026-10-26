import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FormEvent,
} from "react";
import {
  gg,
  type Branch,
  type Commit,
  type FileStatus,
  type ShowResult,
  type StatusResult,
} from "./gg";
import "./App.css";

type PendingAction = {
  action: "checkout" | "merge" | "cherry-pick";
  target: string;
  label: string;
};

const panelClass =
  "flex min-h-[240px] min-w-0 flex-col overflow-hidden rounded-xl border border-line bg-surface shadow-xl shadow-black/20 lg:min-h-0";
const panelHeaderClass =
  "flex items-center justify-between gap-2 border-b border-line bg-surface-2 px-3 py-2";
const panelTitleClass =
  "text-[11px] font-semibold uppercase tracking-[0.16em] text-muted";
const listClass = "min-h-0 flex-1 overflow-auto py-1";
const buttonClass =
  "rounded-md border border-line bg-surface-2 px-2 py-0.5 text-[11px] font-medium text-ink transition hover:border-accent hover:text-accent disabled:cursor-not-allowed disabled:opacity-40";
const inputClass =
  "min-w-0 rounded-lg border border-line bg-surface px-3 py-1.5 font-mono text-xs text-ink outline-none transition placeholder:text-muted focus:border-accent focus:ring-2 focus:ring-accent/20";

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function fileCode(file: FileStatus): string {
  if (file.untracked) return "??";
  if (file.unmerged) return "UU";
  return `${file.index_xy}${file.worktree_xy}`;
}

function codeColor(file: FileStatus): string {
  if (file.untracked) return "text-warning";
  if (file.unmerged) return "text-danger";
  if (isStaged(file)) return "text-success";
  return "text-muted";
}

function formatDate(iso: string): string {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleString("pt-BR");
}

function isStaged(file: FileStatus): boolean {
  return file.index_xy !== "." && file.index_xy !== "";
}

function hasWorktreeChange(file: FileStatus): boolean {
  return file.worktree_xy !== "." && file.worktree_xy !== "";
}

function DiffView({ diff }: { diff: string }) {
  return (
    <pre className="min-h-0 flex-1 overflow-auto whitespace-pre px-3 py-2 font-mono text-[11.5px] leading-[1.45]">
      {diff.split("\n").map((line, index) => {
        let color = "text-ink/80";
        if (line.startsWith("+++") || line.startsWith("---")) {
          color = "text-muted";
        } else if (line.startsWith("+")) {
          color = "text-success";
        } else if (line.startsWith("-")) {
          color = "text-danger";
        } else if (line.startsWith("@@")) {
          color = "text-accent";
        } else if (
          line.startsWith("diff ") ||
          line.startsWith("index ") ||
          line.startsWith("\\ No newline")
        ) {
          color = "text-muted";
        }

        return (
          <span key={index} className={`block ${color}`}>
            {line || " "}
          </span>
        );
      })}
    </pre>
  );
}

function App() {
  // O caminho do repo fica salvo entre execuções. ".." funciona no `tauri dev`
  // porque o processo roda dentro de desktop/, ainda dentro deste repositório.
  const [repo, setRepo] = useState(
    () => localStorage.getItem("gg.repo") ?? "..",
  );
  const [status, setStatus] = useState<StatusResult | null>(null);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [commits, setCommits] = useState<Commit[]>([]);
  const [selectedBranch, setSelectedBranch] = useState<Branch | null>(null);
  const [selectedCommit, setSelectedCommit] = useState<Commit | null>(null);
  const [pending, setPending] = useState<PendingAction | null>(null);
  const [commitMessage, setCommitMessage] = useState("");
  const [diff, setDiff] = useState("");
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);

  // Evita que uma resposta antiga de `show` sobrescreva um clique mais novo.
  const diffRequest = useRef(0);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [nextStatus, nextBranches, nextCommits] = await Promise.all([
        gg<StatusResult>(repo, "status"),
        gg<Branch[]>(repo, "branches"),
        gg<Commit[]>(repo, "log"),
      ]);
      setStatus(nextStatus);
      setBranches(nextBranches);
      setCommits(nextCommits);
    } catch (err) {
      setError(message(err));
    } finally {
      setLoading(false);
    }
  }, [repo]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    localStorage.setItem("gg.repo", repo);
  }, [repo]);

  async function openCommit(commit: Commit) {
    const request = ++diffRequest.current;
    setSelectedCommit(commit);
    setDiff("");

    try {
      const result = await gg<ShowResult>(repo, "show", commit.hash);
      if (request === diffRequest.current) {
        setDiff(result.diff);
      }
    } catch (err) {
      if (request === diffRequest.current) {
        setDiff(`erro: ${message(err)}`);
      }
    }
  }

  // Executa uma ação do contrato --json, recarrega e mostra o resultado.
  // Ações que mudam branches/commits limpam a seleção; stage/unstage não.
  const runAction = useCallback(
    async (
      args: string[],
      success: string,
      resetSelection = true,
    ): Promise<boolean> => {
      setBusy(true);
      setError("");
      setNotice("");

      try {
        await gg<null>(repo, ...args);
        await load();

        if (resetSelection) {
          setSelectedBranch(null);
          setSelectedCommit(null);
          setDiff("");
        }
        setPending(null);
        setNotice(success);
        return true;
      } catch (err) {
        setError(message(err));
        return false;
      } finally {
        setBusy(false);
      }
    },
    [repo, load],
  );

  async function executePending() {
    if (!pending) return;

    switch (pending.action) {
      case "checkout":
        await runAction(
          ["checkout", pending.target],
          `checkout: ${pending.target}`,
        );
        break;
      case "merge":
        await runAction(["merge", pending.target], `merge: ${pending.target}`);
        break;
      case "cherry-pick":
        await runAction(
          ["cherry-pick", pending.target],
          `cherry-pick: ${pending.target.slice(0, 7)}`,
        );
        break;
    }
  }

  async function submitCommit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const text = commitMessage.trim();
    if (!text) return;

    const done = await runAction(["commit", text], `commit: ${text}`);
    if (done) {
      setCommitMessage("");
    }
  }

  const files = (status?.files ?? []).filter((file) => !file.ignored);
  const currentBranch = status?.branch ?? "HEAD";

  return (
    <div className="flex h-screen flex-col gap-3 bg-bg p-3 text-sm">
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2.5">
          <span className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-accent text-[11px] font-black tracking-tight text-black">
            gg
          </span>
          <span className="shrink-0 rounded-full border border-line bg-surface px-2.5 py-0.5 font-mono text-[11px] text-success">
            {status?.branch ?? "—"}
          </span>
          {status?.upstream && (
            <span className="truncate font-mono text-[11px] text-muted">
              → {status.upstream}
            </span>
          )}
          {(status?.ahead ?? 0) > 0 && (
            <span className="shrink-0 rounded bg-accent/15 px-1.5 py-0.5 text-[10px] font-semibold text-accent">
              ↑{status?.ahead}
            </span>
          )}
          {(status?.behind ?? 0) > 0 && (
            <span className="shrink-0 rounded bg-warning/15 px-1.5 py-0.5 text-[10px] font-semibold text-warning">
              ↓{status?.behind}
            </span>
          )}
        </div>

        <div className="flex w-full items-center gap-2 sm:w-auto">
          <input
            value={repo}
            onChange={(event) => setRepo(event.currentTarget.value)}
            spellCheck={false}
            aria-label="Caminho do repositório"
            className={`${inputClass} flex-1 sm:w-[340px] sm:flex-none`}
          />
          <button
            type="button"
            onClick={() => void load()}
            disabled={loading || busy}
            className="shrink-0 rounded-lg border border-line bg-surface px-3 py-1.5 text-xs font-medium text-ink transition hover:border-accent hover:text-accent disabled:cursor-progress disabled:opacity-50"
          >
            {loading ? "Carregando…" : "Atualizar"}
          </button>
        </div>
      </header>

      {error && (
        <div className="whitespace-pre-wrap rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 font-mono text-xs text-danger">
          {error}
        </div>
      )}

      {notice && (
        <div className="whitespace-pre-wrap rounded-lg border border-success/40 bg-success/10 px-3 py-2 font-mono text-xs text-success">
          {notice}
        </div>
      )}

      {pending && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-warning/40 bg-warning/10 px-3 py-2 text-xs text-warning">
          <span>{pending.label}</span>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => void executePending()}
              disabled={busy}
              className={buttonClass}
            >
              Confirmar
            </button>
            <button
              type="button"
              onClick={() => setPending(null)}
              disabled={busy}
              className={buttonClass}
            >
              Cancelar
            </button>
          </div>
        </div>
      )}

      <main className="grid min-h-0 flex-1 grid-cols-1 gap-3 overflow-auto lg:grid-cols-3 lg:overflow-hidden">
        <section className={panelClass}>
          <header className={panelHeaderClass}>
            <h2 className={panelTitleClass}>Branches</h2>
            <div className="flex items-center gap-1.5">
              {selectedBranch && (
                <>
                  <button
                    type="button"
                    disabled={busy || selectedBranch.head}
                    onClick={() =>
                      setPending({
                        action: "checkout",
                        target: selectedBranch.name,
                        label: `checkout ${selectedBranch.name}${
                          selectedBranch.remote ? " (cria branch local)" : ""
                        }?`,
                      })
                    }
                    className={buttonClass}
                  >
                    Checkout
                  </button>
                  <button
                    type="button"
                    disabled={busy || selectedBranch.head}
                    onClick={() =>
                      setPending({
                        action: "merge",
                        target: selectedBranch.name,
                        label: `merge ${selectedBranch.name} em ${currentBranch}?`,
                      })
                    }
                    className={buttonClass}
                  >
                    Merge
                  </button>
                </>
              )}
              <span className="text-[10px] tabular-nums text-muted">
                {branches.length}
              </span>
            </div>
          </header>
          <ul className={listClass}>
            {branches.map((branch) => {
              const selected =
                selectedBranch?.name === branch.name &&
                selectedBranch.remote === branch.remote;
              return (
                <li
                  key={`${branch.remote ? "r" : "l"}:${branch.name}`}
                  className={`flex cursor-pointer items-center gap-2 border-l-2 px-3 py-1.5 transition ${
                    selected
                      ? "border-accent bg-accent/10"
                      : "border-transparent hover:bg-surface-2"
                  }`}
                  onClick={() => setSelectedBranch(branch)}
                >
                  <span
                    className={`h-1.5 w-1.5 shrink-0 rounded-full border ${
                      branch.head
                        ? "border-success bg-success"
                        : "border-muted"
                    }`}
                  />
                  <span className="truncate font-mono text-[12.5px] text-ink/90">
                    {branch.name}
                  </span>
                  {branch.remote && (
                    <span className="ml-auto shrink-0 rounded bg-accent/15 px-1.5 py-0.5 text-[9px] font-semibold uppercase tracking-wide text-accent">
                      remota
                    </span>
                  )}
                  {!branch.remote && branch.upstream && (
                    <span className="ml-auto shrink-0 truncate font-mono text-[10px] text-muted">
                      → {branch.upstream}
                    </span>
                  )}
                </li>
              );
            })}
            {branches.length === 0 && (
              <li className="px-3 py-2 text-xs italic text-muted">
                nenhuma branch
              </li>
            )}
          </ul>
        </section>

        <section className={panelClass}>
          <header className={panelHeaderClass}>
            <h2 className={panelTitleClass}>Commits</h2>
            <div className="flex items-center gap-1.5">
              {selectedCommit && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() =>
                    setPending({
                      action: "cherry-pick",
                      target: selectedCommit.hash,
                      label: `cherry-pick ${selectedCommit.hash.slice(
                        0,
                        7,
                      )} em ${currentBranch}?`,
                    })
                  }
                  className={buttonClass}
                >
                  Cherry-pick
                </button>
              )}
              <span className="text-[10px] tabular-nums text-muted">
                {commits.length}
              </span>
            </div>
          </header>
          <ul className={listClass}>
            {commits.map((commit) => (
              <li
                key={commit.hash}
                className={`flex cursor-pointer flex-col gap-0.5 border-l-2 px-3 py-1.5 transition ${
                  selectedCommit?.hash === commit.hash
                    ? "border-accent bg-accent/10"
                    : "border-transparent hover:bg-surface-2"
                }`}
                onClick={() => void openCommit(commit)}
              >
                <div className="flex items-baseline gap-2">
                  <span className="shrink-0 font-mono text-[11px] text-muted">
                    {commit.hash.slice(0, 7)}
                  </span>
                  <span className="truncate text-[13px] text-ink/90">
                    {commit.subject}
                  </span>
                </div>
                <div className="text-[10px] text-muted">
                  {commit.author} · {formatDate(commit.time)}
                </div>
              </li>
            ))}
            {commits.length === 0 && (
              <li className="px-3 py-2 text-xs italic text-muted">
                nenhum commit
              </li>
            )}
          </ul>
        </section>

        <section className={panelClass}>
          <header className={panelHeaderClass}>
            <h2 className={panelTitleClass}>Status</h2>
            <span className="text-[10px] tabular-nums text-muted">
              {files.length}
            </span>
          </header>
          <ul className={listClass}>
            {files.map((file) => (
              <li
                key={`${file.path}:${fileCode(file)}`}
                className="flex items-center gap-2 px-3 py-1.5 transition hover:bg-surface-2"
              >
                <span
                  className={`w-5 shrink-0 font-mono text-[11px] ${codeColor(
                    file,
                  )}`}
                >
                  {fileCode(file)}
                </span>
                <span className="min-w-0 flex-1 truncate font-mono text-[12px] text-ink/90">
                  {file.path}
                </span>
                <span className="flex shrink-0 gap-1">
                  {(file.untracked ||
                    file.unmerged ||
                    hasWorktreeChange(file)) && (
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() =>
                        void runAction(
                          ["stage", file.path],
                          `stage: ${file.path}`,
                          false,
                        )
                      }
                      className={buttonClass}
                    >
                      Stage
                    </button>
                  )}
                  {isStaged(file) && (
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() =>
                        void runAction(
                          ["unstage", file.path],
                          `unstage: ${file.path}`,
                          false,
                        )
                      }
                      className={buttonClass}
                    >
                      Unstage
                    </button>
                  )}
                </span>
              </li>
            ))}
            {files.length === 0 && (
              <li className="px-3 py-2 text-xs italic text-muted">
                working tree limpo
              </li>
            )}
          </ul>

          <form
            className="flex gap-2 border-t border-line bg-surface-2 p-2"
            onSubmit={(e) => void submitCommit(e)}
          >
            <input
              value={commitMessage}
              onChange={(event) => setCommitMessage(event.currentTarget.value)}
              placeholder="mensagem do commit"
              disabled={busy}
              className={`${inputClass} flex-1`}
            />
            <button
              type="submit"
              disabled={busy || !commitMessage.trim()}
              className="shrink-0 rounded-md bg-accent px-3 py-1.5 text-xs font-semibold text-black transition hover:bg-accent/90 disabled:cursor-not-allowed disabled:opacity-40"
            >
              Commit
            </button>
          </form>
        </section>
      </main>

      <section className="flex h-[28vh] min-h-[140px] max-h-[300px] shrink-0 flex-col overflow-hidden rounded-xl border border-line bg-surface shadow-xl shadow-black/20">
        <header className={panelHeaderClass}>
          <h2 className={panelTitleClass}>
            {selectedCommit
              ? `Diff ${selectedCommit.hash.slice(0, 7)}`
              : "Diff"}
          </h2>
        </header>
        {selectedCommit ? (
          diff ? (
            <DiffView diff={diff} />
          ) : (
            <p className="px-3 py-2 text-xs text-muted">carregando…</p>
          )
        ) : (
          <p className="px-3 py-2 text-xs text-muted">
            Selecione um commit para ver o diff.
          </p>
        )}
      </section>
    </div>
  );
}

export default App;
