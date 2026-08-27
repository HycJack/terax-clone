// Simple-shell-style PTY bridge. The Go pump emits `pty:<id>` events onto
// the Wails event bus as envelopes of the form `{data: "<base64>"}`.
// We subscribe to that single event name per session and forward the
// decoded bytes to xterm. No Channels, no polling.
//
// The historical Wails event-bus dropouts (commits 10013e7, 86a08c0,
// 8c18fc8) are mitigated here by the underlying `gopty`/ConPTY pump
// running on a single goroutine that holds the master fd; this side
// simply replays events into xterm in the order they arrive.

import { invoke } from "@/lib/wails/core";
import { EventsOff, EventsOn } from "#wails/runtime/runtime";
import { currentWorkspaceEnv } from "@/modules/workspace";

const textEncoder = new TextEncoder();

export type PtyHandlers = {
  onData: (bytes: Uint8Array) => void;
  onExit?: (code: number) => void;
};

export type PtySession = {
  id: number;
  write: (data: string) => Promise<void>;
  resize: (cols: number, rows: number) => Promise<void>;
  close: () => Promise<void>;
};

function decodeEnvelope(raw: unknown): Uint8Array | null {
  // Two shapes reach us:
  //   1. {data: "<base64>"} — the envelope our Go pump emits.
  //   2. "<base64>" — a bare string (older fallback path).
  //   3. ArrayBuffer — raw bytes (defensive fallback).
  let b64: string | undefined;
  if (raw && typeof raw === "object" && !Array.isArray(raw)) {
    const data = (raw as Record<string, unknown>).data;
    if (typeof data === "string") b64 = data;
  } else if (typeof raw === "string") {
    b64 = raw;
  } else if (raw instanceof ArrayBuffer) {
    return new Uint8Array(raw);
  }
  if (!b64) return null;
  try {
    const bin = atob(b64);
    const ua = new Uint8Array(bin.length);
    for (let i = 0; i < bin.length; i++) ua[i] = bin.charCodeAt(i);
    return ua;
  } catch {
    return null;
  }
}

export async function openPty(
  cols: number,
  rows: number,
  handlers: PtyHandlers,
  cwd?: string,
  _blocks?: boolean,
  shell?: string,
): Promise<PtySession> {
  const id = await invoke<number>("pty_open", {
    cols,
    rows,
    cwd: cwd ?? null,
    workspace: currentWorkspaceEnv(),
    shell: shell ?? null,
  });

  // Subscribe to the dynamic event name AFTER the session is opened: the
  // event name embeds the session id, and we don't know it until the
  // backend responds.
  const dataEvent = `pty:${id}`;
  const exitEvent = `pty:exit:${id}`;

  const unsubData = EventsOn(dataEvent, (raw: unknown) => {
    const bytes = decodeEnvelope(raw);
    if (bytes) handlers.onData(bytes);
  });
  const unsubExit = EventsOn(exitEvent, (code: unknown) => {
    const c = typeof code === "number" ? code : Number(code) || 0;
    handlers.onExit?.(c);
  });

  // Listeners for `pty:<id>` are now registered. Only now is it safe to
  // start the backend pump — Wails' event bus has no buffering, so any
  // output emitted before this subscription is silently dropped (this is
  // what made the shell banner/prompt disappear on startup).
  try {
    await invoke("pty_start", { id });
  } catch (err) {
    // Unsubscribe and tear down the backend PTY we already opened, otherwise
    // this failure leaks the event listeners AND a live backend session
    // (openPtyWithRetry would then compound the leak on each failed spawn).
    try {
      unsubData();
      unsubExit();
      EventsOff(dataEvent);
      EventsOff(exitEvent);
    } catch {
      /* ignore */
    }
    try {
      await invoke("pty_close", { id });
    } catch {
      /* session may already be gone */
    }
    throw err;
  }

  let closed = false;
  return {
    id,
    write: async (data) => {
      if (closed) return;
      await invoke("pty_write", {
        id,
        data: Array.from(textEncoder.encode(data)),
      });
    },
    resize: (c, r) => invoke("pty_resize", { id, cols: c, rows: r }),
    close: async () => {
      if (closed) return;
      closed = true;
      try {
        unsubData();
        unsubExit();
        EventsOff(dataEvent);
        EventsOff(exitEvent);
      } catch {
        /* ignore */
      }
      try {
        await invoke("pty_close", { id });
      } catch {
        /* session may already be gone */
      }
    },
  };
}