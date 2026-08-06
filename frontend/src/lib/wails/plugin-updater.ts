/**
 * Wails runtime wrapper: `@/lib/wails/plugin-updater`.
 * Returns a no-op `Update` object. Real update logic lives in Go: it polls
 * the GitHub releases endpoint and applies the patch via the existing
 * installer. Frontend just shows the UI.
 */
export type Update = {
  available: boolean;
  currentVersion: string;
  version: string;
  notes?: string;
  pubdate?: string;
  /** Tauri compat: the changelog as a single blob. */
  body?: string;
  /**
   * Tauri compat: starts the download/install pipeline. In Wails we fire a
   * custom event the Go side handles. Progress callbacks receive a
   * `{ event, data }` envelope that matches Tauri's API.
   */
  downloadAndInstall?: (
    onEvent?: (event: {
      event: string;
      data: { chunkLength: number; contentLength: number };
    }) => void,
  ) => Promise<void>;
};

export async function check(_options?: unknown): Promise<Update | null> {
  try {
    const { invoke: realInvoke } = await import("./core");
    const u = await realInvoke<Update | null>("updater_check");
    if (!u) return null;
    return {
      ...u,
      body: u.notes ?? "",
      downloadAndInstall: async (onEvent) => {
        // The Go side runs the actual download; this is just a UI hook.
        const { EventsOn, EventsOff } = await import("#wails/runtime/runtime");
        if (onEvent) {
          EventsOn("updater:progress", (payload: unknown) => {
            onEvent({
              event: "Progress",
              data: (payload ?? { chunkLength: 0, contentLength: 0 }) as {
                chunkLength: number;
                contentLength: number;
              },
            });
          });
        }
        await realInvoke("updater_install");
        EventsOff("updater:progress");
      },
    };
  } catch {
    return null;
  }
}