package estado

import (
	"fmt"
	"time"
)

// DatosPanel es lo que 'qdd estado ver --json' entrega a quien dibuja el estado fuera de la CLI (el mod de Claude Code,
// RFC 0006 §3.4): ya resuelto, para que el que dibuja no interprete reglas ni lea el YAML.
type DatosPanel struct {
	Formato      int           `json:"formato"`
	Actualizado  string        `json:"actualizado,omitempty"`
	Resumen      string        `json:"resumen"`
	Etapas       []string      `json:"etapas"`
	Abreviaturas []string      `json:"abreviaturas"`
	Frentes      []FrentePanel `json:"frentes"`
	Ocultos      int           `json:"ocultos"` // pendientes que no entran en el tope de DatosPendientes (los más antiguos)
	// Totales de las tareas pendientes, entren o no en el panel: los mismos que la barra (FND-086).
	Tareas  int         `json:"tareas"`
	Esperan int         `json:"esperan"`
	Fallan  int         `json:"fallan"`
	Esperas []Pendiente `json:"esperas"`
	Fallas  []Pendiente `json:"fallas"`
	Actual  string      `json:"actual,omitempty"` // la última tarea activada, si sigue en curso (§3.13)
	// Hace es cuánto pasó desde la última escritura («hace 9 h»); Viejo, que hay tareas pendientes y nadie tocó el
	// tablero en TableroViejo (FND-088). Sin fecha legible van vacíos: no se inventan.
	Hace  string `json:"hace,omitempty"`
	Viejo bool   `json:"viejo,omitempty"`
	// Objetivos en el orden en que se dibujan (el foco primero); Frentes viene en orden de árbol y cada fila dice su
	// Objetivo ("" son las tareas sin objetivo, al final). Vacío si el tablero no usa objetivos.
	Objetivos []ObjetivoPanel `json:"objetivos,omitempty"`
	// Ciclos: los propios (no «proyecto») de los objetivos que se ven, con sus etapas; las filas de un objetivo con
	// ciclo traen un símbolo por etapa de ese ciclo (EPT-K05).
	Ciclos []CicloPanel `json:"ciclos,omitempty"`
}

// ObjetivoPanel es la cabecera de un grupo del árbol, ya resuelta: su estado (Simbolo y Resumen) y la pregunta que el
// panel le envía al agente al pincharlo (EPT-A03).
type ObjetivoPanel struct {
	ID       string `json:"id"`
	Nombre   string `json:"nombre"`
	Criterio string `json:"criterio"`
	Foco     bool   `json:"foco,omitempty"`
	Simbolo  string `json:"simbolo"`
	Resumen  string `json:"resumen"`
	Pregunta string `json:"pregunta"`
	Ciclo    string `json:"ciclo,omitempty"` // el ciclo propio que siguen sus tareas; sin él, el del proyecto
}

// CicloPanel es un ciclo propio con sus etapas y sus abreviaturas, para la cabecera de sus objetivos.
type CicloPanel struct {
	Nombre       string   `json:"nombre"`
	Etapas       []string `json:"etapas"`
	Abreviaturas []string `json:"abreviaturas"`
}

func cicloPropio(o Objetivo) string {
	if o.Ciclo == CicloProyecto {
		return ""
	}
	return o.Ciclo
}

// TableroViejo: con tareas pendientes y sin escrituras en este tiempo, el panel avisa que el tablero puede no estar al
// día (lo que dice ⏳ ya pudo pasar).
const TableroViejo = 2 * time.Hour

// hace dice cuánto pasó, redondeado hacia abajo: «hace 20 min», «hace 9 h», «hace 2 d».
func hace(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("hace %d min", int(d.Minutes()))
	}
	if d < 48*time.Hour {
		return fmt.Sprintf("hace %d h", int(d.Hours()))
	}
	return fmt.Sprintf("hace %d d", int(d.Hours()/24))
}

