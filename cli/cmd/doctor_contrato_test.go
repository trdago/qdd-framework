package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// Cada grupo que arma doctor tiene al menos una prueba que lo llama: un grupo nuevo sin prueba hace fallar esta.
// contrato: doctor#41288493
func TestDoctor_CadaGrupoTieneSuPrueba(t *testing.T) {
	fuente, err := os.ReadFile("doctor.go")
	if err != nil {
		t.Fatal(err)
	}
	grupos := regexp.MustCompile(`\b(build[A-Z][A-Za-z]*Checks)\(`).FindAllStringSubmatch(string(fuente), -1)
	if len(grupos) < 10 {
		t.Fatalf("doctor.go arma sus grupos con build…Checks: %d", len(grupos))
	}
	pruebas, _ := filepath.Glob("*_test.go")
	var todas strings.Builder
	for _, p := range pruebas {
		if p == "doctor_contrato_test.go" {
			continue
		}
		b, _ := os.ReadFile(p)
		todas.Write(b)
	}
	for _, g := range grupos {
		if !strings.Contains(todas.String(), g[1]+"(") {
			t.Errorf("el grupo %s de doctor no tiene una prueba que lo llame", g[1])
		}
	}
}
