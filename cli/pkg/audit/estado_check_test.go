package audit

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestEstadoCheck_SilentWithoutFile(t *testing.T) {
	if v := RunEstadoCheck(t.TempDir()); len(v) != 0 {
		t.Fatalf("sin .qdd/project/estado.yaml no debe haber violaciones: %v", v)
	}
}

func TestEstadoCheck_ReportsCheckViolations(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "project", "estado.yaml"), `formato: 1
etapas: [Código, Pruebas]
frentes:
  - id: a
    nombre: A
    estado: abierto
    celdas:
      Código: {estado: "✅"}
      Pruebas: {estado: "—"}
`)
	v := RunEstadoCheck(dir)
	if !hasRule(v, "EPF-01-CHECK") || !strings.Contains(v[0].Description, "EPF-C02") {
		t.Fatalf("un ✅ sin evidencia debe bloquear la auditoría: %v", v)
	}
}

func TestEstadoCheck_ReportsUnreadableFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "project", "estado.yaml"), "formato: [roto")
	if !hasRule(RunEstadoCheck(dir), "EPF-01-CHECK") {
		t.Fatal("un estado.yaml ilegible debe reportarse")
	}
}
