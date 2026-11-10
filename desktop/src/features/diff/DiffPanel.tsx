import { useMemo, useState } from "react";
import { Check, Copy, FileCode2, GitCommitHorizontal, WrapText } from "lucide-react";
import { EmptyState } from "../../components/ui/EmptyState";
import { IconButton } from "../../components/ui/IconButton";
import { Skeleton } from "../../components/ui/Skeleton";
import { cn } from "../../lib/cn";
import type { Commit, FileStatus } from "../../lib/gitcontrol";
import { shortHash } from "../../lib/format";

function lineStyle(line: string): { text: string; background: string } {
  if (line.startsWith("+++") || line.startsWith("---")) {
    return { text: "text-muted", background: "" };
  }
  if (line.startsWith("+")) {
    return { text: "text-success", background: "bg-success/[0.06]" };
  }
  if (line.startsWith("-")) {
    return { text: "text-danger", background: "bg-danger/[0.06]" };
  }
  if (line.startsWith("@@")) {
    return { text: "text-accent", background: "bg-accent/[0.05]" };
  }
  if (
    line.startsWith("diff ") ||
    line.startsWith("index ") ||
    line.startsWith("\\ No newline")
  ) {
    return { text: "text-muted", background: "" };
  }
  return { text: "text-ink/80", background: "" };
}

export function DiffPanel({
  selected,
  selectedFile,
  diff,
  loading,
  error,
  onCopy,
  copied,
}: {
  selected: Commit | null;
  selectedFile: FileStatus | null;
  diff: string;
  loading: boolean;
  error: string;
  onCopy: () => void;
  copied: boolean;
}) {
  const [wrap, setWrap] = useState(false);
  const lines = useMemo(() => (diff ? diff.split("\n") : []), [diff]);

  return (
    <section className="flex h-full min-h-0 flex-col overflow-hidden">
      <header className="flex h-9 shrink-0 items-center gap-2 border-b border-line bg-surface-2 px-3">
        <h2 className="text-[11px] font-semibold uppercase tracking-[0.14em] text-muted">
          Diff
        </h2>
        {selected && (
          <>
            <span className="font-mono text-[11px] text-accent">
              {shortHash(selected.hash)}
            </span>
            <span className="min-w-0 truncate text-[12px] text-muted">
              {selected.subject}
            </span>
          </>
        )}
        {selectedFile && (
          <>
            <span className="font-mono text-[11px] text-accent">
              working tree
            </span>
            <span className="min-w-0 truncate text-[12px] text-muted">
              {selectedFile.path}
            </span>
          </>
        )}
        <div className="ml-auto flex items-center gap-1">
          {diff && (
            <span className="mr-1 font-mono text-[10px] text-muted/70">
              {lines.length} linhas
            </span>
          )}
          <IconButton
            label={wrap ? "Desativar quebra de linha" : "Quebrar linhas longas"}
            icon={<WrapText size={14} />}
            aria-pressed={wrap}
            className={cn(wrap && "bg-accent/10 text-accent")}
            onClick={() => setWrap((current) => !current)}
          />
          <IconButton
            label="Copiar diff"
            icon={copied ? <Check size={14} /> : <Copy size={14} />}
            onClick={onCopy}
            disabled={!diff}
          />
        </div>
      </header>

      {!selected && !selectedFile ? (
        <EmptyState
          icon={
            selectedFile ? (
              <FileCode2 size={24} />
            ) : (
              <GitCommitHorizontal size={24} />
            )
          }
          title="Selecione um commit ou arquivo"
          description="O diff do commit ou do arquivo do working tree aparece aqui, com linhas adicionadas e removidas destacadas."
        />
      ) : loading ? (
        <Skeleton rows={10} />
      ) : error ? (
        <div className="m-3 whitespace-pre-wrap rounded-lg border border-danger/40 bg-danger/10 p-3 font-mono text-[12px] text-danger">
          {error}
        </div>
      ) : lines.length === 0 ? (
        <EmptyState
          title="Sem alterações"
          description="Este commit não alterou nenhum arquivo."
        />
      ) : (
        <div className="min-h-0 flex-1 overflow-auto font-mono text-[12px] leading-[1.5]">
          {lines.map((line, index) => {
            const style = lineStyle(line);
            return (
              <div
                key={index}
                className={cn("flex", style.background)}
              >
                <span className="w-11 shrink-0 select-none border-r border-line/50 px-2 text-right text-muted/40">
                  {index + 1}
                </span>
                <span
                  className={cn(
                    "min-w-0 px-3",
                    style.text,
                    wrap ? "whitespace-pre-wrap break-words" : "whitespace-pre",
                  )}
                >
                  {line || " "}
                </span>
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
