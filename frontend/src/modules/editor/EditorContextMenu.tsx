import {
  ContextMenu,
  ContextMenuContent,
  ContextMenuItem,
  ContextMenuSeparator,
  ContextMenuShortcut,
  ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { useCallback, useState, type ReactNode } from "react";

export type EditorCmHandle = {
  path: string;
  runKeyBinding: (key: string) => boolean;
  getSelection: () => string | null;
  cursorWord: () => { from: number; to: number; text: string } | null;
};

type Props = {
  children: ReactNode;
  cm: EditorCmHandle;
};

export function EditorContextMenu({ children, cm }: Props) {
  const [nonce, setNonce] = useState(0);
  const [hasSelection, setHasSelection] = useState(false);
  const [actionsEnabled, setActionsEnabled] = useState(true);

  // Use a counter so the menu re-anchors on every right-click.
  const handleContextMenu = useCallback(() => {
    const word = cm.cursorWord();
    setActionsEnabled(word != null);
    const sel = cm.getSelection();
    setHasSelection(sel != null && sel.length > 0);
    setNonce((n) => n + 1);
  }, [cm]);

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild onContextMenu={handleContextMenu}>
        {children}
      </ContextMenuTrigger>
      <ContextMenuContent
        key={nonce}
        className="min-w-52 rounded-2xl p-1.5"
        onCloseAutoFocus={(e) => e.preventDefault()}
      >
        <ContextMenuItem
          className="rounded-xl px-2.5 py-1.5 text-xs gap-2"
          disabled={!actionsEnabled}
          onSelect={() => cm.runKeyBinding("F12")}
        >
          Go to Definition
          <ContextMenuShortcut>F12</ContextMenuShortcut>
        </ContextMenuItem>
        <ContextMenuItem
          className="rounded-xl px-2.5 py-1.5 text-xs gap-2"
          disabled={!actionsEnabled}
          onSelect={() => cm.runKeyBinding("Alt-F12")}
        >
          Peek Definition
          <ContextMenuShortcut>⌥F12</ContextMenuShortcut>
        </ContextMenuItem>
        <ContextMenuItem
          className="rounded-xl px-2.5 py-1.5 text-xs gap-2"
          disabled={!actionsEnabled}
          onSelect={() => cm.runKeyBinding("Shift-F12")}
        >
          Find References
          <ContextMenuShortcut>⇧F12</ContextMenuShortcut>
        </ContextMenuItem>
        <ContextMenuItem
          className="rounded-xl px-2.5 py-1.5 text-xs gap-2"
          disabled={!actionsEnabled}
          onSelect={() => cm.runKeyBinding("F2")}
        >
          Rename Symbol
          <ContextMenuShortcut>F2</ContextMenuShortcut>
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem
          className="rounded-xl px-2.5 py-1.5 text-xs gap-2"
          onSelect={() => cm.runKeyBinding("Shift-Alt-f")}
        >
          Format Document
          <ContextMenuShortcut>⇧⌥F</ContextMenuShortcut>
        </ContextMenuItem>
        <ContextMenuSeparator />
        <ContextMenuItem
          className="rounded-xl px-2.5 py-1.5 text-xs gap-2"
          onSelect={() => copyPath(cm.path)}
        >
          Copy Path
        </ContextMenuItem>
        <ContextMenuItem
          className="rounded-xl px-2.5 py-1.5 text-xs gap-2"
          onSelect={() => {
            const name = cm.path.split(/[\\/]/).pop() ?? cm.path;
            copyPath(name);
          }}
        >
          Copy File Name
        </ContextMenuItem>
        {hasSelection && (
          <ContextMenuItem
            className="rounded-xl px-2.5 py-1.5 text-xs gap-2"
            onSelect={() => {
              const sel = cm.getSelection();
              if (sel) void navigator.clipboard.writeText(sel);
            }}
          >
            Copy Selection
            <ContextMenuShortcut>⌘C</ContextMenuShortcut>
          </ContextMenuItem>
        )}
      </ContextMenuContent>
    </ContextMenu>
  );
}

async function copyPath(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    // Fallback for environments without clipboard API
    const ta = document.createElement("textarea");
    ta.value = text;
    ta.style.position = "fixed";
    ta.style.opacity = "0";
    document.body.appendChild(ta);
    ta.select();
    document.execCommand("copy");
    document.body.removeChild(ta);
  }
}
