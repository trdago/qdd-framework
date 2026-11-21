package consola

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0014, fase C: activar, actualizar y doctor --fix dejan la extensión del panel en Antigravity y VS Code en la
// versión de la CLI. Nunca ejecutan el IDE: escriben su carpeta de extensiones y su extensions.json, sin tocar las
// entradas ajenas.

const entradaAjena = `{"identifier":{"id":"golang.go","uuid":"u-1"},"version":"0.50.0","location":{"$mid":1,"path":"/x/golang.go-0.50.0","scheme":"file"},"relativeLocation":"golang.go-0.50.0","metadata":{"source":"gallery"}}`

// conIDE es una consola en una máquina con un IDE cuyas extensiones viven en una carpeta temporal.
func conIDE(t *testing.T, version string, inicial string) (Consola, IDE) {
	t.Helper()
	ide := IDE{Nombre: "Antigravity", Dir: filepath.Join(t.TempDir(), "extensions")}
	if inicial != "" {
		escribirArchivoIDE(t, filepath.Join(ide.Dir, "extensions.json"), inicial)
	}
	c := nuevaConsola(t, version, &claudeFalso{})
	c.BuscarIDEs = func() []IDE { return []IDE{ide} }
	return c, ide
}

func escribirArchivoIDE(t *testing.T, ruta, texto string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
}

func entradas(t *testing.T, ide IDE) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(ide.Dir, "extensions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("extensions.json ilegible: %v\n%s", err, data)
	}
	return out
}

func idDe(e map[string]any) string {
	id, _ := e["identifier"].(map[string]any)
	s, _ := id["id"].(string)
	return s
}

func nuestras(t *testing.T, ide IDE) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, e := range entradas(t, ide) {
		if idDe(e) == IDExtension {
			out = append(out, e)
		}
	}
	return out
}

func carpetasNuestras(t *testing.T, ide IDE) []string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(ide.Dir, IDExtension+"-*"))
	return m
}

