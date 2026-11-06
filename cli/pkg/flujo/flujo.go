// Package flujo guarda y dibuja los flujos del proyecto (RFC 0012, ampliación): nodos y aristas hacia abajo, para que la
// IA y la persona vean la misma forma en vez de describirla en prosa. Cada flujo es un archivo versionado.
package flujo

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// Carpeta guarda un archivo por flujo. Se versiona: estado.yaml no siempre lo está.
	Carpeta = ".qdd/project/flujos"
	Formato = 1

	TipoPaso      = "paso"      // se dibuja como caja
	TipoResultado = "resultado" // se dibuja como círculo: un inicio, un fin o un resultado

	maxNodos = 200
)

var idValido = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Nodo es un paso o un resultado. Siguen son los nodos que vienen después; Tarea nombra una tarea del tablero.
type Nodo struct {
	ID          string   `yaml:"id" json:"id"`
	Tipo        string   `yaml:"tipo" json:"tipo"`
	Nombre      string   `yaml:"nombre" json:"nombre"`
	Descripcion string   `yaml:"descripcion,omitempty" json:"descripcion,omitempty"`
	Tarea       string   `yaml:"tarea,omitempty" json:"tarea,omitempty"`
	Siguen      []string `yaml:"siguen,omitempty" json:"siguen,omitempty"`
}

// Flujo empieza en su primer nodo; todos los demás tienen que poder alcanzarse desde ahí.
type Flujo struct {
	Formato     int    `yaml:"formato" json:"formato"`
	ID          string `yaml:"id" json:"id"`
	Nombre      string `yaml:"nombre" json:"nombre"`
	Descripcion string `yaml:"descripcion,omitempty" json:"descripcion,omitempty"`
	Nodos       []Nodo `yaml:"nodos" json:"nodos"`
}

// Resumen es lo que el panel necesita para saber que un flujo existe o cambió, sin dibujarlo.
type Resumen struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Nodos    int    `json:"nodos"`
	Huella   string `json:"huella"`
	Problema string `json:"problema,omitempty"`
}

// Validar lista todo lo que impide usar el flujo.
func Validar(f Flujo) []string {
	var problemas []string
	if f.Formato != Formato {
		problemas = append(problemas, fmt.Sprintf("formato %d: se usa formato: %d", f.Formato, Formato))
	}
	if !idValido.MatchString(f.ID) {
		problemas = append(problemas, fmt.Sprintf("id %q: usa minúsculas, números y guiones", f.ID))
	}
	if strings.TrimSpace(f.Nombre) == "" {
		problemas = append(problemas, "el flujo no tiene nombre")
	}
	if len(f.Nodos) == 0 {
		return append(problemas, "el flujo no tiene ningún nodo")
	}
	if len(f.Nodos) > maxNodos {
		problemas = append(problemas, fmt.Sprintf("%d nodos: el tope es %d", len(f.Nodos), maxNodos))
	}
	return append(problemas, problemasDeNodos(f)...)
}

func problemasDeNodos(f Flujo) []string {
	var problemas []string
	vistos := map[string]bool{}
	for _, n := range f.Nodos {
		problemas = append(problemas, problemasDeNodo(n, vistos)...)
		vistos[n.ID] = true
	}
	for _, n := range f.Nodos {
		for _, s := range n.Siguen {
			if !vistos[s] {
				problemas = append(problemas, fmt.Sprintf("el nodo %q sigue en %q, que no existe", n.ID, s))
			}
		}
	}
	if len(problemas) > 0 {
		return problemas
	}
	alcanzados := alcanzables(f, f.Nodos[0].ID)
	for _, n := range f.Nodos {
		if !alcanzados[n.ID] {
			problemas = append(problemas, fmt.Sprintf("al nodo %q no se llega desde el inicio (%q)", n.ID, f.Nodos[0].ID))
		}
	}
	return problemas
}

func problemasDeNodo(n Nodo, vistos map[string]bool) []string {
	var problemas []string
	if !idValido.MatchString(n.ID) {
		problemas = append(problemas, fmt.Sprintf("id de nodo %q: usa minúsculas, números y guiones", n.ID))
	}
	if vistos[n.ID] {
		problemas = append(problemas, fmt.Sprintf("id de nodo repetido: %q", n.ID))
	}
	if n.Tipo != TipoPaso && n.Tipo != TipoResultado {
		problemas = append(problemas, fmt.Sprintf("el nodo %q tiene tipo %q: usa %s o %s", n.ID, n.Tipo, TipoPaso, TipoResultado))
	}
	if strings.TrimSpace(n.Nombre) == "" {
		problemas = append(problemas, fmt.Sprintf("el nodo %q está sin nombre", n.ID))
	}
	return problemas
}

