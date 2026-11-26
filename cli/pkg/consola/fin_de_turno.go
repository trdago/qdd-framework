package consola

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Al terminar un turno de la conversación principal (MEJ-026, issue #150), qdd revisa si la IA le deja algo a la persona
// solo en el chat: desde el teléfono no se ve el chat, y el trabajo quedaría detenido sin que nadie lo note. Es
// conservador: cuenta como pregunta solo un mensaje que termina en «?», y como comando solo una orden en imperativo con el
// comando entre comillas invertidas. Una devolución por turno: el que ya volvió termina.

// TurnoTerminado es lo que el panel sabe al terminar un turno: el último mensaje de la IA y las herramientas que llamó.
type TurnoTerminado struct {
	Mensaje      string            `json:"mensaje"`
	Herramientas []LlamadaDelTurno `json:"herramientas"`
	// YaDevuelto: el turno ya volvió una vez a la IA (stop_hook_active de Claude Code).
	YaDevuelto bool `json:"ya_devuelto"`
}

// LlamadaDelTurno es una herramienta de la conversación principal; la entrada va solo en las que pueden dejar algo en el
// panel.
type LlamadaDelTurno struct {
	Tool    string         `json:"tool"`
	Entrada map[string]any `json:"entrada,omitempty"`
}

const topePregunta = 200

var (
	// Un imperativo al empezar una línea, un punto de lista o una oración, y el comando entre comillas invertidas que no
	// es un comando de barra de Claude Code (/reload-plugins lo escribe la persona en el chat, no va al panel).
	reComandoParaLaPersona = regexp.MustCompile("(?im)(?:^|[.:;!?]\\s+)(?:[-*]\\s+|\\d+[.)]\\s+)?(?:corre|ejecuta|escribe|lanza|pega|run|execute|type)\\b[^`\\n]{0,40}`([^`/\\n][^`\\n]*)`")
	// «! comando»: la forma de Claude Code de pedirle a la persona que corra algo en la sesión.
	reComandoConBang = regexp.MustCompile("`(!\\s*[^`\\n]+)`")
	rePedirPorCLI    = regexp.MustCompile(`\bqdd\S*\s+consola\s+pedir\b`)
	reEsperaPorCLI   = regexp.MustCompile(`\bqdd\S*\s+estado\s+set\b[^\n]*(?:\bespera\b|⏳)`)
)

// QueFaltaAlTerminar dice qué tiene que dejar la IA en el panel antes de terminar, o "" si nada.
func QueFaltaAlTerminar(t TurnoTerminado) string {
	if t.YaDevuelto || dejoAlgoParaLaPersona(t.Herramientas) {
		return ""
	}
	if pregunta := preguntaFinal(t.Mensaje); pregunta != "" {
		return fmt.Sprintf("Terminaste con una pregunta para la persona («%s») que no quedó en el panel ni en el teléfono, y desde ahí no se ve el chat. "+
			"Déjala como ⏳ con qdd_estado (estado espera, bloqueo con la pregunta ¿…?, de_quien) o, si es elegir entre opciones, con AskUserQuestion y la opción recomendada. "+
			"Escríbela para que se entienda sola: qué se decide y para qué. Si ya está en el panel, termina sin repetirla.", pregunta)
	}
	if comando := comandoParaLaPersona(t.Mensaje); comando != "" {
		return fmt.Sprintf("Le pediste a la persona que corra «%s» y no quedó en el panel ni en el teléfono. "+
			"Déjalo con qdd_consola_pedir: el motivo dice qué hace y para qué, y de_persona=true si siempre debe esperar su «s  Sí», aunque el modo goal esté activo. "+
			"Si ya está en el panel, termina sin repetirlo.", comando)
	}
	return ""
}

// preguntaFinal: la última oración del mensaje si termina en «?», recortada; "" si el mensaje no termina preguntando.
func preguntaFinal(mensaje string) string {
	texto := strings.TrimRight(mensaje, " \t\r\n*_)")
	if !strings.HasSuffix(texto, "?") {
		return ""
	}
	linea := texto[strings.LastIndex(texto, "\n")+1:]
	desde := strings.LastIndex(linea, "¿")
	if desde < 0 {
		desde = inicioDeOracion(linea)
	}
	return recortar(strings.Trim(linea[desde:], " *_"), topePregunta)
}

// inicioDeOracion: dónde empieza la última oración de una línea sin «¿».
func inicioDeOracion(linea string) int {
	sinFinal := strings.TrimSuffix(linea, "?")
	corte := strings.LastIndexAny(sinFinal, ".!?")
	if corte < 0 {
		return 0
	}
	return corte + 1
}

func comandoParaLaPersona(mensaje string) string {
	if m := reComandoConBang.FindStringSubmatch(mensaje); m != nil {
		return recortar(m[1], topePregunta)
	}
	if m := reComandoParaLaPersona.FindStringSubmatch(mensaje); m != nil {
		return recortar(m[1], topePregunta)
	}
	return ""
}

// dejoAlgoParaLaPersona: en el turno quedó un ⏳, un pedido, una pregunta de AskUserQuestion o una ficha en borrador
// (que llega a «▶ Para ti»).
func dejoAlgoParaLaPersona(llamadas []LlamadaDelTurno) bool {
	for _, l := range llamadas {
		if dejaAlgo(l) {
			return true
		}
	}
	return false
}

func dejaAlgo(l LlamadaDelTurno) bool {
	texto := func(k string) string { s, _ := l.Entrada[k].(string); return strings.TrimSpace(s) }
	switch {
	case l.Tool == "AskUserQuestion", strings.HasSuffix(l.Tool, "qdd_module_save"):
		return true
	case strings.HasSuffix(l.Tool, "qdd_consola_pedir"):
		return texto("comando") != ""
	case strings.HasSuffix(l.Tool, "qdd_estado"):
		return estado.NormalizarEstado(texto("estado")) == estado.Espera
	case l.Tool == "Bash", l.Tool == "PowerShell":
		return rePedirPorCLI.MatchString(texto("command")) || reEsperaPorCLI.MatchString(texto("command"))
	}
	return false
}
