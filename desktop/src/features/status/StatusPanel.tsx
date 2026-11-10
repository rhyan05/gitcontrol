import { CheckCircle2, FilePenLine, Minus, Plus, TriangleAlert } from "lucide-react";
import { Button } from "../../components/ui/Button";
import { EmptyState } from "../../components/ui/EmptyState";
import { IconButton } from "../../components/ui/IconButton";
import { Input } from "../../components/ui/Input";
import { Panel } from "../../components/ui/Panel";
import { Skeleton } from "../../components/ui/Skeleton";
import { cn } from "../../lib/cn";
import type { FileStatus } from "../../lib/gitcontrol";
import { fileCode } from "../../lib/format";
import type { StatusGroups } from "../../lib/selectors";

type GroupTone = "danger" | "success" | "warning" | "accent";

const toneText: Record<GroupTone, string> = {
  danger: "text-danger",
  success: "text-success",
  warning: "text-warning",
  accent: "text-accent",
};

export function StatusPanel({
  groups,
  total,
  query,
  loading,
  busy,
  operation,
  selectedFile,
  commitMessage,
  onCommitMessageChange,
  onCommit,
  onStage,
  onUnstage,
  onSelectFile,
  onOpenFile,
  onAbort,
  onContinue,
}: {
  groups: StatusGroups;
  total: number;
  query: string;
  loading: boolean;
  busy: boolean;
  operation?: "merge" | "cherry-pick" | "rebase";
  selectedFile: FileStatus | null;
  commitMessage: string;
  onCommitMessageChange: (value: string) => void;
  onCommit: () => void;
  onStage: (paths: string[]) => void;
  onUnstage: (paths: string[]) => void;
  onSelectFile: (file: FileStatus) => void;
  onOpenFile: (file: FileStatus) => void;
  onAbort: () => void;
  onContinue: () => void;
}) {
  const canCommit = groups.staged.length > 0 && commitMessage.trim().length > 0;

  return (
    <Panel
      title="Status"
      count={total}
      bodyClassName="overflow-auto"
      footer={
        <form
          className="flex shrink-0 gap-2 border-t border-line bg-surface-2 p-2"
          onSubmit={(event) => {
            event.preventDefault();
            if (canCommit) onCommit();
          }}
        >
          <Input
            value={commitMessage}
            onChange={(event) => onCommitMessageChange(event.currentTarget.value)}
            placeholder="Mensagem do commit"
            aria-label="Mensagem do commit"
            disabled={busy}
            className="h-8 flex-1 bg-bg"
          />
          <Button
            type="submit"
            variant="primary"
            disabled={busy || !canCommit}
            title={
              groups.staged.length === 0
                ? "Nada staged para commitar"
                : "Criar commit"
            }
          >
            Commit
          </Button>
        </form>
      }
    >
      {operation && (
        <div className="flex items-center gap-2 border-b border-line bg-warning/10 px-3 py-1.5">
          <TriangleAlert size={12} className="text-warning" />
          <span className="text-[11px] text-warning">
            {operation} em andamento
          </span>
          <div className="ml-auto flex gap-1">
            <Button onClick={onContinue} disabled={busy} className="h-6 px-2">
              Continuar
            </Button>
            <Button onClick={onAbort} disabled={busy} className="h-6 px-2">
              Abortar
            </Button>
          </div>
        </div>
      )}
      {loading ? (
        <Skeleton rows={7} />
      ) : total === 0 ? (
        <EmptyState
          icon={<CheckCircle2 size={24} />}
          title={query ? "Nada encontrado" : "Working tree limpo"}
          description={
            query
              ? "Nenhum arquivo corresponde ao filtro."
              : "Não há mudanças para mostrar. Bom trabalho."
          }
        />
      ) : (
        <>
          <StatusGroup
            title="Conflitos"
            tone="danger"
            files={groups.conflicts}
            action="stage"
            bulkLabel="Marcar todos"
            busy={busy}
            onStage={onStage}
            onUnstage={onUnstage}
            selectedFile={selectedFile}
            onSelectFile={onSelectFile}
            onOpenFile={onOpenFile}
          />
          <StatusGroup
            title="Staged"
            tone="success"
            files={groups.staged}
            action="unstage"
            bulkLabel="Unstage all"
            busy={busy}
            onStage={onStage}
            onUnstage={onUnstage}
            selectedFile={selectedFile}
            onSelectFile={onSelectFile}
            onOpenFile={onOpenFile}
          />
          <StatusGroup
            title="Alterações"
            tone="warning"
            files={groups.changes}
            action="stage"
            bulkLabel="Stage all"
            busy={busy}
            onStage={onStage}
            onUnstage={onUnstage}
            selectedFile={selectedFile}
            onSelectFile={onSelectFile}
            onOpenFile={onOpenFile}
          />
          <StatusGroup
            title="Untracked"
            tone="accent"
            files={groups.untracked}
            action="stage"
            bulkLabel="Stage all"
            busy={busy}
            onStage={onStage}
            onUnstage={onUnstage}
            selectedFile={selectedFile}
            onSelectFile={onSelectFile}
            onOpenFile={onOpenFile}
          />
        </>
      )}
    </Panel>
  );
}

