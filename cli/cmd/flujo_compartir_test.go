package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 1: qdd flujo compartir escribe el archivo de solo lectura, por defecto en ./tmp del proyecto.

func TestFlujo_CompartirEscribeElArchivoEnTmp(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir()) // el baúl de prueba está vacío: nunca se lee el llavero real
	root := proyectoConFlujo(t)
	salida := correrFlujo(t, flujoCompartirCmd, map[string]string{"para": "Ana Pérez"}, "pago")
	archivo := filepath.Join(root, "tmp", "flujo-pago-ana-perez.html")
	datos, err := os.ReadFile(archivo)
	if err != nil {
		t.Fatalf("el archivo queda en ./tmp: %v (%s)", err, salida)
	}
	if !strings.Contains(salida, archivo) || !strings.Contains(salida, "solo lectura") {
		t.Fatalf("dice dónde quedó y que es de solo lectura: %q", salida)
	}
	if !strings.Contains(string(datos), "Cómo se cobra") || strings.Contains(string(datos), `"rc"`) {
		t.Fatal("lleva el flujo, pero no la tarea del tablero")
	}
	otro := filepath.Join(t.TempDir(), "x.html")
	correrFlujo(t, flujoCompartirCmd, map[string]string{"salida": otro}, "pago")
	if _, err := os.Stat(otro); err != nil {
		t.Fatalf("--salida elige el archivo: %v", err)
	}
}

// contrato: flujo#55d83602
func TestFlujo_CompartirNoLlevaLaRutaDelProyectoNiLoInterno(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir()) // el baúl de prueba está vacío: nunca se lee el llavero real
	root := filepath.Join(t.TempDir(), "proyecto-secreto-xyz")
	if err := estado.Init(root, []string{"Código"}, ahoraDePruebaFlujo()); err != nil {
		t.Fatal(err)
	}
	estadoDir = root
	t.Cleanup(func() { estadoDir = "" })
	f := flujo.Flujo{Formato: flujo.Formato, ID: "pago", Nombre: "Cómo se cobra",
		Descripcion: "el código está en " + filepath.Join(root, "src", "cobro"),
		Comentarios: []flujo.Comentario{{De: flujo.DePersona, Texto: "coment-flujo", Cuando: "x", Pregunta: true}},
		Historias:   []flujo.Historia{{ID: "h1", Quiero: "cobrar", Origen: "origen-secreto", Tarea: "tarea-secreta"}},
		Nodos: []flujo.Nodo{
			{ID: "a", Tipo: flujo.TipoPaso, Nombre: "Publica", Tarea: "rc", Siguen: []string{"b"},
				Items: []flujo.Item{
					{ID: "i1", Tipo: flujo.ItemPrueba, Texto: "prueba-secreta", Estado: "pasa", Evidencia: "ev-secreta"},
					{ID: "i2", Tipo: flujo.ItemNota, Texto: "nota-interna"},
				},
				Comentarios: []flujo.Comentario{{De: flujo.DeIA, Texto: "coment-nodo", Cuando: "x"}}},
			{ID: "b", Tipo: flujo.TipoResultado, Nombre: "Fin"},
		},
	}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	salida := filepath.Join(t.TempDir(), "cliente.html")
	correrFlujo(t, flujoCompartirCmd, map[string]string{"salida": salida}, "pago")
	datos, err := os.ReadFile(salida)
	if err != nil {
		t.Fatal(err)
	}
	html := string(datos)
	if !strings.Contains(html, "Cómo se cobra") {
		t.Fatal("el archivo lleva el flujo")
	}
	prohibidos := []string{"proyecto-secreto-xyz", "ev-secreta", "prueba-secreta", "nota-interna", "coment-flujo", "coment-nodo",
		"origen-secreto", "tarea-secreta", `"rc"`}
	for _, prohibido := range prohibidos {
		if strings.Contains(html, prohibido) {
			t.Errorf("el archivo para el cliente lleva %q", prohibido)
		}
	}
}
