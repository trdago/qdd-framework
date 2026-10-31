package consola

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

var ahora = time.Date(2026, 10, 29, 15, 0, 0, 0, time.Local)

// Barra v7 del blueprint consola-claude-code (pedido del usuario, 2026-10-31): a la izquierda solo el modelo y el modo
// de la IA; a la derecha, en lista y pegados al borde, el contexto, el uso de 5 h y el de la semana. Sin carpeta ni
// segmento de tareas (eso está en el panel).

// derecha devuelve la línea sin el relleno de la izquierda y comprueba que termina a MargenBarra del borde.
func derecha(t *testing.T, linea string, ancho int) string {
	t.Helper()
	if n := utf8.RuneCountInString(linea); n != ancho-MargenBarra {
		t.Fatalf("la línea %q mide %d; pegada a la derecha mide %d", linea, n, ancho-MargenBarra)
	}
	return strings.TrimLeft(linea, " "+Relleno)
}

// FND-091 (captura del usuario, 2026-10-31): Claude Code quita los espacios al inicio de cada línea de la barra (5h y
// semana quedaron a la izquierda) y corta con «…» la que llega casi al borde («contexto …»).
func TestBarra_LaListaNoEmpiezaConEspaciosYDejaMargen_FND091(t *testing.T) {
	in := `{"model":{"display_name":"Opus 5.5"},"context_window":{"used_percentage":38},"rate_limits":{"five_hour":{"used_percentage":12}}}`
	got := strings.Split(Barra([]byte(in), ahora, "auto", 70), "\n")
	if len(got) != 2 || !strings.HasPrefix(got[1], Relleno) || strings.TrimLeft(got[1], " "+Relleno) != "5h 12%" {
		t.Fatalf("una línea de la lista empieza con un blanco que Claude Code no borra: %q", got)
	}
	if MargenBarra < 6 {
		t.Fatalf("Claude Code deja menos ancho que la terminal: el margen derecho es de al menos 6 columnas, no %d", MargenBarra)
	}
}

func TestBarra_ModeloYModoALaIzquierdaUsosEnListaALaDerecha_CCC_C01(t *testing.T) {
	resets5h := time.Date(2026, 10, 29, 18, 30, 0, 0, time.Local).Unix()
	in := fmt.Sprintf(`{"workspace":{"project_dir":"/home/a/proyectos/notificador"},"model":{"display_name":"Opus 5.5"},
"context_window":{"used_percentage":37.6},
"rate_limits":{"five_hour":{"used_percentage":12.2,"resets_at":%d},"seven_day":{"used_percentage":40,"resets_at":"2026-10-31T09:00:00-03:00"}}}`, resets5h)
	got := strings.Split(Barra([]byte(in), ahora, "auto", 70), "\n")
	sabado := time.Date(2026, 10, 31, 9, 0, 0, 0, time.FixedZone("", -3*3600)).In(time.Local)
	if len(got) != 3 {
		t.Fatalf("tres líneas: %q", got)
	}
	derecha(t, got[0], 70)
	if !strings.HasPrefix(got[0], "Opus 5.5 · auto ") || !strings.HasSuffix(got[0], " contexto 38%") {
		t.Fatalf("modelo y modo a la izquierda, contexto a la derecha: %q", got[0])
	}
	if derecha(t, got[1], 70) != "5h 12% ↻ 18:30" {
		t.Fatalf("5 h debajo, a la derecha: %q", got[1])
	}
	if want := "semana 40% ↻ " + dias[sabado.Weekday()] + " " + sabado.Format("15:04"); derecha(t, got[2], 70) != want {
		t.Fatalf("semana al final, a la derecha: %q, quiero %q", got[2], want)
	}
	for _, l := range got {
		if strings.Contains(l, "notificador") || strings.Contains(l, "📁") {
			t.Fatalf("la barra ya no muestra la carpeta: %q", l)
		}
	}
}

func TestBarra_CamposAusentesSeOmiten_CCC_C02(t *testing.T) {
	if got := Barra([]byte(`{"workspace":{"current_dir":"/x/y/proyecto"},"model":{"display_name":"Sonnet"}}`), ahora, "", 70); got != "Sonnet" {
		t.Fatalf("sin modo, contexto ni tramos y sin ceros inventados: %q", got)
	}
	if got := Barra([]byte(`no es json`), ahora, "auto", 70); got != "" {
		t.Fatalf("JSON ilegible no muestra nada: %q", got)
	}
}

