package consola

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Pedido de sfrc (2026-11-03): toda consulta a la persona pasa por el tablero. Un ⏳ «de la persona» en el estado es
// una pregunta de respuesta libre (origen estado) que se saca del estado cada vez que se lee la cola: no hay copia que
// se desincronice, y sale sola cuando la celda cambia. El id sale de la tarea, la etapa y lo que espera, así que es el
// mismo en cada lectura. La respuesta del teléfono queda en la cola (solo para decir «ya respondida» y no volver a
// preguntar lo mismo durante VidaEsperaRespondida) y le llega a la IA como una instrucción del teléfono.
// MEJ-004 (qdd-movile, 2026-11-03): de 11 ⏳ publicados, casi todos eran notas de avance o esperaban un pedido ya armado
// en el panel. Un ⏳ es pregunta solo si pregunta algo (¿…?) y no nombra con [tecla] un pedido abierto, que ya lo cubre;
// el teléfono ve solo la pregunta, no el historial de la tarea.
// FND-124 (qdd-movile, 2026-11-03): sin una sesión abierta la respuesta se perdía (la instrucción vence a los 30 min y la
// pregunta seguía «respondida» 24 h): esa respuesta se rechaza, y qdd remoto no publica el ⏳ (sinEsperasSinSesion).

const (
	// OrigenEstado es el origen de una pregunta que sale de un ⏳ de la persona en el estado.
	OrigenEstado = "estado"
	// VidaEsperaRespondida es cuánto se recuerda una espera respondida: si la IA no la refleja en ese tiempo, el
	// teléfono vuelve a preguntarla.
	VidaEsperaRespondida = 24 * time.Hour

	prefijoEspera = "espera-"

	maxLineaEspera = 160
)

var (
	frasePregunta = regexp.MustCompile(`¿[^¿?]*\?`)
	teclaNombrada = regexp.MustCompile(`\[([a-z0-9])\]`)
)

// preguntaDelBloqueo son las preguntas (¿…?) del bloqueo, juntas; vacío si no pregunta nada.
func preguntaDelBloqueo(bloqueo string) string {
	return strings.Join(frasePregunta.FindAllString(bloqueo, -1), " ")
}

// cubiertaPorPedido: un ⏳ que nombra la tecla de un pedido abierto ([b]) ya lo cubre el panel.
func cubiertaPorPedido(bloqueo string, abiertas map[string]bool) bool {
	for _, m := range teclaNombrada.FindAllStringSubmatch(bloqueo, -1) {
		if abiertas[m[1]] {
			return true
		}
	}
	return false
}

// esperasDeLaPersona son las preguntas pendientes de los ⏳ de la persona de las tareas abiertas.
func esperasDeLaPersona(root string, now time.Time) []Pregunta {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []Pregunta
	abiertas := teclasAbiertas(PedidosA(root, now))
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			bloqueo := strings.TrimSpace(c.Bloqueo)
			if estado.NormalizarEstado(c.Estado) != estado.Espera || !esDeLaPersona(c.DeQuien) || preguntaDelBloqueo(bloqueo) == "" {
				continue
			}
			if cubiertaPorPedido(bloqueo, abiertas) {
				continue
			}
			out = append(out, preguntaDeEspera(f, etapa, bloqueo, now))
		}
	}
	return out
}

// EsperaSinPregunta es un ⏳ de la persona que no llega al teléfono: no pregunta nada y ningún pedido abierto lo cubre.
type EsperaSinPregunta struct {
	Tarea   string `json:"tarea"`
	Etapa   string `json:"etapa"`
	Bloqueo string `json:"bloqueo"`
}

// Linea es la espera en una línea, con el bloqueo recortado: el --json lo trae entero.
func (e EsperaSinPregunta) Linea() string {
	return recortar(e.Tarea+" · "+e.Etapa+": "+e.Bloqueo, maxLineaEspera)
}

// EsperasSinPregunta las lista para 'qdd estado check' (pedido de sfrc, decidido por la persona, 2026-11-03). Una nota
// de avance es válida, así que no es una violación: es un aviso, y su cuenta es la medición que decide el gate en el
// Stop hook.
func EsperasSinPregunta(root string, now time.Time) []EsperaSinPregunta {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []EsperaSinPregunta
	abiertas := teclasAbiertas(PedidosA(root, now))
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			bloqueo := strings.TrimSpace(c.Bloqueo)
			if estado.NormalizarEstado(c.Estado) != estado.Espera || !esDeLaPersona(c.DeQuien) || preguntaDelBloqueo(bloqueo) != "" {
				continue
			}
			if cubiertaPorPedido(bloqueo, abiertas) {
				continue
			}
			out = append(out, EsperaSinPregunta{Tarea: f.Nombre, Etapa: etapa, Bloqueo: bloqueo})
		}
	}
	return out
}

