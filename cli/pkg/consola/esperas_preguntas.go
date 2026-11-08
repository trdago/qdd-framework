package consola

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
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
// el teléfono ve desde la pregunta, no el historial de la tarea.
// FND-176 (qdd-movile, 2026-11-07): el pedido cubría su ⏳ solo mientras seguía abierto. Al terminar salía de la cola, y
// «¿Confirmas [a]?» volvía al teléfono ya respondida. Un ⏳ que nombra un pedido hecho de su misma tarea (en las últimas
// VidaHechos) ya no pregunta nada: 'qdd estado check' le pide a la IA actualizarlo (EsperasTrasPedido). Una tecla que
// nunca tuvo pedido sigue siendo pregunta.
// FND-124 (qdd-movile, 2026-11-03): sin una sesión abierta la respuesta se perdía (la instrucción vence a los 30 min y la
// pregunta seguía «respondida» 24 h).
// Botones (propuesta de qdd-movile aprobada por la persona, 2026-11-07): cada ⏳ trae «Ya lo hice», «Todavía no» y «Ya no
// va», además de «Otra», y la respuesta se escribe en el estado: no se pierde aunque no haya una sesión (sin_sesion).
// «Ya lo hice» y «Otra» pasan el ⏳ a la IA para que lo verifique; «Ya no va» deja la etapa en —; «Todavía no» no cambia
// el estado y vuelve a preguntarse a las VidaEsperaRespondida. Con una sesión abierta, la IA además se entera al tiro.

const (
	// OrigenEstado es el origen de una pregunta que sale de un ⏳ de la persona en el estado.
	OrigenEstado = "estado"
	// VidaEsperaRespondida es cuánto se recuerda una espera respondida: si la IA no la refleja en ese tiempo, el
	// teléfono vuelve a preguntarla.
	VidaEsperaRespondida = 24 * time.Hour

	prefijoEspera = "espera-"

	// Las opciones de un ⏳ (el valor que manda el teléfono o el panel).
	EsperaHecho     = "hecho"
	EsperaTodaviaNo = "todavia_no"
	EsperaYaNoVa    = "ya_no_va"

	// deQuienVerifica: un ⏳ respondido pasa a la IA, que lo verifica y lo refleja.
	deQuienVerifica = "la IA"

	maxLineaEspera = 160
)

var opcionesEspera = []OpcionPregunta{
	{Valor: EsperaHecho, Etiqueta: "Ya lo hice", Descripcion: "La IA lo verifica"},
	{Valor: EsperaTodaviaNo, Etiqueta: "Todavía no", Descripcion: "Vuelve a preguntarse mañana"},
	{Valor: EsperaYaNoVa, Etiqueta: "Ya no va", Descripcion: "La etapa queda en —"},
}

var (
	frasePregunta = regexp.MustCompile(`¿[^¿?]*\?`)
	teclaNombrada = regexp.MustCompile(`\[([a-z0-9])\]`)
)

// preguntaDelBloqueo es el bloqueo desde su primera pregunta (¿…?): lo que sigue suele decir qué hacer (FND-155), y lo
// anterior es historial (MEJ-004). Vacío si no pregunta nada.
func preguntaDelBloqueo(bloqueo string) string {
	inicio := frasePregunta.FindStringIndex(bloqueo)
	if inicio == nil {
		return ""
	}
	return bloqueo[inicio[0]:]
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

// respondidaPorPedido: el ⏳ nombra un pedido de su tarea que ya terminó bien, y ninguno abierto con esa tecla.
func respondidaPorPedido(tarea, bloqueo string, abiertas map[string]bool, hechas map[string]map[string]bool) bool {
	if cubiertaPorPedido(bloqueo, abiertas) {
		return false
	}
	return cubiertaPorPedido(bloqueo, hechas[tarea])
}

// esperasDeLaPersona son las preguntas pendientes de los ⏳ de la persona de las tareas abiertas.
func esperasDeLaPersona(root string, now time.Time) []Pregunta {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []Pregunta
	abiertas := teclasAbiertas(PedidosA(root, now))
	hechas := teclasHechas(root, now)
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			bloqueo := strings.TrimSpace(c.Bloqueo)
			if estado.NormalizarEstado(c.Estado) != estado.Espera || !esDeLaPersona(c.DeQuien) || preguntaDelBloqueo(bloqueo) == "" {
				continue
			}
			// Uno con verificación se responde solo (estado/verificar.go): no se le pregunta a la persona.
			if c.Verifica != "" {
				continue
			}
			if cubiertaPorPedido(bloqueo, abiertas) || respondidaPorPedido(f.ID, bloqueo, abiertas, hechas) {
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
	hechas := teclasHechas(root, now)
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			bloqueo := strings.TrimSpace(c.Bloqueo)
			if estado.NormalizarEstado(c.Estado) != estado.Espera || !esDeLaPersona(c.DeQuien) || preguntaDelBloqueo(bloqueo) != "" || c.Verifica != "" {
				continue
			}
			if cubiertaPorPedido(bloqueo, abiertas) || respondidaPorPedido(f.ID, bloqueo, abiertas, hechas) {
				continue
			}
			out = append(out, EsperaSinPregunta{Tarea: f.Nombre, Etapa: etapa, Bloqueo: bloqueo})
		}
	}
	return out
}

