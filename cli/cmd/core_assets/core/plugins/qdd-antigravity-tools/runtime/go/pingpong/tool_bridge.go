// Package pingpong implements the QDD Tool Bridge in Go.
// Strictly follows Zero-Else and Early Return.
package pingpong

import (
	"fmt"
	"sync"
)

type ToolHandler func(args map[string]interface{}) (interface{}, error)

type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Handler     ToolHandler            `json:"-"`
}

type ToolBridge struct {
	mu    sync.RWMutex
	tools map[string]ToolDefinition
}

func NewToolBridge() *ToolBridge {
	return &ToolBridge{
		tools: make(map[string]ToolDefinition),
	}
}

func (tb *ToolBridge) RegisterTool(name, description string, params map[string]interface{}, handler ToolHandler) error {
	if name == "" {
		return fmt.Errorf("tool name no puede estar vacío")
	}
	if handler == nil {
		return fmt.Errorf("handler no puede ser nil")
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.tools[name] = ToolDefinition{
		Name:        name,
		Description: description,
		Parameters:  params,
		Handler:     handler,
	}
	return nil
}

func (tb *ToolBridge) ListTools() []ToolDefinition {
	tb.mu.RLock()
	defer tb.mu.RUnlock()

	list := make([]ToolDefinition, 0, len(tb.tools))
	for _, t := range tb.tools {
		list = append(list, ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		})
	}
	return list
}

func (tb *ToolBridge) ExecuteTool(name string, args map[string]interface{}) (interface{}, error) {
	tb.mu.RLock()
	tool, exists := tb.tools[name]
	tb.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("herramienta '%s' no registrada", name)
	}

	return tool.Handler(args)
}
