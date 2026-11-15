package remoto

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// Logrado bajo pedido (contrato con qdd-movile-83, 2026-11-15): un teléfono que declara la capacidad
// logrado_bajo_pedido pide lo logrado de un proyecto al abrirlo, y el estado deja de llevarlo cuando todos los
// teléfonos mirando la declaran. Lo logrado era lo que más pesaba del estado (FND-209).

func telefonoCon(dispositivo string, capacidades ...string) Evento {
	m, _ := json.Marshal(map[string]any{"rol": "movil", "dispositivo": dispositivo, "phx_ref": "r-" + dispositivo, "version_app": "1.0.0", "capacidades": capacidades})
	pc, _ := json.Marshal(map[string]string{"rol": "pc", "phx_ref": "r-pc"})
	return Evento{Tipo: TipoPresencia, Presencias: map[string][]json.RawMessage{"pc": {pc}, dispositivo: {m}}}
}

func juntarTelefonos(a, b Evento) Evento {
	todo := map[string][]json.RawMessage{}
	for k, v := range a.Presencias {
		todo[k] = v
	}
	for k, v := range b.Presencias {
		todo[k] = v
	}
	return Evento{Tipo: TipoPresencia, Presencias: todo}
}

// anotarLogradoPrueba deja n logros del más nuevo al más viejo en el archivo local del proyecto.
func anotarLogradoPrueba(t *testing.T, root string, n int, salida string) {
	t.Helper()
	ahora := time.Now().UTC()
	l := make([]consola.Logro, n)
	for i := range l {
		l[i] = consola.Logro{Tipo: consola.LogroPedido, ID: fmt.Sprintf("pedido:%d", i), Titulo: fmt.Sprintf("Pedido %d", i), En: ahora.Add(-time.Duration(i) * time.Minute).Format(time.RFC3339), Salida: salida}
	}
	ruta, err := estado.RutaLocal(root, "logrado")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(l)
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, b, 0o600); err != nil {
		t.Fatal(err)
	}
}

func proyectoConLogrado(t *testing.T, nombre string, n int, salida string, en time.Time) (string, string) {
	t.Helper()
	root := proyecto(t, nombre, true)
	anotarLogradoPrueba(t, root, n, salida)
	if err := registro.Anotar(root, en); err != nil {
		t.Fatal(err)
	}
	reg, _ := registro.Leer()
	for _, p := range reg.Proyectos {
		if p.Ruta == root {
			return root, p.Clave
		}
	}
	t.Fatal("el proyecto no quedó en el registro")
	return "", ""
}

func estadoEnviado(t *testing.T, e enviado) Estado {
	t.Helper()
	var est Estado
	if err := json.Unmarshal(e.Payload, &est); err != nil {
		t.Fatal(err)
	}
	return est
}

func pedirLogrado(proyecto, solicitud string) Evento {
	p, _ := json.Marshal(map[string]any{"version": VersionProtocolo, "proyecto": proyecto, "solicitud": solicitud})
	return Evento{Tipo: TipoBroadcast, Evento: EventoPedirLogrado, Payload: p}
}

