// Package registro es el registro de proyectos de la máquina (RFC 0013, rc.1): <QDD_HOME>/proyectos.json, los
// proyectos que qdd remoto publica. No depende de pkg/consola para que el ciclo del panel lo pueda anotar.
package registro

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/candado"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Se anota en qdd init y en cada qdd consola ciclo (cada mensaje de la persona): sin escribir si el proyecto ya está y
// se vio hace menos de Refresco, porque el ciclo no debe costar más que leer un archivo. Escribir es atómico (tmp y
// rename) bajo un bloqueo (pkg/candado), porque varias sesiones anotan a la vez.

const (
	// Formato es la versión del archivo.
	Formato = 1
	// Refresco: un proyecto visto hace menos que esto no se vuelve a escribir.
	Refresco = time.Hour
	archivo  = "proyectos.json"
)

// Proyecto es un proyecto de la máquina. Ruta es estado.ClaveLocal (un worktree es su checkout principal); Clave, el
// nombre de sus archivos locales (RutaLocal sin carpeta ni extensión): la misma en todo QDD y no expone la ruta.
type Proyecto struct {
	Ruta  string    `json:"ruta"`
	Clave string    `json:"clave"`
	Visto time.Time `json:"visto"`
}

// Registro es el contenido de <QDD_HOME>/proyectos.json.
type Registro struct {
	Formato   int        `json:"formato"`
	Proyectos []Proyecto `json:"proyectos"`
}

// Ruta es <QDD_HOME>/proyectos.json.
func Ruta() (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, archivo), nil
}

// Existe dice si la máquina ya tiene registro (doctor revisa el grupo solo entonces).
func Existe() bool {
	ruta, err := Ruta()
	if err != nil {
		return false
	}
	_, err = os.Stat(ruta)
	return err == nil
}

// ClaveDe es la clave local del proyecto: la que usan la cola de pedidos y la actividad.
func ClaveDe(root string) (string, error) {
	local, err := estado.RutaLocal(root, "registro")
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(filepath.Base(local), filepath.Ext(local)), nil
}

// Leer devuelve el registro, el más reciente primero. Sin archivo es un registro vacío; uno ilegible es un error (no
// se trata como vacío: anotar encima perdería los proyectos).
func Leer() (Registro, error) {
	ruta, err := Ruta()
	if err != nil {
		return Registro{}, err
	}
	return leerDe(ruta)
}

func leerDe(ruta string) (Registro, error) {
	b, err := os.ReadFile(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return Registro{Formato: Formato}, nil
	}
	if err != nil {
		return Registro{}, err
	}
	var reg Registro
	if err := json.Unmarshal(b, &reg); err != nil {
		return Registro{}, fmt.Errorf("%s ilegible: %w (qdd doctor --fix lo aparta)", ruta, err)
	}
	if reg.Formato > Formato {
		return Registro{}, fmt.Errorf("%s es de un qdd más nuevo (formato %d): actualiza qdd", ruta, reg.Formato)
	}
	reg.Formato = Formato
	ordenar(reg.Proyectos)
	return reg, nil
}

func ordenar(ps []Proyecto) {
	sort.SliceStable(ps, func(i, j int) bool { return ps[i].Visto.After(ps[j].Visto) })
}

// Anotar deja el proyecto de root en el registro, visto ahora. Si ya está y se vio hace menos de Refresco, solo lee.
func Anotar(root string, now time.Time) error {
	ruta, err := Ruta()
	if err != nil {
		return err
	}
	clave := estado.ClaveLocal(RutaReal(root))
	reg, err := leerDe(ruta)
	if err != nil {
		return err
	}
	if p, ok := buscar(reg, clave); ok && now.Sub(p.Visto) < Refresco && !now.Before(p.Visto) {
		return nil
	}
	nombre, err := ClaveDe(RutaReal(root))
	if err != nil {
		return err
	}
	return cambiar(ruta, func(reg Registro) Registro {
		var resto []Proyecto
		for _, p := range reg.Proyectos {
			if p.Ruta != clave && RutaReal(p.Ruta) != clave {
				resto = append(resto, p)
			}
		}
		reg.Proyectos = append(resto, Proyecto{Ruta: clave, Clave: nombre, Visto: now})
		return reg
	})
}

