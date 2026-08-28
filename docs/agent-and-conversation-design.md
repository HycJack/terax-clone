# Agent 设计与会话存储

> 适用范围：Terax 的 AI 编码助手（agent）体系与对话/会话持久化。
> 主要代码：`frontend/src/modules/ai/`、`frontend/src/modules/agents/`、`internal/agent/`、`internal/store/`、`helpers.go`。

---

## 1. 总览

Terax 的 AI 能力是一条**纯前端驱动的 agent 运行链路**，Go 后端只承担三件事：

1. **持久化**——把前端 `LazyStore` 的 JSON 数据袋写到磁盘（`store_load` / `store_save`）。
2. **密钥保管**——通过 OS keyring 存取各 provider 的 API key（`secrets_*`）。
3. **agent 钩子状态**——一个非常薄的内存标志 + 窗口事件（`agent_enable_hooks` / `agent_hooks_status`）。

真正的 agent 编排（模型调用、流式输出、工具执行、审批、上下文压缩、会话管理）全部发生在浏览器侧的 TypeScript 里，基于 **Vercel AI SDK**（`@ai-sdk/react` 的 `Chat` + `ai` 的 `streamText` / `generateText`）。

```
┌─────────────────────────── 前端 (React / zustand / AI SDK) ───────────────────────────┐
│                                                                                       │
│  UI ── AiChat / AiMiniWindow / AgentRunBridge ── status / messages / approvals        │
│        │                                                                              │
│  chatStore.ts ── 会话列表、活跃会话、agentMeta、apiKeys、LRU Chat 缓存、debounce 持久化 │
│        │                                                                              │
│  chatRuntime.ts ── 构造 Chat<UIMessage> + ContextAwareTransport                       │
│        │                                                                              │
│  transport.ts / agent.ts ── runAgentStream：组装 system、压缩历史、buildTools、stream │
│        │                                                                              │
│  tools/* ── fs/edit/search/shell/subagent/terminal/todo/managedAgent                 │
└───────────┬─────────────────────────────────────┬─────────────────────────────────────┘
            │ invoke                               │ keyring / store
┌───────────▼──────────┐                ┌──────────▼─────────────────────────────────────┐
│ internal/agent        │                │ internal/store + helpers.go（JSON 袋、原子写）  │
│ (hooks 内存状态+事件) │                │ go-keyring（OS 钥匙串）                        │
└──────────────────────┘                └────────────────────────────────────────────────┘
```

---

## 2. Agent 设计

### 2.1 Agent 人格（Persona）

Agent 是「对话当前绑定的人格」，决定系统提示词中的 `## ACTIVE AGENT` 段。

- **内置人格** `frontend/src/modules/ai/lib/agents.ts` 的 `BUILTIN_AGENTS`：
  `coder` / `architect` / `reviewer` / `security` / `designer`，每条都有独立的 `instructions`。
- **自定义人格**：用户通过设置界面增删，存于 `terax-ai-agents.json`，运行时由
  `agentsStore.ts` 合并成 `all() = [...BUILTIN_AGENTS, ...customAgents]`。
- 模型选择 + 人格切换在 `AgentSwitcher.tsx` / `AiStatusBarControls.tsx`，切换会广播
  `terax://ai-agents-changed` 事件让多窗口同步。

### 2.2 运行链路（一次发送的完整流程）

1. 用户发送消息 → `chatRuntime.sendMessage(text)` → 校验 provider key →
   `getOrCreateChat(sessionId)` → `Chat.sendMessage({ text })`。
2. AI SDK 的 `Chat` 把消息交给 `ContextAwareTransport`（`lib/transport.ts`）：
   - 读取工作区 `TERAX.md` 作为项目记忆（30s 缓存，截断到 32KB）；
   - 把活跃终端 cwd / workspace root / active file / private 模式注入最后一条 user 消息（`<env>` 块）；
   - 调用 `runAgentStream`。
