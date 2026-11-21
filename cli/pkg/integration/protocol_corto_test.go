package integration

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-062: el bloque se carga en cada sesión y pesaba 8,3 KB. Se acortó a la mitad sin perder ninguna regla: cada frase
// de abajo es una regla (o su parte que cambia la conducta) del bloque de la revisión 4.

const topeBloque = 4148 // la mitad de los 8296 bytes de la revisión 4

func TestProtocolo_EsCorto_MEJ062(t *testing.T) {
	if n := len(strings.TrimSpace(QDDCommandProtocol)); n > topeBloque {
		t.Fatalf("el bloque pesa %d bytes: se carga en cada sesión y no debe pasar de %d", n, topeBloque)
	}
}

func TestProtocolo_ConservaCadaRegla_MEJ062(t *testing.T) {
	reglas := map[string][]string{
		"catálogo de comandos": {"/qdd", "init", "validate", "certify", "review", "ui", "api", "db", "release", "resolve", "goal", "report", "evolution", "docs", "qdd --help"},
		"bloque administrado":  {"never edit this block", ".qdd/"},
		"antes de tocar código": {"qdd_project_context", "qdd modules context", "Tests first", "zero else", "fewer people read code line by line",
			"at a glance", "guard clauses", "Every bug is a finding with a regression test", "qdd_module_save", "qdd modules approve", "Never commit or deploy unless 'qdd audit' passes"},
		"blueprints": {"qdd_blueprint_add", "never invent one", "qdd_blueprint_apply", "design questions exactly as returned", "never answer them yourself",
			"not negotiable", "central folder", ".qdd/blueprints.lock.yaml", "never copy or edit", "qdd_blueprint_contribute", "qdd_blueprint_propose",
			"user's confirmation", "never set a version", "BP-05-LEGACY", "BP-06-BEHIND", "qdd doctor --fix"},
		"tarea en curso": {"CURRENT TASK", "accion=activar", "qdd estado activa", "2 minutes", "accion=terminar"},
		"comandos para la persona": {"qdd_consola_pedir", "qdd consola pedir", "saying why", "never run it yourself", "qdd consola accion",
			"--despues", "only what can run now", "exit 75", "One command per human step", "chain of merges", "Another repository"},
		"preguntas":        {"QUESTIONS FOR THE HUMAN", "never stay only in the chat", "AskUserQuestion", "recommended option", "qdd_estado", "¿…?", "phone", "qdd estado check"},
		"temporales":       {"./tmp", "never in the root or source folders", "with its test"},
		"modo de permisos": {"auto mode", "acceptEdits", "bypassPermissions", "dontAsk", "approved module record", "blueprint decision or invariant", "version, tag, publication or deployment", "cannot be undone", "design questions stay the human's"},
		"modo goal":        {"GOAL MODE", "never activate it", "recommended option", "🎯", "next open task", "tool permission stops the mode"},
		"auto-asociación":  {"without a /qdd command", "closest QDD command", "and why"},
	}
	for regla, frases := range reglas {
		for _, frase := range frases {
			if !strings.Contains(QDDCommandProtocol, frase) {
				t.Errorf("%s: el bloque perdió %q", regla, frase)
			}
		}
	}
}
