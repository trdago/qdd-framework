package consola

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// El panel en OpenCode (RFC 0006 §3.10): solo la forma de actualizar y sincronizar del panel de Claude Code. Un plugin
// de la TUI en la barra lateral lee cada 2 s 'qdd consola panel --json' y redibuja si algo cambió; no tiene acciones.
// Viaja dentro del binario y activar/actualizar/doctor --fix lo dejan en la versión de la CLI.

const (
	// marcaVersion es lo que se reemplaza por la versión de la CLI en el plugin embebido.
	marcaVersion = "__QDD_VERSION__"
	archivoTUI   = "tui.json"
	esquemaTUI   = "https://opencode.ai/tui.json"
)

//go:embed mod/opencode/qdd-tareas.tsx mod/opencode/panel.js
var openCodeFS embed.FS

var versionPlugin = regexp.MustCompile(`const VERSION = "([^"]*)"`)

// openCodeReal devuelve la configuración de OpenCode ($XDG_CONFIG_HOME/opencode o ~/.config/opencode) y si OpenCode
// está en esta máquina: el comando en el PATH o esa carpeta ya existe.
func openCodeReal() (string, bool) {
	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", false
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "opencode")
	if _, err := exec.LookPath("opencode"); err == nil {
		return dir, true
	}
	if info, err := os.Stat(dir); err == nil && info.IsDir() {
		return dir, true
	}
	return dir, false
}

func (c Consola) buscarOpenCode() (string, bool) {
	if c.BuscarOpenCode != nil {
		return c.BuscarOpenCode()
	}
	return openCodeReal()
}

// DirPluginOpenCode es la carpeta del plugin, dentro de la carpeta central de QDD.
func (c Consola) DirPluginOpenCode() string { return filepath.Join(c.QDDHome, "opencode") }

// RutaPluginOpenCode es el archivo que queda registrado en tui.json.
func (c Consola) RutaPluginOpenCode() string {
	c, err := c.completar()
	if err != nil {
		return ""
	}
	return filepath.Join(c.DirPluginOpenCode(), "qdd-tareas.tsx")
}

// escribirPluginOpenCode deja los archivos del plugin con la versión de la CLI.
func (c Consola) escribirPluginOpenCode() error {
	for _, nombre := range []string{"qdd-tareas.tsx", "panel.js"} {
		data, err := openCodeFS.ReadFile("mod/opencode/" + nombre)
		if err != nil {
			return err
		}
		data = bytes.ReplaceAll(data, []byte(marcaVersion), []byte(c.Version))
		if err := escribir(filepath.Join(c.DirPluginOpenCode(), nombre), data); err != nil {
			return err
		}
	}
	return nil
}

// leerTUI devuelve tui.json como mapa; ok=false si existe y no es JSON (con comentarios, por ejemplo): no se toca.
func leerTUI(dir string) (map[string]json.RawMessage, bool) {
	cfg := map[string]json.RawMessage{}
	data, err := os.ReadFile(filepath.Join(dir, archivoTUI))
	if os.IsNotExist(err) {
		return cfg, true
	}
	if err != nil || json.Unmarshal(data, &cfg) != nil {
		return nil, false
	}
	return cfg, true
}

func pluginsDe(cfg map[string]json.RawMessage) []json.RawMessage {
	var lista []json.RawMessage
	if raw, ok := cfg["plugin"]; ok {
		_ = json.Unmarshal(raw, &lista)
	}
	return lista
}

// esEntrada dice si una entrada de "plugin" (spec o [spec, opciones]) es la del plugin de qdd.
func esEntrada(raw json.RawMessage, ruta string) bool {
	var spec string
	if json.Unmarshal(raw, &spec) == nil {
		return spec == ruta || spec == "file://"+ruta
	}
	var par []json.RawMessage
	if json.Unmarshal(raw, &par) != nil || len(par) == 0 {
		return false
	}
	return esEntrada(par[0], ruta)
}

func registrado(cfg map[string]json.RawMessage, ruta string) bool {
	for _, p := range pluginsDe(cfg) {
		if esEntrada(p, ruta) {
			return true
		}
	}
	return false
}

