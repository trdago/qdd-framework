package remoto

import (
	"os"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// MEJ-082 (FND-233): con dos sesiones de Claude Code en el proyecto, el teléfono veía la consola de una y la instrucción
// la recibía la otra. La consola dice qué sesión muestra, y una instrucción va a esa sesión si sigue latiendo.

// dosSesiones deja dos transcripts en root (s-vieja escrita antes que s-nueva) y las dos sesiones latiendo.
func dosSesiones(t *testing.T, root string) {
	t.Helper()
	vieja := escribirTranscript(t, root, "s-vieja", turno("assistant", "de la vieja"))
	antes := time.Now().Add(-time.Hour)
	os.Chtimes(vieja, antes, antes)
	escribirTranscript(t, root, "s-nueva", turno("assistant", "de la nueva"))
	consola.LatirSesion(root, "s-vieja", time.Now())
	consola.LatirSesion(root, "s-nueva", time.Now())
}

func TestSesionDeLaConsola_EsLaDelTranscriptQueSeMuestraMientrasLate_MEJ082(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	dosSesiones(t, root)
	if got := sesionDeLaConsola(root, time.Now()); got != "s-nueva" {
		t.Fatalf("la sesión de la consola que se publica: %q", got)
	}
	consola.CerrarSesion(root, "s-nueva", time.Now())
	if got := sesionDeLaConsola(root, time.Now()); got != "" {
		t.Fatalf("si esa sesión ya no late, la instrucción no va dirigida: %q", got)
	}
}

func TestEjecutar_LaInstruccionVaALaSesionQueElTelefonoVe_MEJ082(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t)
	dosSesiones(t, root)
	mandarEjecucion(tr, ejecucionFirmada("bc6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", "Prueba desde el teléfono", "d1", ahoraEn()))
	estadoDe(t, tr, time.Second)
	estadoDe(t, tr, 2*time.Second)
	if otra, _ := consola.TomarParaEntregar(root, "s-vieja", time.Now()); len(otra) != 0 {
		t.Fatalf("la sesión que el teléfono no ve no la toma: %+v", otra)
	}
	if tomadas, _ := consola.TomarParaEntregar(root, "s-nueva", time.Now()); len(tomadas) != 1 {
		t.Fatalf("la toma la sesión de la consola: %+v", tomadas)
	}
}

func TestConsola_DiceQueSesionMuestra_MEJ082(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	dosSesiones(t, root)
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", clave)
	if m := consolaDe(t, tr.esperar(t, 2*time.Second, "consola")); m.Sesion != "s-nueva" {
		t.Fatalf("el evento consola dice de qué sesión es: %+v", m)
	}
}
