import { EditorView } from "@codemirror/view";

/**
 * Shared styling for LSP overlay UI rendered inside the editor DOM:
 * peek panel, quick-pick location list and hover-doc links. Colors follow
 * the app's theme tokens (--background, --border, --primary, ...).
 */
export const lspUiTheme = EditorView.theme({
  // Clickable links inside hover documentation.
  ".cm-lsp-doc-link": {
    color: "var(--primary)",
    textDecoration: "underline",
    textUnderlineOffset: "2px",
    cursor: "pointer",
  },
  ".cm-lsp-doc-link:hover": {
    opacity: "0.85",
  },

  // Peek definition panel (top).
  ".cm-lsp-peek": {
    background: "var(--background)",
    border: "1px solid var(--border)",
    borderTop: "none",
    color: "var(--foreground)",
    boxShadow: "0 6px 24px rgba(0,0,0,0.35)",
    fontFamily: "var(--font-mono, ui-monospace, SFMono-Regular, monospace)",
    fontSize: "12px",
    lineHeight: "1.55",
  },
  ".cm-lsp-peek-header": {
    display: "flex",
    alignItems: "center",
    gap: "8px",
    padding: "4px 10px",
    background: "var(--muted)",
    borderBottom: "1px solid var(--border)",
    color: "var(--muted-foreground)",
    fontSize: "11px",
  },
  ".cm-lsp-peek-title": {
    flex: "1",
    whiteSpace: "nowrap",
    overflow: "hidden",
    textOverflow: "ellipsis",
  },
  ".cm-lsp-peek-header button": {
    border: "1px solid var(--border)",
    background: "transparent",
    color: "var(--foreground)",
    borderRadius: "5px",
    padding: "1px 8px",
    fontSize: "11px",
    cursor: "pointer",
  },
  ".cm-lsp-peek-header button:hover": {
    background: "var(--accent)",
  },
  ".cm-lsp-peek-body": {
    overflow: "auto",
    maxHeight: "220px",
  },
  ".cm-lsp-peek-code": {
    margin: "0",
    padding: "4px 0",
    overflow: "hidden",
  },
  ".cm-lsp-peek-line": {
    display: "flex",
    gap: "10px",
    padding: "0 10px",
    whiteSpace: "pre",
  },
  ".cm-lsp-peek-line-no": {
    color: "var(--muted-foreground)",
    userSelect: "none",
    minWidth: "2.2em",
    textAlign: "right",
  },
  ".cm-lsp-peek-line-active": {
    background: "color-mix(in oklch, var(--primary) 18%, transparent)",
  },
  ".cm-lsp-peek-message": {
    padding: "10px",
    color: "var(--muted-foreground)",
  },

  // Quick-pick location list overlay.
  ".cm-lsp-locations": {
    position: "absolute",
    top: "8px",
    left: "50%",
    transform: "translateX(-50%)",
    zIndex: "10",
    width: "min(520px, calc(100% - 24px))",
    background: "var(--popover, var(--background))",
    border: "1px solid var(--border)",
    borderRadius: "10px",
    boxShadow: "0 10px 34px rgba(0,0,0,0.4)",
    color: "var(--foreground)",
    overflow: "hidden",
  },
  ".cm-lsp-locations-header": {
    padding: "6px 12px",
    borderBottom: "1px solid var(--border)",
    color: "var(--muted-foreground)",
    fontSize: "11px",
    fontWeight: "600",
  },
  ".cm-lsp-locations-list": {
    margin: "0",
    padding: "4px",
    listStyle: "none",
    maxHeight: "260px",
    overflow: "auto",
  },
  ".cm-lsp-locations-list li": {
    padding: "5px 10px",
    borderRadius: "6px",
    cursor: "pointer",
    fontSize: "12px",
    fontFamily: "var(--font-mono, ui-monospace, SFMono-Regular, monospace)",
    whiteSpace: "nowrap",
    overflow: "hidden",
    textOverflow: "ellipsis",
  },
  ".cm-lsp-locations-list li.cm-lsp-locations-active": {
    background: "color-mix(in oklch, var(--primary) 22%, transparent)",
    color: "var(--foreground)",
  },
});