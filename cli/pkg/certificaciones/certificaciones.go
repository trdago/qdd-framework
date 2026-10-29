// Package certificaciones lee las certificaciones de un proyecto y del
// catálogo que trae QDD. El dashboard las tomaba del grafo, donde el catálogo
// pisaba a la del proyecto con el mismo id, y daba por cumplida toda
// certificación sin estado (FND-060).
//
// Una certificación del catálogo (.qdd/core/certification) es un estándar
// disponible para adoptar: su status dice que el estándar está publicado, no
// que el proyecto lo cumpla. Una del proyecto (.qdd/project/certification)
// declara su estado en el YAML.
package certificaciones

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Estado normalizado de una certificación del proyecto.
type Estado string

const (
	Cumple    Estado = "cumple"
	NoCumple  Estado = "no_cumple"
	Pendiente Estado = "pendiente"
	// SinEstado: el archivo no dice su estado, o dice uno que QDD no reconoce.
	// No cuenta como cumplida: lo decide una persona.
	SinEstado Estado = ""
)

var sinonimos = map[string]Estado{
	"certified": Cumple, "certificada": Cumple, "passed": Cumple, "pass": Cumple,
	"approved": Cumple, "aprobada": Cumple, "cumple": Cumple,
	"failed": NoCumple, "fail": NoCumple, "rejected": NoCumple, "rechazada": NoCumple, "no_cumple": NoCumple,
	"pending": Pendiente, "pendiente": Pendiente, "draft": Pendiente, "borrador": Pendiente,
	"in_progress": Pendiente, "in-progress": Pendiente,
}

// Normalizar traduce el status escrito en el archivo a un Estado.
func Normalizar(status string) Estado {
	return sinonimos[strings.ToLower(strings.TrimSpace(status))]
}

// Origen dice de dónde viene una certificación.
type Origen string

const (
	Proyecto Origen = "proyecto"
	Catalogo Origen = "catalogo"
)

// Certificacion es lo que QDD lee de un archivo de certificación.
type Certificacion struct {
	Archivo        string
	ID             string
	Nombre         string
	Version        string
	Origen         Origen
	Estado         Estado
	EstadoOriginal string
	Datos          map[string]interface{}
}

func carpeta(cwd, origen string) string {
	return filepath.Join(cwd, ".qdd", origen, "certification")
}

// Leer devuelve las certificaciones del proyecto y las del catálogo que el
// proyecto no adoptó, cada una una sola vez y ordenadas por id. Sin carpetas
// devuelve una lista vacía. Un archivo que no se puede leer como YAML se omite.
func Leer(cwd string) ([]Certificacion, error) {
	proyecto, err := leerCarpeta(carpeta(cwd, "project"), Proyecto)
	if err != nil {
		return nil, err
	}
	catalogo, err := leerCarpeta(carpeta(cwd, "core"), Catalogo)
	if err != nil {
		return nil, err
	}
	adoptadas := map[string]bool{}
	for _, c := range proyecto {
		adoptadas[c.ID] = true
	}
	out := proyecto
	for _, c := range catalogo {
		if !adoptadas[c.ID] {
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func leerCarpeta(dir string, origen Origen) ([]Certificacion, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Certificacion
	for _, e := range entries {
		c, ok := leerArchivo(dir, e, origen)
		if ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func leerArchivo(dir string, e os.DirEntry, origen Origen) (Certificacion, bool) {
	if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
		return Certificacion{}, false
	}
	data, err := os.ReadFile(filepath.Join(dir, e.Name()))
	if err != nil {
		return Certificacion{}, false
	}
	var datos map[string]interface{}
	if yaml.Unmarshal(data, &datos) != nil {
		return Certificacion{}, false
	}
	id := primerTexto(datos, "id")
	if id == "" {
		id = strings.TrimSuffix(e.Name(), ".yaml")
	}
	nombre := primerTexto(datos, "title", "name")
	if nombre == "" {
		nombre = id
	}
	c := Certificacion{
		Archivo: e.Name(),
		ID:      id,
		Nombre:  nombre,
		Version: primerTexto(datos, "version"),
		Origen:  origen,
		Datos:   datos,
	}
	if origen == Proyecto {
		c.EstadoOriginal = primerTexto(datos, "status")
		c.Estado = Normalizar(c.EstadoOriginal)
	}
	return c, true
}

func primerTexto(datos map[string]interface{}, claves ...string) string {
	for _, k := range claves {
		v, ok := datos[k]
		if ok && v != nil {
			return strings.TrimSpace(fmt.Sprintf("%v", v))
		}
	}
	return ""
}

// Conteo resume las certificaciones. Las del catálogo cuentan como
// disponibles, nunca como cumplidas.
type Conteo struct {
	Cumplen     int `json:"cumplen"`
	NoCumplen   int `json:"no_cumplen"`
	Pendientes  int `json:"pendientes"`
	SinEstado   int `json:"sin_estado"`
	Disponibles int `json:"disponibles"`
}

// Adoptadas son las certificaciones del proyecto.
func (c Conteo) Adoptadas() int {
	return c.Cumplen + c.PorCumplir()
}

// PorCumplir son las del proyecto que no declaran que cumplen: las que no
// cumplen, las pendientes y las que no dicen su estado.
func (c Conteo) PorCumplir() int {
	return c.NoCumplen + c.Pendientes + c.SinEstado
}

// ContarEn lee y resume las certificaciones del proyecto y del catálogo.
func ContarEn(cwd string) Conteo {
	cs, _ := Leer(cwd)
	return Contar(cs)
}

// Contar resume una lista de certificaciones.
func Contar(cs []Certificacion) Conteo {
	var c Conteo
	for _, cert := range cs {
		sumar(&c, cert)
	}
	return c
}

func sumar(c *Conteo, cert Certificacion) {
	if cert.Origen == Catalogo {
		c.Disponibles++
		return
	}
	switch cert.Estado {
	case Cumple:
		c.Cumplen++
	case NoCumple:
		c.NoCumplen++
	case Pendiente:
		c.Pendientes++
	default:
		c.SinEstado++
	}
}

// PorDecidir devuelve las certificaciones del proyecto sin un estado
// reconocido. doctor las informa: el estado lo decide una persona.
func PorDecidir(cwd string) []Certificacion {
	cs, _ := leerCarpeta(carpeta(cwd, "project"), Proyecto)
	var out []Certificacion
	for _, c := range cs {
		if c.Estado == SinEstado {
			out = append(out, c)
		}
	}
	return out
}
