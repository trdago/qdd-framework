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
		if got := estadoDe(t, tr, 5*time.Second); got.Estado != want {
			t.Fatalf("esperaba %s: %+v", want, got)
		}
	}
	ejecuciones := consola.EjecucionesA(root, time.Now())
	if len(ejecuciones) != 1 || ejecuciones[0].Tipo != consola.EjecutarIA || !strings.Contains(ejecuciones[0].Texto, "sí, fuera") || ejecuciones[0].Nombre != "Pixel de d1" {
		t.Fatalf("la respuesta queda para la IA, del teléfono que respondió: %+v", ejecuciones)
	}
}

// FND-124: sin una sesión de Claude Code abierta nadie puede resolver un ⏳, así que no se publica al teléfono.
func TestEstado_SinSesionUnEsperaNoSePublica_FND124(t *testing.T) {
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
			if a.Tipo == consola.TipoPregunta && a.Origen == consola.OrigenEstado {
				n++
			}
		}
		return n
	}
	now := time.Now()
	if n := publicadas(now); n != 0 {
		t.Fatalf("sin sesión no se publica: %d", n)
	}
	if err := consola.LatirSesion(root, "s1", now); err != nil {
		t.Fatal(err)
	}
	if n := publicadas(now); n != 1 {
		t.Fatalf("con sesión sí: %d", n)
	}
	if n := publicadas(now.Add(consola.VidaSesion + time.Second)); n != 0 {
		t.Fatalf("cuando la sesión deja de latir, sale: %d", n)
	}
}
