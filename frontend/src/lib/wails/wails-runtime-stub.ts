// Runtime wrapper for `@wailsio/runtime`. Rolldown (Vite 8) treats
// `.d.ts` files as type-only and won't allow value imports from them, so we
// route value imports through this `.ts` file. TypeScript still resolves
// the module via the d.ts for type checking.
//
// CRITICAL: Wails v3 `Events.On` delivers a `WailsEvent` object `{name, data}`
// to callbacks, but the existing codebase expects the raw event data directly.
// The wrappers below unwrap WailsEvent automatically so all consumers work
// without modification.
import { Events, Browser, Window } from "@wailsio/runtime";

// ── Helpers ────────────────────────────────────────────────────────────

/** Unwrap a WailsEvent to its raw `.data` payload. */
function unwrapWailsEvent(raw: unknown): unknown {
  if (raw && typeof raw === "object" && "data" in (raw as object)) {
    return (raw as Record<string, unknown>).data;
  }
  return raw;
}

// ── Events ─────────────────────────────────────────────────────────────

/** Subscribe to an event. Callbacks receive the unwrapped data (not WailsEvent). */
export function EventsOn(
  name: string,
  callback: (data: unknown) => void,
): () => void {
  return Events.On(name, (ev) => callback(unwrapWailsEvent(ev)));
}

export const EventsOff = Events.Off;

export function EventsOnce(
  name: string,
  callback: (data: unknown) => void,
): () => void {
  return Events.Once(name, (ev) => callback(unwrapWailsEvent(ev)));
}

export function EventsOnMultiple(
  name: string,
  callback: (data: unknown) => void,
  maxCallbacks: number,
): () => void {
  return Events.OnMultiple(name, (ev) => callback(unwrapWailsEvent(ev)), maxCallbacks);
}

export const EventsOffAll = Events.OffAll;
export const EventsEmit = Events.Emit;

// ── Browser ────────────────────────────────────────────────────────────

export const BrowserOpenURL = Browser.OpenURL;

// ── Window operations ──────────────────────────────────────────────────

export const WindowReload = () => Window.Reload();
export const WindowReloadApp = () => Window.ForceReload();
export const WindowSetTitle = (title: string) => Window.SetTitle(title);
export const WindowSetSize = (width: number, height: number) => Window.SetSize(width, height);
export const WindowSetPosition = (x: number, y: number) => Window.SetPosition(x, y);
export const WindowSetMinSize = (width: number, height: number) => Window.SetMinSize(width, height);
export const WindowSetMaxSize = (width: number, height: number) => Window.SetMaxSize(width, height);
export const WindowGetSize = () => Window.Size();
export const WindowGetPosition = () => Window.Position();
export const WindowMaximise = () => Window.Maximise();
export const WindowToggleMaximise = () => Window.ToggleMaximise();
export const WindowUnmaximise = () => Window.UnMaximise();
export const WindowIsMaximised = () => Window.IsMaximised();
export const WindowIsMinimised = () => Window.IsMinimised();
export const WindowIsFullscreen = () => Window.IsFullscreen();
export const WindowIsNormal = () => Window.Size().then(() => true).catch(() => false);
export const WindowFullscreen = () => Window.Fullscreen();
export const WindowUnfullscreen = () => Window.UnFullscreen();
export const WindowMinimise = () => Window.Minimise();
export const WindowUnminimise = () => Window.UnMinimise();
export const WindowHide = () => Window.Hide();
export const WindowShow = () => Window.Show();
export const WindowShowInactive = () => Window.Show();
export const WindowCenter = () => Window.Center();
