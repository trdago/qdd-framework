package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/mejoras"
)

// Rule: Prohibido usar 'else'
// Cola de mejoras (pkg/mejoras): doctor la revisa si el proyecto tiene cola. MEJ-01: sus archivos se leen (no se
// repara solo: dice cuál). MEJ-02: no más de 2 en tarea (lo deja así una edición a mano; dice qué hacer). MEJ-03: cada
// mejora en tarea tiene su tarea en el tablero; --fix corre el ciclo sin red, que la devuelve a la cola y la entrega.

const grupoMejoras = "10. Cola de mejoras (qdd mejoras)"

const (
	itemMejorasLegible = "Archivos de la cola legibles (MEJ-01)"
	itemMejorasTope    = "Como mucho 2 mejoras en tarea (MEJ-02)"
	itemMejorasTareas  = "Cada mejora en tarea tiene su tarea (MEJ-03)"
)

func buildMejorasChecks(projectPath string) *CheckGroup {
	g := &CheckGroup{Name: grupoMejoras}
	r := mejoras.Revisar(projectPath)
	if !r.Hay {
		return g
	}
	if r.Ilegible != nil {
		g.Items = append(g.Items, &CheckItem{Name: itemMejorasLegible, Error: r.Ilegible.Error() + ": corrígelo o bórralo a mano"})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMejorasLegible, Success: true})
	tope := &CheckItem{Name: itemMejorasTope, Success: r.EnTarea <= mejoras.MaxTareas, Error: fmt.Sprintf("%d en tarea", r.EnTarea)}
	if !tope.Success {
		tope.Error = fmt.Sprintf("%d en tarea (tope %d): cierra una tarea o devuelve su mejora a en_cola", r.EnTarea, mejoras.MaxTareas)
	}
	g.Items = append(g.Items, tope)
	if len(r.SinTarea) == 0 {
		g.Items = append(g.Items, &CheckItem{Name: itemMejorasTareas, Success: true})
		return g
	}
	g.Items = append(g.Items, &CheckItem{Name: itemMejorasTareas, Error: "sin su tarea en el tablero (qdd doctor --fix corre el ciclo): " + strings.Join(r.SinTarea, ", ")})
	return g
}

// repairMejoras corre el ciclo sin red: sigue las tareas y entrega de nuevo las que perdieron la suya.
func repairMejoras(projectPath string, g *CheckGroup) {
	for _, item := range g.Items {
		if item.Success || item.Name != itemMejorasTareas {
			continue
		}
		fmt.Println("  [~] Corriendo el ciclo de mejoras (sin GitHub)...")
		r, err := mejoras.Ciclo(projectPath, mejoras.Fuentes{}, time.Now())
		imprimirReparado(r.Tareas, err)
	}
}
