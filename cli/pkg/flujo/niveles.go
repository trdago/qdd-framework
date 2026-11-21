package flujo

import (
	"fmt"
	"slices"
	"strings"
)

// Rule: Prohibido usar 'else'
// Flujos grandes, parte B (#186): niveles y grupos. Cada nodo tiene un nivel: 1 es un hito (se ve en el panorama), 2 un
// paso del camino principal y 3 un desvío o un rechazo. Se calcula desde el camino principal y la persona lo corrige en la
// tarjeta (Nodo.Nivel). Grupo junta pasos seguidos bajo un nombre: en el panorama, el grupo es una caja.

const (
	NivelHito   = 1
	NivelPaso   = 2
	NivelDesvio = 3
)

// CaminoPrincipal es el camino más largo desde el primer nodo, sin repetir nodos. En un empate gana la salida que va
// primero.
func CaminoPrincipal(f Flujo) []string {
	if len(f.Nodos) == 0 {
		return nil
	}
	porID := map[string]Nodo{}
	for _, n := range f.Nodos {
		porID[n.ID] = n
	}
	// desde[id] es el camino más largo desde id; enCurso corta los ciclos (una vuelta atrás no alarga el camino).
	desde := map[string][]string{}
	enCurso := map[string]bool{}
	var largo func(id string) []string
	largo = func(id string) []string {
		if c, ok := desde[id]; ok {
			return c
		}
		enCurso[id] = true
		var mejor []string
		for _, s := range porID[id].Siguen {
			if enCurso[s] {
				continue
			}
			if c := largo(s); len(c) > len(mejor) {
				mejor = c
			}
		}
		enCurso[id] = false
		desde[id] = append([]string{id}, mejor...)
		return desde[id]
	}
	return largo(f.Nodos[0].ID)
}

// Niveles da el nivel de cada nodo: el que puso la persona o el calculado.
func Niveles(f Flujo) map[string]int {
	camino := CaminoPrincipal(f)
	niveles := map[string]int{}
	for _, n := range f.Nodos {
		niveles[n.ID] = NivelDesvio
	}
	for i, id := range camino {
		niveles[id] = NivelPaso
		if i == 0 || i == len(camino)-1 {
			niveles[id] = NivelHito
		}
	}
	for _, n := range f.Nodos {
		if n.Nivel >= NivelHito && n.Nivel <= NivelDesvio {
			niveles[n.ID] = n.Nivel
		}
	}
	return niveles
}

// Grupos son los nombres de grupo del flujo, en el orden en que aparecen.
func Grupos(f Flujo) []string {
	var grupos []string
	for _, n := range f.Nodos {
		if n.Grupo == "" || slices.Contains(grupos, n.Grupo) {
			continue
		}
		grupos = append(grupos, n.Grupo)
	}
	return grupos
}

// IDGrupo es el id del nodo que representa un grupo plegado.
func IDGrupo(grupo string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(grupo) {
		r = sinAcento(r)
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "-") {
			b.WriteByte('-')
		}
	}
	return "grupo-" + strings.TrimSuffix(b.String(), "-")
}

func sinAcento(r rune) rune {
	i := strings.IndexRune("áéíóúüñ", r)
	if i < 0 {
		return r
	}
	return []rune("aeiouun")[len([]rune("áéíóúüñ"[:i]))]
}

// Vista es cómo se mira el flujo: el nivel, los grupos que en el panorama se ven con sus pasos y un nodo que se muestra
// aunque su nivel lo esconda (el que la persona eligió en la lista).
type Vista struct {
	Nivel    int
	Abiertos []string
	Mostrar  string
}

// Plegar devuelve el flujo que se dibuja con esa vista.
func Plegar(f Flujo, v Vista) Flujo {
	if v.Nivel >= NivelDesvio || v.Nivel < NivelHito || len(f.Nodos) == 0 {
		return f
	}
	p := plegado{Vista: v, f: f, niveles: Niveles(f)}
	return p.armar()
}

type plegado struct {
	Vista
	f       Flujo
	niveles map[string]int
}

// representante: el nodo mismo si se ve, su grupo si va plegado en el panorama, o "" si se esconde.
func (p plegado) representante(n Nodo) string {
	nivel := p.niveles[n.ID]
	if n.ID == p.Mostrar {
		return n.ID
	}
	if n.ID == p.f.Nodos[0].ID {
		nivel = NivelHito // el flujo empieza donde empezaba, aunque la persona lo haya bajado de nivel
	}
	if nivel == NivelDesvio {
		return ""
	}
	if p.Nivel == NivelPaso {
		return n.ID
	}
	if n.Grupo != "" && !p.abierto(n.Grupo) {
		return IDGrupo(n.Grupo)
	}
	if n.Grupo != "" || nivel == NivelHito {
		return n.ID
	}
	return ""
}

