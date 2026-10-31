package consola

import (
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Atender una etapa (pedido del usuario, 2026-10-31): cada etapa ⏳ o ❌ de una tarea abierta es un botón del panel
// que le pide a la IA que se pronuncie: intenta completarla y, si falta algo, deja un pedido en el panel o pregunta.
// qdd arma el mensaje exacto y lo vuelve a calcular al pedirlo; el panel solo pasa el id y lo envía a la conversación.

const (
	TipoAtender    = "atender"
	prefijoAtender = "atender:"

	// MaxMensajeAtender es el tope del mensaje que recibe la IA: un resumen estilo chat, no un párrafo (pedido del
	// usuario, 2026-10-31).
	MaxMensajeAtender = 280
	maxDato           = 90 // tope del bloqueo o la falla citados en el mensaje
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
				Etiqueta: c.Estado + " " + etapa + " · " + f.Nombre, Detalle: mensajeAtender(f, etapa, c),
			})
		}
	}
	return out
}

// mensajeAtender es lo que la IA recibe como mensaje de la persona, comprimido como un chat: qué etapa, por qué está
// así y qué hacer si falta algo (un pedido con su tarea, para que salga del panel si la tarea sigue cambiando, o una
// pregunta). Se acortan el nombre y lo citado, nunca las instrucciones; el total no pasa de MaxMensajeAtender.
func mensajeAtender(f estado.Frente, etapa string, c estado.Celda) string {
	situacion := "espera " + primero(c.Bloqueo, "sin detalle") + " (" + primero(c.DeQuien, "¿?") + ")"
	pedido := "ciérrala"
	if c.Estado == estado.Falla {
		situacion = "falla: " + primero(c.Falla, "sin detalle")
		pedido = "busca la causa y corrígela"
	}
	cabeza := fmt.Sprintf("%s %s · %s (tarea %s): %s.", c.Estado, etapa, recortar(f.Nombre, 50), f.ID, recortar(situacion, maxDato))
	cola := fmt.Sprintf(" → %s · ¿falta algo mío? qdd_consola_pedir (tarea %s) · ¿una decisión? pregúntame · ¿bloqueada? di qué falta · fin: qdd estado set", pedido, f.ID)
	return recortar(cabeza, MaxMensajeAtender-utf8.RuneCountInString(cola)) + cola
}

// recortar deja s en n caracteres como mucho, con «…» si lo cortó.
func recortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// atender entrega el mensaje para la IA tal cual: el panel lo envía a la conversación.
func atender(a Accion, salida io.Writer) int {
	fmt.Fprintln(salida, a.Detalle)
	return 0
}
