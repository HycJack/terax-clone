/**
 * Tauri→Wails shim: `@tauri-apps/api/window`.
 * Wraps the Wails window API to expose the same `getCurrentWindow()`,
 * `Window.show/minimize/close/...` surface the frontend expects.
 */
import {
  WindowHide,
  WindowMaximise,
  WindowMinimise,
  WindowSetTitle,
  WindowToggleMaximise,
  WindowUnmaximise,
  WindowIsMaximised,
  WindowShow,
} from "#wails/runtime/runtime";

export type Theme = "light" | "dark";

/**
 * WebView2 doesn't always fire a JS `resize` event after a host-side
 * maximize in frameless mode. Dispatch one manually so absolute-positioned
 * UI (e.g. the AI mini chat) reclamp immediately.
 */
function dispatchResize() {
  window.dispatchEvent(new Event("resize"));
}

export class Window {
  label = "main";

  /** Tauri exposes this as a property; mimic the shape. */
  async show(): Promise<void> {
    await WindowShow();
  }

  async hide(): Promise<void> {
    await WindowHide();
  }

  async close(): Promise<void> {
    const { EventsEmit } = await import("#wails/runtime/runtime");
    EventsEmit("wails:close");
  }

  async minimize(): Promise<void> {
    await WindowMinimise();
  }

  async unminimize(): Promise<void> {
    // Wails has no dedicated unminimize; restore + show again.
    await WindowShow();
  }

  async maximize(): Promise<void> {
    await WindowMaximise();
    dispatchResize();
  }

  async unmaximize(): Promise<void> {
    await WindowUnmaximise();
    dispatchResize();
  }

  async toggleMaximize(): Promise<void> {
    await WindowToggleMaximise();
    dispatchResize();
  }

  async isMaximized(): Promise<boolean> {
    return await WindowIsMaximised();
  }

  async setTitle(title: string): Promise<void> {
    await WindowSetTitle(title);
  }

  /** Tauri event subscription. We use the native browser `resize` event
   * because Wails v2 doesn't auto-emit a `wails:resize` event — it only
   * resizes the WebView2 internally. The native `resize` event fires
   * reliably for maximize, unmaximize, and edge-drag resizes.
   */
  async onResized(_cb: () => void): Promise<() => void> {
    const handler = () => {
      try {
        _cb();
      } catch (e) {
        console.error("onResized handler threw:", e);
      }
    };
    window.addEventListener("resize", handler);
    return () => window.removeEventListener("resize", handler);
  }

  async onFocusChanged(_cb: (event: { payload: boolean }) => void): Promise<() => void> {
    const { EventsOn } = await import("#wails/runtime/runtime");
    const unsub = EventsOn("wails:focus", (focused: unknown) => {
      try {
        _cb({ payload: Boolean(focused) });
      } catch (e) {
        console.error("onFocusChanged handler threw:", e);
      }
    });
    return unsub;
  }

  /**
   * Tauri's `onCloseRequested` lets the webview intercept the close and
   * prompt the user. Wails doesn't expose this natively, so we fire a
   * synthetic event from the Go side whenever the user requests close.
   */
  async onCloseRequested(
    _cb: (event: { payload: unknown; preventDefault: () => void }) => void,
  ): Promise<() => void> {
    const { EventsOn } = await import("#wails/runtime/runtime");
    const unsub = EventsOn("wails:close-requested", (payload: unknown) => {
      try {
        _cb({
          payload,
          preventDefault: () => {
            // Best-effort cancel — emit a custom event the Go side
            // listens to. The actual cancellation must be implemented on
            // the Go side, which currently doesn't honour it.
            import("#wails/runtime/runtime").then(({ EventsEmit }) =>
              EventsEmit("wails:cancel-close"),
            );
          },
        });
      } catch (e) {
        console.error("onCloseRequested handler threw:", e);
      }
    });
    return unsub;
  }
}

export function getCurrentWindow(): Window {
  return new Window();
}

export function getAllWindows(): Window[] {
  return [new Window()];
}
