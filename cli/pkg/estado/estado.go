// Package estado implementa el estado por frente en archivo (RFC 0006, blueprint estado-por-tareas, antes estado-por-frente; destino «archivo»):
// la tabla de frentes × etapas en .qdd/project/estado.yaml, su lector automático y sus vistas para la consola.
package estado

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'

// Ruta del archivo de estado, relativa a la raíz del proyecto.
const Ruta = ".qdd/project/estado.yaml"

// Formato vigente del archivo.
const Formato = 1

// Estados cerrados de una celda.
const (
	Hecho    = "✅"
	Espera   = "⏳"
	Falla    = "❌"
	NoAplica = "—"
)

// Estados de un frente.
const (
	EstadoAbierto    = "abierto"
	EstadoCerrado    = "cerrado"
	EstadoDescartado = "descartado"
)

// RecienCerrado: cuánto tiempo sigue a la vista un frente cerrado con el alcance «abiertos».
const RecienCerrado = 7 * 24 * time.Hour

// Celda es el valor de un frente en una etapa.
type Celda struct {
	Estado             string `yaml:"estado"`
	Evidencia          string `yaml:"evidencia,omitempty"`
	Bloqueo            string `yaml:"bloqueo,omitempty"`
	DeQuien            string `yaml:"de_quien,omitempty"`
	Falla              string `yaml:"falla,omitempty"`
	RequiereAprobacion string `yaml:"requiere_aprobacion,omitempty"`
	Aprobado           bool   `yaml:"aprobado,omitempty"`
}

// Frente es una fila de la tabla.
type Frente struct {
	ID           string           `yaml:"id"`
	Nombre       string           `yaml:"nombre"`
	Estado       string           `yaml:"estado"`
	CerradoEl    string           `yaml:"cerrado_el,omitempty"`
	MotivoCierre string           `yaml:"motivo_cierre,omitempty"`
	Celdas       map[string]Celda `yaml:"celdas"`
}

// Estado es el archivo completo.
type Estado struct {
	Formato     int      `yaml:"formato"`
	Etapas      []string `yaml:"etapas"`
	Actualizado string   `yaml:"actualizado,omitempty"`
	Frentes     []Frente `yaml:"frentes"`
}

// NormalizarEstado acepta el emoji o un alias escribible (ok, espera, falla, na) y devuelve el estado cerrado; "" si no
// es ninguno.
func NormalizarEstado(s string) string {
	v := strings.ToLower(strings.TrimSpace(strings.ReplaceAll(s, "️", "")))
	switch v {
	case Hecho, "ok", "hecho":
		return Hecho
	case Espera, "espera", "curso", "en-curso":
		return Espera
	case Falla, "falla", "error", "x":
		return Falla
	case NoAplica, "na", "-", "n/a", "no-aplica":
		return NoAplica
	}
	return ""
}

// Parse lee el archivo de estado.
func Parse(data []byte) (Estado, error) {
	var e Estado
	if err := yaml.Unmarshal(data, &e); err != nil {
		return Estado{}, fmt.Errorf("estado.yaml ilegible: %w", err)
	}
	if e.Formato != Formato {
		return Estado{}, fmt.Errorf("estado.yaml: formato %d no soportado (se espera %d)", e.Formato, Formato)
	}
	return e, nil
}

// Load lee el archivo de estado de la raíz del proyecto.
func Load(root string) (Estado, error) {
	data, err := os.ReadFile(filepath.Join(root, Ruta))
	if err != nil {
		return Estado{}, err
	}
	return Parse(data)
}

// Save escribe el archivo de forma atómica (temporal + rename): nunca queda a medias.
func Save(root string, e Estado) error {
	data, err := yaml.Marshal(e)
	if err != nil {
		return err
	}
	dest := filepath.Join(root, Ruta)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".estado-*.yaml")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append([]byte("# Estado por frente (RFC 0006). Se edita con 'qdd estado set' y 'qdd estado cerrar'.\n"), data...)); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dest)
}

// Abiertos devuelve los frentes abiertos.
func (e Estado) Abiertos() []Frente {
	var out []Frente
	for _, f := range e.Frentes {
		if f.Estado == EstadoAbierto {
			out = append(out, f)
		}
	}
	return out
}

// Alcance de la tabla (pregunta «alcance» del blueprint).
type Alcance string

const (
	AlcanceAbiertos Alcance = "abiertos"
	AlcanceTodos    Alcance = "todos"
)

// EnAlcance: los abiertos siempre; los cerrados, con «todos» o si se cerraron hace menos de RecienCerrado.
func (f Frente) EnAlcance(alcance Alcance, now time.Time) bool {
	if f.Estado == EstadoAbierto || alcance == AlcanceTodos {
		return true
	}
	t, err := time.Parse(time.RFC3339, f.CerradoEl)
	if err != nil {
		return false
	}
	return now.Sub(t) < RecienCerrado
}

func (f Frente) tiene(estado string) bool {
	for _, c := range f.Celdas {
		if NormalizarEstado(c.Estado) == estado {
			return true
		}
	}
	return false
}

func (e Estado) buscar(id string) int {
	for i, f := range e.Frentes {
		if f.ID == id {
			return i
		}
	}
	return -1
}

func (e Estado) tieneEtapa(etapa string) bool {
	for _, s := range e.Etapas {
		if s == etapa {
			return true
		}
	}
	return false
}
