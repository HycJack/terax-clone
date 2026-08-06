// WebKitGTK can't read external copies, so the native plugin is Linux-only and
// lazy-loaded to keep it out of the mac/win bundle.
const IS_LINUX =
  typeof navigator !== "undefined" &&
  /Linux/.test(navigator.userAgent) &&
  !/Android/.test(navigator.userAgent);

function webClipboard(): Clipboard | null {
  if (typeof navigator === "undefined") return null;
  return navigator.clipboard ?? null;
}

export async function readTerminalClipboard(): Promise<string> {
  if (IS_LINUX) {
    try {
      const { readText } = await import("@/lib/wails/plugin-clipboard");
      return await readText();
    } catch {}
  }
  try {
    return (await webClipboard()?.readText()) ?? "";
  } catch {
    return "";
  }
}

export async function writeTerminalClipboard(text: string): Promise<void> {
  if (IS_LINUX) {
    try {
      const { writeText } = await import("@/lib/wails/plugin-clipboard");
      await writeText(text);
      return;
    } catch {}
  }
  try {
    await webClipboard()?.writeText(text);
  } catch {}
}