func (p plegado) abierto(grupo string) bool {
	return slices.ContainsFunc(p.Abiertos, func(a string) bool { return a == grupo || a == IDGrupo(grupo) })
}

func (p plegado) armar() Flujo {
	porID := map[string]Nodo{}
	rep := map[string]string{}
	for _, n := range p.f.Nodos {
		porID[n.ID] = n
		rep[n.ID] = p.representante(n)
	}
	var orden []string
	nodos := map[string]*Nodo{}
	escondidos := map[string][]string{} // a quién se le cuentan los nodos escondidos
	miembros := map[string][]Nodo{}
	for _, n := range p.f.Nodos {
		r := rep[n.ID]
		if r == "" {
			continue
		}
		miembros[r] = append(miembros[r], n)
		if nodos[r] != nil {
			continue
		}
		orden = append(orden, r)
		nodos[r] = p.nodoDe(r, n)
	}
	for _, n := range p.f.Nodos {
		r := rep[n.ID]
		if r == "" {
			continue
		}
		for _, destino := range alcanzados(n, porID, rep) {
			if destino == r || slices.Contains(nodos[r].Siguen, destino) {
				continue
			}
			nodos[r].Siguen = append(nodos[r].Siguen, destino)
		}
		escondidos[r] = p.escondidosDesde(n, porID, rep, escondidos[r])
	}
	out := Flujo{Formato: p.f.Formato, ID: p.f.ID, Nombre: p.f.Nombre, Descripcion: p.f.Descripcion, Comentarios: p.f.Comentarios}
	for _, r := range orden {
		n := nodos[r]
		n.Descripcion = p.resumen(r, miembros[r], escondidos[r], porID)
		out.Nodos = append(out.Nodos, *n)
	}
	return out
}

// nodoDe es el nodo que se dibuja: el original sin descripción, o la caja de un grupo.
func (p plegado) nodoDe(r string, n Nodo) *Nodo {
	if r != n.ID {
		return &Nodo{ID: r, Tipo: TipoPaso, Nombre: n.Grupo}
	}
	copia := n
	copia.Siguen, copia.Descripcion = nil, ""
	return &copia
}

// alcanzados son los representantes visibles a los que se llega desde n pasando solo por escondidos.
func alcanzados(n Nodo, porID map[string]Nodo, rep map[string]string) []string {
	var out []string
	vistos := map[string]bool{n.ID: true}
	cola := slices.Clone(n.Siguen)
	for len(cola) > 0 {
		id := cola[0]
		cola = cola[1:]
		if vistos[id] {
			continue
		}
		vistos[id] = true
		if rep[id] != "" {
			out = append(out, rep[id])
			continue
		}
		cola = append(cola, porID[id].Siguen...)
	}
	return out
}

// escondidosDesde suma a ya los escondidos a los que se llega desde n antes de un visible.
func (p plegado) escondidosDesde(n Nodo, porID map[string]Nodo, rep map[string]string, ya []string) []string {
	vistos := map[string]bool{n.ID: true}
	cola := slices.Clone(n.Siguen)
	for len(cola) > 0 {
		id := cola[0]
		cola = cola[1:]
		if vistos[id] || rep[id] != "" {
			continue
		}
		vistos[id] = true
		if !slices.Contains(ya, id) {
			ya = append(ya, id)
		}
		cola = append(cola, porID[id].Siguen...)
	}
	return ya
}

// resumen dice lo que el nodo o el grupo esconde: sus hitos, cuántos pasos y cuántos desvíos.
func (p plegado) resumen(r string, miembros []Nodo, escondidos []string, porID map[string]Nodo) string {
	var partes []string
	if miembros[0].ID != r {
		var hitos []string
		for _, m := range miembros {
			if p.niveles[m.ID] == NivelHito {
				hitos = append(hitos, m.Nombre)
			}
		}
		partes = append(partes, plural(len(miembros), "paso", "pasos"))
		if len(hitos) > 0 {
			partes = append(partes, strings.Join(hitos, " → "))
		}
	}
	pasos, desvios := 0, 0
	for _, id := range escondidos {
		if p.niveles[id] == NivelDesvio {
			desvios++
			continue
		}
		pasos++
	}
	if pasos > 0 {
		partes = append(partes, "+"+plural(pasos, "paso", "pasos"))
	}
	if desvios > 0 {
		partes = append(partes, "+"+plural(desvios, "desvío", "desvíos"))
	}
	return strings.Join(partes, " · ")
}

func plural(n int, uno, varios string) string {
	if n == 1 {
		return "1 " + uno
	}
	return fmt.Sprintf("%d %s", n, varios)
}