// EsperasTrasPedido son los ⏳ de la persona que esperan un pedido que ya terminó bien (FND-176): el teléfono ya no los
// pregunta, y 'qdd estado check' le pide a la IA reflejar en la tabla lo que pasó.
func EsperasTrasPedido(root string, now time.Time) []EsperaSinPregunta {
	e, err := estado.Load(root)
	if err != nil {
		return nil
	}
	var out []EsperaSinPregunta
	abiertas := teclasAbiertas(PedidosA(root, now))
	hechas := teclasHechas(root, now)
	for _, f := range e.Abiertos() {
		for _, etapa := range e.EtapasDe(f) {
			c := f.Celdas[etapa]
			bloqueo := strings.TrimSpace(c.Bloqueo)
			if estado.NormalizarEstado(c.Estado) != estado.Espera || !esDeLaPersona(c.DeQuien) {
				continue
			}
			if !respondidaPorPedido(f.ID, bloqueo, abiertas, hechas) {
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
		Preguntas: []ItemPregunta{{ID: "q1", Encabezado: donde, Texto: pregunta, Otra: true, Opciones: opcionesEspera}},
		SinSesion: true,
		Creada:    now.UTC(), Vista: now.UTC(),
		// El turno no depende del texto: reescribir la pregunta no la manda al final (RFC 0019).
		claveTurno: prefijoEspera + f.ID + ":" + etapa, tareaTurno: f.ID,
		etapa: etapa, bloqueo: bloqueo,
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
// sale, salvo la recién respondida (VidaCerrada), para que otro teléfono vea «ya respondida en …». Las pendientes de la
// consola van primero, después las esperas, y al final las cerradas.
func conEsperas(cola, esperas []Pregunta, now time.Time) []Pregunta {
	vigentes := map[string]bool{}
	for _, e := range esperas {
		vigentes[e.ID] = true
	}
	enCola := map[string]bool{}
	var pendientes, cerradas []Pregunta
	for _, p := range cola {
		if p.Origen == OrigenEstado && !vigentes[p.ID] && !recienCerrada(p, now) {
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

func recienCerrada(p Pregunta, now time.Time) bool {
	return p.Cerrada != nil && now.Sub(*p.Cerrada) <= VidaCerrada
}

// responderEspera aplica la respuesta a un ⏳ de la persona: la primera gana, y queda en el estado (aplicarAlEstado).
// Desde un teléfono, si hay una sesión abierta, la IA además la recibe como instrucción; desde el PC, el panel se la
// envía al chat.
func responderEspera(root, id, respuesta, lugar, en string, now time.Time) (Pregunta, error) {
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
		p.Estado, p.RespondidaEn, p.Lugar, p.Respuesta, p.Cerrada = PreguntaRespondida, en, strings.TrimSpace(lugar), r, &cerrada
		aplicado, err := aplicarAlEstado(root, p, now)
		if err != nil {
			return nil, err
		}
		p.Aplicado = aplicado
		if err := avisarALaIA(root, p, now); err != nil {
			return nil, err
		}
		out = p
		return append([]Pregunta{p}, cola...), nil
	})
	return out, err
}

// ResponderEsperaEnPC es un botón del panel del PC sobre un ⏳ («Para ti»): devuelve la pregunta y el mensaje que el
// panel le envía a la IA (vacío si no hay nada que hacer).
func ResponderEsperaEnPC(root, id, valor string, now time.Time) (Pregunta, string, error) {
	id = IDPregunta(id)
	if !strings.HasPrefix(id, prefijoEspera) {
		return Pregunta{}, "", ErrPreguntaDesconocida
	}
	respuesta := `{"q1":{"opciones":[` + strconv.Quote(valor) + `]}}`
	p, err := responderEspera(root, id, respuesta, "", EnPC, now)
	if err != nil {
		return Pregunta{}, "", err
	}
	return p, mensajeEspera(p), nil
}

// aplicarAlEstado escribe la respuesta en la celda del ⏳ y dice qué quedó, en palabras para el teléfono.
func aplicarAlEstado(root string, p Pregunta, now time.Time) (string, error) {
	r := p.Respuesta[p.Preguntas[0].ID]
	cuando := now.Local().Format("2006-01-02 15:04") + ", " + desdeDonde(p)
	c := estado.Cambio{Frente: p.tareaTurno, Etapa: p.etapa, Estado: estado.Espera, DeQuien: deQuienVerifica}
	if r.Otra != nil {
		c.Bloqueo = "La persona respondió (" + cuando + "): «" + *r.Otra + "». Era: " + p.bloqueo
		return "el ⏳ pasa a la IA con tu respuesta", estado.Set(root, c, now)
	}
	valor := r.Opciones[0]
	if valor == EsperaTodaviaNo {
		return "vuelve a preguntarse en " + strconv.Itoa(int(VidaEsperaRespondida.Hours())) + " h", nil
	}
	if valor == EsperaYaNoVa {
		c.Estado, c.DeQuien = estado.NoAplica, ""
		c.Evidencia = "La persona dice que ya no va (" + cuando + "). Era: " + p.bloqueo
		return "la etapa queda en — (ya no va)", estado.Set(root, c, now)
	}
	c.Bloqueo = "Verificar: la persona dice que ya lo hizo (" + cuando + "). Era: " + p.bloqueo
	return "el ⏳ pasa a la IA para que lo verifique", estado.Set(root, c, now)
}

func desdeDonde(p Pregunta) string {
	if p.RespondidaEn == EnPC {
		return "desde el PC"
	}
	return "desde " + LugarDe(p)
}

// avisarALaIA deja el mensaje para la sesión abierta cuando responde un teléfono. La respuesta ya está en el estado, así
// que la pregunta no sigue esta instrucción (FND-141): si no llega, no se pierde nada.
func avisarALaIA(root string, p Pregunta, now time.Time) error {
	mensaje := mensajeEspera(p)
	if p.RespondidaEn != EnTelefono || mensaje == "" || !HaySesion(root, now) {
		return nil
	}
	instruccion := Ejecucion{Solicitud: fmt.Sprintf("%s-%d", p.ID, now.Unix()), Tipo: EjecutarIA, Texto: mensaje, Nombre: p.Lugar}
	_, err := RegistrarEjecucion(root, instruccion, now)
	return err
}

// EsperaDelPanel es una pregunta de la persona como la dibuja «▶ Para ti» (issue #149). Atender es la acción que la
// envía al chat, donde se responde.
type EsperaDelPanel struct {
	Pregunta string `json:"pregunta"` // el id de su acción, que da su lugar en la lista (RFC 0019)
	Atender  string `json:"atender"`
	Donde    string `json:"donde"`
	Texto    string `json:"texto"`
}

// EsperasDelPanel son lo que el teléfono pregunta y todavía no se respondió, en el orden de la tabla. Una nota no
// pregunta nada: sigue como ⏳ en la tabla.
func EsperasDelPanel(root string, now time.Time) []EsperaDelPanel {
	out := []EsperaDelPanel{}
	e, err := estado.Load(root)
	if err != nil {
		return out
	}
	pendientes := map[string]bool{}
	for _, p := range PreguntasA(root, now) {
		pendientes[p.ID] = p.Origen == OrigenEstado && p.Estado == PreguntaPendiente
	}
	for _, f := range e.Abiertos() {
		for i, etapa := range e.EtapasDe(f) {
			p := preguntaDeEspera(f, etapa, strings.TrimSpace(f.Celdas[etapa].Bloqueo), now)
			if !pendientes[p.ID] {
				continue
			}
			q := p.Preguntas[0]
			out = append(out, EsperaDelPanel{Pregunta: prefijoPregunta + p.ID, Atender: prefijoAtender + f.ID + ":" + strconv.Itoa(i), Donde: q.Encabezado, Texto: q.Texto})
		}
	}
	return out
}

// ReabrirEspera vuelve a preguntar un ⏳ cuya respuesta no llegó a ninguna sesión (FND-141). Solo si la respuesta
// guardada es la de esa instrucción: una más nueva no se toca.
func ReabrirEspera(root, id, instruccion string, now time.Time) error {
	return conPreguntas(root, now, func(cola []Pregunta) ([]Pregunta, error) {
		i := indicePregunta(cola, id)
		if instruccion == "" || i < 0 || cola[i].Instruccion != instruccion {
			return cola, nil
		}
		return append(cola[:i], cola[i+1:]...), nil
	})
}

// mensajeEspera es lo que lee la IA: a qué ⏳ responde, la respuesta tal cual y qué hacer. «Todavía no» no le pide nada.
func mensajeEspera(p Pregunta) string {
	q := p.Preguntas[0]
	r := p.Respuesta[q.ID]
	if r.Otra != nil {
		return mensajeDeRespuesta(q, *r.Otra)
	}
	valor := r.Opciones[0]
	if valor == EsperaTodaviaNo {
		return ""
	}
	que := "Quedó en el estado como un ⏳ tuyo: verifícalo y refléjalo (qdd_estado set)."
	if valor == EsperaYaNoVa {
		que = "La etapa quedó en — en el estado. Si algo dependía de eso, ajústalo."
	}
	return "Respondo a tu ⏳ «" + q.Encabezado + ": " + q.Texto + "»: " + etiquetaDe(q, valor) + ".\n\n" + que
}

func mensajeDeRespuesta(q ItemPregunta, texto string) string {
	return "Respondo a tu ⏳ «" + q.Encabezado + ": " + q.Texto + "»:\n\n" + texto +
		"\n\nCuando lo resuelvas, refléjalo en el estado (qdd_estado set)."
}
