package audit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/modules"
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

// BP-08: lo que no se puede medir sin red (el CI) se informa sin bloquear el gate.
func TestRunBlueprintCheck_ConformidadSinMedirEsRecomendacion_BP08(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "acceso"), 0755); err != nil {
		t.Fatal(err)
	}
	m := modules.Module{ID: "acceso", Nombre: "acceso", Rutas: []string{"internal/acceso"},
		ObjetivoOperativo:   modules.ObjetivoOperativo{Problema: "Entrar.", CriterioDeExito: "Entra."},
		ObjetivoTecnologico: modules.ObjetivoTecnologico{Arquitectura: "Un login."},
		Pruebas:             modules.Pruebas{Comandos: []string{"go test ./internal/..."}},
		Implementa:          []modules.Implementacion{{Blueprint: "auth"}}}
	if _, problems := modules.Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
	v := RunBlueprintCheck(root)
	if len(v) != 1 || v[0].RuleID != "BP-08-SIN-MEDIR" || !v[0].Recomendacion || len(Bloqueantes(v)) != 0 {
		t.Fatalf("sin el blueprint en la máquina, la conformidad queda sin medir y no bloquea: %+v", v)
	}
}
