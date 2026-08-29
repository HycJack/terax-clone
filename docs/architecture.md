# Terax 架构设计与实现文档

> 本文档基于当前代码库整理，描述 Terax 的整体架构、前后端通信、模块职责与关键实现。
> 覆盖范围：Go 后端（`main.go` / `app.go` / `helpers.go` / `internal/*`）、前端桥梁层
> （`@/lib/wails/core`）以及二者之间的数据契约。

---

## 1. 项目概览

Terax 是一款跨平台桌面终端 + AI 编码助手，技术栈为 **Wails v3（Go + React/TypeScript + Vite）**。

核心能力：

- **内置 PTY 终端**：多 shell（cmd / PowerShell / WSL / bash / zsh / fish）
- **AI 编码 Agent**：LLM 流式补全、工具执行、工作区文件上下文
- **文件浏览器**：目录浏览、全文搜索（grep/glob）、文件监听（fsnotify）
- **Git 集成**：status / diff / stage / commit / log / branch 管理
- **LSP 集成**：JSON-RPC over stdin/stdout，语言智能（查看定义、引用等）
- **工作区管理**：目录授权 + WSL 分发版支持
- **密钥管理**：系统 keyring（Windows Credential Manager / macOS Keychain / Linux Secret Service）
- **系统托盘**：最小化到托盘，快速显示/隐藏
- **多窗口**：原生 OS 窗口管理
- **应用菜单栏**：全功能键盘快捷键

### 1.1 启动流程

```
main.go  application.New(...)
   │  application.NewWindow(...) → mainWindow
   │  application.SystemTray.New(...) → tray
   │  buildAppMenu(wailsApp) → menu
   ├─ Services: []application.Service{application.NewService(app)}
   │     · app.ServiceStartup(ctx, opts)
   │       · a.ctx = ctx
   │       · a.fsWatcher.BindContext(ctx)
   │       · appDir() 建目录 → storeInit / historyInit
   │       · workspace.InitLaunchCwd(home)
   │       · winctrl.Register(ctx)
   │       · events.RegisterAll(ctx)
   │       · events.RegisterMcp(ctx, app)  // MCP Server (JSON-RPC 2.0)
   │     · app.OnShutdown → app.shutdown()
   │       · ptyMgr.CloseAll() / lspMgr.KillAll() / fsWatcher.Close()
   └─ Window creation with WebviewWindowOptions (URL, title, size, background)
```

---

## 2. 分层与目录结构

```
terax/
├── main.go          # 入口：Wails v3 app 配置（窗口、菜单、托盘、生命周期）
├── app.go           # App 结构体 —— Wails v3 Service 绑定（命令转发中心）
├── helpers.go       # 顶层辅助函数（store/secrets/net/history/autostart/opener 等转发）
├── localfile.go     # 本地文件服务（/local-file/ 路径拦截）
├── Taskfile.yml     # 构建任务（pnpm + wails3）
├── wails.json       # Wails v3 项目配置
├── internal/
│   ├── agent/       # AI Agent hooks 就绪状态 + 信号事件
│   ├── events/      # 设置页事件桥 + MCP Server (JSON-RPC 2.0)
│   ├── fs/          # 文件系统：读写、目录、搜索、grep、glob、fsnotify 监听
│   ├── git/         # Git CLI 封装：status/diff/stage/commit/log/branch/push/fetch
│   ├── history/     # 命令历史（内存缓存 + 每行一条的持久化文件）
│   ├── lsp/         # LSP 会话管理：spawn/send/kill、Content-Length 分帧
│   ├── mcp/         # MCP Server：JSON-RPC 2.0 over stdio
│   ├── net/         # AI HTTP 客户端：流式代理、SSRF 防护、LM 健康检查
│   ├── pty/         # 伪终端：ConPTY(Windows)/openpty(Unix)/旧版管道回退
│   ├── secrets/     # 系统 keyring 封装
│   ├── shell/       # 一次性命令 / 后台任务 / 交互会话
│   ├── store/       # LazyStore JSON 持久化（带路径穿越防护）
│   ├── sysproc/     # 跨平台进程辅助（隐藏窗口、进程树击杀）
│   ├── types/       # 共享数据结构（与前端 TS 类型一一对应）
│   ├── winctrl/     # 窗口控制事件处理（v3 events）
│   └── workspace/   # 目录授权注册表、当前 cwd、WSL 分发版
├── frontend/
│   ├── bindings/    # 自动生成的 Wails v3 TypeScript 绑定
│   └── src/
│       ├── lib/wails/core.ts   # ★ 桥梁层：Tauri 风格 invoke → Wails v3 绑定
│       ├── modules/            # 功能模块（ai/terminal/explorer/source-control/lsp/...）
│       ├── settings/           # 设置页（独立路由）
│       ├── app/ components/    # 全局组件
│       └── styles/ main.tsx    # 入口
├── build/           # 构建资源（图标、appicon.png）
└── docs/
    ├── architecture.md          # 本文档
    └── definition-and-references.md  # LSP「查看定义/引用」特性设计
```

