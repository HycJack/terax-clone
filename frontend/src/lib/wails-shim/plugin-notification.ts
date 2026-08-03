/**
 * Tauri→Wails shim: `@tauri-apps/plugin-notification`.
 * Uses the browser's `Notification` API when granted; falls back to
 * `console.info` for the dev experience.
 */

export type Permission = "granted" | "denied" | "default";

export type Options = {
  title: string;
  body?: string;
  icon?: string;
  silent?: boolean;
};

export async function isPermissionGranted(): Promise<boolean> {
  if (typeof Notification === "undefined") return false;
  return Notification.permission === "granted";
}

export async function requestPermission(): Promise<Permission> {
  if (typeof Notification === "undefined") return "denied";
  const r = await Notification.requestPermission();
  return (r as Permission) ?? "default";
}

export function sendNotification(options: string | Options): void {
  if (typeof Notification === "undefined") return;
  if (Notification.permission !== "granted") return;
  if (typeof options === "string") {
    new Notification(options);
    return;
  }
  new Notification(options.title, {
    body: options.body,
    icon: options.icon,
    silent: options.silent,
  });
}