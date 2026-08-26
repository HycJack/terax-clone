# 查看定义 & 查找所有引用 —— 完整实现方案（Terax LSP 架构）

> 目标：在 Terax（Wails v2 + Go 后端 + React 前端 + CodeMirror 6 编辑器）中实现
> **查看定义（Go to Definition）** 与 **查找所有引用（Find All References）**，
> 并支持跨文件跳转、悬停预览（hover）、结果面板、重命名（rename）。

---

## 0. 核心设计原则（先纠正一个误区）

你给出的技术流程里，"客户端解析 AST、自己建符号表（Symbol Table）、做作用域过滤 / 类型推断"
这些步骤**不应该在前端（编辑器）做**，而应全部委托给**语言服务器（LSP Server）**。

理由：

| 误区 | 正确做法 |
|------|---------|
| 前端自己解析 AST | 语言服务器已有完整解析器（TypeScript 编译器、rust-analyzer、pyright 等），前端重复解析只会重复造轮子且极易出错 |
| 前端维护"符号名 → 定义位置"的倒排索引 | 是否可靠取决于**作用域 + 类型推断**（同名不同作用域的 `count`），这正是语言服务器的强项，前端没法做对 |
| 前端做正则全文搜索 | 会误匹配注释、字符串、无关同名变量；只能作兜底，不能作主方案 |

所以架构是：

```
编辑器(CodeMirror)  --LSP请求-->  转发层(Wails transport)  -->  语言服务器子进程
        ^                                                              |
        +------------------ LSP响应(通知事件/回调) <--------------------+
```

- **前端**：只负责采集光标位置、发请求、渲染跳转和结果。
- **后端（Go）**：只负责把 JSON-RPC 消息用 `Content-Length` 分帧，在子进程 stdin/stdout 间搬运（进程生命周期、崩溃重启、内存上限、stderr 收集）。
- **语言服务器**：负责真正的 AST、符号表、作用域过滤、类型推断。
- **第三方库**：`codemirror-languageserver` 提供 `LanguageServerClient`，封装了 LSP 协议握手、同步、请求/通知、初始化能力协商，我们只需做薄封装。

---

## 1. 整体架构与模块划分

### 1.1 端到端数据流（以"查看定义"为例）

```
[用户] Ctrl+点击 / F12 / 右键"转到定义"
   │
   ▼
[前端] client.ts  lspInteractions()
   │  · 取光标位置 line/character（0-based）
   │  · 调 client.textDocumentDefinition({ textDocument:{uri}, position })
   ▼
[codemirror-languageserver] LanguageServerClient.raw.request("textDocument/definition", params, timeout)
   │  生成 JSON-RPC：{"jsonrpc":"2.0","id":1,"method":"textDocument/definition","params":{...}}
   ▼
[transport.ts] TauriLspTransport.send()
   │  invoke("lsp_send", { id: sessionId, message })
   ▼
[后端] internal/lsp/lsp.go  Manager.Send()
   │  写 stdin（前端已带好 Content-Length 头）
   ▼
[语言服务器] 解析 AST → 查符号表 → 返回 Location[] / LocationLink[]
   │  stdout: {"jsonrpc":"2.0","id":1,"result":[{uri,range},...]}
   ▼
[后端] Session.read() 读 stdout → EventsEmit(OnMessageEvent, body)
   │  Wails Channel<ArrayBuffer> 送达前端
   ▼
[前端] transport.onMessage → codemirror-languageserver 解析响应
   │  → client.textDocumentDefinition 的 Promise resolve
   ▼
[client.ts] normalizeLocations() → showResults()
   │  · 1 个结果 → navigate()（同文件移动光标 / 跨文件调 onExternal 打开文件）
   │  · 多个结果 → openLocationsPanel()（底部结果面板，上下键/回车切换）
   ▼
[navigator.ts] getLspNavigator()?.openFile(path, line)
   → 编辑器打开目标文件并移动光标
```

