import { beforeEach, describe, expect, it, vi } from "vitest";
import { usePreferencesStore } from "@/modules/settings/preferences";
import type { ToolContext } from "../tools/context";

const nativeMock = vi.hoisted(() => ({
  canonicalize: vi.fn(async (path: string) => path),
}));

vi.mock("../lib/native", () => ({
  native: nativeMock,
}));

import { alwaysNeedsApproval, writeNeedsApproval } from "./approval";

function norm(p: string): string {
  // Faithful-enough stand-in for the backend's filepath.Abs + EvalSymlinks:
  // collapse "." / ".." segments so `../outside.ts` resolves like a real FS.
  const parts = p.split("/");
  const out: string[] = [];
  for (const seg of parts) {
    if (seg === "." || seg === "") continue;
    if (seg === "..") {
      if (out.length > 0) out.pop();
      continue;
    }
    out.push(seg);
  }
  return `/${out.join("/")}`;
}

function makeContext(workspaceRoot: string | null, cwd: string): ToolContext {
  return {
    getCwd: () => cwd,
    getWorkspaceRoot: () => workspaceRoot,
    getTerminalContext: () => null,
    isActiveTerminalPrivate: () => false,
    injectIntoActivePty: () => false,
    openPreview: () => false,
    spawnAgent: () => null,
    readAgentOutput: () => null,
    readCache: new Map(),
    getSessionId: () => "session",
  };
}

describe("AI tool approval policy", () => {
  beforeEach(() => {
    nativeMock.canonicalize.mockReset();
    nativeMock.canonicalize.mockImplementation(async (p: string) => norm(p));
    usePreferencesStore.setState({ agentApprovalMode: "always" });
  });

  it("always mode approves every write", async () => {
    usePreferencesStore.setState({ agentApprovalMode: "always" });
    const ctx = makeContext("/workspace", "/workspace");
    expect(await writeNeedsApproval(ctx, "src/a.ts")).toBe(true);
    expect(await writeNeedsApproval(ctx, "/etc/passwd")).toBe(true);
  });

  it("critical mode auto-approves in-workspace writes", async () => {
    usePreferencesStore.setState({ agentApprovalMode: "critical" });
    const ctx = makeContext("/workspace", "/workspace");
    expect(await writeNeedsApproval(ctx, "src/a.ts")).toBe(false);
    expect(await writeNeedsApproval(ctx, "/workspace/src/a.ts")).toBe(false);
  });

  it("critical mode still asks for out-of-workspace writes", async () => {
    usePreferencesStore.setState({ agentApprovalMode: "critical" });
    const ctx = makeContext("/workspace", "/workspace");
    expect(await writeNeedsApproval(ctx, "/home/me/other/x.ts")).toBe(true);
    // ../ escapes the workspace.
    expect(await writeNeedsApproval(ctx, "../outside.ts")).toBe(true);
  });

  it("critical mode asks when a symlink resolves outside the workspace", async () => {
    usePreferencesStore.setState({ agentApprovalMode: "critical" });
    nativeMock.canonicalize.mockImplementation(async (p: string) =>
      p === "/workspace/src/link.ts" ? "/home/me/secret.ts" : norm(p),
    );
    const ctx = makeContext("/workspace", "/workspace");
    expect(await writeNeedsApproval(ctx, "src/link.ts")).toBe(true);
  });

  it("critical mode falls back to asking without a workspace root", async () => {
    usePreferencesStore.setState({ agentApprovalMode: "critical" });
    const ctx = makeContext(null, "/workspace");
    expect(await writeNeedsApproval(ctx, "src/a.ts")).toBe(true);
  });

  it("exec / delegation tools always ask", () => {
    expect(alwaysNeedsApproval()).toBe(true);
  });
});