// Runtime implementation of the `#wails/runtime/window` module. This file
// is what Vite/rolldown actually bundles — the `.d.ts` declaration file
// just lists the function signatures. We re-export the Wails v2 window
// runtime functions and provide two extras (`WindowMinimise`, `WindowClose`)
// that don't exist in Wails v2 but are needed by the Tauri-style shim.

import { EventsEmit } from "../../../wailsjs/runtime/runtime";
import {
  WindowSetTitle as _WindowSetTitle,
  WindowMaximise as _WindowMaximise,
  WindowUnmaximise as _WindowUnmaximise,
  WindowToggleMaximise as _WindowToggleMaximise,
  WindowIsMaximised as _WindowIsMaximised,
  WindowIsFullscreen as _WindowIsFullscreen,
  WindowFullscreen as _WindowFullscreen,
  WindowUnfullscreen as _WindowUnfullscreen,
  WindowHide as _WindowHide,
  WindowShow as _WindowShow,
  WindowCenter as _WindowCenter,
  WindowReload as _WindowReload,
} from "../../../wailsjs/runtime/runtime";

export const WindowSetTitle: typeof _WindowSetTitle = _WindowSetTitle;
export const WindowMaximise: typeof _WindowMaximise = _WindowMaximise;
export const WindowUnmaximise: typeof _WindowUnmaximise = _WindowUnmaximise;
export const WindowToggleMaximise: typeof _WindowToggleMaximise = _WindowToggleMaximise;
export const WindowIsMaximised: typeof _WindowIsMaximised = _WindowIsMaximised;
export const WindowIsFullscreen: typeof _WindowIsFullscreen = _WindowIsFullscreen;
export const WindowFullscreen: typeof _WindowFullscreen = _WindowFullscreen;
export const WindowUnfullscreen: typeof _WindowUnfullscreen = _WindowUnfullscreen;
export const WindowHide: typeof _WindowHide = _WindowHide;
export const WindowShow: typeof _WindowShow = _WindowShow;
export const WindowCenter: typeof _WindowCenter = _WindowCenter;
export const WindowReload: typeof _WindowReload = _WindowReload;

export function WindowMinimise(): void {
  EventsEmit("wails:minimise");
}

export function WindowClose(): void {
  // When on the settings page, "close" means go back to the main page,
  // not quit the app. The settings page is loaded in the same webview
  // via `window.location.assign()` — there is no second window to close.
  if (window.location.pathname === "/settings.html") {
    // Preserve any search params the user may want (e.g. workspace hint).
    window.location.assign(window.location.origin || "/");
    return;
  }
  // Main page: emit the event so the Go side can call runtime.Quit.
  EventsEmit("wails:close");
}