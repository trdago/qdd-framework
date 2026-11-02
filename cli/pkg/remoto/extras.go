package remoto

import (
	"context"
	"encoding/json"
	"time"
)

// Rule: Prohibido usar 'else'
// Lo que el bucle hace además de publicar el estado: las acciones firmadas (rc.3) y la consola (rc.4). En rc.2 el PC
// es de solo lectura: una acción que llegue no se ejecuta.

type resultadoAccion struct{}

type extrasBucle struct {
	resultados  chan resultadoAccion
	consolaTick <-chan time.Time
}

func (b *Bucle) iniciarExtras() {}
func (b *Bucle) alAccion(ctx context.Context, p json.RawMessage) {
	b.logf("acción recibida: este qdd todavía no ejecuta acciones remotas")
}
func (b *Bucle) alResultado(ctx context.Context, r resultadoAccion) {}
func (b *Bucle) seguirSolicitudes(ctx context.Context, e Estado)    {}
func (b *Bucle) alCambiarEspectadores(ctx context.Context)          {}
func (b *Bucle) alCambiarPlan()                                     {}
func (b *Bucle) tickConsola(ctx context.Context)                    {}
