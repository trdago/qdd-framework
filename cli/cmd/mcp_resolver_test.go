package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/findings"
)

// Rule: Prohibido usar 'else'

// FND-312: el diagnóstico quedaba con un estado que FND-01 bloquea, y el acta decía «verificado» y «100%» sin mirar nada.
func TestResolveDiagnose_DejaUnFindingAbiertoYLegible_FND312(t *testing.T) {
	dir := enProyecto(t)
	texto, falla := llamarTool(t, registerResolveDiagnoseTool, "qdd_resolve_diagnose", map[string]interface{}{
		"issue_type": "bug", "title": "Carrera al renovar el token", "root_cause": "falta un candado",
		"impact_analysis": "sesiones simultáneas", "affected_files": "a.go; b.go",
	})
	if falla {
		t.Fatal(texto)
	}
	todos, err := findings.Leer(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(todos) != 1 || todos[0].Estado != findings.Abierto || todos[0].Titulo != "Carrera al renovar el token" {
		t.Fatalf("se esperaba un finding abierto con su título: %+v", todos)
	}
}

// contrato: mcp#d8800878
func TestResolveCertify_SoloDeclaraLoQueComprueba_FND312(t *testing.T) {
	dir := enProyecto(t)
	mustWrite(t, filepath.Join(dir, "golden.json"), "{}")
	afuera := filepath.Join(filepath.Dir(dir), "afuera.json")
	mustWrite(t, afuera, "{}")
	base := map[string]interface{}{"resolution_id": "FIX-101", "golden_set_path": "golden.json", "qa_approved": true, "qa_notes": "bordes", "devops_approved": true, "devops_notes": "sin impacto"}

	for nombre, cambio := range map[string]map[string]interface{}{
		"sin golden set":          {"golden_set_path": "no-existe.json"},
		"golden set de afuera":    {"golden_set_path": afuera},
		"id que sale de su lugar": {"resolution_id": "../../fuera"},
		"sin QA":                  {"qa_approved": false},
		"sin DevOps":              {"devops_approved": false},
	} {
		if texto, falla := llamarTool(t, registerResolveCertifyTool, "qdd_resolve_certify", con(base, cambio)); !falla {
			t.Errorf("%s: debía rechazarse y respondió %q", nombre, texto)
		}
	}

	texto, falla := llamarTool(t, registerResolveCertifyTool, "qdd_resolve_certify", base)
	if falla {
		t.Fatal(texto)
	}
	acta, err := os.ReadFile(filepath.Join(dir, ".qdd", "project", "certification", "resolution_FIX-101.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, frase := range []string{"100%", "Verificado", "CERTIFICAD", "certificado para release"} {
		if strings.Contains(string(acta)+texto, frase) {
			t.Errorf("afirma «%s» sin haberlo comprobado", frase)
		}
	}
	if !strings.Contains(string(acta), "no se ejecutó") {
		t.Errorf("el acta no dice qué no se comprobó:\n%s", acta)
	}
}

func con(base, cambio map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range cambio {
		out[k] = v
	}
	return out
}

func enProyecto(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "proyecto")
	mustMkdir(t, filepath.Join(dir, ".qdd", "project"))
	t.Chdir(dir)
	return dir
}

func llamarTool(t *testing.T, registrar func(*server.MCPServer), nombre string, args map[string]interface{}) (string, bool) {
	t.Helper()
	s := server.NewMCPServer("prueba", "1")
	registrar(s)
	req := mcp.CallToolRequest{}
	req.Params.Name, req.Params.Arguments = nombre, args
	res, err := s.ListTools()[nombre].Handler(context.Background(), req)
	if err != nil {
		return err.Error(), true
	}
	texto, _ := res.Content[0].(mcp.TextContent)
	return texto.Text, res.IsError
}
