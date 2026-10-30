package consola

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Rule: Prohibido usar 'else'
// 'qdd consola activar' (RFC 0006 §3.6): un solo interruptor deja en Claude Code el panel de estado por frente (mod
// qdd-estado) y la barra en Go. El mod viaja dentro del binario, así que su versión es siempre la de la CLI que le
// entrega 'qdd estado ver --json'. Se instala con los comandos oficiales de Claude Code desde un marketplace local en
// la carpeta central de QDD; nada se escribe en el proyecto.

const (
	// Marketplace y Plugin forman el id que ve Claude Code: qdd-estado@qdd.
	Marketplace = "qdd"
	Plugin      = "qdd-estado"
	// ComandoBarra es lo que queda en statusLine.command.
	ComandoBarra = "qdd consola statusline"
	// RefrescoBarra (segundos) hace que la barra se actualice sola, también sin actividad en la sesión.
	RefrescoBarra = 2
)

// ID es el id del plugin en Claude Code.
func ID() string { return Plugin + "@" + Marketplace }

//go:embed mod/qdd-estado/.claude-plugin/plugin.json mod/qdd-estado/hooks/hooks.json mod/qdd-estado/hooks/register.js
var modFS embed.FS

// huellaReferencia100 es el SHA-256 (sin espacios al final) de referencias/statusline.sh del blueprint
// consola-claude-code v1.0.0: solo esa barra, sin cambios, se reemplaza por la de QDD.
var huellaReferencia100 = "274b5f0fa0b0758ff7b7a82812f1d6b170ac1228f173d589ad6ae1e95521e2d6"

// Ejecutar corre el CLI de Claude Code con esos argumentos y devuelve su salida combinada.
type Ejecutar func(args ...string) ([]byte, error)

// Claude es el Ejecutar real; falla con un mensaje claro si 'claude' no está en el PATH.
func Claude(args ...string) ([]byte, error) {
	bin, err := exec.LookPath("claude")
	if err != nil {
		return nil, errors.New("no se encontró 'claude' (Claude Code) en el PATH")
	}
	return exec.Command(bin, args...).CombinedOutput()
}

// Consola agrupa lo que activar necesita; los campos vacíos toman su valor real.
type Consola struct {
	Version   string   // versión de la CLI; va al plugin.json del mod
	QDDHome   string   // carpeta central de QDD (por defecto, blueprints.Home)
	ClaudeDir string   // configuración de Claude Code (CLAUDE_CONFIG_DIR o ~/.claude)
	Run       Ejecutar // por defecto, Claude
	Recursos  *bool    // segunda línea de recursos en la barra; nil conserva lo elegido antes
}

// Registro es lo que QDD recuerda de la activación, en <QDDHome>/claude/activo.json.
type Registro struct {
	Version       string          `json:"version"`
	Barra         bool            `json:"barra"`
	BarraAnterior json.RawMessage `json:"barra_anterior,omitempty"`
	Recursos      bool            `json:"recursos,omitempty"`
}

// Diagnostico es un control de doctor.
type Diagnostico struct {
	Nombre  string
	OK      bool
	Detalle string
}

var semver = regexp.MustCompile(`^\d+\.\d+\.\d+([-+][0-9A-Za-z.-]+)?$`)

func (c Consola) completar() (Consola, error) {
	if c.Run == nil {
		c.Run = Claude
	}
	if c.QDDHome == "" {
		h, err := blueprints.Home()
		if err != nil {
			return c, err
		}
		c.QDDHome = h
	}
	if c.ClaudeDir == "" {
		c.ClaudeDir = claudeDir()
	}
	if !semver.MatchString(strings.TrimPrefix(c.Version, "v")) {
		c.Version = "0.0.0-dev"
	}
	c.Version = strings.TrimPrefix(c.Version, "v")
	return c, nil
}

func claudeDir() string {
	if d := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".claude"
	}
	return filepath.Join(home, ".claude")
}

