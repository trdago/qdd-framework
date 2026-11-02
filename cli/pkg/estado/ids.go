package estado

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Ids de QDD (FND-109, pedido del usuario, 2026-11-02: «forzar una forma estándar de tener id… usando a qdd»): toda
// entidad que QDD administra tiene un id que QDD da o valida, como los findings (FND-NNN) y los pedidos. Nombrar una
// tarea, un objetivo o un ciclo nunca lo crea: crear es explícito (qdd estado nueva, o --nombre) y una referencia que no
// existe es un error que sugiere la más parecida, así la IA lo corrige en el siguiente intento sin dejar una tarea
// fantasma. El formato se exige al crear; un id de un archivo anterior sigue valiendo.

// MaxID es el largo máximo de un id; PalabrasID, cuántas palabras del nombre toma el id que da qdd.
const (
	MaxID      = 40
	PalabrasID = 4
	// maxSugerencias y maxAbiertas: lo que cabe en un error estilo chat.
	maxSugerencias = 3
	maxAbiertas    = 8
	maxNombre      = 40 // un nombre largo se acorta en la sugerencia: el id es lo que la IA necesita
	// umbralParecida: desde qué puntaje (0…1) una referencia se parece a una tarea; umbralDuplicada, desde qué parte de
	// las palabras del nombre más corto un nombre nuevo repite el de una abierta.
	umbralParecida  = 0.5
	umbralDuplicada = 0.8
)

var formatoID = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// palabrasVacias no cuentan para el id ni para parecerse (español e inglés).
var palabrasVacias = map[string]bool{
	"de": true, "del": true, "la": true, "el": true, "los": true, "las": true, "en": true, "y": true, "o": true, "e": true,
	"a": true, "al": true, "con": true, "por": true, "para": true, "un": true, "una": true, "que": true, "se": true,
	"su": true, "sus": true, "lo": true, "the": true, "of": true, "and": true, "to": true, "in": true, "for": true, "on": true,
}

var sinTilde = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "à", "a", "è", "e", "ì", "i", "ò", "o", "ù", "u", "ç", "c")

// palabras: las del texto en minúsculas, sin tildes ni palabras vacías, cada una una vez y en su orden.
func palabras(s string) []string {
	var out []string
	vistas := map[string]bool{}
	for _, p := range noSlug.Split(sinTilde.Replace(strings.ToLower(s)), -1) {
		if p == "" || palabrasVacias[p] || vistas[p] {
			continue
		}
		vistas[p] = true
		out = append(out, p)
	}
	return out
}

// SlugDe es el id que qdd da a un nombre: sus primeras PalabrasID palabras, unidas con guiones y hasta MaxID.
func SlugDe(nombre string) string {
	ps := palabras(nombre)
	if len(ps) > PalabrasID {
		ps = ps[:PalabrasID]
	}
	slug := strings.Join(ps, "-")
	if len(slug) > MaxID {
		slug = strings.TrimRight(slug[:MaxID], "-")
	}
	return slug
}

// IDValido: minúsculas, números y guiones simples, sin guion al borde y hasta MaxID.
func IDValido(id string) error {
	if len(id) > MaxID || !formatoID.MatchString(id) {
		return fmt.Errorf("%q no es un id de QDD: minúsculas, números y guiones (≤%d), p. ej. %q; o deja que qdd lo dé con 'qdd estado nueva \"<nombre>\"'", id, MaxID, ejemploID(id))
	}
	return nil
}

func ejemploID(id string) string {
	if s := SlugDe(id); s != "" {
		return s
	}
	return "barra-estado"
}

// idLibre: el slug tal cual, o con -2, -3… si ya lo usa una tarea.
func (e Estado) idLibre(base string) string {
	id := base
	for n := 2; e.buscar(id) >= 0; n++ {
		id = base + "-" + strconv.Itoa(n)
	}
	return id
}

// candidato es algo que se puede referenciar por id: una tarea, un objetivo o un ciclo.
type candidato struct {
	ID, Nombre string
	Abierto    bool
}

type puntuado struct {
	candidato
	puntaje float64
	orden   int
}

