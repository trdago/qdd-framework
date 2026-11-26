package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Rule: Prohibido usar 'else'
// Servidor MCP del proyecto (FND-308): init y doctor --fix registran qdd en Claude Code (.mcp.json) y en OpenCode
// (opencode.json), los dos con alcance de proyecto. Solo agregan la entrada qdd: lo demás del archivo y una entrada qdd
// propia quedan. El comando es «qdd» (del PATH), porque el archivo se commitea y una ruta de la máquina se rompe en otra.

const (
	MCPConectado = "conectado"
	MCPFalta     = "falta"
	MCPIlegible  = "ilegible"
	// MCPAMano: el proyecto usa opencode.jsonc, que tiene comentarios; qdd no lo reescribe.
	MCPAMano = "a mano"

	archivoMCPClaude   = ".mcp.json"
	archivoMCPOpenCode = "opencode.json"
	esquemaOpenCode    = "https://opencode.ai/config.json"
)

// EstadoArchivoMCP dice si un archivo de configuración de una IA trae el servidor qdd.
type EstadoArchivoMCP struct {
	IA      string
	Archivo string
	Estado  string
}

// destinoMCP es un archivo donde va el servidor: bajo qué clave y con qué entrada.
type destinoMCP struct {
	ia, archivo, clave string
	entrada            map[string]any
	nuevo              map[string]any
}

func destinosMCP() []destinoMCP {
	return []destinoMCP{
		{ia: "Claude Code", archivo: archivoMCPClaude, clave: "mcpServers",
			entrada: map[string]any{"command": "qdd", "args": []string{"mcp-server"}},
			nuevo:   map[string]any{}},
		{ia: "OpenCode", archivo: archivoMCPOpenCode, clave: "mcp",
			entrada: map[string]any{"type": "local", "command": []string{"qdd", "mcp-server"}, "enabled": true},
			nuevo:   map[string]any{"$schema": esquemaOpenCode}},
	}
}

// EstadoMCP revisa los dos archivos. Una carpeta de proyectos no lleva el servidor: no se revisa.
func EstadoMCP(projectPath string) []EstadoArchivoMCP {
	if Contenedor(projectPath) {
		return nil
	}
	var out []EstadoArchivoMCP
	for _, d := range destinosMCP() {
		out = append(out, EstadoArchivoMCP{IA: d.ia, Archivo: d.archivo, Estado: estadoDestino(projectPath, d)})
	}
	return out
}

func estadoDestino(projectPath string, d destinoMCP) string {
	if d.archivo == archivoMCPOpenCode && usaJSONC(projectPath) {
		return MCPAMano
	}
	cfg, err := leerConfigMCP(filepath.Join(projectPath, d.archivo))
	if err != nil {
		return MCPIlegible
	}
	if cfg == nil || !tieneQDD(cfg, d.clave) {
		return MCPFalta
	}
	return MCPConectado
}

// ConectarMCP agrega la entrada qdd donde falta y devuelve los archivos que cambió. Un archivo ilegible no se pisa.
func ConectarMCP(projectPath string) ([]string, error) {
	if Contenedor(projectPath) {
		return nil, nil
	}
	var cambios []string
	var errs []error
	for _, d := range destinosMCP() {
		cambio, err := conectarDestino(projectPath, d)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if cambio {
			cambios = append(cambios, d.archivo)
		}
	}
	return cambios, errors.Join(errs...)
}

func conectarDestino(projectPath string, d destinoMCP) (bool, error) {
	if d.archivo == archivoMCPOpenCode && usaJSONC(projectPath) {
		return false, nil
	}
	ruta := filepath.Join(projectPath, d.archivo)
	cfg, err := leerConfigMCP(ruta)
	if err != nil {
		return false, fmt.Errorf("%s no es JSON válido: no lo toco; arréglalo y corre qdd doctor --fix", d.archivo)
	}
	if cfg == nil {
		cfg = d.nuevo
	}
	if tieneQDD(cfg, d.clave) {
		return false, nil
	}
	servidores, _ := cfg[d.clave].(map[string]any)
	if servidores == nil {
		servidores = map[string]any{}
	}
	servidores["qdd"] = d.entrada
	cfg[d.clave] = servidores
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}
	return true, os.WriteFile(ruta, append(b, '\n'), 0o644)
}

// leerConfigMCP devuelve nil sin error si el archivo no existe.
func leerConfigMCP(ruta string) (map[string]any, error) {
	b, err := os.ReadFile(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cfg map[string]any
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return cfg, nil
}

func tieneQDD(cfg map[string]any, clave string) bool {
	servidores, _ := cfg[clave].(map[string]any)
	_, ok := servidores["qdd"]
	return ok
}

func usaJSONC(projectPath string) bool {
	_, err := os.Stat(filepath.Join(projectPath, "opencode.jsonc"))
	return err == nil
}
