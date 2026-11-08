package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// RFC 0025 (decisión de la persona, 2026-11-08): revisiones y versiones con nombre, y plantillas por dominio.

func TestFlujo_VersionCompararYVolver(t *testing.T) {
	proyectoConFlujo(t)
	if s := correrFlujo(t, flujoVersionCmd, nil, "pago", "acordada con Ana"); !strings.Contains(s, "v1") || !strings.Contains(s, "r1") {
		t.Fatalf("versión 1 sobre la revisión 1 (crear es la 1): %q", s)
	}
	correrFlujo(t, flujoAgregarCmd, map[string]string{"nombre": "Cobra", "de": "persona"}, "pago", "a")
	if s := correrFlujo(t, flujoCompararCmd, nil, "pago", "v1", "actual"); !strings.Contains(s, "+ cobra") {
		t.Fatalf("de v1 a hoy se agregó cobra: %q", s)
	}
	var vs []flujo.Version
	if err := json.Unmarshal([]byte(correrFlujo(t, flujoVersionesCmd, map[string]string{"json": "true"}, "pago")), &vs); err != nil || len(vs) != 1 {
		t.Fatalf("versiones: %+v %v", vs, err)
	}
	correrFlujo(t, flujoVolverCmd, map[string]string{"de": "persona"}, "pago", "v1")
	if d := verFlujo(t, "pago", nil); len(d.Nodos) != 2 || d.Revision != 3 {
		t.Fatalf("volver deja los nodos de v1 en una revisión nueva: %+v", d)
	}
	if s := correrFlujo(t, flujoHistorialCmd, nil, "pago"); !strings.Contains(s, "r3") || !strings.Contains(s, "persona") || !strings.Contains(s, "vuelve a v1") {
		t.Fatalf("historial: %q", s)
	}
}

func TestFlujo_PlantillasYNivelesEnElDibujo(t *testing.T) {
	proyectoConFlujo(t)
	if s := correrFlujo(t, flujoPlantillasCmd, nil); !strings.Contains(s, "web") || !strings.Contains(s, "Páginas") {
		t.Fatalf("plantillas: %q", s)
	}
	d := verFlujo(t, "pago", nil)
	if d.Plantilla != flujo.PlantillaProceso || d.NombresNivel != [3]string{"Hitos", "Pasos", "Desvíos"} {
		t.Fatalf("el dibujo trae la plantilla y el nombre de sus niveles: %+v", d)
	}
}
