package estado

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Rule: Prohibido usar 'else'
// Vistas: tabla markdown (respuesta), una línea para la barra de estado y el panel de 'watch'.

func celdaTexto(c Celda) string {
	estado := NormalizarEstado(c.Estado)
	switch estado {
	case Hecho:
		return strings.TrimSpace(Hecho + " " + c.Evidencia)
	case Espera:
		return fmt.Sprintf("%s %s (%s)", Espera, c.Bloqueo, c.DeQuien)
	case Falla:
		return strings.TrimSpace(Falla + " " + c.Falla)
	case NoAplica, SinEmpezar:
		return estado
	}
	return c.Estado
}

func pendienteDeAprobacion(c Celda) string {
	if c.RequiereAprobacion == "" || c.Aprobado {
		return ""
	}
	return " · requiere aprobación de " + c.RequiereAprobacion
}

func escaparMD(s string) string { return strings.ReplaceAll(s, "|", "\\|") }

func etiquetaFrente(f Frente) string {
	if f.Estado == EstadoAbierto {
		return f.Nombre
	}
	return fmt.Sprintf("%s (%s: %s)", f.Nombre, f.Estado, f.MotivoCierre)
}

// Markdown arma la tabla de la respuesta con el alcance pedido.
func Markdown(e Estado, now time.Time, alcance Alcance) string {
	var b strings.Builder
	b.WriteString("| Tarea | " + strings.Join(e.Etapas, " | ") + " |\n")
	b.WriteString("|---" + strings.Repeat("|---", len(e.Etapas)) + "|\n")
	for _, f := range e.Frentes {
		if !f.EnAlcance(alcance, now) {
			continue
		}
		celdas := make([]string, 0, len(e.Etapas))
		for _, etapa := range e.Etapas {
			c := f.Celdas[etapa]
			celdas = append(celdas, escaparMD(celdaTexto(c)+pendienteDeAprobacion(c)))
		}
		b.WriteString("| " + escaparMD(etiquetaFrente(f)) + " | " + strings.Join(celdas, " | ") + " |\n")
	}
	return b.String()
}

// Mensajeria arma la tabla como texto para aplicaciones de mensajería (sin tablas markdown).
func Mensajeria(e Estado, now time.Time, alcance Alcance) string {
	var b strings.Builder
	for _, f := range e.Frentes {
		if !f.EnAlcance(alcance, now) {
			continue
		}
		b.WriteString("*" + etiquetaFrente(f) + "*\n")
		for _, etapa := range e.Etapas {
			c := f.Celdas[etapa]
			b.WriteString("  " + etapa + ": " + celdaTexto(c) + pendienteDeAprobacion(c) + "\n")
		}
	}
	return b.String()
}

// CheckRespuesta (EPF-A01): la tabla de una respuesta tiene los mismos frentes y estados que el archivo.
func CheckRespuesta(e Estado, respuesta string, now time.Time) []Violacion {
	filas, ok := leerTablaMD(respuesta, len(e.Etapas))
	if !ok {
		return []Violacion{{ID: "EPF-A01", Mensaje: "la respuesta no trae la tabla de estado por tarea"}}
	}
	var vs []Violacion
	esperados := map[string]bool{}
	for _, f := range e.Frentes {
		if !f.EnAlcance(AlcanceAbiertos, now) {
			continue
		}
		etiqueta := etiquetaFrente(f)
		esperados[etiqueta] = true
		celdas, hay := filas[etiqueta]
		if !hay {
			vs = append(vs, Violacion{ID: "EPF-A01", Frente: f.ID, Mensaje: "está en el archivo y no en la respuesta"})
			continue
		}
		for i, etapa := range e.Etapas {
			want := NormalizarEstado(f.Celdas[etapa].Estado)
			if primerEstado(celdas[i]) != want {
				vs = append(vs, Violacion{ID: "EPF-A01", Frente: f.ID, Etapa: etapa, Mensaje: fmt.Sprintf("la respuesta dice %q y el archivo %s", celdas[i], want)})
			}
		}
	}
	for etiqueta := range filas {
		if !esperados[etiqueta] {
			vs = append(vs, Violacion{ID: "EPF-A01", Frente: etiqueta, Mensaje: "está en la respuesta y no en el archivo"})
		}
	}
	return vs
}

func primerEstado(celda string) string {
	campos := strings.Fields(celda)
	if len(campos) == 0 {
		return ""
	}
	return NormalizarEstado(campos[0])
}

