package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/findings"
)

// FND-062: el Zero-Panic Guard registraba cada pánico como RESOLVED (atrapar
// el pánico no corrige el defecto) y metía el mensaje entre comillas simples
// sin escaparlo: un apóstrofo dejaba el YAML inválido y el finding desaparecía
// de todos los indicadores.
func TestFindingDePanico_AbiertoYLegible(t *testing.T) {
	dir := t.TempDir()
	contenido, err := findingDePanico("FND-PANIC-1", "can't index: nil map 'x'", time.Date(2026, 10, 29, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(findings.Dir(dir), "FND-PANIC-1.yaml")
	os.MkdirAll(filepath.Dir(path), 0755)
	os.WriteFile(path, contenido, 0644)

	fs, _ := findings.Leer(dir)
	if len(fs) != 1 {
		t.Fatalf("el lector de findings no pudo leer el finding del pánico:\n%s", contenido)
	}
	if fs[0].Estado != findings.Abierto || fs[0].EstadoOriginal != "open" {
		t.Errorf("estado = %q (%q); want open: el defecto que causó el pánico sigue sin corregir", fs[0].Estado, fs[0].EstadoOriginal)
	}
	if fs[0].Impacto == "" || fs[0].ID != "FND-PANIC-1" {
		t.Errorf("finding = %+v; want id e impacto con el mensaje", fs[0])
	}
}
