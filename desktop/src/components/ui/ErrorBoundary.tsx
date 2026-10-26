import { Component, type ErrorInfo, type ReactNode } from "react";
import { Button } from "./Button";

type State = { error: Error | null };

export class ErrorBoundary extends Component<
  { children: ReactNode },
  State
> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error("gg crash:", error, info.componentStack);
  }

  render() {
    if (!this.state.error) return this.props.children;

    return (
      <div className="flex h-screen flex-col items-center justify-center gap-3 bg-bg p-6 text-center text-ink">
        <h1 className="text-lg font-semibold">Algo quebrou na interface</h1>
        <p className="max-w-[52ch] font-mono text-[12px] text-danger">
          {this.state.error.message}
        </p>
        <Button variant="primary" onClick={() => window.location.reload()}>
          Recarregar
        </Button>
      </div>
    );
  }
}