---

## 3. 前后端通信（核心桥梁 `@/lib/wails/core`）

Wails v3 使用 `@wailsio/runtime@3.0.0-beta.15`，前端面向 **Tauri 式的命令面**（snake_case 命令名 + `invoke` + `Channel`），
后端是 **Wails v3 绑定**（PascalCase 方法名 + 结构体参数 + `$Call.ByID(numericHash, args)`）。
`core.ts` 完成翻译。

### 3.1 命令解析流程

```
invoke("pty_open", args)
   │ 1. SPECIAL[cmd]？ → 先做副作用（注册 Channel 监听、生成事件名）
   │ 2. serializeArgs() → 剔除 Channel/函数等无法 JSON 序列化的字段
   │ 3. resolveCommand() ：
   │      · 先精确匹配 App 上的 PascalCase 方法
   │      · 否则 snake_case → PascalCase（维护 FF/URL/PID 缩写映射）
   │ 4. SINGLE_ARG[cmd] ？ → 只从载荷中提取单个字段当参数
   │      （如 fs_canonicalize:"path"、pty_close:"id"）
   │     否则：把整个 payload 作为结构体参数
   │ 5. 通过 generated bindings 调用 `$Call.ByID(numericHash, args)`
```

### 3.2 Channel 处理（`SPECIAL` 表）

部分命令需要流式/推送语义，Tauri 用 `Channel<T>`。`core.ts` 把 Channel 转换为
**唯一事件名**，并预注册事件监听，最后改写 args 里的字段名让 Go 侧接收事件名：

| 命令 | 前端字段 → Go 字段 | 事件命名空间 |
|------|---------------------|--------------|
| `lsp_spawn` | `onMessage` → `onMessageEvent`，`onExit` → `onExitEvent` | `lsp:msg` / `lsp:exit` |
| `ai_http_stream` | `onEvent` → `onEventEvent` | `net:ai` |
| `agent_enable_hooks` | `onHooksReady` → `onHooksReadyEvent` | `agent:ready` |

### 3.3 事件总线（非请求/响应）

Wails v3 事件 API：`app.Event.Emit(name, data)` / `app.Event.On(name, cb)`。
前端 `EventsOn`/`EventsOnce` 自动解包 `WailsEvent` wrapper。

- **PTY 输出**：Go 端 pump 每次读到数据 → `app.Event.Emit("pty:<id>", {data: "<base64>"})`
  - 前端订阅 `pty:<id>` 与 `pty:exit:<id>`，base64 解码后喂给 xterm。
- **文件变更**：fsnotify 监听 → 合并 50ms 突发 → `app.Event.Emit("fs:changed", paths)`
- **Agent 信号**：`terax:agent-signal`
- **菜单事件**：`menu:new-terminal`、`menu:open-folder`、`menu:settings` 等
- **设置页事件桥**：`store:load/save`、`secrets:set/get/delete/getAll` 及各自的 `:result`

### 3.4 设置页的特殊性

设置页通过新窗口 `settings.html?tab=...` 打开，不再导航销毁绑定。
`openSettingsWindow.ts` 调用 Go IPC `OpenSettingsWindow` 创建新窗口或聚焦已有窗口。

---

## 4. App 结构体（`app.go`） —— 命令分组

`App` 持有四个管理器，所有 Wails 绑定方法按业务域分组转发到 `internal/*`：

