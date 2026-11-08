package flujo

import (
	"strings"

	"github.com/qdd-framework/qdd/pkg/materializar"
)

const (
	separacion   = 2  // columnas entre dos ramas lado a lado
	sangria      = 3  // lo que ocupa «├─ » en una rama con sangría
	minAnchoRama = 14 // más angosto que esto, las ramas van una debajo de otra
	minAncho     = 12 // más angosto que esto, un nodo ya no se lee
	maxDesvio    = 2  // nodos que caben en la etiqueta de un desvío
)

// bloque es un trozo de dibujo. entrada y salida son las columnas por donde se conecta arriba y abajo; una salida -1 dice
// que por ahí no sigue nada. tramos dice qué columnas de qué línea son de qué nodo (tramos.go): se mueven con el bloque.
type bloque struct {
	lineas  []string
	entrada int
	salida  int
	tramos  []tramo
}

// Dibujar es el flujo en líneas de texto, de hasta ancho columnas. estadoDe da los símbolos de una tarea del tablero.
func Dibujar(f Flujo, ancho int, estadoDe func(string) string) []string {
	return DibujarElegido(f, ancho, estadoDe, "")
}

// DibujarElegido resalta un nodo, sin moverlo: el que la persona está editando en el panel.
func DibujarElegido(f Flujo, ancho int, estadoDe func(string) string, elegido string) []string {
	lineas, _ := DibujarConSegmentos(f, ancho, estadoDe, elegido)
	return lineas
}

// DibujarConSegmentos es el mismo dibujo y, por cada línea, sus tramos con el nodo de cada uno (nil si la línea no
// tiene nodos): el panel hace pulsable cada nodo donde se ve.
func DibujarConSegmentos(f Flujo, ancho int, estadoDe func(string) string, elegido string) ([]string, [][]Segmento) {
	if len(f.Nodos) == 0 {
		return nil, nil
	}
	d := dibujante{f: f, porID: map[string]Nodo{}, hechos: map[string]bool{}, camino: map[string]bool{}, estadoDe: estadoDe, elegido: elegido}
	for _, n := range f.Nodos {
		d.porID[n.ID] = n
	}
	final := d.secuencia(f.Nodos[0].ID, nil, ancho)
	margen := max(0, (ancho-anchoMayor(final.lineas))/2)
	lineas := mover(final.lineas, margen)
	return lineas, segmentar(lineas, moverTramos(final.tramos, 0, margen))
}

type dibujante struct {
	f        Flujo
	porID    map[string]Nodo
	hechos   map[string]bool
	camino   map[string]bool // lo que está encima del nodo actual: volver a uno de ellos es un ciclo
	estadoDe func(string) string
	elegido  string
}

// secuencia dibuja desde id hasta una de las paradas (que no dibuja) o hasta el final.
func (d *dibujante) secuencia(id string, paradas []string, ancho int) bloque {
	var total bloque
	vacio := true
	var recorridos []string
	defer func() {
		for _, r := range recorridos {
			delete(d.camino, r)
		}
	}()
	for {
		if contiene(paradas, id) {
			if vacio {
				return bloque{lineas: []string{"│"}}
			}
			return total
		}
		if d.hechos[id] {
			return apilar(total, vacio, d.referencia(id, ancho), ancho)
		}
		n := d.porID[id]
		d.hechos[id] = true
		d.camino[id] = true
		recorridos = append(recorridos, id)
		total = apilar(total, vacio, d.nodo(n, ancho), ancho)
		vacio = false
		if len(n.Siguen) == 0 {
			total.salida = -1
			return total
		}
		siguen := n.Siguen
		if len(siguen) > 1 {
			total, siguen = d.colgarDesvios(total, siguen, paradas, ancho)
		}
		if len(siguen) == 1 {
			id = siguen[0]
			continue
		}
		encuentro := d.encuentro(siguen)
		rama := d.bifurcacion(siguen, paradas, encuentro, ancho)
		total = apilar(total, false, rama, ancho)
		if encuentro == "" || rama.salida < 0 {
			return total
		}
		id = encuentro
	}
}

