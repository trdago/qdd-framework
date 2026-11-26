package consola

import (
	"fmt"
	"strings"
)

// Rule: Prohibido usar 'else'
// FND-236 (2026-11-15): Claude Code lee el mod al abrir la consola y no lo vuelve a leer. Una consola abierta antes de
// instalar una versión nueva sigue dibujando con el mod anterior (en proyecto-b y fairplay_agent, «Para
// ti» seguía mezclando los proyectos después de rc.88). El instalador deja su versión en el mod (marcaVersionPanel);
// el panel la manda como «panel:<versión>» en --con, y qdd avisa si no es la suya. Un qdd anterior ignora lo que no
// conoce en --con.

const (
	// marcaVersionPanel es lo que el mod dice en el repositorio; el instalador la cambia por su versión.
	marcaVersionPanel = "__QDD_VERSION__"
	// PrefijoVersionPanel es cómo el panel dice su versión en --con.
	PrefijoVersionPanel = "panel:"
)

// AvisoDeReinicio: el panel de esta consola es de otra versión que el qdd instalado. Vacío si es la misma, si el mod
// no pasó por el instalador o si qdd no sabe su versión (un binario compilado a mano).
func AvisoDeReinicio(panel, instalada string) string {
	p, i := strings.TrimPrefix(strings.TrimSpace(panel), "v"), strings.TrimPrefix(strings.TrimSpace(instalada), "v")
	if p == "" || p == marcaVersionPanel || i == "" || i == "dev" || p == i {
		return ""
	}
	return fmt.Sprintf("⟳ Este panel es de qdd %s y está instalado %s: cierra esta consola y ábrela con claude --continue (sigue la misma conversación) para usar lo nuevo", p, i)
}
