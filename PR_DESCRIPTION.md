# Wails v3 Migration

## Summary

Migrate terax-clone from Wails v2 to Wails v3 (beta.15), modernizing the backend with the new service-based architecture and improving the frontend with proper TypeScript bindings.

## Changes

### Backend (Go)
- **Wails v3 API**: `application.New()` replaces `wails.Run()`, `Services` replaces `Bind`
- **Service lifecycle**: `ServiceStartup(ctx, opts)` replaces `OnStartup`; `OnShutdown` is a field
- **Events**: `app.Event.Emit(name, data)` / `app.Event.On(name, cb)` replaces `runtime.EventsEmit/On`
- **Window**: `app.Window.NewWithOptions()` for multi-window support
- **System tray**: `app.SystemTray.New()` with menu and click handler
- **Application menu**: Custom menu bar with File/Edit/View/Terminal/Help, all with keyboard shortcuts
- **Dialog**: `SetDirectory` replaces `SetDefaultDirectory`; `PromptForSingleSelection` replaces `PickDirectory`
- **RGBA**: `Red, Green, Blue, Alpha` fields; use `application.NewRGBA()`
- **Middleware**: `AssetOptions.Middleware` for `/local-file/` path interception (replaces `Handler`)

### Frontend (React/TypeScript)
- **Runtime bindings**: `@wailsio/runtime@3.0.0-beta.15` replaces Tauri-style shims
- **IPC bridge**: `core.ts` uses `$Call.ByID(numericHash, args)` from generated bindings
- **Event unwrapping**: Wails v3 delivers `WailsEvent {name, data}` — wrappers auto-unwrap to raw data
- **Settings window**: Settings open in a new OS window (`settings.html?tab=...`) instead of in-app dialog
- **Menu events**: Frontend listens to `menu:*` events for all menu actions
- **Build system**: `Taskfile.yml` + `build/` directory for Wails v3 build pipeline

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

### Cleanup
- Removed `CollapsedAiBar.tsx` (unused)
- Removed `.task/` directory from git (build cache)
- Updated `.gitignore` for `.task/`
- Updated `docs/architecture.md` for Wails v3
- Updated About section: repo link → `HycJack/terax-clone`, bundle ID → `com.hycjack.terax`
- Settings window macOS titlebar transparent with `HideToolbarSeparator`

## Testing
- `wails3 build` succeeds
- App launches with correct menu bar and keyboard shortcuts
- Settings window opens in new OS window with transparent titlebar
- System tray works (minimize to tray, show/hide)

## Migration Notes
See README.md → "Wails v3 Migration (This Branch)" section for full API differences.
