package consola

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// conOpenCode es una consola en una máquina con OpenCode: su configuración vive en dir.
func conOpenCode(t *testing.T, version string, f *claudeFalso, dir string) Consola {
	t.Helper()
	c := nuevaConsola(t, version, f)
	c.BuscarOpenCode = func() (string, bool) { return dir, true }
	return c
}

func pluginsTUI(t *testing.T, dir string) []json.RawMessage {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "tui.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Plugin []json.RawMessage `json:"plugin"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("tui.json ilegible: %v\n%s", err, data)
	}
	return cfg.Plugin
}

// RFC 0006 §3.10: activar deja el plugin con la versión de la CLI y lo registra una sola vez en tui.json, sin tocar lo demás.
func TestActivar_OpenCode_InstalaYRegistraUnaVez_CCC_C23(t *testing.T) {
	dir := t.TempDir()
	escribirArchivo(t, filepath.Join(dir, "tui.json"), `{"$schema": "https://opencode.ai/tui.json", "theme": "system", "plugin": ["@acme/otro@1.0.0"]}`)
	c := conOpenCode(t, "2.5.0", &claudeFalso{}, dir)
	for i := 0; i < 2; i++ {
		if _, err := c.Activar(false); err != nil {
			t.Fatal(err)
		}
	}
	plugin, err := os.ReadFile(c.RutaPluginOpenCode())
	if err != nil || !strings.Contains(string(plugin), `const VERSION = "2.5.0"`) || !strings.Contains(string(plugin), "sidebar_content") {
		t.Fatalf("el plugin lleva la versión de la CLI y va en la barra lateral: %v", err)
	}
	entradas := pluginsTUI(t, dir)
	spec, _ := json.Marshal(c.RutaPluginOpenCode())
	if len(entradas) != 2 || string(entradas[0]) != `"@acme/otro@1.0.0"` || string(entradas[1]) != string(spec) {
		t.Fatalf("se agrega una sola vez y se conserva lo que había: %s", entradas)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "tui.json"))
	if !strings.Contains(string(data), `"theme": "system"`) {
		t.Fatalf("lo demás de tui.json no se toca:\n%s", data)
	}
	if _, err := os.Stat(filepath.Join(dir, "tui.json.qdd.bak")); err != nil {
		t.Fatal("antes de tocar tui.json por primera vez queda un respaldo")
	}
	if d := c.Diagnosticar(); len(d) != 2 || !d[1].OK {
		t.Fatalf("doctor ve el panel de OpenCode al día: %+v", d)
	}
}

func TestActivar_OpenCode_SinTUIJSONLoCrea(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "opencode")
	c := conOpenCode(t, "2.5.0", &claudeFalso{}, dir)
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	if got := pluginsTUI(t, dir); len(got) != 1 {
		t.Fatalf("tui.json nuevo con la entrada de qdd: %s", got)
	}
}

func TestActivar_OpenCode_TUIJSONInvalidoNoSeToca(t *testing.T) {
	dir := t.TempDir()
	invalido := "{ // comentario\n \"theme\": \"system\" }"
	escribirArchivo(t, filepath.Join(dir, "tui.json"), invalido)
	c := conOpenCode(t, "2.5.0", &claudeFalso{}, dir)
	hecho, err := c.Activar(false)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "tui.json"))
	if string(data) != invalido || !strings.Contains(strings.Join(hecho, "\n"), "no se tocó") {
		t.Fatalf("un tui.json que no es JSON no se toca y se avisa: %q\n%s", hecho, data)
	}
}

// §3.10: actualizar y doctor --fix mantienen el plugin en la versión de la CLI; desactivar lo quita sin tocar lo demás.
func TestOpenCode_SeSincronizaConLaCLIYDesactivarLoQuita(t *testing.T) {
	dir := t.TempDir()
	escribirArchivo(t, filepath.Join(dir, "tui.json"), `{"plugin": ["@acme/otro@1.0.0"]}`)
	f := &claudeFalso{}
	vieja := conOpenCode(t, "2.4.0", f, dir)
	if _, err := vieja.Activar(false); err != nil {
		t.Fatal(err)
	}
	nueva := vieja
	nueva.Version = "2.5.0"
	if d := nueva.Diagnosticar(); d[1].OK || !strings.Contains(d[1].Detalle, "2.4.0") {
		t.Fatalf("doctor detecta el plugin de OpenCode de otra versión: %+v", d)
	}
	if _, err := nueva.Reparar(); err != nil {
		t.Fatal(err)
	}
	if d := nueva.Diagnosticar(); !d[1].OK {
		t.Fatalf("reparar lo deja en la versión de la CLI: %+v", d)
	}
	if _, err := nueva.Desactivar(); err != nil {
		t.Fatal(err)
	}
	if got := pluginsTUI(t, dir); len(got) != 1 || string(got[0]) != `"@acme/otro@1.0.0"` {
		t.Fatalf("desactivar quita solo la entrada de qdd: %s", got)
	}
	if _, err := os.Stat(nueva.RutaPluginOpenCode()); err == nil {
		t.Fatal("desactivar borra el plugin")
	}
}

func TestActivar_SinOpenCodeNoLoToca(t *testing.T) {
	c := nuevaConsola(t, "2.5.0", &claudeFalso{})
	hecho, err := c.Activar(false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(hecho, "\n"), "OpenCode") {
		t.Fatalf("sin OpenCode no se menciona ni se instala: %v", hecho)
	}
	if _, err := os.Stat(c.RutaPluginOpenCode()); err == nil {
		t.Fatal("sin OpenCode no se escribe el plugin")
	}
}