| 分组 | 关键方法 |
|------|---------|
| 启动目录 | `GetLaunchDir` / `GetLaunchFiles` |
| PTY | `PtyOpen` / `PtyWrite` / `PtyResize` / `PtyStart` / `PtyClose` / `PtyCloseAll` / `PtyHasForegroundProcess\|Job` / `PtyShellName` / `PtyListShells` |
| FS | `FsReadDir` / `FsReadFile` / `FsWriteFile` / `FsCreateFile` / `FsCreateDir` / `FsRename` / `FsDelete` / `FsCopy` / `FsStat` / `FsCanonicalize` / `FsWatchAdd` / `FsWatchRemove` |
| FS 搜索 | `FsSearch` / `FsListFiles` / `FsGrep` / `FsGrepInteractive` / `FsGlob`（`ListSubdirs` 供面包屑） |
| LSP | `LspDetect` / `LspHostPID` / `LspResolveRoot` / `LspSpawn` / `LspSend` / `LspKill` |
| Git | `GitResolveRepo` / `GitPanelSnapshot` / `GitStatus` / `GitDiff` / `GitDiffContent` / `GitStage` / `GitUnstage` / `GitDiscard` / `GitCommit` / `GitFetch` / `GitPullFFOnly` / `GitPush` / `GitLog` / `GitShowCommit` / `GitCommitFiles` / `GitCommitFileDiff` / `GitRemoteURL` / `GitListBranches` / `GitCheckoutBranch` |
| Shell | `ShellRunCommand` / `ShellSessionOpen` / `ShellSessionRun` / `ShellSessionClose` / `ShellBgSpawn` / `ShellBgLogs` / `ShellBgKill` / `ShellBgList` |
| 工作区/WSL | `WorkspaceAuthorize` / `WorkspaceCurrentDir` / `WorkspaceSetCwd` / `WorkspacePickDirectory` / `WslListDistros` / `WslDefaultDistro` / `WslHome` |
| Agent | `AgentEnableHooks` / `AgentHooksStatus` |
| Secrets | `SecretsGet` / `SecretsSet` / `SecretsDelete` / `SecretsGetAll` |
| 网络 | `LmPing` / `AiHttpRequest` / `AiHttpStream` |
| 历史 | `HistorySuggest` / `HistoryCommands` / `HistoryRecord` / `HistoryList` |
| 进程/系统 | `ProcessRelaunch` / `ProcessExit` / `Autostart*` / `UpdaterCheck` / `Opener*` |
| 持久化 | `StoreLoad` / `StoreSave` / `AppConfigDir` / `AppDataDir` / `AppHomeDir` |
| 杂项 | `OpenSettingsWindow` |

---

## 5. 各模块关键实现

### 5.1 `internal/pty` —— 伪终端

- **后端选择**：
  - Windows 10 1809+/Server 2019+ → ConPTY（`aymanbagabas/go-pty`）
  - 旧版 Windows → 纯 `exec.Cmd` + 匿名管道（无真实 PTY 语义）
  - Unix → `openpty`（`creack/pty`）
- **`Session.startChild`** 在会话创建时按 shell 注入集成参数（bash 用 `--rcfile`，
  PowerShell 用 `-File profile.ps1`），并用 OSC 7（cwd）/ OSC 133（markers）让前端同步目录树。
- **输出泵**：每个会话一个 `pump` 协程，读 master → base64 封装 → `EventsEmit("pty:<id>", {data})`。
  生命周期：`Open`（注册会话）→ `Start`（启动 pump）→ `Write/Resize` → `Close`（关 master + 进程树击杀）。
- **退出**：读循环结束 → `s.proc.Wait()` → `EventsEmit("pty:exit:<id>", code)`。
- 默认 shell：Windows 优先 `pwsh > powershell > cmd`，Unix 用 `$SHELL`。

### 5.2 `internal/fs` —— 文件系统 + 监听

- 统一把路径反斜杠转 `/` 返回前端，保持与 Rust 后端契约一致。
- **授权**：`Fs*` 写操作入口均过 `workspace.IsAuthorized`（见 §6 安全）。
- **ReadDir**：可选 git 装饰（`git check-ignore --no-index`）标注被忽略项。
- **grep**：优先 `rg`（`--fixed-strings` 按字面子串匹配，`-i` 控制大小写），否则回退 Go 行扫描。
- **ListFiles / Search**：`filepath.WalkDir` 收集到上限即截断。
- **监听**：单例 `WatcherManager`（fsnotify），50ms 突发合并，事件 → `fs:changed`。

### 5.3 `internal/net` —— AI HTTP 客户端

- `newClient(allowPrivate)`：自定义 DialContext，**默认拒绝私网/环回/链路本地**（SSRF 防护），
  `allowPrivateNetwork` 显式开启放行。
- `AiHTTPRequest`：单次请求，10MB 响应上限，返回 `{status, headers, body}`。
- `AiHTTPStream`：流式读 body（8KB 块）→ 分片 `EventsEmit(args.OnEventEvent, AiStreamEvent)`，
  事件 `kind` 有 `headers/chunk/end/error`；非 2xx 读取限 1MB 错误体。
- `LmPing`：2s 超时健康检查，返回状态码。

### 5.4 `internal/git` —— Git CLI 封装

- 全部委托系统 `git`（不引 go-git），cwd 定位到仓库根。
- 关键：`run()` 合并 stdout/stderr 报错；`ResolveRepo` 用 `rev-parse --show-toplevel`，
  不在仓库内返回 `nil`（模拟 Rust `Option`）。
