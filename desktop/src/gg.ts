import { Command } from "@tauri-apps/plugin-shell";

// Tipos espelham exatamente os JSON tags do core (internal/git/types.go).
export type FileStatus = {
  path: string;
  old_path?: string;
  index_xy: string;
  worktree_xy: string;
  untracked?: boolean;
  ignored?: boolean;
  unmerged?: boolean;
};

export type StatusResult = {
  branch: string;
  head_oid: string;
  upstream?: string;
  ahead?: number;
  behind?: number;
  files: FileStatus[];
};

export type Commit = {
  hash: string;
  parents?: string[];
  author: string;
  time: string;
  subject: string;
};

export type Branch = {
  name: string;
  upstream?: string;
  head?: boolean;
  remote?: boolean;
};

export type ShowResult = {
  hash: string;
  diff: string;
};

type ErrorBody = { code: string; message: string };

export type Envelope<T> =
  | { ok: true; data: T }
  | { ok: false; error: ErrorBody };

// gg executa o sidecar com o contrato --json e devolve só o data.
// O envelope de erro vira uma exceção com "code: message".
export async function gg<T>(repo: string, ...args: string[]): Promise<T> {
  const command = Command.sidecar("binaries/gg-cli", ["--json", ...args], {
    cwd: repo,
  });
  const output = await command.execute();

  let envelope: Envelope<T>;
  try {
    envelope = JSON.parse(output.stdout) as Envelope<T>;
  } catch {
    throw new Error(
      output.stderr.trim() || `gg terminou com código ${output.code}`,
    );
  }

  if (!envelope.ok) {
    throw new Error(`${envelope.error.code}: ${envelope.error.message}`);
  }
  return envelope.data;
}
