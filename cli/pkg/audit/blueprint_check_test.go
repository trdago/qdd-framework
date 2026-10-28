package audit

import (
	"path/filepath"
	"testing"
)

func TestBlueprintCheck_SilentWithoutBlueprints(t *testing.T) {
	if v := RunBlueprintCheck(t.TempDir()); len(v) != 0 {
		t.Fatalf("sin blueprints no debe haber violaciones: %v", v)
	}
}

func TestBlueprintCheck_ReportsLegacyCopyInsideTheProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "blueprints", "auth", "blueprint.yaml"), `id: auth
nombre: Auth
version: 1.0.0
ficha:
  nombre: Auth
  objetivo_operativo: {problema: p, criterio_de_exito: c}
  objetivo_tecnologico: {arquitectura: a}
`)
	if !hasRule(RunBlueprintCheck(dir), "BP-05-LEGACY") {
		t.Fatal("una copia dentro del proyecto (esquema anterior o a mano) debe reportarse")
	}
}