3. `runAgentStream`（`lib/agent.ts`）：
   - `buildConfiguredLanguageModel` 按 provider 构造模型（OpenAI/Anthropic/Google/xAI/Cerebras/
     DeepSeek/Mistral/Groq/OpenRouter + LM Studio/MLX/Ollama/OpenAI-compatible 本地推理），模型实例按
     `provider+key+baseURL+modelId` 缓存；
   - `buildStableSystem` 组装稳定 system：基础提示（`config.selectSystemPrompt`）+
     `TERAX.md` 项目记忆 + 激活 agent 人格 + 用户自定义指令；
   - `convertToModelMessages` + `pruneMessages`（按是否保留 reasoning 裁剪）+
     `compactModelMessagesDetailed`（上下文压缩，见 §2.6）；
   - `prepareAgentPrompt`：对 Anthropic 打 ephemeral cache 标记；
   - `streamText({ tools: buildTools(...), stopWhen: stepCountIs(MAX_AGENT_STEPS=24) })`，
     `onStepFinish` 上报步骤 label 与 usage。
4. 流式结果经 `result.toUIMessageStream()` 回灌给 `Chat`，UI 组件（`AgentRunBridge`）
   监听状态变化 → 更新 `agentMeta`（thinking / streaming / awaiting-approval / error）、
   打开审批弹窗、把文件变更打开成 AI diff 标签页、把消息落盘。

### 2.3 工具系统（tools/*）

`buildTools`（`tools/tools.ts`）合并八组工具：

| 分组 | 文件 | 工具 |
|------|------|------|
| FS | `fs.ts` | `read_file`, `list_directory`, `create_directory`, `write_file` |
| Edit | `edit.ts` | `edit`, `multi_edit` |
| Search | `search.ts` | `grep`, `glob` |
| Shell | `shell.ts` | `bash_run`, `bash_background`, `bash_logs`, `bash_list`, `bash_kill`, `suggest_command` |
| Subagent | `subagent.ts` | `run_subagent` |
| Terminal | `terminal.ts` | 终端注入/读取 |
| Todo | `todo.ts` | `todo_write` |
| Managed agent | `agent.ts` | `spawn_coding_agent`, `send_to_agent`, `read_agent_output` |

**审批与安全策略**：
- 只读工具（`read_file` / `list_directory` / `grep` / `glob`）**自动执行**，但经
  `lib/security.ts` 的安全守卫拒绝明显敏感路径（`.env*`、`.ssh/`、凭据文件等）。
- 写操作（`write_file` / `edit` / `multi_edit` / `create_directory` / `run_command`）
  需要**显式审批**——AI SDK 在 tool-call 处暂停，生成 `tool-approval-request` part，
  UI 渲染成确认卡片；`AgentRunBridge` 把待审的 `write_file` 变成编辑器区的 AI diff 标签页。
- `edit` / `multi_edit` 强制执行 **read-before-edit** 不变量（本会话内必须先 `read_file` 该路径）。
- 所有路径先经 `resolvePath` 相对活跃终端 cwd 解析成绝对路径再交给模型。

### 2.4 子代理（Subagent）

`tools/subagent.ts` + `agents/registry.ts` + `agents/runSubagent.ts`：

- 四种内置子代理：`explore` / `code-review` / `security` / `general`，全部**只读**。
- 主模型调用 `run_subagent` 时，`runSubagent` 用 `generateText` + 受限工具集（只读四件套，
  显式排除写工具与 `run_subagent` 本身防递归）+ 独立 system 提示，跑最多 12 步，
  返回单条文本摘要。子代理无会话记忆，prompt 必须自包含。
- 目的：把大范围只读调查（大搜索、代码评审、安全审计）隔离出去，不污染主代理上下文。

### 2.5 托管编码 Agent（Claude Code 集成）

`tools/agent.ts` + `agents/store/managedAgentsStore.ts`：

- 把任务**委派给真正的 Claude Code CLI**：`spawn_coding_agent` 在新终端标签页开一个
  Claude Code 进程并注入 prompt（需审批）；`send_to_agent` 把后续指令打进该终端并回车；
  `read_agent_output` 读取该标签页缓冲尾部。
- 每个会话最多 `DEFAULT_MAX_ROUNDS = 3` 轮，`managedAgentsStore` 跟踪
  `spawning → working → reviewing → done` 生命周期。
- 与主 agent 的关系：主 agent 通过这三个工具“外包”编码执行，自己负责协调与汇报。

