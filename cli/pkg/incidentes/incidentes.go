// Package incidentes convierte las fallas de QDD mismo (un pánico de la CLI o de una tool MCP, una autorreparación de
// qdd doctor) en issues del repositorio de QDD, con un formato estándar que una IA puede leer, reproducir y reparar
// (pedido del usuario, 2026-11-01). Lo que viaja es solo de QDD: versión, sistema, el comando con sus valores
// reemplazados por marcadores y la traza dentro del código de QDD. Nada del proyecto: ni rutas, ni nombres, ni datos.
//
// Cada falla queda primero en un registro local (<QDD_HOME>/incidentes, un archivo por firma); publicar es aparte y
// en segundo plano, sin duplicar: la misma firma suma una ocurrencia y comenta el issue que ya existe.
package incidentes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'

const (
	TipoPanico          = "panico"
	TipoAutorreparacion = "autorreparacion"
	// TipoPanel es una vista del panel que lanzó al dibujarse (FND-189): el panel siguió, mostrando la falla.
	TipoPanel = "panel"
	Formato             = "qdd-incidente/1"
	// RepoPorDefecto es el repositorio de QDD; QDD_INCIDENTES_REPO lo cambia (un fork, o pruebas manuales).
	RepoPorDefecto = "trdago/qdd-framework"
	RepoEnv        = "QDD_INCIDENTES_REPO"
	// TopeDiario: a lo más tantas publicaciones (issues nuevos o comentarios) por día y por máquina.
	TopeDiario = 5
	// HomeEnv es la carpeta central de QDD (la misma de los blueprints); ApagadoEnv=off apaga los incidentes.
	HomeEnv    = "QDD_HOME"
	ApagadoEnv = "QDD_INCIDENTES"
	maxTraza   = 12
)

// Incidente es un archivo del registro: una falla de QDD por firma, con cuántas veces pasó y dónde se publicó.
type Incidente struct {
	Formato     string   `yaml:"formato"`
	Firma       string   `yaml:"firma"`
	Tipo        string   `yaml:"tipo"`
	Origen      string   `yaml:"origen"`
	Comando     string   `yaml:"comando,omitempty"` // el comando con sus valores como marcadores; no entra en la firma
	Mensaje     string   `yaml:"mensaje"`
	Traza       []string `yaml:"traza,omitempty"`
	Reparacion  string   `yaml:"reparacion,omitempty"`
	Version     string   `yaml:"version"`
	SO          string   `yaml:"so"`
	Ocurrencias int      `yaml:"ocurrencias"`
	Primera     string   `yaml:"primera"`
	Ultima      string   `yaml:"ultima"`
	Issue       string   `yaml:"issue,omitempty"`
	Publicadas  int      `yaml:"publicadas,omitempty"`
	PublicadoEl string   `yaml:"publicado_el,omitempty"`
}

// Nuevo arma un incidente ya anonimizado (mensaje y origen pasan por Sanitizar antes) y le pone su firma.
func Nuevo(tipo, origen, mensaje string, traza []string, version string) Incidente {
	i := Incidente{Formato: Formato, Tipo: tipo, Origen: origen, Mensaje: mensaje, Traza: traza, Version: version,
		SO: runtime.GOOS + "/" + runtime.GOARCH}
	i.Firma = firma(i)
	return i
}

var digitos = regexp.MustCompile(`[0-9]+`)

// firma identifica el defecto, no la ocurrencia: tipo, origen, el mensaje sin números y las tres primeras funciones de
// la traza (sin archivo ni línea, que cambian entre versiones).
func firma(i Incidente) string {
	partes := []string{i.Tipo, i.Origen, digitos.ReplaceAllString(i.Mensaje, "N")}
	for n, f := range i.Traza {
		if n == 3 {
			break
		}
		partes = append(partes, strings.SplitN(f, " (", 2)[0])
	}
	suma := sha256.Sum256([]byte(strings.Join(partes, "|")))
	return hex.EncodeToString(suma[:])[:12]
}

// Activo dice si se registran incidentes. QDD_INCIDENTES=off los apaga; en las pruebas solo con un QDD_HOME explícito,
// para que una prueba nunca escriba en la carpeta real.
func Activo() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(ApagadoEnv))) {
	case "off", "0", "no", "false":
		return false
	}
	if enPruebas() && strings.TrimSpace(os.Getenv(HomeEnv)) == "" {
		return false
	}
	return true
}

// enPruebas reconoce un binario de go test por su nombre, sin importar el paquete testing en el producto.
func enPruebas() bool {
	nombre := filepath.Base(os.Args[0])
	return strings.HasSuffix(nombre, ".test") || strings.HasSuffix(nombre, ".test.exe")
}

// Dir es el registro: <QDD_HOME>/incidentes, fuera de todo proyecto.
func Dir() (string, error) {
	if h := strings.TrimSpace(os.Getenv(HomeEnv)); h != "" {
		return filepath.Join(h, "incidentes"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "share", "qdd", "incidentes"), nil
}

// Registrar guarda el incidente o, si su firma ya estaba, le suma una ocurrencia.
func Registrar(dir string, i Incidente, now time.Time) (Incidente, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return i, err
	}
	ruta := filepath.Join(dir, i.Firma+".yaml")
	cuando := now.UTC().Format(time.RFC3339)
	prev, err := leerUno(ruta)
	if err != nil {
		i.Ocurrencias, i.Primera, i.Ultima = 1, cuando, cuando
		return i, guardar(ruta, i)
	}
	prev.Ocurrencias++
	prev.Ultima, prev.Version, prev.SO = cuando, i.Version, i.SO
	return prev, guardar(ruta, prev)
}

// Leer devuelve los incidentes del registro, del más antiguo al más nuevo; el error nombra los ilegibles, que
// ApartarIlegibles deja a un lado.
func Leer(dir string) ([]Incidente, error) {
	rutas, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	var out []Incidente
	var errs []error
	for _, r := range rutas {
		i, err := leerUno(r)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s ilegible: %w", filepath.Base(r), err))
			continue
		}
		out = append(out, i)
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Primera < out[b].Primera })
	return out, errors.Join(errs...)
}

// ApartarIlegibles renombra cada registro ilegible a <archivo>.ilegible: deja de estorbar sin perderse.
func ApartarIlegibles(dir string) ([]string, error) {
	rutas, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	var hecho []string
	for _, r := range rutas {
		if _, err := leerUno(r); err == nil {
			continue
		}
		if err := os.Rename(r, r+".ilegible"); err != nil {
			return hecho, err
		}
		hecho = append(hecho, "apartado "+filepath.Base(r)+".ilegible")
	}
	return hecho, nil
}

func leerUno(ruta string) (Incidente, error) {
	var i Incidente
	data, err := os.ReadFile(ruta)
	if err != nil {
		return i, err
	}
	if err := yaml.Unmarshal(data, &i); err != nil {
		return i, err
	}
	if i.Firma == "" || i.Formato == "" {
		return i, errors.New("sin firma o sin formato")
	}
	return i, nil
}

func guardar(ruta string, i Incidente) error {
	data, err := yaml.Marshal(i)
	if err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

// Pendiente: nunca publicado, o con ocurrencias que el issue todavía no cuenta.
func (i Incidente) Pendiente() bool {
	return i.Issue == "" || i.Publicadas < i.Ocurrencias
}
