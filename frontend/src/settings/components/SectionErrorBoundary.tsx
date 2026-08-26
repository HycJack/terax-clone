import { Component, type ReactNode } from "react";

type Props = { children: ReactNode };
type State = { error: Error | null };

/**
 * Catches render errors inside a settings section so a single broken section
 * can't unmount the whole React tree (the app previously went fully blank).
 */
export class SectionErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error): void {
    console.error("[settings] section crashed:", error);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="flex flex-col gap-2 p-6 text-[12px]">
          <span className="font-medium text-destructive">
            This settings section failed to render.
          </span>
          <span className="font-mono text-[11px] text-muted-foreground">
            {String(this.state.error?.message ?? this.state.error)}
          </span>
          <button
            type="button"
            onClick={() => this.setState({ error: null })}
            className="mt-1 h-7 w-fit rounded-md border border-border px-3 text-[11.5px] hover:bg-muted"
          >
            Retry
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}