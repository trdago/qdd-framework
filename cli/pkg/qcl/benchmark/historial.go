package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Rule: Prohibido usar 'else'
// FND-059: cada fuente tiene su archivo. cognitive_history.json guarda solo certificados del benchmark;
// mcp_eventos.json, los eventos de las herramientas del MCP (los últimos TopeEventosMCP).

// TopeEventosMCP es cuántos eventos del MCP se guardan (los más nuevos).
const TopeEventosMCP = 100

// RutaHistorial es el historial de certificados cognitivos del proyecto.
func RutaHistorial(root string) string {
	return filepath.Join(root, ".qdd", "project", "metrics", "cognitive_history.json")
}

// RutaEventosMCP es el registro de las herramientas del MCP del proyecto.
func RutaEventosMCP(root string) string {
	return filepath.Join(root, ".qdd", "project", "metrics", "mcp_eventos.json")
}

// esCertificado: un certificado trae sus puntajes; un evento del MCP, la herramienta.
func esCertificado(entrada json.RawMessage) bool {
	var campos map[string]json.RawMessage
	if json.Unmarshal(entrada, &campos) != nil {
		return false
	}
	_, ok := campos["scores"]
	return ok
}

// partes lee el historial y separa los certificados de lo demás (eventos del MCP de antes de FND-059).
func partes(root string) ([]json.RawMessage, []json.RawMessage, error) {
	data, err := os.ReadFile(RutaHistorial(root))
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("error leyendo historial cognitivo: %v", err)
	}
	var todas []json.RawMessage
	if err := json.Unmarshal(data, &todas); err != nil {
		return nil, nil, fmt.Errorf("error parseando historial cognitivo: %v", err)
	}
	var certificados, otras []json.RawMessage
	for _, e := range todas {
		if esCertificado(e) {
			certificados = append(certificados, e)
			continue
		}
		otras = append(otras, e)
	}
	return certificados, otras, nil
}

// Mezclado dice si el historial trae algo que no es un certificado (qdd doctor, FND-059).
func Mezclado(root string) (bool, error) {
	_, otras, err := partes(root)
	return len(otras) > 0, err
}

// SepararEventosMCP pasa al registro del MCP lo que no es un certificado, antes de lo que ya tenía, y deja en el
// historial solo los certificados. No pierde nada: el registro no se recorta al separar.
func SepararEventosMCP(root string) error {
	certificados, otras, err := partes(root)
	if err != nil || len(otras) == 0 {
		return err
	}
	eventos, err := LeerEventosMCP(root)
	if err != nil {
		return err
	}
	if err := escribirLista(RutaEventosMCP(root), append(otras, eventos...)); err != nil {
		return err
	}
	return escribirLista(RutaHistorial(root), certificados)
}

// LeerEventosMCP es el registro del MCP; vacío si no existe.
func LeerEventosMCP(root string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(RutaEventosMCP(root))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var eventos []json.RawMessage
	if err := json.Unmarshal(data, &eventos); err != nil {
		return nil, fmt.Errorf("mcp_eventos.json ilegible: %v", err)
	}
	return eventos, nil
}

// AnotarEventoMCP agrega un evento al registro del MCP y deja los últimos TopeEventosMCP.
func AnotarEventoMCP(root string, evento any) error {
	b, err := json.Marshal(evento)
	if err != nil {
		return err
	}
	eventos, err := LeerEventosMCP(root)
	if err != nil {
		return err
	}
	eventos = append(eventos, b)
	if len(eventos) > TopeEventosMCP {
		eventos = eventos[len(eventos)-TopeEventosMCP:]
	}
	return escribirLista(RutaEventosMCP(root), eventos)
}

func escribirLista(ruta string, lista []json.RawMessage) error {
	if lista == nil {
		lista = []json.RawMessage{}
	}
	out, err := json.MarshalIndent(lista, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	return os.WriteFile(ruta, out, 0o644)
}
