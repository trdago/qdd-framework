package consola

import (
	"testing"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// FND-153 (2026-11-05): dos sesiones en un repositorio. El pedido del #132 de una quedó asociado a mej-002, la tarea en
// curso de la otra; al cerrarla esa otra sesión, el pedido se retiró. Es el error derivado que reportó sfrc en MEJ-010.

const sesionClaude = "CLAUDE_CODE_SESSION_ID"

func pedirSinTarea(t *testing.T, root, comando string) Pedido {
	t.Helper()
	p, err := Pedir(root, comando, "Motivo", "", enCola)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPedido_NoHeredaLaTareaEnCursoDeOtraSesion_FND153(t *testing.T) {
	root := proyectoConTareas(t)
	t.Setenv(sesionClaude, "sesion-a")
	if err := estado.Activar(root, "barra", "mcp", enCola); err != nil {
		t.Fatal(err)
	}
	t.Setenv(sesionClaude, "sesion-b")
	if p := pedirSinTarea(t, root, "echo otra"); p.Tarea != "" {
		t.Fatalf("la tarea en curso es de otra sesión: %+v", p)
	}
	t.Setenv(sesionClaude, "sesion-a")
	if p := pedirSinTarea(t, root, "echo mia"); p.Tarea != "barra" {
		t.Fatalf("la propia sí se hereda: %+v", p)
	}
}

func TestPedido_ElPedidoDeOtraSesionNoSeRetiraAlCambiarEsaTarea_FND153(t *testing.T) {
	root := proyectoConTareas(t)
	t.Setenv(sesionClaude, "sesion-a")
	if err := estado.Activar(root, "barra", "mcp", enCola); err != nil {
		t.Fatal(err)
	}
	t.Setenv(sesionClaude, "sesion-b")
	p := pedirSinTarea(t, root, "echo otra")
	if err := AnotarHuellas(root, enCola); err != nil {
		t.Fatal(err)
	}
	cambiarTarea(t, root, "barra", "commit a2")
	if got := pedidoDe(t, root, p.ID, enCola).Estado; got == EstadoRetirado {
		t.Fatalf("no era de esa tarea: %s", got)
	}
}

func TestPedido_SinSesionSigueHeredandoLaTareaEnCurso_FND153(t *testing.T) {
	root := proyectoConTareas(t)
	t.Setenv(sesionClaude, "")
	if err := estado.Activar(root, "barra", "cli", enCola); err != nil {
		t.Fatal(err)
	}
	if p := pedirSinTarea(t, root, "echo sola"); p.Tarea != "barra" {
		t.Fatalf("una IA sin id de sesión conserva lo que ya hacía: %+v", p)
	}
}

func TestPedido_ConSesionNoHeredaUnaActividadSinSesion_FND153(t *testing.T) {
	root := proyectoConTareas(t)
	t.Setenv(sesionClaude, "")
	if err := estado.Activar(root, "barra", "cli", enCola); err != nil {
		t.Fatal(err)
	}
	t.Setenv(sesionClaude, "sesion-a")
	if p := pedirSinTarea(t, root, "echo con"); p.Tarea != "" {
		t.Fatalf("no se sabe de quién es: %+v", p)
	}
}
