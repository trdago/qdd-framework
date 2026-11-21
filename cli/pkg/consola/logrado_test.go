package consola

import (
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Logrado (rediseño de QDD Móvil elegido por la persona, 2026-11-09: «el historial por día de lo que salió bien, con
// pedidos hechos y su salida, tareas cerradas con su motivo, objetivos cumplidos con su evidencia y lo que mandaste
// desde el teléfono con cómo terminó»). qdd lo arma; el panel lo recibe con --con logrado.

func proyectoConEstado(t *testing.T) string {
	t.Helper()
	root := proyectoConCola(t)
	if err := estado.Init(root, []string{"Código", "PR"}, enCola); err != nil {
		t.Fatal(err)
	}
	return root
}

func nuevaTarea(t *testing.T, root, nombre string) string {
	t.Helper()
	id, err := estado.Nueva(root, estado.CambioNueva{Nombre: nombre}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func porID(l []Logro) map[string]Logro {
	m := map[string]Logro{}
	for _, x := range l {
		m[x.ID] = x
	}
	return m
}

func TestLogrado_TareasCerradasYObjetivosCumplidosNoLosDescartados(t *testing.T) {
	root := proyectoConEstado(t)
	hecha, descartada := nuevaTarea(t, root, "Barra inferior"), nuevaTarea(t, root, "Idea vieja")
	if err := estado.Cerrar(root, hecha, "quedó en la rc.52", false, enCola); err != nil {
		t.Fatal(err)
	}
	if err := estado.Cerrar(root, descartada, "ya no va", true, enCola); err != nil {
		t.Fatal(err)
	}
	if err := estado.GuardarObjetivo(root, estado.CambioObjetivo{ID: "app", Nombre: "App en la tienda", Criterio: "se instala desde Play"}, enCola); err != nil {
		t.Fatal(err)
	}
	if err := estado.CerrarObjetivo(root, "app", "la instalé desde Play", false, enCola.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	l := porID(Logrado(root, enCola.Add(2*time.Hour)))
	if got := l[hecha]; got.Tipo != LogroTarea || got.Titulo != "Barra inferior" || got.Motivo != "quedó en la rc.52" || got.En == "" {
		t.Fatalf("la tarea cerrada va con su motivo: %+v", got)
	}
	if _, ok := l[descartada]; ok {
		t.Fatal("una tarea descartada no salió bien: no va")
	}
	if got := l["app"]; got.Tipo != LogroObjetivo || got.Criterio != "se instala desde Play" || got.Evidencia != "la instalé desde Play" {
		t.Fatalf("el objetivo cumplido va con su criterio y su evidencia: %+v", got)
	}
}

func TestLogrado_UnPedidoHechoQuedaAunqueSalgaDeLaCola(t *testing.T) {
	root := proyectoConCola(t)
	bien := mustPedir(t, root, "echo listo", "Publicar la rc.52", "")
	mal := mustPedir(t, root, "false", "Algo que falla", "")
	terminar(root, bien.ID, 0, strings.Repeat("x", 3000)+"fin", false)
	terminar(root, mal.ID, 1, "error", false)
	mas := time.Now().Add(VidaHecho + time.Hour)
	if len(PedidosA(root, mas)) != 1 {
		t.Fatalf("el hecho ya salió de la cola: %+v", PedidosA(root, mas))
	}
	l := porID(Logrado(root, mas))
	got, ok := l[prefijoLogroPedido+bien.ID]
	if !ok || got.Tipo != LogroPedido || got.Titulo != "Publicar la rc.52" || got.Detalle != "echo listo" || got.Origen != OrigenPanel {
		t.Fatalf("el pedido hecho queda en lo logrado: %+v", got)
	}
	if len([]rune(got.Salida)) > TopeSalidaLogro || !strings.HasSuffix(got.Salida, "fin") {
		t.Fatalf("la salida va recortada, con su final: %d", len(got.Salida))
	}
	if _, ok := l[prefijoLogroPedido+mal.ID]; ok {
		t.Fatal("un pedido que falló no va")
	}
}

func TestLogrado_LoDelTelefono(t *testing.T) {
	root := proyectoConCola(t)
	if _, err := RegistrarEjecucion(root, Ejecucion{Solicitud: "s1", Tipo: EjecutarQDD, Texto: "qdd audit", Dispositivo: "d"}, enCola); err != nil {
		t.Fatal(err)
	}
	if err := TerminarEjecucion(root, "s1", 0, "aprobada", enCola.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := RegistrarEjecucion(root, Ejecucion{Solicitud: "s2", Tipo: EjecutarIA, Texto: "revisa el flujo", Dispositivo: "d"}, enCola); err != nil {
		t.Fatal(err)
	}
	if _, err := TomarInstrucciones(root, "sesion-1", enCola.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	l := porID(Logrado(root, enCola.Add(time.Hour)))
	if got := l[prefijoLogroTelefono+"s1"]; got.Tipo != LogroPedido || got.Origen != OrigenTelefono || got.Detalle != "qdd audit" || got.Salida != "aprobada" {
		t.Fatalf("un comando del teléfono es un pedido de origen telefono: %+v", got)
	}
	if got := l[prefijoLogroTelefono+"s2"]; got.Tipo != LogroMensaje || got.Titulo != "revisa el flujo" || got.Estado != "entregado" {
		t.Fatalf("una instrucción entregada es un mensaje: %+v", got)
	}
}

func TestLogrado_DelMasNuevoAlMasViejoCatorceDiasYTope(t *testing.T) {
	root := proyectoConEstado(t)
	vieja := nuevaTarea(t, root, "De hace un mes")
	if err := estado.Cerrar(root, vieja, "listo", false, enCola.Add(-30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < TopeLogrado+5; i++ {
		id := nuevaTarea(t, root, "Tarea "+string(rune('a'+i%26))+string(rune('a'+i/26)))
		if err := estado.Cerrar(root, id, "listo", false, enCola.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	l := Logrado(root, enCola.Add(24*time.Hour))
	if len(l) != TopeLogrado {
		t.Fatalf("como mucho %d: %d", TopeLogrado, len(l))
	}
	if porID(l)[vieja].ID != "" {
		t.Fatal("lo de hace más de 14 días no va")
	}
	for i := 1; i < len(l); i++ {
		if l[i].En > l[i-1].En {
			t.Fatalf("del más nuevo al más viejo: %s antes de %s", l[i-1].En, l[i].En)
		}
	}
}

func TestLogrado_ElPanelLoTraeSoloSiLoPide(t *testing.T) {
	root := proyectoConEstado(t)
	if Panel(root, enCola).Logrado != nil {
		t.Fatal("sin --con logrado no va: un panel anterior no lo pide")
	}
	d := Panel(root, enCola, ConLogrado)
	if d.Logrado == nil || len(*d.Logrado) != 0 {
		t.Fatalf("pedido y sin nada logrado es una lista vacía: %+v", d.Logrado)
	}
}
