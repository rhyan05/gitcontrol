import { createContext, useContext } from "react";

export type ToastTone = "success" | "error" | "info";

export type ToastApi = {
  success: (message: string) => void;
  error: (message: string) => void;
  info: (message: string) => void;
};

export const ToastContext = createContext<ToastApi | null>(null);

export function useToast(): ToastApi {
  const api = useContext(ToastContext);
  if (!api) {
    throw new Error("useToast precisa estar dentro de ToastProvider");
  }
  return api;
}
