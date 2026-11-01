package estado

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Objetivos y dependencias (blueprint estado-por-tareas, pregunta «objetivos»; pedido del usuario 2026-11-01): un
// objetivo dice para qué se hacen sus tareas y cuándo se cumple; una tarea puede depender de otras y no termina antes
// que ellas. El panel recibe el árbol objetivo → tareas → dependientes ya resuelto.

// Estados de un objetivo.
const (
	ObjetivoAbierto    = "abierto"
	ObjetivoCumplido   = "cumplido"
	ObjetivoDescartado = "descartado"
)

// TareasListas es lo que dice un objetivo abierto con todas sus tareas terminadas: terminarlas no basta (EPT-O04).
const TareasListas = "tareas listas, falta verificar el criterio"

// Objetivo agrupa tareas con su criterio de éxito.
type Objetivo struct {
	ID        string `yaml:"id"`
	Nombre    string `yaml:"nombre"`
	Criterio  string `yaml:"criterio"`
	Estado    string `yaml:"estado"`
	Foco      bool   `yaml:"foco,omitempty"`
	Evidencia string `yaml:"evidencia,omitempty"` // de que se cumple el criterio
	Motivo    string `yaml:"motivo,omitempty"`    // de un descarte
	CerradoEl string `yaml:"cerrado_el,omitempty"`
}

// usaFormato2: objetivos o dependencias. Un qdd anterior rechaza el formato 2 en vez de leerlo y perderlos al escribir.
func (e Estado) usaFormato2() bool {
	if len(e.Objetivos) > 0 {
		return true
	}
	for _, f := range e.Frentes {
		if f.Objetivo != "" || len(f.DependeDe) > 0 {
			return true
		}
	}
	return false
}

func (e Estado) objetivo(id string) (Objetivo, bool) {
	for _, o := range e.Objetivos {
		if o.ID == id {
			return o, true
		}
	}
	return Objetivo{}, false
}

func (e Estado) indiceObjetivo(id string) int {
	for i, o := range e.Objetivos {
		if o.ID == id {
			return i
		}
	}
	return -1
}

// TareaTerminada: cerrada, o con todas sus etapas en ✅ o —. Una descartada no termina: quien la espera tiene que
// quitarla de sus dependencias.
func (e Estado) TareaTerminada(f Frente) bool {
	return f.Estado == EstadoCerrado || (f.Estado == EstadoAbierto && f.Terminado(e.Etapas))
}

// pendientesDe son las dependencias de f que no terminaron (las que no existen no cuentan aquí: las dice EPT-D01).
func (e Estado) pendientesDe(f Frente) []Frente {
	var out []Frente
	for _, id := range f.DependeDe {
		i := e.buscar(id)
		if i >= 0 && !e.TareaTerminada(e.Frentes[i]) {
			out = append(out, e.Frentes[i])
		}
	}
	return out
}

// ultimaQueAplica es la última etapa de f que no está en —; "" si ninguna aplica.
func (e Estado) ultimaQueAplica(f Frente) string {
	for i := len(e.Etapas) - 1; i >= 0; i-- {
		if NormalizarEstado(f.Celdas[e.Etapas[i]].Estado) != NoAplica {
			return e.Etapas[i]
		}
	}
	return ""
}

func (e Estado) alcanza(desde, hasta string, vistos map[string]bool) bool {
	if desde == hasta {
		return true
	}
	if vistos[desde] {
		return false
	}
	vistos[desde] = true
	i := e.buscar(desde)
	if i < 0 {
		return false
	}
	for _, d := range e.Frentes[i].DependeDe {
		if e.alcanza(d, hasta, vistos) {
			return true
		}
	}
	return false
}

