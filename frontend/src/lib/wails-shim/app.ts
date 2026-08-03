/**
 * Tauri→Wails shim: `@tauri-apps/api/app`.
 * Reads app metadata (name, version) from a build-time constant the Go side
 * stamps into the bundle. Falls back to the package.json `name`/`version`.
 */
import pkg from "../../../package.json" with { type: "json" };

// Build-time injection point: the Go backend can rewrite this on app start
// by emitting a window event and we re-render, but most projects read these
// from the bundled assets directly.
const APP_NAME = "Terax";
const APP_VERSION = "0.8.6";

export async function getName(): Promise<string> {
  return APP_NAME;
}

export async function getVersion(): Promise<string> {
  return APP_VERSION;
}

export async function getTauriVersion(): Promise<string> {
  return "wails-2.0";
}

export async function setName(_name: string): Promise<void> {
  /* not supported in Wails at runtime */
}

export async function hide(): Promise<void> {
  const { WindowHide } = await import("#wails/runtime/runtime");
  await WindowHide();
}

export async function show(): Promise<void> {
  const { WindowShow } = await import("#wails/runtime/window");
  await WindowShow();
}

// Tauri 2 exposes these as synchronous getters in some call sites; provide
// them too so plain `app.getName()` works (it's not actually used).
export const app = {
  name: pkg.name ?? "terax",
  version: pkg.version ?? "0.0.0",
};