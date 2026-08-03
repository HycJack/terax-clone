/**
 * Tauri→Wails shim: `@tauri-apps/api/path`.
 * Tauri exposes OS path helpers as async IPC; for our needs (UI-side cwd /
 * config dir lookups) we resolve them directly via the browser context.
 * 
 * Note: the real OS path data lives in Go; this shim returns reasonable
 * defaults that match the OS the app is running on.
 */

const isWindows =
  typeof navigator !== "undefined" &&
  navigator.platform?.toLowerCase().includes("win");

export async function homeDir(): Promise<string> {
  // We can't read HOME / USERPROFILE from the browser. The backend's
  // `workspace_current_dir` command is the source of truth; this stub is
  // here only so the import doesn't crash if the frontend needs a value
  // before that command lands.
  return isWindows ? "C:\\Users\\Default" : "/root";
}

export async function appConfigDir(): Promise<string> {
  // Wails-managed dir; the Go side has the real path. Frontend callers should
  // prefer the Go-provided path via `invoke("app_config_dir")` for writes.
  return isWindows
    ? "C:\\Users\\Default\\AppData\\Roaming\\terax"
    : "/root/.config/terax";
}

export async function appDataDir(): Promise<string> {
  return isWindows
    ? "C:\\Users\\Default\\AppData\\Roaming\\terax"
    : "/root/.local/share/terax";
}

export async function appLocalDataDir(): Promise<string> {
  return isWindows
    ? "C:\\Users\\Default\\AppData\\Local\\terax"
    : "/root/.local/share/terax";
}

export async function appCacheDir(): Promise<string> {
  return isWindows
    ? "C:\\Users\\Default\\AppData\\Local\\terax\\cache"
    : "/root/.cache/terax";
}

export async function appLogDir(): Promise<string> {
  return isWindows
    ? "C:\\Users\\Default\\AppData\\Local\\terax\\logs"
    : "/root/.local/share/terax/logs";
}

export async function tempDir(): Promise<string> {
  return isWindows ? "C:\\Users\\Default\\AppData\\Local\\Temp" : "/tmp";
}

export async function desktopDir(): Promise<string> {
  return isWindows ? "C:\\Users\\Default\\Desktop" : "/root/Desktop";
}

export async function documentDir(): Promise<string> {
  return isWindows ? "C:\\Users\\Default\\Documents" : "/root/Documents";
}

export async function downloadDir(): Promise<string> {
  return isWindows ? "C:\\Users\\Default\\Downloads" : "/root/Downloads";
}

export async function pictureDir(): Promise<string> {
  return isWindows ? "C:\\Users\\Default\\Pictures" : "/root/Pictures";
}

export async function videoDir(): Promise<string> {
  return isWindows ? "C:\\Users\\Default\\Videos" : "/root/Videos";
}

export async function audioDir(): Promise<string> {
  return isWindows ? "C:\\Users\\Default\\Music" : "/root/Music";
}

export async function resourceDir(): Promise<string> {
  return "";
}

export async function executableDir(): Promise<string> {
  return "";
}

export async function rootDir(): Promise<string> {
  return isWindows ? "C:\\" : "/";
}

/**
 * Join paths. The browser-only fallback uses forward slashes; the Go side
 * gives correct platform-aware joining on demand.
 */
export async function join(...parts: string[]): Promise<string> {
  if (parts.length === 0) return "";
  const sep = isWindows ? "\\" : "/";
  let out = parts[0] ?? "";
  for (let i = 1; i < parts.length; i++) {
    const p = parts[i] ?? "";
    if (out.endsWith("/") || out.endsWith("\\")) {
      out += p;
    } else if (p.startsWith("/") || (isWindows && p.startsWith("\\"))) {
      out += p;
    } else {
      out += sep + p;
    }
  }
  return out;
}

export async function resolve(...parts: string[]): Promise<string> {
  return join(...parts);
}

export async function normalize(path: string): Promise<string> {
  return path.replace(/[\\/]+/g, isWindows ? "\\" : "/");
}

export async function dirname(path: string): Promise<string> {
  const idx = Math.max(path.lastIndexOf("/"), path.lastIndexOf("\\"));
  return idx <= 0 ? "" : path.slice(0, idx);
}

export async function basename(path: string, ext?: string): Promise<string> {
  const idx = Math.max(path.lastIndexOf("/"), path.lastIndexOf("\\"));
  let base = idx < 0 ? path : path.slice(idx + 1);
  if (ext && base.endsWith(ext)) base = base.slice(0, -ext.length);
  return base;
}

export async function extname(path: string): Promise<string> {
  const base = await basename(path);
  const idx = base.lastIndexOf(".");
  return idx <= 0 ? "" : base.slice(idx);
}

export async function sep(): Promise<string> {
  return isWindows ? "\\" : "/";
}

export async function delimiter(): Promise<string> {
  return isWindows ? ";" : ":";
}