// parecidos ordena los candidatos que se parecen a ref: por las palabras que comparten con su id y su nombre, o por
// estar a una o dos letras del id (un error de tipeo). Los abiertos primero a igual puntaje.
func parecidos(ref string, cands []candidato) []candidato {
	buscadas := palabras(ref)
	var ps []puntuado
	for i, c := range cands {
		p := puntaje(ref, buscadas, c)
		if p < umbralParecida {
			continue
		}
		ps = append(ps, puntuado{c, p, i})
	}
	sort.SliceStable(ps, func(i, j int) bool {
		if ps[i].puntaje != ps[j].puntaje {
			return ps[i].puntaje > ps[j].puntaje
		}
		return ps[i].Abierto && !ps[j].Abierto
	})
	var out []candidato
	for _, p := range ps {
		out = append(out, p.candidato)
	}
	if len(out) > maxSugerencias {
		out = out[:maxSugerencias]
	}
	return out
}

func puntaje(ref string, buscadas []string, c candidato) float64 {
	mejor := 0.0
	if d := distancia(strings.ToLower(ref), c.ID); d <= 2 && d < len(c.ID)/2 {
		mejor = 0.9
	}
	if len(buscadas) == 0 {
		return mejor
	}
	suyas := map[string]bool{}
	for _, p := range append(palabras(c.ID), palabras(c.Nombre)...) {
		suyas[p] = true
	}
	comunes := 0
	for _, p := range buscadas {
		if suyas[p] {
			comunes++
		}
	}
	return max(mejor, float64(comunes)/float64(len(buscadas)))
}

// distancia es la de Levenshtein entre dos ids.
func distancia(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		cur := make([]int, len(rb)+1)
		cur[0] = i
		for j := 1; j <= len(rb); j++ {
			costo := 1
			if ra[i-1] == rb[j-1] {
				costo = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+costo)
		}
		prev = cur
	}
	return prev[len(rb)]
}

// desconocido arma el error de una referencia que no existe: la más parecida primero, o las abiertas si ninguna se
// parece, y cómo crear una nueva.
func desconocido(que, ref string, cands []candidato, crear string) error {
	msg := fmt.Sprintf("no hay %s %q", que, ref)
	sug := parecidos(ref, cands)
	if len(sug) > 0 {
		msg += ": ¿quisiste decir " + nombrado(sug[0]) + "?"
	}
	if len(sug) > 1 {
		var otras []string
		for _, c := range sug[1:] {
			otras = append(otras, nombrado(c))
		}
		msg += " También: " + strings.Join(otras, ", ") + "."
	}
	if len(sug) == 0 {
		msg += " (abiertas: " + abiertas(cands) + ")."
	}
	return errors.New(msg + " " + crear)
}

func nombrado(c candidato) string {
	if c.Nombre == "" || c.Nombre == c.ID {
		return c.ID
	}
	nombre := []rune(c.Nombre)
	if len(nombre) > maxNombre {
		nombre = append(nombre[:maxNombre-1], '…')
	}
	return c.ID + " («" + string(nombre) + "»)"
}

