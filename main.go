package main

import (
	"embed"
	"runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	mac "github.com/wailsapp/wails/v2/pkg/options/mac"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Create an instance of the app structure
	app := NewApp()

	// Create application with options
	err := wails.Run(&options.App{
		Title:  "terax",
		Width:  1024,
		Height: 768,
		// Windows: frameless drops the native Win32 caption / title bar so the
		// React chrome (WindowControls.tsx) can drive its own drag region and
		// traffic-light-style buttons.
		//
		// macOS: keep the native traffic lights ("overlay title bar") — a
		// frameless window hides them entirely and the custom controls are
		// intentionally disabled on Mac (see lib/platform.ts). TitleBarHidden
		// makes the title bar transparent/full-size while keeping close /
		// minimize / zoom buttons in the top-left.
		Frameless: runtime.GOOS == "windows",
		Mac: &mac.Options{
			TitleBar: mac.TitleBarHidden(),
		},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
}
