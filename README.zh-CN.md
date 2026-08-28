# Terax

一个基于 [Wails](https://wails.io)（Go）+ React/TypeScript 前端的跨平台桌面 **终端模拟器 + AI 编码助手**。

> **本项目是 Terax 的 Go 克隆版本。** 原版 [Terax](https://github.com/crynta/terax-ai) 是 Rust（Tauri）应用；本项目将其前端移植到 Go（Wails v2）后端。非常感谢原作者及所有贡献者。
>
> **English**: See [README.md](./README.md).

## 项目简介

Terax 把功能完备的终端、完整的文件/Git 操作界面和一个 AI 编码助手集成到一个原生桌面应用中。**AI agent 完全由前端编排**（基于 Vercel AI SDK），Go 后端只负责持久化、系统钥匙串、事件以及 PTY/文件桥接——因此调整提示词或工具时无需重新编译 Go 代码。

核心特性：

- **内置 PTY 终端** —— 支持 cmd / PowerShell / WSL / bash / zsh，基于 xterm.js 渲染，多面板、多标签
- **AI 编码助手** —— 流式对话、工具调用、写操作审批、read-before-edit、Plan 模式、子代理
- **多模型支持** —— OpenAI、Anthropic、Google、xAI、Groq、Cerebras、DeepSeek、OpenRouter，以及本地模型（LM Studio / MLX / Ollama / 任意 OpenAI 兼容服务）
- **文件管理与搜索** —— 目录树、grep、glob、文件监听、快速打开
- **Git 集成** —— 状态、diff、暂存、提交、历史图、分支管理
- **内置编辑器** —— 基于 CodeMirror 6，支持 LSP 与 Markdown 预览
- **LSP 集成** —— 通过 JSON-RPC 提供语言智能
- **工作区管理** —— 按目录授权、环境管理、WSL 支持
- **密钥管理** —— 各 provider 的 API key 存于系统钥匙串，磁盘不落明文
- **多会话** —— 多会话聊天历史、自动标题、持久化

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25 + Wails v2 |
| 前端 | React 19 + TypeScript + Vite（rolldown） |
| AI 编排 | Vercel AI SDK（`ai` + `@ai-sdk/*`），前端驱动 |
| 终端 | xterm.js + PTY（`go-pty`，ConPTY / forkpty） |
| 编辑器 | CodeMirror 6 + LSP |
| 状态管理 | zustand |
| 持久化 | 按用户 JSON 存储（Go 侧原子写入） |
| 密钥 | 系统钥匙串（`go-keyring`） |
| UI 组件库 | shadcn/ui + Radix + Tailwind CSS 4 |

## 架构

采用前后端分层，迭代非常快：

- **前端负责 agent**：聊天 store、`Chat` 运行时、上下文感知 transport、带审批策略的工具定义、上下文压缩、会话持久化。
- **Go 负责系统边界**：Wails 绑定（store 读写、密钥、agent 钩子事件）、PTY/文件/工作区桥接。

设计文档位于 [`docs/`](./docs/)：

- [`agent-and-conversation-design.md`](./docs/agent-and-conversation-design.md) —— agent 设计与会话存储
- [`architecture.md`](./docs/architecture.md) —— 前端为中心的 agent 架构蓝图
- [`definition-and-references.md`](./docs/definition-and-references.md) —— 术语与参考资料

## 环境要求

- [Go](https://go.dev/dl/) 1.25+
- [Node.js](https://nodejs.org/) `^20.19.0 || >=22.12.0`
- [pnpm](https://pnpm.io/installation)
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## 开发

启动带热重载的开发服务器：

```bash
wails dev
```

它会：
1. 启动 React 前端的 Vite 开发服务器
2. 以原生窗口运行 Go 后端
3. 对前端改动启用热重载

Vite 服务也监听在 `http://localhost:34115`，可仅用浏览器调试前端（调用 Go 方法仍需 Wails 窗口）。

## 构建

构建可分发的生产二进制：

```bash
wails build
```

`wails build` 会先执行 `pnpm build`（前端产物输出到 `frontend/dist`），再把资源嵌入 Go 二进制并打包到 `build/bin/`。`frontend/dist` 是构建产物，不纳入版本控制。

## 项目结构

```
terax/
├── main.go                 # 入口，Wails 应用配置
├── app.go                  # 主 App 结构体 —— Wails 绑定（store/secrets/git/pty/…）
├── helpers.go              # 内部辅助（store、secrets、工作区初始化）
├── localfile.go            # 前端本地文件服务
├── internal/
│   ├── agent/              # AI agent 钩子状态 + 窗口事件
│   ├── events/             # 设置页 store/secrets 操作的事件桥
│   ├── fs/                 # 文件系统操作（读/写/搜索/grep/glob/监听）
│   ├── git/                # Git 集成（status/diff/commit/log/…）
│   ├── history/            # Shell/命令历史
│   ├── lsp/                # 语言服务器协议支持
│   ├── net/                # AI HTTP 流式桥接
│   ├── pty/                # PTY 终端管理（ConPTY / forkpty）
│   ├── secrets/            # 系统钥匙串封装
│   ├── shell/              # Shell 命令/会话执行
│   ├── store/              # JSON 存储（原子写入、路径防护）
│   ├── sysproc/            # 平台进程辅助
│   ├── types/              # 共享类型定义
│   ├── winctrl/            # 窗口控制
│   └── workspace/          # 工作区 & WSL 管理
├── frontend/
│   ├── src/
│   │   ├── modules/        # 功能模块：ai、terminal、editor、explorer、
│   │   │                   #   source-control、git-history、tabs、settings 等
│   │   ├── lib/            # 工具 + Wails 运行时 shim
│   │   └── styles/         # 全局样式 / 终端主题
│   └── dist/               # 前端构建产物（内嵌，不入库）
├── docs/                   # 架构与设计文档
├── build/                  # 构建产物
└── wails.json              # Wails 项目配置
```

## 配置

编辑 `wails.json` 可配置：

- 应用名称与输出文件名
- 前端安装 / 构建 / 开发命令
- 作者信息

详见 [Wails 项目配置文档](https://wails.io/docs/reference/project-config)。

## 许可证

MIT

## 致谢

- 本项目是 [**Terax**](https://github.com/crynta/terax-ai) 的 **Go（Wails）克隆版本** —— 原版是由 [crynta](https://github.com/crynta) 开发的 Rust/Tauri 终端 + AI 编码助手。本项目的界面设计、功能集与架构均受到原项目启发 / 派生自原项目。
- 感谢 Terax 原作者及所有贡献者创造了这样出色的工具，也感谢它为这个克隆版本提供的思路与基础。
