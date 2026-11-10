import { describe, expect, it } from "vitest";
import {
  filterBranches,
  filterCommits,
  groupStatus,
  matches,
} from "./selectors";
import { fileCode, formatRelativeTime, initials, shortHash } from "./format";
import type { Branch, Commit, FileStatus } from "./gitcontrol";

const branch = (name: string, extra: Partial<Branch> = {}): Branch => ({
  name,
  ...extra,
});

const commit = (subject: string, extra: Partial<Commit> = {}): Commit => ({
  hash: "abcdef1234567890",
  author: "Rhyan Souza",
  time: new Date().toISOString(),
  subject,
  ...extra,
});

const file = (path: string, extra: Partial<FileStatus> = {}): FileStatus => ({
  path,
  index_xy: ".",
  worktree_xy: "M",
  ...extra,
});

describe("selectors", () => {
  it("filtra branches por nome e upstream", () => {
    const branches = [
      branch("main", { upstream: "origin/main" }),
      branch("feature/login"),
      branch("origin/main", { remote: true }),
    ];

    expect(filterBranches(branches, "login")).toHaveLength(1);
    expect(filterBranches(branches, "origin")).toHaveLength(2);
    expect(filterBranches(branches, "  ")).toHaveLength(3);
  });

  it("filtra commits por subject, hash e autor", () => {
    const commits = [
      commit("feat: login"),
      commit("fix: layout", { hash: "fedcba0987654321" }),
    ];

    expect(filterCommits(commits, "login")).toHaveLength(1);
    expect(filterCommits(commits, "1234")).toHaveLength(1);
    expect(filterCommits(commits, "rhyan")).toHaveLength(2);
  });

  it("agrupa staged, changes, untracked e conflitos", () => {
    const groups = groupStatus([
      file("staged.ts", { index_xy: "M", worktree_xy: "." }),
      file("changed.ts", { index_xy: ".", worktree_xy: "M" }),
      file("both.ts", { index_xy: "M", worktree_xy: "M" }),
      file("new.ts", { index_xy: "", worktree_xy: "", untracked: true }),
      file("conflict.ts", { unmerged: true }),
      file("ignored.ts", { ignored: true, untracked: true }),
    ]);

    expect(groups.staged.map((item) => item.path)).toEqual([
      "staged.ts",
      "both.ts",
    ]);
    expect(groups.changes.map((item) => item.path)).toEqual([
      "changed.ts",
      "both.ts",
    ]);
    expect(groups.untracked.map((item) => item.path)).toEqual(["new.ts"]);
    expect(groups.conflicts.map((item) => item.path)).toEqual(["conflict.ts"]);
  });

  it("matches ignora caixa e espaços", () => {
    expect(matches("Feature/Login", " login ")).toBe(true);
    expect(matches("main", "dev")).toBe(false);
  });
});

describe("format", () => {
  it("encurta hash e iniciais", () => {
    expect(shortHash("abcdef123456")).toBe("abcdef1");
    expect(initials("Rhyan Souza")).toBe("RS");
    expect(initials("Rhyan")).toBe("RH");
  });

  it("traduz estados de arquivo", () => {
    expect(fileCode(file("a.ts", { untracked: true }))).toBe("??");
    expect(fileCode(file("a.ts", { unmerged: true }))).toBe("UU");
    expect(fileCode(file("a.ts", { index_xy: "M", worktree_xy: "." }))).toBe(
      "M.",
    );
  });

  it("formata tempo relativo", () => {
    const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeTime(twoHoursAgo)).toContain("2");
  });
});
