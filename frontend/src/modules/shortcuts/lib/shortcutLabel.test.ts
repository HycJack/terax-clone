import { beforeEach, describe, expect, it, vi } from "vitest";
import type { KeyBinding } from "../shortcuts";

const prefsMock = vi.hoisted(() => ({
  shortcuts: {} as Record<string, KeyBinding[]>,
}));

vi.mock("@/modules/settings/preferences", () => ({
  usePreferencesStore: {
    getState: () => ({ shortcuts: prefsMock.shortcuts }),
  },
}));

// `shortcutLabel` renders default bindings via getBindingTokens, whose token
// branch depends on IS_MAC. Pin to non-mac so expectations are deterministic.
vi.mock("@/lib/platform", () => ({
  IS_MAC: false,
  IS_LINUX: true,
  IS_WINDOWS: false,
  MOD_PROP: "ctrl",
}));

import { shortcutLabel } from "./shortcutLabel";

beforeEach(() => {
  prefsMock.shortcuts = {};
});

describe("shortcutLabel", () => {
  it("formats the default binding when there is no user override", () => {
    // commandPalette.open defaults to Ctrl+P.
    expect(shortcutLabel("commandPalette.open")).toBe("Ctrl P");
  });

  it("prefers a user override over the default binding", () => {
    prefsMock.shortcuts = {
      "commandPalette.open": [{ ctrl: true, shift: true, key: "k" }],
    };
    expect(shortcutLabel("commandPalette.open")).toBe("Ctrl Shift K");
  });
});
