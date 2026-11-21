package consola

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// El panel en Antigravity y VS Code (RFC 0014, fase C): la extensión mod/vscode viaja dentro del binario y activar,
// actualizar y doctor --fix la dejan en la versión de la CLI en cada IDE de la máquina. Nunca se ejecuta el IDE
// ('antigravity --version' abre la aplicación): se escribe su carpeta de extensiones y su extensions.json, con
// escritura atómica y sin tocar las entradas ajenas. Un extensions.json que no se entiende no se reescribe.

const (
	// IDExtension es publisher.name de mod/vscode/package.json: la carpeta y la entrada de extensions.json.
	IDExtension         = "qdd.qdd-panel"
	archivoExtensiones  = "extensions.json"
	archivoObsoletas    = ".obsolete"
	archivoBinarioIDE   = "qdd.json"
	raizExtensionEmbeda = "mod/vscode"
)

//go:embed mod/vscode/package.json mod/vscode/extension.js mod/vscode/lector.js mod/vscode/panel.js mod/vscode/media
var extensionFS embed.FS

// IDE es un editor de la familia de VS Code: su nombre y su carpeta de extensiones.
type IDE struct {
	Nombre string `json:"nombre"`
	Dir    string `json:"dir"`
}

// idesReales son los IDE con carpeta de extensiones en esta máquina.
func idesReales() []IDE {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	conocidos := []IDE{
		{Nombre: "Antigravity", Dir: filepath.Join(home, ".antigravity-ide", "extensions")},
		{Nombre: "VS Code", Dir: filepath.Join(home, ".vscode", "extensions")},
	}
	var out []IDE
	for _, ide := range conocidos {
		if info, err := os.Stat(ide.Dir); err == nil && info.IsDir() {
			out = append(out, ide)
		}
	}
	return out
}

func (c Consola) buscarIDEs() []IDE {
	if c.BuscarIDEs != nil {
		return c.BuscarIDEs()
	}
	return idesReales()
}

func (c Consola) carpetaExtension() string { return IDExtension + "-" + c.Version }

// activarExtensiones la instala en cada IDE y devuelve lo hecho y los IDE donde quedó registrada.
func (c Consola) activarExtensiones() ([]string, []IDE, error) {
	var hecho []string
	var dirs []IDE
	for _, ide := range c.buscarIDEs() {
		msg, ok, err := c.activarExtension(ide)
		hecho = append(hecho, msg)
		if err != nil {
			return hecho, dirs, err
		}
		if ok {
			dirs = append(dirs, ide)
		}
	}
	return hecho, dirs, nil
}

func (c Consola) activarExtension(ide IDE) (string, bool, error) {
	lista, err := leerExtensiones(ide.Dir)
	if err != nil {
		return fmt.Sprintf("%s: %v; no se toca y la extensión no se instala ahí", ide.Nombre, err), false, nil
	}
	destino := filepath.Join(ide.Dir, c.carpetaExtension())
	if err := c.escribirExtension(destino); err != nil {
		return "", false, err
	}
	if err := quitarOtrasVersiones(ide.Dir, c.carpetaExtension()); err != nil {
		return "", false, err
	}
	entrada, err := json.Marshal(c.entradaExtension(destino))
	if err != nil {
		return "", false, err
	}
	lista = append(sinNuestras(lista), entrada)
	if err := guardarExtensiones(ide.Dir, lista); err != nil {
		return "", false, err
	}
	if err := sacarDeObsoletas(ide.Dir); err != nil {
		return "", false, err
	}
	return fmt.Sprintf("Panel de QDD en %s (%s %s): ábrelo desde «QDD» en la barra de actividad.", ide.Nombre, IDExtension, c.Version), true, nil
}

// escribirExtension deja los archivos de la extensión con la versión de la CLI y, si se conoce, la ruta del binario:
// en Windows, execFile sin shell no encuentra el qdd.cmd de npm.
func (c Consola) escribirExtension(destino string) error {
	if err := os.RemoveAll(destino); err != nil {
		return err
	}
	err := fs.WalkDir(extensionFS, raizExtensionEmbeda, func(ruta string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := extensionFS.ReadFile(ruta)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(ruta, raizExtensionEmbeda+"/")
		if rel == "package.json" {
			data, err = conVersion(data, c.Version)
		}
		if err != nil {
			return err
		}
		return escribir(filepath.Join(destino, filepath.FromSlash(rel)), data)
	})
	if err != nil || c.Binario == "" {
		return err
	}
	data, _ := json.MarshalIndent(map[string]string{"binario": c.Binario}, "", "  ")
	return escribir(filepath.Join(destino, archivoBinarioIDE), append(data, '\n'))
}

func quitarOtrasVersiones(dir, actual string) error {
	viejas, _ := filepath.Glob(filepath.Join(dir, IDExtension+"-*"))
	for _, v := range viejas {
		if filepath.Base(v) == actual {
			continue
		}
		if err := os.RemoveAll(v); err != nil {
			return err
		}
	}
	return nil
}

