import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useChat, type UIMessage } from "@ai-sdk/react";
import {
	Maximize01Icon,
	Minimize01Icon,
	StopCircleIcon,
} from "@hugeicons/core-free-icons";
import { HugeiconsIcon } from "@hugeicons/react";
import { useEffect, useMemo } from "react";
import { useChatStore } from "../store/chatStore";
import { getOrCreateChat } from "../store/chatRuntime";
import { usePlanStore } from "../store/planStore";
import { getLspNavigator } from "@/modules/lsp/lib/navigator";
import { AiChatView } from "./AiChat";
import { PlanDiffReview } from "./PlanDiffReview";
import { TodoStrip } from "./TodoStrip";
import { Spinner } from "@/components/ui/spinner";

export function AiSidebar({ onExpand, onCollapse }: { onExpand?: () => void; onCollapse?: () => void }) {
  const sessionId = useChatStore((s) => s.activeSessionId);
  const focusInput = useChatStore((s) => s.focusInput);

  const chat = useMemo(
    () => (sessionId ? getOrCreateChat(sessionId) : null),
    [sessionId],
  );
  const helpers = useChat<UIMessage>({ chat: chat! });
  const isBusy =
    helpers.status === "submitted" || helpers.status === "streaming";
  const step = useChatStore((s) => s.agentMeta.step);

  // Auto-focus input when sidebar opens
  useEffect(() => {
    if (sessionId) {
      const timer = setTimeout(() => focusInput(), 100);
      return () => clearTimeout(timer);
    }
  }, [sessionId, focusInput]);

  if (!sessionId) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3 bg-card p-6 text-center">
        <div className="text-sm font-medium text-foreground">
          AI Assistant
        </div>
        <div className="text-xs text-muted-foreground">
          Start a conversation from the status bar or use
          <kbd className="mx-1 rounded border border-border/60 bg-muted/40 px-1 py-0.5 font-mono text-[10px]">
            Ctrl+L
          </kbd>
        </div>
        <Button
          variant="outline"
          size="sm"
          className="mt-2 text-xs"
          onClick={() => {
            useChatStore.getState().openPanel();
            useChatStore.getState().newSession();
          }}
        >
          New Chat
        </Button>
      </div>
    );
  }

  return (
    <div className="flex h-full flex-col bg-card">
      {/* Header */}
      <div className="flex shrink-0 items-center gap-2 border-b border-border/60 px-3 py-2">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <span className="truncate text-xs font-medium">AI Assistant</span>
          {isBusy && (
            <div className="flex items-center gap-1.5 text-[10px] text-muted-foreground">
              <Spinner />
              <span className="truncate">{step ?? "Thinking…"}</span>
            </div>
          )}
        </div>
        <div className="flex shrink-0 items-center gap-0.5">
          {helpers.messages.length > 0 && (
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={helpers.stop}
              disabled={!isBusy}
              aria-label="Stop generation"
              className="text-muted-foreground hover:text-foreground"
            >
              <HugeiconsIcon icon={StopCircleIcon} size={14} strokeWidth={1.75} />
            </Button>
          )}
          {onExpand && (
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={onExpand}
              aria-label="Expand to full panel"
              className="text-muted-foreground hover:text-foreground"
            >
              <HugeiconsIcon icon={Maximize01Icon} size={13} strokeWidth={1.75} />
            </Button>
          )}
          {onCollapse && (
            <Button
              variant="ghost"
              size="icon-xs"
              onClick={onCollapse}
              aria-label="Collapse sidebar"
              className="text-muted-foreground hover:text-foreground"
            >
              <HugeiconsIcon icon={Minimize01Icon} size={13} strokeWidth={1.75} />
            </Button>
          )}
        </div>
      </div>

      {/* Plan mode strip */}
      <PlanModeStrip />

      {/* Chat content */}
      <div className="flex min-h-0 flex-1 flex-col">
        <ScrollArea className="flex-1">
          <div className="p-3">
            {helpers.messages.length === 0 ? (
              <EmptySidebarState onPick={focusInput} />
            ) : (
              <div className="[&_.text-sm]:text-[12px] [&_p]:leading-relaxed">
                <AiChatView
                  messages={helpers.messages}
                  status={helpers.status}
                  error={helpers.error}
                  clearError={helpers.clearError}
                  addToolApprovalResponse={helpers.addToolApprovalResponse}
                  stop={helpers.stop}
                  onOpenContentHit={(...args) =>
                    getLspNavigator()?.openFile(...args)
                  }
                />
              </div>
            )}
          </div>
        </ScrollArea>
      </div>

      {/* Todo strip */}
      <TodoStrip sessionId={sessionId} />

      <PlanDiffReview />
    </div>
  );
}

function PlanModeStrip() {
  const active = usePlanStore((s) => s.active);
  const queueLen = usePlanStore((s) => s.queue.length);
  const disable = usePlanStore((s) => s.disable);
  if (!active) return null;
  return (
    <div className="flex shrink-0 items-center gap-2 border-b border-border/40 bg-muted/40 px-3 py-1.5">
      <span className="size-1.5 shrink-0 rounded-full bg-amber-500" />
      <span className="text-[11px] font-medium text-foreground">Plan mode</span>
      <span className="text-[11px] text-muted-foreground">
        {queueLen > 0 ? `· ${queueLen} queued` : "· no edits queued"}
      </span>
      <span className="flex-1" />
      <button
        type="button"
        onClick={() => disable()}
        className="rounded px-1.5 py-0.5 text-[10.5px] text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
      >
        Exit
      </button>
    </div>
  );
}

function EmptySidebarState({ onPick }: { onPick: () => void }) {
  const suggestions = [
    { text: "Explain the last error", hint: "Read terminal output" },
    { text: "Generate a unit test", hint: "For the current file" },
    { text: "Fix the code", hint: "Auto-detect issues" },
    { text: "Refactor this function", hint: "Improve readability" },
  ];

  return (
    <div className="flex flex-col gap-3 py-4">
      <div className="text-center text-xs text-muted-foreground">
        Ask anything about your code
      </div>
      <div className="flex flex-col gap-1.5">
        {suggestions.map((s) => (
          <button
            key={s.text}
            type="button"
            onClick={() => {
              onPick();
              // The input will be focused; user can type or we could pre-fill
            }}
            className="rounded-md border border-border/50 bg-muted/20 px-3 py-2 text-left transition-colors hover:bg-muted/50"
          >
            <div className="text-xs font-medium text-foreground">{s.text}</div>
            <div className="text-[10px] text-muted-foreground">{s.hint}</div>
          </button>
        ))}
      </div>
    </div>
  );
}
