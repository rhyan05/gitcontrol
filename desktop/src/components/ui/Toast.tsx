import { useCallback, useState, type ReactNode } from "react";
import { CheckCircle2, Info, X, XCircle } from "lucide-react";
import { cn } from "../../lib/cn";
import { IconButton } from "./IconButton";
import {
  ToastContext,
  type ToastApi,
  type ToastTone,
} from "./toast-context";

type Toast = {
  id: number;
  tone: ToastTone;
  message: string;
};

const tones: Record<ToastTone, { className: string; icon: ReactNode }> = {
  success: {
    className: "border-success/40 bg-success/10 text-success",
    icon: <CheckCircle2 size={15} />,
  },
  error: {
    className: "border-danger/40 bg-danger/10 text-danger",
    icon: <XCircle size={15} />,
  },
  info: {
    className: "border-line bg-surface-2 text-ink",
    icon: <Info size={15} />,
  },
};

export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);

  const dismiss = useCallback((id: number) => {
    setToasts((current) => current.filter((toast) => toast.id !== id));
  }, []);

  const push = useCallback(
    (tone: ToastTone, message: string) => {
      const id = Date.now() + Math.random();
      setToasts((current) => [...current, { id, tone, message }]);
      window.setTimeout(() => dismiss(id), 4500);
    },
    [dismiss],
  );

  const api: ToastApi = {
    success: (message) => push("success", message),
    error: (message) => push("error", message),
    info: (message) => push("info", message),
  };

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="pointer-events-none fixed bottom-4 right-4 z-50 flex w-[min(92vw,340px)] flex-col gap-2">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            role={toast.tone === "error" ? "alert" : "status"}
            className={cn(
              "pointer-events-auto flex items-start gap-2 rounded-lg border px-3 py-2 font-mono text-[12px] shadow-xl shadow-black/40",
              tones[toast.tone].className,
            )}
          >
            <span className="mt-0.5 shrink-0">{tones[toast.tone].icon}</span>
            <span className="min-w-0 flex-1 whitespace-pre-wrap break-words">
              {toast.message}
            </span>
            <IconButton
              label="Fechar aviso"
              icon={<X size={13} />}
              className="-mr-1 -mt-1 h-5 w-5"
              onClick={() => dismiss(toast.id)}
            />
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
