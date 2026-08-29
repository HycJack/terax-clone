// Package winctrl registers Wails event handlers for the custom window
// chrome buttons (close / minimise). The JS shim emits "wails:close" and
// "wails:minimise" when the user clicks the custom titlebar buttons;
// this package translates those into native Wails runtime calls.
package winctrl

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Register wires the custom-chrome event hooks. Call once during startup.
func Register(_ context.Context) {
	app := application.Get()
	if app == nil {
		return
	}
	app.Event.On("wails:close", func(_ *application.CustomEvent) {
		app.Quit()
	})
	app.Event.On("wails:minimise", func(_ *application.CustomEvent) {
		if w := app.Window.Current(); w != nil {
			w.Minimise()
		}
	})
}
