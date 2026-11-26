package remoto

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// FND-247: un push de pregunta no se quitó del teléfono al responderla y el registro de qdd remoto no decía nada de los
// avisos: no se podía saber si el PC mandó el pregunta_resuelto. Cada push que el PC pide queda en el registro, también
// el resuelto que un relay viejo rechaza.
func TestAvisos_CadaPushQuedaEnElRegistro_FND247(t *testing.T) {
	rl := nuevoRelayBucle()
	b := bucleDeAvisos(rl)
	var log bytes.Buffer
	b.Log = &log
	ctx := context.Background()
	b.avisarNuevos(ctx, conPreguntas(nil))
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"pregunta:1": consola.OrigenIA}))
	esperarAviso(t, rl, AvisoPregunta)
	rl.errAviso = ErrRechazada{Texto: "el relay respondió 400"}
	b.avisarNuevos(ctx, conPreguntas(nil, "pregunta:1"))
	esperarAviso(t, rl, AvisoPreguntaResuelta)
	for _, linea := range []string{"push pregunta de p", "push pregunta_resuelto de p"} {
		if !strings.Contains(log.String(), linea) {
			t.Fatalf("falta %q en el registro:\n%s", linea, log.String())
		}
	}
}
