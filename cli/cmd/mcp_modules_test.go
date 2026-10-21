package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func chdirToTempRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "crm"), 0755); err != nil {
		t.Fatal(err)
	}
	orig, _ := os.Getwd()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
	return root
}

func toolRequest(args map[string]interface{}) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Arguments = args
	return req
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestModuleSaveTool_RejectsIncompleteRecord(t *testing.T) {
	chdirToTempRepo(t)
	res, err := handleModuleSaveTool(context.Background(), toolRequest(map[string]interface{}{
		"ficha_yaml": "id: crm\nnombre: CRM\nrutas: [internal/crm]\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(resultText(t, res), "criterio_de_exito") {
		t.Fatalf("se esperaba rechazo con el detalle de lo que falta: %q", resultText(t, res))
	}
}

func TestModuleSaveAndProjectContextTools(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no disponible")
	}
	root := chdirToTempRepo(t)
	ficha := `id: crm
nombre: CRM de licitaciones
estado: aprobado
rutas: [internal/crm]
objetivo_operativo:
  problema: Gestionar el embudo de licitaciones del cliente
  criterio_de_exito: Ninguna licitación vencida aparece como recomendada
objetivo_tecnologico:
  arquitectura: Servicio Go con PostgreSQL
  invariantes: ["deadline > NOW() para toda recomendación"]
pruebas:
  comandos: ["go test ./internal/crm/..."]
`
	res, err := handleModuleSaveTool(context.Background(), toolRequest(map[string]interface{}{"ficha_yaml": ficha}))
	if err != nil || res.IsError {
		t.Fatalf("ficha válida rechazada: %v %q", err, resultText(t, res))
	}
	saved, err := os.ReadFile(filepath.Join(root, ".qdd", "project", "modules", "crm.yaml"))
	if err != nil || !strings.Contains(string(saved), "estado: borrador") {
		t.Fatalf("la ficha debe quedar en borrador aunque la IA pida 'aprobado': %s %v", saved, err)
	}

	res, err = handleProjectContextTool(context.Background(), toolRequest(map[string]interface{}{"paths": "internal/crm/service.go"}))
	if err != nil {
		t.Fatal(err)
	}
	text := resultText(t, res)
	for _, want := range []string{"Forma de trabajo", "CRM de licitaciones", "deadline > NOW()", "Ficha en borrador"} {
		if !strings.Contains(text, want) {
			t.Errorf("qdd_project_context no incluye %q:\n%s", want, text)
		}
	}
}
