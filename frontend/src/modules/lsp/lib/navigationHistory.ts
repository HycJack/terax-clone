import { EditorView } from "@codemirror/view";
import { getLspNavigator } from "./navigator";
import { fileUriToPath } from "./uri";

/**
 * VS Code-style back/forward navigation history (Ctrl+- / Ctrl+Shift+-).
 *
 * Every successful LSP jump (definition, references, peek → open) records the
 * source position and the target position. Back walks to the source, forward
 * walks back to the target. Positions in other files are opened through the
 * shared navigator so the correct tab is focused.
 */

export type NavEntry = {
  path: string;
  line: number;
};

const MAX_ENTRIES = 200;

let entries: NavEntry[] = [];
let index = -1;

type Listener = (state: { canGoBack: boolean; canGoForward: boolean }) => void;
const listeners = new Set<Listener>();

function toPath(uriOrPath: string): string {
  if (uriOrPath.startsWith("file://")) {
    return fileUriToPath(uriOrPath) ?? uriOrPath;
  }
  return uriOrPath;
}

export function getNavState(): {
  canGoBack: boolean;
  canGoForward: boolean;
} {
  return { canGoBack: index > 0, canGoForward: index < entries.length - 1 };
}

/** Subscribe to history changes; returns an unsubscribe function. */
export function subscribe(listener: Listener): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

function notify(): void {
  const state = getNavState();
  for (const l of listeners) l(state);
}

function pushNavigation(path: string, line: number): void {
  const entry: NavEntry = { path: toPath(path), line };
  const current = entries[index];
  if (current && current.path === entry.path && current.line === entry.line) {
    return;
  }
  entries = entries.slice(0, index + 1);
  entries.push(entry);
  if (entries.length > MAX_ENTRIES) entries.shift();
  index = entries.length - 1;
  notify();
}

/** Current cursor line (1-based) of a view. */
export function cursorLine(view: EditorView): number {
  return view.state.doc.lineAt(view.state.selection.main.head).number;
}

/**
 * Record a jump from the cursor in `view` (in `fromUri`) to `toUri:toLine`
 * (`toLine` is 0-based). Call BEFORE performing the jump so the source
 * position is captured.
 */
export function recordJump(
  view: EditorView,
  fromUri: string,
  toUri: string,
  toLine: number,
): void {
  pushNavigation(fromUri, cursorLine(view));
  pushNavigation(toUri, toLine + 1);
}

function restore(
  view: EditorView | null,
  docUri: string,
  entry: NavEntry,
): boolean {
  if (view) {
    const path = toPath(docUri);
    if (entry.path === path) {
      const line = Math.max(1, Math.min(entry.line, view.state.doc.lines));
      const at = view.state.doc.line(line).from;
      view.dispatch({
        selection: { anchor: at },
        effects: EditorView.scrollIntoView(at, { y: "center" }),
      });
      view.focus();
      return true;
    }
  }
  const nav = getLspNavigator();
  if (!nav) return false;
  nav.openFile(entry.path, entry.line);
  return true;
}

/** Ctrl+- — walk back to the previous navigation position. */
export function goBack(view: EditorView | null, docUri: string): boolean {
  if (index <= 0) return false;
  index -= 1;
  notify();
  return restore(view, docUri, entries[index]);
}

/** Ctrl+Shift+- — walk forward to the next navigation position. */
export function goForward(view: EditorView | null, docUri: string): boolean {
  if (index >= entries.length - 1) return false;
  index += 1;
  notify();
  return restore(view, docUri, entries[index]);
}