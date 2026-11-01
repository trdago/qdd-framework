package blueprints

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'
// Conformidad (qdd blueprint conformance): la misma regla de la certificación ciega, sobre el código real. Cada
// escenario que un módulo debe cumplir tiene una prueba que lleva su id. Un módulo que aplicó un blueprint se mide con
// sus respuestas; uno que lo implementa como origen (su ficha dice implementa), con todas las opciones que ofrece.

// Conformidad es lo que un módulo cumple de un blueprint.
type Conformidad struct {
	Modulo     string
	Blueprint  string // id@versión
	Escenarios []string
	Faltan     []string // escenarios sin una prueba que lleve su id
}

// ConformidadDe mide un módulo contra un blueprint: busca el id de cada escenario en los archivos de prueba de rutas.
func ConformidadDe(root, modulo string, b Blueprint, impl modules.Implementacion, rutas []string) Conformidad {
	c := Conformidad{Modulo: modulo, Blueprint: b.Ref(), Escenarios: escenariosQueDebe(b, impl.Respuestas)}
	pruebas := textoDePruebas(root, rutas)
	for _, id := range c.Escenarios {
		if !nombraEscenario(pruebas, id) {
			c.Faltan = append(c.Faltan, id)
		}
	}
	return c
}

// escenariosQueDebe: con respuestas, los de esa variante; sin ellas, los de todas las opciones.
func escenariosQueDebe(b Blueprint, respuestas map[string]string) []string {
	escenarios := todosLosEscenarios(b)
	if len(respuestas) > 0 {
		_, escenarios = b.componer(respuestas)
	}
	vistos := map[string]bool{}
	var ids []string
	for _, e := range escenarios {
		if !vistos[e.ID] {
			vistos[e.ID] = true
			ids = append(ids, e.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

// nombraEscenario: el id aparece entero, con «-» o con «_» (un nombre de prueba no admite «-» en muchos lenguajes).
func nombraEscenario(texto, id string) bool {
	partes := strings.Split(regexp.QuoteMeta(id), "-")
	return regexp.MustCompile(`(^|[^A-Za-z0-9])` + strings.Join(partes, "[-_]") + `([^A-Za-z0-9]|$)`).MatchString(texto)
}

// textoDePruebas junta los archivos de prueba de las rutas (archivos o carpetas).
func textoDePruebas(root string, rutas []string) string {
	var s strings.Builder
	for _, r := range rutas {
		_ = filepath.Walk(filepath.Join(root, filepath.FromSlash(r)), func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() && (info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "vendor") {
				return filepath.SkipDir
			}
			if info.IsDir() || info.Size() > 1<<20 || !esArchivoDePrueba(path) {
				return nil
			}
			data, _ := os.ReadFile(path)
			s.Write(data)
			s.WriteByte('\n')
			return nil
		})
	}
	return s.String()
}

// esArchivoDePrueba reconoce las convenciones comunes de cualquier lenguaje: x_test.go, x.test.ts, x.spec.js,
// test_x.py, x_spec.rb o un archivo dentro de test/, tests/, __tests__ o spec/.
func esArchivoDePrueba(path string) bool {
	nombre := strings.ToLower(filepath.Base(path))
	for _, marca := range []string{"_test.", ".test.", ".spec.", "_spec."} {
		if strings.Contains(nombre, marca) {
			return true
		}
	}
	if strings.HasPrefix(nombre, "test_") {
		return true
	}
	for _, dir := range strings.Split(filepath.ToSlash(filepath.Dir(path)), "/") {
		if dir == "test" || dir == "tests" || dir == "__tests__" || dir == "spec" {
			return true
		}
	}
	return false
}

// Conformar mide cada módulo que aplicó un blueprint (sus registros de aplicación) o que lo implementa (su ficha).
// El blueprint sale de lo importado o, si el proyecto no lo importa, de lo publicado en la fuente de la organización.
func Conformar(root string) ([]Conformidad, error) {
	fichas, _ := modules.Load(root)
	porID := map[string]modules.Module{}
	for _, m := range fichas {
		porID[m.ID] = m
	}
	type pedido struct {
		modulo string
		impl   modules.Implementacion
	}
	var pedidos []pedido
	apps, _ := LoadApplications(root)
	for _, a := range apps {
		pedidos = append(pedidos, pedido{a.Modulo, modules.Implementacion{Blueprint: a.Blueprint, Respuestas: a.Respuestas}})
	}
	for _, m := range fichas {
		for _, impl := range m.Implementa {
			pedidos = append(pedidos, pedido{m.ID, impl})
		}
	}
	var out []Conformidad
	for _, p := range pedidos {
		b, err := blueprintParaConformar(root, p.impl.Blueprint)
		if err != nil {
			return out, err
		}
		out = append(out, ConformidadDe(root, p.modulo, b, p.impl, porID[p.modulo].Rutas))
	}
	return out, nil
}

func blueprintParaConformar(root, id string) (Blueprint, error) {
	if b, err := Find(root, id); err == nil {
		return b, nil
	}
	bps, _, err := publicados(root, []string{id})
	if err != nil || len(bps) == 0 {
		return Blueprint{}, fmt.Errorf("'%s': no está importado ni publicado: %v", id, err)
	}
	return bps[0], nil
}

// LineaConformidad: «estado · estado-por-tareas@1.1.0 · escenarios con prueba 23/25 · faltan EPT-C11, EPT-C12».
func LineaConformidad(c Conformidad) string {
	linea := fmt.Sprintf("%s · %s · escenarios con prueba %d/%d", c.Modulo, c.Blueprint, len(c.Escenarios)-len(c.Faltan), len(c.Escenarios))
	if len(c.Faltan) == 0 {
		return "✅ " + linea
	}
	return "❌ " + linea + " · faltan " + strings.Join(c.Faltan, ", ")
}