// checkDependencias (EPT-D01, EPT-D02).
func (e Estado) checkDependencias(f Frente) []Violacion {
	var vs []Violacion
	for _, d := range f.DependeDe {
		if d == f.ID {
			vs = append(vs, Violacion{ID: "EPT-D01", Frente: f.ID, Mensaje: "depende de sí misma"})
			continue
		}
		if e.buscar(d) < 0 {
			vs = append(vs, Violacion{ID: "EPT-D01", Frente: f.ID, Mensaje: fmt.Sprintf("depende de %q, que no está en el archivo", d)})
			continue
		}
		if e.alcanza(d, f.ID, map[string]bool{}) {
			vs = append(vs, Violacion{ID: "EPT-D01", Frente: f.ID, Mensaje: fmt.Sprintf("depender de %q hace un ciclo", d)})
		}
	}
	pend := e.pendientesDe(f)
	if len(pend) == 0 || f.Estado == EstadoDescartado {
		return vs
	}
	nombres := nombresDe(pend)
	if f.Estado == EstadoCerrado {
		return append(vs, Violacion{ID: "EPT-D02", Frente: f.ID, Mensaje: "cerrada antes que sus dependencias: espera a " + nombres})
	}
	if u := e.ultimaQueAplica(f); u != "" && NormalizarEstado(f.Celdas[u].Estado) == Hecho {
		vs = append(vs, Violacion{ID: "EPT-D02", Frente: f.ID, Etapa: u, Mensaje: "espera a " + nombres + ": su última etapa no pasa a ✅ hasta que terminen"})
	}
	return vs
}

func nombresDe(fs []Frente) string {
	n := make([]string, 0, len(fs))
	for _, f := range fs {
		n = append(n, f.Nombre)
	}
	return strings.Join(n, ", ")
}

// checkObjetivos (EPT-O01…O05).
func (e Estado) checkObjetivos(prev *Estado) []Violacion {
	var vs []Violacion
	vistos := map[string]bool{}
	focos := 0
	for _, o := range e.Objetivos {
		vs = append(vs, checkObjetivo(o, vistos)...)
		vistos[o.ID] = true
		if o.Foco && o.Estado == ObjetivoAbierto {
			focos++
		}
	}
	if focos > 1 {
		vs = append(vs, Violacion{ID: "EPT-O03", Mensaje: fmt.Sprintf("%d objetivos abiertos son el foco: a lo más uno", focos)})
	}
	for _, f := range e.Frentes {
		if f.Objetivo != "" && !vistos[f.Objetivo] {
			vs = append(vs, Violacion{ID: "EPT-O02", Frente: f.ID, Mensaje: fmt.Sprintf("sirve al objetivo %q, que no está en el archivo", f.Objetivo)})
		}
	}
	if prev == nil {
		return vs
	}
	for _, o := range prev.Objetivos {
		if _, ok := e.objetivo(o.ID); !ok {
			vs = append(vs, Violacion{ID: "EPT-O05", Frente: o.ID, Mensaje: "el objetivo desapareció: ciérralo cumplido o descartado con 'qdd estado objetivo', no lo borres"})
		}
	}
	return vs
}

func checkObjetivo(o Objetivo, vistos map[string]bool) []Violacion {
	var vs []Violacion
	if strings.TrimSpace(o.ID) == "" || strings.TrimSpace(o.Nombre) == "" || strings.TrimSpace(o.Criterio) == "" {
		vs = append(vs, Violacion{ID: "EPT-O01", Frente: o.ID, Mensaje: "un objetivo tiene id, nombre y criterio de éxito"})
	}
	if vistos[o.ID] {
		vs = append(vs, Violacion{ID: "EPT-O01", Frente: o.ID, Mensaje: "id de objetivo repetido"})
	}
	vs = append(vs, checkTexto(o.ID, "", o.Nombre, o.Criterio, o.Evidencia, o.Motivo)...)
	switch o.Estado {
	case ObjetivoAbierto:
	case ObjetivoCumplido:
		if strings.TrimSpace(o.Evidencia) == "" {
			vs = append(vs, Violacion{ID: "EPT-O04", Frente: o.ID, Mensaje: "cumplido sin evidencia de su criterio: terminar sus tareas no basta"})
		}
	case ObjetivoDescartado:
		if strings.TrimSpace(o.Motivo) == "" {
			vs = append(vs, Violacion{ID: "EPT-O05", Frente: o.ID, Mensaje: "descartado sin motivo"})
		}
	default:
		vs = append(vs, Violacion{ID: "EPT-O01", Frente: o.ID, Mensaje: fmt.Sprintf("estado del objetivo %q: debe ser abierto, cumplido o descartado", o.Estado)})
	}
	return vs
}

