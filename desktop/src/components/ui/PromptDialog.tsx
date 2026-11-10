import { useEffect, useId, useRef, useState } from "react";
import { Button } from "./Button";
import { Input } from "./Input";

export function PromptDialog({
  open,
  title,
  description,
  placeholder,
  confirmLabel = "Confirmar",
  busy = false,
  allowEmpty = false,
  onSubmit,
  onClose,
}: {
  open: boolean;
  title: string;
  description?: string;
  placeholder?: string;
  confirmLabel?: string;
  busy?: boolean;
  allowEmpty?: boolean;
  onSubmit: (value: string) => void;
  onClose: () => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  const [value, setValue] = useState("");

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;
    if (open && !dialog.open) {
      setValue("");
      dialog.showModal();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open]);

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget && !busy) onClose();
      }}
      className="w-[min(92vw,460px)] rounded-xl border border-line bg-surface p-0 text-ink shadow-2xl shadow-black/60 backdrop:bg-black/70 backdrop:backdrop-blur-[2px]"
    >
      <form
        onSubmit={(event) => {
          event.preventDefault();
          if (!allowEmpty && !value.trim()) return;
          onSubmit(value.trim());
        }}
      >
        <div className="p-4">
          <h2 id={titleId} className="text-[14px] font-semibold text-ink">
            {title}
          </h2>
          {description && (
            <p className="mt-1 text-[12px] leading-relaxed text-muted">
              {description}
            </p>
          )}
          <Input
            autoFocus
            value={value}
            onChange={(event) => setValue(event.currentTarget.value)}
            placeholder={placeholder}
            aria-label={title}
            disabled={busy}
            className="mt-3 w-full bg-bg"
          />
        </div>
        <footer className="flex justify-end gap-2 border-t border-line bg-surface-2 px-4 py-3">
          <Button type="button" onClick={onClose} disabled={busy}>
            Cancelar
          </Button>
          <Button
            type="submit"
            variant="primary"
            disabled={busy || (!allowEmpty && !value.trim())}
          >
            {busy ? "Executando..." : confirmLabel}
          </Button>
        </footer>
      </form>
    </dialog>
  );
}
