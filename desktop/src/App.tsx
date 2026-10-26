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

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function fileCode(file: FileStatus): string {
  if (file.untracked) return "??";
  if (file.unmerged) return "UU";
  return `${file.index_xy}${file.worktree_xy}`;
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
    <div className="app">
      <header className="topbar">
        <div className="brand">
          <strong>gg</strong>
          <span className="branch">{status?.branch ?? "—"}</span>
          {status?.upstream && (
            <span className="muted">→ {status.upstream}</span>
          )}
          {(status?.ahead ?? 0) > 0 && (
            <span className="tag">↑{status?.ahead}</span>
          )}
          {(status?.behind ?? 0) > 0 && (
            <span className="tag">↓{status?.behind}</span>
          )}
        </div>

        <div className="repo">
          <input
            value={repo}
            onChange={(event) => setRepo(event.currentTarget.value)}
            spellCheck={false}
            aria-label="Caminho do repositório"
          />
          <button
            type="button"
            onClick={() => void load()}
            disabled={loading || busy}
          >
            {loading ? "Carregando…" : "Atualizar"}
          </button>
        </div>
      </header>

      {error && <div className="error">{error}</div>}
      {notice && <div className="notice">{notice}</div>}

      {pending && (
        <div className="confirm">
          <span>{pending.label}</span>
          <div>
            <button
              type="button"
              onClick={() => void executePending()}
              disabled={busy}
            >
              Confirmar
            </button>
            <button
              type="button"
              onClick={() => setPending(null)}
              disabled={busy}
            >
              Cancelar
            </button>
          </div>
        </div>
      )}

      <main className="panels">
        <section className="panel">
          <header>
            <h2>Branches</h2>
            <div className="panel-tools">
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
                  >
                    Merge
                  </button>
                </>
              )}
              <span className="count">{branches.length}</span>
            </div>
          </header>
          <ul>
            {branches.map((branch) => {
              const selected =
                selectedBranch?.name === branch.name &&
                selectedBranch.remote === branch.remote;
              return (
                <li
                  key={`${branch.remote ? "r" : "l"}:${branch.name}`}
                  className={selected ? "selected" : ""}
                  onClick={() => setSelectedBranch(branch)}
                >
                  <span className={branch.head ? "dot head" : "dot"} />
                  <span className="name">{branch.name}</span>
                  {branch.remote && <span className="tag">remota</span>}
                  {!branch.remote && branch.upstream && (
                    <span className="muted">→ {branch.upstream}</span>
                  )}
                </li>
              );
            })}
            {branches.length === 0 && <li className="empty">nenhuma branch</li>}
          </ul>
        </section>

        <section className="panel">
          <header>
            <h2>Commits</h2>
            <div className="panel-tools">
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
                >
                  Cherry-pick
                </button>
              )}
              <span className="count">{commits.length}</span>
            </div>
          </header>
          <ul>
            {commits.map((commit) => (
              <li
                key={commit.hash}
                className={selectedCommit?.hash === commit.hash ? "selected" : ""}
                onClick={() => void openCommit(commit)}
              >
                <span className="hash">{commit.hash.slice(0, 7)}</span>
                <span className="subject">{commit.subject}</span>
                <span className="meta">
                  {commit.author} · {formatDate(commit.time)}
                </span>
              </li>
            ))}
            {commits.length === 0 && <li className="empty">nenhum commit</li>}
          </ul>
        </section>

        <section className="panel">
          <header>
            <h2>Status</h2>
            <span className="count">{files.length}</span>
          </header>
          <ul>
            {files.map((file) => (
              <li key={`${file.path}:${fileCode(file)}`}>
                <span className={file.untracked ? "code untracked" : "code"}>
                  {fileCode(file)}
                </span>
                <span className="path">{file.path}</span>
                <span className="file-actions">
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
                    >
                      Unstage
                    </button>
                  )}
                </span>
              </li>
            ))}
            {files.length === 0 && <li className="empty">working tree limpo</li>}
          </ul>

          <form className="commit-form" onSubmit={(e) => void submitCommit(e)}>
            <input
              value={commitMessage}
              onChange={(event) => setCommitMessage(event.currentTarget.value)}
              placeholder="mensagem do commit"
              disabled={busy}
            />
            <button type="submit" disabled={busy || !commitMessage.trim()}>
              Commit
            </button>
          </form>
        </section>
      </main>

      <section className="diff">
        <header>
          <h2>
            {selectedCommit
              ? `Diff ${selectedCommit.hash.slice(0, 7)}`
              : "Diff"}
          </h2>
        </header>
        {selectedCommit ? (
          <pre>{diff || "carregando…"}</pre>
        ) : (
          <p className="empty">Selecione um commit para ver o diff.</p>
        )}
      </section>
    </div>
  );
}

export default App;
