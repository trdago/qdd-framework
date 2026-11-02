// Package materializar calcula, cuadro a cuadro, cómo aparece (o se va) un texto del panel: el efecto de las ventanas
// del usuario (hypr-materialize) llevado a celdas de terminal. Un frente avanza de arriba abajo y de izquierda a
// derecha con un borde irregular; detrás del frente cada celda pasa por caracteres que parpadean en el acento, bloques
// que se afinan (█ ▓ ▒ ░) y el texto nítido, que pierde el tinte del acento. Es una función pura: la misma entrada da
// los mismos cuadros, y ningún cuadro cambia el ancho del texto (nada se mueve de lugar).
package materializar

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Las mismas constantes que el shader de hypr-materialize (materialize_open.glsl).
const (
	span   = 0.25 // vida de una celda dentro del progreso total (alto del frente)
	jitter = 0.1  // irregularidad del borde del frente
	finUno = 0.35 // hasta aquí, caracteres que parpadean
	finDos = 0.8  // hasta aquí, bloques que se afinan; después, el texto que pierde el tinte

	// CuadroMs es cada cuánto cambia el cuadro: 25 por segundo, lo que un terminal dibuja sin tirones.
	CuadroMs = 40
	// AbrirMs y CerrarMs son las duraciones de hypr-materialize (0,8 s al abrir y 0,4 s al cerrar).
	AbrirMs  = 800
	CerrarMs = 400
)

// Glifos son los caracteres de la primera etapa: todos de una columna, para que el ancho no cambie.
const Glifos = "01#$%&*+=<>/\\|?!:;@ｱｲｳｴｵｶｷｸｹｺｻｼｽ"

var glifos = []rune(Glifos)

// Segmento es un trozo de texto con un solo estilo. Color es un hex (#rrggbb) o un color del tema (yellow, cyan…).
type Segmento struct {
	Texto   string `json:"texto"`
	Color   string `json:"color,omitempty"`
	Negrita bool   `json:"negrita,omitempty"`
	Tenue   bool   `json:"tenue,omitempty"`
}

// Linea es una línea del panel: sus segmentos en orden.
type Linea []Segmento

// Opciones dicen hacia dónde va el efecto, cuánto dura y con qué acento.
type Opciones struct {
	Abrir      bool   `json:"abrir"` // true: aparece; false: se va
	DuracionMs int    `json:"duracion_ms,omitempty"`
	CuadroMs   int    `json:"cuadro_ms,omitempty"`
	Acento     string `json:"acento,omitempty"`
	Semilla    uint32 `json:"semilla,omitempty"`
}

// celda es lo que ocupa una posición del texto: un carácter con lo que lo acompaña (un selector de variante) y su
// ancho en columnas.
type celda struct {
	texto  string
	ancho  int
	estilo Segmento
}

// Cuadros devuelve cada cuadro del efecto. Al abrir, el último es el texto tal cual; al cerrar, queda en blanco.
func Cuadros(lineas []Linea, o Opciones) [][]Linea {
	n := cantidad(o)
	celdas := partir(lineas)
	cuadros := make([][]Linea, 0, n)
	for k := 0; k < n; k++ {
		cuadros = append(cuadros, cuadro(celdas, progreso(k, n, o.Abrir), k, o))
	}
	return cuadros
}

func cantidad(o Opciones) int {
	ms := o.CuadroMs
	if ms <= 0 {
		ms = CuadroMs
	}
	d := o.DuracionMs
	if d <= 0 {
		d = AbrirMs
	}
	return max(1, (d+ms-1)/ms)
}

func progreso(k, n int, abrir bool) float64 {
	p := float64(k+1) / float64(n)
	if abrir {
		return p
	}
	return 1 - p
}

func cuadro(celdas [][]celda, p float64, k int, o Opciones) []Linea {
	out := make([]Linea, 0, len(celdas))
	for r, linea := range celdas {
		ancho := 0
		for _, c := range linea {
			ancho += c.ancho
		}
		col := 0
		var l Linea
		for i, c := range linea {
			pos := (float64(r) + (float64(col)+0.5)/float64(max(1, ancho))) / float64(len(celdas))
			umbral := (1 - span) * clamp(mezcla(pos, azar(o.Semilla, r, i, 0), jitter))
			l = append(l, pintar(c, clamp((p-umbral)/span), k, r, i, o))
			col += c.ancho
		}
		out = append(out, l)
	}
	return juntar(out)
}

func pintar(c celda, t float64, k, r, i int, o Opciones) Segmento {
	if t >= 1 {
		return conTexto(c.estilo, c.texto)
	}
	if t <= 0 {
		return Segmento{Texto: strings.Repeat(" ", c.ancho)}
	}
	if t < finUno {
		return glifo(c, t/finUno, k, r, i, o)
	}
	if t < finDos {
		return bloque(c, (t-finUno)/(finDos-finUno), o)
	}
	return nitido(c, suave((t-finDos)/(1-finDos)), o)
}

