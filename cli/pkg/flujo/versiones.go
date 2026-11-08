package flujo

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'
// Versiones (RFC 0025, decisión de la persona 2026-11-08: «revisiones + versiones con nombre»). Cada cambio guardado sube
// Flujo.Revision y deja una línea en <id>.historial.jsonl, junto al flujo y versionado: quién cambió qué, que git no dice.
// Una versión con nombre guarda la foto entera en versiones/<id>/<n>.yaml. Volver a una es una revisión nueva: la
// historia no se reescribe. Deshacer una edición que nadie recibió es la excepción: la borra como si no hubiera pasado.

// Version es una foto con nombre del flujo, sobre una revisión.
type Version struct {
	N        int    `yaml:"n" json:"n"`
	Nombre   string `yaml:"nombre" json:"nombre"`
	Revision int    `yaml:"revision" json:"revision"`
	En       string `yaml:"en" json:"en"`
}

// Revision es una línea del historial.
type Revision struct {
	Rev     int    `json:"rev"`
	En      string `json:"en"`
	Autor   string `json:"autor"`
	Resumen string `json:"resumen"`
}

// CambioNodo es un nodo que está en las dos fotos con algo distinto.
type CambioNodo struct {
	ID     string   `json:"id"`
	Nombre string   `json:"nombre"`
	Campos []string `json:"campos"`
}

// Diferencias es lo que cambió de una foto a otra.
type Diferencias struct {
	Agregados []Nodo       `json:"agregados"`
	Quitados  []Nodo       `json:"quitados"`
	Cambiados []CambioNodo `json:"cambiados"`
}

func rutaHistorial(root, id string) string {
	return filepath.Join(root, Carpeta, id+".historial.jsonl")
}

func rutaVersion(root, id string, n int) string {
	return filepath.Join(root, Carpeta, "versiones", id, fmt.Sprintf("%d.yaml", n))
}

func autorDe(de string) string {
	if de == "" {
		return DeIA
	}
	return de
}

// guardarRevision guarda el flujo una revisión después de previa y lo anota. Quien llama tiene el candado.
func guardarRevision(root string, f *Flujo, previa int, autor, resumen string, ahora time.Time) error {
	f.Revision = previa + 1
	if err := Guardar(root, *f); err != nil {
		return err
	}
	r := Revision{Rev: f.Revision, En: ahora.UTC().Format(time.RFC3339), Autor: autorDe(autor), Resumen: resumen}
	return anotarRevision(root, f.ID, r)
}

