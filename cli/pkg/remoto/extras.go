package remoto

import (
	"os"
	"time"
)

// Rule: Prohibido usar 'else'
// Lo que el bucle lleva además del estado: las solicitudes recordadas y los resultados de las acciones (rc.3), y la
// consola (rc.4).

type extrasBucle struct {
	resultados  chan resultadoAccion
	solicitudes map[string]Solicitud
	ocultadores map[string]ocultadorLeido
	consolaTick <-chan time.Time
	consola     estadoConsola
}

func (b *Bucle) iniciarExtras() {
	b.extras.resultados = make(chan resultadoAccion, 16)
	b.extras.solicitudes = leerSolicitudes()
	b.extras.ocultadores = map[string]ocultadorLeido{}
	b.extras.consola.miradas = map[string]*mirada{}
	if b.Bin == "" {
		b.Bin, _ = os.Executable()
	}
}