func respuestaDe(t *testing.T, e enviado) RespuestaLogrado {
	t.Helper()
	var r RespuestaLogrado
	if err := json.Unmarshal(e.Payload, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestLogrado_SiTodosLoPidenElEstadoNoLoLlevaYAnunciaQueSePide(t *testing.T) {
	homeTemporal(t)
	proyectoConLogrado(t, "p", 3, "ok", time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	est := estadoEnviado(t, tr.esperar(t, 10*time.Second, "estado"))
	panel := est.Proyectos[0].Panel
	if panel.Logrado != nil || !panel.LogradoBajoPedido {
		t.Fatalf("todos lo piden: sin logrado y con el anuncio: %+v %v", panel.Logrado, panel.LogradoBajoPedido)
	}
}

func TestLogrado_SiUnTelefonoNoLoPideElEstadoLoLleva(t *testing.T) {
	homeTemporal(t)
	proyectoConLogrado(t, "p", 3, "ok", time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- juntarTelefonos(telefonoCon("d1", CapacidadLogradoBajoPedido), telefonoCon("d2"))
	panel := estadoEnviado(t, tr.esperar(t, 10*time.Second, "estado")).Proyectos[0].Panel
	if panel.Logrado == nil || len(*panel.Logrado) != 3 || !panel.LogradoBajoPedido {
		t.Fatalf("uno no lo pide: el estado lo lleva, y el PC igual anuncia que sabe: %+v", panel.Logrado)
	}
}

func TestLogrado_SiSeSumaUnTelefonoSinLaCapacidadVaUnEstadoCompletoConLogrado(t *testing.T) {
	homeTemporal(t)
	proyectoConLogrado(t, "p", 2, "ok", time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	if est := estadoEnviado(t, tr.esperar(t, 10*time.Second, "estado")); est.Proyectos[0].Panel.Logrado != nil {
		t.Fatal("al principio todos lo piden")
	}
	tr.eventos <- juntarTelefonos(telefonoCon("d1", CapacidadLogradoBajoPedido), telefonoCon("d2"))
	panel := estadoEnviado(t, tr.esperar(t, 10*time.Second, "estado")).Proyectos[0].Panel
	if panel.Logrado == nil || len(*panel.Logrado) != 2 {
		t.Fatalf("el teléfono nuevo no lo pide: estado completo con logrado, sin esperar un cambio: %+v", panel.Logrado)
	}
}

func TestLogrado_PedirloDevuelveSusEntradasConSuSolicitud(t *testing.T) {
	homeTemporal(t)
	_, clave := proyectoConLogrado(t, "p", 3, strings.Repeat("x", 1500)+"fin", time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	tr.esperar(t, 10*time.Second, "estado")
	tr.eventos <- pedirLogrado(clave, "11111111-1111-1111-1111-111111111111")
	r := respuestaDe(t, tr.esperar(t, 10*time.Second, EventoLogrado))
	if r.Version != VersionProtocolo || r.Proyecto != clave || r.Solicitud != "11111111-1111-1111-1111-111111111111" || r.Error != "" || r.Recortado {
		t.Fatalf("%+v", r)
	}
	if len(r.Entradas) != 3 || r.Entradas[0].ID != "pedido:0" {
		t.Fatalf("las entradas, la más nueva primero: %+v", r.Entradas)
	}
	if n := len([]rune(r.Entradas[0].Salida)); n != 1503 {
		t.Fatalf("a pedido va la salida del contrato (≤ 2000), no la recortada del estado: %d", n)
	}
}

func TestLogrado_UnPedidoInvalidoSeIgnora(t *testing.T) {
	homeTemporal(t)
	_, clave := proyectoConLogrado(t, "p", 1, "ok", time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	tr.esperar(t, 10*time.Second, "estado")
	tr.eventos <- pedirLogrado("Con Mayúsculas", "s1")
	tr.eventos <- pedirLogrado(clave, strings.Repeat("a", 65))
	tr.eventos <- pedirLogrado(clave, "")
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: EventoPedirLogrado, Payload: json.RawMessage(`{"proyecto": 3}`)}
	time.Sleep(150 * time.Millisecond)
	for _, e := range tr.Enviados() {
		if e.Evento == EventoLogrado {
			t.Fatalf("un pedido inválido no se responde: %s", e.Payload)
		}
	}
}

func TestLogrado_RepetidoEn5sRecibeLaMismaRespuestaSinRecalcular(t *testing.T) {
	homeTemporal(t)
	root, clave := proyectoConLogrado(t, "p", 2, "ok", time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	tr.esperar(t, 10*time.Second, "estado")
	tr.eventos <- pedirLogrado(clave, "s1")
	primera := respuestaDe(t, tr.esperar(t, 10*time.Second, EventoLogrado))
	anotarLogradoPrueba(t, root, 5, "ok")
	tr.eventos <- pedirLogrado(clave, "s2")
	segunda := respuestaDe(t, tr.esperar(t, 10*time.Second, EventoLogrado))
	if segunda.Solicitud != "s2" || len(segunda.Entradas) != len(primera.Entradas) {
		t.Fatalf("dentro de 5 s, la misma respuesta con su propia solicitud: %d → %d (%s)", len(primera.Entradas), len(segunda.Entradas), segunda.Solicitud)
	}
}

func TestLogrado_MasDe64KBQuitaLoMasViejoYDiceRecortado(t *testing.T) {
	homeTemporal(t)
	_, clave := proyectoConLogrado(t, "p", consola.TopeLogrado, strings.Repeat("salida larga ", 160), time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	tr.esperar(t, 10*time.Second, "estado")
	tr.eventos <- pedirLogrado(clave, "s1")
	e := tr.esperar(t, 10*time.Second, EventoLogrado)
	r := respuestaDe(t, e)
	if !r.Recortado || len(r.Entradas) == 0 || len(r.Entradas) >= consola.TopeLogrado || len(e.Payload) > TopeRespuestaLogrado {
		t.Fatalf("cabe en 64 KB sacando lo más viejo: %d entradas, %d bytes, recortado=%v", len(r.Entradas), len(e.Payload), r.Recortado)
	}
	if r.Entradas[0].ID != "pedido:0" {
		t.Fatalf("se queda lo más nuevo: %s", r.Entradas[0].ID)
	}
}

func TestLogrado_ProyectoDesconocidoOFueraDelPlanDiceElError(t *testing.T) {
	homeTemporal(t)
	_, viejo := proyectoConLogrado(t, "viejo", 1, "ok", time.Now().Add(-time.Hour))
	proyectoConLogrado(t, "nuevo", 1, "ok", time.Now())
	uno := 1
	_, tr, _, _ := bucleDePrueba(t, func(b *Bucle) { b.R.(*relayBucleFalso).plan = Plan{Plan: "gratis", PCs: 1, Proyectos: &uno} })
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	tr.esperar(t, 10*time.Second, "estado")
	tr.eventos <- pedirLogrado("no-existe-abc", "s1")
	if r := respuestaDe(t, tr.esperar(t, 10*time.Second, EventoLogrado)); r.Error != ErrorLogradoProyecto || r.Entradas == nil || len(r.Entradas) != 0 {
		t.Fatalf("desconocido: %+v", r)
	}
	tr.eventos <- pedirLogrado(viejo, "s2")
	if r := respuestaDe(t, tr.esperar(t, 10*time.Second, EventoLogrado)); r.Error != ErrorLogradoPlan || len(r.Entradas) != 0 {
		t.Fatalf("fuera del plan: %+v", r)
	}
}

func TestLogrado_SinTelefonoNoSeResponde(t *testing.T) {
	homeTemporal(t)
	_, clave := proyectoConLogrado(t, "p", 1, "ok", time.Now())
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- pedirLogrado(clave, "s1")
	time.Sleep(150 * time.Millisecond)
	for _, e := range tr.Enviados() {
		if e.Evento == EventoLogrado {
			t.Fatal("sin un teléfono mirando no se publica nada (inv. 12)")
		}
	}
}

// El peso del estado con datos de prueba grandes: 3 proyectos con lo logrado lleno.
func TestLogrado_ElEstadoPesaMenosCuandoTodosLoPiden(t *testing.T) {
	homeTemporal(t)
	for _, n := range []string{"a", "b", "c"} {
		proyectoConLogrado(t, n, consola.TopeLogrado, strings.Repeat("salida ", 300), time.Now())
	}
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefonoCon("d2")
	con := tr.esperar(t, 10*time.Second, "estado")
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	sin := tr.esperar(t, 10*time.Second, "estado")
	t.Logf("estado con logrado %d bytes, bajo pedido %d bytes", len(con.Payload), len(sin.Payload))
	if len(sin.Payload) >= len(con.Payload)/2 {
		t.Fatalf("sin lo logrado el estado baja de forma medible: %d → %d", len(con.Payload), len(sin.Payload))
	}
}

// FND-214: con todos los teléfonos pidiendo lo logrado, armar el estado no lo lee ni lo pasa por el
// ocultador para después descartarlo. Ese trabajo, con 60 entradas por proyecto, hacía cada estado lento y la prueba de
// peso fallaba con la máquina cargada.
func TestLogrado_SinLogradoNoSeArmaNiSeOculta(t *testing.T) {
	homeTemporal(t)
	root, clave := proyectoConLogrado(t, "a", 3, "hecho", time.Now())
	reg := registro.Registro{Proyectos: []registro.Proyecto{{Ruta: root, Clave: clave}}}
	if p := ArmarCon("v", reg, time.Now(), 0, false).Proyectos[0]; p.Panel == nil || p.Panel.Logrado != nil {
		t.Fatalf("sin lo logrado, el panel no lo trae: %+v", p.Panel)
	}
	if p := ArmarCon("v", reg, time.Now(), 0, true).Proyectos[0]; p.Panel == nil || p.Panel.Logrado == nil || len(*p.Panel.Logrado) != 3 {
		t.Fatalf("con lo logrado, el panel lo trae: %+v", p.Panel)
	}
}
