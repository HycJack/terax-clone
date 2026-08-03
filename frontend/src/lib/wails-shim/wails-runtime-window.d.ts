// Type declarations for `#wails/runtime/window` — a Wails v3 path that the
// Tauri-style shim imports but Wails v2 doesn't actually export. The actual
// implementations live in `wails-runtime-stub-runtime.ts` and
// `wails-runtime-stub-extras.ts`; we declare them here directly so the
// bundler can resolve the imports without following a re-export chain.

declare module "#wails/runtime/window" {
  export function WindowSetTitle(title: string): void;
  export function WindowMaximise(): void;
  export function WindowUnmaximise(): void;
  export function WindowToggleMaximise(): void;
  export function WindowIsMaximised(): Promise<boolean>;
  export function WindowIsFullscreen(): Promise<boolean>;
  export function WindowFullscreen(): void;
  export function WindowUnfullscreen(): void;
  export function WindowHide(): void;
  export function WindowShow(): void;
  export function WindowMinimise(): void;
  export function WindowClose(): void;
  export function WindowCenter(): void;
  export function WindowReload(): void;
}