// glifo: caracteres al azar que cambian en cada cuadro, del acento brillante hacia el color de la celda.
func glifo(c celda, a float64, k, r, i int, o Opciones) Segmento {
	if azar(o.Semilla, r*7919+i, k, 1) <= 0.5 {
		return Segmento{Texto: strings.Repeat(" ", c.ancho)}
	}
	var b strings.Builder
	for j := 0; j < c.ancho; j++ {
		b.WriteRune(glifos[int(azar(o.Semilla, r, i*2+j, k+11)*float64(len(glifos)))%len(glifos)])
	}
	return Segmento{Texto: b.String(), Color: tenido(brillante(o.Acento), c.estilo.Color, a*0.6, o.Acento), Tenue: a < 0.4}
}

// bloque: el carácter se ve como un bloque cada vez más fino; un espacio sigue en blanco (no tiene nada que mostrar).
func bloque(c celda, s float64, o Opciones) Segmento {
	if strings.TrimSpace(c.texto) == "" {
		return Segmento{Texto: c.texto}
	}
	forma := []string{"█", "▓", "▒", "░"}[min(3, int(s*4))]
	return Segmento{Texto: strings.Repeat(forma, c.ancho), Color: tenido(c.estilo.Color, o.Acento, 0.4*(1-s)+0.3, o.Acento)}
}

// nitido: el texto real, que pierde lo que le queda del acento.
func nitido(c celda, f float64, o Opciones) Segmento {
	s := conTexto(c.estilo, c.texto)
	s.Color = tenido(c.estilo.Color, brillante(o.Acento), 0.3*(1-f), c.estilo.Color)
	return s
}

// tenido mezcla dos colores hex (f es cuánto de b); si alguno no es hex (un color del tema), no se inventa uno: vale
// sinMezcla.
func tenido(a, b string, f float64, sinMezcla string) string {
	ra, okA := rgb(a)
	rb, okB := rgb(b)
	if !okA || !okB {
		return sinMezcla
	}
	var m [3]float64
	for i := range m {
		m[i] = ra[i] + (rb[i]-ra[i])*clamp(f)
	}
	return hex(m)
}

// brillante es el acento por 1,5, como los caracteres del shader.
func brillante(color string) string {
	c, ok := rgb(color)
	if !ok {
		return color
	}
	return hex([3]float64{math.Min(1, c[0]*1.5), math.Min(1, c[1]*1.5), math.Min(1, c[2]*1.5)})
}

func rgb(color string) ([3]float64, bool) {
	s := strings.TrimPrefix(color, "#")
	if len(s) != 6 || !strings.HasPrefix(color, "#") {
		return [3]float64{}, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return [3]float64{}, false
	}
	return [3]float64{float64(v>>16&0xff) / 255, float64(v>>8&0xff) / 255, float64(v&0xff) / 255}, true
}

func hex(c [3]float64) string {
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round(c[0]*255)), int(math.Round(c[1]*255)), int(math.Round(c[2]*255)))
}

func conTexto(s Segmento, texto string) Segmento {
	s.Texto = texto
	return s
}

// partir separa cada línea en celdas; un carácter de ancho cero (un selector de variante, una tilde que combina) va
// con el anterior.
func partir(lineas []Linea) [][]celda {
	out := make([][]celda, 0, len(lineas))
	for _, l := range lineas {
		var cs []celda
		for _, s := range l {
			estilo := conTexto(s, "")
			for _, r := range s.Texto {
				w := anchoRuna(r)
				if w == 0 && len(cs) > 0 {
					ultima := &cs[len(cs)-1]
					ultima.texto += string(r)
					ultima.ancho = max(ultima.ancho, anchoCon(r, ultima.ancho))
					continue
				}
				cs = append(cs, celda{texto: string(r), ancho: max(w, 0), estilo: estilo})
			}
		}
		out = append(out, cs)
	}
	return out
}

// juntar une los segmentos vecinos con el mismo estilo y quita los vacíos.
func juntar(lineas []Linea) []Linea {
	out := make([]Linea, 0, len(lineas))
	for _, l := range lineas {
		var j Linea
		for _, s := range l {
			if s.Texto == "" {
				continue
			}
			if n := len(j); n > 0 && mismoEstilo(j[n-1], s) {
				j[n-1].Texto += s.Texto
				continue
			}
			j = append(j, s)
		}
		out = append(out, j)
	}
	return out
}

func mismoEstilo(a, b Segmento) bool {
	return a.Color == b.Color && a.Negrita == b.Negrita && a.Tenue == b.Tenue
}

// azar es un número en [0, 1) que solo depende de sus argumentos: el mismo en cada corrida.
func azar(semilla uint32, a, b, c int) float64 {
	h := semilla*0x9E3779B1 ^ uint32(a)*0x85EBCA77 ^ uint32(b)*0xC2B2AE3D ^ uint32(c)*0x27D4EB2F
	h ^= h >> 15
	h *= 0x2C1B3C6D
	h ^= h >> 12
	h *= 0x297A2D39
	h ^= h >> 15
	return float64(h) / 4294967296.0
}

func mezcla(a, b, f float64) float64 { return a + (b-a)*f }

func clamp(x float64) float64 { return math.Max(0, math.Min(1, x)) }

func suave(x float64) float64 {
	x = clamp(x)
	return x * x * (3 - 2*x)
}
