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
)

// bloque es un trozo de dibujo. entrada y salida son las columnas por donde se conecta arriba y abajo; una salida -1 dice
// que por ahí no sigue nada.
type bloque struct {
	lineas  []string
	entrada int
	salida  int
}

// Dibujar es el flujo en líneas de texto, de hasta ancho columnas. estadoDe da los símbolos de una tarea del tablero.
func Dibujar(f Flujo, ancho int, estadoDe func(string) string) []string {
	if len(f.Nodos) == 0 {
		return nil
	}
	d := dibujante{f: f, porID: map[string]Nodo{}, hechos: map[string]bool{}, camino: map[string]bool{}, estadoDe: estadoDe}
	for _, n := range f.Nodos {
		d.porID[n.ID] = n
	}
	final := d.secuencia(f.Nodos[0].ID, nil, ancho)
	return centrar(final.lineas, ancho)
}

type dibujante struct {
	f        Flujo
	porID    map[string]Nodo
	hechos   map[string]bool
	camino   map[string]bool // lo que está encima del nodo actual: volver a uno de ellos es un ciclo
	estadoDe func(string) string
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
		if len(n.Siguen) == 1 {
			id = n.Siguen[0]
			continue
		}
		encuentro := d.encuentro(n.Siguen)
		rama := d.bifurcacion(n.Siguen, paradas, encuentro, ancho)
		total = apilar(total, false, rama, ancho)
		if encuentro == "" || rama.salida < 0 {
			return total
		}
		id = encuentro
	}
}

// referencia es lo que se dibuja al llegar a un nodo que ya está en el dibujo.
func (d *dibujante) referencia(id string, ancho int) bloque {
	texto := "→ sigue en " + d.porID[id].Nombre
	if d.camino[id] {
		texto = "↺ vuelve a " + d.porID[id].Nombre
	}
	return bloque{lineas: partir(texto, max(minAncho, ancho)), salida: -1}
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
	sigue := false
	for i, r := range ramas {
		b := d.secuencia(r, topes, interior)
		sigue = sigue || b.salida >= 0
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
		return bloque{lineas: lineas, salida: -1}
	}
	// El encuentro cuelga del mismo tronco: la última rama deja de ser la última.
	for i, l := range lineas {
		lineas[i] = reabrirTronco(l)
	}
	return bloque{lineas: lineas}
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
	if n.Tipo == TipoResultado {
		return circulo(textos)
	}
	return caja(textos)
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
		return bloque{lineas: append(lineas, b.lineas...), entrada: a.entrada, salida: b.salida}
	}
	lineas := mover(a.lineas, desplazaA)
	lineas = append(lineas, strings.Repeat(" ", a.salida+desplazaA)+"│")
	lineas = append(lineas, mover(b.lineas, desplazaB)...)
	salida := b.salida
	if salida >= 0 {
		salida += desplazaB
	}
	return bloque{lineas: lineas, entrada: a.entrada + desplazaA, salida: salida}
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
	columna, alto := 0, 0
	for i, r := range ramas {
		anchos[i] = anchoMayor(r.lineas)
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
		return bloque{lineas: lineas, entrada: eje, salida: -1}
	}
	return bloque{lineas: append(lineas, lineaDeUnion(salidas, eje, false)), entrada: eje, salida: eje}
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

// centrar deja el dibujo en el medio del ancho disponible.
func centrar(lineas []string, ancho int) []string {
	return mover(lineas, max(0, (ancho-anchoMayor(lineas))/2))
}
