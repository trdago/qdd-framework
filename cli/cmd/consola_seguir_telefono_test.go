package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// MEJ-020 (#108): con QDD Móvil, cada sesión lanzaba «telefono latir» cada 5 s, 720 procesos por hora en reposo. Late el
// flujo que ya está abierto, y lo mandado desde el teléfono viaja por ahí.

func instruccionDelTelefono(t *testing.T, root string, now time.Time) {
	t.Helper()
	e := consola.Ejecucion{Solicitud: "s1", Tipo: consola.EjecutarIA, Texto: "Revisa el build", Nombre: "Pixel"}
	if _, err := consola.RegistrarEjecucion(root, e, now); err != nil {
		t.Fatal(err)
	}
}

func TestSesionDelFlujo_LaLeeDeLoQuePideElPanel(t *testing.T) {
	if got := sesionDelFlujo([]string{"atender", "telefono:s-1", "flujos"}); got != "s-1" {
		t.Fatalf("%q", got)
	}
	for _, con := range [][]string{nil, {"atender"}, {"telefono:"}} {
		if got := sesionDelFlujo(con); got != "" {
			t.Fatalf("%v: %q", con, got)
		}
	}
}

// FND-191: sin QDD Móvil el flujo no anotaba la sesión, y «qdd baul usar» decía que no latía y nunca dejaba el pedido
// 🔑 (Windows, rc.54). La sesión viva la usa el baúl, no solo el teléfono: se anota siempre; entregar, solo con teléfono.
func TestLatidoDelFlujo_SinQDDMovilAnotaLaSesionPeroNoEmite_FND191(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	emitir := func([]byte) error {
		t.Fatal("sin QDD Móvil no hay nada que entregar")
		return nil
	}
	now := time.Now()
	if err := latidoDelFlujo(root, "s", now, emitir); err != nil {
		t.Fatal(err)
	}
	if !consola.SesionLate(root, "s", now) {
		t.Fatal("sin QDD Móvil la sesión igual late: el baúl pide el permiso a una sesión que late")
	}
}

func TestLatidoDelFlujo_AnotaLaSesionYEntregaUnaSolaVez(t *testing.T) {
	emparejado(t)
	root := t.TempDir()
	now := time.Now()
	instruccionDelTelefono(t, root, now)
	var lineas [][]byte
	emitir := func(l []byte) error {
		lineas = append(lineas, l)
		return nil
	}
	for range 2 {
		if err := latidoDelFlujo(root, "s", now, emitir); err != nil {
			t.Fatal(err)
		}
	}
	if !consola.HaySesion(root, now) {
		t.Fatal("el teléfono debe saber que hay una sesión abierta")
	}
	if len(lineas) != 1 {
		t.Fatalf("se entrega una sola vez: %q", lineas)
	}
	var l struct {
		Entregar []entrega `json:"telefono_entregar"`
	}
	if err := json.Unmarshal(lineas[0], &l); err != nil {
		t.Fatal(err)
	}
	if len(l.Entregar) != 1 || l.Entregar[0].Solicitud != "s1" || !strings.Contains(l.Entregar[0].Mensaje, "Revisa el build") {
		t.Fatalf("%s", lineas[0])
	}
}

func TestLatidoDelFlujo_SiNoLlegaAlPanelQuedaSinEntregar(t *testing.T) {
	emparejado(t)
	root := t.TempDir()
	now := time.Now()
	instruccionDelTelefono(t, root, now)
	roto := errors.New("nadie lee")
	if err := latidoDelFlujo(root, "s", now, func([]byte) error { return roto }); !errors.Is(err, roto) {
		t.Fatalf("%v", err)
	}
	es := consola.EjecucionesA(root, now)
	if len(es) != 1 || es[0].Estado != consola.EstadoFallo {
		t.Fatalf("no se pierde en silencio: %+v", es)
	}
}

func TestLecturaPanel_DiceSiElFlujoLate(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	conLate, err := lecturaPanel(root, []string{"telefono:s"})()
	if err != nil {
		t.Fatal(err)
	}
	sinLate, err := lecturaPanel(root, nil)()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(conLate), `"telefono_en_flujo":true`) || strings.Contains(string(sinLate), "telefono_en_flujo") {
		t.Fatalf("%s\n%s", conLate, sinLate)
	}
}

func TestLatirElFlujo_LateAlAbrirYTerminaConElFlujo(t *testing.T) {
	emparejado(t)
	root := t.TempDir()
	instruccionDelTelefono(t, root, time.Now())
	ctx, cancelar := context.WithCancel(context.Background())
	lineas := make(chan []byte, 1)
	termino := make(chan struct{})
	go func() {
		latirElFlujo(ctx, root, "s", func(l []byte) error {
			lineas <- l
			return nil
		})
		close(termino)
	}()
	select {
	case <-lineas:
	case <-time.After(5 * time.Second):
		t.Fatal("no esperó el primer tic: late al abrir")
	}
	cancelar()
	select {
	case <-termino:
	case <-time.After(5 * time.Second):
		t.Fatal("debe terminar con el flujo")
	}
}

func TestLatirElFlujo_SinSesionNoLate(t *testing.T) {
	emparejado(t)
	root := t.TempDir()
	latirElFlujo(context.Background(), root, "", func([]byte) error { return nil })
	if consola.HaySesion(root, time.Now()) {
		t.Fatal("un panel que no pidió el latido no anota nada")
	}
}
