package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/qcl/benchmark"
)

// Rule: Prohibido usar 'else'
// FND-304: Claude Code abre el servidor MCP con la consola y no lo reinicia al actualizar qdd, así que una corrección
// publicada no llegaba a la IA y nadie lo avisaba. El servidor no se reinicia solo: avisa, y la persona lo reconecta.

// binarioCambio dice si el binario de este servidor se reemplazó desde que empezó (nunca, fuera de qdd mcp-server).
var binarioCambio = func() bool { return false }

var avisoDado bool

func vigilarBinario() {
	bin, err := os.Executable()
	if err != nil {
		return
	}
	binarioCambio = consola.VigiaBinario(bin)
}

// eventoMCP es una línea de mcp_eventos.json.
type eventoMCP struct {
	Timestamp string      `json:"timestamp"`
	Tool      string      `json:"tool"`
	Status    string      `json:"status"`
	Details   interface{} `json:"details"`
	PID       int         `json:"pid,omitempty"`
	Version   string      `json:"version,omitempty"`
	Atrasado  bool        `json:"atrasado,omitempty"`
}

// conAvisoDeBinario suma el aviso a la respuesta la primera vez y en qdd_project_context, que se llama antes de cambiar
// código; en cada llamada sería un costo fijo de tokens.
func conAvisoDeBinario(tool string, res *mcp.CallToolResult) *mcp.CallToolResult {
	if res == nil || !binarioCambio() || (avisoDado && tool != "qdd_project_context") {
		return res
	}
	avisoDado = true
	aviso := fmt.Sprintf("⚠️ QDD se actualizó y estas herramientas siguen corriendo %s. Dile a la persona que escriba /mcp y reconecte qdd, o que reabra la consola; mientras tanto usa el comando qdd.", Version)
	res.Content = append(res.Content, mcp.NewTextContent(aviso))
	return res
}

// checkMCPAlDia informa, sin fallar, que una consola abierta en este proyecto usa herramientas de una versión anterior.
func checkMCPAlDia(projectPath string) *CheckItem {
	eventos, err := benchmark.LeerEventosMCP(projectPath)
	if err != nil {
		return nil
	}
	visto := map[int]bool{}
	for _, linea := range slices.Backward(eventos) {
		var e eventoMCP
		if json.Unmarshal(linea, &e) != nil || visto[e.PID] {
			continue
		}
		visto[e.PID] = true
		if e.Atrasado && consola.ProcesoVivo(e.PID) {
			return &CheckItem{Name: nombreMCPAlDia, Success: true, Error: fmt.Sprintf("corre %s (proceso %d): escribe /mcp en ella y reconecta qdd, o reábrela", e.Version, e.PID)}
		}
	}
	return nil
}

const nombreMCPAlDia = "Aviso: una consola abierta con herramientas de una versión anterior"
