/**
 * Wails runtime wrapper: `@/lib/wails/path`.
 * Delegates to the Go backend for real OS paths (home, config, data dirs).
 * Falls back to reasonable defaults if the backend call fails.
 */

import { invoke } from "@/lib/wails/core";

const isWindows =
  typeof navigator !== "undefined" &&
  navigator.platform?.toLowerCase().includes("win");

let cachedHome: string | null = null;
let cachedConfigDir: string | null = null;
let cachedDataDir: string | null = null;

export async function homeDir(): Promise<string> {
  if (cachedHome) return cachedHome;
  try {
    const h = await invoke<string>("app_home_dir");
    if (h) {
      cachedHome = h;
      return h;
    }
  } catch {
    /* fall through to default */
  }
  return isWindows ? "C:\\Users\\Default" : "/root";
}

export async function appConfigDir(): Promise<string> {
  if (cachedConfigDir) return cachedConfigDir;
  try {
    const d = await invoke<string>("app_config_dir");
    if (d) {
      cachedConfigDir = d;
      return d;
    }
  } catch {
    /* fall through */
  }
  return isWindows
    ? "C:\\Users\\Default\\AppData\\Roaming\\terax"
    : "/root/.config/terax";
}

export async function appDataDir(): Promise<string> {
  if (cachedDataDir) return cachedDataDir;
  try {
    const d = await invoke<string>("app_data_dir");
    if (d) {
      cachedDataDir = d;
      return d;
    }
  } catch {
    /* fall through */
  }
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