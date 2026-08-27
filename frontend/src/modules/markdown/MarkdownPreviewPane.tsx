import { MarkdownCode } from "@/components/ai-elements/markdown-code";
import { cn } from "@/lib/utils";
import { currentWorkspaceEnv } from "@/modules/workspace";
import { invoke } from "@/lib/wails/core";
import { useEffect, useState } from "react";
import { Streamdown } from "streamdown";
import { MarkdownViewToggle } from "./MarkdownViewToggle";

type ReadResult =
  | { kind: "text"; content: string; size: number }
  | { kind: "binary"; size: number }
  | { kind: "toolarge"; size: number; limit: number };

type Status =
  | { kind: "loading" }
  | { kind: "ready"; content: string }
  | { kind: "binary" }
  | { kind: "toolarge"; size: number; limit: number }
  | { kind: "error"; message: string };

type Props = {
  path: string;
  visible: boolean;
  onSetView: (mode: "rendered" | "raw") => void;
};

const components = { code: MarkdownCode };

export function MarkdownPreviewPane({ path, visible, onSetView }: Props) {
  const [status, setStatus] = useState<Status>({ kind: "loading" });

  useEffect(() => {
    let cancelled = false;
    const load = () => {
      invoke<ReadResult>("fs_read_file", {
        path,
        workspace: currentWorkspaceEnv(),
      })
        .then((res) => {
          if (cancelled) return;
          if (res.kind === "text") {
            // Only re-render when the content actually changed — the poll
            // below fires every second while visible.
            setStatus((prev) =>
              prev.kind === "ready" && prev.content === res.content
                ? prev
                : { kind: "ready", content: res.content },
            );
          } else if (res.kind === "binary") {
            setStatus({ kind: "binary" });
          } else {
            setStatus({ kind: "toolarge", size: res.size, limit: res.limit });
          }
        })
        .catch((e) => {
          if (!cancelled) setStatus({ kind: "error", message: String(e) });
        });
    };
    // Immediate load on open (and again when the tab becomes visible, so
    // edits made while it was hidden show up).
    load();
    if (!visible) return () => {
      cancelled = true;
    };
    // Poll while visible: the editor writes to the same file, and the
    // rendered view should follow without re-opening the tab.
    const timer = setInterval(load, 1000);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [path, visible]);

  return (
    <div
      className={cn(
        "relative flex h-full w-full flex-col overflow-hidden rounded-md border border-border/60 bg-background",
        !visible && "pointer-events-none",
      )}
    >
      <MarkdownViewToggle mode="rendered" onChange={onSetView} />
      <div className="flex-1 overflow-auto">
        <div className="px-8 py-6">
          {status.kind === "loading" && (
            <p className="text-[12px] text-muted-foreground">Loading…</p>
          )}
          {status.kind === "error" && (
            <p className="text-[12px] text-destructive">
              Failed to read file: {status.message}
            </p>
          )}
          {status.kind === "binary" && (
            <p className="text-[12px] text-muted-foreground">
              Binary file — cannot render as markdown.
            </p>
          )}
          {status.kind === "toolarge" && (
            <p className="text-[12px] text-muted-foreground">
              File is {status.size} bytes; limit {status.limit}.
            </p>
          )}
          {status.kind === "ready" && (
            <Streamdown
              className="select-text [&>*:first-child]:mt-0 [&>*:last-child]:mb-0"
              components={components}
              mode="static"
              parseIncompleteMarkdown={false}
            >
              {status.content}
            </Streamdown>
          )}
        </div>
      </div>
    </div>
  );
}
