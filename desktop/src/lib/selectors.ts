import type { Branch, Commit, FileStatus } from "./gg";
import { hasWorktreeChange, isStaged } from "./format";

export function matches(text: string, query: string): boolean {
  return text.toLowerCase().includes(query.trim().toLowerCase());
}

export function filterBranches(branches: Branch[], query: string): Branch[] {
  if (!query.trim()) return branches;
  return branches.filter(
    (branch) =>
      matches(branch.name, query) || matches(branch.upstream ?? "", query),
  );
}

export function filterCommits(commits: Commit[], query: string): Commit[] {
  if (!query.trim()) return commits;
  return commits.filter(
    (commit) =>
      matches(commit.subject, query) ||
      matches(commit.hash, query) ||
      matches(commit.author, query),
  );
}

export function filterFiles(files: FileStatus[], query: string): FileStatus[] {
  if (!query.trim()) return files;
  return files.filter((file) => matches(file.path, query));
}

export type StatusGroups = {
  conflicts: FileStatus[];
  staged: FileStatus[];
  changes: FileStatus[];
  untracked: FileStatus[];
};

// Mesmo arquivo pode aparecer em staged e changes (index + working tree).
export function groupStatus(files: FileStatus[]): StatusGroups {
  const groups: StatusGroups = {
    conflicts: [],
    staged: [],
    changes: [],
    untracked: [],
  };

  for (const file of files) {
    if (file.ignored) continue;

    if (file.unmerged) {
      groups.conflicts.push(file);
      continue;
    }
    if (file.untracked) {
      groups.untracked.push(file);
      continue;
    }
    if (isStaged(file)) {
      groups.staged.push(file);
    }
    if (hasWorktreeChange(file)) {
      groups.changes.push(file);
    }
  }

  return groups;
}

export function visibleFiles(files: FileStatus[]): FileStatus[] {
  return files.filter((file) => !file.ignored);
}
