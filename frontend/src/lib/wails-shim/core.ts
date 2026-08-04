/**
 * Tauri→Wails shim: `@tauri-apps/api/core`.
 *
 * Bridges the existing Tauri command surface (`invoke`, `Channel`,
 * `convertFileSrc`) onto the Wails runtime. Frontend code that imports from
 * `@tauri-apps/api/core` is left untouched.
 */
import { EventsOff, EventsOn } from "#wails/runtime/runtime";
import * as App from "../../../wailsjs/go/main/App";

// One counter is enough for channel/event-name uniqueness within a session.
let channelSeq = 0;
function nextChannelId(prefix: string): string {
  channelSeq += 1;
  return `${prefix}-${Date.now().toString(36)}-${channelSeq}`;
}

/**
 * Tauri `Channel<T>` equivalent. Stores a callback, lets the Go side push
 * values via a unique event name. We register the listener here so JS code
 * can keep using `onmessage` semantics.
 */
export class Channel<T = unknown> {
  onmessage: ((data: T) => void) | null = null;
  readonly eventName: string;
  private cancelled = false;

  constructor() {
    this.eventName = nextChannelId("ch");
  }

  // Internal: subscribe to backend events. Called from `invoke()` when it
  // detects a Channel argument and passes the channel instance through.
  _attach(prefix: string): string {
    const name = `${prefix}:${this.eventName}`;
    EventsOn(name, (data: unknown) => {
      if (this.cancelled) return;
      // Wails wraps EventsEmit extra args into a data array:
      //   EventsEmit(ctx, name, b64) → JS callback receives [b64]
      // Unwrap the first element.
      const raw = Array.isArray(data) && data.length === 1 ? data[0] : data;
      if (typeof raw === "string") {
        try {
          const bin = atob(raw);
          const bytes = new Uint8Array(bin.length);
          for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
          this.onmessage?.(bytes.buffer as T);
        } catch (e) {
          console.error("[terax] Channel._attach: atob failed:", e);
        }
      } else {
        this.onmessage?.(raw as T);
      }
    });
    return name;
  }

  _detach(eventName: string): void {
    this.cancelled = true;
    try {
      EventsOff(eventName);
    } catch {
      /* ignore */
    }
  }
}

/**
 * Look up a bound Go method by command name. Generated bindings live under
 * `wailsjs/go/main/App` (App struct) with PascalCase method names; the
 * frontend calls them using Tauri-style snake_case so we convert
 * `pty_open` → `PtyOpen` here.
 */
type AnyFn = (...args: unknown[]) => Promise<unknown>;

/**
 * Tauri command names use snake_case (git_pull_ff_only, lsp_host_pid) but
 * our Wails binding is PascalCase (GitPullFFOnly, LspHostPID). The naive
 * toUpperCase translation strips abbreviation casing (FF -> Ff, URL -> Url,
 * PID -> Pid). This map restores canonical PascalCase for any segment that
 * contains an initialism, so the Go method is found correctly.
 */
const SNAKE_PASCAL_OVERRIDES: Record<string, string> = {
  ff: "FF",
  url: "URL",
  pid: "PID",
};

const snakeToPascal = (s: string) =>
  s
    .split("_")
    .map((seg) => {
      // Preserve canonical PascalCase for initialism segments that the
      // naive `s[0].toUpperCase()` translation would mishandle (FF, URL,
      // PID, etc.). When a segment matches an abbreviation, the override
      // replaces it entirely; any trailing characters after the matched
      // prefix are preserved verbatim so mixed cases like `UrlStuff` still
      // degrade gracefully.
      const lower = seg.toLowerCase();
      for (const [abbr, replacement] of Object.entries(SNAKE_PASCAL_OVERRIDES)) {
        if (lower === abbr) return replacement;
        if (lower.startsWith(abbr) && lower.length > abbr.length) {
          return replacement + seg.slice(abbr.length);
        }
      }
      return seg ? seg[0].toUpperCase() + seg.slice(1) : "";
    })
    .join("");

function resolveCommand(name: string): AnyFn | undefined {
  const appMethods = App as unknown as Record<string, AnyFn>;
  // Exact match first (handles PascalCase methods we may have added).
  if (typeof appMethods[name] === "function") return appMethods[name];
  // Fallback: snake_case → PascalCase.
  if (name.includes("_")) {
    const pascal = snakeToPascal(name);
    if (typeof appMethods[pascal] === "function") return appMethods[pascal];
  }
  return undefined;
}

