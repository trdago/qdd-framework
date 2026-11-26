package remoto

import (
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
// Pedido de sfrc (2026-11-03): un ⏳ «de la persona» del estado llega al teléfono como pregunta de respuesta libre, y
// lo que la persona responde le llega a la IA del proyecto como una instrucción del teléfono.

const estadoConEspera = `formato: 1
etapas: [Código, PR]
frentes:
  - {id: auth, nombre: Auth, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: "¿dejamos fuera el segundo factor?", de_quien: la persona}, PR: {estado: "—"}}}
`

func TestResponder_UnEsperaDeLaPersonaSePublicaYSuRespuestaLlegaALaIA(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	if err := os.WriteFile(filepath.Join(root, estado.Ruta), []byte(estadoConEspera), 0o644); err != nil {
		t.Fatal(err)
	}
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	if err := consola.LatirSesion(root, "s1", time.Now()); err != nil {
		t.Fatal(err)
	}
	e := Armar("v", registro.Registro{Proyectos: []registro.Proyecto{{Ruta: root, Clave: clave}}}, time.Now(), 0)
	var id string
	for _, a := range e.Proyectos[0].Panel.Acciones {
		if a.Tipo == consola.TipoPregunta && a.Origen == consola.OrigenEstado {
			id = a.ID
		}
	}
	if id == "" {
		t.Fatalf("el teléfono recibe el ⏳ como pregunta: %+v", e.Proyectos[0].Panel.Acciones)
	}

	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	mandarRespuesta(tr, respuestaFirmada("3b6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, id, `{"q1": {"otra": "sí, fuera"}}`, "d1", ahoraEn()))
	for _, want := range []string{EstadoRecibida, EstadoHecha} {
		if got := estadoDe(t, tr, 5*time.Second); got.Estado != want || (want == EstadoHecha && !strings.Contains(got.Resultado, "IA")) {
			t.Fatalf("esperaba %s, y al final qué quedó en el estado: %+v", want, got)
		}
	}
	ejecuciones := consola.EjecucionesA(root, time.Now())
	if len(ejecuciones) != 1 || ejecuciones[0].Tipo != consola.EjecutarIA || !strings.Contains(ejecuciones[0].Texto, "sí, fuera") || ejecuciones[0].Nombre != "Pixel de d1" {
		t.Fatalf("con una sesión, la IA también se entera, del teléfono que respondió: %+v", ejecuciones)
	}
}

// FND-141: el teléfono decía «entregada» al dejar la respuesta en la cola, antes de que una sesión la tomara. Desde los
// botones (2026-11-07) la respuesta queda en el estado: está hecha aunque ninguna sesión la tome.
func TestResponder_SinSesionLaRespuestaQuedaEnElEstado_FND141(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	if err := os.WriteFile(filepath.Join(root, estado.Ruta), []byte(estadoConEspera), 0o644); err != nil {
		t.Fatal(err)
	}
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	var id string
	for _, p := range consola.PreguntasA(root, time.Now()) {
		id = p.ID
	}
	_, tr, _ := bucleConFirma(t)
	tr.eventos <- telefono("d1", "")
	tr.esperar(t, time.Second, "estado")
	mandarRespuesta(tr, respuestaFirmada("4b6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50", clave, id, `{"q1": {"opciones": ["ya_no_va"]}}`, "d1", ahoraEn()))
	estadoDe(t, tr, 5*time.Second)
	if got := estadoDe(t, tr, 5*time.Second); got.Estado != EstadoHecha || !strings.Contains(got.Resultado, "ya no va") {
		t.Fatalf("hecha, y dice qué quedó: %+v", got)
	}
	e, err := estado.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if c := e.Frentes[0].Celdas["Código"]; c.Estado != estado.NoAplica {
		t.Fatalf("la etapa quedó en —: %+v", c)
	}
	if x := consola.EjecucionesA(root, time.Now()); len(x) != 0 {
		t.Fatalf("sin sesión no queda una instrucción que vaya a vencer: %+v", x)
	}
}

// FND-124 decía que sin una sesión nadie podía resolver un ⏳ y no se publicaba. Desde los botones (2026-11-07) la
// respuesta queda en el estado, así que se publica siempre, marcada sin_sesion.
func TestEstado_SinSesionUnEsperaSePublica_FND124(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "app", true)
	if err := os.WriteFile(filepath.Join(root, estado.Ruta), []byte(estadoConEspera), 0o644); err != nil {
		t.Fatal(err)
	}
	clave, _ := registro.ClaveDe(root)
	p := registro.Proyecto{Ruta: root, Clave: clave}
	publicadas := func(now time.Time) int {
		n := 0
		for _, a := range PanelDe("v", p, now).Panel.Acciones {
			if a.Tipo == consola.TipoPregunta && a.Origen == consola.OrigenEstado && a.SinSesion {
				n++
			}
		}
		return n
	}
	if n := publicadas(time.Now()); n != 1 {
		t.Fatalf("sin sesión también se publica: %d", n)
	}
}
