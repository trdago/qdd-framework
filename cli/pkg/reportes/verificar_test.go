package reportes

import (
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/incidentes"
)

// Rule: Prohibido usar 'else'
// MEJ-087 (REP-503, propuesta de la persona): quien reporta valida. Un bug dice desde el inicio cómo lo verificará quien
// lo reporta, y la verificación vuelve como un reporte que nombra al original. Todo viaja en el texto firmado: el
// relay no cambia.

func nuevoReporte(t *testing.T, d Datos) (Reporte, error) {
	t.Helper()
	d.Version, d.SO = "2.10.0-rc.96", "linux/amd64"
	return Nuevo(d, incidentes.Contexto{}, nil)
}

func TestVerifico_UnBugDiceComoLoVerificaQuienReporta_MEJ087(t *testing.T) {
	if _, err := nuevoReporte(t, Datos{Titulo: "Se cae el panel", Detalle: "al abrir"}); err == nil || !strings.Contains(err.Error(), "--verifico") {
		t.Fatalf("un bug sin «cómo lo verifico» no sale: %v", err)
	}
	r, err := nuevoReporte(t, Datos{Titulo: "Se cae el panel", Detalle: "al abrir", Verifico: "abro el panel con 3 proyectos y no se cae"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(r.Detalle, "al abrir") || VerificoDe(r.Detalle) != "abro el panel con 3 proyectos y no se cae" {
		t.Fatalf("va en el detalle firmado y se lee de vuelta: %q", r.Detalle)
	}
	for _, tipo := range []string{"mejora", "pregunta"} {
		if _, err := nuevoReporte(t, Datos{Titulo: "Idea", Tipo: tipo}); err != nil {
			t.Fatalf("%s no lo exige: %v", tipo, err)
		}
	}
}

func TestVerifique_LaVerificacionNombraAlReporteYTraeLaEvidencia_MEJ087(t *testing.T) {
	r, err := nuevoReporte(t, Datos{VerificaDe: "REP-500", Evidencia: "mandé 3 mensajes a la sesión libre: llegaron en 2 s"})
	if err != nil {
		t.Fatal(err)
	}
	if r.Titulo != "Verificado REP-500 en 2.10.0-rc.96" || r.Tipo != "mejora" || r.Detalle != "mandé 3 mensajes a la sesión libre: llegaron en 2 s" {
		t.Fatalf("verificado: %+v", r)
	}
	r, err = nuevoReporte(t, Datos{VerificaDe: "REP-500", Evidencia: "sigue en Entregando", SigueFallando: true})
	if err != nil || r.Titulo != "Sigue fallando REP-500 en 2.10.0-rc.96" || r.Tipo != "bug" {
		t.Fatalf("sigue fallando reabre el mismo, sin pedir otro «cómo lo verifico»: %+v %v", r, err)
	}
	if _, err := nuevoReporte(t, Datos{VerificaDe: "REP-500"}); err == nil || !strings.Contains(err.Error(), "--evidencia") {
		t.Fatalf("se cierra solo con evidencia: %v", err)
	}
	if _, err := nuevoReporte(t, Datos{VerificaDe: "500", Evidencia: "x"}); err == nil {
		t.Fatal("nombra un REP-N")
	}
}
