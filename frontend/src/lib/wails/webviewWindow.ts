/**
 * Wails runtime wrapper: `@/lib/wails/webviewWindow`.
 * Minimal stub: there's only one webview per Wails window in v2, so
 * `getCurrentWebviewWindow()` returns the same window object.
 */
import { Window } from "./window";

export class WebviewWindow extends Window {
  async listen<T = unknown>(
    name: string,
    handler: (event: { event: string; payload: T }) => void,
  ): Promise<() => void> {
    const { EventsOn } = await import("#wails/runtime/runtime");
    const unsub = EventsOn(name, (payload: T) => {
      try {
        handler({ event: name, payload });
      } catch (e) {
        console.error(`webviewWindow listen(${name}) handler threw:`, e);
      }
    });
    return unsub;
  }

  async emit(name: string, payload?: unknown): Promise<void> {
    const { EventsEmit } = await import("#wails/runtime/runtime");
    EventsEmit(name, payload);
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