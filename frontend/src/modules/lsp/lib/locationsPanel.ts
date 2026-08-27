import { StateEffect, StateField } from "@codemirror/state";
import { type EditorView, ViewPlugin, type ViewUpdate } from "@codemirror/view";

/**
 * VS Code-style quick-pick list for multi-result LSP responses (definitions,
 * references). Rendered as a floating overlay near the top of the editor;
 * ArrowUp/Down navigates, Enter/click picks, Escape closes.
 */

export type LocationItem = {
  uri: string;
  /** 0-based */
  line: number;
  character: number;
  label: string;
};

type PanelSpec = {
  title: string;
  items: LocationItem[];
  onPick: (item: LocationItem) => void;
};

export const setLocationList = StateEffect.define<PanelSpec | null>();

const locationsField = StateField.define<PanelSpec | null>({
  create: () => null,
  update(value, tr) {
    for (const e of tr.effects) {
      if (e.is(setLocationList)) return e.value;
    }
    return value;
  },
});

export function openLocationsPanel(view: EditorView, spec: PanelSpec): void {
  view.dispatch({ effects: setLocationList.of(spec) });
}

function closePanel(view: EditorView): void {
  view.dispatch({ effects: setLocationList.of(null) });
  view.focus();
}

class LocationPicker {
  private spec: PanelSpec | null = null;
  private active = 0;
  private rows: HTMLElement[] = [];
  private readonly dom: HTMLElement;
  private readonly header: HTMLElement;
  private readonly list: HTMLElement;

  constructor(view: EditorView) {
    this.dom = document.createElement("div");
    this.dom.className = "cm-lsp-locations";
    this.dom.style.display = "none";

    this.header = document.createElement("div");
    this.header.className = "cm-lsp-locations-header";
    this.list = document.createElement("ul");
    this.list.className = "cm-lsp-locations-list";
    this.list.tabIndex = 0;

    this.dom.appendChild(this.header);
    this.dom.appendChild(this.list);
    view.dom.appendChild(this.dom);
  }

  private render(view: EditorView, spec: PanelSpec): void {
    this.spec = spec;
    this.active = 0;
    this.dom.style.display = "block";
    this.header.textContent = `${spec.title} (${spec.items.length})`;

    this.list.replaceChildren();
    this.rows = spec.items.map((item, i) => {
      const li = document.createElement("li");
      li.textContent = item.label;
      li.addEventListener("mousedown", (e) => {
        e.preventDefault();
        e.stopPropagation();
        this.pick(view, i);
      });
      this.list.appendChild(li);
      return li;
    });
    this.highlight();
    this.list.focus();
  }

  private highlight(): void {
    this.rows.forEach((row, i) => {
      row.classList.toggle("cm-lsp-locations-active", i === this.active);
    });
    this.rows[this.active]?.scrollIntoView({ block: "nearest" });
  }

  private pick(view: EditorView, i: number): void {
    const spec = this.spec;
    if (!spec) return;
    const item = spec.items[i];
    closePanel(view);
    spec.onPick(item);
  }

  private onKey(view: EditorView, e: KeyboardEvent): boolean {
    const spec = this.spec;
    if (!spec) return false;
    if (e.key === "ArrowDown") {
      this.active = Math.min(this.active + 1, spec.items.length - 1);
      this.highlight();
    } else if (e.key === "ArrowUp") {
      this.active = Math.max(this.active - 1, 0);
      this.highlight();
    } else if (e.key === "Enter") {
      this.pick(view, this.active);
    } else if (e.key === "Escape") {
      closePanel(view);
    } else {
      return false;
    }
    e.preventDefault();
    return true;
  }

  onKeydown(view: EditorView, e: KeyboardEvent): boolean {
    return this.onKey(view, e);
  }

  update(update: ViewUpdate): void {
    const next = update.state.field(locationsField);
    if (next === this.spec) return;
    if (next) {
      this.render(update.view, next);
    } else {
      this.spec = null;
      this.dom.style.display = "none";
    }
  }

  destroy(): void {
    this.dom.remove();
  }
}

export const locationsPanel = [
  locationsField,
  ViewPlugin.fromClass(LocationPicker, {
    eventHandlers: {
      keydown(e, view) {
        // Return true only when the picker handled the key, so handled keys
        // stop propagation and never also move the editor cursor.
        return this.onKeydown(view, e);
      },
    },
  }),
];