### 2.6 上下文管理

- **稳定 system**：`selectSystemPrompt`（按模型选基座）+ 项目记忆 + 人格 + 自定义指令。
- **PLAN MODE**：`usePlanStore.active` 时注入额外指令，把写工具改成“排队等审”，并禁止执行
  bash——用户审完一份 diff 再继续。
- **压缩 `lib/compact.ts`**：按 `approxBytes/4` 估算 token；
  - ≥55% 上下文时先做 `dropSupersededReads`——把“读过的文件后来被改过”的旧 `read_file`
    结果与陈旧 tool-result 替换为占位符（`__elided`）；
  - 仍 ≥70% 时对非末尾 `KEEP_TAIL=24` 条之前的 tool-result 逐一 elide；
  - 压缩后保留最近尾部，避免整个对话被清空。
- **Anthropic 缓存**：`prepareAgentPrompt` 对 system 与最后一条消息打
  `cacheControl: { type: "ephemeral" }`，多轮少计费。
- **`MAX_AGENT_STEPS = 24`**：`stepCountIs` 强制截断，`onFinishMeta` 上报 `hitStepCap`。

### 2.7 Agent 钩子（Go 侧）

`internal/agent/agent.go` 是占位实现：`EnableHooks` / `HooksStatus` 只维护内存
`map[string]bool`，并通过 `terax:hooks-ready` / `terax:agent-signal` 两个窗口事件通知前端。
当前不真正监听外部 agent 的 shell 钩子输出，属“待接线”的薄层。

---

## 3. 会话（对话）存储

### 3.1 存储介质与路径

| 数据 | 文件（位于用户数据目录） | 说明 |
|------|--------------------------|------|
| 会话列表 / 活跃会话 | `terax-ai-sessions.json` | `sessions` + `activeId` 两个键 |
| 每个会话的消息 | 同上（`messages:<id>` 键） | 每个会话一个键，懒加载 |
| 自定义 agent | `terax-ai-agents.json` | `customAgents` + `activeAgentId` |

用户数据目录 = `appDir()`（macOS/Linux `~/.config/terax`，Windows `%APPDATA%\terax`），
在 `startup` 时 `storeInit(dir)` 创建。

### 3.2 LazyStore（前端持久化门面）

`frontend/src/lib/wails/plugin-store.ts` 实现 `LazyStore`（对齐 Tauri 的
`tauri-plugin-store` API）：

- **双写**：读命中 `localStorage` 缓存（快），写则标记 `dirty` 并按 `autoSave`（默认 200ms）
  防抖调用 `invoke("store_save", { path, data })` 落到 Go。
- 首次访问时 `store_load` 拉取远端 JSON 并合并 `defaults`；跨 store 同步靠 `storage` 事件模拟
  `onChange`。
- `entries()` 一次 IPC 拉全部键，冷启动只需一次调用。

### 3.3 Go 持久化后端

`internal/store/store.go`（`helpers.go` 的 `storeLoad`/`storeSave` 调它）：

