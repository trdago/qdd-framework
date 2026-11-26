package cmd

import (
	"encoding/json"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/mejoras"
)

// Rule: Prohibido usar 'else'
// MEJ-020: lo que el panel leía con un proceso propio viaja en su JSON si lo pide: herramientas y si toca el ciclo de mejoras.

func panelJSON(t *testing.T, root string, con ...string) map[string]json.RawMessage {
	t.Helper()
	b, err := json.Marshal(datosPanel(root, time.Now(), con))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func proyectoParaPanel(t *testing.T) string {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".qdd", "herramientas"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPanel_LasHerramientasVanSoloSiSePiden_MEJ020(t *testing.T) {
	root := proyectoParaPanel(t)
	if _, esta := panelJSON(t, root)["herramientas"]; esta {
		t.Fatal("quien no las pide (el teléfono, un mod anterior) no las recibe")
	}
	if got := string(panelJSON(t, root, "herramientas")["herramientas"]); got != "[]" {
		t.Fatalf("sin herramientas, la lista va vacía: así el panel sabe que este qdd las trae: %q", got)
	}
	def := "id: bot\nnombre: Bot\ncomando: [\"bot\"]\nacciones:\n  - id: ver\n    nombre: Ver\n    args: [\"ver\"]\n"
	if err := os.WriteFile(filepath.Join(root, ".qdd", "herramientas", "bot.yaml"), []byte(def), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := string(panelJSON(t, root, "herramientas")["herramientas"]); !strings.Contains(got, `"id":"bot"`) {
		t.Fatalf("trae la misma lista que 'herramienta listar': %s", got)
	}
}

func TestPanel_DiceSiTocaElCicloDeMejorasSoloSiSePide_MEJ020(t *testing.T) {
	root := proyectoParaPanel(t)
	if _, esta := panelJSON(t, root)["mejoras_toca"]; esta {
		t.Fatal("quien no lo pide no lo recibe")
	}
	if got := string(panelJSON(t, root, "mejoras")["mejoras_toca"]); got != "true" {
		t.Fatalf("sin un ciclo anterior, toca: %q", got)
	}
	if _, err := mejoras.Ciclo(root, mejoras.Fuentes{}, time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := string(panelJSON(t, root, "mejoras")["mejoras_toca"]); got != "false" {
		t.Fatalf("recién corrido, no toca, y el campo sigue ahí: %q", got)
	}
}

func TestPanel_ElFlujoVigilaLasHerramientas_MEJ020(t *testing.T) {
	root := proyectoParaPanel(t)
	carpeta := filepath.Join(root, ".qdd", "herramientas")
	for _, c := range carpetasDelFlujo(root) {
		if c == carpeta {
			return
		}
	}
	t.Fatalf("una herramienta guardada se ve al instante: %v", carpetasDelFlujo(root))
}

// RFC 0012: los flujos viajan en la línea del panel si se piden; vacía sale [] para que el panel sepa que este qdd los trae.
func TestPanel_LosFlujosVanSoloSiSePiden(t *testing.T) {
	root := proyectoParaPanel(t)
	if _, esta := panelJSON(t, root)["flujos"]; esta {
		t.Fatal("sin pedirlos no van")
	}
	if got := string(panelJSON(t, root, "flujos")["flujos"]); got != "[]" {
		t.Fatalf("sin flujos, la lista va vacía: %q", got)
	}
	f := flujo.Flujo{Formato: flujo.Formato, ID: "pago", Nombre: "Pago", Nodos: []flujo.Nodo{{ID: "a", Tipo: flujo.TipoPaso, Nombre: "A"}}}
	if err := flujo.Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	if got := string(panelJSON(t, root, "flujos")["flujos"]); !strings.Contains(got, `"id":"pago"`) || !strings.Contains(got, `"huella"`) {
		t.Fatalf("lista: %q", got)
	}
}

func TestPanel_ElFlujoVigilaLosFlujos(t *testing.T) {
	root := proyectoParaPanel(t)
	carpeta := filepath.Join(root, flujo.Carpeta)
	for _, c := range carpetasDelFlujo(root) {
		if c == carpeta {
			return
		}
	}
	t.Fatalf("un flujo guardado se ve al instante: %v", carpetasDelFlujo(root))
}
