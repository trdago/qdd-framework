package remoto

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'

// La consola es de solo lectura (inv. 7): lo que un teléfono manda por el canal con forma de consola no escribe en la
// conversación ni en la cola de la sesión, y sin Pro tampoco una instrucción a la IA entra a la cola.
// contrato: remoto#179be606
func TestConsola_ElTelefonoNoEscribeEnLaConsolaYSinProNiUnaInstruccion(t *testing.T) {
	root, clave, _, tr := bucleConQDDFalso(t, func(b *Bucle) {
		b.R = &relayBucleFalso{plan: PlanGratis(), avisos: make(chan aviso, 32)}
	})
	transcript := escribirTranscript(t, root, "s", turno("user", "hola"))
	antes, err := os.ReadFile(transcript)
	if err != nil {
		t.Fatal(err)
	}
	tr.eventos <- telefono("d1", clave)
	tr.esperar(t, 2*time.Second, "consola")

	lineas, _ := json.Marshal(MensajeConsola{Version: VersionProtocolo, Proyecto: clave, Lineas: []LineaConsola{{Rol: "persona", Texto: "!rm -rf ~"}}})
	for _, evento := range []string{"consola", "escribir", "teclear"} {
		tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: evento, Payload: lineas}
	}
	ia := ejecucionFirmada("2e6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, "ia", "borra todo", "d1", ahoraEn())
	mandarEjecucion(tr, ia)
	estadoDe(t, tr, time.Second)
	if got := estadoDe(t, tr, 2*time.Second); got.Estado != EstadoRechazada {
		t.Fatalf("sin Pro, una instrucción a la IA se rechaza: %+v", got)
	}
	time.Sleep(200 * time.Millisecond)

	if cola := consola.EjecucionesA(root, time.Now()); len(cola) != 0 {
		t.Fatalf("nada quedó para la sesión: %+v", cola)
	}
	despues, _ := os.ReadFile(transcript)
	if string(despues) != string(antes) {
		t.Fatalf("la conversación no cambió:\n%s", despues)
	}
	if tomadas, _ := consola.TomarInstrucciones(root, "sesion-a", time.Now()); len(tomadas) != 0 {
		t.Fatalf("ninguna sesión recibe nada: %+v", tomadas)
	}
}

// Sin ajustes, el bucle junta los cambios 300 ms: dos cambios a 150 ms uno del otro salen en un solo delta (inv. 13).
// contrato: remoto#a79ebe5c
func TestBucle_SinAjustesJuntaLosCambiosDe300ms(t *testing.T) {
	homeTemporal(t)
	a := proyecto(t, "a", true)
	registro.Anotar(a, time.Now())
	_, tr, _, _ := bucleDePrueba(t, func(b *Bucle) {
		b.Agrupar = 0
		b.CadaSondeo = time.Hour
	})
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	inicio := time.Now()
	consola.Pedir(a, "echo uno", "Uno", "", time.Now())
	time.Sleep(150 * time.Millisecond)
	consola.Pedir(a, "echo dos", "Dos", "", time.Now())
	e := tr.esperar(t, 3*time.Second, "delta")
	if hace := time.Since(inicio); hace < 250*time.Millisecond {
		t.Fatalf("el delta salió a los %v: no esperó la ventana", hace)
	}
	var d struct {
		Proyectos map[string]*consola.DatosConsola
	}
	json.Unmarshal(e.Payload, &d)
	clave, _ := registro.ClaveDe(a)
	pedidos := 0
	for _, ac := range d.Proyectos[clave].Acciones {
		if ac.Tipo == consola.TipoPedido {
			pedidos++
		}
	}
	if pedidos != 2 {
		t.Fatalf("los dos cambios en el mismo delta: %d pedidos (%s)", pedidos, e.Payload)
	}
	if Agrupar != 300*time.Millisecond {
		t.Fatalf("la ventana es de 300 ms: %v", Agrupar)
	}
}
