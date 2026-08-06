/**
 * Wails runtime wrapper: `@/lib/wails/event`.
 * Maps Tauri event names onto the Wails runtime event bus.
 *
 * CRITICAL: Wails EventsEmit wraps extra args into a `data` array:
 *   EventsEmit("ev", {x:1}) → internal payload {name:"ev", data:[{x:1}]}
 *   notifyListeners then passes `data` (the array) to the callback.
 *
 * But Tauri's emit passes the payload directly — our listen shim must
 * unwrap the single-element array to keep existing Tauri consumers working.
 *
 * ALSO CRITICAL: Use Wails' per-listener unsubscribable returned by EventsOn
 * instead of calling EventsOff(name) which kills ALL listeners for that event.
 */
import { EventsOff, EventsOn, EventsOnce } from "#wails/runtime/runtime";

export type UnlistenFn = () => void;
export interface EventCallback<T> {
  (event: { event: string; id: number; payload: T }): void;
}

let listenSeq = 0;

export { EventsOff };

/**
 * `listen(name, handler)` — subscribes to a Wails event. Wails doesn't expose
 * incrementing event ids, so we mint our own per-listener counter.
 */
export async function listen<T = unknown>(
  name: string,
  handler: EventCallback<T>,
): Promise<UnlistenFn> {
  const id = ++listenSeq;
  // Wails passes `data` which is the extra-args array from EventsEmit.
  // Tauri emits with ONE extra arg (the payload), so the Wails array
  // contains one element. Unwrap it: use data[0] as the Tauri payload.
  const unsub = EventsOn(name, (data: unknown) => {
    try {
      const payload = (Array.isArray(data) && data.length === 1 ? data[0] : data) as T;
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
  const unsub = EventsOnce(name, (data: unknown) => {
    try {
      const payload = (Array.isArray(data) && data.length === 1 ? data[0] : data) as T;
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
  const { EventsEmit } = await import("#wails/runtime/runtime");
  EventsEmit(name, payload);
}