func TestBarra_RenovacionEpochTextoOISO_CCC_C03(t *testing.T) {
	epoch := time.Date(2026, 10, 29, 18, 30, 0, 0, time.Local).Unix()
	for _, r := range []string{fmt.Sprint(epoch), fmt.Sprintf("%q", fmt.Sprint(epoch)), `"` + time.Unix(epoch, 0).UTC().Format(time.RFC3339) + `"`} {
		got := Barra([]byte(`{"cwd":"/p","rate_limits":{"five_hour":{"used_percentage":5,"resets_at":`+r+`}}}`), ahora, "", 0)
		if !strings.HasSuffix(got, "5h 5% ↻ 18:30") {
			t.Fatalf("resets_at=%s: %q", r, got)
		}
	}
}

func TestBarra_ElModoSeNombraEnEspañol_CCC_C01(t *testing.T) {
	for modo, want := range map[string]string{"auto": "auto", "plan": "plan", "acceptEdits": "acepta ediciones", "default": "normal", "bypassPermissions": "sin permisos", "dontAsk": "no pregunta"} {
		if got := Barra([]byte(`{"model":{"display_name":"Opus"}}`), ahora, modo, 0); got != "Opus · "+want {
			t.Fatalf("modo %s: %q", modo, got)
		}
	}
}

func TestBarra_SinCarpetaNiSegmentoDeTareas_CCC_C04(t *testing.T) {
	dir := t.TempDir()
	escribirArchivo(t, filepath.Join(dir, ".qdd", "project", "estado.yaml"), `formato: 1
etapas: [Código]
frentes:
  - {id: a, nombre: A, estado: abierto, celdas: {Código: {estado: "⏳", bloqueo: b, de_quien: c}}}
`)
	got := Barra([]byte(`{"workspace":{"project_dir":`+jsonStr(dir)+`},"model":{"display_name":"Opus"}}`), ahora, "auto", 70)
	if got != "Opus · auto" {
		t.Fatalf("ni la carpeta ni «◧ tareas»: el panel ya los muestra: %q", got)
	}
}

func TestBarra_SinAnchoOAngostaNoSeDesordena_CCC_C24(t *testing.T) {
	in := `{"model":{"display_name":"Opus 5.5"},"context_window":{"used_percentage":38},"rate_limits":{"five_hour":{"used_percentage":12}}}`
	if got := Barra([]byte(in), ahora, "auto", 0); got != "Opus 5.5 · auto · contexto 38%\n5h 12%" {
		t.Fatalf("sin el ancho de la terminal, la lista va sin relleno: %q", got)
	}
	got := strings.Split(Barra([]byte(in), ahora, "auto", 24), "\n")
	if len(got) != 3 || got[0] != "Opus 5.5 · auto" || derecha(t, got[1], 24) != "contexto 38%" || derecha(t, got[2], 24) != "5h 12%" {
		t.Fatalf("si el contexto no cabe junto al modelo, baja a la lista: %q", got)
	}
}

// claudeFalso simula el CLI de Claude Code: recuerda marketplaces y versiones instaladas.
type claudeFalso struct {
	llamadas    []string
	marketplace bool
	instalada   string
	carpeta     string // versión del plugin.json del marketplace local
	dir         string
	anterior    bool // sigue instalado el panel con su nombre anterior (qdd-estado@qdd)
}

func (f *claudeFalso) run(args ...string) ([]byte, error) {
	f.llamadas = append(f.llamadas, strings.Join(args, " "))
	cmd := strings.Join(args[:2], " ")
	switch {
	case cmd == "plugin marketplace" && args[2] == "add":
		f.marketplace = true
		f.dir = args[3]
		return []byte("ok"), nil
	case cmd == "plugin marketplace" && args[2] == "remove":
		f.marketplace = false
		return []byte("ok"), nil
	case cmd == "plugin install":
		if !f.marketplace {
			return []byte("marketplace not found"), fmt.Errorf("exit 1")
		}
		if f.instalada == "" {
			f.instalada = f.versionCarpeta()
		}
		return []byte("ok"), nil
	case cmd == "plugin update":
		f.instalada = f.versionCarpeta()
		return []byte("ok"), nil
	case cmd == "plugin uninstall" && args[2] == IDAnterior():
		if !f.anterior {
			return []byte("not installed"), fmt.Errorf("exit 1")
		}
		f.anterior = false
		return []byte("ok"), nil
	case cmd == "plugin uninstall":
		f.instalada = ""
		return []byte("ok"), nil
	case cmd == "plugin list":
		var lista []string
		if f.anterior {
			lista = append(lista, `{"id":"`+IDAnterior()+`","version":"2.4.0","enabled":true}`)
		}
		if f.instalada != "" {
			lista = append(lista, `{"id":"`+ID()+`","version":"`+f.instalada+`","enabled":true}`)
		}
		return []byte("[" + strings.Join(lista, ",") + "]"), nil
	}
	return nil, fmt.Errorf("comando inesperado: %v", args)
}

