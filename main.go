package main

import (
	"embed"
	"net/http"
	"runtime"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

//go:embed build/appicon.png
var trayIcon []byte

func main() {
	app := NewApp()

	// Use Middleware so wails' built-in asset server handles index.html resolution,
	// asset caching, MIME types etc. We only intercept /local-file/ requests.
	localFileMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/local-file/") {
				localFileHandler().ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	wailsApp := application.New(application.Options{
		Name: "terax",
		Assets: application.AssetOptions{
			Handler:   application.AssetFileServerFS(assets),
			Middleware: localFileMiddleware,
		},
		Services: []application.Service{
			application.NewService(app),
		},
		Mac: application.MacOptions{},
		Windows: application.WindowsOptions{
			DisableQuitOnLastWindowClosed: true,
		},
		OnShutdown: func() {
			app.shutdown()
		},
	})

	// Store reference for tray/menu/window operations
	app.wailsApp = wailsApp

	// ── Application Menu ──────────────────────────────────────────────
	appMenu := buildAppMenu(wailsApp)
	wailsApp.Menu.Set(appMenu)

	// ── Main Window ───────────────────────────────────────────────────
	mainWindow := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:    "main",
		Title:   "terax",
		Width:   1024,
		Height:  768,
		Frameless: runtime.GOOS == "windows",
		Mac: application.MacWindow{
			TitleBar: application.MacTitleBar{
				AppearsTransparent:   true,
				FullSizeContent:      true,
				HideTitle:            true,
				HideToolbarSeparator: true,
			},
		},
		BackgroundColour: application.NewRGBA(27, 38, 54, 1),
	})
	app.mainWindow = mainWindow

	// ── System Tray ───────────────────────────────────────────────────
	trayMenu := buildTrayMenu(wailsApp, mainWindow)
	tray := wailsApp.SystemTray.New()
	tray.SetIcon(trayIcon)
	tray.SetTooltip("terax")
	tray.SetMenu(trayMenu)
	tray.AttachWindow(mainWindow)
	tray.WindowOffset(10)
	tray.OnClick(func() {
		mainWindow.Show().Focus()
	})

	if err := wailsApp.Run(); err != nil {
		println("Error:", err.Error())
	}
}

// buildAppMenu creates the macOS/Windows/Linux application menu bar.
func buildAppMenu(wailsApp *application.App) *application.Menu {
	menu := application.NewMenu()

	// macOS App menu — use custom item to avoid system-injected Edit submenus
	if runtime.GOOS == "darwin" {
		appMenu := menu.AddSubmenu("terax")
		appMenu.Add("About terax").OnClick(func(ctx *application.Context) {
			wailsApp.Event.Emit("menu:about")
		})
		appMenu.AddSeparator()
		appMenu.Add("Settings…").SetAccelerator("CmdOrCtrl+,").OnClick(func(ctx *application.Context) {
			wailsApp.Event.Emit("menu:settings")
		})
		appMenu.AddSeparator()
		appMenu.AddRole(application.Hide)
		appMenu.AddRole(application.HideOthers)
		appMenu.AddRole(application.UnHide)
		appMenu.AddSeparator()
		appMenu.AddRole(application.Quit)
	}

	// ── File ──────────────────────────────────────────────────────
	fileMenu := menu.AddSubmenu("File")
	fileMenu.Add("New Window").SetAccelerator("CmdOrCtrl+n").OnClick(func(ctx *application.Context) {
		wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:  "terax — new window",
			Width:  1024,
			Height: 768,
			BackgroundColour: application.NewRGBA(27, 38, 54, 1),
		})
	})
	fileMenu.Add("Open Folder…").SetAccelerator("CmdOrCtrl+o").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:open-folder")
	})
	fileMenu.AddSeparator()
	fileMenu.AddRole(application.CloseWindow)

	// ── Edit ──────────────────────────────────────────────────────
	editMenu := menu.AddSubmenu("Edit")
	editMenu.AddRole(application.Undo)
	editMenu.AddRole(application.Redo)
	editMenu.AddSeparator()
	editMenu.AddRole(application.Cut)
	editMenu.AddRole(application.Copy)
	editMenu.AddRole(application.Paste)
	editMenu.AddRole(application.SelectAll)
	editMenu.AddSeparator()
	editMenu.Add("Find…").SetAccelerator("CmdOrCtrl+f").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:find")
	})

	// ── View ──────────────────────────────────────────────────────
	viewMenu := menu.AddSubmenu("View")
	viewMenu.AddRole(application.Reload)
	viewMenu.AddRole(application.ForceReload)
	viewMenu.AddRole(application.OpenDevTools)
	viewMenu.AddSeparator()
	viewMenu.Add("Toggle Sidebar").SetAccelerator("CmdOrCtrl+b").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:toggle-sidebar")
	})
	viewMenu.Add("Toggle Zen Mode").SetAccelerator("CmdOrCtrl+Shift+f").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:zen-mode")
	})
	viewMenu.AddSeparator()
	viewMenu.AddRole(application.ResetZoom)
	viewMenu.AddRole(application.ZoomIn)
	viewMenu.AddRole(application.ZoomOut)
	viewMenu.AddSeparator()
	viewMenu.AddRole(application.ToggleFullscreen)

	// ── Terminal ──────────────────────────────────────────────────
	termMenu := menu.AddSubmenu("Terminal")
	termMenu.Add("New Terminal").SetAccelerator("CmdOrCtrl+`").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:new-terminal")
	})
	termMenu.Add("Split Terminal").SetAccelerator("CmdOrCtrl+Shift+`").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:split-terminal")
	})
	termMenu.AddSeparator()
	termMenu.Add("Kill Terminal").SetAccelerator("CmdOrCtrl+Shift+w").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:kill-terminal")
	})

	// ── Window ────────────────────────────────────────────────────
	if runtime.GOOS == "darwin" {
		menu.AddRole(application.WindowMenu)
	}

	// ── Help ──────────────────────────────────────────────────────
	helpMenu := menu.AddSubmenu("Help")
	helpMenu.Add("terax Documentation").SetAccelerator("CmdOrCtrl+Shift+?").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:open-docs")
	})
	helpMenu.AddSeparator()
	helpMenu.Add("About terax").OnClick(func(ctx *application.Context) {
		wailsApp.Event.Emit("menu:about")
	})

	return menu
}

// buildTrayMenu creates the right-click context menu for the system tray.
func buildTrayMenu(wailsApp *application.App, mainWindow application.Window) *application.Menu {
	menu := application.NewMenu()

	menu.Add("Show terax").OnClick(func(ctx *application.Context) {
		mainWindow.Show().Focus()
	})
	menu.Add("Hide terax").OnClick(func(ctx *application.Context) {
		mainWindow.Hide()
	})
	menu.AddSeparator()
	menu.Add("New Window").SetAccelerator("CmdOrCtrl+n").OnClick(func(ctx *application.Context) {
		wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
			Title:  "terax — new window",
			Width:  1024,
			Height: 768,
		})
	})
	menu.AddSeparator()
	menu.Add("Quit terax").SetAccelerator("CmdOrCtrl+q").OnClick(func(ctx *application.Context) {
		wailsApp.Quit()
	})

	return menu
}
