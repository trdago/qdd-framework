package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// FND-308: qdd init no registraba el servidor MCP en Claude Code ni en OpenCode; la IA tenía las reglas pero no las
// tools de QDD. init (y doctor --fix) agregan solo la entrada qdd a .mcp.json y a opencode.json, sin tocar lo demás. El
// comando es «qdd», no una ruta de la máquina: el archivo se commitea.
// contrato: integracion#a827c9a2

func leerJSON(t *testing.T, ruta string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("%s ilegible: %v", ruta, err)
	}
	return v
}

func escribir(t *testing.T, ruta, contenido string) {
	t.Helper()
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMCPProyecto_ProyectoNuevoQuedaConectado_FND308(t *testing.T) {
	dir := t.TempDir()
	if _, err := ConectarMCP(dir); err != nil {
		t.Fatal(err)
	}
	claude := leerJSON(t, filepath.Join(dir, ".mcp.json"))
	quiere := map[string]any{"command": "qdd", "args": []any{"mcp-server"}}
	if got := claude["mcpServers"].(map[string]any)["qdd"]; !reflect.DeepEqual(got, quiere) {
		t.Fatalf(".mcp.json: %v", got)
	}
	opencode := leerJSON(t, filepath.Join(dir, "opencode.json"))
	quiereOC := map[string]any{"type": "local", "command": []any{"qdd", "mcp-server"}, "enabled": true}
	if got := opencode["mcp"].(map[string]any)["qdd"]; !reflect.DeepEqual(got, quiereOC) {
		t.Fatalf("opencode.json: %v", got)
	}
	for _, e := range EstadoMCP(dir) {
		if e.Estado != MCPConectado {
			t.Errorf("después de conectar: %+v", e)
		}
	}
}

func TestMCPProyecto_NoTocaLoQueYaTienen_FND308(t *testing.T) {
	dir := t.TempDir()
	escribir(t, filepath.Join(dir, ".mcp.json"), `{"mcpServers":{"otro":{"command":"x","args":["y"]},"qdd":{"command":"/opt/qdd","args":["mcp-server"]}}}`)
	escribir(t, filepath.Join(dir, "opencode.json"), `{"$schema":"https://opencode.ai/config.json","model":"m","mcp":{"otro":{"type":"remote","url":"https://e"}}}`)
	if _, err := ConectarMCP(dir); err != nil {
		t.Fatal(err)
	}
	claude := leerJSON(t, filepath.Join(dir, ".mcp.json"))["mcpServers"].(map[string]any)
	if claude["qdd"].(map[string]any)["command"] != "/opt/qdd" || claude["otro"] == nil {
		t.Fatalf("una entrada qdd propia y los demás servidores quedan: %v", claude)
	}
	opencode := leerJSON(t, filepath.Join(dir, "opencode.json"))
	mcp := opencode["mcp"].(map[string]any)
	if opencode["model"] != "m" || mcp["otro"] == nil || mcp["qdd"] == nil {
		t.Fatalf("agrega qdd y conserva lo demás: %v", opencode)
	}
}

func TestMCPProyecto_SegundaVezNoCambiaNada_FND308(t *testing.T) {
	dir := t.TempDir()
	ConectarMCP(dir)
	antes, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	cambios, err := ConectarMCP(dir)
	despues, _ := os.ReadFile(filepath.Join(dir, ".mcp.json"))
	if err != nil || len(cambios) != 0 || string(antes) != string(despues) {
		t.Fatalf("ya conectado no escribe: %v %q", err, cambios)
	}
}

// Un archivo que no se puede leer como JSON no se pisa: doctor lo dice y la persona lo arregla.
func TestMCPProyecto_UnArchivoIlegibleNoSePisa_FND308(t *testing.T) {
	dir := t.TempDir()
	escribir(t, filepath.Join(dir, ".mcp.json"), "{ esto no es json")
	if _, err := ConectarMCP(dir); err == nil {
		t.Fatal("dice que no pudo")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, ".mcp.json")); string(b) != "{ esto no es json" {
		t.Fatalf("no lo pisa: %q", b)
	}
	if e := EstadoMCP(dir)[0]; e.Estado != MCPIlegible {
		t.Fatalf("doctor lo ve ilegible: %+v", e)
	}
}

// OpenCode también lee opencode.jsonc: si el proyecto usa ese, no se crea un opencode.json al lado.
func TestMCPProyecto_ConOpencodeJSONCNoCreaOtro_FND308(t *testing.T) {
	dir := t.TempDir()
	escribir(t, filepath.Join(dir, "opencode.jsonc"), "{\n  // mío\n}\n")
	ConectarMCP(dir)
	if _, err := os.Stat(filepath.Join(dir, "opencode.json")); !os.IsNotExist(err) {
		t.Fatal("no crea opencode.json junto a opencode.jsonc")
	}
	if e := EstadoMCP(dir)[1]; e.Estado != MCPAMano {
		t.Fatalf("doctor dice que va a mano en opencode.jsonc: %+v", e)
	}
}

// Una carpeta de proyectos no lleva el servidor: cada proyecto de adentro tiene el suyo.
func TestMCPProyecto_UnaCarpetaDeProyectosNoLoLleva_FND308(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "a", ".qdd"), 0o755)
	ConectarMCP(dir)
	if _, err := os.Stat(filepath.Join(dir, ".mcp.json")); !os.IsNotExist(err) {
		t.Fatal("una carpeta de proyectos no lleva .mcp.json")
	}
	if len(EstadoMCP(dir)) != 0 {
		t.Fatal("ni doctor lo pide")
	}
}

// init y doctor --fix sincronizan con SyncAll: ahí queda conectado.
func TestMCPProyecto_SyncAllLoConecta_FND308(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".qdd"), 0o755)
	if err := NewIntegrationManager().SyncAll(dir); err != nil {
		t.Fatal(err)
	}
	for _, e := range EstadoMCP(dir) {
		if e.Estado != MCPConectado {
			t.Errorf("después de SyncAll: %+v", e)
		}
	}
}
