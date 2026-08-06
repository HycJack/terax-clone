/**
 * Wails runtime wrapper: `@/lib/wails/plugin-clipboard`.
 * Uses the browser's navigator clipboard. Browser contexts require a secure
 * (https / localhost) origin, but Wails runs the webview from a custom
 * `http://wails.localhost` scheme so the clipboard works without HTTPS.
 */

export async function readText(): Promise<string> {
  if (typeof navigator === "undefined" || !navigator.clipboard) return "";
  try {
    return await navigator.clipboard.readText();
  } catch {
    return "";
  }
}

export async function writeText(text: string): Promise<void> {
  if (typeof navigator === "undefined" || !navigator.clipboard) return;
  try {
    await navigator.clipboard.writeText(text);
  } catch {
    /* ignore */
  }
}

export async function readImage(): Promise<Uint8Array | null> {
  return null;
}

export async function writeImage(_bytes: Uint8Array): Promise<void> {
  /* image clipboard isn't supported in this build */
}

export async function writeHtml(html: string, _altText?: string): Promise<void> {
  if (typeof navigator === "undefined" || !navigator.clipboard?.write) return;
  try {
    const item = new ClipboardItem({
      "text/html": new Blob([html], { type: "text/html" }),
    });
    await navigator.clipboard.write([item]);
  } catch {
    /* fall back to text */
    await writeText(html);
  }
}