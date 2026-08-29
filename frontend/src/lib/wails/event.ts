/**
 * Wails runtime wrapper: `@/lib/wails/event`.
 * Maps Tauri event names onto the Wails v3 runtime event bus.
 *
 * CRITICAL: Wails v3 `Events.On` delivers a `WailsEvent` object `{name, data}`
 * to callbacks. This module unwraps it so consumers get the raw data directly.
 */
import { Events } from "@wailsio/runtime";

export type UnlistenFn = () => void;
export interface EventCallback<T> {
  (event: { event: string; id: number; payload: T }): void;
}

let listenSeq = 0;

/** Unwrap a WailsEvent to its raw `.data` payload. */
function unwrapWailsEvent(raw: unknown): unknown {
  if (raw && typeof raw === "object" && "data" in (raw as object)) {
    return (raw as Record<string, unknown>).data;
  }
  return raw;
}

export const EventsOff = Events.Off;

/**
 * `listen(name, handler)` — subscribes to a Wails event.
 */
export async function listen<T = unknown>(
  name: string,
  handler: EventCallback<T>,
): Promise<UnlistenFn> {
  const id = ++listenSeq;
  const unsub = Events.On(name, (ev) => {
    try {
      const payload = unwrapWailsEvent(ev) as T;
      handler({ event: name, id, payload });
    } catch (e) {
      console.error(`event handler for "${name}" threw:`, e);
    }
  });
  return unsub;
}

export async function once<T = unknown>(
  name: string,
  handler: EventCallback<T>,
): Promise<UnlistenFn> {
  const unsub = Events.Once(name, (ev) => {
    try {
      const payload = unwrapWailsEvent(ev) as T;
      handler({ event: name, id: 0, payload });
    } catch (e) {
      console.error(`event handler for "${name}" threw:`, e);
    }
  });
  return unsub;
}

export async function emit(
  name: string,
  payload?: unknown,
): Promise<void> {
  Events.Emit(name, payload);
}