// referencia es lo que se dibuja al llegar a un nodo que ya está en el dibujo.
func (d *dibujante) referencia(id string, ancho int) bloque {
	lineas := partir(d.textoReferencia(id, false), max(minAncho, ancho))
	return bloque{lineas: lineas, salida: -1, tramos: tramosDeLineas(lineas, id)}
}

func (d *dibujante) textoReferencia(id string, ciclo bool) string {
	if ciclo || d.camino[id] {
		return "↺ vuelve a " + d.porID[id].Nombre
	}
	return "→ sigue en " + d.porID[id].Nombre
}

// desvio es una rama corta que no vuelve al camino: se escribe como etiqueta al costado.
type desvio struct {
	rama    string
	texto   string
	nodos   []string
	atajo   string // la rama misma, si es un atajo: otra rama que se dibuja tiene que alcanzarla
	destino string // el nodo ya dibujado al que lleva («→ sigue en», «↺ vuelve a», «↷ salta a»), si lo nombra
}

// candidatos son los nodos que nombra la etiqueta, en orden.
func (dv desvio) candidatos() []string {
	if dv.destino == "" {
		return dv.nodos
	}
	return append(append([]string{}, dv.nodos...), dv.destino)
}

// colgarDesvios cuelga del nodo, al costado de su eje, las ramas que son desvíos (#186): un rechazo, una vuelta atrás o
// un atajo. Las demás siguen con todo el ancho. Si todas son desvíos, o no queda espacio al costado, nada cambia.
func (d *dibujante) colgarDesvios(total bloque, ramas, paradas []string, ancho int) (bloque, []string) {
	interior := ancho - total.salida - sangria
	if interior < minAncho {
		return total, ramas
	}
	var desvios []desvio
	var resto []string
	for _, r := range ramas {
		dv, ok := d.desvio(r, sin(ramas, r), paradas)
		if !ok {
			resto = append(resto, r)
			continue
		}
		dv.rama = r
		desvios = append(desvios, dv)
	}
	desvios, resto = d.atajosAlcanzados(desvios, resto)
	if len(resto) == 0 {
		desvios, resto = caminoEntreDesvios(desvios)
	}
	if len(desvios) == 0 || len(resto) == 0 {
		return total, ramas
	}
	lineas := append([]string{}, total.lineas...)
	tramos := append([]tramo{}, total.tramos...)
	margen := strings.Repeat(" ", total.salida)
	for _, dv := range desvios {
		for _, n := range dv.nodos {
			d.hechos[n] = true
		}
		for j, parte := range partir(dv.texto, interior) {
			prefijo := "│  "
			if j == 0 {
				prefijo = "├─ "
			}
			tramos = append(tramos, d.tramosDeEtiqueta(parte, len(lineas), total.salida+sangria, dv.candidatos())...)
			lineas = append(lineas, margen+prefijo+parte)
		}
	}
	return bloque{lineas: lineas, entrada: total.entrada, salida: total.salida, tramos: tramos}, resto
}

// atajosAlcanzados: un atajo solo se escribe si una rama que se dibuja llega a su destino. Dos ramas que se alcanzan
// entre sí no se tapan una a la otra: la que nadie más alcanza vuelve a dibujarse.
func (d *dibujante) atajosAlcanzados(desvios []desvio, resto []string) ([]desvio, []string) {
	for cambio := true; cambio; {
		cambio = false
		for i, dv := range desvios {
			if dv.atajo == "" || d.alcanzaOtra(dv.atajo, resto) {
				continue
			}
			desvios = append(append([]desvio{}, desvios[:i]...), desvios[i+1:]...)
			resto, cambio = append(resto, dv.atajo), true
			break
		}
	}
	return desvios, resto
}

