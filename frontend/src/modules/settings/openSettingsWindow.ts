export type SettingsTab =
  | "general"
  | "editor"
  | "themes"
  | "shortcuts"
  | "models"
  | "agents"
  | "about";

// Keyed by SettingsTab so a "models" tab opened from a session survives a
// restart — the user gets back to the same view. Falls back to "general"
// for unknown / missing values.
const SETTINGS_TAB_KEY = "terax:settings:lastTab";

function rememberTab(tab: SettingsTab | undefined) {
  try {
    if (tab) localStorage.setItem(SETTINGS_TAB_KEY, tab);
  } catch {
    /* private mode etc. */
  }
}

/**
 * Subscribable state for the in-app settings dialog.
 * Instead of navigating to a separate HTML page (which breaks window.go
 * in Wails v2), we open a dialog overlay in the main app context.
 */
let _resolveSettingsDialog: ((tab?: SettingsTab) => void) | null = null;

export function registerSettingsDialog(
  resolve: (tab?: SettingsTab) => void,
): void {
  _resolveSettingsDialog = resolve;
}

export function unregisterSettingsDialog(): void {
  _resolveSettingsDialog = null;
}

export async function openSettingsWindow(tab?: SettingsTab): Promise<void> {
  rememberTab(tab);
  if (_resolveSettingsDialog) {
    _resolveSettingsDialog(tab);
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
