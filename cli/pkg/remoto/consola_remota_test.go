package remoto

import (
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.4): la consola sale solo mientras un teléfono la mira y se corta al irse (inv. 6), como mucho dos
// mensajes por segundo (inv. 13), sin secretos (inv. 16), y si el relay no la permite avisa una vez y para (inv. 14).

func consolaDe(t *testing.T, e enviado) MensajeConsola {
	t.Helper()
	var m MensajeConsola
	if err := json.Unmarshal(e.Payload, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func textos(m MensajeConsola) string {
	var out []string
	for _, l := range m.Lineas {
		out = append(out, l.Rol+":"+l.Texto)
	}
	return strings.Join(out, "\n")
}

func TestConsola_SoloMientrasLaMiranSinSecretosYDosPorSegundo(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	escribirTranscript(t, root, "s", turno("user", "la clave es sk-ant-api03-abcdefghijklmnopqrstuvwxyz0123456789"),
		turno("assistant", []map[string]any{{"type": "text", "text": "Listo."}, {"type": "tool_use", "name": "Edit"}}))
	_, tr, rl, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	tr.nada(t, 700*time.Millisecond)

	tr.eventos <- telefono("d1", clave)
	m := consolaDe(t, tr.esperar(t, 2*time.Second, "consola"))
	if m.Proyecto != clave || !m.Inicio || m.Version != VersionProtocolo {
		t.Fatalf("la primera tanda de quien empieza a mirar: %+v", m)
	}
	txt := textos(m)
	if strings.Contains(txt, "abcdefghijklmnop") || !strings.Contains(txt, Oculto) || !strings.Contains(txt, "ia:Listo.") || !strings.Contains(txt, "herramienta:Edit") {
		t.Fatalf("la cola, sin el secreto:\n%s", txt)
	}

	antes := tr.contar("consola")
	for i := 0; i < 20; i++ {
		escribirTranscript(t, root, "s", turno("assistant", "paso"))
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(600 * time.Millisecond)
	enviados := tr.contar("consola") - antes
	if enviados < 1 || enviados > 4 {
		t.Fatalf("20 líneas en ~1,6 s salen juntas, como mucho 2 mensajes por segundo: %d", enviados)
	}
	vistas := 0
	for _, e := range tr.Enviados() {
		if e.Evento == "consola" {
			vistas += strings.Count(textos(consolaDe(t, e)), "ia:paso")
		}
	}
	if vistas != 20 {
		t.Fatalf("no se pierde ninguna: %d", vistas)
	}

	tr.eventos <- telefono("d1", "")
	time.Sleep(100 * time.Millisecond)
	antes = tr.contar("consola")
	escribirTranscript(t, root, "s", turno("assistant", "nadie lo mira"))
	time.Sleep(time.Second)
	if tr.contar("consola") != antes {
		t.Fatal("se corta cuando nadie la mira (inv. 6)")
	}
	if len(rl.Latidos()) == 0 {
		t.Fatal("hubo latido")
	}
}

func TestConsola_UnPedidoQueTerminaSaleConSuSalida(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("comando de prueba en sh")
	}
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", clave)
	tr.esperar(t, 2*time.Second, "consola")
	p, err := consola.Pedir(root, "echo TOKEN=supersecreto123", "Probar", "", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	panel := PanelDe("v", registro.Proyecto{Ruta: root, Clave: clave}, time.Now()).Panel
	accion, err := consola.BuscarAccion(panel.Acciones, "pedido:"+p.ID)
	if err != nil {
		t.Fatal(err)
	}
	bin, _ := os.Executable()
	if c := accion.Lanzar(root, bin, "prueba", io.Discard); c != 0 {
		t.Fatalf("lanzar: %d", c)
	}
	esperarQueTermine(t, root, p.ID)
	limite := time.After(10 * time.Second)
	for {
		m := consolaDe(t, tr.esperar(t, 10*time.Second, "consola"))
		txt := textos(m)
		if strings.Contains(txt, "pedido:") {
			if !strings.Contains(txt, "Probar · terminó bien") || strings.Contains(txt, "supersecreto123") {
				t.Fatalf("el pedido, cómo terminó y su salida sin secretos:\n%s", txt)
			}
			return
		}
		select {
		case <-limite:
			t.Fatal("no salió el pedido")
		default:
		}
	}
}

func TestConsola_SiElRelayNoLaPermiteAvisaUnaVezYPara(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	escribirTranscript(t, root, "s", turno("user", "hola"))
	motivo := "sin minutos de consola este mes"
	_, tr, rl, _ := bucleDePrueba(t, func(b *Bucle) {
		b.R.(*relayBucleFalso).plan = Plan{Plan: "gratis", ConsolaPermitida: false, Motivo: &motivo}
	})
	tr.eventos <- telefono("d1", clave)
	m := consolaDe(t, tr.esperar(t, 2*time.Second, "consola"))
	if m.Pausada != motivo || len(m.Lineas) != 0 {
		t.Fatalf("avisa por qué: %+v", m)
	}
	escribirTranscript(t, root, "s", turno("user", "otra"))
	time.Sleep(1200 * time.Millisecond)
	if n := tr.contar("consola"); n != 1 {
		t.Fatalf("una sola vez y no transmite: %d", n)
	}
	for _, l := range rl.Latidos() {
		if l["consola_min"] != 0 {
			t.Fatalf("pausada no consume minutos: %v", l)
		}
	}
}

func TestConsola_CuandoVuelveAEstarPermitidaEmpiezaDeNuevo(t *testing.T) {
	b := &Bucle{plan: Plan{ConsolaPermitida: false}}
	b.iniciarExtras()
	b.extras.consola.miradas["p"] = &mirada{clave: "p", pausada: true}
	b.alCambiarPlan()
	if b.extras.consola.permitida || !b.extras.consola.miradas["p"].pausada {
		t.Fatal("sigue pausada")
	}
	b.plan.ConsolaPermitida = true
	b.alCambiarPlan()
	m := b.extras.consola.miradas["p"]
	if !b.extras.consola.permitida || m.pausada || !m.inicio {
		t.Fatalf("vuelve con la cola de nuevo: %+v", m)
	}
}

func dosTelefonos(d1, mirando1, d2, mirando2 string) Evento {
	m1, _ := json.Marshal(map[string]string{"rol": "movil", "dispositivo": d1, "mirando": mirando1, "phx_ref": "r-" + d1})
	m2, _ := json.Marshal(map[string]string{"rol": "movil", "dispositivo": d2, "mirando": mirando2, "phx_ref": "r-" + d2})
	pc, _ := json.Marshal(map[string]string{"rol": "pc", "phx_ref": "r-pc"})
	return Evento{Tipo: TipoPresencia, Presencias: map[string][]json.RawMessage{"pc": {pc}, d1: {m1}, d2: {m2}}}
}

// FND-110: un segundo teléfono que empieza a mirar una consola que otro ya miraba recibía solo lo nuevo, sin la
// tanda de inicio, y si estaba pausada no se enteraba del motivo.
func TestConsola_UnSegundoTelefonoQueEmpiezaAMirarRecibeElInicio_FND110(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	escribirTranscript(t, root, "s", turno("user", "hola"), turno("assistant", "Listo."))
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", clave)
	if m := consolaDe(t, tr.esperar(t, 2*time.Second, "consola")); !m.Inicio {
		t.Fatalf("d1 recibe el inicio: %+v", m)
	}
	antes := tr.contar("consola")
	tr.eventos <- dosTelefonos("d1", clave, "d2", clave)
	for plazo := time.Now().Add(2 * time.Second); time.Now().Before(plazo); time.Sleep(50 * time.Millisecond) {
		vistos := 0
		for _, e := range tr.Enviados() {
			if e.Evento != "consola" {
				continue
			}
			vistos++
			if m := consolaDe(t, e); vistos > antes && m.Inicio && strings.Contains(textos(m), "ia:Listo.") {
				return
			}
		}
	}
	t.Fatal("d2 empezó a mirar y no recibió la tanda de inicio con lo que ya había")
}

func TestConsola_UnSegundoTelefonoSeEnteraDeLaPausa_FND110(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	escribirTranscript(t, root, "s", turno("user", "hola"))
	motivo := "horas"
	_, tr, _, _ := bucleDePrueba(t, func(b *Bucle) {
		b.R.(*relayBucleFalso).plan = Plan{Plan: "gratis", ConsolaPermitida: false, Motivo: &motivo}
	})
	tr.eventos <- telefono("d1", clave)
	tr.esperar(t, 2*time.Second, "consola")
	tr.eventos <- dosTelefonos("d1", clave, "d2", clave)
	time.Sleep(1200 * time.Millisecond)
	if n := tr.contar("consola"); n != 2 {
		t.Fatalf("el segundo teléfono también se entera de la pausa, una vez: %d", n)
	}
}