// DirMarketplace es la carpeta del marketplace local.
func (c Consola) DirMarketplace() string { return filepath.Join(c.QDDHome, "claude") }

// DirRecursos es donde la barra guarda la caché de la línea de recursos (CPU y GPU entre lecturas).
func (c Consola) DirRecursos() string {
	c, err := c.completar()
	if err != nil {
		return ""
	}
	return c.DirMarketplace()
}

func (c Consola) rutaRegistro() string { return filepath.Join(c.DirMarketplace(), "activo.json") }

func (c Consola) rutaSettings() string { return filepath.Join(c.ClaudeDir, "settings.json") }

// LeerRegistro devuelve el registro de activación; ok=false si la consola no está activada.
func (c Consola) LeerRegistro() (Registro, bool) {
	c, err := c.completar()
	if err != nil {
		return Registro{}, false
	}
	var r Registro
	data, err := os.ReadFile(c.rutaRegistro())
	if err != nil || json.Unmarshal(data, &r) != nil {
		return Registro{}, false
	}
	return r, true
}

// EscribirMarketplace deja el marketplace local con el mod de esta versión, borrando lo que hubiera antes.
func (c Consola) EscribirMarketplace() error {
	destino := filepath.Join(c.DirMarketplace(), "plugins", Plugin)
	if err := os.RemoveAll(destino); err != nil {
		return err
	}
	err := fs.WalkDir(modFS, "mod/qdd-estado", func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := modFS.ReadFile(ruta)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(ruta, "mod/qdd-estado/")
		if rel == ".claude-plugin/plugin.json" {
			data, err = conVersion(data, c.Version)
			if err != nil {
				return err
			}
		}
		return escribir(filepath.Join(destino, filepath.FromSlash(rel)), data)
	})
	if err != nil {
		return err
	}
	catalogo := map[string]any{
		"name":  Marketplace,
		"owner": map[string]string{"name": "QDD Framework"},
		"plugins": []map[string]string{{
			"name":        Plugin,
			"source":      "./plugins/" + Plugin,
			"description": "Panel con el estado por frente del proyecto (qdd estado)",
			"version":     c.Version,
		}},
	}
	data, _ := json.MarshalIndent(catalogo, "", "  ")
	return escribir(filepath.Join(c.DirMarketplace(), ".claude-plugin", "marketplace.json"), append(data, '\n'))
}

func conVersion(manifiesto []byte, version string) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(manifiesto, &m); err != nil {
		return nil, fmt.Errorf("plugin.json embebido ilegible: %w", err)
	}
	m["version"] = version
	data, _ := json.MarshalIndent(m, "", "  ")
	return append(data, '\n'), nil
}

func escribir(ruta string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(ruta), ".qdd-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), ruta)
}

