package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// contrato: mcp#f76f665c
// Una tool registrada con AddTool directo no tendría la guarda Zero-Panic: AddTool solo puede aparecer dentro de
// RegisterToolWithTelemetry.
func TestMCP_TodaToolPasaPorRegisterToolWithTelemetry(t *testing.T) {
	agrega := regexp.MustCompile(`\.AddTools?\(`)
	archivos, _ := filepath.Glob("*.go")
	encontradas := 0
	for _, archivo := range archivos {
		if strings.HasSuffix(archivo, "_test.go") {
			continue
		}
		fuente, err := os.ReadFile(archivo)
		if err != nil {
			t.Fatal(err)
		}
		for _, pos := range agrega.FindAllIndex(fuente, -1) {
			encontradas++
			if funcionEn(string(fuente), pos[0]) != "RegisterToolWithTelemetry" {
				t.Errorf("%s registra una tool fuera de RegisterToolWithTelemetry (en %s): no tendría la guarda Zero-Panic", archivo, funcionEn(string(fuente), pos[0]))
			}
		}
	}
	if encontradas == 0 {
		t.Fatal("no se encontró ningún AddTool: la prueba ya no mira donde se registran las tools")
	}
}

// funcionEn es el nombre de la última func declarada antes de pos.
func funcionEn(fuente string, pos int) string {
	inicio := strings.LastIndex(fuente[:pos], "\nfunc ")
	if inicio < 0 {
		return ""
	}
	firma := strings.TrimPrefix(fuente[inicio:], "\nfunc ")
	if strings.HasPrefix(firma, "(") {
		firma = firma[strings.Index(firma, ")")+1:]
	}
	nombre, _, _ := strings.Cut(strings.TrimSpace(firma), "(")
	return nombre
}
