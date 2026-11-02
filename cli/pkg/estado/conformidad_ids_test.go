package estado

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Los escenarios EPT-I01…I04 de estado-por-tareas 2.2.0, con los datos exactos del blueprint.

func contenidoDelArchivo(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, Ruta))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestConformidad_NombrarNoCrea_EPT_I01(t *testing.T) {
	root := proyectoNotificador(t)
	antes := contenidoDelArchivo(t, root)
	intentos := map[string]error{
		"celda":       Set(root, Cambio{Frente: "segmento-barra", Etapa: "Código", Estado: "ok", Evidencia: "commit abc1"}, ahora),
		"cerrar":      Cerrar(root, "segmento-barra", "listo", false, ahora),
		"en curso":    Activar(root, "segmento-barra", "claude-code", ahora),
		"dependencia": GuardarTarea(root, CambioTarea{ID: "barra-estado", DependeDe: []string{"segmento-barra"}, CambiaDependencias: true}, ahora),
	}
	for como, err := range intentos {
		if err == nil {
			t.Fatalf("%s de una tarea que no existe es un error", como)
		}
	}
	if contenidoDelArchivo(t, root) != antes || cantidadDeTareas(t, root) != 3 {
		t.Fatal("el archivo no cambia y no aparece ninguna tarea nueva")
	}
}

func TestConformidad_CrearDaElID_EPT_I02(t *testing.T) {
	root := proyectoNotificador(t)
	id, err := Nueva(root, CambioNueva{Nombre: "Ticket de MP en logs de error"}, ahora)
	if err != nil || id != "ticket-mp-logs-error" {
		t.Fatalf("crear por el nombre da el id: %q %v", id, err)
	}
	e, _ := Load(root)
	f := e.Frentes[e.buscar(id)]
	if f.Nombre != "Ticket de MP en logs de error" || f.Estado != EstadoAbierto {
		t.Fatalf("queda abierta con ese nombre: %+v", f)
	}
	for etapa, c := range f.Celdas {
		if c.Estado != "·" {
			t.Fatalf("ninguna etapa con avance: %s está en %q", etapa, c.Estado)
		}
	}
	otra, err := Nueva(root, CambioNueva{Nombre: "Ticket de MP en logs de error", AunqueExista: true}, ahora)
	if err != nil || otra != "ticket-mp-logs-error-2" {
		t.Fatalf("otra con el mismo nombre, confirmando, lleva -2: %q %v", otra, err)
	}
	if err := Set(root, Cambio{Frente: "Ticket_MP", Nombre: "Ticket de MP", Etapa: "Código", Estado: "ok", Evidencia: "commit abc1"}, ahora); err == nil {
		t.Fatal("crear con el id a mano «Ticket_MP» se rechaza")
	}
}

func TestConformidad_UnaReferenciaEquivocadaSugiereLaQueEra_EPT_I03(t *testing.T) {
	root := proyectoNotificador(t)
	esperadas := []struct{ escrito, sugiere string }{
		{"segmento-barra", "barra-estado"},
		{"qdd-estado", "estado-por-frente"},
		{"barra-estdo", "barra-estado"},
	}
	for _, c := range esperadas {
		err := Set(root, Cambio{Frente: c.escrito, Etapa: "Código", Estado: "ok", Evidencia: "commit abc1"}, ahora)
		if err == nil || !strings.Contains(err.Error(), "¿quisiste decir "+c.sugiere) {
			t.Fatalf("%s sugiere primero %s: %v", c.escrito, c.sugiere, err)
		}
	}
	err := Set(root, Cambio{Frente: "xyz", Etapa: "Código", Estado: "ok", Evidencia: "commit abc1"}, ahora)
	if err == nil || !strings.Contains(err.Error(), "abiertas: barra-estado, estado-por-frente, claves-gemini") {
		t.Fatalf("con «xyz» lista las abiertas: %v", err)
	}
}

func TestConformidad_SinDuplicados_EPT_I04(t *testing.T) {
	root := proyectoNotificador(t)
	_, err := Nueva(root, CambioNueva{Nombre: "Quitar claves de Gemini"}, ahora)
	if err == nil || !strings.Contains(err.Error(), "claves-gemini") {
		t.Fatalf("no se crea y el error nombra claves-gemini: %v", err)
	}
	if cantidadDeTareas(t, root) != 3 {
		t.Fatal("no se creó")
	}
	if _, err := Nueva(root, CambioNueva{Nombre: "Quitar claves de Gemini", AunqueExista: true}, ahora); err != nil {
		t.Fatalf("confirmando que es otra, se crea: %v", err)
	}
}
