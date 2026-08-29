# Terax

A cross-platform desktop **terminal emulator + AI coding assistant** built with [Wails v3](https://wails.io) (Go) and a React/TypeScript frontend.

> **这是 Terax 的 Go 克隆版本 (Wails v3 分支)。** The original [Terax](https://github.com/crynta/terax-ai) is a Rust (Tauri) app; this project ports its frontend to a Go (Wails v3) backend.
>
> **中文版说明**: 中文文档见 [README.zh-CN.md](./README.zh-CN.md)。

## Overview

Terax is a modern developer tool that puts a feature-rich terminal, a full file/git surface, and an AI coding agent into one native desktop app. The agent is **orchestrated entirely in the frontend** (Vercel AI SDK) with a thin Go backend that owns persistence, the OS keyring, events, and PTY/file bridges — so prompt and tool changes never require a Go rebuild.

Highlights:

- **Built-in PTY terminal** — cmd / PowerShell / WSL / bash / zsh, xterm.js rendering, multiple panes & tabs
- **AI coding agent** — streaming chat, tool calling, mutation approvals, read-before-edit, plan mode, subagents
- **Model flexibility** — OpenAI, Anthropic, Google, xAI, Groq, Cerebras, DeepSeek, OpenRouter, plus local endpoints (LM Studio / MLX / Ollama / any OpenAI-compatible server)
- **File explorer & search** — tree, grep, glob, file watching, quick open
- **Git integration** — status, diff, staging, commit, history graph, branch management
- **Editor** — CodeMirror 6 based with LSP support and markdown preview
- **LSP integration** — language intelligence over JSON-RPC
- **Workspace management** — per-directory authorization, environments, WSL support
- **Secrets management** — provider API keys in the OS keyring, never on disk in plaintext
- **Sessions** — multi-session chat history, auto-titling, persistence
- **System tray** — minimize to tray, quick show/hide
- **Multi-window** — native OS window management via Wails v3
- **Native menu bar** — keyboard shortcuts for all major actions

## Wails v3 Migration (This Branch)

This branch (`wails3`) migrates the project from Wails v2 to Wails v3. Key changes:

### Backend (Go)

| Change | Details |
|--------|---------|
| **Wails v3 API** | `application.New()` replaces `wails.Run()`, `Services` replaces `Bind` |
| **Service lifecycle** | `ServiceStartup(ctx, opts)` replaces `OnStartup`; `OnShutdown` is a field |
| **Events** | `app.Event.Emit(name, data)` / `app.Event.On(name, cb)` replaces `runtime.EventsEmit/On` |
| **Window** | `app.Window.NewWithOptions()` for multi-window support |
| **System tray** | `app.SystemTray.New()` with menu and click handler |
| **Application menu** | Custom menu bar with File/Edit/View/Terminal/Help, all with accelerators |
| **Dialog** | `SetDirectory` replaces `SetDefaultDirectory`; `PromptForSingleSelection` replaces `PickDirectory` |
| **RGBA** | `Red, Green, Blue, Alpha` fields; use `application.NewRGBA()` |
| **Middleware** | `AssetOptions.Middleware` for `/local-file/` path interception (replaces `Handler`) |

### Frontend (React/TypeScript)

| Change | Details |
|--------|---------|
| **Runtime bindings** | `@wailsio/runtime@3.0.0-beta.15` replaces Tauri-style shims |
| **IPC bridge** | `core.ts` uses `$Call.ByID(numericHash, args)` from generated bindings |
| **Event unwrapping** | Wails v3 delivers `WailsEvent {name, data}` — wrappers auto-unwrap to raw data |
| **Settings window** | Settings open in a new OS window (`settings.html?tab=...`) instead of in-app dialog |
| **Menu events** | Frontend listens to `menu:*` events for all menu actions |
| **Build system** | `Taskfile.yml` + `build/` directory for Wails v3 build pipeline |

### Menu Bar

| Menu | Items |
|------|-------|
| **terax** (macOS) | About, Settings…, Hide, Quit |
| **File** | New Window, Open Folder…, Close Window |
| **Edit** | Undo/Redo, Cut/Copy/Paste/Select All, Find… |
| **View** | Reload, Force Reload, Dev Tools, Toggle Sidebar, Toggle Zen Mode, Zoom, Fullscreen |
| **Terminal** | New Terminal, Split Terminal, Kill Terminal |
| **Help** | Documentation, About terax |

### Keyboard Shortcuts

| Action | Shortcut |
|--------|----------|
| Settings | `Cmd+,` |
| New Window | `Cmd+N` |
| Open Folder | `Cmd+O` |
| Find | `Cmd+F` |
| Toggle Sidebar | `Cmd+B` |
| Toggle Zen Mode | `Cmd+Shift+F` |
| New Terminal | `Cmd+`` ` |
| Split Terminal | `Cmd+Shift+`` ` |
| Kill Terminal | `Cmd+Shift+W` |

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25 + Wails v3 (beta.15) |
| Frontend | React 19 + TypeScript + Vite (rolldown) |
| AI orchestration | Vercel AI SDK (`ai` + `@ai-sdk/*`) — frontend-driven |
| Terminal | xterm.js + PTY via `go-pty` (ConPTY / forkpty) |
| Editor | CodeMirror 6 + LSP |
| State | zustand |
| Persistence | per-user JSON stores (Go side, atomic writes) |
| Secrets | OS keyring (`go-keyring`) |
| UI kit | shadcn/ui + Radix + Tailwind CSS 4 |

## Architecture

Two-layer split that keeps iteration fast:

- **Frontend owns the agent**: chat store, `Chat` runtime, context-aware transport, tool definitions with an approval policy, context compaction, session persistence.
- **Go owns the OS boundaries**: Wails bindings for store load/save, secrets (keyring), agent hook events, and PTY/file/workspace bridges.

Design docs live in [`docs/`](./docs/):

- [`agent-and-conversation-design.md`](./docs/agent-and-conversation-design.md) — agent design + conversation storage
- [`architecture.md`](./docs/architecture.md) — frontend-centric agent architecture blueprint

## Prerequisites

- [Go](https://go.dev/dl/) 1.25+
- [Node.js](https://nodejs.org/) `^20.19.0 || >=22.12.0`
- [pnpm](https://pnpm.io/installation)
- [Wails CLI v3](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

## Development

Start the live development server with hot-reload:

```bash
wails3 dev
```

This will:
1. Start a Vite dev server for the React frontend
2. Run the Go backend with a native window
3. Enable hot-reload for frontend changes

## Building

Build a redistributable production binary:

```bash
wails3 build
```

Output: `bin/terax`

## Project Structure

```
terax/
├── main.go                 # Entry point, Wails v3 app, menu, tray, window
├── app.go                  # Main App struct — Wails v3 service bindings
├── helpers.go              # Internal helpers (store, secrets, workspace init)
├── localfile.go            # Local file serving for the frontend
├── Taskfile.yml            # Build tasks (pnpm)
├── internal/
│   ├── agent/              # AI agent hook state + window events
│   ├── events/             # Event-bridge for settings page store/secrets ops
│   ├── fs/                 # File system ops (read/write/search/grep/glob/watch)
│   ├── git/                # Git integration (status/diff/commit/log/…)
│   ├── history/            # Shell/command history
│   ├── lsp/                # Language Server Protocol support
│   ├── mcp/                # MCP Server (JSON-RPC 2.0 over stdio)
│   ├── net/                # AI HTTP streaming bridge
│   ├── pty/                # PTY terminal management (ConPTY / forkpty)
│   ├── secrets/            # OS keyring wrapper
│   ├── shell/              # Shell command/session execution
│   ├── store/              # JSON-bag persistence (atomic, path-guarded)
│   ├── sysproc/            # Platform process helpers
│   ├── types/              # Shared type definitions
│   ├── winctrl/            # Window chrome controls (v3 events)
│   └── workspace/          # Workspace & WSL management
├── frontend/
│   ├── src/
│   │   ├── modules/        # Feature modules: ai, terminal, editor, explorer,
│   │   │                   #   source-control, git-history, tabs, settings, …
│   │   ├── lib/            # Utilities + Wails v3 runtime wrapper
│   │   └── styles/         # Global styles / terminal themes
│   ├── bindings/           # Auto-generated Wails v3 TypeScript bindings
│   └── dist/               # Built frontend (embedded, not committed)
├── build/                  # Build assets (icons, appicon.png)
├── docs/                   # Architecture & design documents
└── wails.json              # Wails v3 project configuration
```

## Configuration

Edit `wails.json` to configure:

- App name and output filename
- Frontend install/build/dev commands
- Author metadata

See the [Wails project config docs](https://wails.io/docs/reference/project-config) for details.

## License

MIT

## Credits & Acknowledgements

- This project is a **Go (Wails) clone** of [**Terax**](https://github.com/crynta/terax-ai) — an amazing Rust/Tauri terminal + AI coding assistant by [crynta](https://github.com/crynta). The frontend design, feature set, and architecture are inspired by / derived from the original project.
- Thanks to the original author and contributors of Terax for creating such a great tool and for the ideas this clone builds on.
