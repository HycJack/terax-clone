/**
 * Wails v3 runtime wrapper: `@/lib/wails/core`.
 *
 * Bridges the existing Tauri command surface (`invoke`, `Channel`,
 * `convertFileSrc`) onto the Wails v3 runtime. Frontend code that imports from
 * `@/lib/wails/core` is left untouched.
 */
import { Events } from "@wailsio/runtime";
import * as AppBinding from "../../../bindings/terax/app";

/** Unwrap a WailsEvent to its raw `.data` payload. */
function unwrapWailsEvent(raw: unknown): unknown {
  if (raw && typeof raw === "object" && "data" in (raw as object)) {
    return (raw as Record<string, unknown>).data;
  }
  return raw;
}

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
  private attachedName: string | null = null;

  constructor() {
    this.eventName = nextChannelId("ch");
  }

  // Internal: subscribe to backend events. Called from `invoke()` when it
  // detects a Channel argument and passes the channel instance through.
  _attach(prefix: string): string {
    const name = `${prefix}:${this.eventName}`;
    this.attachedName = name;
    Events.On(name, (ev) => {
      if (this.cancelled) return;
      // Wails v3 delivers a WailsEvent {name, data}; unwrap to get the raw data.
      const data = unwrapWailsEvent(ev);
      // Handle base64-encoded binary data (Channel pattern).
      if (typeof data === "string") {
        try {
          const bin = atob(data);
          const bytes = new Uint8Array(bin.length);
          for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
          this.onmessage?.(bytes.buffer as T);
        } catch (e) {
          console.error("[terax] Channel._attach: atob failed:", e);
        }
      } else {
        this.onmessage?.(data as T);
      }
    });
    return name;
  }

  _detach(eventName?: string): void {
    this.cancelled = true;
    const name = eventName ?? this.attachedName;
    if (!name) return;
    try {
      Events.Off(name);
    } catch {
      /* ignore */
    }
  }
}

/**
 * Look up a bound Go method by command name. Generated v3 bindings live under
 * `bindings/terax/app.ts` with PascalCase method names; the frontend calls
 * them using Tauri-style snake_case so we convert `pty_open` → `PtyOpen` here.
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
  const appMethods = AppBinding as unknown as Record<string, AnyFn>;
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

/**
 * Channels attached for long-lived subscriptions must NOT be detached here.
 * `lsp_spawn`'s message/exit channels live for the whole LSP session — the
 * transport tears them down in `TauriLspTransport.close()` when the session
 * ends. Detaching here would cancel the listener before gopls ever responds,
 * dropping the initialize result and leaving the session uninitialized
 * ("no views" on every request). Only one-shot channel invocations are
 * cleaned up in the invoke `finally`.
 */
function detachChannels(cmd: string, args: Record<string, unknown>): void {
  if (cmd === "ai_http_stream") {
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
  pty_read_output: "id",
  pty_shell_name: "id",
  // lsp
  lsp_detect: "command",
  lsp_kill: "id",
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
 * Mirrors `invoke()` from `@/lib/wails/core`. Args is the named-args bag
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
      `wails: no Go binding for command "${cmd}". Add a method to internal/ backend and regenerate bindings.`,
    );
  }

  // Tauri and Wails use slightly different arg conventions; the Go method
  // signature is the source of truth. Struct-arg commands receive the whole
  // payload; single-positional-arg commands (see SINGLE_ARG) receive just
  // the named field.
  const singleKey = SINGLE_ARG[cmd];
  const callArg = singleKey !== undefined ? payload[singleKey] : payload;

  try {
    return (await fn(callArg)) as T;
  } catch (e) {
    throw e;
  } finally {
    detachChannels(cmd, raw);
  }
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
