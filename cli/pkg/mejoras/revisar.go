package mejoras

import (
	"os"
	"path/filepath"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Revisar es lo que qdd doctor ve de la cola (grupo «Cola de mejoras»): que sus archivos se lean (MEJ-01), que no haya
// más de MaxTareas en tarea (MEJ-02) y que cada una en tarea tenga su tarea en el tablero (MEJ-03; doctor --fix corre
// el ciclo, que la devuelve a la cola y la vuelve a entregar).

// Revision es el resultado de Revisar.
type Revision struct {
	Hay      bool     // el proyecto tiene cola
	Ilegible error    // MEJ-01
	EnTarea  int      // MEJ-02
	SinTarea []string // MEJ-03
}

// Revisar mira la cola sin cambiarla.
func Revisar(root string) Revision {
	if _, err := os.Stat(filepath.Join(root, Dir)); err != nil {
		return Revision{}
	}
	r := Revision{Hay: true}
	cola, err := Lista(root)
	if err != nil {
		r.Ilegible = err
		return r
	}
	tareas := map[string]bool{}
	if e, err := estado.Load(root); err == nil {
		for _, f := range e.Frentes {
			tareas[f.ID] = true
		}
	}
	for _, m := range cola {
		if m.Estado != EnTarea {
			continue
		}
		r.EnTarea++
		if !tareas[m.Tarea] {
			r.SinTarea = append(r.SinTarea, m.ID)
		}
	}
	return r
}
