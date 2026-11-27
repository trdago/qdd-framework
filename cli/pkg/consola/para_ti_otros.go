package consola

import (
	"os"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// «Para ti» de todos los proyectos (2026-11-15): la persona mira el panel de una sesión, y cada panel mostraba solo su
// proyecto; así no vio las fichas ni las preguntas de proyecto-b, abiertas en otra sesión. Con --con otros, el panel
// trae lo que espera a la persona en los demás proyectos de la máquina (registro de proyectos, el visto más
// recientemente primero), con su carpeta. MEJ-086 (la persona eligió «Solo este + una línea», 2026-11-15): el panel
// ya no los dibuja; solo los cuenta y nombra en una línea que abre la vista Proyectos.

const (
	// ConOtros es el opcional del panel que trae los demás proyectos.
	ConOtros = "otros"
	// OtrosRecientes: un proyecto que no se abrió en este tiempo no se lee.
	OtrosRecientes = 7 * 24 * time.Hour
	// MaxOtros es cuántos proyectos con algo para la persona trae, como mucho.
	MaxOtros = 6
)

// ProyectoEsperando es otro proyecto con algo para la persona: lo que lleva turno en su panel y el texto de sus ⏳.
type ProyectoEsperando struct {
	Proyecto string           `json:"proyecto"`
	Carpeta  string           `json:"carpeta"`
	Acciones []Accion         `json:"acciones"`
	Esperas  []EsperaDelPanel `json:"esperas"`
}

// OtrosProyectos lee cada proyecto reciente del registro (menos el de root) como lo leería su propio panel y deja los
// que esperan a la persona. Nunca falla: sin registro, no hay otros.
func OtrosProyectos(root string, now time.Time) []ProyectoEsperando {
	out := []ProyectoEsperando{}
	reg, err := registro.Leer()
	if err != nil {
		return out
	}
	propio := estado.ClaveLocal(root)
	for _, p := range reg.Proyectos {
		if len(out) >= MaxOtros {
			break
		}
		if p.Ruta == propio || now.Sub(p.Visto) > OtrosRecientes {
			continue
		}
		if info, err := os.Stat(p.Ruta); err != nil || !info.IsDir() {
			continue
		}
		acciones := esperanALaPersona(Panel(p.Ruta, now, TipoPregunta).Acciones)
		if len(acciones) == 0 {
			continue
		}
		out = append(out, ProyectoEsperando{Proyecto: registro.NombreDe(p.Ruta), Carpeta: p.Ruta, Acciones: acciones, Esperas: EsperasDelPanel(p.Ruta, now)})
	}
	return out
}

func esperanALaPersona(acciones []Accion) []Accion {
	out := []Accion{}
	for _, a := range acciones {
		if a.Turno == 0 || !sePuedeContestar(a) {
			continue
		}
		out = append(out, a)
	}
	return out
}
