package rendimiento

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
)

// Estados de una medida.
const (
	EstadoOK       = "ok"
	EstadoSobre    = "sobre"     // pasó su presupuesto
	EstadoFalla    = "falla"     // dejó procesos: no es lento, está mal
	EstadoSinMedir = "sin_medir" // no se pudo medir en este equipo: nunca un cero inventado
)

// Medida es un número con su unidad, su presupuesto y cómo quedó. Valor nil es «sin medir».
type Medida struct {
	Nombre      string   `json:"nombre"`
	Valor       *float64 `json:"valor"`
	Unidad      string   `json:"unidad"`
	Presupuesto float64  `json:"presupuesto"`
	Estado      string   `json:"estado"`
	Que         string   `json:"que"`
}

// Informe es una corrida de qdd rendimiento en un equipo.
type Informe struct {
	Version      string    `json:"version"`
	SO           string    `json:"so"`
	Arquitectura string    `json:"arquitectura"`
	Nucleos      int       `json:"nucleos"`
	Cuando       time.Time `json:"cuando"`
	Segundos     int       `json:"segundos"`
	Medidas      []Medida  `json:"medidas"`
	Restos       []Proceso `json:"restos,omitempty"` // lo que quedó vivo al terminar; la medición lo cerró
}

// Falla dice si algo pasó su presupuesto o dejó procesos.
func (i Informe) Falla() bool {
	for _, m := range i.Medidas {
		if m.Estado == EstadoFalla || m.Estado == EstadoSobre {
			return true
		}
	}
	return false
}

// Resumen es una línea: cuántas medidas quedaron en cada estado.
func (i Informe) Resumen() string {
	cuenta := map[string]int{}
	for _, m := range i.Medidas {
		cuenta[m.Estado]++
	}
	partes := []string{fmt.Sprintf("%d ok", cuenta[EstadoOK])}
	for _, e := range []string{EstadoSobre, EstadoFalla, EstadoSinMedir} {
		if cuenta[e] == 0 {
			continue
		}
		partes = append(partes, fmt.Sprintf("%d %s", cuenta[e], strings.ReplaceAll(e, "_", " ")))
	}
	return strings.Join(partes, " · ")
}

func carpeta() (string, error) {
	home, err := blueprints.Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "rendimiento"), nil
}

func rutaDe(version string) (string, error) {
	if version == "" || strings.ContainsAny(version, `/\`) || strings.Contains(version, "..") {
		return "", fmt.Errorf("versión inválida: %q", version)
	}
	dir, err := carpeta()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, version+".json"), nil
}

// Guardar deja el informe de esa versión en la carpeta de qdd del usuario; uno nuevo reemplaza al anterior.
func Guardar(inf Informe) error {
	ruta, err := rutaDe(inf.Version)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(inf, "", "  ")
	if err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, ruta)
}

// ErrSinMedir: esa versión no se midió en este equipo.
var ErrSinMedir = errors.New("sin medir")

// Ultimo es el informe guardado de esa versión.
func Ultimo(version string) (Informe, error) {
	ruta, err := rutaDe(version)
	if err != nil {
		return Informe{}, err
	}
	b, err := os.ReadFile(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return Informe{}, ErrSinMedir
	}
	if err != nil {
		return Informe{}, err
	}
	var inf Informe
	if err := json.Unmarshal(b, &inf); err != nil {
		return Informe{}, fmt.Errorf("informe ilegible: %w", err)
	}
	return inf, nil
}