// EstadoObjetivo es el estado de un objetivo y su texto: ✅ con su evidencia, — descartado con su motivo, o ⏳ con
// cuántas de sus tareas están listas (TareasListas si lo están todas).
func (e Estado) EstadoObjetivo(o Objetivo) (string, string) {
	if o.Estado == ObjetivoCumplido {
		return Hecho, o.Evidencia
	}
	if o.Estado == ObjetivoDescartado {
		return NoAplica, "descartado: " + o.Motivo
	}
	total, listas := 0, 0
	for _, f := range e.Frentes {
		if f.Objetivo != o.ID || f.Estado == EstadoDescartado {
			continue
		}
		total++
		if e.TareaTerminada(f) {
			listas++
		}
	}
	if total > 0 && listas == total {
		return Espera, TareasListas
	}
	return Espera, fmt.Sprintf("%d/%d tareas listas", listas, total)
}

// Pregunta es lo que el panel le envía al agente al pinchar una tarea (EPT-A03).
func PreguntaTarea(f Frente) string {
	return fmt.Sprintf("¿Cómo vamos con «%s»? Avance, bloqueos y siguiente paso; actualiza sus celdas con evidencia.", f.Nombre)
}

// PreguntaObjetivo es lo que el panel le envía al agente al pinchar un objetivo (EPT-A03).
func PreguntaObjetivo(o Objetivo) string {
	return fmt.Sprintf("¿Cómo vamos con el objetivo «%s» (criterio: %s)? Avance de sus tareas, bloqueos y siguiente paso; actualiza el estado con evidencia.", o.Nombre, o.Criterio)
}

// FilaArbol es una tarea en el árbol: su prefijo de dibujo («│  └─ ») y a quién espera.
type FilaArbol struct {
	Frente  Frente
	Prefijo string
	Notas   []string
}

// GrupoArbol son las filas de un objetivo (Objetivo nil: las tareas sin objetivo).
type GrupoArbol struct {
	Objetivo *Objetivo
	Filas    []FilaArbol
}

// Arbol agrupa las tareas que incluir deja por objetivo (el foco primero, después en orden del archivo, y al final las
// sin objetivo) y, en cada grupo, cuelga cada tarea de su primera dependencia visible del mismo grupo. Sin objetivos
// hay un solo grupo y sus raíces van sin conector. Las dependencias que no terminaron y no son su padre se dicen en Notas.
func (e Estado) Arbol(incluir func(Frente) bool) []GrupoArbol {
	conObjetivos := len(e.Objetivos) > 0
	var grupos []GrupoArbol
	for _, o := range e.objetivosEnOrden() {
		g := e.grupo(o.ID, incluir, true)
		if len(g) == 0 && o.Estado != ObjetivoAbierto {
			continue
		}
		obj := o
		grupos = append(grupos, GrupoArbol{Objetivo: &obj, Filas: g})
	}
	if sin := e.grupo("", incluir, conObjetivos); len(sin) > 0 {
		grupos = append(grupos, GrupoArbol{Filas: sin})
	}
	return grupos
}

