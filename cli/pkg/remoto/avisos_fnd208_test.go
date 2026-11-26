package remoto

import (
	"context"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// FND-208: el relay y QDD Móvil aceptan los push «pregunta» y «permiso» (y sus _resuelto), pero el PC solo pedía los de
// pedidos y fichas. Una pregunta de la IA o un permiso de herramienta que esperaba a la persona no sonaba en el teléfono.

func conPreguntas(pendientes map[string]string, cerradas ...string) Estado {
	acciones := []consola.Accion{}
	for id, origen := range pendientes {
		acciones = append(acciones, consola.Accion{ID: id, Tipo: consola.TipoPregunta, Estado: consola.PreguntaPendiente, Origen: origen})
	}
	for _, id := range cerradas {
		acciones = append(acciones, consola.Accion{ID: id, Tipo: consola.TipoPregunta, Estado: consola.PreguntaRespondida, Origen: consola.OrigenIA})
	}
	return Estado{Proyectos: []ProyectoEstado{{Clave: "c", Nombre: "p", Panel: &consola.DatosConsola{Acciones: acciones}}}}
}

// contrato: remoto#c4f9cc27
func TestAvisos_UnaPreguntaNuevaPideSuPush_FND208(t *testing.T) {
	rl := nuevoRelayBucle()
	b := bucleDeAvisos(rl)
	ctx := context.Background()
	b.avisarNuevos(ctx, conPreguntas(nil))
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"pregunta:1": consola.OrigenIA}))
	esperarAviso(t, rl, AvisoPregunta)
	b.avisarNuevos(ctx, conPreguntas(nil, "pregunta:1"))
	esperarAviso(t, rl, AvisoPreguntaResuelta)
	sinAviso(t, rl, "una pregunta respondida no vuelve a pedir push")
}

func TestAvisos_UnPermisoDeHerramientaPideElPushPermiso_FND208(t *testing.T) {
	rl := nuevoRelayBucle()
	b := bucleDeAvisos(rl)
	ctx := context.Background()
	b.avisarNuevos(ctx, conPreguntas(nil))
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"pregunta:2": consola.OrigenPermiso}))
	esperarAviso(t, rl, AvisoPermiso)
	b.avisarNuevos(ctx, conPreguntas(nil))
	esperarAviso(t, rl, AvisoPermisoResuelto)
}

func TestAvisos_UnRelayQueRechazaElResueltoDeUnaPreguntaNoEsFalla_FND208(t *testing.T) {
	for _, tipo := range []string{AvisoPreguntaResuelta, AvisoPermisoResuelto} {
		if !esResuelto(tipo) {
			t.Fatalf("%s es un push de resuelto", tipo)
		}
	}
	rl := nuevoRelayBucle()
	rl.errAviso = ErrRechazada{Texto: "el relay respondió 400"}
	b := bucleDeAvisos(rl)
	b.avisar(context.Background(), AvisoPreguntaResuelta, "p", "c")
	esperarAviso(t, rl, AvisoPreguntaResuelta)
	select {
	case err := <-b.avisos:
		t.Fatalf("no se reporta como falla: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
}
