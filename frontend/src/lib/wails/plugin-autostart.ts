/**
 * Wails runtime wrapper: `@/lib/wails/plugin-autostart`.
 * The OS-specific autostart registration is implemented in Go (writes the
 * per-platform Run key / .desktop file / LaunchAgent). This module is just
 * a thin RPC wrapper.
 */
async function invoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const { invoke: realInvoke } = await import("./core");
  return realInvoke<T>(cmd, args);
}

export async function enable(): Promise<void> {
  await invoke("autostart_enable");
}

export async function disable(): Promise<void> {
  await invoke("autostart_disable");
}

export async function isEnabled(): Promise<boolean> {
  return await invoke<boolean>("autostart_is_enabled");
}