func (e Estado) objetivosEnOrden() []Objetivo {
	var out []Objetivo
	for _, o := range e.Objetivos {
		if o.Foco && o.Estado == ObjetivoAbierto {
			out = append(out, o)
		}
	}
	for _, o := range e.Objetivos {
		if !(o.Foco && o.Estado == ObjetivoAbierto) {
			out = append(out, o)
		}
	}
	return out
}

// grupoDe: el objetivo de f si existe; si no, el grupo sin objetivo.
func (e Estado) grupoDe(f Frente) string {
	if _, ok := e.objetivo(f.Objetivo); ok {
		return f.Objetivo
	}
	return ""
}

func (e Estado) grupo(objetivo string, incluir func(Frente) bool, conectarRaices bool) []FilaArbol {
	var miembros []Frente
	esta := map[string]bool{}
	for _, f := range e.Frentes {
		if incluir(f) && e.grupoDe(f) == objetivo {
			miembros = append(miembros, f)
			esta[f.ID] = true
		}
	}
	padre := map[string]string{}
	hijos := map[string][]Frente{}
	var raices []Frente
	for _, f := range miembros {
		for _, d := range f.DependeDe {
			if esta[d] && d != f.ID {
				padre[f.ID] = d
				break
			}
		}
		if p, ok := padre[f.ID]; ok {
			hijos[p] = append(hijos[p], f)
			continue
		}
		raices = append(raices, f)
	}
	var filas []FilaArbol
	vistos := map[string]bool{}
	var visita func(f Frente, ancestros []bool, ultimo, conector bool)
	visita = func(f Frente, ancestros []bool, ultimo, conector bool) {
		if vistos[f.ID] {
			return
		}
		vistos[f.ID] = true
		var b strings.Builder
		for _, u := range ancestros {
			b.WriteString(map[bool]string{true: "   ", false: "│  "}[u])
		}
		if conector {
			b.WriteString(map[bool]string{true: "└─ ", false: "├─ "}[ultimo])
		}
		filas = append(filas, FilaArbol{Frente: f, Prefijo: b.String(), Notas: e.notas(f)})
		siguientes := ancestros
		if conector {
			siguientes = append(append([]bool{}, ancestros...), ultimo)
		}
		var hs []Frente
		for _, h := range hijos[f.ID] {
			if !vistos[h.ID] {
				hs = append(hs, h)
			}
		}
		for i, h := range hs {
			visita(h, siguientes, i == len(hs)-1, true)
		}
	}
	for i, r := range raices {
		visita(r, nil, i == len(raices)-1, conectarRaices)
	}
	for _, f := range miembros { // un ciclo (EPT-D01) no deja una tarea sin dibujar
		visita(f, nil, true, conectarRaices)
	}
	return filas
}

// notas: «espera a X» por cada dependencia que no terminó, la de su padre incluida; la de otro objetivo lo dice.
func (e Estado) notas(f Frente) []string {
	var out []string
	for _, p := range e.pendientesDe(f) {
		if e.grupoDe(p) != e.grupoDe(f) {
			out = append(out, "espera a "+p.Nombre+" (otro objetivo)")
			continue
		}
		out = append(out, "espera a "+p.Nombre)
	}
	return out
}

// CambioObjetivo crea o actualiza un objetivo. Foco lo marca y desmarca a los demás.
type CambioObjetivo struct {
	ID, Nombre, Criterio string
	Foco                 bool
}

// GuardarObjetivo crea o actualiza un objetivo (qdd estado objetivo). Uno nuevo necesita nombre y criterio (EPT-O01).
func GuardarObjetivo(root string, c CambioObjetivo, now time.Time) error {
	e, err := cargarParaEscribir(root)
	if err != nil {
		return err
	}
	i := e.indiceObjetivo(c.ID)
	if i < 0 {
		e.Objetivos = append(e.Objetivos, Objetivo{ID: c.ID, Nombre: c.ID, Estado: ObjetivoAbierto})
		i = len(e.Objetivos) - 1
	}
	if c.Nombre != "" {
		e.Objetivos[i].Nombre = c.Nombre
	}
	if c.Criterio != "" {
		e.Objetivos[i].Criterio = c.Criterio
	}
	if c.Foco {
		for j := range e.Objetivos {
			e.Objetivos[j].Foco = j == i
		}
	}
	return guardarObjetivoValidado(root, e, c.ID, now)
}