// caminoEntreDesvios: si todas las ramas son desvíos, la que tiene más nodos es el camino. Si ninguna tiene más que
// las otras (un «sí» y un «no»), van lado a lado como siempre.
func caminoEntreDesvios(desvios []desvio) ([]desvio, []string) {
	mayor, empate := 0, false
	for i, dv := range desvios {
		if i == 0 || len(dv.nodos) > len(desvios[mayor].nodos) {
			mayor, empate = i, false
			continue
		}
		empate = empate || len(dv.nodos) == len(desvios[mayor].nodos)
	}
	if empate {
		return nil, nil
	}
	return append(append([]desvio{}, desvios[:mayor]...), desvios[mayor+1:]...), []string{desvios[mayor].rama}
}

// desvio dice si la rama r es un desvío y cómo se escribe: un nodo ya dibujado, un atajo a uno que otra rama alcanza más
// adelante, o hasta
// maxDesvio nodos en fila que terminan o vuelven a uno ya dibujado. Un nodo que otra rama también alcanza, uno elegido o
// una parada de más arriba no van en una etiqueta: se dibujan.
func (d *dibujante) desvio(r string, otras, paradas []string) (desvio, bool) {
	if contiene(paradas, r) || r == d.elegido {
		return desvio{}, false
	}
	if d.hechos[r] {
		return desvio{texto: d.textoReferencia(r, false), destino: r}, true
	}
	if d.alcanzaOtra(r, otras) {
		return desvio{texto: "↷ salta a " + d.porID[r].Nombre, atajo: r, destino: r}, true
	}
	var partes, nodos []string
	actual := r
	for len(nodos) < maxDesvio {
		n := d.porID[actual]
		if contiene(paradas, actual) || actual == d.elegido || d.alcanzaOtra(actual, otras) {
			return desvio{}, false
		}
		partes, nodos = append(partes, d.etiqueta(n)), append(nodos, actual)
		if len(n.Siguen) == 0 && n.Tipo != TipoResultado {
			return desvio{}, false // un paso sin salida es el final del camino, no un desvío
		}
		if len(n.Siguen) == 0 {
			return desvio{texto: strings.Join(partes, " → "), nodos: nodos}, true
		}
		if len(n.Siguen) > 1 {
			return desvio{}, false
		}
		actual = n.Siguen[0]
		if d.hechos[actual] || contiene(nodos, actual) {
			return desvio{texto: strings.Join(partes, " → ") + " " + d.textoReferencia(actual, contiene(nodos, actual)), nodos: nodos, destino: actual}, true
		}
	}
	return desvio{}, false
}

func (d *dibujante) alcanzaOtra(id string, otras []string) bool {
	for _, o := range otras {
		if contiene(d.recorrido(o), id) {
			return true
		}
	}
	return false
}

// etiqueta es el nodo en una línea: un resultado entre paréntesis, como su círculo, y el estado de su tarea.
func (d *dibujante) etiqueta(n Nodo) string {
	texto := n.Nombre
	if n.Tipo == TipoResultado {
		texto = "( " + n.Nombre + " )"
	}
	if n.Tarea == "" {
		return texto
	}
	return texto + " " + d.estadoDe(n.Tarea)
}

func sin(lista []string, quitar string) []string {
	var resto []string
	for _, x := range lista {
		if x != quitar {
			resto = append(resto, x)
		}
	}
	return resto
}

// encuentro es el primer nodo sin dibujar al que llegan todas las ramas: se dibuja una vez, debajo de ellas.
func (d *dibujante) encuentro(ramas []string) string {
	primero := d.recorrido(ramas[0])
	for _, candidato := range primero {
		if d.llegaDeTodas(candidato, ramas) {
			return candidato
		}
	}
	return ""
}

func (d *dibujante) llegaDeTodas(candidato string, ramas []string) bool {
	for _, r := range ramas {
		if !contiene(d.recorrido(r), candidato) {
			return false
		}
	}
	return true
}

