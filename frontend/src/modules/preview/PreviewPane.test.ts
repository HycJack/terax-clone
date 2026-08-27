import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

/**
 * Source-level regression test for the preview iframe's security attributes.
 * Rendering this component for real requires jsdom + a working
 * useImperativeHandle stub; for a focused security check we just verify the
 * static JSX + the `sandboxFor` helper still carry the expected semantics —
 * if a future change silently weakens them, this test fails.
 */

const here = path.dirname(fileURLToPath(import.meta.url));
const src = readFileSync(path.join(here, "PreviewPane.tsx"), "utf8");
const iframeMatch = src.match(/<iframe[\s\S]*?\/>/);
// Strip JSX comments (`// …` inside `{…}` and `{/* … */}` blocks) so the
// assertions only see actual attribute syntax — the source explains in a
// comment why `allow-top-navigation` is intentionally omitted, which we
// don't want to match.
const iframeJsx = (iframeMatch?.[0] ?? "")
  .replace(/\/\*[\s\S]*?\*\//g, "")
  .replace(/\/\/[^\n]*/g, "");

const sandboxMatch = src.match(/function sandboxFor[\s\S]*?^}/m);
const sandboxFn = sandboxMatch?.[0] ?? "";

describe("PreviewPane iframe sandbox", () => {
  it("declares an iframe in the source", () => {
    expect(iframeJsx).not.toBe("");
  });

  it("computes the sandbox via sandboxFor(url)", () => {
    expect(iframeJsx).toMatch(/sandbox=\{sandboxFor\(url\)\}/);
  });

  it("always grants allow-scripts (dev previews need JS)", () => {
    expect(sandboxFn).toMatch(/allow-scripts/);
  });

  it("denies allow-same-origin for /local-file/ URLs (opaque origin)", () => {
    // A /local-file/ page is served from the app's own origin; granting
    // allow-same-origin there would let it reach window.parent.go (Wails IPC).
    const localBranch = sandboxFn.match(
      /if \(url\.startsWith\("\/local-file\/"\)\) return ([^;]+);/,
    );
    expect(localBranch).not.toBeNull();
    expect(localBranch?.[1]).not.toMatch(/allow-same-origin/);
  });

  it("keeps allow-same-origin for external / dev-server URLs", () => {
    // Cross-origin URLs need their own origin for cookies/storage.
    const externalBranch = sandboxFn.match(/return `\$\{base\} ([^`]+)`;/);
    expect(externalBranch).not.toBeNull();
    expect(externalBranch?.[1]).toMatch(/allow-same-origin/);
  });

  it("does NOT include allow-top-navigation* tokens", () => {
    // The whole point of sandboxing here: forbid the iframe from navigating
    // the parent webview to an attacker origin. Top-nav permissions must
    // never be added.
    expect(iframeJsx).not.toMatch(/allow-top-navigation/);
    expect(sandboxFn).not.toMatch(/allow-top-navigation/);
  });

  it("does NOT include allow-popups-without-allow-popups-to-escape-sandbox combo", () => {
    // If popups are allowed, they MUST escape the sandbox cleanly — otherwise
    // a popup window inherits sandbox flags and we get hard-to-debug behavior.
    if (/allow-popups\b/.test(sandboxFn)) {
      expect(sandboxFn).toMatch(/allow-popups-to-escape-sandbox/);
    }
  });

  it("sets referrerPolicy to no-referrer", () => {
    expect(iframeJsx).toMatch(/referrerPolicy="no-referrer"/);
  });
});