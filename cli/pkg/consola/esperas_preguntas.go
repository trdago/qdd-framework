package consola

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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

const (
	// OrigenEstado es el origen de una pregunta que sale de un ⏳ de la persona en el estado.
	OrigenEstado = "estado"
	// VidaEsperaRespondida es cuánto se recuerda una espera respondida: si la IA no la refleja en ese tiempo, el
	// teléfono vuelve a preguntarla.
	VidaEsperaRespondida = 24 * time.Hour

	prefijoEspera = "espera-"
)

// esperasDeLaPersona son las preguntas pendientes de los ⏳ de la persona de las tareas abiertas.
func esperasDeLaPersona(root string, now time.Time) []Pregunta {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []Pregunta
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			if estado.NormalizarEstado(c.Estado) != estado.Espera || !esDeLaPersona(c.DeQuien) || strings.TrimSpace(c.Bloqueo) == "" {
				continue
			}
			out = append(out, preguntaDeEspera(f, etapa, strings.TrimSpace(c.Bloqueo), now))
		}
	}
	return out
}

func preguntaDeEspera(f estado.Frente, etapa, bloqueo string, now time.Time) Pregunta {
	suma := sha256.Sum256([]byte(f.ID + "\x00" + etapa + "\x00" + bloqueo))
	donde := f.Nombre + " · " + etapa
	return Pregunta{
		ID: prefijoEspera + hex.EncodeToString(suma[:6]), Origen: OrigenEstado, Estado: PreguntaPendiente,
		Etiqueta:  recortar("⏳ "+donde+": "+bloqueo, MaxMotivo),
		Preguntas: []ItemPregunta{{ID: "q1", Encabezado: donde, Texto: bloqueo, Otra: true}},
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
		r, err := validarRespuesta(p, respuesta)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrRespuestaInvalida, err)
		}
		cerrada := now.UTC()
		p.Estado, p.RespondidaEn, p.Lugar, p.Respuesta, p.Cerrada = PreguntaRespondida, EnTelefono, strings.TrimSpace(lugar), r, &cerrada
		instruccion := Ejecucion{Solicitud: fmt.Sprintf("%s-%d", p.ID, now.Unix()), Tipo: EjecutarIA, Texto: mensajeEspera(p), Nombre: p.Lugar}
		if _, err := RegistrarEjecucion(root, instruccion, now); err != nil {
			return nil, err
		}
		out = p
		return append([]Pregunta{p}, cola...), nil
	})
	return out, err
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
