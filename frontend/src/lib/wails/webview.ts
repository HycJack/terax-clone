/**
 * Wails runtime wrapper: `@/lib/wails/webview`.
 * Tauri webview has separate APIs from window; in Wails v2 they overlap.
 */
import { getCurrentWebviewWindow } from "./webviewWindow";

export class Webview {
  label = "main";
  async listen<T = unknown>(
    name: string,
    handler: (event: { event: string; payload: T }) => void,
  ): Promise<() => void> {
    return getCurrentWebviewWindow().listen(name, handler);
  }
  async emit(name: string, payload?: unknown): Promise<void> {
    return getCurrentWebviewWindow().emit(name, payload);
  }

  /**
   * Tauri exposes `onDragDropEvent` for native file drops. Wails v2 only
   * fires a string event name on the webview side; we surface the raw
   * HTML5 drag/drop events as a polyfill so the frontend code can keep
   * working unchanged.
   */
  async onDragDropEvent(
    _cb: (
      event:
        | { payload: { type: "enter"; paths: string[]; position: { x: number; y: number } } }
        | { payload: { type: "over"; position: { x: number; y: number } } }
        | { payload: { type: "drop"; paths: string[]; position: { x: number; y: number } } }
        | { payload: { type: "leave" } },
    ) => void,
  ): Promise<() => void> {
    const onDragOver = (e: DragEvent) => {
      e.preventDefault();
      _cb({ payload: { type: "over", position: { x: e.clientX, y: e.clientY } } });
    };
    const onDragLeave = () => {
      _cb({ payload: { type: "leave" } });
    };
    const onDrop = (e: DragEvent) => {
      e.preventDefault();
      const files = e.dataTransfer?.files;
      const paths: string[] = [];
      if (files) {
        for (let i = 0; i < files.length; i++) {
          const f = files[i] as File & { path?: string };
          if (f.path) paths.push(f.path);
        }
      }
      _cb({
        payload: {
          type: "drop",
          paths,
          position: { x: e.clientX, y: e.clientY },
        },
      });
    };
    window.addEventListener("dragover", onDragOver);
    window.addEventListener("dragleave", onDragLeave);
    window.addEventListener("drop", onDrop);
    return () => {
      window.removeEventListener("dragover", onDragOver);
      window.removeEventListener("dragleave", onDragLeave);
      window.removeEventListener("drop", onDrop);
    };
  }
}

export function getCurrentWebview(): Webview {
  return new Webview();
}