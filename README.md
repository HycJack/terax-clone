# Terax

A cross-platform desktop **terminal emulator + AI coding assistant** built with [Wails](https://wails.io) (Go) and a React/TypeScript frontend.

> **这是 Terax 的 Go 克隆版本。** The original [Terax](https://github.com/crynta/terax-ai) is a Rust (Tauri) app; this project ports its frontend to a Go (Wails v2) backend. A big thank-you to the original author and contributors.
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

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25 + Wails v2 |
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
- [Wails CLI v2](https://wails.io/docs/gettingstarted/installation)

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

## Development

Start the live development server with hot-reload:

```bash
wails dev
```

This will:
1. Start a Vite dev server for the React frontend
2. Run the Go backend with a native window
3. Enable hot-reload for frontend changes

The Vite server is also reachable at `http://localhost:34115` for browser-only work (Go methods still require the Wails window).

## Building

Build a redistributable production binary:

```bash
wails build
```

`wails build` runs `pnpm build` (frontend → `frontend/dist`), embeds the assets into the Go binary, and packages the app under `build/bin/`. `frontend/dist` is a build artifact and is not committed.

## Project Structure

```
terax/
├── main.go                 # Entry point, Wails app configuration
├── app.go                  # Main App struct — Wails bindings (store/secrets/git/pty/…)
├── helpers.go              # Internal helpers (store, secrets, workspace init)
├── localfile.go            # Local file serving for the frontend
├── internal/
│   ├── agent/              # AI agent hook state + window events
│   ├── events/             # Event-bridge for settings page store/secrets ops
│   ├── fs/                 # File system ops (read/write/search/grep/glob/watch)
│   ├── git/                # Git integration (status/diff/commit/log/…)
│   ├── history/            # Shell/command history
│   ├── lsp/                # Language Server Protocol support
│   ├── net/                # AI HTTP streaming bridge
│   ├── pty/                # PTY terminal management (ConPTY / forkpty)
│   ├── secrets/            # OS keyring wrapper
│   ├── shell/              # Shell command/session execution
│   ├── store/              # JSON-bag persistence (atomic, path-guarded)
│   ├── sysproc/            # Platform process helpers
│   ├── types/              # Shared type definitions
│   ├── winctrl/            # Window chrome controls
│   └── workspace/          # Workspace & WSL management
├── frontend/
│   ├── src/
│   │   ├── modules/        # Feature modules: ai, terminal, editor, explorer,
│   │   │                   #   source-control, git-history, tabs, settings, …
│   │   ├── lib/            # Utilities + Wails runtime shim
│   │   └── styles/         # Global styles / terminal themes
│   └── dist/               # Built frontend (embedded, not committed)
├── docs/                   # Architecture & design documents
├── build/                  # Build artifacts
└── wails.json              # Wails project configuration
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
