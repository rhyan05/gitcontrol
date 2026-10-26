import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Search } from "lucide-react";
import { Kbd } from "../../components/ui/Kbd";
import { cn } from "../../lib/cn";

export type Command = {
  id: string;
  label: string;
  hint?: string;
  icon?: ReactNode;
  run: () => void;
};

export function CommandPalette({
  open,
  commands,
  onClose,
}: {
  open: boolean;
  commands: Command[];
  onClose: () => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState("");
  const [index, setIndex] = useState(0);

  const filtered = useMemo(() => {
    const term = query.trim().toLowerCase();
    if (!term) return commands;
    return commands.filter((command) =>
      `${command.label} ${command.hint ?? ""}`.toLowerCase().includes(term),
    );
  }, [commands, query]);

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open && !dialog.open) {
      dialog.showModal();
      window.setTimeout(() => inputRef.current?.focus(), 0);
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open]);

  function close() {
    setQuery("");
    setIndex(0);
    onClose();
  }

  function run(command: Command | undefined) {
    if (!command) return;
    close();
    command.run();
  }

  return (
    <dialog
      ref={dialogRef}
      aria-label="Paleta de comandos"
      onCancel={(event) => {
        event.preventDefault();
        close();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget) close();
      }}
      className="w-[min(92vw,560px)] rounded-xl border border-line bg-surface p-0 text-ink shadow-2xl shadow-black/60 backdrop:bg-black/70 backdrop:backdrop-blur-[2px]"
    >
      <div className="flex items-center gap-2 border-b border-line px-3">
        <Search size={14} className="text-muted" />
        <input
          ref={inputRef}
          value={query}
          onChange={(event) => {
            setQuery(event.currentTarget.value);
            setIndex(0);
          }}
          onKeyDown={(event) => {
            if (event.key === "ArrowDown") {
              event.preventDefault();
              setIndex((current) =>
                Math.min(current + 1, Math.max(filtered.length - 1, 0)),
              );
            } else if (event.key === "ArrowUp") {
              event.preventDefault();
              setIndex((current) => Math.max(current - 1, 0));
            } else if (event.key === "Enter") {
              event.preventDefault();
              run(filtered[index]);
            }
          }}
          placeholder="Digite um comando…"
          aria-label="Buscar comando"
          className="h-11 min-w-0 flex-1 bg-transparent text-[13px] text-ink outline-none placeholder:text-muted"
        />
        <Kbd>Esc</Kbd>
      </div>

      <ul className="max-h-[340px] overflow-auto p-1">
        {filtered.map((command, commandIndex) => (
          <li key={command.id}>
            <button
              type="button"
              onClick={() => run(command)}
              onMouseEnter={() => setIndex(commandIndex)}
              className={cn(
                "flex w-full items-center gap-2 rounded-md px-2.5 py-2 text-left text-[13px] transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/60",
                commandIndex === index
                  ? "bg-accent/10 text-ink"
                  : "text-muted hover:bg-surface-2 hover:text-ink",
              )}
            >
              <span className="shrink-0 text-muted">{command.icon}</span>
              <span className="min-w-0 flex-1 truncate">{command.label}</span>
              {command.hint && (
                <span className="shrink-0 font-mono text-[10px] text-muted">
                  {command.hint}
                </span>
              )}
            </button>
          </li>
        ))}
        {filtered.length === 0 && (
          <li className="px-3 py-6 text-center text-[12px] text-muted">
            Nenhum comando encontrado.
          </li>
        )}
      </ul>
    </dialog>
  );
}
