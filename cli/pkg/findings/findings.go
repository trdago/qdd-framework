// Package findings es el único lector de .qdd/project/findings. El dashboard,
// qdd_score, qdd_sprint, report y evolution interpretaban el estado de un
// finding cada uno a su manera y daban conteos distintos (FND-056).
package findings

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Estado normalizado de un finding.
type Estado string

const (
	Abierto    Estado = "open"
	Resuelto   Estado = "resolved"
	Descartado Estado = "wontfix"
	// SinEstado: el archivo no dice su estado, o dice uno que QDD no reconoce.
	// No se cuenta como abierto ni como resuelto: lo decide una persona.
	SinEstado Estado = ""
)

var sinonimos = map[string]Estado{
	"open": Abierto, "abierto": Abierto, "pending": Abierto, "pendiente": Abierto,
	"in-progress": Abierto, "in_progress": Abierto, "en-progreso": Abierto,
	"resolved": Resuelto, "resuelto": Resuelto, "closed": Resuelto, "cerrado": Resuelto, "fixed": Resuelto,
	"wontfix": Descartado, "descartado": Descartado,
}

// Normalizar traduce el status escrito en el archivo a un Estado.
func Normalizar(status string) Estado {
	return sinonimos[strings.ToLower(strings.TrimSpace(status))]
}

// Finding es lo que QDD lee de un archivo de finding.
type Finding struct {
	Archivo        string
	ID             string
	Titulo         string
	Descripcion    string
	Impacto        string
	Severidad      string
	Estado         Estado
	EstadoOriginal string
}

type archivo struct {
	ID          string `yaml:"id"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Impact      string `yaml:"impact"`
	Severity    string `yaml:"severity"`
	Status      string `yaml:"status"`
}

// Dir devuelve la carpeta de findings del proyecto.
func Dir(cwd string) string {
	return filepath.Join(cwd, ".qdd", "project", "findings")
}

// Leer devuelve los findings del proyecto ordenados por archivo. Sin carpeta
// devuelve una lista vacía. Un archivo que no se puede leer como YAML se omite.
func Leer(cwd string) ([]Finding, error) {
	entries, err := os.ReadDir(Dir(cwd))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, e := range entries {
		f, ok := leerArchivo(cwd, e)
		if ok {
			out = append(out, f)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Archivo < out[j].Archivo })
	return out, nil
}

func leerArchivo(cwd string, e os.DirEntry) (Finding, bool) {
	if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
		return Finding{}, false
	}
	data, err := os.ReadFile(filepath.Join(Dir(cwd), e.Name()))
	if err != nil {
		return Finding{}, false
	}
	var a archivo
	if yaml.Unmarshal(data, &a) != nil {
		return Finding{}, false
	}
	return Finding{
		Archivo:        e.Name(),
		ID:             a.ID,
		Titulo:         a.Title,
		Descripcion:    a.Description,
		Impacto:        a.Impact,
		Severidad:      a.Severity,
		Estado:         Normalizar(a.Status),
		EstadoOriginal: a.Status,
	}, true
}

// Conteo resume los findings por estado.
type Conteo struct {
	Abiertos    int `json:"abiertos"`
	Resueltos   int `json:"resueltos"`
	Descartados int `json:"descartados"`
	SinEstado   int `json:"sin_estado"`
}

// Contar resume una lista de findings.
func Contar(fs []Finding) Conteo {
	var c Conteo
	for _, f := range fs {
		sumar(&c, f.Estado)
	}
	return c
}

func sumar(c *Conteo, e Estado) {
	switch e {
	case Abierto:
		c.Abiertos++
	case Resuelto:
		c.Resueltos++
	case Descartado:
		c.Descartados++
	default:
		c.SinEstado++
	}
}

// ContarEn lee y resume los findings del proyecto.
func ContarEn(cwd string) Conteo {
	fs, _ := Leer(cwd)
	return Contar(fs)
}
