package consola

import (
	"fmt"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Revisar lo que espera a la persona (pedido de la persona, 2026-11-26): la tecla l del panel, además de releer, le
// pide a la IA que vuelva a evaluar todo lo que espera a la persona y deje solo lo que aporta valor. qdd arma la lista
// completa, también lo que el tope del panel esconde; decidir si algo ya se respondió es de la IA, que lee git, los PR
// y la conversación. Nunca responde por la persona: sin evidencia, el ⏳ queda.

// MaxTextoRevision es cuántos caracteres de cada bloqueo van en la revisión: lo justo para reconocerlo sin gastar de más.
const MaxTextoRevision = 160

// RevisionPanel es el mensaje para la IA y cuántas cosas revisa; Texto vacío si nada espera a la persona.
type RevisionPanel struct {
	Texto string `json:"texto"`
	Total int    `json:"total"`
}

const instruccionesRevision = `Revisa todo lo que el panel espera de mí y deja solo lo que aporta valor (tecla l del panel). Por cada uno, mira la evidencia (git, PR, findings, cola de mejoras, esta conversación) antes de decidir:
- Ya está respondido o hecho: qdd estado set <tarea> <etapa> ok --evidencia "…"; un pedido que hice yo: qdd consola descartar <tecla> --hecho.
- Ya no va: qdd estado cerrar <tarea> --descartado --motivo "…"; un pedido que sobra: qdd consola descartar <tecla>.
- Se responde solo (npm:, pr:, tag:, pedido:, archivo:): ponle --verifica y deja de preguntármelo.
- Es una nota de avance o espera a otro, no a mí: cámbiale de_quien o pásala a la etapa que corresponde.
- Es una decisión con opción recomendada que tu modo de permisos te deja tomar (no cambia una ficha aprobada ni un blueprint, no es una versión ni una publicación, y no borra datos): tómala tú, anótala como elegida por la IA con la opción recomendada y sigue.
- No es para ahora (una función nueva en un congelamiento, o algo que espera a otra tarea): ponle depende_de y de_quien «después de …», para que no me espere.
- Solo yo puedo hacerlo o decidirlo (una prueba en mi equipo, un acceso que no tienes, un cambio de lo definido): que se entienda solo, como pregunta ¿…? con la opción recomendada.
Nunca respondas por mí: sin evidencia, queda como está. Al terminar, en una línea: cuántos quitaste y qué queda.`

// Revision arma la revisión de lo que espera a la persona en root: cada ⏳ suyo de las tareas abiertas y cada pedido
// que espera su confirmación.
func Revision(root string, now time.Time) RevisionPanel {
	esperas := esperasParaRevisar(root)
	pedidos := pedidosParaRevisar(root, now)
	total := len(esperas) + len(pedidos)
	if total == 0 {
		return RevisionPanel{}
	}
	var b strings.Builder
	b.WriteString(instruccionesRevision)
	if len(esperas) > 0 {
		fmt.Fprintf(&b, "\n\n⏳ que me esperan (%d):\n%s", len(esperas), strings.Join(esperas, "\n"))
	}
	if len(pedidos) > 0 {
		fmt.Fprintf(&b, "\n\nPedidos sin confirmar (%d):\n%s", len(pedidos), strings.Join(pedidos, "\n"))
	}
	return RevisionPanel{Texto: b.String(), Total: total}
}

func esperasParaRevisar(root string) []string {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			if estado.NormalizarEstado(c.Estado) != estado.Espera || !estado.DeLaPersona(c.DeQuien) {
				continue
			}
			out = append(out, "- "+f.ID+" · "+etapa+": "+cortarRevision(c.Bloqueo))
		}
	}
	return out
}

func pedidosParaRevisar(root string, now time.Time) []string {
	var out []string
	for _, p := range PedidosA(root, now) {
		if !p.Ejecutable() {
			continue
		}
		out = append(out, "- ["+p.Tecla+"] "+cortarRevision(p.Motivo))
	}
	return out
}

// cortarRevision deja el texto en una línea y en MaxTextoRevision caracteres a lo más, con «…» si se cortó.
func cortarRevision(s string) string {
	linea := strings.Join(strings.Fields(s), " ")
	r := []rune(linea)
	if len(r) <= MaxTextoRevision {
		return linea
	}
	return strings.TrimSpace(string(r[:MaxTextoRevision])) + "…"
}