// leerTablaMD toma la primera tabla cuyo encabezado empieza por «Tarea» (o «Frente», la tabla de antes) y devuelve sus filas por etiqueta.
func leerTablaMD(texto string, etapas int) (map[string][]string, bool) {
	filas := map[string][]string{}
	enTabla := false
	for _, linea := range strings.Split(texto, "\n") {
		l := strings.TrimSpace(linea)
		if !strings.HasPrefix(l, "|") {
			if enTabla {
				break
			}
			continue
		}
		celdas := partirFila(l)
		if !enTabla {
			enTabla = len(celdas) > 0 && (strings.EqualFold(celdas[0], "Tarea") || strings.EqualFold(celdas[0], "Frente"))
			continue
		}
		if len(celdas) != etapas+1 || strings.HasPrefix(celdas[0], "---") {
			continue
		}
		filas[celdas[0]] = celdas[1:]
	}
	return filas, enTabla
}

func partirFila(l string) []string {
	l = strings.TrimSuffix(strings.TrimPrefix(l, "|"), "|")
	l = strings.ReplaceAll(l, "\\|", "\x00")
	partes := strings.Split(l, "|")
	for i, p := range partes {
		partes[i] = strings.TrimSpace(strings.ReplaceAll(p, "\x00", "|"))
	}
	return partes
}

// Statusline es la línea corta para la barra: tareas pendientes, cuántos ⏳ y cuántos ❌ tienen (los mismos totales
// que el panel, FND-086). "" si no hay tareas abiertas.
func Statusline(e Estado, now time.Time) string {
	if len(e.Abiertos()) == 0 {
		return ""
	}
	t := e.Totales()
	if t.Tareas == 0 {
		return "◧ sin pendientes · " + Hecho
	}
	palabra := "tareas"
	if t.Tareas == 1 {
		palabra = "tarea"
	}
	if t.Esperan == 0 && t.Fallan == 0 {
		return fmt.Sprintf("◧ %d %s · %s", t.Tareas, palabra, Hecho)
	}
	return fmt.Sprintf("◧ %d %s · %s%d %s%d", t.Tareas, palabra, Espera, t.Esperan, Falla, t.Fallan)
}

// BuscarRaiz sube desde dir hasta encontrar el archivo de estado; "" si no hay.
func BuscarRaiz(dir string) string {
	d, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(d, Ruta)); err == nil {
			return d
		}
		padre := filepath.Dir(d)
		if padre == d {
			return ""
		}
		d = padre
	}
}