// CerrarObjetivo lo cumple con la evidencia de su criterio o lo descarta con su motivo (EPT-O04, EPT-O05).
func CerrarObjetivo(root, id, texto string, descartado bool, now time.Time) error {
	if strings.TrimSpace(texto) == "" && descartado {
		return errors.New("descartar un objetivo exige su motivo (--motivo)")
	}
	if strings.TrimSpace(texto) == "" {
		return errors.New("un objetivo se cumple con evidencia de su criterio (--evidencia): terminar sus tareas no basta")
	}
	e, err := cargarParaEscribir(root)
	if err != nil {
		return err
	}
	i := e.indiceObjetivo(id)
	if i < 0 {
		return fmt.Errorf("no hay un objetivo %q", id)
	}
	o := &e.Objetivos[i]
	o.Estado, o.Evidencia, o.Motivo, o.Foco = ObjetivoCumplido, texto, "", false
	if descartado {
		o.Estado, o.Evidencia, o.Motivo = ObjetivoDescartado, "", texto
	}
	o.CerradoEl = now.Format(time.RFC3339)
	return guardarObjetivoValidado(root, e, id, now)
}

// ReabrirObjetivo vuelve a abrir un objetivo cumplido o descartado.
func ReabrirObjetivo(root, id string, now time.Time) error {
	e, err := cargarParaEscribir(root)
	if err != nil {
		return err
	}
	i := e.indiceObjetivo(id)
	if i < 0 {
		return fmt.Errorf("no hay un objetivo %q", id)
	}
	e.Objetivos[i].Estado, e.Objetivos[i].Evidencia, e.Objetivos[i].Motivo, e.Objetivos[i].CerradoEl = ObjetivoAbierto, "", "", ""
	return guardarObjetivoValidado(root, e, id, now)
}

func guardarObjetivoValidado(root string, e Estado, id string, now time.Time) error {
	var propias []string
	for _, v := range e.checkObjetivos(nil) {
		if v.Frente == id || v.ID == "EPT-O03" {
			propias = append(propias, v.String())
		}
	}
	if len(propias) > 0 {
		return fmt.Errorf("no se escribió: %s", strings.Join(propias, "; "))
	}
	e.Actualizado = now.Format(time.RFC3339)
	return Save(root, e)
}

// CambioTarea cambia lo estructural de una tarea: su nombre, su objetivo y sus dependencias. CambiaObjetivo y
// CambiaDependencias distinguen «no tocar» de «dejar vacío».
type CambioTarea struct {
	ID, Nombre, Objetivo string
	DependeDe            []string
	CambiaObjetivo       bool
	CambiaDependencias   bool
}

// GuardarTarea crea la tarea si no existe (todas sus etapas en «·») y aplica el cambio (qdd estado tarea).
func GuardarTarea(root string, c CambioTarea, now time.Time) error {
	e, err := cargarParaEscribir(root)
	if err != nil {
		return err
	}
	i := e.buscar(c.ID)
	if i < 0 {
		e.Frentes = append(e.Frentes, frenteNuevo(e, Cambio{Frente: c.ID, Nombre: c.Nombre}))
		i = len(e.Frentes) - 1
	}
	if c.Nombre != "" {
		e.Frentes[i].Nombre = c.Nombre
	}
	if c.CambiaObjetivo {
		e.Frentes[i].Objetivo = c.Objetivo
	}
	if c.CambiaDependencias {
		e.Frentes[i].DependeDe = c.DependeDe
	}
	return guardarValidado(root, e, c.ID, now)
}
