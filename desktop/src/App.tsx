import { useCallback, useEffect, useRef, useState } from "react";
import {
  gg,
  type Branch,
  type Commit,
  type FileStatus,
  type ShowResult,
  type StatusResult,
} from "./gg";
import "./App.css";

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

function App() {
  // O caminho do repo fica salvo entre execuções. ".." funciona no `tauri dev`
  // porque o processo roda dentro de desktop/, ainda dentro deste repositório.
  const [repo, setRepo] = useState(
    () => localStorage.getItem("gg.repo") ?? "..",
  );
  const [status, setStatus] = useState<StatusResult | null>(null);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [commits, setCommits] = useState<Commit[]>([]);
  const [selected, setSelected] = useState<Commit | null>(null);
  const [diff, setDiff] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(true);

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
    setSelected(commit);
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

  const files = (status?.files ?? []).filter((file) => !file.ignored);

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
          <button type="button" onClick={() => void load()} disabled={loading}>
            {loading ? "Carregando…" : "Atualizar"}
          </button>
        </div>
      </header>

      {error && <div className="error">{error}</div>}

      <main className="panels">
        <section className="panel">
          <header>
            <h2>Branches</h2>
            <span className="count">{branches.length}</span>
          </header>
          <ul>
            {branches.map((branch) => (
              <li key={`${branch.remote ? "r" : "l"}:${branch.name}`}>
                <span className={branch.head ? "dot head" : "dot"} />
                <span className="name">{branch.name}</span>
                {branch.remote && <span className="tag">remota</span>}
                {!branch.remote && branch.upstream && (
                  <span className="muted">→ {branch.upstream}</span>
                )}
              </li>
            ))}
            {branches.length === 0 && <li className="empty">nenhuma branch</li>}
          </ul>
        </section>

        <section className="panel">
          <header>
            <h2>Commits</h2>
            <span className="count">{commits.length}</span>
          </header>
          <ul>
            {commits.map((commit) => (
              <li
                key={commit.hash}
                className={selected?.hash === commit.hash ? "selected" : ""}
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
              </li>
            ))}
            {files.length === 0 && <li className="empty">working tree limpo</li>}
          </ul>
        </section>
      </main>

      <section className="diff">
        <header>
          <h2>{selected ? `Diff ${selected.hash.slice(0, 7)}` : "Diff"}</h2>
        </header>
        {selected ? (
          <pre>{diff || "carregando…"}</pre>
        ) : (
          <p className="empty">Selecione um commit para ver o diff.</p>
        )}
      </section>
    </div>
  );
}

export default App;
