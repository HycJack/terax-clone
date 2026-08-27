import { usePreferencesStore } from "@/modules/settings/preferences";
import type { AgentApprovalMode } from "@/modules/settings/store";
import { resolvePath, type ToolContext } from "../tools/context";
import { native } from "./native";

/**
 * Tool-approval policy for the AI agent.
 *
 * Two modes (Settings → Agents → "Tool approval"):
 *  - "always":   every mutating tool (edit, write, shell, delegation) pauses
 *                for the user — the historical default.
 *  - "critical": only high-risk steps pause: shell execution, background
 *                processes, agent delegation, and writes/edits whose resolved
 *                path falls OUTSIDE the current workspace. In-workspace file
 *                edits auto-execute (the security guard + read-before-edit
 *                invariants still run inside `execute`).
 *
 * Read-only tools never ask in either mode.
 */
export function approvalMode(): AgentApprovalMode {
  return usePreferencesStore.getState().agentApprovalMode;
}

export function isCriticalMode(): boolean {
  return approvalMode() === "critical";
}

/** True when `abs` is inside (or equal to) the workspace root. */
function isWithinWorkspace(ctx: ToolContext, abs: string): boolean {
  const root = ctx.getWorkspaceRoot();
  if (!root) return false;
  const norm = root.replace(/\/+$/, "");
  return abs === norm || abs.startsWith(`${norm}/`);
}

/**
 * Policy for file-mutation tools (edit / multi_edit / write_file /
 * create_directory). "always" mode → approve everything. "critical" mode →
 * approve only when the resolved target is outside the workspace.
 *
 * The path is canonicalized (symlinks resolved) so a link that lexically
 * lives inside the workspace but points outside still gets flagged.
 */
export async function writeNeedsApproval(
  ctx: ToolContext,
  inputPath: string,
): Promise<boolean> {
  if (approvalMode() !== "critical") return true;
  let abs: string;
  try {
    abs = resolvePath(inputPath, ctx.getCwd());
  } catch {
    // Unresolvable relative path — let `execute` surface the real error, but
    // err on the side of asking.
    return true;
  }
  try {
    abs = await native.canonicalize(abs);
  } catch {
    // Target (or parent) may not exist yet — fall back to the lexical path.
  }
  return !isWithinWorkspace(ctx, abs);
}

/**
 * Policy for exec / delegation tools (bash_run, bash_background,
 * spawn_coding_agent, send_to_agent). These are critical steps — always ask.
 */
export function alwaysNeedsApproval(): boolean {
  return true;
}