# Terax

A cross-platform desktop terminal and AI coding assistant built with [Wails](https://wails.io) (Go + React/TypeScript).

## Overview

Terax is a modern developer tool that combines a feature-rich terminal emulator with AI-assisted coding capabilities. It provides a native desktop experience with a web-based frontend, offering:

- **Built-in PTY terminal** with multiple shell support (cmd, PowerShell, WSL, bash, zsh)
- **AI coding agent** with LLM integration (streaming completions, tool execution)
- **File explorer** with search, grep, glob, and file watching
- **Git integration** (status, diff, staging, commit, log, branch management)
- **LSP integration** for language intelligence
- **Workspace management** with directory authorization and WSL support
- **Secrets management** via OS keyring

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go + Wails v2 |
| Frontend | React + TypeScript + Vite |
| Terminal | PTY via `conpty` (Windows) / `forkpty` (Unix) |
| AI | OpenAI-compatible HTTP streaming |
| LSP | JSON-RPC over stdin/stdout |
| Build | Wails CLI (`wails build`, `wails dev`) |

## Prerequisites

- [Go](https://go.dev/dl/) 1.21+
- [Node.js](https://nodejs.org/) 18+
- [pnpm](https://pnpm.io/installation)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

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
2. Run the Go backend with native window
3. Enable hot-reload for frontend changes

For browser-only frontend development, the Vite server is also accessible at `http://localhost:34115`.

## Building

Build a redistributable production binary:

```bash
wails build
```

The output binary will be placed in the `build/` directory.

## Project Structure

```
terax/
├── app.go                 # Main Wails App struct — binds all Go commands
├── main.go                # Entry point, Wails app configuration
├── helpers.go             # Internal helpers (store, secrets, etc.)
├── cmd/                   # CLI-related subcommands
├── internal/
│   ├── agent/             # AI agent hook management
│   ├── fs/                # File system operations (read, write, search, grep)
│   ├── git/               # Git integration
│   ├── lsp/               # Language Server Protocol support
│   ├── pty/               # PTY terminal management
│   ├── shell/             # Shell command/session execution
│   ├── types/             # Shared type definitions
│   └── workspace/         # Workspace & WSL management
├── frontend/
│   ├── src/
│   │   ├── modules/ai/    # AI agent UI & tool definitions
│   │   ├── components/    # React components
│   │   └── lib/           # Frontend utilities & Wails shim
│   └── dist/              # Built frontend (embedded in Go binary)
├── build/                 # Build artifacts
└── wails.json             # Wails project configuration
```

## Configuration

Edit `wails.json` to configure:

- App name and output filename
- Frontend install/build/dev commands
- Author metadata

See the [Wails project config docs](https://wails.io/docs/reference/project-config) for details.

## License

MIT
