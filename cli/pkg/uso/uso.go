// Package uso mide cuánto se usa de verdad cada versión de qdd (pedido del usuario, 2026-10-30): una versión que no es
// un fix se publica recién después de horas reales de uso medidas (regla VER-05 del gate). Lo medido vive fuera de los
// proyectos, en <QDD_HOME>/uso/<versión>.json; 'qdd uso evidencia' lo deja en el proyecto que se va a publicar.
package uso

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'

// Ventana: dos eventos separados por menos que esto son uso continuo y se cuenta el tiempo entre ellos. Un evento
// aislado no suma: el uso se mide, nunca se estima.
const Ventana = 5 * time.Minute

// RutaEvidencia es la carpeta del proyecto donde 'qdd uso evidencia' deja las horas medidas de una versión.
const RutaEvidencia = ".qdd/project/evidence/uso"

var versionValida = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?$`)

// Uso es lo medido de una versión.
type Uso struct {
	Formato   int              `json:"formato"`
	Version   string           `json:"version"`
	Desde     string           `json:"desde"`
	Ultimo    string           `json:"ultimo"`
	Segundos  int64            `json:"segundos"`
	Proyectos map[string]int64 `json:"proyectos"` // carpeta del proyecto → segundos
}

// Duracion es lo medido como time.Duration.
func (u Uso) Duracion() time.Duration { return time.Duration(u.Segundos) * time.Second }

func normalizar(version string) (string, error) {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	if !versionValida.MatchString(v) {
		return "", fmt.Errorf("versión %q: no es SemVer", version)
	}
	return v, nil
}

// nucleo es MAJOR.MINOR.PATCH sin la parte de candidata (2.8.0-rc.1 → 2.8.0).
func nucleo(v string) string {
	return strings.SplitN(v, "-", 2)[0]
}

// Carpeta es <QDD_HOME>/uso.
func Carpeta() (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "uso"), nil
}

func ruta(version string) (string, error) {
	dir, err := Carpeta()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, version+".json"), nil
}

// Registrar anota un evento de uso real de la versión en el proyecto. Una compilación sin versión («dev») no se mide.
func Registrar(version, proyecto string, now time.Time) error {
	if strings.TrimSpace(version) == "" || version == "dev" {
		return nil
	}
	v, err := normalizar(version)
	if err != nil {
		return err
	}
	dest, err := ruta(v)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return err
	}
	soltar, err := bloquear(dest + ".lock")
	if err != nil {
		return err
	}
	defer soltar()
	u, err := Leer(v)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return guardar(dest, sumar(u, v, proyecto, now))
}

// sumar agrega el tiempo desde el evento anterior si es uso continuo; un reloj que vuelve atrás no suma ni retrocede.
func sumar(u Uso, v, proyecto string, now time.Time) Uso {
	hora := now.UTC().Format(time.RFC3339)
	u.Formato, u.Version = 1, v
	if u.Proyectos == nil {
		u.Proyectos = map[string]int64{}
	}
	if u.Desde == "" {
		u.Desde = hora
	}
	ultimo, err := time.Parse(time.RFC3339, u.Ultimo)
	if err != nil {
		u.Ultimo = hora
		return u
	}
	gap := now.Sub(ultimo)
	if gap < 0 {
		return u
	}
	u.Ultimo = hora
	if gap > Ventana {
		return u
	}
	s := int64(gap / time.Second)
	u.Segundos += s
	u.Proyectos[proyecto] += s
	return u
}

// Leer devuelve lo medido de una versión; os.ErrNotExist si nunca se usó.
func Leer(version string) (Uso, error) {
	v, err := normalizar(version)
	if err != nil {
		return Uso{}, err
	}
	dest, err := ruta(v)
	if err != nil {
		return Uso{}, err
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		return Uso{}, err
	}
	var u Uso
	if err := json.Unmarshal(data, &u); err != nil {
		return Uso{}, fmt.Errorf("%s ilegible: %w", dest, err)
	}
	return u, nil
}

// Todas devuelve lo medido de cada versión, ordenado por versión. Los archivos ilegibles van en ilegibles.
func Todas() (usos []Uso, ilegibles []string) {
	dir, err := Carpeta()
	if err != nil {
		return nil, nil
	}
	nombres, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(nombres)
	for _, n := range nombres {
		u, err := Leer(strings.TrimSuffix(filepath.Base(n), ".json"))
		if err != nil {
			ilegibles = append(ilegibles, n)
			continue
		}
		usos = append(usos, u)
	}
	return usos, ilegibles
}

// Total suma lo medido de la versión y de sus candidatas (2.8.0 junta 2.8.0-rc.1, 2.8.0-rc.2 y 2.8.0).
func Total(version string) (time.Duration, []Uso) {
	v, err := normalizar(version)
	if err != nil {
		return 0, nil
	}
	var total time.Duration
	var de []Uso
	usos, _ := Todas()
	for _, u := range usos {
		if nucleo(u.Version) != nucleo(v) {
			continue
		}
		total += u.Duracion()
		de = append(de, u)
	}
	return total, de
}

// Registro es lo que queda en el proyecto: las horas medidas de una versión, de dónde salen y en qué proyectos.
type Registro struct {
	Version     string             `yaml:"version"`
	Horas       float64            `yaml:"horas"`
	MedidoHasta string             `yaml:"medido_hasta"`
	Versiones   []string           `yaml:"versiones"`
	Proyectos   map[string]float64 `yaml:"proyectos"`
}

func horas(d time.Duration) float64 { return math.Round(d.Hours()*100) / 100 }

// Evidencia escribe <root>/.qdd/project/evidence/uso/v<núcleo>.yaml con lo medido de la versión y sus candidatas.
func Evidencia(root, version string, now time.Time) (string, Registro, error) {
	v, err := normalizar(version)
	if err != nil {
		return "", Registro{}, err
	}
	n := nucleo(v)
	total, de := Total(n)
	if total == 0 {
		return "", Registro{}, fmt.Errorf("no hay uso medido de %s ni de sus candidatas en esta máquina", n)
	}
	ev := Registro{Version: n, Horas: horas(total), MedidoHasta: now.UTC().Format(time.RFC3339), Proyectos: map[string]float64{}}
	porProyecto := map[string]time.Duration{}
	for _, u := range de {
		ev.Versiones = append(ev.Versiones, fmt.Sprintf("%s: %.2f h", u.Version, horas(u.Duracion())))
		for p, s := range u.Proyectos {
			porProyecto[p] += time.Duration(s) * time.Second
		}
	}
	for p, d := range porProyecto {
		ev.Proyectos[p] = horas(d)
	}
	data, err := yaml.Marshal(ev)
	if err != nil {
		return "", Registro{}, err
	}
	dest := filepath.Join(root, RutaEvidencia, "v"+n+".yaml")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", Registro{}, err
	}
	cabecera := "# Uso real medido de qdd " + n + " (qdd uso evidencia). El gate lo lee para publicar (VER-05).\n"
	return dest, ev, os.WriteFile(dest, append([]byte(cabecera), data...), 0o644)
}

// LeerEvidencia lee la evidencia de uso de una versión dejada en el proyecto.
func LeerEvidencia(root, version string) (Registro, error) {
	v, err := normalizar(version)
	if err != nil {
		return Registro{}, err
	}
	data, err := os.ReadFile(filepath.Join(root, RutaEvidencia, "v"+nucleo(v)+".yaml"))
	if err != nil {
		return Registro{}, err
	}
	var ev Registro
	if err := yaml.Unmarshal(data, &ev); err != nil {
		return Registro{}, err
	}
	return ev, nil
}

func guardar(dest string, u Uso) error {
	data, err := json.MarshalIndent(u, "", "  ")
	if err != nil {
		return err
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

// bloquear toma un archivo de bloqueo exclusivo; uno de más de 10 s es de un proceso que murió y se descarta.
func bloquear(lock string) (func(), error) {
	for intento := 0; intento < 200; intento++ {
		f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			f.Close()
			return func() { os.Remove(lock) }, nil
		}
		if info, errStat := os.Stat(lock); errStat == nil && time.Since(info.ModTime()) > 10*time.Second {
			os.Remove(lock)
			continue
		}
		time.Sleep(5 * time.Millisecond)
	}
	return nil, fmt.Errorf("el registro de uso está ocupado (%s)", lock)
}

// ApartarIlegibles renombra cada registro ilegible a <archivo>.ilegible: deja de estorbar sin perderse.
func ApartarIlegibles() ([]string, error) {
	_, ilegibles := Todas()
	var hecho []string
	for _, n := range ilegibles {
		if err := os.Rename(n, n+".ilegible"); err != nil {
			return hecho, err
		}
		hecho = append(hecho, "apartado "+n+".ilegible")
	}
	return hecho, nil
}