// recorrido son los nodos sin dibujar que se alcanzan desde id, de más cerca a más lejos.
func (d *dibujante) recorrido(id string) []string {
	if d.hechos[id] {
		return nil
	}
	orden := []string{id}
	vistos := map[string]bool{id: true}
	for i := 0; i < len(orden); i++ {
		for _, s := range d.porID[orden[i]].Siguen {
			if vistos[s] || d.hechos[s] {
				continue
			}
			vistos[s] = true
			orden = append(orden, s)
		}
	}
	return orden
}

func (d *dibujante) bifurcacion(ramas, paradas []string, encuentro string, ancho int) bloque {
	topes := paradas
	if encuentro != "" {
		topes = append(append([]string{}, paradas...), encuentro)
	}
	porRama := (ancho - separacion*(len(ramas)-1)) / len(ramas)
	if porRama < minAnchoRama {
		return d.conSangria(ramas, topes, ancho)
	}
	bloques := make([]bloque, len(ramas))
	for i, r := range ramas {
		bloques[i] = d.secuencia(r, topes, porRama)
	}
	return ladoALado(bloques)
}

// conSangria pone las ramas una debajo de otra, colgando de un tronco en la columna 0.
func (d *dibujante) conSangria(ramas, topes []string, ancho int) bloque {
	interior := max(minAncho, ancho-sangria)
	var lineas []string
	var tramos []tramo
	sigue := false
	for i, r := range ramas {
		b := d.secuencia(r, topes, interior)
		sigue = sigue || b.salida >= 0
		tramos = append(tramos, moverTramos(b.tramos, len(lineas), sangria)...)
		primera, resto := "├─ ", "│  "
		if i == len(ramas)-1 {
			primera, resto = "└─ ", "   "
		}
		for j, l := range b.lineas {
			prefijo := resto
			if j == 0 {
				prefijo = primera
			}
			lineas = append(lineas, prefijo+l)
		}
	}
	if !sigue {
		return bloque{lineas: lineas, salida: -1, tramos: tramos}
	}
	// El encuentro cuelga del mismo tronco: la última rama deja de ser la última.
	for i, l := range lineas {
		lineas[i] = reabrirTronco(l)
	}
	return bloque{lineas: lineas, tramos: tramos}
}

func reabrirTronco(l string) string {
	if strings.HasPrefix(l, "└─ ") {
		return "├─ " + strings.TrimPrefix(l, "└─ ")
	}
	if strings.HasPrefix(l, "   ") {
		return "│  " + strings.TrimPrefix(l, "   ")
	}
	return l
}

// nodo es la caja de un paso o el círculo de un resultado.
func (d *dibujante) nodo(n Nodo, ancho int) bloque {
	interior := max(minAncho, ancho) - 4
	var textos []string
	textos = append(textos, partir(n.Nombre, interior)...)
	if n.Descripcion != "" {
		textos = append(textos, partir(n.Descripcion, interior)...)
	}
	if n.Tarea != "" {
		textos = append(textos, d.estadoDe(n.Tarea))
	}
	b := caja(textos)
	if n.Tipo == TipoResultado {
		b = circulo(textos)
	}
	if n.ID == d.elegido {
		b.lineas = resaltar(b.lineas, n.Tipo)
	}
	b.tramos = tramosDeLineas(b.lineas, n.ID)
	return b
}

// resaltar cambia solo los bordes: el nodo elegido ocupa lo mismo, y el dibujo no salta al elegir otro.
func resaltar(lineas []string, tipo string) []string {
	resaltadas := make([]string, len(lineas))
	for i, l := range lineas {
		interior := []rune(l)
		interior = interior[1 : len(interior)-1]
		resaltadas[i] = bordeResaltado(string(interior), tipo, i == 0, i == len(lineas)-1)
	}
	return resaltadas
}

func bordeResaltado(interior, tipo string, primera, ultima bool) string {
	if tipo == TipoResultado {
		return "▶" + interior + "◀"
	}
	if primera {
		return "┏" + strings.ReplaceAll(interior, "─", "━") + "┓"
	}
	if ultima {
		return "┗" + strings.ReplaceAll(interior, "─", "━") + "┛"
	}
	return "┃" + interior + "┃"
}

