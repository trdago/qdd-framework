package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Ejecutar desde el teléfono»: el mod de cada sesión interactiva late con 'qdd consola telefono latir' y
// recibe, una sola vez, las instrucciones que la persona mandó desde el teléfono; el panel dice qué acepta el PC
// (ejecutar) solo con QDD Móvil activado y Pro.

type latido struct {
	Activo   bool `json:"activo"`
	Entregar []struct {
		Solicitud, Nombre, Mensaje string
	} `json:"entregar"`
}

func latir(t *testing.T, root, sesion string, fin bool) latido {
	t.Helper()
	var salida, errores bytes.Buffer
	if codigo := telefonoLatir(root, sesion, fin, &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	var l latido
	if err := json.Unmarshal(salida.Bytes(), &l); err != nil {
		t.Fatalf("%v: %s", err, salida.String())
	}
	return l
}

func TestConsolaTelefono_SinQDDMovilNoLateNiEntrega(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	consola.RegistrarEjecucion(root, consola.Ejecucion{Solicitud: "s1", Tipo: consola.EjecutarIA, Texto: "hola", Dispositivo: "d"}, time.Now())
	if l := latir(t, root, "a", false); l.Activo || len(l.Entregar) != 0 || consola.HaySesion(root, time.Now()) {
		t.Fatalf("sin QDD Móvil activado no anota la sesión ni entrega nada: %+v", l)
	}
}

func TestConsolaTelefono_LaSesionLateYRecibeLaInstruccionUnaVez(t *testing.T) {
	emparejado(t)
	root := t.TempDir()
	consola.RegistrarEjecucion(root, consola.Ejecucion{Solicitud: "s1", Tipo: consola.EjecutarIA, Texto: "Revisa el build", Dispositivo: "d", Nombre: "Pixel"}, time.Now())
	l := latir(t, root, "a", false)
	if !l.Activo || len(l.Entregar) != 1 || l.Entregar[0].Solicitud != "s1" || !strings.HasPrefix(l.Entregar[0].Mensaje, "📱") || !strings.Contains(l.Entregar[0].Mensaje, "Revisa el build") {
		t.Fatalf("la recibe como mensaje del teléfono: %+v", l)
	}
	if !consola.HaySesion(root, time.Now()) {
		t.Fatal("y la sesión queda viva")
	}
	if l := latir(t, root, "b", false); len(l.Entregar) != 0 {
		t.Fatalf("otra sesión no la recibe de nuevo: %+v", l)
	}
	var salida, errores bytes.Buffer
	if codigo := telefonoFallo(root, "s1", "a", "prompt.submit falló", &salida, &errores); codigo != 0 {
		t.Fatalf("%d %s", codigo, errores.String())
	}
	if e, _ := consola.EjecucionDe(root, "s1", time.Now()); e.Estado != consola.EstadoFallo {
		t.Fatalf("si no se pudo entregar queda con falla: %+v", e)
	}
	latir(t, root, "a", true)
	latir(t, root, "b", true)
	if consola.HaySesion(root, time.Now()) {
		t.Fatal("al cerrarse ya no cuenta")
	}
	if codigo := telefonoLatir(root, "", false, &salida, &errores); codigo == 0 {
		t.Fatal("sin --sesion es un error")
	}
}

func TestConsolaPanel_EjecutarSoloConQDDMovilYPro(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if d := datosPanel(root, time.Now(), nil); d.Ejecutar != nil {
		t.Fatalf("sin QDD Móvil no hay ejecutar: %+v", d.Ejecutar)
	}
	emparejado(t)
	remoto.GuardarPlan(remoto.PlanGratis(), time.Now())
	if d := datosPanel(root, time.Now(), nil); d.Ejecutar != nil {
		t.Fatalf("en Gratis no hay ejecutar: %+v", d.Ejecutar)
	}
	remoto.GuardarPlan(remoto.Plan{Plan: "pro"}, time.Now())
	d := datosPanel(root, time.Now(), nil)
	if d.Ejecutar == nil || !d.Ejecutar.QDD || d.Ejecutar.IA || len(d.Ejecutar.Sugeridos) == 0 {
		t.Fatalf("con Pro: comandos qdd y, sin sesión, sin instrucciones: %+v", d.Ejecutar)
	}
	b, _ := json.Marshal(d)
	if !strings.Contains(string(b), `"ejecutar":{"qdd":true,"ia":false,"sugeridos":["audit"`) {
		t.Fatalf("el JSON del panel: %s", b)
	}
}
