import { StateEffect, StateField } from "@codemirror/state";
import { type EditorView, type Panel, showPanel } from "@codemirror/view";
import { invoke } from "@/lib/wails/core";
import { fileUriToPath } from "./uri";

/**
 * Alt+F12 peek definition — a VS Code-style inline preview of a definition
 * shown in a top panel. Enter/click "open" jumps to the real location,
 * Esc closes, and Arrow keys cycle through multiple definitions.
 */

export type PeekLocation = {
  uri: string;
  /** 0-based */
  line: number;
  character: number;
  label: string;
};

type PeekSpec = {
  locations: PeekLocation[];
  index: number;
  title: string;
  currentUri: string;
  onOpen: (loc: PeekLocation) => void;
};

export const setPeek = StateEffect.define<PeekSpec | null>();

const peekField = StateField.define<PeekSpec | null>({
  create: () => null,
  update(value, tr) {
    for (const e of tr.effects) {
      if (e.is(setPeek)) return e.value;
    }
    return value;
  },
  provide: (f) =>
    showPanel.from(f, (spec) => (spec ? (view) => createPeekPanel(view, spec) : null)),
});

export function openPeek(view: EditorView, spec: PeekSpec): void {
  view.dispatch({ effects: setPeek.of({ ...spec }) });
}

export function closePeek(view: EditorView): void {
  view.dispatch({ effects: setPeek.of(null) });
  view.focus();
}

type ReadResult =
  | { kind: "text"; content: string; size: number; mtime: number }
  | { kind: "binary"; size: number }
  | { kind: "toolarge"; size: number; limit: number };

const CONTEXT_BEFORE = 3;
const CONTEXT_AFTER = 4;

function renderCode(
  container: HTMLElement,
  content: string,
  line: number,
  langHint: string | null,
): void {
  const lines = content.split("\n");
  const start = Math.max(0, line - CONTEXT_BEFORE);
  const end = Math.min(lines.length, line + CONTEXT_AFTER + 1);
  const pre = document.createElement("pre");
  pre.className = "cm-lsp-peek-code";
  pre.dataset.lang = langHint ?? "";
  lines.slice(start, end).forEach((text, i) => {
    const row = document.createElement("div");
    row.className = "cm-lsp-peek-line";
    const no = document.createElement("span");
    no.className = "cm-lsp-peek-line-no";
    no.textContent = String(start + i + 1);
    const code = document.createElement("span");
    code.className = "cm-lsp-peek-line-code";
    code.textContent = text || " ";
    row.appendChild(no);
    row.appendChild(code);
    if (start + i === line) row.classList.add("cm-lsp-peek-line-active");
    pre.appendChild(row);
  });
  container.replaceChildren(pre);
}

function createPeekPanel(view: EditorView, spec: PeekSpec): Panel {
  const dom = document.createElement("div");
  dom.className = "cm-lsp-peek";
  const header = document.createElement("div");
  header.className = "cm-lsp-peek-header";
  const body = document.createElement("div");
  body.className = "cm-lsp-peek-body";

  const titleEl = document.createElement("span");
  titleEl.className = "cm-lsp-peek-title";
  const navEl = document.createElement("span");
  navEl.className = "cm-lsp-peek-nav";
  const openBtn = document.createElement("button");
  openBtn.type = "button";
  openBtn.textContent = "Open";
  const closeBtn = document.createElement("button");
  closeBtn.type = "button";
  closeBtn.textContent = "Close";
  header.append(titleEl, navEl, openBtn, closeBtn);
  dom.append(header, body);

  let pending = false;

  const load = async (index: number) => {
    if (pending) return;
    pending = true;
    try {
      const loc = spec.locations[index];
      titleEl.textContent = spec.title;
      if (loc.uri === spec.currentUri) {
        // Same document: read straight from the editor state.
        const lineNo = Math.min(loc.line + 1, view.state.doc.lines);
        const lineObj = view.state.doc.line(lineNo);
        const from = Math.max(1, lineObj.number - CONTEXT_BEFORE);
        const to = Math.min(view.state.doc.lines, lineObj.number + CONTEXT_AFTER);
        const content = view.state.doc.sliceString(
          view.state.doc.line(from).from,
          view.state.doc.line(to).to,
        );
        renderCode(body, content, loc.line - (from - 1), null);
        return;
      }
      const path = fileUriToPath(loc.uri);
      if (!path) {
        body.replaceChildren(createMessage("Cannot resolve location"));
        return;
      }
      const res = await invoke<ReadResult>("fs_read_file", { path });
      if (res.kind === "text") {
        renderCode(body, res.content, loc.line, null);
      } else {
        body.replaceChildren(createMessage(`File ${res.kind}`));
      }
    } finally {
      pending = false;
    }
  };

  const openCurrent = () => {
    const loc = spec.locations[spec.index];
    closePeek(view);
    spec.onOpen(loc);
  };

  const cycle = (delta: number) => {
    if (spec.locations.length <= 1) return;
    spec.index =
      (spec.index + delta + spec.locations.length) % spec.locations.length;
    openPeek(view, { ...spec });
  };

  openBtn.addEventListener("click", openCurrent);
  closeBtn.addEventListener("click", () => closePeek(view));

  dom.addEventListener("keydown", (e) => {
    if (e.key === "Escape") {
      e.preventDefault();
      closePeek(view);
    } else if (e.key === "Enter") {
      e.preventDefault();
      openCurrent();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      cycle(1);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      cycle(-1);
    }
  });

  const render = () => {
    navEl.textContent =
      spec.locations.length > 1
        ? ` (${spec.index + 1}/${spec.locations.length})`
        : "";
    void load(spec.index);
  };

  render();

  return {
    dom,
    top: true,
    mount: () => {
      dom.tabIndex = 0;
      dom.focus();
    },
  };
}

function createMessage(text: string): HTMLElement {
  const div = document.createElement("div");
  div.className = "cm-lsp-peek-message";
  div.textContent = text;
  return div;
}

export const peekPanel = peekField;