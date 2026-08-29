/**
 * Wails runtime wrapper: `@/lib/wails/webviewWindow`.
 * Minimal stub: there's only one webview per Wails window in v2, so
 * `getCurrentWebviewWindow()` returns the same window object.
 */
import { Window } from "./window";
import { Events } from "@wailsio/runtime";

export class WebviewWindow extends Window {
  async listen<T = unknown>(
    name: string,
    handler: (event: { event: string; payload: T }) => void,
  ): Promise<() => void> {
    const unsub = Events.On(name, (ev) => {
      try {
        handler({ event: name, payload: ev.data as T });
      } catch (e) {
        console.error(`webviewWindow listen(${name}) handler threw:`, e);
      }
    });
    return unsub;
  }

  async emit(name: string, payload?: unknown): Promise<void> {
    Events.Emit(name, payload);
  }

  /** Tauri-only API: re-focuses the webview. Wails v2 lacks an equivalent;
   * we attempt to refocus the DOM window so any focused iframe regains
   * focus.
   */
  async setFocus(): Promise<void> {
    window.focus();
  }
}

export function getCurrentWebviewWindow(): WebviewWindow {
  return new WebviewWindow();
}
