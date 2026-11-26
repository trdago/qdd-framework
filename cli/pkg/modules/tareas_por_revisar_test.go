package modules

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// RFC 0017 §3: las tareas cuya ficha cambió después de su última celda llegan al contexto como una revisión pendiente.
func TestBuildContext_PideRevisarLasTareasCuyaFichaCambio(t *testing.T) {
	root := gitRepo(t)
	ctx := BuildContext(ContextInput{Root: root, TareasPorRevisar: []string{"Tarea cobro («Cobro»): la ficha pagos cambió su(s) decisiones"}})
	if !strings.Contains(ctx, "## Tareas por revisar") || !strings.Contains(ctx, "- Tarea cobro («Cobro»)") {
		t.Fatalf("el contexto pide revisar la tarea:\n%s", ctx)
	}
	if strings.Contains(BuildContext(ContextInput{Root: root}), "Tareas por revisar") {
		t.Fatal("sin tareas por revisar, el contexto no agrega la sección")
	}
}