func (f *claudeFalso) versionCarpeta() string {
	data, _ := os.ReadFile(filepath.Join(f.dir, "plugins", Plugin, ".claude-plugin", "plugin.json"))
	var m struct {
		Version string `json:"version"`
	}
	_ = json.Unmarshal(data, &m)
	return m.Version
}

func nuevaConsola(t *testing.T, version string, f *claudeFalso) Consola {
	t.Helper()
	return Consola{Version: version, QDDHome: filepath.Join(t.TempDir(), "qdd"), ClaudeDir: filepath.Join(t.TempDir(), "claude"), Run: f.run,
		BuscarOpenCode: func() (string, bool) { return "", false }}
}

func TestActivar_InstalaElPanelConLaVersionDeLaCLIYEsIdempotente(t *testing.T) {
	f := &claudeFalso{}
	c := nuevaConsola(t, "2.2.0", f)
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	if f.instalada != "2.2.0" {
		t.Fatalf("el plugin.json del mod lleva la versión de la CLI: %q", f.instalada)
	}
	for _, rel := range []string{".claude-plugin/marketplace.json", "plugins/qdd-tareas/hooks/register.js", "plugins/qdd-tareas/hooks/hooks.json"} {
		if _, err := os.Stat(filepath.Join(c.DirMarketplace(), rel)); err != nil {
			t.Fatalf("falta %s en el marketplace local: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(c.DirMarketplace(), "plugins/qdd-tareas/tests")); err == nil {
		t.Fatal("las pruebas del mod no viajan al marketplace")
	}
	antes := len(f.llamadas)
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
	for _, l := range f.llamadas[antes:] {
		if strings.HasPrefix(l, "plugin update") {
			t.Fatal("repetir activar con la misma versión no actualiza")
		}
	}
	if d := c.Diagnosticar(); len(d) != 1 || !d[0].OK {
		t.Fatalf("doctor ve el panel al día: %+v", d)
	}
}

func TestActivar_ConCLINuevaActualizaElPanelYDoctorLoDetecta(t *testing.T) {
	f := &claudeFalso{}
	vieja := nuevaConsola(t, "2.2.0", f)
	if _, err := vieja.Activar(false); err != nil {
		t.Fatal(err)
	}
	nueva := vieja
	nueva.Version = "2.3.0"
	if d := nueva.Diagnosticar(); d[0].OK || !strings.Contains(d[0].Detalle, "2.2.0") {
		t.Fatalf("doctor detecta la versión vieja: %+v", d)
	}
	if _, err := nueva.Reparar(); err != nil {
		t.Fatal(err)
	}
	if f.instalada != "2.3.0" || !nueva.Diagnosticar()[0].OK {
		t.Fatalf("reparar actualiza el panel: %q", f.instalada)
	}
}

func TestDiagnosticar_SinActivarNoRevisaNada(t *testing.T) {
	if d := nuevaConsola(t, "2.2.0", &claudeFalso{}).Diagnosticar(); d != nil {
		t.Fatalf("sin activar no hay controles: %+v", d)
	}
}

func TestActivar_BarraSoloReemplazaLaReferenciaYDesactivarLaRestaura(t *testing.T) {
	f := &claudeFalso{}
	c := nuevaConsola(t, "2.2.0", f)
	script := filepath.Join(t.TempDir(), "statusline.sh")
	ref := "#!/usr/bin/env bash\n# barra de referencia v1.0.0 (simulada)\n"
	suma := sha256.Sum256([]byte(strings.TrimRight(ref, " \t\r\n")))
	original := huellaReferencia100
	huellaReferencia100 = hex.EncodeToString(suma[:])
	t.Cleanup(func() { huellaReferencia100 = original })
	escribirArchivo(t, script, ref)
	escribirArchivo(t, filepath.Join(c.ClaudeDir, "settings.json"), `{"model":"opus","statusLine":{"type":"command","command":`+jsonStr(script)+`}}`)
	if _, err := c.Activar(true); err != nil {
		t.Fatal(err)
	}
	settings := leerJSON(t, filepath.Join(c.ClaudeDir, "settings.json"))
	if !strings.Contains(settings["statusLine"], ComandoBarra) || !strings.Contains(settings["statusLine"], `"refreshInterval":2`) || settings["model"] != `"opus"` {
		t.Fatalf("la referencia v1.0.0 se reemplaza por la barra en Go y lo demás queda: %v", settings)
	}
	if _, err := os.Stat(filepath.Join(c.ClaudeDir, "settings.json.qdd.bak")); err != nil {
		t.Fatal("se deja copia de settings.json")
	}
	if d := c.Diagnosticar(); len(d) != 2 || !d[1].OK {
		t.Fatalf("doctor revisa la barra: %+v", d)
	}
	if _, err := c.Desactivar(); err != nil {
		t.Fatal(err)
	}
	settings = leerJSON(t, filepath.Join(c.ClaudeDir, "settings.json"))
	if !strings.Contains(settings["statusLine"], "statusline.sh") || f.instalada != "" {
		t.Fatalf("desactivar restaura la barra anterior y desinstala: %v %q", settings, f.instalada)
	}
	if _, ok := c.LeerRegistro(); ok {
		t.Fatal("desactivar borra el registro")
	}
}

func TestActivar_UnaBarraPropiaNoSeToca(t *testing.T) {
	c := nuevaConsola(t, "2.2.0", &claudeFalso{})
	propia := `{"statusLine":{"type":"command","command":"mi-barra --linda"}}`
	escribirArchivo(t, filepath.Join(c.ClaudeDir, "settings.json"), propia)
	hecho, err := c.Activar(true)
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(filepath.Join(c.ClaudeDir, "settings.json")); string(data) != propia {
		t.Fatalf("la barra propia queda igual: %s", data)
	}
	if !strings.Contains(strings.Join(hecho, "\n"), "propia") {
		t.Fatalf("avisa que no la cambió: %v", hecho)
	}
}

func TestActivar_SinClaudeFallaConMensajeClaro(t *testing.T) {
	c := nuevaConsola(t, "2.2.0", nil)
	c.Run = func(args ...string) ([]byte, error) {
		return nil, fmt.Errorf("no se encontró 'claude' (Claude Code) en el PATH")
	}
	if _, err := c.Activar(false); err == nil || !strings.Contains(err.Error(), "claude") {
		t.Fatalf("error claro sin claude: %v", err)
	}
}

func escribirArchivo(t *testing.T, ruta, contenido string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func leerJSON(t *testing.T, ruta string) map[string]string {
	t.Helper()
	data, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for k, v := range raw {
		var compacto strings.Builder
		var x any
		_ = json.Unmarshal(v, &x)
		b, _ := json.Marshal(x)
		compacto.Write(b)
		out[k] = compacto.String()
	}
	return out
}

func jsonStr(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// Son tareas, no frentes: el panel pasa a llamarse qdd-tareas@qdd y activar quita el de nombre anterior.
func TestActivar_RenombraElPanelAQddTareasYQuitaElAnterior(t *testing.T) {
	f := &claudeFalso{anterior: true}
	c := nuevaConsola(t, "2.5.0", f)
	hecho, err := c.Activar(false)
	if err != nil {
		t.Fatal(err)
	}
	if ID() != "qdd-tareas@qdd" || f.anterior || f.instalada != "2.5.0" {
		t.Fatalf("queda solo qdd-tareas@qdd: id=%s anterior=%v instalada=%q", ID(), f.anterior, f.instalada)
	}
	if !strings.Contains(strings.Join(hecho, "\n"), IDAnterior()) {
		t.Fatalf("activar dice que quitó el panel de nombre anterior: %v", hecho)
	}
	if _, err := c.Activar(false); err != nil {
		t.Fatal(err)
	}
}