func abiertas(cands []candidato) string {
	var ids []string
	for _, c := range cands {
		if c.Abierto {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		return "ninguna"
	}
	if len(ids) > maxAbiertas {
		return strings.Join(ids[:maxAbiertas], ", ") + "…"
	}
	return strings.Join(ids, ", ")
}

const (
	crearTarea    = "Para una nueva: qdd estado nueva \"<nombre>\" (o --nombre)."
	crearObjetivo = "Para uno nuevo: --nombre y --criterio."
	crearCiclo    = "Para uno nuevo: qdd estado ciclo <nombre> <etapas…>."
)

func (e Estado) candidatosTarea() []candidato {
	out := make([]candidato, 0, len(e.Frentes))
	for _, f := range e.Frentes {
		out = append(out, candidato{ID: f.ID, Nombre: f.Nombre, Abierto: f.Estado == EstadoAbierto})
	}
	return out
}

func (e Estado) candidatosObjetivo() []candidato {
	out := make([]candidato, 0, len(e.Objetivos))
	for _, o := range e.Objetivos {
		out = append(out, candidato{ID: o.ID, Nombre: o.Nombre, Abierto: o.Estado == ObjetivoAbierto})
	}
	return out
}

func (e Estado) candidatosCiclo() []candidato {
	out := []candidato{{ID: CicloProyecto, Abierto: true}}
	nombres := make([]string, 0, len(e.Ciclos))
	for n := range e.Ciclos {
		nombres = append(nombres, n)
	}
	sort.Strings(nombres)
	for _, n := range nombres {
		out = append(out, candidato{ID: n, Abierto: true})
	}
	return out
}

// ValidarTarea dice si la tarea existe; si no, el error sugiere la más parecida (lo usan los pedidos con --tarea).
func (e Estado) ValidarTarea(id string) error {
	if e.buscar(id) >= 0 {
		return nil
	}
	return desconocido("una tarea", id, e.candidatosTarea(), crearTarea)
}

func (e Estado) validarObjetivo(id string) error {
	if e.indiceObjetivo(id) >= 0 {
		return nil
	}
	return desconocido("un objetivo", id, e.candidatosObjetivo(), crearObjetivo)
}

func (e Estado) validarCiclo(nombre string) error {
	if _, ok := e.etapasDelCiclo(nombre); ok {
		return nil
	}
	return desconocido("un ciclo", nombre, e.candidatosCiclo(), crearCiclo)
}

// casiIgual: la tarea abierta cuyo nombre repite el nuevo (la mayor parte de las palabras del más corto, al menos dos),
// o cuyo id es el que daría el nombre nuevo.
func (e Estado) casiIgual(nombre string) (Frente, bool) {
	nuevas := palabras(nombre)
	for _, f := range e.Abiertos() {
		if f.ID == SlugDe(nombre) || repite(nuevas, palabras(f.Nombre)) {
			return f, true
		}
	}
	return Frente{}, false
}

func repite(a, b []string) bool {
	corto := min(len(a), len(b))
	if corto < 2 {
		return false
	}
	en := map[string]bool{}
	for _, p := range b {
		en[p] = true
	}
	comunes := 0
	for _, p := range a {
		if en[p] {
			comunes++
		}
	}
	return float64(comunes)/float64(corto) >= umbralDuplicada
}

// puedeCrear: el id es de QDD y el nombre no repite el de una tarea abierta (salvo aunqueExista).
func (e Estado) puedeCrear(id, nombre string, aunqueExista bool) error {
	if err := IDValido(id); err != nil {
		return err
	}
	if f, ok := e.casiIgual(nombre); ok && !aunqueExista {
		return fmt.Errorf("ya hay una tarea abierta parecida: %s; usa esa, o crea esta con --aunque-exista si de verdad es otra", nombrado(candidato{ID: f.ID, Nombre: f.Nombre}))
	}
	return nil
}

// CambioNueva es una tarea nueva: qdd le da el id desde su nombre.
type CambioNueva struct {
	Nombre, Objetivo string
	DependeDe        []string
	AunqueExista     bool
}

// Nueva crea una tarea abierta con todas sus etapas en «·» y devuelve el id que le dio qdd (qdd estado nueva).
func Nueva(root string, c CambioNueva, now time.Time) (string, error) {
	nombre := strings.TrimSpace(c.Nombre)
	base := SlugDe(nombre)
	if base == "" {
		return "", errors.New("una tarea nueva necesita un nombre con al menos una palabra")
	}
	e, err := cargarParaEscribir(root)
	if err != nil {
		return "", err
	}
	id := e.idLibre(base)
	if err := e.puedeCrear(id, nombre, c.AunqueExista); err != nil {
		return "", err
	}
	if err := e.validarReferencias(c.Objetivo, c.DependeDe); err != nil {
		return "", err
	}
	f := frenteNuevo(e, Cambio{Frente: id, Nombre: nombre})
	f.Objetivo, f.DependeDe = c.Objetivo, c.DependeDe
	e.Frentes = append(e.Frentes, f)
	if err := reformar(&e.Frentes[len(e.Frentes)-1], e.EtapasDe(f), e.CicloDe(f)); err != nil {
		return "", err
	}
	return id, guardarValidado(root, e, id, now)
}

// validarReferencias: el objetivo ("" ninguno) y cada dependencia existen.
func (e Estado) validarReferencias(objetivo string, dependeDe []string) error {
	if objetivo != "" {
		if err := e.validarObjetivo(objetivo); err != nil {
			return err
		}
	}
	for _, d := range dependeDe {
		if err := e.ValidarTarea(d); err != nil {
			return fmt.Errorf("depende de una tarea que no existe: %w", err)
		}
	}
	return nil
}

// Sospechosas (doctor, FND-109): tareas abiertas que parecen creadas por un id equivocado. Una cuyo nombre es su id
// (set la creaba así, sin --nombre) o dos cuyos nombres casi se repiten. No se reparan solas: cuál es la buena lo decide
// la persona.
func Sospechosas(e Estado) []string {
	var out []string
	abiertas := e.Abiertos()
	for i, f := range abiertas {
		if f.Nombre == f.ID {
			out = append(out, f.ID+" se llama como su id (¿se creó por un id equivocado?)")
		}
		for _, g := range abiertas[i+1:] {
			if repite(palabras(f.Nombre), palabras(g.Nombre)) {
				out = append(out, f.ID+" y "+g.ID+" casi se llaman igual")
			}
		}
	}
	return out
}