// FrentePanel es una fila: un símbolo por etapa, en el orden de Etapas.
type FrentePanel struct {
	ID       string   `json:"id"`
	Nombre   string   `json:"nombre"`
	Estado   string   `json:"estado"`
	Simbolos []string `json:"simbolos"`
	// Objetivo, Prefijo («│  └─ ») y Notas («espera a X») dibujan el árbol tal cual (EPT-A02); Pregunta es lo que el
	// panel le envía al agente al pinchar la tarea (EPT-A03).
	Objetivo string   `json:"objetivo,omitempty"`
	Prefijo  string   `json:"prefijo,omitempty"`
	Notas    []string `json:"notas,omitempty"`
	Pregunta string   `json:"pregunta"`
	// Actividad solo va en una tarea en curso (§3.13): el panel muestra «● running» y destella cuando cambia Pulso.
	Actividad *ActividadPanel `json:"actividad,omitempty"`
}

// ActividadPanel es la señal de vida de una tarea en curso, ya resuelta para el que dibuja.
type ActividadPanel struct {
	Running bool   `json:"running"`
	Pulso   string `json:"pulso"`
	Fuente  string `json:"fuente"`
}

// Pendiente es un ⏳ (qué espera y de quién) o un ❌ (la falla nombrada).
type Pendiente struct {
	Frente  string `json:"frente"`
	Etapa   string `json:"etapa"`
	Texto   string `json:"texto"`
	DeQuien string `json:"de_quien,omitempty"`
}

// Datos arma DatosPanel con los frentes abiertos y los cerrados hace menos de RecienCerrado.
func Datos(e Estado, now time.Time) DatosPanel {
	return datos(e, now, func(f Frente) bool { return f.EnAlcance(AlcanceAbiertos, now) })
}

// DatosPendientes (§3.9, panel de Claude Code) deja solo los frentes con algo por hacer: abiertos y con alguna etapa
// que no esté en ✅ ni en — (una en «·» todavía está por hacer), y de ellos los tope más nuevos (los últimos del archivo, donde 'qdd estado set' agrega).
// Un frente terminado, cerrado o descartado sale del panel en el acto; sigue en el archivo y en 'qdd estado ver --todos'.
// Ocultos cuenta los pendientes más antiguos que no entran.
func DatosPendientes(e Estado, now time.Time, tope int) DatosPanel {
	return DatosEnCurso(e, Actividades{}, now, tope)
}

// DatosEnCurso es DatosPendientes con la actividad (§3.13): una tarea abierta en curso siempre se ve, aunque no esté
// entre las tope más nuevas, y lleva su Actividad. Ocultos cuenta solo las pendientes que no se ven.
func DatosEnCurso(e Estado, act Actividades, now time.Time, tope int) DatosPanel {
	enCurso := EnCurso(act, now)
	visibles := map[string]bool{}
	var pendientes []string
	for _, f := range e.Frentes {
		if f.Estado != EstadoAbierto {
			continue
		}
		if _, ok := enCurso[f.ID]; ok {
			visibles[f.ID] = true
			continue
		}
		if !f.Terminado(e.EtapasDe(f)) {
			pendientes = append(pendientes, f.ID)
		}
	}
	ocultos := max(len(pendientes)-tope, 0)
	for _, id := range pendientes[ocultos:] {
		visibles[id] = true
	}
	d := datos(e, now, func(f Frente) bool { return f.Estado == EstadoAbierto && visibles[f.ID] })
	d.Ocultos = ocultos
	for i, f := range d.Frentes {
		a, ok := enCurso[f.ID]
		if !ok {
			continue
		}
		d.Frentes[i].Actividad = &ActividadPanel{Running: true, Pulso: a.Pulso, Fuente: a.Fuente}
	}
	if _, ok := enCurso[act.Ultima]; ok && visibles[act.Ultima] {
		d.Actual = act.Ultima
	}
	return d
}

// Totales cuenta las tareas pendientes (abiertas y no terminadas) y cada ⏳ y cada ❌ que tienen. La barra y el panel
// usan esta misma cuenta (FND-086).
type Totales struct {
	Tareas, Esperan, Fallan int
}

