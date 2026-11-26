package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'

// Una IA guarda fichas por qdd_module_save y siempre quedan en borrador, aunque la ficha ya estuviera aprobada o la IA
// pida «aprobado»; ninguna tool MCP aprueba: aprobar es de la persona (qdd modules approve, el panel o el teléfono) o de
// quien ella delegó (el modo goal que ella activa).
// contrato: fichas#aefe38af
func TestFichas_LaIAGuardaEnBorradorYNoApruebaPorMCP(t *testing.T) {
	paquete, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := chdirToTempRepo(t)
	ficha := `id: crm
nombre: CRM de licitaciones
estado: aprobado
rutas: [internal/crm]
objetivo_operativo:
  problema: Gestionar el embudo de licitaciones
  criterio_de_exito: Ninguna licitación vencida aparece como recomendada
objetivo_tecnologico:
  arquitectura: Servicio Go con PostgreSQL
pruebas:
  comandos: ["go test ./internal/crm/..."]
`
	if _, err = handleModuleSaveTool(context.Background(), toolRequest(map[string]interface{}{"ficha_yaml": ficha})); err != nil {
		t.Fatal(err)
	}
	if _, problemas := modules.Approve(root, "crm"); len(problemas) > 0 {
		t.Fatal(problemas)
	}
	if m, _ := modules.Find(root, "crm"); m.Estado != modules.EstadoAprobado {
		t.Fatalf("la persona la aprobó: %+v", m.Estado)
	}
	cambiada := strings.Replace(ficha, "Servicio Go con PostgreSQL", "Servicio Go con SQLite", 1)
	res, err := handleModuleSaveTool(context.Background(), toolRequest(map[string]interface{}{"ficha_yaml": cambiada}))
	if err != nil || res.IsError {
		t.Fatalf("la IA puede proponer el cambio: %v %q", err, resultText(t, res))
	}
	m, _ := modules.Find(root, "crm")
	if m.Estado != modules.EstadoBorrador || m.UltimaRevision != "" || m.RevisadoEnCommit != "" {
		t.Fatalf("lo que guarda la IA vuelve a borrador, sin la revisión de la persona: %+v", m)
	}

	s := server.NewMCPServer("fichas", "1")
	registerMCPTools(s)
	for nombre := range s.ListTools() {
		if strings.Contains(nombre, "approve") || strings.Contains(nombre, "aprob") {
			t.Errorf("la tool %s aprobaría fichas: aprobar es de la persona", nombre)
		}
	}
	archivos, _ := filepath.Glob(filepath.Join(paquete, "mcp*.go"))
	if len(archivos) == 0 {
		t.Fatal("no se encontraron las tools MCP: la prueba ya no mira donde se registran")
	}
	for _, archivo := range archivos {
		if strings.HasSuffix(archivo, "_test.go") {
			continue
		}
		fuente, err := os.ReadFile(archivo)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(fuente), "modules.Approve(") {
			t.Errorf("%s aprueba fichas desde una tool MCP", filepath.Base(archivo))
		}
	}
}