func caja(textos []string) bloque {
	interior := anchoMayor(textos)
	lineas := []string{"┌" + strings.Repeat("─", interior+2) + "┐"}
	for _, t := range textos {
		lineas = append(lineas, "│ "+rellenar(t, interior)+" │")
	}
	lineas = append(lineas, "└"+strings.Repeat("─", interior+2)+"┘")
	eje := (interior + 4) / 2
	return bloque{lineas: lineas, entrada: eje, salida: eje}
}

func circulo(textos []string) bloque {
	interior := anchoMayor(textos)
	lineas := make([]string, 0, len(textos))
	for _, t := range textos {
		lineas = append(lineas, "( "+rellenar(t, interior)+" )")
	}
	eje := (interior + 4) / 2
	return bloque{lineas: lineas, entrada: eje, salida: eje}
}

// apilar pone b debajo de a, con sus ejes en la misma columna. Si alinearlos saca el dibujo del ancho, un codo los une.
func apilar(a bloque, vacio bool, b bloque, ancho int) bloque {
	if vacio {
		return b
	}
	if a.salida < 0 {
		return a
	}
	desplazaA := max(0, b.entrada-a.salida)
	desplazaB := max(0, a.salida-b.entrada)
	if anchoMayor(a.lineas)+desplazaA > ancho || anchoMayor(b.lineas)+desplazaB > ancho {
		lineas := append(append([]string{}, a.lineas...), lineaDeUnion([]int{b.entrada}, a.salida, true))
		tramos := append(append([]tramo{}, a.tramos...), moverTramos(b.tramos, len(lineas), 0)...)
		return bloque{lineas: append(lineas, b.lineas...), entrada: a.entrada, salida: b.salida, tramos: tramos}
	}
	lineas := mover(a.lineas, desplazaA)
	lineas = append(lineas, strings.Repeat(" ", a.salida+desplazaA)+"│")
	tramos := append(moverTramos(a.tramos, 0, desplazaA), moverTramos(b.tramos, len(lineas), desplazaB)...)
	lineas = append(lineas, mover(b.lineas, desplazaB)...)
	salida := b.salida
	if salida >= 0 {
		salida += desplazaB
	}
	return bloque{lineas: lineas, entrada: a.entrada + desplazaA, salida: salida, tramos: tramos}
}

func mover(lineas []string, columnas int) []string {
	if columnas == 0 {
		return append([]string{}, lineas...)
	}
	margen := strings.Repeat(" ", columnas)
	movidas := make([]string, len(lineas))
	for i, l := range lineas {
		movidas[i] = margen + l
	}
	return movidas
}

// ladoALado junta las ramas con una línea que las abre arriba y, si alguna sigue, otra que las cierra abajo.
func ladoALado(ramas []bloque) bloque {
	anchos := make([]int, len(ramas))
	entradas := make([]int, len(ramas))
	var salidas []int
	var tramos []tramo
	columna, alto := 0, 0
	for i, r := range ramas {
		anchos[i] = anchoMayor(r.lineas)
		tramos = append(tramos, moverTramos(r.tramos, 1, columna)...)
		entradas[i] = columna + r.entrada
		if r.salida >= 0 {
			salidas = append(salidas, columna+r.salida)
		}
		columna += anchos[i] + separacion
		alto = max(alto, len(r.lineas))
	}
	eje := (entradas[0] + entradas[len(entradas)-1]) / 2
	lineas := []string{lineaDeUnion(entradas, eje, true)}
	for fila := 0; fila < alto; fila++ {
		lineas = append(lineas, filaLadoALado(ramas, anchos, fila))
	}
	if len(salidas) == 0 {
		return bloque{lineas: lineas, entrada: eje, salida: -1, tramos: tramos}
	}
	return bloque{lineas: append(lineas, lineaDeUnion(salidas, eje, false)), entrada: eje, salida: eje, tramos: tramos}
}