func (e Estado) Totales() Totales {
	var t Totales
	for _, f := range e.Abiertos() {
		if f.Terminado(e.EtapasDe(f)) {
			continue
		}
		t.Tareas++
		for _, etapa := range e.EtapasDe(f) {
			switch NormalizarEstado(f.Celdas[etapa].Estado) {
			case Espera:
				t.Esperan++
			case Falla:
				t.Fallan++
			}
		}
	}
	return t
}

// Terminado: todas las etapas en ✅ o en — (sin celdas vacías, «·», ⏳ ni ❌).
func (f Frente) Terminado(etapas []string) bool {
	for _, etapa := range etapas {
		estado := NormalizarEstado(f.Celdas[etapa].Estado)
		if estado != Hecho && estado != NoAplica {
			return false
		}
	}
	return true
}

func datos(e Estado, now time.Time, incluir func(Frente) bool) DatosPanel {
	t := e.Totales()
	d := DatosPanel{
		Formato: e.Formato, Actualizado: e.Actualizado, Resumen: Statusline(e, now), Etapas: e.Etapas,
		Tareas: t.Tareas, Esperan: t.Esperan, Fallan: t.Fallan,
		Abreviaturas: abreviaturas(e.Etapas), Frentes: []FrentePanel{}, Esperas: []Pendiente{}, Fallas: []Pendiente{},
	}
	if cuando, err := time.Parse(time.RFC3339, e.Actualizado); err == nil && !now.Before(cuando) {
		d.Hace = hace(now.Sub(cuando))
		d.Viejo = t.Tareas > 0 && now.Sub(cuando) > TableroViejo
	}
	var usados []Objetivo
	for _, g := range e.Arbol(incluir) {
		objetivo := ""
		if g.Objetivo != nil {
			o := *g.Objetivo
			simbolo, resumen := e.EstadoObjetivo(o)
			d.Objetivos = append(d.Objetivos, ObjetivoPanel{ID: o.ID, Nombre: o.Nombre, Criterio: o.Criterio, Foco: o.Foco && o.Estado == ObjetivoAbierto, Simbolo: simbolo, Resumen: resumen, Pregunta: PreguntaObjetivo(o), Ciclo: cicloPropio(o)})
			usados = append(usados, o)
			objetivo = o.ID
		}
		for _, fa := range g.Filas {
			d.Frentes = append(d.Frentes, filaPanel(e, fa, objetivo, &d))
		}
	}
	for _, c := range ciclosUsados(usados) {
		etapas, _ := e.etapasDelCiclo(c)
		d.Ciclos = append(d.Ciclos, CicloPanel{Nombre: c, Etapas: etapas, Abreviaturas: abreviaturas(etapas)})
	}
	return d
}

func filaPanel(e Estado, fa FilaArbol, objetivo string, d *DatosPanel) FrentePanel {
	f := fa.Frente
	etapas := e.EtapasDe(f)
	fila := FrentePanel{ID: f.ID, Nombre: f.Nombre, Estado: f.Estado, Simbolos: make([]string, 0, len(etapas)),
		Objetivo: objetivo, Prefijo: fa.Prefijo, Notas: fa.Notas, Pregunta: PreguntaTarea(f)}
	for _, etapa := range etapas {
		c := f.Celdas[etapa]
		fila.Simbolos = append(fila.Simbolos, simbolo(c))
		// De una tarea cerrada o descartada no se espera nada, aunque conserve el símbolo (FND-151).
		if f.Estado != EstadoAbierto {
			continue
		}
		switch NormalizarEstado(c.Estado) {
		case Espera:
			d.Esperas = append(d.Esperas, Pendiente{Frente: f.Nombre, Etapa: etapa, Texto: c.Bloqueo, DeQuien: c.DeQuien})
		case Falla:
			d.Fallas = append(d.Fallas, Pendiente{Frente: f.Nombre, Etapa: etapa, Texto: c.Falla})
		}
	}
	return fila
}
