package consola

import (
	"fmt"
	"io"
	"strconv"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Atender una etapa (pedido del usuario, 2026-10-31): cada etapa ⏳ o ❌ de una tarea abierta es un botón del panel
// que le pide a la IA que se pronuncie: intenta completarla y, si falta algo, deja un pedido en el panel o pregunta.
// qdd arma el mensaje exacto y lo vuelve a calcular al pedirlo; el panel solo pasa el id y lo envía a la conversación.

const (
	TipoAtender    = "atender"
	prefijoAtender = "atender:"
)

// atenciones ofrece una acción por cada etapa en ⏳ o ❌ de las tareas abiertas: atender:<tarea>:<índice de etapa>.
func atenciones(root string) []Accion {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []Accion
	for _, f := range e.Frentes {
		if f.Estado != estado.EstadoAbierto {
			continue
		}
		for i, etapa := range e.Etapas {
			c := f.Celdas[etapa]
			if c.Estado != estado.Espera && c.Estado != estado.Falla {
				continue
			}
			out = append(out, Accion{
				ID: prefijoAtender + f.ID + ":" + strconv.Itoa(i), Tipo: TipoAtender,
				Etiqueta: "Atender «" + etapa + "» de «" + f.Nombre + "»", Detalle: mensajeAtender(f, etapa, c),
			})
		}
	}
	return out
}

// mensajeAtender es lo que la IA recibe como mensaje de la persona: qué etapa, por qué está así y qué hacer si falta
// algo (un pedido con su tarea, para que salga del panel si la tarea sigue cambiando, o una pregunta).
func mensajeAtender(f estado.Frente, etapa string, c estado.Celda) string {
	situacion := "Está ⏳ esperando: " + primero(c.Bloqueo, "sin detalle") + " (de " + primero(c.DeQuien, "¿?") + ")."
	tarea := "Intenta completarla ahora."
	if c.Estado == estado.Falla {
		situacion = "Está ❌ fallando: " + primero(c.Falla, "sin detalle") + "."
		tarea = "Encuentra la causa y corrígela."
	}
	return fmt.Sprintf("Atiende la etapa «%s» de la tarea «%s» (id %s). %s %s "+
		"Si para avanzar hace falta un comando que tengo que ejecutar yo, déjalo en el panel con qdd_consola_pedir (tarea: %s). "+
		"Si falta una decisión o un dato, pregúntame. Si todavía no se puede, dime qué falta y cuándo se podrá. "+
		"Al terminar, deja la celda al día con qdd estado set.", etapa, f.Nombre, f.ID, situacion, tarea, f.ID)
}

// atender entrega el mensaje para la IA tal cual: el panel lo envía a la conversación.
func atender(a Accion, salida io.Writer) int {
	fmt.Fprintln(salida, a.Detalle)
	return 0
}
