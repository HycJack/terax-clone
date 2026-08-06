/**
 * Wails runtime wrapper: `@/lib/wails/plugin-store`.
 *
 * `LazyStore` persists a JSON key/value bag to disk. In Wails we back it
 * with `localStorage` for fast access and a single debounced flush to
 * the Go side via `invoke("store_save", { path, data })`.
 *
 * The Go backend stores under the app's per-user data dir so settings
 * survive across machines (after backup) and we keep the `defaults` and
 * `autoSave` semantics Tauri exposes.
 */

type StorePath = string;

const caches = new Map<
  StorePath,
  {
    data: Record<string, unknown>;
    dirty: boolean;
    flushTimer?: ReturnType<typeof setTimeout>;
  }
>();

async function load(path: StorePath): Promise<Record<string, unknown>> {
  if (caches.has(path)) return caches.get(path)!.data;
  try {
    const remote = await invoke<Record<string, unknown>>("store_load", { path });
    caches.set(path, { data: remote ?? {}, dirty: false });
    return caches.get(path)!.data;
  } catch {
    // First-time use or backend missing: fall back to empty.
    caches.set(path, { data: {}, dirty: false });
    return {};
  }
}

async function persist(path: StorePath): Promise<void> {
  const entry = caches.get(path);
  if (!entry || !entry.dirty) return;
  try {
    await invoke("store_save", { path, data: entry.data });
    entry.dirty = false;
  } catch (e) {
    console.warn(`store_save(${path}) failed:`, e);
  }
}

function scheduleFlush(path: StorePath, autoSave: number): void {
  const entry = caches.get(path);
  if (!entry) return;
  if (entry.flushTimer) clearTimeout(entry.flushTimer);
  entry.flushTimer = setTimeout(() => {
    void persist(path);
  }, autoSave);
}

async function invoke<T = unknown>(cmd: string, args?: Record<string, unknown>): Promise<T> {
  const { invoke: realInvoke } = await import("./core");
  return realInvoke<T>(cmd, args);
}

export type StoreOptions = {
  defaults?: Record<string, unknown>;
  autoSave?: number;
};

export class LazyStore {
  readonly path: string;
  private readonly defaults: Record<string, unknown>;
  private readonly autoSave: number;
  private readyPromise: Promise<void> | null = null;

  constructor(path: string, options: StoreOptions = {}) {
    this.path = path;
    this.defaults = options.defaults ?? {};
    this.autoSave = options.autoSave ?? 100;
  }

  private async ensureReady(): Promise<Record<string, unknown>> {
    if (!this.readyPromise) {
      this.readyPromise = (async () => {
        const loaded = await load(this.path);
        // Merge defaults.
        for (const [k, v] of Object.entries(this.defaults)) {
          if (!(k in loaded)) loaded[k] = v;
        }
        const entry = caches.get(this.path)!;
        entry.data = loaded;
      })();
    }
    await this.readyPromise;
    return caches.get(this.path)!.data;
  }

  async get<T = unknown>(key: string): Promise<T | undefined> {
    const data = await this.ensureReady();
    return data[key] as T | undefined;
  }

  async set(key: string, value: unknown): Promise<void> {
    const data = await this.ensureReady();
    data[key] = value;
    const entry = caches.get(this.path)!;
    entry.dirty = true;
    scheduleFlush(this.path, this.autoSave);
  }

  async has(key: string): Promise<boolean> {
    const data = await this.ensureReady();
    return key in data;
  }

  async delete(key: string): Promise<boolean> {
    const data = await this.ensureReady();
    if (!(key in data)) return false;
    delete data[key];
    const entry = caches.get(this.path)!;
    entry.dirty = true;
    scheduleFlush(this.path, this.autoSave);
    return true;
  }

  async clear(): Promise<void> {
    const data = await this.ensureReady();
    for (const k of Object.keys(data)) delete data[k];
    const entry = caches.get(this.path)!;
    entry.dirty = true;
    scheduleFlush(this.path, this.autoSave);
  }

  async reset(...keys: string[]): Promise<void> {
    const data = await this.ensureReady();
    const targets = keys.length === 0 ? Object.keys(data) : keys;
    for (const k of targets) delete data[k];
    const entry = caches.get(this.path)!;
    entry.dirty = true;
    scheduleFlush(this.path, this.autoSave);
  }

  async keys(): Promise<string[]> {
    const data = await this.ensureReady();
    return Object.keys(data);
  }

  async values<T = unknown>(): Promise<T[]> {
    const data = await this.ensureReady();
    return Object.values(data) as T[];
  }

  async entries<T = unknown>(): Promise<[string, T][]> {
    const data = await this.ensureReady();
    return Object.entries(data) as [string, T][];
  }

  async length(): Promise<number> {
    const data = await this.ensureReady();
    return Object.keys(data).length;
  }

  async save(): Promise<void> {
    await persist(this.path);
  }

  /**
   * Tauri `onChange` only fires within the writing process; the shim
   * simulates this by listening to a debounced local `storage` event so
   * multiple stores can sync within the same webview. Cross-window syncing
   * isn't supported by `localStorage` and is the user's responsibility
   * (the original Rust backend also limited cross-process visibility).
   */
  async onChange<T = unknown>(
    cb: (key: string, value: T) => void,
  ): Promise<() => void> {
    const data = await this.ensureReady();
    const handler = (ev: Event) => {
      const storageEv = ev as StorageEvent;
      if (storageEv.key !== this.path) return;
      try {
        const next = storageEv.newValue
          ? (JSON.parse(storageEv.newValue) as Record<string, unknown>)
          : {};
        for (const [k, v] of Object.entries(next)) {
          if (data[k] !== v) cb(k, v as T);
        }
      } catch {
        /* ignore parse errors */
      }
    };
    window.addEventListener("storage", handler);
    return () => window.removeEventListener("storage", handler);
  }
}