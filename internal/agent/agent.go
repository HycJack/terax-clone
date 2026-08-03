// Package agent wires up agent hooks. The Rust version used notify markers
// piped into a target PTY; here we expose an in-memory state plus a window
// event the frontend listens to (`terax:agent-signal`).
package agent

import (
	"context"
	"sync"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"terax/internal/types"
)

var (
	mu       sync.Mutex
	enabled  = map[string]bool{}
	readyEvt = "terax:hooks-ready"
)

// EnableHooks flips the flag for a given agent and announces readiness.
func EnableHooks(ctx context.Context, agent string) error {
	mu.Lock()
	enabled[agent] = true
	mu.Unlock()
	wailsruntime.EventsEmit(ctx, readyEvt, types.AgentHooksReady{Agent: agent})
	return nil
}

// HooksStatus returns whether hooks are wired up for a given agent.
func HooksStatus(agent string) types.AgentHooksStatus {
	mu.Lock()
	defer mu.Unlock()
	return types.AgentHooksStatus{Ready: enabled[agent]}
}

// EmitSignal pushes a signal to the frontend.
func EmitSignal(ctx context.Context, sig types.AgentSignal) {
	wailsruntime.EventsEmit(ctx, "terax:agent-signal", sig)
}