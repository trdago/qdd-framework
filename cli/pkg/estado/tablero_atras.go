package estado

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/modules"
)

// Rule: Prohibido usar 'else'
// RFC 0017 §2 y §3 (MEJ-010, proyecto-a): un tablero que no se queda atrás. QDD no juzga si un nombre o un criterio siguen
// siendo ciertos; con el módulo de una tarea y la hora de su última celda, solo dice cuándo revisarlos. Es git y nada
// más: solo lo corren qdd estado check y el contexto de la IA, nunca el panel ni la barra.

// presupuestoCommits: si los git log de todas las tareas pasan de esto, el aviso se limita al objetivo en foco (RFC 0017
// §2, medido en Linux: 40 ms en proyecto-a y 57 ms en qdd). Variable para que las pruebas lo agoten.
var presupuestoCommits = 200 * time.Millisecond

// TareaConCommits es una tarea con commits en las rutas de su ficha después de su última celda.
type TareaConCommits struct {
	Tarea   string `json:"tarea"`
	Modulo  string `json:"modulo"`
	Commits int    `json:"commits"`
	Desde   string `json:"desde"`
}

// CommitsEnTareas es el aviso de qdd estado check; SoloFoco dice que el presupuesto se agotó y solo se revisó el foco.
type CommitsEnTareas struct {
	Tareas   []TareaConCommits `json:"tareas"`
	SoloFoco bool              `json:"solo_foco,omitempty"`
}

// Linea es el aviso de una tarea, sin bloquear: un commit que no cambia lo que falta también cuenta.
func (t TareaConCommits) Linea() string {
	commits := "commits"
	if t.Commits == 1 {
		commits = "commit"
	}
	return fmt.Sprintf("tarea %s: %d %s en las rutas de su ficha %s desde su última celda (%s)", t.Tarea, t.Commits, commits, t.Modulo, t.Desde)
}

// tareaConModulo es una tarea abierta con ficha y la hora de su última celda, con esa ficha ya leída.
type tareaConModulo struct {
	frente Frente
	ficha  modules.Module
	desde  time.Time
	foco   bool
}

// conModulo son las tareas abiertas con módulo, ficha existente y última celda legible; las del foco primero. Las
// fichas se leen una vez: leerlas por tarea agotaba el presupuesto antes del primer git log.
func conModulo(root string, e Estado) []tareaConModulo {
	foco := e.focoAbierto()
	var out []tareaConModulo
	var fichas map[string]modules.Module
	for _, f := range e.Frentes {
		if f.Estado != EstadoAbierto || f.Modulo == "" {
			continue
		}
		desde, err := time.Parse(time.RFC3339, f.TocadaEl)
		if err != nil {
			continue
		}
		if fichas == nil {
			fichas = fichasPorID(root)
		}
		ficha, ok := fichas[f.Modulo]
		if !ok {
			continue
		}
		out = append(out, tareaConModulo{frente: f, ficha: ficha, desde: desde, foco: foco != "" && f.Objetivo == foco})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].foco && !out[j].foco })
	return out
}

func fichasPorID(root string) map[string]modules.Module {
	mods, _ := modules.Load(root)
	out := make(map[string]modules.Module, len(mods))
	for _, m := range mods {
		out[m.ID] = m
	}
	return out
}

// CommitsDesdeSuCelda cuenta, por tarea con módulo, los commits en las rutas de su ficha después de su última celda.
// Sin git, sin módulo o sin última celda no opina. El presupuesto mide los git log, no la lectura de las fichas.
func CommitsDesdeSuCelda(root string, e Estado) CommitsEnTareas {
	out := CommitsEnTareas{Tareas: []TareaConCommits{}}
	tareas := conModulo(root, e)
	inicio := time.Now()
	for _, t := range tareas {
		if !t.foco && time.Since(inicio) > presupuestoCommits {
			out.SoloFoco = true
			break
		}
		n := commitsDesde(root, t.desde, t.ficha.Rutas)
		if n <= 0 {
			continue
		}
		out.Tareas = append(out.Tareas, TareaConCommits{Tarea: t.frente.ID, Modulo: t.ficha.ID, Commits: n, Desde: t.frente.TocadaEl})
	}
	return out
}

