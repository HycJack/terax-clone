import { OpenSettingsWindow } from "../../../bindings/terax/app";

export type SettingsTab =
  | "general"
  | "editor"
  | "themes"
  | "shortcuts"
  | "models"
  | "agents"
  | "about";

const SETTINGS_TAB_KEY = "terax:settings:lastTab";

function rememberTab(tab: SettingsTab | undefined) {
  try {
    if (tab) localStorage.setItem(SETTINGS_TAB_KEY, tab);
  } catch {
    /* private mode etc. */
  }
}

/**
 * Open settings in a new Wails v3 window.
 */
export async function openSettingsWindow(tab?: SettingsTab): Promise<void> {
  rememberTab(tab);
  try {
    await OpenSettingsWindow({ tab: tab ?? null });
  } catch (e) {
    console.error("[terax] Failed to open settings window:", e);
  }
}

export function readSettingsDefaultTab(): SettingsTab | undefined {
  try {
    const stored = localStorage.getItem(SETTINGS_TAB_KEY);
    if (stored === "ai" || stored === "connections") return "models";
    return (stored ?? undefined) as SettingsTab | undefined;
  } catch {
    return undefined;
  }
}
