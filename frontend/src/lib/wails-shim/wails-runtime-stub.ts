// Runtime wrapper for `#wails/runtime/runtime`. Rolldown (Vite 8) treats
// `.d.ts` files as type-only and won't allow value imports from them, so we
// route value imports through this `.ts` file. TypeScript still resolves
// the module via the d.ts for type checking.
import * as r from "../../../wailsjs/runtime/runtime";

export const EventsOn = r.EventsOn;
export const EventsOff = r.EventsOff;
export const EventsOnce = r.EventsOnce;
export const EventsOnMultiple = r.EventsOnMultiple;
export const EventsOffAll = r.EventsOffAll;
export const EventsEmit = r.EventsEmit;
export const BrowserOpenURL = r.BrowserOpenURL;
export const LogPrint = r.LogPrint;
export const LogTrace = r.LogTrace;
export const LogDebug = r.LogDebug;
export const LogInfo = r.LogInfo;
export const LogWarning = r.LogWarning;
export const LogError = r.LogError;
export const LogFatal = r.LogFatal;
export const WindowReload = r.WindowReload;
export const WindowReloadApp = r.WindowReloadApp;
export const WindowSetTitle = r.WindowSetTitle;
export const WindowSetSize = r.WindowSetSize;
export const WindowSetPosition = r.WindowSetPosition;
export const WindowSetMinSize = r.WindowSetMinSize;
export const WindowSetMaxSize = r.WindowSetMaxSize;
export const WindowGetSize = r.WindowGetSize;
export const WindowGetPosition = r.WindowGetPosition;
// WindowGetScreenSize is not in Wails v2 runtime; alias to WindowGetSize.
export const WindowGetScreenSize: typeof r.WindowGetSize = r.WindowGetSize;
// WindowShowInactive is not in Wails v2 runtime; export it as a no-op alias
// for WindowShow so any frontend imports stay type-safe.
export const WindowMaximise = r.WindowMaximise;
export const WindowToggleMaximise = r.WindowToggleMaximise;
export const WindowUnmaximise = r.WindowUnmaximise;
export const WindowIsMaximised = r.WindowIsMaximised;
export const WindowIsMinimised = r.WindowIsMinimised;
export const WindowIsFullscreen = r.WindowIsFullscreen;
export const WindowIsNormal = r.WindowIsNormal;
export const WindowFullscreen = r.WindowFullscreen;
export const WindowUnfullscreen = r.WindowUnfullscreen;
export const WindowMinimise = r.WindowMinimise;
export const WindowUnminimise = r.WindowUnminimise;
export const WindowHide = r.WindowHide;
export const WindowShow = r.WindowShow;
export const WindowShowInactive = r.WindowShow;
export const WindowCenter = r.WindowCenter;