// guardarTUI deja un respaldo la primera vez (tui.json.qdd.bak) y escribe de forma atómica.
func guardarTUI(dir string, cfg map[string]json.RawMessage) error {
	ruta := filepath.Join(dir, archivoTUI)
	_, errBak := os.Stat(ruta + ".qdd.bak")
	if previo, err := os.ReadFile(ruta); err == nil && os.IsNotExist(errBak) {
		if err := escribir(ruta+".qdd.bak", previo); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return escribir(ruta, append(data, '\n'))
}

// activarOpenCode instala el plugin y lo registra en tui.json una sola vez. Devuelve lo hecho y si quedó registrado.
func (c Consola) activarOpenCode(dir string) ([]string, bool, error) {
	if err := c.escribirPluginOpenCode(); err != nil {
		return nil, false, fmt.Errorf("no se pudo escribir el plugin de OpenCode: %w", err)
	}
	hecho := []string{fmt.Sprintf("Panel de qdd %s instalado en OpenCode (barra lateral, solo lectura).", c.Version)}
	cfg, ok := leerTUI(dir)
	if !ok {
		aviso := fmt.Sprintf("%s no es JSON válido y no se tocó; agrega a mano \"plugin\": [%q].", filepath.Join(dir, archivoTUI), c.RutaPluginOpenCode())
		return append(hecho, aviso), false, nil
	}
	ruta := c.RutaPluginOpenCode()
	if registrado(cfg, ruta) {
		return hecho, true, nil
	}
	spec, _ := json.Marshal(ruta)
	cfg["plugin"], _ = json.Marshal(append(pluginsDe(cfg), spec))
	if _, ok := cfg["$schema"]; !ok {
		cfg["$schema"], _ = json.Marshal(esquemaTUI)
	}
	if err := guardarTUI(dir, cfg); err != nil {
		return hecho, false, err
	}
	return append(hecho, "Registrado en "+filepath.Join(dir, archivoTUI)+"; abre OpenCode de nuevo para verlo."), true, nil
}

// desactivarOpenCode quita solo la entrada de qdd de tui.json y borra el plugin.
func (c Consola) desactivarOpenCode(dir string) ([]string, error) {
	var hecho []string
	cfg, ok := leerTUI(dir)
	ruta := c.RutaPluginOpenCode()
	if ok && registrado(cfg, ruta) {
		var quedan []json.RawMessage
		for _, p := range pluginsDe(cfg) {
			if !esEntrada(p, ruta) {
				quedan = append(quedan, p)
			}
		}
		cfg["plugin"], _ = json.Marshal(quedan)
		if len(quedan) == 0 {
			delete(cfg, "plugin")
		}
		if err := guardarTUI(dir, cfg); err != nil {
			return hecho, err
		}
		hecho = append(hecho, "Panel de qdd quitado de OpenCode.")
	}
	return hecho, os.RemoveAll(c.DirPluginOpenCode())
}

// diagnosticarOpenCode revisa que el plugin siga registrado y en la versión de la CLI.
func (c Consola) diagnosticarOpenCode(dir string) Diagnostico {
	d := Diagnostico{Nombre: "Panel de tareas en OpenCode"}
	cfg, ok := leerTUI(dir)
	if !ok || !registrado(cfg, c.RutaPluginOpenCode()) {
		d.Detalle = "no está registrado en " + filepath.Join(dir, archivoTUI) + "; 'qdd doctor --fix' lo registra"
		return d
	}
	data, err := os.ReadFile(c.RutaPluginOpenCode())
	if err != nil {
		d.Detalle = "falta el plugin; 'qdd doctor --fix' lo instala"
		return d
	}
	m := versionPlugin.FindSubmatch(data)
	if m == nil || string(m[1]) != c.Version {
		instalada := "desconocida"
		if m != nil {
			instalada = string(m[1])
		}
		d.Detalle = fmt.Sprintf("instalado %s, la CLI es %s; 'qdd doctor --fix' lo actualiza", instalada, c.Version)
		return d
	}
	d.OK = true
	return d
}
