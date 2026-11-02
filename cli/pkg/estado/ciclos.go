package estado

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Ciclos con nombre (blueprint estado-por-tareas, objetivos; decisión de la persona, 2026-11-01): las etapas del
// archivo son el ciclo del proyecto; «ciclos» declara otros, cada uno con sus etapas, y un objetivo puede seguir uno.
// Sus tareas toman las etapas de ese ciclo. Un blueprint que trae su propio ciclo (p. ej. ciclo-de-vida-agentico) lo
// declara al aplicarse, sin cambiar las etapas del proyecto.

// CicloProyecto nombra el ciclo de las etapas del archivo: el de una tarea sin objetivo o cuyo objetivo no dice ciclo.
const CicloProyecto = "proyecto"

// FormatoConCiclos: el archivo declara ciclos o un objetivo sigue uno. Un qdd anterior lo rechaza en vez de leerlo y
// perder los ciclos al escribir.
const FormatoConCiclos = 3

func (e Estado) usaCiclos() bool {
	if len(e.Ciclos) > 0 {
		return true
	}
	for _, o := range e.Objetivos {
		if o.Ciclo != "" {
			return true
		}
	}
	return false
}

// etapasDelCiclo: las del archivo para «proyecto» (o sin nombre); las declaradas, para otro.
func (e Estado) etapasDelCiclo(nombre string) ([]string, bool) {
	if nombre == "" || nombre == CicloProyecto {
		return e.Etapas, true
	}
	etapas, ok := e.Ciclos[nombre]
	return etapas, ok
}

// CicloDe es el ciclo que sigue una tarea: el de su objetivo, o el del proyecto.
func (e Estado) CicloDe(f Frente) string {
	o, ok := e.objetivo(f.Objetivo)
	if !ok || o.Ciclo == "" {
		return CicloProyecto
	}
	return o.Ciclo
}

// EtapasDe son las etapas de la tarea: las de su ciclo. Un ciclo que no existe (EPT-K01) cae en las del proyecto.
func (e Estado) EtapasDe(f Frente) []string {
	etapas, ok := e.etapasDelCiclo(e.CicloDe(f))
	if !ok {
		return e.Etapas
	}
	return etapas
}

func contiene(lista []string, s string) bool {
	for _, x := range lista {
		if x == s {
			return true
		}
	}
	return false
}

