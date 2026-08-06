// Package events owns the event-bridge handlers used by the settings page.
//
// The settings page navigates via window.location.assign("/settings.html")
// which destroys window['go'], so the frontend falls back to EventsEmit/EventsOn.
// Each handler is a self-contained function that reads args, calls the backend,
// and emits a result event.
package events

import (
	"context"
	"fmt"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"terax/internal/secrets"
	"terax/internal/store"
)

// RegisterAll attaches every event-bridge handler. Call once during startup.
func RegisterAll(ctx context.Context) {
	registerStoreHandlers(ctx)
	registerSecretsHandlers(ctx)
}

// =========================================================================
// Store handlers
// =========================================================================

func registerStoreHandlers(ctx context.Context) {
	// store:load { path } → store:load:result
	wailsruntime.EventsOn(ctx, "store:load", func(args ...interface{}) {
		if len(args) < 1 {
			return
		}
		m, ok := args[0].(map[string]interface{})
		if !ok {
			return
		}
		path, _ := m["path"].(string)
		if path == "" {
			return
		}
		data, err := store.Load(store.LoadArgs{Path: path})
		if err != nil {
			wailsruntime.EventsEmit(ctx, "store:load:result", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		wailsruntime.EventsEmit(ctx, "store:load:result", map[string]interface{}{
			"success": true,
			"data":    data,
		})
	})

	// store:save { path, data } → store:save:result
	wailsruntime.EventsOn(ctx, "store:save", func(args ...interface{}) {
		if len(args) < 1 {
			return
		}
		m, ok := args[0].(map[string]interface{})
		if !ok {
			return
		}
		path, _ := m["path"].(string)
		rawData, _ := m["data"].(map[string]interface{})
		if path == "" {
			return
		}
		if err := store.Save(store.SaveArgs{Path: path, Data: rawData}); err != nil {
			wailsruntime.EventsEmit(ctx, "store:save:result", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		wailsruntime.EventsEmit(ctx, "store:save:result", map[string]interface{}{
			"success": true,
		})
	})
}

// =========================================================================
// Secrets handlers
// =========================================================================

func registerSecretsHandlers(ctx context.Context) {
	// secrets:set { service, account, password } → secrets:set:result
	wailsruntime.EventsOn(ctx, "secrets:set", func(args ...interface{}) {
		if len(args) < 1 {
			return
		}
		m, ok := args[0].(map[string]interface{})
		if !ok {
			return
		}
		service, _ := m["service"].(string)
		account, _ := m["account"].(string)
		password, _ := m["password"].(string)
		if account == "" || password == "" {
			return
		}
		applyService(service)
		if err := secrets.Set(account, password); err != nil {
			wailsruntime.LogError(ctx, fmt.Sprintf("secrets:set failed: %v", err))
			wailsruntime.EventsEmit(ctx, "secrets:set:result", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		wailsruntime.EventsEmit(ctx, "secrets:set:result", map[string]interface{}{
			"success": true,
		})
	})

	// secrets:get { service, account } → secrets:get:result
	wailsruntime.EventsOn(ctx, "secrets:get", func(args ...interface{}) {
		if len(args) < 1 {
			return
		}
		m, ok := args[0].(map[string]interface{})
		if !ok {
			return
		}
		service, _ := m["service"].(string)
		account, _ := m["account"].(string)
		if account == "" {
			return
		}
		applyService(service)
		v, err := secrets.Get(account)
		if err != nil {
			wailsruntime.EventsEmit(ctx, "secrets:get:result", map[string]interface{}{
				"account": account,
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		wailsruntime.EventsEmit(ctx, "secrets:get:result", map[string]interface{}{
			"account": account,
			"success": true,
			"value":   v,
		})
	})

	// secrets:delete { service, account } → secrets:delete:result
	wailsruntime.EventsOn(ctx, "secrets:delete", func(args ...interface{}) {
		if len(args) < 1 {
			return
		}
		m, ok := args[0].(map[string]interface{})
		if !ok {
			return
		}
		service, _ := m["service"].(string)
		account, _ := m["account"].(string)
		if account == "" {
			return
		}
		applyService(service)
		if err := secrets.Delete(account); err != nil {
			wailsruntime.EventsEmit(ctx, "secrets:delete:result", map[string]interface{}{
				"account": account,
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		wailsruntime.EventsEmit(ctx, "secrets:delete:result", map[string]interface{}{
			"account": account,
			"success": true,
		})
	})

	// secrets:getAll { service, accounts } → secrets:getAll:result
	wailsruntime.EventsOn(ctx, "secrets:getAll", func(args ...interface{}) {
		if len(args) < 1 {
			return
		}
		m, ok := args[0].(map[string]interface{})
		if !ok {
			return
		}
		service, _ := m["service"].(string)
		rawAccounts, _ := m["accounts"].([]interface{})
		accounts := make([]string, 0, len(rawAccounts))
		for _, a := range rawAccounts {
			if s, ok := a.(string); ok {
				accounts = append(accounts, s)
			}
		}
		if len(accounts) == 0 {
			return
		}
		applyService(service)
		values, err := secrets.GetAll(accounts)
		if err != nil {
			wailsruntime.EventsEmit(ctx, "secrets:getAll:result", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		out := make([]interface{}, len(values))
		for i, v := range values {
			out[i] = v
		}
		wailsruntime.EventsEmit(ctx, "secrets:getAll:result", map[string]interface{}{
			"success":  true,
			"values":   out,
			"accounts": accounts,
		})
	})
}

// applyService sets the keyring service name if provided.
func applyService(service string) {
	if service != "" {
		secrets.SetService(service)
	}
}