/**
 * Maps each Tauri command to the Go method that implements it. Some Tauri
 * commands need side effects (channel subscription, event setup) before the
 * underlying Go call — those are listed here.
 */
const SPECIAL: Record<
  string,
  (args: Record<string, unknown>, headers: Record<string, string>) => unknown
> = {
  pty_open: (args) => {
    const onData = args.onData as Channel<ArrayBuffer> | undefined;
    const onExit = args.onExit as Channel<number> | undefined;
    if (onData) args.onDataEvent = onData._attach("pty:data");
    if (onExit) args.onExitEvent = onExit._attach("pty:exit");
    return args;
  },
  lsp_spawn: (args) => {
    const onMessage = args.onMessage as Channel<ArrayBuffer> | undefined;
    const onExit = args.onExit as Channel<unknown> | undefined;
    if (onMessage) args.onMessageEvent = onMessage._attach("lsp:msg");
    if (onExit) args.onExitEvent = onExit._attach("lsp:exit");
    return args;
  },
  ai_http_stream: (args) => {
    const onEvent = args.onEvent as Channel<unknown> | undefined;
    if (onEvent) args.onEventEvent = onEvent._attach("net:ai");
    return args;
  },
  agent_enable_hooks: (args) => {
    const hooksReady = args.onHooksReady as Channel<unknown> | undefined;
    if (hooksReady) args.onHooksReadyEvent = hooksReady._attach("agent:ready");
    return args;
  },
};

function detachChannels(cmd: string, args: Record<string, unknown>): void {
  if (cmd === "pty_open") {
    const a = args as Record<string, unknown>;
    (a.onData as Channel | undefined)?._detach(String(a.onDataEvent ?? ""));
    (a.onExit as Channel | undefined)?._detach(String(a.onExitEvent ?? ""));
  } else if (cmd === "lsp_spawn") {
    const a = args as Record<string, unknown>;
    (a.onMessage as Channel | undefined)?._detach(String(a.onMessageEvent ?? ""));
    (a.onExit as Channel | undefined)?._detach(String(a.onExitEvent ?? ""));
  } else if (cmd === "ai_http_stream") {
    const a = args as Record<string, unknown>;
    (a.onEvent as Channel | undefined)?._detach(String(a.onEventEvent ?? ""));
  } else if (cmd === "agent_enable_hooks") {
    const a = args as Record<string, unknown>;
    (a.onHooksReady as Channel | undefined)?._detach(String(a.onHooksReadyEvent ?? ""));
  }
}

/**
 * Some Go methods take a single positional argument (`PtyShellName(id int)`,
 * `FsCanonicalize(path string)`) instead of a struct. Wails generates a
 * matching `PtyShellName(arg1: number)` binding; passing our full payload
 * object would fail JSON unmarshalling. This map tells the shim which key
 * to lift out of the payload before dispatching.
 *
 * Commands NOT listed here are assumed to take a struct arg (the whole
 * payload is forwarded).
 */
const SINGLE_ARG: Record<string, string> = {
  // fs
  fs_canonicalize: "path",
  fs_read_file: "path",
  fs_stat: "path",
  // pty
  pty_close: "id",
  pty_has_foreground_process: "id",
  pty_has_foreground_job: "id",
  pty_shell_name: "id",
  // lsp
  lsp_detect: "command",
  lsp_kill: "id",
  // NOTE: `git_resolve_repo` is a struct-arg command (`GitResolveRepoArgs{Cwd}`),
  // not single-positional — keep it OUT of this map so the whole payload
  // crosses the bridge intact.
  // `workspace_authorize` is also a struct-arg command.
  // `lsp_resolve_root` takes (path, []markers) — keep it OUT as well.
};

export type InvokeOptions = { headers?: Record<string, string> };

/**
 * Drop keys that JS-only objects (Channel, functions) can't traverse through
 * Wails' JSON serializer. The Channel has been replaced with its event name
 * via `SPECIAL` already; this is just a defensive sweep.
 */
function serializeArgs(args: Record<string, unknown>): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(args)) {
    if (v instanceof Channel) continue;
    if (typeof v === "function") continue;
    out[k] = v;
  }
  return out;
}

