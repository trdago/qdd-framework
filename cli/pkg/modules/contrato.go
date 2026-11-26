package modules

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Rule: Prohibido usar 'else'
// Contrato de una ficha: cada invariante es una promesa de QDD sobre el módulo, y la demuestran las pruebas que la citan
// con «contrato: <ficha>#<id>» en un comentario (Go, TypeScript, JS, shell: cualquier archivo de pruebas). El id sale
// del texto del invariante: si la promesa cambia, la cita vieja deja de contar y alguien tiene que revisar que la prueba
// todavía la demuestre. Medido 2026-11-21: de 262 promesas de las 34 fichas de qdd (228 invariantes y 34 criterios de
// éxito), 118 tenían una prueba que fallaría si se rompían.

var reCita = regexp.MustCompile(`contrato: ([a-z0-9][a-z0-9_-]{0,63})#([0-9a-f]{8})\b`)

// carpetasFuera no tienen pruebas del proyecto.
var carpetasFuera = map[string]bool{".git": true, "node_modules": true, "vendor": true, "tmp": true, ".claude": true, "dist": true, "build": true}

// IDInvariante son 8 hexadecimales del texto del invariante, con los espacios normalizados.
func IDInvariante(texto string) string {
	suma := sha256.Sum256([]byte(strings.Join(strings.Fields(texto), " ")))
	return hex.EncodeToString(suma[:4])
}

// CitaInvariante es lo que una prueba escribe en un comentario para decir que demuestra ese invariante.
func CitaInvariante(ficha, id string) string {
	return "contrato: " + ficha + "#" + id
}

// Clausula es un invariante de una ficha con las pruebas que lo citan (ruta relativa:línea).
type Clausula struct {
	Ficha   string
	ID      string
	Texto   string
	Pruebas []string
}

// CitasDePruebas recorre los archivos de pruebas del proyecto: «ficha#id» → dónde se cita.
func CitasDePruebas(root string) map[string][]string {
	citas := map[string][]string{}
	filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() && carpetasFuera[d.Name()] {
			return filepath.SkipDir
		}
		if d.IsDir() || !esArchivoDePruebas(p) {
			return nil
		}
		anotarCitas(root, p, citas)
		return nil
	})
	return citas
}

func esArchivoDePruebas(p string) bool {
	nombre := filepath.Base(p)
	for _, marca := range []string{"_test.", ".test.", ".spec.", "_spec."} {
		if strings.Contains(nombre, marca) {
			return true
		}
	}
	dir := "/" + filepath.ToSlash(filepath.Dir(p)) + "/"
	return strings.Contains(dir, "/tests/") || strings.Contains(dir, "/e2e/")
}

func anotarCitas(root, p string, citas map[string][]string) {
	f, err := os.Open(p)
	if err != nil {
		return
	}
	defer f.Close()
	rel, _ := filepath.Rel(root, p)
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 64<<10), 1<<20)
	for n := 1; s.Scan(); n++ {
		for _, m := range reCita.FindAllStringSubmatch(s.Text(), -1) {
			clave := m[1] + "#" + m[2]
			citas[clave] = append(citas[clave], fmt.Sprintf("%s:%d", filepath.ToSlash(rel), n))
		}
	}
}

// Contrato son los invariantes de la ficha, en su orden, con sus pruebas.
func Contrato(m Module, citas map[string][]string) []Clausula {
	out := make([]Clausula, 0, len(m.ObjetivoTecnologico.Invariantes))
	for _, texto := range m.ObjetivoTecnologico.Invariantes {
		id := IDInvariante(texto)
		pruebas := append([]string(nil), citas[m.ID+"#"+id]...)
		sort.Strings(pruebas)
		out = append(out, Clausula{Ficha: m.ID, ID: id, Texto: texto, Pruebas: pruebas})
	}
	return out
}

// SinPrueba son las cláusulas que ninguna prueba cita.
func SinPrueba(c []Clausula) []Clausula {
	var out []Clausula
	for _, cl := range c {
		if len(cl.Pruebas) == 0 {
			out = append(out, cl)
		}
	}
	return out
}

// lineaDeClausula es como el contexto muestra un invariante: con su id y dónde se prueba, o el aviso de que no.
func lineaDeClausula(cl Clausula) string {
	base := fmt.Sprintf("[%s#%s] %s", cl.Ficha, cl.ID, cl.Texto)
	if len(cl.Pruebas) == 0 {
		return base + " (⚠️ sin prueba que lo cite: una prueba que lo demuestre lleva «" + CitaInvariante(cl.Ficha, cl.ID) + "» en un comentario)"
	}
	return base + " (probado en " + strings.Join(cl.Pruebas, ", ") + ")"
}
