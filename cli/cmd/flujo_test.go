package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

const flujoDePrueba = `formato: 1
id: pago
nombre: Cómo se cobra
nodos:
  - {id: a, tipo: paso, nombre: Publica, tarea: rc, siguen: [b]}
  - {id: b, tipo: resultado, nombre: Fin}
`

func proyectoConFlujo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := estado.Init(root, []string{"Código"}, ahoraDePruebaFlujo()); err != nil {
		t.Fatal(err)
	}
	estadoDir = root
	t.Cleanup(func() { estadoDir, flujoJSON, flujoAncho = "", false, anchoFlujoPorDefecto })
	archivo := filepath.Join(root, "f.yaml")
	if err := os.WriteFile(archivo, []byte(flujoDePrueba), 0o644); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	flujoCrearCmd.SetOut(&salida)
	flujoCrearCmd.Run(flujoCrearCmd, []string{archivo})
	if !strings.Contains(salida.String(), "Flujo 'pago' guardado") {
		t.Fatalf("crear: %q", salida.String())
	}
	return root
}

func TestFlujo_CrearYVerConElAnchoQueSePide(t *testing.T) {
	proyectoConFlujo(t)
	var salida bytes.Buffer
	flujoVerCmd.SetOut(&salida)
	flujoJSON, flujoAncho = true, 40
	flujoVerCmd.Run(flujoVerCmd, []string{"pago"})
	var d flujoDibujado
	if err := json.Unmarshal(salida.Bytes(), &d); err != nil {
		t.Fatalf("%v: %q", err, salida.String())
	}
	dibujo := strings.Join(d.Lineas, "\n")
	if d.ID != "pago" || d.Nombre != "Cómo se cobra" || !strings.Contains(dibujo, "Publica") || !strings.Contains(dibujo, "( Fin )") {
		t.Fatalf("dibujo: %+v", d)
	}
	if !strings.Contains(dibujo, "tarea rc no está") {
		t.Fatalf("una tarea que no existe se dice:\n%s", dibujo)
	}
}

func TestFlujo_UnaTareaDelTableroMuestraSusSimbolos(t *testing.T) {
	root := proyectoConFlujo(t)
	if err := estado.Set(root, estado.Cambio{Frente: "rc", Nombre: "RC", Etapa: "Código", Estado: "ok", Evidencia: "abc", AunqueExista: true}, ahoraDePruebaFlujo()); err != nil {
		t.Fatal(err)
	}
	if got := estadoDeTarea(root)("rc"); got != "✅" {
		t.Fatalf("símbolos: %q", got)
	}
}

func TestFlujo_ListaDiceCuandoNoHay(t *testing.T) {
	estadoDir = t.TempDir()
	t.Cleanup(func() { estadoDir = "" })
	var salida bytes.Buffer
	flujoListaCmd.SetOut(&salida)
	flujoListaCmd.Run(flujoListaCmd, nil)
	if !strings.Contains(salida.String(), "no tiene flujos") {
		t.Fatalf("lista: %q", salida.String())
	}
}

func ahoraDePruebaFlujo() time.Time { return time.Date(2026, 11, 5, 21, 0, 0, 0, time.UTC) }
