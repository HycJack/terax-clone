// Package agent wires up agent hooks. The Rust version used notify markers
// piped into a target PTY; here we expose an in-memory state plus a window
// event the frontend listens to (`terax:agent-signal`).
package agent

import (
	"context"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
	"terax/internal/types"
)

var (
	mu       sync.Mutex
	enabled  = map[string]bool{}
	readyEvt = "terax:hooks-ready"
)

// emit is a helper to emit a custom event via the v3 event system.
func emit(name string, data interface{}) {
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}

// EnableHooks flips the flag for a given agent and announces readiness.
func EnableHooks(_ context.Context, agent string) error {
	mu.Lock()
	enabled[agent] = true
	mu.Unlock()
	emit(readyEvt, types.AgentHooksReady{Agent: agent})
	return nil
}

// HooksStatus returns whether hooks are wired up for a given agent.
func HooksStatus(agent string) types.AgentHooksStatus {
	mu.Lock()
	defer mu.Unlock()
	return types.AgentHooksStatus{Ready: enabled[agent]}
}

// EmitSignal pushes a signal to the frontend.
func EmitSignal(_ context.Context, sig types.AgentSignal) {
	emit("terax:agent-signal", sig)
}