// Activar deja el panel instalado y al día y, con barra, la barra en Go. Repetirla no cambia nada. Devuelve lo que hizo.
func (c Consola) Activar(barra bool) ([]string, error) {
	c, err := c.completar()
	if err != nil {
		return nil, err
	}
	var hecho []string
	if err := c.EscribirMarketplace(); err != nil {
		return hecho, fmt.Errorf("no se pudo escribir el marketplace local: %w", err)
	}
	if out, err := c.Run("plugin", "marketplace", "add", c.DirMarketplace(), "--scope", "user"); err != nil {
		return hecho, fmt.Errorf("claude plugin marketplace add: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := c.Run("plugin", "install", ID(), "--scope", "user"); err != nil {
		return hecho, fmt.Errorf("claude plugin install: %v: %s", err, strings.TrimSpace(string(out)))
	}
	instalada, _ := c.versionInstalada()
	if instalada != c.Version {
		if out, err := c.Run("plugin", "update", ID(), "--scope", "user"); err != nil {
			return hecho, fmt.Errorf("claude plugin update: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}
	hecho = append(hecho, fmt.Sprintf("Panel %s %s instalado en Claude Code (alcance de usuario).", ID(), c.Version))

	previo, _ := c.LeerRegistro()
	reg := Registro{Version: c.Version, Barra: false, BarraAnterior: previo.BarraAnterior, Recursos: previo.Recursos}
	if c.Recursos != nil {
		reg.Recursos = *c.Recursos
	}
	if barra {
		msg, anterior, puesta, err := c.ponerBarra()
		if err != nil {
			return hecho, err
		}
		hecho = append(hecho, msg)
		reg.Barra = puesta
		if anterior != nil {
			reg.BarraAnterior = anterior
		}
	}
	if reg.Barra && reg.Recursos {
		hecho = append(hecho, "La barra suma la línea de recursos: D (disco del proyecto) · G0… (GPU) · R (RAM) · C (CPU).")
	}
	data, _ := json.MarshalIndent(reg, "", "  ")
	if err := escribir(c.rutaRegistro(), append(data, '\n')); err != nil {
		return hecho, err
	}
	return append(hecho, "Abre una sesión nueva de Claude Code (o /reload-plugins) para ver el panel; /estado lo abre."), nil
}

// versionInstalada lee 'claude plugin list --json'; "" si el panel no está instalado.
func (c Consola) versionInstalada() (string, bool) {
	out, err := c.Run("plugin", "list", "--json")
	if err != nil {
		return "", false
	}
	var lista []struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		Enabled bool   `json:"enabled"`
	}
	if json.Unmarshal(sinPrefijo(out), &lista) != nil {
		return "", false
	}
	for _, p := range lista {
		if p.ID == ID() {
			return p.Version, p.Enabled
		}
	}
	return "", false
}

// sinPrefijo descarta los avisos que el CLI pueda imprimir antes del JSON.
func sinPrefijo(out []byte) []byte {
	s := string(out)
	if i := strings.Index(s, "["); i >= 0 {
		return []byte(s[i:])
	}
	return out
}

// ponerBarra cambia statusLine a la barra en Go solo si no había barra, si ya es la de QDD o si es el script de
// referencia v1.0.0 sin cambios. Una barra propia no se toca. Devuelve el statusLine anterior para poder restaurarlo.
func (c Consola) ponerBarra() (string, json.RawMessage, bool, error) {
	settings, err := c.leerSettings()
	if err != nil {
		return "", nil, false, err
	}
	actual := settings["statusLine"]
	if esBarraQDD(actual) {
		return "La barra ya es «" + ComandoBarra + "».", nil, true, nil
	}
	if len(actual) > 0 && !c.esReferencia(actual) {
		return "Tu barra (statusLine) es propia y no se cambió; para sumar el estado agrega «qdd estado statusline --dir <proyecto>».", nil, false, nil
	}
	nueva, _ := json.Marshal(map[string]any{"type": "command", "command": ComandoBarra, "refreshInterval": RefrescoBarra})
	settings["statusLine"] = nueva
	if err := c.guardarSettings(settings); err != nil {
		return "", nil, false, err
	}
	return "Barra cambiada a «" + ComandoBarra + "» (se refresca cada 2 s).", actual, true, nil
}

func esBarraQDD(raw json.RawMessage) bool {
	var s struct {
		Command string `json:"command"`
	}
	return len(raw) > 0 && json.Unmarshal(raw, &s) == nil && strings.TrimSpace(s.Command) == ComandoBarra
}

// esReferencia dice si statusLine apunta a un script idéntico a la referencia v1.0.0 del blueprint.
func (c Consola) esReferencia(raw json.RawMessage) bool {
	var s struct {
		Command string `json:"command"`
	}
	if json.Unmarshal(raw, &s) != nil || s.Command == "" {
		return false
	}
	ruta := s.Command
	if strings.HasPrefix(ruta, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		ruta = filepath.Join(home, ruta[2:])
	}
	data, err := os.ReadFile(ruta)
	if err != nil {
		return false
	}
	suma := sha256.Sum256([]byte(strings.TrimRight(string(data), " \t\r\n")))
	return hex.EncodeToString(suma[:]) == huellaReferencia100
}

func (c Consola) leerSettings() (map[string]json.RawMessage, error) {
	settings := map[string]json.RawMessage{}
	data, err := os.ReadFile(c.rutaSettings())
	if os.IsNotExist(err) {
		return settings, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("%s no es JSON válido; no se cambió la barra: %w", c.rutaSettings(), err)
	}
	return settings, nil
}

// guardarSettings deja una copia (settings.json.qdd.bak) y escribe de forma atómica.
func (c Consola) guardarSettings(settings map[string]json.RawMessage) error {
	if previo, err := os.ReadFile(c.rutaSettings()); err == nil {
		if err := escribir(c.rutaSettings()+".qdd.bak", previo); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return escribir(c.rutaSettings(), append(data, '\n'))
}

// Desactivar desinstala el panel, quita el marketplace local y devuelve la barra anterior si la había cambiado QDD.
func (c Consola) Desactivar() ([]string, error) {
	c, err := c.completar()
	if err != nil {
		return nil, err
	}
	var hecho []string
	if _, err := c.Run("plugin", "uninstall", ID(), "--scope", "user"); err == nil {
		hecho = append(hecho, "Panel "+ID()+" desinstalado.")
	}
	if _, err := c.Run("plugin", "marketplace", "remove", Marketplace); err == nil {
		hecho = append(hecho, "Marketplace local «"+Marketplace+"» quitado.")
	}
	reg, ok := c.LeerRegistro()
	if ok && reg.Barra {
		msg, err := c.restaurarBarra(reg.BarraAnterior)
		if err != nil {
			return hecho, err
		}
		hecho = append(hecho, msg)
	}
	if err := os.RemoveAll(c.DirMarketplace()); err != nil {
		return hecho, err
	}
	return hecho, nil
}

func (c Consola) restaurarBarra(anterior json.RawMessage) (string, error) {
	settings, err := c.leerSettings()
	if err != nil {
		return "", err
	}
	if !esBarraQDD(settings["statusLine"]) {
		return "La barra ya no era la de QDD; no se tocó.", nil
	}
	delete(settings, "statusLine")
	msg := "Barra de QDD quitada."
	if len(anterior) > 0 {
		settings["statusLine"] = anterior
		msg = "Barra anterior restaurada."
	}
	return msg, c.guardarSettings(settings)
}

// Diagnosticar revisa la consola solo si se activó en esta máquina; sin activar no hay nada que revisar.
func (c Consola) Diagnosticar() []Diagnostico {
	reg, ok := c.LeerRegistro()
	if !ok {
		return nil
	}
	c, err := c.completar()
	if err != nil {
		return []Diagnostico{{Nombre: "Panel de estado en Claude Code", Detalle: err.Error()}}
	}
	version, enabled := c.versionInstalada()
	panel := Diagnostico{Nombre: "Panel de estado en Claude Code (" + ID() + ")", OK: version == c.Version && enabled}
	if version == "" {
		panel.Detalle = "no está instalado; 'qdd doctor --fix' o 'qdd consola activar' lo instala"
	}
	if version != "" && version != c.Version {
		panel.Detalle = fmt.Sprintf("instalado %s, la CLI es %s; 'qdd doctor --fix' lo actualiza", version, c.Version)
	}
	if version != "" && !enabled {
		panel.Detalle = "instalado pero desactivado en Claude Code"
	}
	diags := []Diagnostico{panel}
	if !reg.Barra {
		return diags
	}
	settings, err := c.leerSettings()
	barra := Diagnostico{Nombre: "Barra de la consola con «" + ComandoBarra + "»", OK: err == nil && esBarraQDD(settings["statusLine"])}
	if !barra.OK {
		barra.Detalle = "statusLine ya no apunta a la barra de QDD; 'qdd consola activar' la vuelve a poner"
	}
	return append(diags, barra)
}

// Reparar vuelve a activar con lo que se eligió al activar.
func (c Consola) Reparar() ([]string, error) {
	reg, ok := c.LeerRegistro()
	if !ok {
		return nil, nil
	}
	return c.Activar(reg.Barra)
}