### 1.2 模块清单（与现有代码一一对应）

| 层 | 文件 | 职责 |
|----|------|------|
| Go 后端 | `internal/lsp/lsp.go` | LSP 会话管理：spawn、send、kill、`Content-Length` 分帧、stdout→事件、stderr 收集 |
| Go 后端 | `app.go` (LSP 段 370-421) | Wails 暴露 `lsp_detect / lsp_host_pid / lsp_resolve_root / lsp_spawn / lsp_send / lsp_kill` |
| Go 后端 | `internal/types/types.go` | `LspSpawnArgs` / `LspExitInfo` 传输结构 |
| 前端 | `lib/transport.ts` | `TauriLspTransport`：Wails Channel 双向桥、server→client 请求应答（config/registerCapability） |
| 前端 | `lib/client.ts` | `TeraxLspClient extends LanguageServerClient`：封装 def/refs/reference、init 能力、didClose/didSave，及交互扩展 |
| 前端 | `lib/sessionManager.ts` | 会话生命周期：按 `preset+root` 复用会话、引用计数、空闲/崩溃退避、OOM 保护 |
| 前端 | `lib/useLspExtension.ts` | React Hook：文档挂载时按语言/激活状态获取 LSP 扩展 |
| 前端 | `lib/locationsPanel.ts` | 底部"位置结果面板"（多定义/多引用选择） |
| 前端 | `lib/navigator.ts` | 跨文件跳转回调注入点（`openFile(path, line)`） |
| 前端 | `lib/uri.ts` | `file://` URI ⟷ 本地路径互转（平台差异处理） |
| 前端 | `lib/presets.ts` | 每种语言的 server 启动参数（command/args/rootMarkers/languages 映射） |
| 前端 | `lib/runtimeStore.ts` | 运行时状态（会话状态、崩溃计数、generation 触发重绑） |

---

## 2. LSP 协议细节（wire format）

### 2.1 传输分帧（Go 后端 `Session.read`）

LSP 使用 `Content-Length` 头 + 空行 + JSON 体的 HTTP 风格分帧。后端只做搬运：

```
Content-Length: 123\r\n
\r\n
{"jsonrpc":"2.0","id":1,"method":"textDocument/definition","params":{...}}
```

前端发出的**完整消息**（`transport.send(message)`）已包含头部，后端 `Manager.Send` 直接写入 stdin。

### 2.2 查看定义请求

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "textDocument/definition",
  "params": {
    "textDocument": { "uri": "file:///C:/proj/src/foo.ts" },
    "position": { "line": 12, "character": 5 }   // 0-based
  }
}
```

响应可能为：
- `null` / `[]`：无定义
- 单个 `Location`
- `Location[]`：多个定义（重载 / 接口实现）
- `LocationLink[]`：带 `targetSelectionRange`，可精确选中整个符号名

`client.ts` 的 `normalizeLocations()` 统一归一化成 `LspLocation[]`。

### 2.3 查找所有引用请求

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "textDocument/references",
  "params": {
    "textDocument": { "uri": "file:///C:/proj/src/foo.ts" },
    "position": { "line": 12, "character": 5 },
    "context": { "includeDeclaration": true }
  }
}
```

返回 `Location[]`（**作用域过滤由服务器完成**，前端不做正则搜索）。