function StatusGroup({
  title,
  tone,
  files,
  action,
  bulkLabel,
  busy,
  onStage,
  onUnstage,
  selectedFile,
  onSelectFile,
  onOpenFile,
}: {
  title: string;
  tone: GroupTone;
  files: FileStatus[];
  action: "stage" | "unstage";
  bulkLabel: string;
  busy: boolean;
  onStage: (paths: string[]) => void;
  onUnstage: (paths: string[]) => void;
  selectedFile: FileStatus | null;
  onSelectFile: (file: FileStatus) => void;
  onOpenFile: (file: FileStatus) => void;
}) {
  if (files.length === 0) return null;

  const run = (paths: string[]) => {
    if (action === "stage") onStage(paths);
    else onUnstage(paths);
  };

  return (
    <section>
      <header className="sticky top-0 z-10 flex h-7 items-center gap-2 border-b border-line/60 bg-surface/95 px-3 backdrop-blur">
        <span
          className={cn(
            "text-[10px] font-semibold uppercase tracking-[0.16em]",
            toneText[tone],
          )}
        >
          {title}
        </span>
        <span className="text-[10px] tabular-nums text-muted">
          {files.length}
        </span>
        <button
          type="button"
          disabled={busy}
          onClick={() => run(files.map((file) => file.path))}
          className="ml-auto rounded px-1.5 py-0.5 text-[10px] font-medium text-muted transition hover:bg-surface-2 hover:text-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 disabled:pointer-events-none disabled:opacity-40"
        >
          {bulkLabel}
        </button>
      </header>
      {files.map((file) => (
        <div
          key={`${title}:${file.path}:${fileCode(file)}`}
          className={cn(
            "group flex items-center gap-2 px-3 py-1.5 transition hover:bg-surface-2",
            selectedFile?.path === file.path && "bg-accent/10",
          )}
        >
          {file.unmerged ? (
            <TriangleAlert size={12} className="shrink-0 text-danger" />
          ) : (
            <span
              className={cn(
                "w-5 shrink-0 font-mono text-[11px]",
                toneText[tone],
              )}
            >
              {fileCode(file)}
            </span>
          )}
          <button
            type="button"
            onClick={() => onSelectFile(file)}
            className="min-w-0 flex-1 truncate text-left font-mono text-[12px] text-ink/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60"
          >
            {file.path}
          </button>
          {file.unmerged && (
            <IconButton
              label={`Abrir ${file.path} no editor`}
              icon={<FilePenLine size={13} />}
              disabled={busy}
              onClick={() => onOpenFile(file)}
              className="h-6 w-6 opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
            />
          )}
          <IconButton
            label={
              action === "stage"
                ? `Stage ${file.path}`
                : `Unstage ${file.path}`
            }
            icon={action === "stage" ? <Plus size={13} /> : <Minus size={13} />}
            disabled={busy}
            onClick={() => run([file.path])}
            className="h-6 w-6 opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
          />
        </div>
      ))}
    </section>
  );
}