func TestExtension_ActivarLaInstalaConLaVersionYLaRegistraUnaVez(t *testing.T) {
	c, ide := conIDE(t, "2.10.0-rc.82", "["+entradaAjena+"]")
	for range 2 {
		if _, err := c.Activar(false); err != nil {
			t.Fatal(err)
		}
	}
	carpeta := filepath.Join(ide.Dir, IDExtension+"-2.10.0-rc.82")
	for _, rel := range []string{"package.json", "extension.js", "lector.js", "panel.js", "media/vista.js", "media/pasos.js", "media/vista.css", "media/icono.svg"} {
		if _, err := os.Stat(filepath.Join(carpeta, rel)); err != nil {
			t.Fatalf("falta %s en la extensión instalada: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(carpeta, "panel.test.js")); err == nil {
		t.Fatal("las pruebas no viajan con la extensión")
	}
	var m struct{ Version, Publisher, Name string }
	data, _ := os.ReadFile(filepath.Join(carpeta, "package.json"))
	if err := json.Unmarshal(data, &m); err != nil || m.Version != "2.10.0-rc.82" || m.Publisher+"."+m.Name != IDExtension {
		t.Fatalf("el package.json lleva la versión de la CLI: %+v %v", m, err)
	}
	todas := entradas(t, ide)
	if len(todas) != 2 || idDe(todas[0]) != "golang.go" || !strings.Contains(string(mustJSON(t, todas[0])), `"uuid":"u-1"`) {
		t.Fatalf("la entrada ajena queda intacta y la nuestra una sola vez: %v", todas)
	}
	e := nuestras(t, ide)[0]
	loc, _ := e["location"].(map[string]any)
	if e["relativeLocation"] != IDExtension+"-2.10.0-rc.82" || e["version"] != "2.10.0-rc.82" || loc["scheme"] != "file" || !strings.HasSuffix(loc["path"].(string), "/"+IDExtension+"-2.10.0-rc.82") {
		t.Fatalf("la entrada apunta a la carpeta instalada: %v", e)
	}
	reg, _ := c.LeerRegistro()
	if len(reg.IDEs) != 1 || reg.IDEs[0] != ide {
		t.Fatalf("el registro recuerda dónde la instaló, para doctor y desactivar: %+v", reg)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestExtension_SinExtensionsJSONLoCrea(t *testing.T) {
	c, ide := conIDE(t, "2.10.0", "")
	if err := os.MkdirAll(ide.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	if len(nuestras(t, ide)) != 1 {
		t.Fatal("sin extensions.json, lo crea con la nuestra")
	}
}

func TestExtension_UnExtensionsJSONIlegibleNoSeToca(t *testing.T) {
	c, ide := conIDE(t, "2.10.0", "[{roto")
	hecho, err := c.Activar(false)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(ide.Dir, "extensions.json"))
	if string(data) != "[{roto" {
		t.Fatalf("un extensions.json que no se entiende no se reescribe: %s", data)
	}
	if !strings.Contains(strings.Join(hecho, "\n"), "extensions.json") {
		t.Fatalf("lo dice: %v", hecho)
	}
	if reg, _ := c.LeerRegistro(); len(reg.IDEs) != 0 {
		t.Fatalf("no queda registrada donde no se pudo: %+v", reg)
	}
}

func TestExtension_ActualizarReemplazaLaVersionAnteriorYDoctorLoNota(t *testing.T) {
	c, ide := conIDE(t, "2.10.0-rc.82", "["+entradaAjena+"]")
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	c.Version = "2.10.0-rc.83"
	d := diagnosticoDe(t, c, "Antigravity")
	if d.OK || !strings.Contains(d.Detalle, "2.10.0-rc.82") || !strings.Contains(d.Detalle, "doctor --fix") {
		t.Fatalf("doctor nota una extensión de otra versión: %+v", d)
	}
	escribirArchivoIDE(t, filepath.Join(ide.Dir, ".obsolete"), `{"`+IDExtension+`-2.10.0-rc.83":true,"golang.go-0.49.0":true}`)
	if _, err := c.Reparar(); err != nil {
		t.Fatal(err)
	}
	if d := diagnosticoDe(t, c, "Antigravity"); !d.OK {
		t.Fatalf("reparada queda al día: %+v", d)
	}
	if m := carpetasNuestras(t, ide); len(m) != 1 || filepath.Base(m[0]) != IDExtension+"-2.10.0-rc.83" {
		t.Fatalf("la carpeta de la versión anterior se quita: %v", m)
	}
	if n := nuestras(t, ide); len(n) != 1 || n[0]["version"] != "2.10.0-rc.83" || len(entradas(t, ide)) != 2 {
		t.Fatalf("una sola entrada nuestra, en la versión nueva: %v", entradas(t, ide))
	}
	obsoleta, _ := os.ReadFile(filepath.Join(ide.Dir, ".obsolete"))
	if strings.Contains(string(obsoleta), IDExtension) || !strings.Contains(string(obsoleta), "golang.go-0.49.0") {
		t.Fatalf("el IDE no borra la versión recién instalada al abrir (.obsolete), y lo ajeno queda: %s", obsoleta)
	}
}

func diagnosticoDe(t *testing.T, c Consola, nombre string) Diagnostico {
	t.Helper()
	for _, d := range c.Diagnosticar() {
		if strings.Contains(d.Nombre, nombre) {
			return d
		}
	}
	t.Fatalf("doctor no revisa la extensión en %s: %+v", nombre, c.Diagnosticar())
	return Diagnostico{}
}

func TestExtension_DoctorNotaQueFaltaLaCarpetaOLaEntrada(t *testing.T) {
	c, ide := conIDE(t, "2.10.0", "[]")
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(ide.Dir, IDExtension+"-2.10.0")); err != nil {
		t.Fatal(err)
	}
	if d := diagnosticoDe(t, c, "Antigravity"); d.OK || !strings.Contains(d.Detalle, "doctor --fix") {
		t.Fatalf("sin la carpeta, doctor lo nota: %+v", d)
	}
	if _, err := c.Reparar(); err != nil {
		t.Fatal(err)
	}
	escribirArchivoIDE(t, filepath.Join(ide.Dir, "extensions.json"), "[]")
	if d := diagnosticoDe(t, c, "Antigravity"); d.OK {
		t.Fatalf("sin su entrada en extensions.json, el IDE no la carga y doctor lo nota: %+v", d)
	}
}

func TestExtension_ConElBinarioInstaladoLeDejaSuRuta(t *testing.T) {
	c, ide := conIDE(t, "2.10.0", "[]")
	c.Binario = filepath.Join(t.TempDir(), "qdd")
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	var cfg struct{ Binario string }
	data, _ := os.ReadFile(filepath.Join(ide.Dir, IDExtension+"-2.10.0", "qdd.json"))
	if err := json.Unmarshal(data, &cfg); err != nil || cfg.Binario != c.Binario {
		t.Fatalf("en Windows execFile sin shell no encuentra qdd.cmd: la extensión recibe la ruta del binario real: %q %v", data, err)
	}
}

func TestExtension_DesactivarLaQuitaSinTocarLoAjeno(t *testing.T) {
	c, ide := conIDE(t, "2.10.0", "["+entradaAjena+"]")
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Desactivar(); err != nil {
		t.Fatal(err)
	}
	if len(carpetasNuestras(t, ide)) != 0 || len(nuestras(t, ide)) != 0 || len(entradas(t, ide)) != 1 {
		t.Fatalf("desactivar quita la carpeta y la entrada, y deja la ajena: %v", entradas(t, ide))
	}
}

func TestExtension_SinIDENoToca(t *testing.T) {
	c := nuevaConsola(t, "2.10.0", &claudeFalso{})
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	for _, d := range c.Diagnosticar() {
		if strings.Contains(d.Nombre, "Antigravity") || strings.Contains(d.Nombre, "VS Code") {
			t.Fatalf("sin IDE no hay nada que revisar: %+v", d)
		}
	}
}

func TestExtension_EmbebidaLlevaTodosLosArchivosDeLaExtension(t *testing.T) {
	err := filepath.WalkDir("mod/vscode", func(ruta string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(ruta, ".test.js") {
			return err
		}
		if _, err := extensionFS.ReadFile(filepath.ToSlash(ruta)); err != nil {
			t.Errorf("%s no viaja en el binario: la extensión instalada no carga sin él", ruta)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
