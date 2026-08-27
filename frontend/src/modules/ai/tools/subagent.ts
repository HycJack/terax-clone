import { tool } from "ai";
import { z } from "zod";
import { runSubagent } from "../agents/runSubagent";
import { SUBAGENTS, type SubagentType } from "../agents/registry";
import { useChatStore } from "../store/chatStore";
import { usePreferencesStore } from "@/modules/settings/preferences";
import type { LocalProviderConfig } from "../lib/agent";
import type { ToolContext } from "./context";

const TYPE_KEYS = Object.keys(SUBAGENTS) as [SubagentType, ...SubagentType[]];

export function buildSubagentTools(ctx: ToolContext) {
  return {
    run_subagent: tool({
      description: `Spawn an isolated subagent with its own restricted toolset and a fresh message history. Use when you need to delegate a self-contained read-only investigation (large search, code review, security audit) without polluting your own context. The subagent returns a single text summary; pick a 'type' that matches its job.

Types:
${TYPE_KEYS.map((k) => `- ${k}: ${SUBAGENTS[k].description}`).join("\n")}

Auto-executes (no approval) — subagents are read-only by design.`,
      inputSchema: z.object({
        type: z.enum(TYPE_KEYS),
        prompt: z
          .string()
          .describe(
            "Self-contained instruction. The subagent has no memory of prior conversation — include all relevant context.",
          ),
        description: z
          .string()
          .optional()
          .describe("Short label shown in the chat UI for the spawn card."),
      }),
      execute: async ({ type, prompt, description }, options) => {
        const { apiKeys, selectedModelId, patchAgentMeta, customEndpointKeys } =
          useChatStore.getState();
        // Forward the full local-provider configuration so subagents use the
        // same base URLs / model ids / custom endpoints as the main agent.
        const p = usePreferencesStore.getState();
        const local: LocalProviderConfig = {
          lmstudioBaseURL: p.lmstudioBaseURL,
          lmstudioModelId: p.lmstudioModelId,
          mlxBaseURL: p.mlxBaseURL,
          mlxModelId: p.mlxModelId,
          ollamaBaseURL: p.ollamaBaseURL,
          ollamaModelId: p.ollamaModelId,
          openaiCompatibleBaseURL: p.openaiCompatibleBaseURL,
          openaiCompatibleModelId: p.openaiCompatibleModelId,
          openrouterModelId: p.openrouterModelId,
          customEndpoints: p.customEndpoints,
          customEndpointKeys,
        };
        try {
          const r = await runSubagent({
            type,
            prompt,
            keys: apiKeys,
            modelId: selectedModelId,
            toolContext: ctx,
            local,
            abortSignal: options?.abortSignal,
            onStep: (label) => patchAgentMeta({ step: label }),
          });
          return {
            type,
            description,
            summary: r.summary,
            stepCount: r.stepCount,
            durationMs: r.durationMs,
          };
        } catch (e) {
          return { error: String(e), type };
        }
      },
    }),
  } as const;
}
