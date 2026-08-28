import { usePreferencesStore } from "@/modules/settings/preferences";
import type { AgentApprovalMode } from "@/modules/settings/store";
import { resolvePath, type ToolContext } from "../tools/context";
import { native } from "./native";

/**
 * Tool-approval policy for the AI agent.
 *
 * Three modes (Settings → Agents → "Tool approval"):
 *  - "always":   every mutating tool (edit, write, shell, delegation) pauses
 *                for the user — the historical default.
 *  - "critical": file edits/writes inside the workspace auto-execute; shell
 *                execution, background processes, agent delegation, and
 *                out-of-workspace writes still ask.
 *  - "trusted":  the agent runs hands-off inside the workspace — file edits
 *                AND shell commands auto-execute. Out-of-workspace writes and
 *                agent delegation still ask.
 *
 * Read-only tools never ask in any mode. Regardless of mode, the security
 * guard runs inside every `execute` (sensitive-path refusal, catastrophic
 * shell-command blocking, read-before-edit), so an auto-approval can never
 * blow past those invariants.
 */
export function approvalMode(): AgentApprovalMode {
  return usePreferencesStore.getState().agentApprovalMode;
}

/** Modes where in-workspace mutations are relaxed. */
export function isRelaxedMode(): boolean {
  const m = approvalMode();
  return m === "critical" || m === "trusted";
}

export function isTrustedMode(): boolean {
  return approvalMode() === "trusted";
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
 * create_directory). "always" mode → approve everything. "critical"/"trusted"
 * → approve only when the resolved target is outside the workspace.
 *
 * The path is canonicalized (symlinks resolved) so a link that lexically
 * lives inside the workspace but points outside still gets flagged.
 */
export async function writeNeedsApproval(
  ctx: ToolContext,
  inputPath: string,
): Promise<boolean> {
  if (!isRelaxedMode()) return true;
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
 * Policy for shell tools (bash_run, bash_background). "always"/"critical" →
 * always ask (arbitrary command execution is a high-risk step). "trusted" →
 * auto-approve; the catastrophic-command guard (checkShellCommand) still runs
 * inside `execute`.
 */
export function shellNeedsApproval(): boolean {
  return !isTrustedMode();
}

/**
 * Policy for delegation tools (spawn_coding_agent, send_to_agent). These hand
 * control to another agent — always ask in every mode.
 */
export function alwaysNeedsApproval(): boolean {
  return true;
}