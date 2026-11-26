package remoto

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// FND-289: en qdd siempre queda algún ⏳ del tablero con una pregunta («¿…?») para la persona, y esos cuentan como
// preguntas. El pregunta_resuelto solo salía cuando no quedaba ninguna, así que el aviso de una pregunta de la IA que
// el modo goal o la persona respondió nunca se quitaba del teléfono (13:18:53, 2026-11-21: «push pregunta de qdd» y
// ningún resuelto; los logs del relay confirman que no llegó ninguno).

// Respondida la pregunta que pidió el push, sale su resuelto aunque sigan los ⏳ que esperaban desde antes y que nunca
// se avisaron.
func TestAvisos_ElResueltoSaleCuandoSeRespondeLoQueSeAvisoAunqueQuedenOtras_FND289(t *testing.T) {
	rl := nuevoRelayBucle()
	b := bucleDeAvisos(rl)
	ctx := context.Background()
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA}))
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA, "pregunta:7": consola.OrigenIA}))
	esperarAviso(t, rl, AvisoPregunta)
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA}, "pregunta:7"))
	esperarAviso(t, rl, AvisoPreguntaResuelta)
	sinAviso(t, rl, "el ⏳ de antes no se vuelve a avisar")
}

// Con dos avisadas, el resuelto espera a que se respondan las dos: el aviso del teléfono agrupa las de su tipo.
func TestAvisos_ConDosAvisadasElResueltoEsperaALaUltima_FND289(t *testing.T) {
	rl := nuevoRelayBucle()
	b := bucleDeAvisos(rl)
	ctx := context.Background()
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA}))
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA, "pregunta:1": consola.OrigenIA, "pregunta:2": consola.OrigenIA}))
	esperarAviso(t, rl, AvisoPregunta)
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA, "pregunta:2": consola.OrigenIA}, "pregunta:1"))
	sinAviso(t, rl, "queda una avisada: el aviso sigue")
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA}, "pregunta:1", "pregunta:2"))
	esperarAviso(t, rl, AvisoPreguntaResuelta)
}

// Si qdd remoto se reinicia (una actualización) entre el aviso y la respuesta, recuerda qué pidió el push: lo avisado
// queda en <QDD_HOME>/remoto/avisados.json, y en el primer ciclo del bucle nuevo sale el resuelto de lo ya respondido.
func TestAvisos_LoAvisadoSobreviveUnReinicioYSuResueltoSale_FND289(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	ctx := context.Background()
	rl := nuevoRelayBucle()
	antes := bucleDeAvisos(rl)
	antes.cargarAvisados()
	antes.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA}))
	antes.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA, "pregunta:9": consola.OrigenIA}))
	esperarAviso(t, rl, AvisoPregunta)

	despues := bucleDeAvisos(rl)
	despues.cargarAvisados()
	despues.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA}, "pregunta:9"))
	esperarAviso(t, rl, AvisoPreguntaResuelta)
	despues.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA}, "pregunta:9"))
	sinAviso(t, rl, "el resuelto sale una vez")

	otra := bucleDeAvisos(rl)
	otra.cargarAvisados()
	otra.avisarNuevos(ctx, conPreguntas(map[string]string{"estado:antigua": consola.OrigenIA}, "pregunta:9"))
	sinAviso(t, rl, "lo resuelto ya no queda guardado")
}

// Sin cargarAvisados (las pruebas que arman el bucle a mano) no se escribe nada en QDD_HOME.
func TestAvisos_SinCargarNoGuardaNada_FND289(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	rl := nuevoRelayBucle()
	b := bucleDeAvisos(rl)
	ctx := context.Background()
	b.avisarNuevos(ctx, conPreguntas(nil))
	b.avisarNuevos(ctx, conPreguntas(map[string]string{"pregunta:1": consola.OrigenIA}))
	esperarAviso(t, rl, AvisoPregunta)
	if _, err := os.Stat(filepath.Join(home, "remoto", "avisados.json")); !os.IsNotExist(err) {
		t.Fatalf("sin cargarAvisados no se guarda: %v", err)
	}
}