func commitsDesde(root string, desde time.Time, rutas []string) int {
	if len(rutas) == 0 {
		return 0
	}
	args := append([]string{"-C", root, "rev-list", "--count", "--since=" + desde.Format(time.RFC3339), "HEAD", "--"}, rutas...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return n
}

// TareaPorRevisar es una tarea cuya ficha cambió lo que la define después de su última celda (RFC 0017 §3).
type TareaPorRevisar struct {
	Tarea   string
	Nombre  string
	Modulo  string
	Desde   string
	Cambios []string // «decisiones», «objetivo operativo»
}

// Linea es lo que el contexto le pide a la IA: revisar el nombre y el criterio, no reescribirlos a ciegas.
func (r TareaPorRevisar) Linea() string {
	return fmt.Sprintf("Tarea %s («%s»): la ficha %s cambió su(s) %s después de su última celda (%s). Revisa el nombre de la tarea y el criterio de su objetivo, y formúlalos según lo que falta (qdd estado tarea / objetivo).",
		r.Tarea, r.Nombre, r.Modulo, strings.Join(r.Cambios, " y "), r.Desde)
}

// Revisiones son las tareas por revisar; SoloFoco dice que el presupuesto se agotó y solo se revisó el foco.
type Revisiones struct {
	Tareas   []TareaPorRevisar
	SoloFoco bool
}

// TareasPorRevisar compara la ficha de cada tarea con módulo tal como estaba en git a la hora de su última celda con
// la de la carpeta. Sin git, o sin una versión de la ficha anterior a esa hora, no opina. Corre con cada contexto de la
// IA: con el mismo presupuesto que check, pasado el cual solo revisa el objetivo en foco.
func TareasPorRevisar(root string, e Estado) Revisiones {
	var out Revisiones
	tareas := conModulo(root, e)
	inicio := time.Now()
	for _, t := range tareas {
		if !t.foco && time.Since(inicio) > presupuestoCommits {
			out.SoloFoco = true
			break
		}
		antes, ok := fichaEn(root, t.ficha.ID, t.desde)
		if !ok {
			continue
		}
		cambios := cambiosQueDefinen(antes, t.ficha)
		if len(cambios) == 0 {
			continue
		}
		out.Tareas = append(out.Tareas, TareaPorRevisar{Tarea: t.frente.ID, Nombre: t.frente.Nombre, Modulo: t.ficha.ID, Desde: t.frente.TocadaEl, Cambios: cambios})
	}
	return out
}

// LineasPorRevisar son las líneas de TareasPorRevisar para el contexto de la IA; sin estado.yaml, ninguna.
func LineasPorRevisar(root string) []string {
	e, err := Load(root)
	if err != nil {
		return nil
	}
	r := TareasPorRevisar(root, e)
	var out []string
	for _, t := range r.Tareas {
		out = append(out, t.Linea())
	}
	if r.SoloFoco {
		out = append(out, "Se revisaron solo las tareas del objetivo en foco: git tardó más de lo previsto (qdd estado check las nombra con sus commits).")
	}
	return out
}

// fichaEn es la ficha del último commit anterior a esa hora.
func fichaEn(root, id string, en time.Time) (modules.Module, bool) {
	ruta := filepath.ToSlash(filepath.Join(modules.Dir, id+".yaml"))
	commit, err := exec.Command("git", "-C", root, "log", "-1", "--format=%H", "--before="+en.Format(time.RFC3339), "--", ruta).Output()
	if err != nil || strings.TrimSpace(string(commit)) == "" {
		return modules.Module{}, false
	}
	data, err := exec.Command("git", "-C", root, "show", strings.TrimSpace(string(commit))+":"+ruta).Output()
	if err != nil {
		return modules.Module{}, false
	}
	m, err := modules.Parse(data)
	if err != nil {
		return modules.Module{}, false
	}
	return m, true
}

// cambiosQueDefinen: solo las decisiones y el objetivo operativo cambian qué falta; las pruebas o las rutas, no.
func cambiosQueDefinen(antes, ahora modules.Module) []string {
	var out []string
	if !reflect.DeepEqual(antes.ObjetivoTecnologico.Decisiones, ahora.ObjetivoTecnologico.Decisiones) {
		out = append(out, "decisiones")
	}
	if antes.ObjetivoOperativo != ahora.ObjetivoOperativo {
		out = append(out, "objetivo operativo")
	}
	return out
}

// fichaExiste: una tarea solo nombra el módulo de una ficha que existe (nombrar no crea, como un objetivo).
func fichaExiste(root string, c CambioTarea) error {
	if !c.CambiaModulo || c.Modulo == "" {
		return nil
	}
	if _, ok := modules.Find(root, c.Modulo); ok {
		return nil
	}
	return fmt.Errorf("no se escribió: el módulo %q no tiene ficha en %s (qdd modules list)", c.Modulo, modules.Dir)
}