// StatuslineDir es Statusline para la barra de la consola: nunca falla; sin archivo o con uno dañado, "".
func StatuslineDir(dir string, now time.Time) string {
	raiz := BuscarRaiz(dir)
	if raiz == "" {
		return ""
	}
	e, err := Load(raiz)
	if err != nil {
		return ""
	}
	return Statusline(e, now)
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

// AnchoVisible cuenta columnas de terminal: sin secuencias ANSI; ✅ ⏳ ❌ ocupan 2 y — · una.
func AnchoVisible(s string) int {
	n := 0
	for _, r := range ansi.ReplaceAllString(s, "") {
		n += anchoRuna(r)
	}
	return n
}

func anchoRuna(r rune) int {
	switch {
	case r == '️':
		return 0
	case r == '✅' || r == '⏳' || r == '❌':
		return 2
	case r >= 0x1F300 && r <= 0x1FAFF:
		return 2
	}
	return 1
}

// Truncar corta s a w columnas visibles, con «…» si sobra.
func Truncar(s string, w int) string {
	if AnchoVisible(s) <= w {
		return s
	}
	if w <= 0 {
		return ""
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		a := anchoRuna(r)
		if n+a > w-1 {
			break
		}
		b.WriteRune(r)
		n += a
	}
	return b.String() + "…"
}

func abreviar(etapa string) string {
	if utf8.RuneCountInString(etapa) <= 2 {
		return etapa
	}
	return string([]rune(etapa)[:2])
}

// abreviaturas da a cada etapa 2 letras distintas: las 2 primeras si no chocan; si chocan (Pruebas y Producción), la
// primera más la siguiente letra que las distinga (Pu, Po), y como último recurso la primera más un dígito. Chocar es
// leerse igual: «Pr» y «PR» chocan aunque cambie la mayúscula (FND-087), y la letra nueva no repite otra columna.
func abreviaturas(etapas []string) []string {
	out := make([]string, len(etapas))
	for i, e := range etapas {
		out[i] = abreviar(e)
	}
	for j := 2; j < 16 && repetidas(out) != nil; j++ {
		rep := repetidas(out)
		for i, e := range etapas {
			letras := []rune(strings.ReplaceAll(e, " ", ""))
			if !rep[clave(out[i])] || len(letras) <= j {
				continue
			}
			candidata := string(letras[0]) + string(letras[j])
			if usada(out, i, candidata) {
				continue
			}
			out[i] = candidata
		}
	}
	rep := repetidas(out)
	for i, e := range etapas {
		if rep[clave(out[i])] && e != "" {
			out[i] = string([]rune(e)[0]) + strconv.Itoa(i%10)
		}
	}
	return out
}

// clave es cómo se lee una abreviatura: sin distinguir mayúsculas.
func clave(a string) string { return strings.ToLower(a) }

// usada dice si otra columna ya se lee como candidata.
func usada(abrev []string, propia int, candidata string) bool {
	for i, a := range abrev {
		if i != propia && clave(a) == clave(candidata) {
			return true
		}
	}
	return false
}

// repetidas devuelve, por su clave, las abreviaturas que se leen igual más de una vez, o nil si no hay.
func repetidas(abrev []string) map[string]bool {
	cuenta := map[string]int{}
	for _, a := range abrev {
		cuenta[clave(a)]++
	}
	var rep map[string]bool
	for a, n := range cuenta {
		if n < 2 {
			continue
		}
		if rep == nil {
			rep = map[string]bool{}
		}
		rep[a] = true
	}
	return rep
}

// Panel arma las líneas del panel de 'watch' para un ancho dado: tabla compacta (un estado por celda), lo que espera
// cada ⏳ y de quién, y las fallas. Función pura: se prueba sin terminal.
func Panel(e Estado, proyecto string, ancho int, now time.Time) []string {
	if ancho < 20 {
		ancho = 20
	}
	var out []string
	add := func(s string) { out = append(out, Truncar(s, ancho)) }
	titulo := "Estado por tarea · " + proyecto
	if t, err := time.Parse(time.RFC3339, e.Actualizado); err == nil {
		titulo += " · " + t.In(now.Location()).Format("15:04")
	}
	add(titulo)
	add(strings.Repeat("─", ancho))

	abrev := abreviaturas(e.Etapas)
	nombreAncho := AnchoVisible("Tarea")
	for _, f := range e.Frentes {
		if f.EnAlcance(AlcanceAbiertos, now) && AnchoVisible(f.Nombre) > nombreAncho {
			nombreAncho = AnchoVisible(f.Nombre)
		}
	}
	if disponible := ancho - 3*len(e.Etapas) - 1; nombreAncho > disponible {
		nombreAncho = disponible
	}
	if nombreAncho < 8 {
		nombreAncho = 8
	}
	cab := relleno("Tarea", nombreAncho)
	for _, a := range abrev {
		cab += " " + relleno(a, 2)
	}
	add(cab)

	var esperas, fallas []string
	for _, f := range e.Frentes {
		if !f.EnAlcance(AlcanceAbiertos, now) {
			continue
		}
		fila := relleno(Truncar(f.Nombre, nombreAncho), nombreAncho)
		for _, etapa := range e.Etapas {
			c := f.Celdas[etapa]
			fila += " " + relleno(simbolo(c), 2)
			switch NormalizarEstado(c.Estado) {
			case Espera:
				esperas = append(esperas, "• "+f.Nombre+" › "+etapa, "  de "+c.DeQuien+": "+c.Bloqueo)
			case Falla:
				fallas = append(fallas, "• "+f.Nombre+" › "+etapa, "  "+c.Falla)
			}
		}
		add(fila)
	}
	add("")
	add(leyenda(e.Etapas, abrev))
	if len(esperas) > 0 {
		add("")
		add(Espera + " Esperando")
		for _, l := range esperas {
			add(l)
		}
	}
	if len(fallas) > 0 {
		add("")
		add(Falla + " Fallas")
		for _, l := range fallas {
			add(l)
		}
	}
	return out
}

func simbolo(c Celda) string {
	s := NormalizarEstado(c.Estado)
	if s == "" {
		return "?"
	}
	if s == Hecho && pendienteDeAprobacion(c) != "" {
		return "?"
	}
	return s
}

func leyenda(etapas, abrev []string) string {
	partes := make([]string, 0, len(etapas))
	for i, e := range etapas {
		partes = append(partes, abrev[i]+"="+e)
	}
	return strings.Join(partes, " ")
}

func relleno(s string, w int) string {
	falta := w - AnchoVisible(s)
	if falta <= 0 {
		return s
	}
	return s + strings.Repeat(" ", falta)
}