// RutaReal (MEJ-077): una ruta que llegó codificada como URL (Sistema%20de%20gesti%C3%B3n, de un file:// del editor) y
// cuya carpeta no existe así, pero sí decodificada, es la decodificada. Una carpeta real con % queda tal cual.
func RutaReal(ruta string) string {
	if !strings.Contains(ruta, "%") || existe(ruta) {
		return ruta
	}
	dec, err := url.PathUnescape(ruta)
	if err != nil || !existe(dec) {
		return ruta
	}
	return dec
}

// NombreDe es el nombre del proyecto que ven el panel y el teléfono: su carpeta, decodificada si llegó codificada como
// URL y no existe así (MEJ-077). Lo que no se puede decodificar queda tal cual.
func NombreDe(ruta string) string {
	base := filepath.Base(ruta)
	if !strings.Contains(base, "%") || existe(ruta) {
		return base
	}
	dec, err := url.PathUnescape(base)
	if err != nil {
		return base
	}
	return dec
}

func existe(ruta string) bool {
	_, err := os.Stat(ruta)
	return err == nil
}

func buscar(reg Registro, ruta string) (Proyecto, bool) {
	for _, p := range reg.Proyectos {
		if p.Ruta == ruta {
			return p, true
		}
	}
	return Proyecto{}, false
}

// Muertas son las rutas del registro que ya no existen (REM-01).
func Muertas(reg Registro) []string {
	var out []string
	for _, p := range reg.Proyectos {
		if info, err := os.Stat(p.Ruta); err != nil || !info.IsDir() {
			out = append(out, p.Ruta)
		}
	}
	return out
}

// QuitarMuertas saca del registro las rutas que ya no existen (qdd doctor --fix) y dice cuáles quitó.
func QuitarMuertas() ([]string, error) {
	ruta, err := Ruta()
	if err != nil {
		return nil, err
	}
	var quitadas []string
	err = cambiar(ruta, func(reg Registro) Registro {
		quitadas = Muertas(reg)
		muerta := map[string]bool{}
		for _, m := range quitadas {
			muerta[m] = true
		}
		var vivas []Proyecto
		for _, p := range reg.Proyectos {
			if !muerta[p.Ruta] {
				vivas = append(vivas, p)
			}
		}
		reg.Proyectos = vivas
		return reg
	})
	return quitadas, err
}

// Apartar deja un registro ilegible como proyectos.json.ilegible (no se borra) para que se vuelva a anotar.
func Apartar() (string, error) {
	ruta, err := Ruta()
	if err != nil {
		return "", err
	}
	soltar, err := candado.Tomar(ruta+".lock", 2*time.Second, "el registro de proyectos")
	if err != nil {
		return "", err
	}
	defer soltar()
	if _, err := leerDe(ruta); err == nil {
		return "", nil
	}
	return ruta + ".ilegible", os.Rename(ruta, ruta+".ilegible")
}

// cambiar lee, cambia y guarda el registro con el bloqueo tomado; uno ilegible no se toca.
func cambiar(ruta string, f func(Registro) Registro) error {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	soltar, err := candado.Tomar(ruta+".lock", 2*time.Second, "el registro de proyectos")
	if err != nil {
		return err
	}
	defer soltar()
	reg, err := leerDe(ruta)
	if err != nil {
		return err
	}
	reg = f(reg)
	reg.Formato = Formato
	ordenar(reg.Proyectos)
	if reg.Proyectos == nil {
		reg.Proyectos = []Proyecto{}
	}
	b, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.%d.tmp", ruta, os.Getpid())
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}
