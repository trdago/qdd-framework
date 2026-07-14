package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/dashboard"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Start the QDD dashboard watcher
	go dashboard.RunWatcher()

	// Listen to the broker and emit events to the frontend via Wails Runtime
	go func() {
		ch := dashboard.Broker.AddClient()
		defer dashboard.Broker.RemoveClient(ch)
		for {
			select {
			case <-a.ctx.Done():
				return
			case data := <-ch:
				// Emit the updated state to the frontend
				var state dashboard.QDDState
				if err := json.Unmarshal(data, &state); err == nil {
					runtime.EventsEmit(a.ctx, "state_update", state)
				}
			}
		}
	}()
}

// GetState returns the current QDD state
func (a *App) GetState() dashboard.QDDState {
	return dashboard.BuildState()
}

// GetPolicies returns the current audit policies
func (a *App) GetPolicies() audit.QDDPolicies {
	cwd, _ := os.Getwd()
	return audit.LoadPolicies(cwd)
}

// SavePolicies saves the given policies
func (a *App) SavePolicies(p audit.QDDPolicies) map[string]string {
	cwd, _ := os.Getwd()
	audit.SavePolicies(cwd, p)

	// Broadcast new state
	res := dashboard.BuildState()
	data, _ := json.Marshal(res)
	dashboard.Broker.Broadcast(data)

	return map[string]string{"status": "ok"}
}

// Intent mock endpoint
func (a *App) Intent(input string) map[string]interface{} {
	return map[string]interface{}{
		"status":         "DEPRECATED",
		"message":        "El motor cognitivo interno basado en API ha sido deprecado. QDD ahora actúa exclusivamente como un Harness MCP para inteligencias artificiales externas (Antigravity, Claude, Cursor). Por favor envía tu intención directamente a tu asistente de IA.",
		"input_received": input,
	}
}