func filaLadoALado(ramas []bloque, anchos []int, fila int) string {
	var b strings.Builder
	for i, r := range ramas {
		b.WriteString(rellenar(celdaDeRama(r, fila), anchos[i]))
		if i < len(ramas)-1 {
			b.WriteString(strings.Repeat(" ", separacion))
		}
	}
	return strings.TrimRight(b.String(), " ")
}

// celdaDeRama es la línea de la rama en esa fila; pasada su altura, el «│» que baja hasta la unión si sigue.
func celdaDeRama(r bloque, fila int) string {
	if fila < len(r.lineas) {
		return r.lineas[fila]
	}
	if r.salida < 0 {
		return ""
	}
	return strings.Repeat(" ", r.salida) + "│"
}

// Lados de una celda de línea, para elegir su carácter.
const (
	arriba = 1 << iota
	abajo
	izquierda
	derecha
)

var trazos = map[int]string{
	arriba | abajo:                       "│",
	izquierda | derecha:                  "─",
	abajo | derecha:                      "┌",
	abajo | izquierda:                    "┐",
	arriba | derecha:                     "└",
	arriba | izquierda:                   "┘",
	arriba | abajo | derecha:             "├",
	arriba | abajo | izquierda:           "┤",
	abajo | izquierda | derecha:          "┬",
	arriba | izquierda | derecha:         "┴",
	arriba | abajo | izquierda | derecha: "┼",
}

// lineaDeUnion une columnas con el eje. Abriendo, el eje viene de arriba y las columnas bajan; cerrando, al revés.
func lineaDeUnion(columnas []int, eje int, abriendo bool) string {
	desde, hasta := eje, eje
	for _, c := range columnas {
		desde, hasta = min(desde, c), max(hasta, c)
	}
	ladoColumna, ladoEje := abajo, arriba
	if !abriendo {
		ladoColumna, ladoEje = arriba, abajo
	}
	celdas := make([]int, hasta+1)
	for c := desde; c < hasta; c++ {
		celdas[c] |= derecha
		celdas[c+1] |= izquierda
	}
	for _, c := range columnas {
		celdas[c] |= ladoColumna
	}
	celdas[eje] |= ladoEje
	var b strings.Builder
	b.WriteString(strings.Repeat(" ", desde))
	for _, lados := range celdas[desde:] {
		b.WriteString(trazos[lados])
	}
	return b.String()
}

// partir corta un texto en líneas de hasta ancho columnas, por palabras; una palabra más larga se corta.
func partir(texto string, ancho int) []string {
	var lineas []string
	actual := ""
	for _, palabra := range strings.Fields(texto) {
		for materializar.Ancho(palabra) > ancho {
			cabeza, cola := cortar(palabra, ancho)
			if actual != "" {
				lineas = append(lineas, actual)
				actual = ""
			}
			lineas = append(lineas, cabeza)
			palabra = cola
		}
		if actual == "" {
			actual = palabra
			continue
		}
		if materializar.Ancho(actual)+1+materializar.Ancho(palabra) > ancho {
			lineas = append(lineas, actual)
			actual = palabra
			continue
		}
		actual += " " + palabra
	}
	if actual == "" {
		return lineas
	}
	return append(lineas, actual)
}

func cortar(palabra string, ancho int) (string, string) {
	usado := 0
	runas := []rune(palabra)
	for i, r := range runas {
		w := materializar.Ancho(string(r))
		if usado+w > ancho {
			return string(runas[:i]), string(runas[i:])
		}
		usado += w
	}
	return palabra, ""
}

func anchoMayor(lineas []string) int {
	mayor := 0
	for _, l := range lineas {
		mayor = max(mayor, materializar.Ancho(l))
	}
	return mayor
}

func rellenar(s string, ancho int) string {
	return s + strings.Repeat(" ", max(0, ancho-materializar.Ancho(s)))
}