func anotarRevision(root, id string, r Revision) error {
	linea, err := json.Marshal(r)
	if err != nil {
		return err
	}
	a, err := os.OpenFile(rutaHistorial(root, id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	if _, err := a.Write(append(linea, '\n')); err != nil {
		a.Close()
		return err
	}
	return a.Close()
}

// Historial son las revisiones anotadas, de la más vieja a la más nueva. Una línea rota se salta.
func Historial(root, id string) []Revision {
	a, err := os.Open(rutaHistorial(root, id))
	if err != nil {
		return nil
	}
	defer a.Close()
	var lista []Revision
	lector := bufio.NewScanner(a)
	lector.Buffer(make([]byte, 64*1024), 1024*1024)
	for lector.Scan() {
		var r Revision
		if json.Unmarshal(lector.Bytes(), &r) != nil {
			continue
		}
		lista = append(lista, r)
	}
	return lista
}

// quitarRevision borra la última línea si es la de rev: deshacer deja el historial como estaba.
func quitarRevision(root, id string, rev int) error {
	h := Historial(root, id)
	if len(h) == 0 || h[len(h)-1].Rev != rev {
		return nil
	}
	var b strings.Builder
	for _, r := range h[:len(h)-1] {
		linea, _ := json.Marshal(r)
		b.Write(append(linea, '\n'))
	}
	return escribirAtomico(rutaHistorial(root, id), []byte(b.String()))
}

// CerrarVersion guarda la foto de la revisión actual con un nombre («v2 acordada con Ana»). No sube la revisión.
func CerrarVersion(root, id, nombre string, ahora time.Time) (Version, error) {
	nombre = limpiar(nombre)
	if nombre == "" {
		return Version{}, errors.New("una versión lleva nombre: qdd flujo version <flujo> \"<nombre>\"")
	}
	soltar, err := tomarFlujo(root, id)
	if err != nil {
		return Version{}, err
	}
	defer soltar()
	f, err := Cargar(root, id)
	if err != nil {
		return Version{}, err
	}
	v := Version{N: 1, Nombre: nombre, Revision: f.Revision, En: ahora.UTC().Format(time.RFC3339)}
	if len(f.Versiones) > 0 {
		v.N = f.Versiones[len(f.Versiones)-1].N + 1
	}
	foto := f
	foto.Versiones = nil
	datos, err := yaml.Marshal(foto)
	if err != nil {
		return Version{}, err
	}
	if err := escribirAtomico(rutaVersion(root, id, v.N), datos); err != nil {
		return Version{}, err
	}
	f.Versiones = append(f.Versiones, v)
	return v, Guardar(root, f)
}

// CargarVersion lee la foto de la versión n.
func CargarVersion(root, id string, n int) (Flujo, error) {
	if !idValido.MatchString(id) {
		return Flujo{}, fmt.Errorf("id de flujo %q: usa minúsculas, números y guiones", id)
	}
	datos, err := os.ReadFile(rutaVersion(root, id, n))
	if err != nil {
		return Flujo{}, fmt.Errorf("el flujo %s no tiene una versión %d: 'qdd flujo versiones %s' dice cuáles hay", id, n, id)
	}
	return Leer(datos)
}

// VolverA es el cambio que deja el flujo como en la versión n, conservando sus versiones: una revisión nueva.
func VolverA(root string, n int) Cambio {
	return func(f *Flujo) (hecho, error) {
		foto, err := CargarVersion(root, f.ID, n)
		if err != nil {
			return hecho{}, err
		}
		i := slices.IndexFunc(f.Versiones, func(v Version) bool { return v.N == n })
		nombre := ""
		if i >= 0 {
			nombre = " «" + f.Versiones[i].Nombre + "»"
		}
		foto.Versiones, foto.Revision = f.Versiones, f.Revision
		*f = foto
		return hecho{linea: fmt.Sprintf("↶ vuelve a v%d%s", n, nombre)}, nil
	}
}

// Volver deja el flujo como en la versión n, como una edición más (se registra y se le cuenta a la IA).
func Volver(root, id string, n int, de string, ahora time.Time) (Flujo, error) {
	f, _, err := Editar(root, id, de, ahora, VolverA(root, n))
	return f, err
}

// Comparar dice qué nodos se agregaron, se quitaron o cambiaron de a a b.
func Comparar(a, b Flujo) Diferencias {
	d := Diferencias{Agregados: []Nodo{}, Quitados: []Nodo{}, Cambiados: []CambioNodo{}}
	enA := map[string]Nodo{}
	for _, n := range a.Nodos {
		enA[n.ID] = n
	}
	enB := map[string]bool{}
	for _, n := range b.Nodos {
		enB[n.ID] = true
		previo, ok := enA[n.ID]
		if !ok {
			d.Agregados = append(d.Agregados, n)
			continue
		}
		if campos := camposDistintos(previo, n); len(campos) > 0 {
			d.Cambiados = append(d.Cambiados, CambioNodo{ID: n.ID, Nombre: n.Nombre, Campos: campos})
		}
	}
	for _, n := range a.Nodos {
		if !enB[n.ID] {
			d.Quitados = append(d.Quitados, n)
		}
	}
	return d
}

func camposDistintos(a, b Nodo) []string {
	var campos []string
	for _, c := range []struct {
		nombre   string
		distinto bool
	}{
		{"nombre", a.Nombre != b.Nombre},
		{"descripcion", a.Descripcion != b.Descripcion},
		{"tipo", a.Tipo != b.Tipo},
		{"siguen", !slices.Equal(a.Siguen, b.Siguen)},
		{"grupo", a.Grupo != b.Grupo},
		{"nivel", a.Nivel != b.Nivel},
	} {
		if c.distinto {
			campos = append(campos, c.nombre)
		}
	}
	return campos
}

// RevisarHistorial dice si el historial no llega a la revisión del flujo (alguien editó el YAML a mano); "" si cuadra.
func RevisarHistorial(root, id string) string {
	f, err := Cargar(root, id)
	if err != nil {
		return ""
	}
	ultima := 0
	if h := Historial(root, id); len(h) > 0 {
		ultima = h[len(h)-1].Rev
	}
	if ultima == f.Revision {
		return ""
	}
	return fmt.Sprintf("el flujo %s está en la revisión %d y su historial llega a la %d", id, f.Revision, ultima)
}

// RepararHistorial anota la revisión actual como una edición sin registro.
func RepararHistorial(root, id string, ahora time.Time) error {
	soltar, err := tomarFlujo(root, id)
	if err != nil {
		return err
	}
	defer soltar()
	if RevisarHistorial(root, id) == "" {
		return nil
	}
	f, err := Cargar(root, id)
	if err != nil {
		return err
	}
	return anotarRevision(root, id, Revision{Rev: f.Revision, En: ahora.UTC().Format(time.RFC3339), Autor: "sin registro", Resumen: "edición sin registro (el YAML cambió a mano)"})
}