// checkCiclos (EPT-K01): un ciclo tiene al menos una etapa, sin repetir ni vacías, y no se llama «proyecto»; un
// objetivo sigue un ciclo que existe.
func (e Estado) checkCiclos() []Violacion {
	var vs []Violacion
	nombres := make([]string, 0, len(e.Ciclos))
	for n := range e.Ciclos {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	for _, n := range nombres {
		if err := cicloValido(n, e.Ciclos[n]); err != nil {
			vs = append(vs, Violacion{ID: "EPT-K01", Frente: n, Mensaje: err.Error()})
		}
	}
	for _, o := range e.Objetivos {
		if _, ok := e.etapasDelCiclo(o.Ciclo); !ok {
			vs = append(vs, Violacion{ID: "EPT-K01", Frente: o.ID, Mensaje: fmt.Sprintf("sigue el ciclo %q, que no existe: decláralo con 'qdd estado ciclo %s <etapas…>'", o.Ciclo, o.Ciclo)})
		}
	}
	return vs
}

func cicloValido(nombre string, etapas []string) error {
	if strings.TrimSpace(nombre) == "" {
		return errors.New("un ciclo necesita nombre")
	}
	if nombre == CicloProyecto {
		return errors.New("«proyecto» es el ciclo de las etapas del archivo: los demás ciclos se llaman distinto")
	}
	if len(etapas) == 0 {
		return fmt.Errorf("el ciclo %q no tiene etapas", nombre)
	}
	vistas := map[string]bool{}
	for _, et := range etapas {
		if strings.TrimSpace(et) == "" {
			return fmt.Errorf("el ciclo %q tiene una etapa sin nombre", nombre)
		}
		if vistas[et] {
			return fmt.Errorf("el ciclo %q repite la etapa %q", nombre, et)
		}
		vistas[et] = true
	}
	return nil
}

// reformar deja en f una celda por etapa del ciclo (EPT-K03): conserva las comunes, agrega las nuevas en «·» y quita
// las que sobran. Si una que sobra tiene avance (✅ ⏳ ❌), no la pierde: es un error que la nombra.
func reformar(f *Frente, etapas []string, ciclo string) error {
	var perdidas []string
	for nombre, c := range f.Celdas {
		if contiene(etapas, nombre) {
			continue
		}
		estado := NormalizarEstado(c.Estado)
		if estado == Hecho || estado == Espera || estado == Falla {
			perdidas = append(perdidas, estado+" en «"+nombre+"»")
		}
	}
	if len(perdidas) > 0 {
		sort.Strings(perdidas)
		return fmt.Errorf("la tarea %s tiene %s, y el ciclo «%s» no tiene esa etapa: ciérrala, o deja la etapa en — con su motivo, antes de cambiarla de ciclo", f.ID, strings.Join(perdidas, " y "), ciclo)
	}
	celdas := map[string]Celda{}
	for _, et := range etapas {
		c, ok := f.Celdas[et]
		if !ok {
			c = Celda{Estado: SinEmpezar}
		}
		celdas[et] = c
	}
	f.Celdas = celdas
	return nil
}

// reformarCiclo reforma las tareas que siguen el ciclo.
func (e *Estado) reformarCiclo(ciclo string) error {
	for i := range e.Frentes {
		if e.CicloDe(e.Frentes[i]) != ciclo {
			continue
		}
		if err := reformar(&e.Frentes[i], e.EtapasDe(e.Frentes[i]), ciclo); err != nil {
			return err
		}
	}
	return nil
}

// DeclararCiclo crea un ciclo o cambia sus etapas (qdd estado ciclo); con «proyecto», las etapas del archivo. Sus
// tareas toman las etapas nuevas sin perder avance (EPT-K03).
func DeclararCiclo(root, nombre string, etapas []string, now time.Time) error {
	if nombre != CicloProyecto {
		if err := cicloValido(nombre, etapas); err != nil {
			return err
		}
	}
	if nombre == CicloProyecto {
		if err := cicloValido("etapas", etapas); err != nil {
			return err
		}
	}
	e, err := cargarParaEscribir(root)
	if err != nil {
		return err
	}
	if nombre == CicloProyecto {
		e.Etapas = etapas
	}
	if nombre != CicloProyecto {
		if e.Ciclos == nil {
			e.Ciclos = map[string][]string{}
		}
		e.Ciclos[nombre] = etapas
	}
	if err := e.reformarCiclo(nombre); err != nil {
		return fmt.Errorf("no se escribió: %w", err)
	}
	e.Actualizado = now.Format(time.RFC3339)
	return Save(root, e)
}

// ciclosUsados: los ciclos propios (no «proyecto») de los objetivos dados, en su orden y sin repetir.
func ciclosUsados(objetivos []Objetivo) []string {
	var out []string
	for _, o := range objetivos {
		if o.Ciclo == "" || o.Ciclo == CicloProyecto || contiene(out, o.Ciclo) {
			continue
		}
		out = append(out, o.Ciclo)
	}
	return out
}

// grupoCiclo es una tabla: los grupos del árbol (por objetivo) que siguen un mismo ciclo.
type grupoCiclo struct {
	Ciclo  string
	Etapas []string
	Grupos []GrupoArbol
}

// porCiclo reparte los grupos del árbol en una tabla por ciclo (EPT-K04): la del proyecto primero y después cada ciclo
// en el orden en que aparece; dentro de cada una, los grupos en su orden. Una tabla sin grupos no va.
func (e Estado) porCiclo(grupos []GrupoArbol) []grupoCiclo {
	out := []grupoCiclo{{Ciclo: CicloProyecto, Etapas: e.Etapas}}
	for _, g := range grupos {
		ciclo := CicloProyecto
		if g.Objetivo != nil && g.Objetivo.Ciclo != "" {
			ciclo = g.Objetivo.Ciclo
		}
		i := indiceCiclo(out, ciclo)
		if i < 0 {
			etapas, _ := e.etapasDelCiclo(ciclo)
			out = append(out, grupoCiclo{Ciclo: ciclo, Etapas: etapas})
			i = len(out) - 1
		}
		out[i].Grupos = append(out[i].Grupos, g)
	}
	if len(out[0].Grupos) == 0 && len(out) > 1 {
		return out[1:]
	}
	return out
}

func indiceCiclo(gs []grupoCiclo, ciclo string) int {
	for i, g := range gs {
		if g.Ciclo == ciclo {
			return i
		}
	}
	return -1
}
