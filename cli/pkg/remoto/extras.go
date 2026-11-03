package remoto

import (
	"os"
	"time"
)

// Rule: Prohibido usar 'else'
// Lo que el bucle lleva además del estado: las solicitudes recordadas y los resultados de las acciones (rc.3), la
// consola (rc.4) y lo que el teléfono mandó a ejecutar (ejecutar.go).

type extrasBucle struct {
	resultados  chan resultadoAccion
	solicitudes map[string]Solicitud
	ocultadores map[string]ocultadorLeido
	consolaTick <-chan time.Time
	consola     estadoConsola
	corriendo   map[string]bool // las solicitudes de un comando qdd del teléfono que corren ahora (ejecutar.go)
	plan        string          // el plan guardado en plan.json (JSON), para escribirlo solo si cambió
}

func (b *Bucle) iniciarExtras() {
	b.extras.resultados = make(chan resultadoAccion, 16)
	b.extras.solicitudes = leerSolicitudes()
	b.extras.ocultadores = map[string]ocultadorLeido{}
	b.extras.consola.miradas = map[string]*mirada{}
	b.extras.corriendo = map[string]bool{}
	if b.Bin == "" {
		b.Bin, _ = os.Executable()
	}
}