/**
 * Mirrors `invoke()` from `@tauri-apps/api/core`. Args is the named-args bag
 * the Tauri code uses; we strip JS-only fields, resolve the Go method, and
 * dispatch.
 */
export async function invoke<T = unknown>(
  cmd: string,
  args?: Record<string, unknown>,
  opts: InvokeOptions = {},
): Promise<T> {
  const headers = opts.headers ?? {};
  const raw = args ?? {};

  // Special transforms: register Channel listeners before invoking.
  if (SPECIAL[cmd]) {
    SPECIAL[cmd](raw, headers);
  }

  // Strip JS-only fields before crossing the Wails boundary.
  const payload = serializeArgs(raw);

  const fn = resolveCommand(cmd);
  if (!fn) {
    throw new Error(
      `wails-shim: no Go binding for command "${cmd}". Add a method to internal/ backend and regenerate bindings.`,
    );
  }

  // Tauri and Wails use slightly different arg conventions; the Go method
  // signature is the source of truth. Struct-arg commands receive the whole
  // payload; single-positional-arg commands (see SINGLE_ARG) receive just
  // the named field.
  const singleKey = SINGLE_ARG[cmd];
  const callArg = singleKey !== undefined ? payload[singleKey] : payload;

  // Check if window.go is available (it may be lost after page navigation
  // in some Wails v2 configurations). If not, fall back to event-bridge.
  const goAvailable = typeof window !== 'undefined' &&
    typeof (window as unknown as Record<string, unknown>)['go'] !== 'undefined';

  if (!goAvailable) {
    return (await invokeViaEventBridge<T>(cmd, payload)) as T;
  }

  try {
    return (await fn(callArg)) as T;
  } finally {
    detachChannels(cmd, raw);
  }
}

/**
 * Fallback: invoke a Go method via Wails EventsEmit + EventsOn.
 * Used when window.go is not available (settings page after navigation).
 * The Go backend must have registered corresponding event listeners.
 */
async function invokeViaEventBridge<T>(
  cmd: string,
  args: Record<string, unknown>,
): Promise<T> {
  // Map known commands to their event name and result event name
  const eventMap: Record<string, { req: string; res: string }> = {
    secrets_set: { req: 'secrets:set', res: 'secrets:set:result' },
    secrets_get: { req: 'secrets:get', res: 'secrets:get:result' },
    secrets_delete: { req: 'secrets:delete', res: 'secrets:delete:result' },
    secrets_get_all: { req: 'secrets:getAll', res: 'secrets:getAll:result' },
    store_load: { req: 'store:load', res: 'store:load:result' },
    store_save: { req: 'store:save', res: 'store:save:result' },
  };

  const mapping = eventMap[cmd];
  if (!mapping) {
    throw new Error(
      `wails-shim: cannot invoke "${cmd}" via event bridge (no mapping). ` +
      'Try building with "wails build" instead of "wails dev".',
    );
  }

  return new Promise<T>((resolve, reject) => {
    // Import runtime dynamically (same pattern as event.ts)
    import('#wails/runtime/runtime').then(({ EventsEmit, EventsOn }) => {
      const timeout = setTimeout(() => {
        reject(new Error(`event bridge timeout for "${cmd}"`));
      }, 10_000);

      const unsub = EventsOn(mapping.res, (result: unknown) => {
        clearTimeout(timeout);
        unsub();
        const r = result as Record<string, unknown>;
        if (r && r['success'] === false) {
          reject(new Error(String(r['error'] ?? 'unknown error')));
        } else {
          resolve(r as T);
        }
      });

      EventsEmit(mapping.req, args);
    }).catch((err) => {
      reject(new Error(`event bridge runtime import failed: ${err}`));
    });
  });
}

/**
 * `convertFileSrc` lets the webview load local files via `file://`-like URLs.
 * In Tauri this goes through the asset protocol; Wails uses the asset server
 * similarly. For our purposes (editor preview, image embeds) we expose the
 * file over the asset server. Paths are served as-is under `/local-file/`.
 */
export function convertFileSrc(filePath: string, _protocol = "asset"): string {
  // Strip Windows extended-length prefix if present.
  const clean = filePath.replace(/^\\\\\?\\/, "");
  return `/local-file/${encodeURI(clean)}`;
}

// Re-export generated model types so call sites that imported `main` keep
// type compatibility.
export type { main } from "#wails/go/models";