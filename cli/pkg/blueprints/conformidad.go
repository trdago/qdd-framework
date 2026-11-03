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
	Fuera      []string // «id (motivo)»: lo que este código no hace, dicho en la ficha
	PorQDD     string   // id@versión que implementa el qdd instalado, si el módulo solo usa un blueprint de qdd
	Motivo     string   // por qué falla, cuando no es por pruebas (p. ej. el qdd instalado no cubre la versión)
}

// ConformidadDe mide un módulo contra un blueprint: busca el id de cada escenario en los archivos de prueba de rutas.
// Un escenario que la ficha declara fuera, con su motivo, no se exige pero siempre se muestra; uno fuera sin motivo, o
// que el blueprint no tiene, falta.
func ConformidadDe(root, modulo string, b Blueprint, impl modules.Implementacion, rutas []string) Conformidad {
	c := Conformidad{Modulo: modulo, Blueprint: b.Ref()}
	todos := escenariosQueDebe(b, impl.Respuestas)
	pruebas := textoDePruebas(root, rutas)
	for _, id := range todos {
		if motivo := strings.TrimSpace(impl.Fuera[id]); motivo != "" {
			c.Fuera = append(c.Fuera, id+" ("+motivo+")")
			continue
		}
		c.Escenarios = append(c.Escenarios, id)
		if !nombraEscenario(pruebas, id) {
			c.Faltan = append(c.Faltan, id)
		}
	}
	for id := range impl.Fuera {
		if !inList(todos, id) {
			c.Faltan = append(c.Faltan, id)
		}
	}
	sort.Strings(c.Faltan)
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
// Antes del id puede ir una minúscula: en Go va pegado al prefijo (TestCVA_C01, FND-103); una mayúscula o un dígito,
// no, porque sería otra palabra (OAUTH_01 no es AUTH-01).
func nombraEscenario(texto, id string) bool {
	partes := strings.Split(regexp.QuoteMeta(id), "-")
	return regexp.MustCompile(`(^|[^A-Z0-9])` + strings.Join(partes, "[-_]") + `([^A-Za-z0-9]|$)`).MatchString(texto)
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

// pedidoDeConformidad es un módulo y el blueprint que aplicó (su registro de aplicación) o implementa (su ficha).
type pedidoDeConformidad struct {
	modulo   string
	impl     modules.Implementacion
	rutas    []string
	archivo  string // dónde se declara: el registro de aplicación o la ficha
	aplicado bool   // lo usa (registro de aplicación); si no, es la implementación de origen
}

// conformidadDe mide un pedido. Un módulo que aplica un blueprint que implementa qdd no lleva su código: lo cumple
// el qdd instalado si cubre la versión aplicada (FND-102). La implementación de origen siempre se mide con sus pruebas.
func conformidadDe(root string, p pedidoDeConformidad, b Blueprint) Conformidad {
	implementa, esDeQDD, motivo := cubiertoPorQDD(b.ID, b.Version)
	if !p.aplicado || !esDeQDD {
		return ConformidadDe(root, p.modulo, b, p.impl, p.rutas)
	}
	return Conformidad{Modulo: p.modulo, Blueprint: b.Ref(), PorQDD: b.ID + "@" + implementa, Motivo: motivo}
}

func pedidosDeConformidad(root string) []pedidoDeConformidad {
	fichas, _ := modules.Load(root)
	porID := map[string]modules.Module{}
	for _, m := range fichas {
		porID[m.ID] = m
	}
	var pedidos []pedidoDeConformidad
	apps, _ := LoadApplications(root)
	for _, a := range apps {
		impl := modules.Implementacion{Blueprint: a.Blueprint, Respuestas: a.Respuestas, Fuera: porID[a.Modulo].Fuera}
		pedidos = append(pedidos, pedidoDeConformidad{a.Modulo, impl, porID[a.Modulo].Rutas, filepath.ToSlash(filepath.Join(ApplicationsDir, a.Modulo+".yaml")), true})
	}
	for _, m := range fichas {
		for _, impl := range m.Implementa {
			pedidos = append(pedidos, pedidoDeConformidad{m.ID, impl, m.Rutas, filepath.ToSlash(filepath.Join(modules.Dir, m.ID+".yaml")), false})
		}
	}
	return pedidos
}

// comoDeclararFuera: un módulo que aplica el blueprint lo dice en la raíz de su ficha; la implementación de origen, en
// su entrada de implementa (FND-121).
func comoDeclararFuera(p pedidoDeConformidad) string {
	if p.aplicado {
		return "fuera: {ID: motivo} en la raíz de la ficha, junto a blueprint"
	}
	return "fuera: {ID: motivo} en su entrada de implementa"
}

// Conformar mide cada módulo que aplicó un blueprint (sus registros de aplicación) o que lo implementa (su ficha).
// El blueprint sale de lo importado o, si el proyecto no lo importa, de lo publicado en la fuente de la organización.
func Conformar(root string) ([]Conformidad, error) {
	var out []Conformidad
	for _, p := range pedidosDeConformidad(root) {
		b, err := blueprintParaConformar(root, p.impl.Blueprint)
		if err != nil {
			return out, err
		}
		out = append(out, conformidadDe(root, p, b))
	}
	return out, nil
}

// CheckConformidad es la conformidad en el gate (qdd audit), sin red:
//   - BP-08-CONFORMIDAD (bloquea): a un módulo que aplica o implementa un blueprint le falta la prueba de un escenario.
//   - BP-08-SIN-MEDIR (recomendación): el blueprint no está importado ni en la copia local del repositorio de la
//     organización (p. ej. en el CI). 'qdd blueprint conformance' lo trae y lo mide.
//
// Sin importarlo, se mide contra la última versión de la copia local, la misma que usa BP-06-BEHIND.
func CheckConformidad(root string) (bloquean, sinMedir []Problema) {
	for _, p := range pedidosDeConformidad(root) {
		b, err := blueprintSinRed(root, p.impl.Blueprint)
		if err != nil {
			sinMedir = append(sinMedir, Problema{"BP-08-SIN-MEDIR", p.archivo, fmt.Sprintf("%s · %s: sin medir, %v. Mídelo con qdd blueprint conformance.", p.modulo, p.impl.Blueprint, err)})
			continue
		}
		c := conformidadDe(root, p, b)
		if c.Motivo != "" {
			bloquean = append(bloquean, Problema{"BP-08-CONFORMIDAD", p.archivo, LineaConformidad(c) + "."})
			continue
		}
		if len(c.Faltan) > 0 {
			bloquean = append(bloquean, Problema{"BP-08-CONFORMIDAD", p.archivo, LineaConformidad(c) + ". Escribe una prueba que lleve el id de cada escenario que falta, o declara en la ficha, con su motivo, lo que este código no hace: " + comoDeclararFuera(p) + "."})
		}
	}
	return bloquean, sinMedir
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

// blueprintSinRed es el blueprint importado o, si no, el de la última versión de la copia local del repositorio de
// la organización. No trae nada de la red.
func blueprintSinRed(root, id string) (Blueprint, error) {
	if b, err := Find(root, id); err == nil {
		return b, nil
	}
	reg, err := LoadRegistro(root)
	if err != nil || reg.Origen == "" {
		return Blueprint{}, fmt.Errorf("no está importado y el proyecto no tiene un repositorio de blueprints")
	}
	mirror, err := mirrorPath(reg.Origen)
	if err != nil || !exists(mirror) {
		return Blueprint{}, fmt.Errorf("no está importado ni hay una copia local de %s en esta máquina", reg.Origen)
	}
	tags, err := versionTags(mirror)
	if err != nil || len(tags) == 0 {
		return Blueprint{}, fmt.Errorf("la copia local de %s no tiene versiones publicadas", reg.Origen)
	}
	dir, err := os.MkdirTemp("", "qdd-conformidad-")
	if err != nil {
		return Blueprint{}, err
	}
	defer os.RemoveAll(dir)
	if err := extraer(mirror, tags[0], dir); err != nil {
		return Blueprint{}, err
	}
	_, disponibles, err := catalogo(dir)
	if err != nil {
		return Blueprint{}, err
	}
	sub, ok := disponibles[id]
	if !ok {
		return Blueprint{}, fmt.Errorf("no está publicado en %s %s", reg.Origen, tags[0])
	}
	return ParseDir(filepath.Join(dir, filepath.FromSlash(sub)))
}

// LineaConformidad: «estado · estado-por-tareas@1.1.0 · escenarios con prueba 23/25 · faltan EPT-C11, EPT-C12».
func LineaConformidad(c Conformidad) string {
	if c.PorQDD != "" {
		linea := fmt.Sprintf("%s · %s · lo implementa qdd (%s)", c.Modulo, c.Blueprint, c.PorQDD)
		if c.Motivo != "" {
			return "❌ " + linea + " · " + c.Motivo
		}
		return "✅ " + linea
	}
	conPrueba := 0
	for _, id := range c.Escenarios {
		if !inList(c.Faltan, id) {
			conPrueba++
		}
	}
	linea := fmt.Sprintf("%s · %s · escenarios con prueba %d/%d", c.Modulo, c.Blueprint, conPrueba, len(c.Escenarios))
	if len(c.Fuera) > 0 {
		linea += " · fuera " + strings.Join(c.Fuera, ", ")
	}
	if len(c.Faltan) == 0 {
		return "✅ " + linea
	}
	return "❌ " + linea + " · faltan " + strings.Join(c.Faltan, ", ")
}