func alcanzables(f Flujo, desde string) map[string]bool {
	porID := map[string]Nodo{}
	for _, n := range f.Nodos {
		porID[n.ID] = n
	}
	vistos := map[string]bool{desde: true}
	pendientes := []string{desde}
	for len(pendientes) > 0 {
		actual := pendientes[0]
		pendientes = pendientes[1:]
		for _, s := range porID[actual].Siguen {
			if vistos[s] {
				continue
			}
			vistos[s] = true
			pendientes = append(pendientes, s)
		}
	}
	return vistos
}

func contiene(lista []string, s string) bool {
	for _, x := range lista {
		if x == s {
			return true
		}
	}
	return false
}

func ruta(root, id string) string {
	return filepath.Join(root, Carpeta, id+".yaml")
}

// Guardar escribe el flujo si es válido, de una vez: un lector nunca ve el archivo a medias.
func Guardar(root string, f Flujo) error {
	if problemas := Validar(f); len(problemas) > 0 {
		return errors.New(strings.Join(problemas, "\n"))
	}
	datos, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	destino := ruta(root, f.ID)
	if err := os.MkdirAll(filepath.Dir(destino), 0755); err != nil {
		return err
	}
	temporal, err := os.CreateTemp(filepath.Dir(destino), ".flujo-*")
	if err != nil {
		return err
	}
	defer os.Remove(temporal.Name())
	if _, err := temporal.Write(datos); err != nil {
		temporal.Close()
		return err
	}
	if err := temporal.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temporal.Name(), 0644); err != nil {
		return err
	}
	return os.Rename(temporal.Name(), destino)
}

// Leer interpreta y valida un YAML.
func Leer(datos []byte) (Flujo, error) {
	var f Flujo
	if err := yaml.Unmarshal(datos, &f); err != nil {
		return Flujo{}, err
	}
	if problemas := Validar(f); len(problemas) > 0 {
		return f, errors.New(strings.Join(problemas, "\n"))
	}
	return f, nil
}

// Cargar lee un flujo del proyecto por su id.
func Cargar(root, id string) (Flujo, error) {
	if !idValido.MatchString(id) {
		return Flujo{}, fmt.Errorf("id de flujo %q: usa minúsculas, números y guiones", id)
	}
	datos, err := os.ReadFile(ruta(root, id))
	if errors.Is(err, os.ErrNotExist) {
		return Flujo{}, fmt.Errorf("no hay un flujo %q en este proyecto: 'qdd flujo lista' dice cuáles hay", id)
	}
	if err != nil {
		return Flujo{}, err
	}
	return Leer(datos)
}

// Listar resume los flujos del proyecto por id. Un archivo roto sale con su Problema: no desaparece en silencio. estadoDe
// da los símbolos de una tarea: entran en la huella porque se dibujan.
func Listar(root string, estadoDe func(string) string) []Resumen {
	archivos, err := filepath.Glob(filepath.Join(root, Carpeta, "*.yaml"))
	if err != nil {
		return nil
	}
	sort.Strings(archivos)
	lista := []Resumen{}
	for _, a := range archivos {
		lista = append(lista, resumir(a, estadoDe))
	}
	return lista
}

func huella(datos []byte) string {
	suma := sha256.Sum256(datos)
	return hex.EncodeToString(suma[:8])
}

func resumir(archivo string, estadoDe func(string) string) Resumen {
	id := strings.TrimSuffix(filepath.Base(archivo), ".yaml")
	datos, err := os.ReadFile(archivo)
	if err != nil {
		return Resumen{ID: id, Nombre: id, Problema: err.Error()}
	}
	r := Resumen{ID: id, Nombre: id, Huella: huella(datos)}
	f, err := Leer(datos)
	if err != nil {
		r.Problema = err.Error()
		return r
	}
	r.Nombre, r.Nodos = f.Nombre, len(f.Nodos)
	for _, n := range f.Nodos {
		if n.Tarea == "" {
			continue
		}
		datos = append(datos, []byte(estadoDe(n.Tarea))...)
	}
	r.Huella = huella(datos)
	return r
}
