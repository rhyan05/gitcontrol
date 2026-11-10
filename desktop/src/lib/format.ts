import type { FileStatus } from "./gitcontrol";

export function shortHash(hash: string): string {
  return hash.slice(0, 7);
}

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

export function fileCode(file: FileStatus): string {
  if (file.untracked) return "??";
  if (file.unmerged) return "UU";
  return `${file.index_xy}${file.worktree_xy}`;
}

export function fileCodeColor(file: FileStatus): string {
  if (file.untracked) return "text-warning";
  if (file.unmerged) return "text-danger";
  if (isStaged(file)) return "text-success";
  return "text-muted";
}

export function isStaged(file: FileStatus): boolean {
  return file.index_xy !== "." && file.index_xy !== "";
}

export function hasWorktreeChange(file: FileStatus): boolean {
  return file.worktree_xy !== "." && file.worktree_xy !== "";
}

export function formatRelativeTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;

  const delta = date.getTime() - Date.now();
  const formatter = new Intl.RelativeTimeFormat("pt-BR", {
    numeric: "auto",
  });

  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 1000 * 60 * 60 * 24 * 365],
    ["month", 1000 * 60 * 60 * 24 * 30],
    ["week", 1000 * 60 * 60 * 24 * 7],
    ["day", 1000 * 60 * 60 * 24],
    ["hour", 1000 * 60 * 60],
    ["minute", 1000 * 60],
    ["second", 1000],
  ];

  for (const [unit, size] of units) {
    if (Math.abs(delta) >= size || unit === "second") {
      return formatter.format(Math.round(delta / size), unit);
    }
  }
  return "agora";
}

export function formatAbsoluteTime(iso: string): string {
  const date = new Date(iso);
  return Number.isNaN(date.getTime())
    ? iso
    : date.toLocaleString("pt-BR", {
        dateStyle: "short",
        timeStyle: "short",
      });
}