// entradaExtension es la entrada de extensions.json, como la escribe el IDE al instalar un .vsix.
func (c Consola) entradaExtension(destino string) map[string]any {
	ruta := filepath.ToSlash(destino)
	if !strings.HasPrefix(ruta, "/") {
		ruta = "/" + ruta
	}
	return map[string]any{
		"identifier":       map[string]any{"id": IDExtension},
		"version":          c.Version,
		"location":         map[string]any{"$mid": 1, "path": ruta, "scheme": "file"},
		"relativeLocation": filepath.Base(destino),
		"metadata":         map[string]any{"installedTimestamp": time.Now().UnixMilli(), "source": "vsix", "pinned": true},
	}
}

// leerExtensiones devuelve las entradas tal cual (las ajenas no se reinterpretan); sin archivo, ninguna.
func leerExtensiones(dir string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(filepath.Join(dir, archivoExtensiones))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var lista []json.RawMessage
	if err := json.Unmarshal(data, &lista); err != nil {
		return nil, fmt.Errorf("%s ilegible", filepath.Join(dir, archivoExtensiones))
	}
	return lista, nil
}

func idDeEntrada(e json.RawMessage) string {
	var v struct {
		Identifier struct {
			ID string `json:"id"`
		} `json:"identifier"`
	}
	_ = json.Unmarshal(e, &v)
	return strings.ToLower(v.Identifier.ID)
}

func sinNuestras(lista []json.RawMessage) []json.RawMessage {
	var out []json.RawMessage
	for _, e := range lista {
		if idDeEntrada(e) == IDExtension {
			continue
		}
		out = append(out, e)
	}
	return out
}

func guardarExtensiones(dir string, lista []json.RawMessage) error {
	if lista == nil {
		lista = []json.RawMessage{}
	}
	data, err := json.Marshal(lista)
	if err != nil {
		return err
	}
	return escribir(filepath.Join(dir, archivoExtensiones), data)
}

// sacarDeObsoletas: el IDE borra al abrir lo que .obsolete marca; una versión nuestra marcada antes no debe caer.
func sacarDeObsoletas(dir string) error {
	ruta := filepath.Join(dir, archivoObsoletas)
	data, err := os.ReadFile(ruta)
	if err != nil {
		return nil
	}
	var marcadas map[string]any
	if json.Unmarshal(data, &marcadas) != nil {
		return nil
	}
	cambio := false
	for k := range marcadas {
		if strings.HasPrefix(strings.ToLower(k), IDExtension+"-") {
			delete(marcadas, k)
			cambio = true
		}
	}
	if !cambio {
		return nil
	}
	nuevo, _ := json.Marshal(marcadas)
	return escribir(ruta, nuevo)
}

// diagnosticarExtension revisa que la extensión esté en la versión de la CLI y registrada para que el IDE la cargue.
func (c Consola) diagnosticarExtension(ide IDE) Diagnostico {
	d := Diagnostico{Nombre: "Panel de QDD en " + ide.Nombre}
	dir := ide.Dir
	instalada := versionExtension(dir)
	if instalada == "" {
		d.Detalle = "no está instalada; 'qdd doctor --fix' la instala"
		return d
	}
	if instalada != c.Version {
		d.Detalle = fmt.Sprintf("instalada %s, la CLI es %s; 'qdd doctor --fix' la actualiza", instalada, c.Version)
		return d
	}
	lista, err := leerExtensiones(dir)
	if err != nil {
		d.Detalle = err.Error() + "; el IDE no la carga"
		return d
	}
	if !registradaEn(lista, c.carpetaExtension()) {
		d.Detalle = "no está en " + archivoExtensiones + " y el IDE no la carga; 'qdd doctor --fix' la registra"
		return d
	}
	d.OK = true
	return d
}

// versionExtension es la versión del package.json de la carpeta nuestra que haya; ninguna, vacía.
func versionExtension(dir string) string {
	carpetas, _ := filepath.Glob(filepath.Join(dir, IDExtension+"-*"))
	for _, carpeta := range carpetas {
		var m struct {
			Version string `json:"version"`
		}
		data, err := os.ReadFile(filepath.Join(carpeta, "package.json"))
		if err != nil || json.Unmarshal(data, &m) != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(carpeta, "extension.js")); err != nil {
			continue
		}
		return m.Version
	}
	return ""
}

func registradaEn(lista []json.RawMessage, carpeta string) bool {
	for _, e := range lista {
		var v struct {
			RelativeLocation string `json:"relativeLocation"`
		}
		_ = json.Unmarshal(e, &v)
		if idDeEntrada(e) == IDExtension && v.RelativeLocation == carpeta {
			return true
		}
	}
	return false
}

// desactivarExtension quita la carpeta y la entrada nuestras; lo ajeno queda.
func desactivarExtension(ide IDE) (string, error) {
	dir := ide.Dir
	if err := quitarOtrasVersiones(dir, ""); err != nil {
		return "", err
	}
	lista, err := leerExtensiones(dir)
	if err != nil {
		return "", nil
	}
	if err := guardarExtensiones(dir, sinNuestras(lista)); err != nil {
		return "", err
	}
	return "Panel de QDD quitado de " + ide.Nombre + ".", nil
}
