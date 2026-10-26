import { useEffect, useId, useRef } from "react";
import { AlertTriangle } from "lucide-react";
import { Button } from "./Button";

export function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel = "Confirmar",
  busy = false,
  tone = "primary",
  onConfirm,
  onClose,
}: {
  open: boolean;
  title: string;
  description?: string;
  confirmLabel?: string;
  busy?: boolean;
  tone?: "primary" | "danger";
  onConfirm: () => void;
  onClose: () => void;
}) {
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  const descriptionId = useId();

  useEffect(() => {
    const dialog = dialogRef.current;
    if (!dialog) return;

    if (open && !dialog.open) {
      dialog.showModal();
    } else if (!open && dialog.open) {
      dialog.close();
    }
  }, [open]);

  return (
    <dialog
      ref={dialogRef}
      aria-labelledby={titleId}
      aria-describedby={description ? descriptionId : undefined}
      onCancel={(event) => {
        event.preventDefault();
        if (!busy) onClose();
      }}
      onClick={(event) => {
        if (event.target === event.currentTarget && !busy) onClose();
      }}
      className="w-[min(92vw,430px)] rounded-xl border border-line bg-surface p-0 text-ink shadow-2xl shadow-black/60 backdrop:bg-black/70 backdrop:backdrop-blur-[2px]"
    >
      <div className="flex gap-3 p-4">
        <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-warning/15 text-warning">
          <AlertTriangle size={16} />
        </span>
        <div className="min-w-0">
          <h2 id={titleId} className="text-[14px] font-semibold text-ink">
            {title}
          </h2>
          {description && (
            <p
              id={descriptionId}
              className="mt-1 text-[12px] leading-relaxed text-muted"
            >
              {description}
            </p>
          )}
        </div>
      </div>
      <footer className="flex justify-end gap-2 border-t border-line bg-surface-2 px-4 py-3">
        <Button autoFocus onClick={onClose} disabled={busy}>
          Cancelar
        </Button>
        <Button
          variant={tone === "danger" ? "danger" : "primary"}
          onClick={onConfirm}
          disabled={busy}
        >
          {busy ? "Executando…" : confirmLabel}
        </Button>
      </footer>
    </dialog>
  );
}