- 每个 store 路径对应磁盘上 `<dir>/<path>.json` 一个 JSON 对象袋。
- `Save` 用 **临时文件 + `os.Rename`** 原子写，避免写一半损坏。
- `safePath` 拒绝 `..`、`/`、`\`，并校验解析后路径仍在数据目录内，防路径穿越。

### 3.4 会话模型与生命周期（`chatStore.ts` / `lib/sessions.ts`）

- `SessionMeta = { id, title, createdAt, updatedAt }`，id 为 `s-<ts36>-<rand>`。
- 启动 `hydrateSessions`：若无待复用会话则创建一条 "New chat"；切换会话时
  `switchSession` **懒加载**该会话消息并写入 `seedMessages`，首次构造 `Chat` 时消费。
- 活跃会话 id 持久化在 `activeId`，重启后恢复。
- 删除会话会 `Chat.stop()`、清缓存/待写队列、`deleteSessionData`、并清掉该会话的 todo。

### 3.5 消息落盘（防抖 + 空闲冲刷）

- 每次消息变更 `AgentRunBridge` 调 `chatStore.persistMessages`，但**内部 300ms 防抖**：
  流式每 token 都触发，直接写会卡 UI。
- 空闲（`status` 离开 submitted/streaming）或卸载时 `flushPersist` 立即冲刷，保证关窗/切会话不丢尾巴。
- **LRU Chat 缓存**（`chats`，上限 8）：切走的会话保留热缓存，超限时冲刷 + `stop()` 淘汰。

### 3.6 标题自动推导

`deriveTitle`：取第一条 user 消息的首个文本 part，剔除 `<terminal-context>` /
`<selection>` / `<file>` 注入块后取首行（>40 字符截断）；"New chat" 在有内容后自动重命名。
仅在标题实际变化时才写会话列表，避免每 token 重写。

### 3.7 密钥存储

API key 不落 JSON：`keyring.ts` 经 `secrets_get/set` 调用 Go 侧 `go-keyring`
（macOS Keychain / Windows Credential Manager / Linux Secret Service），按
`service=terax` + provider 专属 `keyringAccount` 存取。内存中的 `apiKeys` /
`customEndpointKeys` 是运行时副本，仅用于构造模型。

---

## 4. 数据流示例：一次带审批的文件编辑

```
用户: "把 README 的标题改成中文"
  │
  ├─ Chat.sendMessage → ContextAwareTransport
  │    └─ readFile(TERAX.md) + <env>注入 + compact
  │         └─ streamText(model, tools, stopWhen=24步)
  │              ├─ 模型调用 read_file (自动) → 安全守卫 → 结果
  │              ├─ 模型调用 edit (needsApproval)
  │              │    └─ AI SDK 暂停 → tool-approval-request part
  │              │         ├─ AgentRunBridge: agentMeta=awaiting-approval + openMini
  │              │         └─ 打开 AI diff 标签页（编辑前内容 vs 提议内容）
  │              └─ 用户批准 → addToolApprovalResponse → 继续流式
  │
  └─ 每次消息变更 → persistMessages → 300ms 防抖 → store_save → <dir>/terax-ai-sessions.json
```

---

## 5. 关键文件索引

| 模块 | 文件 |
|------|------|
| 会话 store | `frontend/src/modules/ai/store/chatStore.ts` |
| Chat 构造/发送 | `frontend/src/modules/ai/store/chatRuntime.ts` |
| 传输层/环境注入 | `frontend/src/modules/ai/lib/transport.ts` |
| 核心运行 | `frontend/src/modules/ai/lib/agent.ts` |
| 上下文压缩 | `frontend/src/modules/ai/lib/compact.ts` |
| Prompt 组装 | `frontend/src/modules/ai/lib/prompt.ts` |
| 工具注册 | `frontend/src/modules/ai/tools/tools.ts` |
| 子代理 | `frontend/src/modules/ai/agents/{registry,runSubagent}.ts` |
| 托管 agent 工具 | `frontend/src/modules/ai/tools/agent.ts` |
| 托管 agent 状态 | `frontend/src/modules/agents/store/managedAgentsStore.ts` |
| 会话持久化 | `frontend/src/modules/ai/lib/sessions.ts` |
| Agent 定义持久化 | `frontend/src/modules/ai/lib/agents.ts` |
| 密钥 | `frontend/src/modules/ai/lib/keyring.ts` |
| LazyStore | `frontend/src/lib/wails/plugin-store.ts` |
| Go 持久化 | `internal/store/store.go`、`helpers.go` |
| Go agent 钩子 | `internal/agent/agent.go` |
| Go 绑定 | `app.go`（`StoreLoad/StoreSave/Secrets*/Agent*`） |

---

## 6. 设计要点小结

1. **前端为中心的 agent**：编排逻辑全在 TS，Go 只做持久化/密钥/事件，改动模型或工具无需动 Go。
2. **审批驱动安全**：读自动、写必审、edit 强制 read-before-edit、只读子代理隔离大搜索。
3. **多级上下文保护**：TERAX.md 项目记忆 + 压缩（陈旧读取替换/尾部保留）+ 步数上限 + Anthropic 缓存。
4. **会话持久化分层**：localStorage 热缓存 → 200ms Go 落盘；消息 300ms 防抖 + 空闲冲刷；LRU 缓存复用。
5. **密钥不落盘明文**：走系统钥匙串。