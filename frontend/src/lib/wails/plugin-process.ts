/**
 * Wails runtime wrapper: `@/lib/wails/plugin-process`.
 * Minimal: `relaunch()` is the only call we exercise (Updater dialog). The
 * Go side knows the executable path & arguments for self-restart.
 */
async function invoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const { invoke: realInvoke } = await import("./core");
  return realInvoke<T>(cmd, args);
}

export async function relaunch(): Promise<void> {
  await invoke("process_relaunch");
}

export async function exit(code = 0): Promise<void> {
  await invoke("process_exit", { code });
}

export async function kill(_pid: number): Promise<void> {
  /* unused */
}