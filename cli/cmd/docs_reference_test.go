package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// FND-072: la documentación quedaba atrás del binario. Comandos reales
// (qdd modules, qdd undo, qdd certify human…) y tools MCP (qdd_project_context,
// qdd_module_save…) no figuraban en la referencia, y la arquitectura citaba
// comandos que no existen (qdd sync, qdd score).

const referenciaComandos = "../../docs/command-reference.md"

func leerDoc(t *testing.T, ruta string) string {
	t.Helper()
	data, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatalf("no se pudo leer %s: %v", ruta, err)
	}
	return string(data)
}

func comandosCLI(c *cobra.Command) []string {
	var rutas []string
	for _, sub := range c.Commands() {
		if sub.Hidden || sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		rutas = append(rutas, sub.CommandPath())
		rutas = append(rutas, comandosCLI(sub)...)
	}
	return rutas
}

func TestDocs_ReferenciaIncluyeTodosLosComandos(t *testing.T) {
	doc := leerDoc(t, referenciaComandos)
	for _, ruta := range comandosCLI(rootCmd) {
		if !strings.Contains(doc, "`"+ruta) {
			t.Errorf("docs/command-reference.md no documenta el comando `%s`", ruta)
		}
	}
}

func TestDocs_ReferenciaYServidorMCPCoinciden(t *testing.T) {
	s := server.NewMCPServer("test-docs", "1.0.0")
	registerMCPTools(s)
	registradas := s.ListTools()

	doc := leerDoc(t, referenciaComandos)
	for nombre := range registradas {
		if !strings.Contains(doc, "`"+nombre) {
			t.Errorf("docs/command-reference.md no documenta la tool MCP `%s`", nombre)
		}
	}

	documentadas := regexp.MustCompile("`(qdd_[a-z_]+)").FindAllStringSubmatch(doc, -1)
	for _, m := range documentadas {
		if _, ok := registradas[m[1]]; !ok {
			t.Errorf("docs/command-reference.md documenta la tool MCP `%s`, pero el servidor no la registra", m[1])
		}
	}
}

var enlaceMarkdown = regexp.MustCompile(`\]\(([^)\s]+)\)`)

func TestDocs_EnlacesRelativosExisten(t *testing.T) {
	docs, err := filepath.Glob("../../docs/*.md")
	if err != nil {
		t.Fatal(err)
	}
	archivos := append(docs, "../../README.md")
	for _, archivo := range archivos {
		for _, m := range enlaceMarkdown.FindAllStringSubmatch(leerDoc(t, archivo), -1) {
			destino := strings.SplitN(m[1], "#", 2)[0]
			if destino == "" || strings.Contains(destino, "://") || strings.HasPrefix(destino, "mailto:") {
				continue
			}
			ruta := filepath.Join(filepath.Dir(archivo), destino)
			if _, err := os.Stat(ruta); err != nil {
				t.Errorf("%s enlaza a %s, que no existe", filepath.Base(archivo), m[1])
			}
		}
	}
}