func preguntaDeEspera(f estado.Frente, etapa, bloqueo string, now time.Time) Pregunta {
	suma := sha256.Sum256([]byte(f.ID + "\x00" + etapa + "\x00" + bloqueo))
	donde := f.Nombre + " · " + etapa
	pregunta := preguntaDelBloqueo(bloqueo)
	return Pregunta{
		ID: prefijoEspera + hex.EncodeToString(suma[:6]), Origen: OrigenEstado, Estado: PreguntaPendiente,
		Etiqueta:  recortar("⏳ "+donde+": "+pregunta, MaxMotivo),
		Preguntas: []ItemPregunta{{ID: "q1", Encabezado: donde, Texto: pregunta, Otra: true}},
		Creada:    now.UTC(), Vista: now.UTC(),
	}
}

func esperaPorID(root, id string, now time.Time) (Pregunta, bool) {
	for _, p := range esperasDeLaPersona(root, now) {
		if p.ID == id {
			return p, true
		}
	}
	return Pregunta{}, false
}

// conEsperas suma a la cola las esperas vigentes: una respondida ya está en la cola; una que el estado ya no tiene,
// sale. Las pendientes de la consola van primero, después las esperas, y al final las cerradas.
func conEsperas(cola, esperas []Pregunta) []Pregunta {
	vigentes := map[string]bool{}
	for _, e := range esperas {
		vigentes[e.ID] = true
	}
	enCola := map[string]bool{}
	var pendientes, cerradas []Pregunta
	for _, p := range cola {
		if p.Origen == OrigenEstado && !vigentes[p.ID] {
			continue
		}
		enCola[p.ID] = true
		if p.Estado == PreguntaPendiente {
			pendientes = append(pendientes, p)
			continue
		}
		cerradas = append(cerradas, p)
	}
	for _, e := range esperas {
		if enCola[e.ID] {
			continue
		}
		pendientes = append(pendientes, e)
	}
	return append(pendientes, cerradas...)
}

// responderEspera aplica la respuesta de un teléfono a un ⏳ de la persona y se la deja a la IA: la primera gana, y
// una espera que el estado ya no tiene es desconocida.
func responderEspera(root, id, respuesta, lugar string, now time.Time) (Pregunta, error) {
	var out Pregunta
	err := conPreguntas(root, now, func(cola []Pregunta) ([]Pregunta, error) {
		if i := indicePregunta(cola, id); i >= 0 {
			return nil, &PreguntaCerrada{Pregunta: cola[i]}
		}
		p, ok := esperaPorID(root, id, now)
		if !ok {
			return nil, ErrPreguntaDesconocida
		}
		if !HaySesion(root, now) {
			return nil, fmt.Errorf("%w: no hay una sesión de Claude Code abierta en el proyecto que la reciba", ErrPreguntaDesconocida)
		}
		r, err := validarRespuesta(p, respuesta)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrRespuestaInvalida, err)
		}
		cerrada := now.UTC()
		p.Estado, p.RespondidaEn, p.Lugar, p.Respuesta, p.Cerrada = PreguntaRespondida, EnTelefono, strings.TrimSpace(lugar), r, &cerrada
		p.Instruccion = fmt.Sprintf("%s-%d", p.ID, now.Unix())
		instruccion := Ejecucion{Solicitud: p.Instruccion, Tipo: EjecutarIA, Texto: mensajeEspera(p), Nombre: p.Lugar}
		if _, err := RegistrarEjecucion(root, instruccion, now); err != nil {
			return nil, err
		}
		out = p
		return append([]Pregunta{p}, cola...), nil
	})
	return out, err
}

// ReabrirEspera vuelve a preguntar un ⏳ cuya respuesta no llegó a ninguna sesión (FND-141). Solo si la respuesta
// guardada es la de esa instrucción: una más nueva no se toca.
func ReabrirEspera(root, id, instruccion string, now time.Time) error {
	return conPreguntas(root, now, func(cola []Pregunta) ([]Pregunta, error) {
		i := indicePregunta(cola, id)
		if i < 0 || cola[i].Instruccion != instruccion {
			return cola, nil
		}
		return append(cola[:i], cola[i+1:]...), nil
	})
}

// mensajeEspera es lo que lee la IA: a qué ⏳ responde, la respuesta tal cual y cómo reflejarla.
func mensajeEspera(p Pregunta) string {
	q := p.Preguntas[0]
	texto := ""
	if r := p.Respuesta[q.ID]; r.Otra != nil {
		texto = *r.Otra
	}
	return "Respondo a tu ⏳ «" + q.Encabezado + ": " + q.Texto + "»:\n\n" + texto +
		"\n\nCuando lo resuelvas, refléjalo en el estado (qdd_estado set)."
}
