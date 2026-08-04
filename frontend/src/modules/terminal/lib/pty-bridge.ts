import { invoke, Channel } from "@tauri-apps/api/core";
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

// Polling interval (ms) for PTY output when Wails events are unreliable.
const POLL_INTERVAL = 30;

// Active poll loops keyed by PTY id.
const pollLoops = new Map<number, { stop: () => void }>();

function startPollLoop(
  id: number,
  handlers: PtyHandlers,
  releaseHandlers: () => void,
): () => void {
  let running = true;
  const loop = async () => {
    while (running) {
      await new Promise((r) => setTimeout(r, POLL_INTERVAL));
      if (!running) break;
      try {
        const buf = await invoke<number[]>("pty_read_output", { id });
        if (buf && buf.length > 0) {
          handlers.onData(new Uint8Array(buf));
        }
      } catch {
        // PTY may have been closed; stop polling.
        running = false;
        releaseHandlers();
      }
    }
  };
  loop();
  return () => {
    running = false;
  };
}

export async function openPty(
  cols: number,
  rows: number,
  handlers: PtyHandlers,
  cwd?: string,
  blocks?: boolean,
  shell?: string,
): Promise<PtySession> {
  // Raw bytes — no base64/JSON round-trip; messages arrive as ArrayBuffer.
  const onData = new Channel<ArrayBuffer>();
  const onExit = new Channel<number>();

  let released = false;
  const noop = () => {};
  const releaseHandlers = () => {
    if (released) return;
    released = true;
    onData.onmessage = noop;
    onExit.onmessage = noop;
  };

  onData.onmessage = (buf) => handlers.onData(new Uint8Array(buf));
  onExit.onmessage = (code) => {
    handlers.onExit?.(code);
    releaseHandlers();
  };

  const id = await invoke<number>("pty_open", {
    cols,
    rows,
    cwd: cwd ?? null,
    workspace: currentWorkspaceEnv(),
    blocks: blocks ?? false,
    shell: shell ?? null,
    onData,
    onExit,
  });

  // Start polling loop as fallback (Wails EventsEmit unreliable in dev mode).
  const stopPoll = startPollLoop(id, handlers, releaseHandlers);
  pollLoops.set(id, { stop: stopPoll });

  let closed = false;
  const headers = { "x-pty-id": String(id) };

  return {
    id,
    // Raw bytes + id header: no JSON round-trip on the per-keystroke path.
    write: (data) =>
      invoke("pty_write", { id, data: Array.from(textEncoder.encode(data)) }, { headers }),
    resize: (c, r) => invoke("pty_resize", { id, cols: c, rows: r }),
    close: async () => {
      if (closed) return;
      closed = true;
      try {
        await invoke("pty_close", { id });
      } finally {
        releaseHandlers();
        const entry = pollLoops.get(id);
        if (entry) {
          entry.stop();
          pollLoops.delete(id);
        }
      }
    },
  };
}
