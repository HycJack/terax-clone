/**
 * Tauri→Wails shim: `@tauri-apps/plugin-opener`.
 * - `openUrl`: opens external URLs in the default browser.
 * - `openPath`: opens a local path with the OS default.
 * - `revealItemInDir`: opens the OS file explorer at the given path.
 *
 * Wails exposes `BrowserOpenURL` for external URLs. For local paths we
 * delegate to the Go side, which knows the correct per-platform command.
 */
import { BrowserOpenURL } from "#wails/runtime/runtime";

async function invoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const { invoke: realInvoke } = await import("./core");
  return realInvoke<T>(cmd, args);
}

export async function openUrl(url: string, _with?: string): Promise<void> {
  // Wails BrowserOpenURL is fire-and-forget but errors silently on failure;
  // for cases that need explicit error reporting, fall through to backend.
  try {
    await BrowserOpenURL(url);
  } catch {
    try {
      await invoke("opener_open_url", { url });
    } catch (e) {
      console.warn("openUrl failed:", e);
    }
  }
}

export async function openPath(path: string, _with?: string): Promise<void> {
  await invoke("opener_open_path", { path });
}

export async function revealItemInDir(path: string): Promise<void> {
  await invoke("opener_reveal_item", { path });
}

export async function openFile(path: string, _with?: string): Promise<void> {
  await openPath(path);
}

export async function closeStandardWindows(): Promise<void> {
  /* not exposed by Wails at runtime */
}