### 2.4 悬停预览（可选前置，建议一并实现）

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "textDocument/hover",
  "params": {
    "textDocument": { "uri": "file:///C:/proj/src/foo.ts" },
    "position": { "line": 12, "character": 5 }
  }
}
```

返回 `Hover`：`{ contents: MarkupContent|MarkedString[] , range? }`。
`codemirror-languageserver` 已内置 hover 能力（我们只需样式层 `hoverCodeHighlight`）。

> 系列：`textDocument/definition`、`textDocument/references`、`textDocument/hover`、
> `textDocument/rename`（`F2`）均由同一个 client 提供，一起打通最划算。

---

## 3. 前端交互实现（已有，含三入口）

`client.ts` 的 `lspInteractions()` 返回一组 CodeMirror `Extension`：

### 3.1 三个触发入口

| 入口 | 实现 |
|------|------|
| **F12 查看定义** | `keymap` → `gotoDefinition(view, selection.head)` |
| **Shift+F12 引用** | `keymap` → `findReferences(view, selection.head)` |
| **Ctrl/Cmd+点击** | `mousedown` handler：按下时若 `metaKey||ctrlKey && button===0`，`posAtCoords` 取符号，`gotoDefinition` |
| **右键菜单** | 通过 CodeMirror context menu 扩展接入（见 §6 增强项） |
| **Ctrl/Cmd+悬停下划线** | `linkHover`：`mousemove` 时若按住 mod 键，`wordAt(pos)` 画下划线（`cm-lsp-link`） |

### 3.2 定位与跳转 `navigate()`

- **同文件**：`loc.uri === documentUri` → 计算目标偏移 → `view.dispatch({selection, scrollIntoView(center)})`。
- **跨文件**：`opts.onExternal(uri, line)` → `navigator.ts` → 打开目标文件、跳到行号。

### 3.3 多结果面板 `showResults()`

- 0 个结果 → 静默。
- 1 个结果 → 直接跳转。
- 多个结果 → 去重（按 `label`）后打开 `locationsPanel`：
  - 标题形如 `Definitions (3)` / `References (8)`
  - 键盘：`↑/↓` 移动，`Enter` 跳转，`Esc` 关闭
  - 相对路径 + 行号展示（如 `src/foo.ts:13`）

### 3.4 光标位置换算

`positionAt(view, pos)`：`lineAt(pos)` 得 `line.number-1`，`character = pos - line.from`（0-based，LSP 要求）。

---

## 4. 后端（Go）实现要点（已有）

`internal/lsp/lsp.go` 已实现完整骨架：

- `Spawn(ctx, m, args)`：`exec.CommandContext` 隐藏子进程窗口（`sysproc.HideWindow`），合并环境变量，起 **reader goroutine**（解析 `Content-Length` 分帧 → `EventsEmit`）与 **wait goroutine**（进程结束 → `LspExitInfo`，含 stderr 尾部 64KiB `cappedBuffer`）。
- `Manager`：`sessionId → *Session` 注册表，`seq` 自增 ID。
- `Send / Kill / KillAll / Detect / HostPID / ResolveRoot`。
- 模块化测试桩：`stat / parentDir / os_env / getpid` 全部可覆写。

### 4.1 若要新增能力协商透传（增强项）

目前 `Manager.Send` 原样转发消息。若要支持**动态注册**（`client/registerCapability`）或服务端主动请求，
前端 `transport.answerServerRequest` 已能应答 `workspace/configuration`、`client/registerCapability` 等，
后端无需改协议层。

---

## 5. 会话生命周期与可靠性（已有，方案中重点强调）

`sessionManager.ts` 是全项目最关键的可靠性设计，任何对 defs/refs 的增强都要照顾它：

- **按 `preset.id + root` 复用会话**：同一项目所有文件共享一个服务器进程，避免每文件一个进程烧内存。
- **引用计数**：`refs: Map<uri,count>`，文档打开 +1、关闭释放，`didClose` 正确下发。
- **空闲回收**：无引用 3 分钟后 `shutdownGracefully` → `transport.close`。
- **崩溃退避**：5 分钟内 >3 次崩溃则放弃（`crashedOut`），指数退避 `[2s,10s,30s]`，`generation` 触发文档重绑。
- **OOM 保护**：`maxMemoryMb`（如 ts 服务器 3GB）；预算击杀不自动重启。
- **优先级踢除**：每 preset 最多 4 会话，超限踢最老空闲会话。
- **退出竞态**：子进程瞬时死亡（rustup proxy 等）时 `handleServerExit` 兜底回收。

---

## 6. 增强项 / 待办（方案落地清单）

虽然 defs/refs 核心链路已通，以下可作为迭代清单补强体验：

1. **右键"转到定义 / 查找引用"菜单**
   - 用 CodeMirror 的 `EditorView.domEventHandlers({ contextmenu })` 或 editor context menu 扩展
   - 复用现有 `gotoDefinition / findReferences`，不必改后端。

2. **引用统计 / 引用高亮**
   - 对当前符号用 `textDocument/references` 拉单文件结果，渲染 `Decoration`（背景高亮）。

3. **符号大纲 / 文档符号（可选）**
   - `textDocument/documentSymbol` → 侧边栏大纲树，点击跳 `Location`（复用 `navigate`）。

4. **跳转历史（前进/后退）** ✅ 已实现
   - 新建 `lib/jumpHistory.ts`：全局 `JumpHistory` 栈（纯类 + 单例，含 `jumpHistory.test.ts`）。
   - 每次 def/refs 跳转前记录 cursor 位置，`Ctrl/Cmd+Alt+ArrowLeft/Right` 前进后退。
   - 为避免与 CodeMirror 默认 `Alt+箭头` 语法节点移动冲突，改用 `CtrlCmd+Alt+箭头`。
   - 同文件精确 selection；跨文件统一走 `getLspNavigator().openFile`。（注意 `opts.onExternal` 已并入后者）

5. **跳失回退（跨文件打开失败）**
   - `onExternal` 打开文件失败时，`fileUriToPath` 已做平台归一，可在 navigator 层加 toast 提示与绝对路径兜底。

6. **诊断与状态可见性**
   - `runtimeStore` 已有 `LspStatusPill` 展示会话/崩溃状态，建议在"引用面板"也显示所属服务器与根目录，方便排查"为什么没有结果"。

7. **测试**
   - 现有 `uri.test.ts`、`lspSwitchState.test.ts` 为样板。建议为 `client.ts` 的 `normalizeLocations`、`positionAt`、`showResults` 排序去重补单元测试。

---

## 7. 交付验收标准（Definition/References）

- [ ] Ctrl/Cmd+点击符号 → 同文件跳转定义，光标停在目标符号，并居中滚动。
- [ ] 跨文件定义（如 import 的 export）→ 自动打开目标文件并定位。
- [ ] F12 / 右键"转到定义"可触发同上。
- [ ] Shift+F12 / 右键"查找所有引用"→ 底部面板列出所有引用（含声明，`includeDeclaration:true`），可键盘/点击跳转。
- [ ] 同名不同作用域变量**不**被误报为引用（由服务器作用域分析保证）。
- [ ] 无服务器 / 服务器崩溃 / 未找到根目录时静默降级，不破坏编辑。
- [ ] 悬停可见类型签名（`textDocument/hover`）。
- [ ] 多个会话（多仓库）并存，互不干扰；空闲 3 分钟自动回收子进程。
- [ ] 键盘 Esc 可关闭结果面板并聚焦回编辑器。

---

## 8. 参考：消息/请求映射速查

| 功能 | LSP method | 前端 client 方法 | 后端 | 面板 |
|------|-----------|-----------------|------|------|
| 查看定义 | `textDocument/definition` | `textDocumentDefinition` | lsp.go | `locationPanel`（多结果时）|
| 查找引用 | `textDocument/references` | `textDocumentReferences` | lsp.go | `locationPanel` |
| 悬停 | `textDocument/hover` | (库内置) | lsp.go | CodeMirror tooltip |
| 重命名 | `textDocument/rename` | `renameSymbol` | lsp.go | 内联重命名框 |
| 格式化 | `textDocument/formatting` | `textDocumentFormatting` | lsp.go | 直接应用 edits |

---

*归档：`docs/definition-and-references.md`*
*状态：核心功能已在 `frontend/src/modules/lsp/*` 落地；本文档作为设计与迭代基线。*
