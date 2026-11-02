package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// Los blueprints a un clic en el panel: el inventario sale sin red ni IA, y qdd vuelve a calcular sus acciones antes
// de ejecutar una.

func TestConsolaBlueprints_EntregaElInventarioYSusAcciones(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, ".qdd"), 0o755)
	var out bytes.Buffer
	if code := consolaBlueprints(root, true, &out); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	var d consola.DatosBlueprints
	if err := json.Unmarshal(out.Bytes(), &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Acciones) == 0 || d.Acciones[0].ID != "blueprint:sync" || len(d.Inventario.Problemas) == 0 {
		t.Fatalf("sin repositorio de la organización: dice cómo tenerlo y ofrece traer el catálogo: %+v", d)
	}
	out.Reset()
	consolaBlueprints(root, false, &out)
	if !strings.Contains(out.String(), "instalados") {
		t.Fatalf("sin --json, un resumen: %s", out.String())
	}
}

func TestConsolaAccion_UnaDeBlueprintsSeBuscaEnElInventario(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := t.TempDir()
	if _, err := consola.BuscarAccion(accionesPara(root, "blueprint:sync"), "blueprint:sync"); err != nil {
		t.Fatalf("qdd la vuelve a calcular: %v", err)
	}
	var out, errs bytes.Buffer
	if code := consolaAccion(root, "blueprint:instalar:nada", true, "", &out, &errs); code == 0 {
		t.Fatal("una que el inventario no ofrece se rechaza sin ejecutar nada")
	}
}
