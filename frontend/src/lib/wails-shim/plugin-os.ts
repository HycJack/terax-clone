/**
 * Tauri→Wails shim: `@tauri-apps/plugin-os`.
 * The OS detection is best done at runtime by the Go side; we expose a
 * browser-platform fallback that matches the user's OS closely enough for
 * UI gating (keyboard shortcuts, window chrome, font hints).
 */

const ua =
  typeof navigator !== "undefined" ? navigator.userAgent.toLowerCase() : "";
const platform =
  typeof navigator !== "undefined" ? navigator.platform.toLowerCase() : "";

export type Platform =
  | "linux"
  | "macos"
  | "windows"
  | "ios"
  | "android"
  | "freebsd"
  | "dragonfly"
  | "netbsd"
  | "openbsd"
  | "solaris"
  | "unknown";

function detectPlatform(): Platform {
  if (platform.includes("mac")) return "macos";
  if (platform.includes("win")) return "windows";
  if (platform.includes("linux")) return "linux";
  if (/iphone|ipad|ipod/.test(ua)) return "ios";
  if (/android/.test(ua)) return "android";
  return "unknown";
}

const PLATFORM: Platform = detectPlatform();

export function platform$(): Platform {
  return PLATFORM;
}
export { platform$ as platform };

export type Arch =
  | "x86_64"
  | "x86"
  | "arm"
  | "aarch64"
  | "mips"
  | "mips64"
  | "powerpc"
  | "powerpc64"
  | "riscv64"
  | "s390x"
  | "loongarch64"
  | "unknown";

function detectArch(): Arch {
  // Browsers don't expose real CPU arch reliably; default to x86_64.
  if (/arm64|aarch64/i.test(ua)) return "aarch64";
  if (/wow64|win64|x64/i.test(ua)) return "x86_64";
  return "x86_64";
}

export function arch$(): Arch {
  return detectArch();
}
export { arch$ as arch };

export type OsType = "linux" | "macos" | "windows" | string;
export function type(): OsType {
  return PLATFORM;
}

export function version(): string {
  return "wails-2.0";
}

export async function locale(): Promise<string | null> {
  return navigator?.language ?? null;
}

export async function hostname(): Promise<string> {
  return "terax-host";
}