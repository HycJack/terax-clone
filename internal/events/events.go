// Package events owns the event-bridge handlers used by the settings page.
//
// The settings page navigates via window.location.assign("/settings.html")
// which destroys window['go'], so the frontend falls back to EventsEmit/EventsOn.
// Each handler is a self-contained function that reads args, calls the backend,
// and emits a result event.
package events

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"
	"terax/internal/secrets"
	"terax/internal/store"
)

// emit is a helper to emit a custom event via the v3 event system.
func emit(name string, data interface{}) {
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}

// RegisterAll attaches every event-bridge handler. Call once during startup.
func RegisterAll(_ any) {
	app := application.Get()
	if app == nil {
		return
	}
	registerStoreHandlers(app)
	registerSecretsHandlers(app)
}

// =========================================================================
// Store handlers
// =========================================================================

func registerStoreHandlers(app *application.App) {
	app.Event.On("store:load", func(e *application.CustomEvent) {
		m, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}
		path, _ := m["path"].(string)
		if path == "" {
			return
		}
		data, err := store.Load(store.LoadArgs{Path: path})
		if err != nil {
			emit("store:load:result", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		emit("store:load:result", map[string]interface{}{
			"success": true,
			"data":    data,
		})
	})

	app.Event.On("store:save", func(e *application.CustomEvent) {
		m, ok := e.Data.(map[string]interface{})
		if !ok {
			return
		}
		path, _ := m["path"].(string)
		rawData, _ := m["data"].(map[string]interface{})
		if path == "" {
			return
		}
		if err := store.Save(store.SaveArgs{Path: path, Data: rawData}); err != nil {
			emit("store:save:result", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		emit("store:save:result", map[string]interface{}{
			"success": true,
		})
	})
}

// =========================================================================
// Secrets handlers
// =========================================================================

func registerSecretsHandlers(app *application.App) {
	app.Event.On("secrets:set", func(e *application.CustomEvent) {
		m, ok := e.Data.(map[string]interface{})
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
			fmt.Printf("secrets:set failed: %v\n", err)
			emit("secrets:set:result", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		emit("secrets:set:result", map[string]interface{}{
			"success": true,
		})
	})

	app.Event.On("secrets:get", func(e *application.CustomEvent) {
		m, ok := e.Data.(map[string]interface{})
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
			emit("secrets:get:result", map[string]interface{}{
				"account": account,
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		emit("secrets:get:result", map[string]interface{}{
			"account": account,
			"success": true,
			"value":   v,
		})
	})

	app.Event.On("secrets:delete", func(e *application.CustomEvent) {
		m, ok := e.Data.(map[string]interface{})
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
			emit("secrets:delete:result", map[string]interface{}{
				"account": account,
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		emit("secrets:delete:result", map[string]interface{}{
			"account": account,
			"success": true,
		})
	})

	app.Event.On("secrets:getAll", func(e *application.CustomEvent) {
		m, ok := e.Data.(map[string]interface{})
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
			emit("secrets:getAll:result", map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		out := make([]interface{}, len(values))
		for i, v := range values {
			out[i] = v
		}
		emit("secrets:getAll:result", map[string]interface{}{
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
