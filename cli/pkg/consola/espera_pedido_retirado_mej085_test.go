package consola

import (
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-085 (FND-235): dos ⏳ pedían confirmar un pedido por su [tecla]; el pedido se retiró («la tarea siguió
// cambiando») y los ⏳ siguieron pidiendo esa tecla toda la tarde: desde el teléfono, una acción que no existe. Como un
// pedido hecho (FND-176), uno retirado de la misma tarea deja de preguntarse, 'qdd estado check' le pide a la IA
// pedirlo de nuevo o actualizar el ⏳, y el aviso del retiro nombra ese ⏳.

// conPedidoRetirado deja en Barra un pedido que la sesión «s1» pidió y que se retira porque la tarea siguió después de
// un mensaje de la persona: el ⏳ que lo nombra es justamente ese cambio.
func conPedidoRetirado(t *testing.T, root string) Pedido {
	t.Helper()
	t.Setenv("CLAUDE_CODE_SESSION_ID", "s1")
	pd := mustPedirEnOrden(t, root, "gh pr merge 42", Orden{Tarea: "barra"})
	if err := MarcarCiclo(root, "auto", enCola.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	conEspera(t, root, "¿Confirmas ["+pd.Tecla+"] para publicar?")
	if got := pedidoDe(t, root, pd.ID, enCola.Add(2*time.Minute)); got.Estado != EstadoRetirado {
		t.Fatalf("la tarea siguió: el pedido se retira: %+v", got)
	}
	return pd
}

func TestEspera_UnPedidoRetiradoDeSuTareaNoSePreguntaAlTelefono_MEJ085(t *testing.T) {
	root := proyectoConTareas(t)
	pd := conPedidoRetirado(t, root)
	if ps := delEstado(root); len(ps) != 0 {
		t.Fatalf("[%s] ya no está en el panel: el teléfono no pide una acción que no existe: %+v", pd.Tecla, ps)
	}
	tras := EsperasTrasRetiro(root, enCola.Add(2*time.Minute))
	if len(tras) != 1 || tras[0].Tarea != "Barra" || tras[0].Etapa != "PR" {
		t.Fatalf("el check le pide a la IA pedirlo de nuevo o actualizar ese ⏳: %+v", tras)
	}
	if tras := EsperasTrasPedido(root, enCola.Add(2*time.Minute)); len(tras) != 0 {
		t.Fatalf("no terminó bien: no se confunde con uno hecho: %+v", tras)
	}
}

func TestEspera_PedidoDeNuevoVuelveACubrirElEspera_MEJ085(t *testing.T) {
	root := proyectoConTareas(t)
	pd := conPedidoRetirado(t, root)
	otra, err := PedirEnOrden(root, "gh pr merge 42", "Publicar", "", Orden{Tarea: "barra"}, enCola.Add(3*time.Minute))
	if err != nil || otra.Tecla != pd.Tecla {
		t.Fatalf("vuelve con su tecla: %+v %v", otra, err)
	}
	if tras := EsperasTrasRetiro(root, enCola.Add(3*time.Minute)); len(tras) != 0 {
		t.Fatalf("pedido de nuevo, el ⏳ vuelve a nombrar un pedido abierto: %+v", tras)
	}
}

func TestEspera_UnRetiradoDeOtraTareaNoLaCubre_MEJ085(t *testing.T) {
	root := proyectoConTareas(t)
	pd := mustPedirEnOrden(t, root, "gh pr merge 42", Orden{Tarea: "otra"})
	MarcarCiclo(root, "auto", enCola.Add(time.Minute))
	cambiarTarea(t, root, "otra", "commit c3")
	if got := pedidoDe(t, root, pd.ID, enCola.Add(2*time.Minute)); got.Estado != EstadoRetirado {
		t.Fatalf("%+v", got)
	}
	conEspera(t, root, "¿Confirmas ["+pd.Tecla+"] para publicar?")
	if tras := EsperasTrasRetiro(root, enCola.Add(2*time.Minute)); len(tras) != 0 {
		t.Fatalf("la tecla de otra tarea no es la de este ⏳: %+v", tras)
	}
}

func TestPedido_ElAvisoDelRetiroNombraElEsperaQueLoPide_MEJ085(t *testing.T) {
	root := proyectoConTareas(t)
	pd := conPedidoRetirado(t, root)
	avisos := tomarRetirados(root, "s1", enCola.Add(2*time.Minute))
	if len(avisos) != 1 || !strings.Contains(avisos[0], "["+pd.Tecla+"]") || !strings.Contains(avisos[0], "Barra · PR") {
		t.Fatalf("el aviso dice qué ⏳ nombra el pedido retirado, para actualizarlo: %v", avisos)
	}
}
