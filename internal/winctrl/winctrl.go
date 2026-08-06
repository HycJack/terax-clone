// Package winctrl registers Wails event handlers for the custom window
// chrome buttons (close / minimise). The JS shim emits "wails:close" and
// "wails:minimise" when the user clicks the custom titlebar buttons;
// this package translates those into native Wails runtime calls.
package winctrl

import (
	"context"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// Register wires the custom-chrome event hooks. Call once during startup.
func Register(ctx context.Context) {
	wailsruntime.EventsOn(ctx, "wails:close", func(_ ...interface{}) {
		wailsruntime.Quit(ctx)
	})
	wailsruntime.EventsOn(ctx, "wails:minimise", func(_ ...interface{}) {
		wailsruntime.WindowMinimise(ctx)
	})
}
