package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// RFC 0009 §3.4: la ficha del estado se busca por el blueprint que la generó, no por un id fijo.

func fichaDeEstado(t *testing.T, root, id, blueprint, etapas string) {
	t.Helper()
	dir := filepath.Join(root, ".qdd", "project", "modules")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	ficha := "id: " + id + "\nnombre: Estado\nblueprint: " + blueprint + "\nobjetivo_tecnologico:\n    decisiones:\n        - etapas → " + etapas + "\n"
	if err := os.WriteFile(filepath.Join(dir, id+".yaml"), []byte(ficha), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestEstadoOpciones_LeeLaFichaDeEstadoPorTareas(t *testing.T) {
	root := t.TempDir()
	fichaDeEstado(t, root, "estado-por-tareas", "estado-por-tareas@2.0.0", "modelos")
	if got := estadoOpciones(root)["etapas"]; got != "modelos" {
		t.Fatalf("un proyecto nuevo usa la ficha de estado-por-tareas: %q", got)
	}
}

func TestEstadoOpciones_UnProyectoMigradoConservaSuFichaEstadoPorFrente(t *testing.T) {
	root := t.TempDir()
	fichaDeEstado(t, root, "estado-por-frente", "estado-por-tareas@2.0.0", "sin-codigo")
	if got := estadoOpciones(root)["etapas"]; got != "sin-codigo" {
		t.Fatalf("migrado, el módulo conserva su id y se encuentra por su blueprint: %q", got)
	}
	legado := t.TempDir()
	fichaDeEstado(t, legado, "estado-por-frente", "estado-por-frente@1.0.0", "software")
	if got := estadoOpciones(legado)["etapas"]; got != "software" {
		t.Fatalf("sin migrar, sigue funcionando: %q", got)
	}
}

func TestAvisoDeReemplazo_DiceQueMigroYQueFaltaAplicar(t *testing.T) {
	e := blueprints.LockEntry{ID: "estado-por-tareas", Reemplazo: []string{"estado-por-frente"}}
	aviso := avisoDeReemplazo(e)
	if !strings.Contains(aviso, "reemplaza a estado-por-frente") || !strings.Contains(aviso, "qdd blueprint apply estado-por-tareas") {
		t.Fatalf("la salida dice qué migró y que falta volver a aplicarlo: %q", aviso)
	}
	if avisoDeReemplazo(blueprints.LockEntry{ID: "auth"}) != "" {
		t.Fatal("sin reemplazo no hay aviso")
	}
}