- 命令较多但模式统一：`exec.CommandContext(ctx, "git", args...)` + `sysproc.HideWindow`。

### 5.5 `internal/lsp` —— LSP 会话

- `Spawn` 起子进程，`Content-Length` 分帧读 stdout → `EventsEmit(onMessageEvent, body)`。
- `wait` 协程等退出，收集 stderr 尾部（cappedBuffer，默认 64KB）→ `EventsEmit(onExitEvent, LspExitInfo)`。
- `mergeEnv` 合并自定义环境变量。
- 注：`lsp.go` 顶部有一组 `stat/parentDir/os_env/getpid` 的可替换函数变量（便于测试桩）。

### 5.6 `internal/shell` —— 命令/后台任务/会话

与 `pty` 职责有重叠的另一套进程执行层：

- `RunCommand`：一次性 `exec.CommandContext`，返回合并输出。
- `BgSpawn`：后台任务，捕获输出到 Job 缓冲，提供 `BgLogs`/`BgKill`/`BgList`。
- `OpenSession`/`RunInSession`：**每个命令新起一个 shell 进程**，靠内存里的 `Session{cwd}`
  维护跨命令的当前目录（避免交互式回显污染）。

### 5.7 其它模块

- **`workspace`**：`allowed map[string]bool` 授权目录注册表 + 全局 `cwd`；
  `IsAuthorized` 用 `prefix + separator` 前缀匹配防 `/a/b` 越到 `/a/bc`；
  `ResolveWorkspaceEnv` 归一化 WSL 环境；WSL 相关操作仅 Windows 生效。
- **`secrets`**：`go-keyring`，按 `(service, account)` 存取；`SetService` 切换服务名。
- **`store`**：JSON 包写盘为 `<path>.json`，**拒绝 `..`/分隔符**再做 Abs 前缀校验，原子写（tmp + rename）。
- **`history`**：内存缓存 + 每行一条文件持久化；去重、前缀/模糊建议。
- **`winctrl`**：v3 events 事件处理（close/minimise/maximise/unmaximise/unminimise/fullscreen/force-reload）。
- **`agent`**：内存 `enabled` 标志 + `terax:hooks-ready` / `terax:agent-signal` 事件。
- **`events`**：设置页用的 store/secrets 事件桥处理器 + MCP Server (JSON-RPC 2.0 over stdio)。

---

## 6. 安全设计

| 关注点 | 实现 | 位置 |
|--------|------|------|
| SSRF | 默认拒绝私网/环回/链路本地 | `net/newClient` |
| 路径授权 | 读写删改、搜索、grep、glob 入口均校验 `workspace.IsAuthorized` | `app.go` Fs* 各方法 |
| 路径逃逸 | store 拒绝 `..`+分隔符并按 Abs 前缀复核 | `store.safePath` |
| URL scheme | 只放行 `http/https/mailto/tel` | `helpers.openerOpenURL` |
| 密钥 | 系统 keyring，不落明文 | `secrets` |

> 说明：默认 `InitLaunchCwd` 会**自动授权 home 目录**，即默认允许读写用户文件——
> 这是终端类应用的既定设计，而非沙箱。若未来推出「受限模式」需重新评估授权边界。
> 另注意 `FsCopy`/`WriteFile` 的授权校验基于入口路径，未阻止源目录内符号链接跟随到目录外（低危）。

---

## 7. 数据契约

- **返回类型**以前端 TS 类型与 Go 结构体 **JSON tag** 为准，字段名 camelCase。
  例如 `DirEntry{name, kind, size, mtime, gitignored}`、`FsGrepResponse{hits, truncated, filesScanned}`。
- 命令名：前端 snake_case → 后端 PascalCase（缩写映射 FF/URL/PID）。
- 事件负载：PTY 用 base64 包裹原始字节；其余多为 `{data}`/`{result}` 信封或 `{success, error}`。

> 历史 `audit_report.txt` / `audit_return_types.py` / `check_fields.py` 描述的前后端不匹配
> 均为**早期误报**（引用已删除的 `wails-shim/core.ts`），已随本次整理删除；当前契约以
> `@/lib/wails/core` 的 `SPECIAL`/`SINGLE_ARG` 与 `internal/types` 为准。

---

## 8. 已知设计取舍与待清理项

- `pty` 与 `shell` 职责重叠（都能跑进程/会话），边界可合并。
- `helpers.go:autostartSet` / `workspace.WSL*` 通过 `exec.Command` 调系统命令，无 shell 展开故无注入风险。
- `ProcessExit` 忽略传入的 `Code`。
- `FsGrepInteractive` 与 `FsGrep` 等价（流式版本为已知简化）。