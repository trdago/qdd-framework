package blueprints

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// Rule: Prohibido usar 'else'
// Calidad de un blueprint como receta (pedido del usuario, 2026-10-31): un blueprint es como el plano de un arma en el
// juego Rust, con él se construye la pieza entera en cualquier lenguaje sin adivinar. QDD lo mide de forma determinista
// en dos ejes: defectos de la receta (algo falta o está mal puesto para construirla tal cual) y mejoras que la acortan
// sin quitarle nada (cada regla sigue ahí, más corta o en su lugar). Y dice cuántos tokens cuesta leerla.

// Códigos de calidad. Los defectos bajan 10 puntos; las mejoras, 3.
const (
	CalEscenarioDeOpcion = "CAL-01" // defecto: un escenario de la base que solo vale con una opción
	CalVersionMinima     = "CAL-02" // defecto: la opción usa una versión mayor que la que pide
	CalOpcionVacia       = "CAL-03" // defecto: una opción que no dice qué implica
	CalCodigo            = "CAL-04" // defecto: código en la receta (no sería de cualquier lenguaje)
	CalSinEscenario      = "CAL-05" // defecto: una opción con invariantes y sin un escenario que las verifique
	CalLargo             = "CAL-06" // mejora: una regla de más de maxRegla caracteres
	CalRepetido          = "CAL-07" // mejora: una opción repite una regla de la base
	CalTerminoSinUso     = "CAL-08" // mejora: un término del vocabulario que nada usa

	GravedadDefecto = "defecto"
	GravedadMejora  = "mejora"

	maxRegla        = 280 // como un mensaje de chat: una regla se lee de un vistazo
	puntosDefecto   = 10
	puntosMejora    = 3
	maxDetalleTexto = 120
)

var gravedadDe = map[string]string{
	CalEscenarioDeOpcion: GravedadDefecto, CalVersionMinima: GravedadDefecto, CalOpcionVacia: GravedadDefecto,
	CalCodigo: GravedadDefecto, CalSinEscenario: GravedadDefecto,
	CalLargo: GravedadMejora, CalRepetido: GravedadMejora, CalTerminoSinUso: GravedadMejora,
}

// Calidad es la medición de un blueprint.
type Calidad struct {
	Ref       string            `json:"ref"`
	Nota      int               `json:"nota"`
	Defectos  int               `json:"defectos"`
	Mejoras   int               `json:"mejoras"`
	Tokens    TokensDeReceta    `json:"tokens"`
	Hallazgos []HallazgoCalidad `json:"hallazgos,omitempty"`
}

// TokensDeReceta estima (caracteres / 4) lo que cuesta leer el blueprint: su índice y la receta que recibe un
// proyecto con las respuestas por defecto y con las que más traen. Ahorro es lo que deja de leer la receta por
// defecto al mover cada escenario a su opción.
type TokensDeReceta struct {
	Indice     int `json:"indice"`
	PorDefecto int `json:"por_defecto"`
	Maxima     int `json:"maxima"`
	Ahorro     int `json:"ahorro,omitempty"`
}

// HallazgoCalidad es un problema concreto con su arreglo, que nunca quita contenido.
type HallazgoCalidad struct {
	Codigo   string `json:"codigo"`
	Gravedad string `json:"gravedad"`
	Donde    string `json:"donde"`
	Detalle  string `json:"detalle"`
	Arreglo  string `json:"arreglo"`
}

// Texto es el hallazgo en una línea estilo chat.
func (h HallazgoCalidad) Texto() string {
	simbolo := "✂"
	if h.Gravedad == GravedadDefecto {
		simbolo = "⚠"
	}
	return fmt.Sprintf("%s %s %s: %s → %s", simbolo, h.Codigo, h.Donde, recortar(h.Detalle, maxDetalleTexto), h.Arreglo)
}

// CalidadDe mide el blueprint.
func CalidadDe(b Blueprint) Calidad {
	c := Calidad{Ref: b.Ref()}
	medidas := []func(Blueprint) []HallazgoCalidad{escenariosDeOpcion, versionesMinimas, opcionesVacias, conCodigo, sinEscenario, largos, repetidos, terminosSinUso}
	for _, medir := range medidas {
		c.Hallazgos = append(c.Hallazgos, medir(b)...)
	}
	c.Nota = 100
	for i := range c.Hallazgos {
		c.Hallazgos[i].Gravedad = gravedadDe[c.Hallazgos[i].Codigo]
		c.contar(c.Hallazgos[i].Gravedad)
	}
	c.Nota = max(c.Nota, 0)
	c.Tokens = tokensDe(b)
	return c
}

func (c *Calidad) contar(gravedad string) {
	if gravedad == GravedadDefecto {
		c.Defectos++
		c.Nota -= puntosDefecto
		return
	}
	c.Mejoras++
	c.Nota -= puntosMejora
}

// LineaCalidad resume la medición en una línea: ref · nota · defectos y mejoras · tokens de la receta.
func LineaCalidad(c Calidad) string {
	ahorro := ""
	if c.Tokens.Ahorro > 0 {
		ahorro = fmt.Sprintf(" (−%s)", miles(c.Tokens.Ahorro))
	}
	l := fmt.Sprintf("%s · calidad %d/100 · ⚠%d ✂%d · receta ≈%s%s–%s tokens · índice ≈%s", c.Ref, c.Nota, c.Defectos, c.Mejoras,
		miles(c.Tokens.PorDefecto), ahorro, miles(c.Tokens.Maxima), miles(c.Tokens.Indice))
	return recortar(l, MaxLineaIndice)
}

func miles(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	return strconv.FormatFloat(float64(n)/1000, 'f', 1, 64) + "k"
}

// escenariosDeOpcion: un escenario de la base se aplica a todo proyecto. Si dice «pregunta=valor», solo vale con esa
// respuesta: a quien eligió otra le exige algo imposible y le cuesta tokens. Va en esa opción, igual que está.
func escenariosDeOpcion(b Blueprint) []HallazgoCalidad {
	var out []HallazgoCalidad
	for _, e := range b.Conformidad {
		p, valor, ok := respuestaCitada(b, e.Titulo+" "+e.Dado+" "+e.Cuando)
		if !ok {
			continue
		}
		out = append(out, HallazgoCalidad{Codigo: CalEscenarioDeOpcion, Donde: e.ID,
			Detalle: "solo vale con " + p + "=" + valor + ", pero se aplica a todo proyecto",
			Arreglo: "muévelo a la opción " + p + "=" + valor + " (mismo id y texto)"})
	}
	return out
}

func respuestaCitada(b Blueprint, texto string) (string, string, bool) {
	for _, p := range b.Preguntas {
		for _, o := range p.Opciones {
			if citaRespuesta(texto, p.ID, o.Valor) {
				return p.ID, o.Valor, true
			}
		}
	}
	return "", "", false
}

func citaRespuesta(texto, pregunta, valor string) bool {
	re := regexp.MustCompile(`(^|[^\w-])` + regexp.QuoteMeta(pregunta+"="+valor) + `($|[^\w-])`)
	return re.MatchString(texto)
}

var versionPedida = regexp.MustCompile(`([\w.-]+) ≥ (\d+(?:\.\d+){0,2})`)

// versionesMinimas: si una opción depende de «qdd ≥ 2.3.0» pero sus decisiones usan algo de «qdd ≥ 2.6.0», quien
// tenga 2.3.0 cree cumplir y no puede construirla.
func versionesMinimas(b Blueprint) []HallazgoCalidad {
	var out []HallazgoCalidad
	for _, p := range b.Preguntas {
		for _, o := range p.Opciones {
			out = append(out, versionMinima(p.ID+"="+o.Valor, o)...)
		}
	}
	return out
}

func versionMinima(donde string, o Opcion) []HallazgoCalidad {
	pedidas := map[string]string{}
	for _, m := range versionPedida.FindAllStringSubmatch(strings.Join(o.DependeDe, " "), -1) {
		pedidas[m[1]] = m[2]
	}
	usadas := map[string]string{}
	for _, m := range versionPedida.FindAllStringSubmatch(textoDeOpcion(o), -1) {
		if menorVersion(usadas[m[1]], m[2]) {
			usadas[m[1]] = m[2]
		}
	}
	var out []HallazgoCalidad
	for herramienta, pedida := range pedidas {
		usada := usadas[herramienta]
		if !menorVersion(pedida, usada) {
			continue
		}
		out = append(out, HallazgoCalidad{Codigo: CalVersionMinima, Donde: donde,
			Detalle: fmt.Sprintf("depende de %s ≥ %s, pero usa algo de %s ≥ %s", herramienta, pedida, herramienta, usada),
			Arreglo: fmt.Sprintf("en depende_de: %s ≥ %s", herramienta, usada)})
	}
	return out
}

func textoDeOpcion(o Opcion) string {
	partes := append([]string{o.Descripcion}, o.Decisiones...)
	partes = append(partes, o.Invariantes...)
	for _, e := range o.Conformidad {
		partes = append(partes, e.Titulo, e.Dado, e.Cuando, e.Entonces)
	}
	return strings.Join(partes, " ")
}

// menorVersion dice si a < b (versiones x.y.z; "" es la menor de todas).
func menorVersion(a, b string) bool {
	if b == "" {
		return false
	}
	pa, pb := partesVersion(a), partesVersion(b)
	for i := range pb {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return false
}

func partesVersion(v string) [3]int {
	var out [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		out[i], _ = strconv.Atoi(s)
	}
	return out
}

// opcionesVacias: una opción sin descripción ni nada que implique deja a quien construye adivinando qué es elegirla.
func opcionesVacias(b Blueprint) []HallazgoCalidad {
	var out []HallazgoCalidad
	for _, p := range b.Preguntas {
		for _, o := range p.Opciones {
			if strings.TrimSpace(textoDeOpcion(o)) != "" || len(o.DependeDe) > 0 {
				continue
			}
			out = append(out, HallazgoCalidad{Codigo: CalOpcionVacia, Donde: p.ID + "=" + o.Valor,
				Detalle: "no dice qué implica elegirla", Arreglo: "agrega su descripción"})
		}
	}
	return out
}

// conCodigo: la receta se replica en cualquier lenguaje; el código de ejemplo va en referencias/, no en la receta.
func conCodigo(b Blueprint) []HallazgoCalidad {
	var out []HallazgoCalidad
	for _, t := range textosDe(b) {
		if !strings.Contains(t.texto, "```") {
			continue
		}
		out = append(out, HallazgoCalidad{Codigo: CalCodigo, Donde: t.donde,
			Detalle: "trae un bloque de código: la receta es para cualquier lenguaje",
			Arreglo: "descríbelo en palabras (acción y resultado) y deja el código en referencias/"})
	}
	return out
}

// sinEscenario: una regla de una opción sin un escenario que la verifique no se puede probar al construirla.
func sinEscenario(b Blueprint) []HallazgoCalidad {
	var out []HallazgoCalidad
	for _, p := range b.Preguntas {
		for _, o := range p.Opciones {
			if len(o.Invariantes) == 0 || len(o.Conformidad) > 0 {
				continue
			}
			out = append(out, HallazgoCalidad{Codigo: CalSinEscenario, Donde: p.ID + "=" + o.Valor,
				Detalle: fmt.Sprintf("%d invariante(s) sin un escenario que las verifique: «%s»", len(o.Invariantes), o.Invariantes[0]),
				Arreglo: "agrega en esa opción el escenario (dado/cuando/entonces) que la prueba"})
		}
	}
	return out
}

// largos: una regla de más de maxRegla caracteres suele ser varias; dividida, cada una se verifica sola.
func largos(b Blueprint) []HallazgoCalidad {
	var out []HallazgoCalidad
	for _, t := range textosDe(b) {
		n := utf8.RuneCountInString(strings.Join(strings.Fields(t.texto), " "))
		if n <= maxRegla || t.prosa {
			continue
		}
		out = append(out, HallazgoCalidad{Codigo: CalLargo, Donde: t.donde,
			Detalle: fmt.Sprintf("%d caracteres: «%s»", n, recortar(primeraFrase(t.texto), 60)), Arreglo: arregloDeLargo(t.donde)})
	}
	return out
}

// arregloDeLargo: una regla larga se divide; una ayuda larga se acorta, porque lo que implica cada opción ya está en ella.
func arregloDeLargo(donde string) string {
	if strings.HasSuffix(donde, " ayuda") {
		return fmt.Sprintf("acórtala a ≤%d: lo que implica cada opción ya está en la opción", maxRegla)
	}
	return fmt.Sprintf("divídela en reglas de ≤%d sin quitar ninguna condición", maxRegla)
}

// repetidos: lo que la base ya dice se aplica siempre; repetirlo en una opción solo cuesta leerlo dos veces.
func repetidos(b Blueprint) []HallazgoCalidad {
	base := map[string]bool{}
	for _, x := range append(append([]string{}, b.Ficha.ObjetivoTecnologico.Decisiones...), b.Ficha.ObjetivoTecnologico.Invariantes...) {
		base[strings.TrimSpace(x)] = true
	}
	var out []HallazgoCalidad
	for _, p := range b.Preguntas {
		for _, o := range p.Opciones {
			for _, x := range append(append([]string{}, o.Decisiones...), o.Invariantes...) {
				if !base[strings.TrimSpace(x)] {
					continue
				}
				out = append(out, HallazgoCalidad{Codigo: CalRepetido, Donde: p.ID + "=" + o.Valor,
					Detalle: "repite una regla de la base: «" + recortar(x, 60) + "»", Arreglo: "quítala de la opción; la base ya la aplica"})
			}
		}
	}
	return out
}

// terminosSinUso: el vocabulario se copia a cada ficha; un término que nada usa solo cuesta tokens.
func terminosSinUso(b Blueprint) []HallazgoCalidad {
	var out []HallazgoCalidad
	for i, t := range b.Vocabulario {
		if seUsa(t.Termino, strings.ToLower(textoSinTermino(b, i))) {
			continue
		}
		out = append(out, HallazgoCalidad{Codigo: CalTerminoSinUso, Donde: t.Termino,
			Detalle: "ninguna pregunta, regla ni escenario lo usa", Arreglo: "quítalo, o úsalo donde corresponde"})
	}
	return out
}

// seUsa dice si el texto usa el término, aunque sea en plural o con otra terminación («acciones del panel» usa
// «acción del panel»): cada palabra de 4 letras o más debe aparecer por su raíz. Ante la duda, se usa.
func seUsa(termino, texto string) bool {
	for _, palabra := range strings.Fields(strings.ToLower(termino)) {
		r := []rune(palabra)
		if len(r) < 4 {
			continue
		}
		if !strings.Contains(texto, string(r[:max(4, len(r)-2)])) {
			return false
		}
	}
	return true
}

func textoSinTermino(b Blueprint, i int) string {
	var partes []string
	for j, t := range textosDe(b) {
		if j == i {
			continue
		}
		partes = append(partes, t.texto)
	}
	return strings.Join(partes, " ")
}

type textoUbicado struct {
	donde string
	texto string
	prosa bool // descripción y objetivos: prosa que explica, no una regla
}

// textosDe devuelve cada texto del blueprint con dónde está. Los del vocabulario van primero, en su orden.
func textosDe(b Blueprint) []textoUbicado {
	var out []textoUbicado
	for _, t := range b.Vocabulario {
		out = append(out, textoUbicado{donde: "vocabulario " + t.Termino, texto: t.Definicion})
	}
	f := b.Ficha
	out = append(out, textoUbicado{"descripcion", b.Descripcion, true}, textoUbicado{"problema", f.ObjetivoOperativo.Problema, true},
		textoUbicado{"criterio_de_exito", f.ObjetivoOperativo.CriterioDeExito, true}, textoUbicado{"arquitectura", f.ObjetivoTecnologico.Arquitectura, true})
	out = append(out, reglas("decisiones", f.ObjetivoTecnologico.Decisiones)...)
	out = append(out, reglas("invariantes", f.ObjetivoTecnologico.Invariantes)...)
	out = append(out, escenariosUbicados("", b.Conformidad)...)
	for _, p := range b.Preguntas {
		out = append(out, textoUbicado{donde: p.ID + " ayuda", texto: p.Pregunta + " " + p.Ayuda})
		for _, o := range p.Opciones {
			donde := p.ID + "=" + o.Valor
			out = append(out, textoUbicado{donde: donde, texto: o.Descripcion})
			out = append(out, reglas(donde+" decisiones", o.Decisiones)...)
			out = append(out, reglas(donde+" invariantes", o.Invariantes)...)
			out = append(out, escenariosUbicados(donde+" ", o.Conformidad)...)
		}
	}
	return out
}

func reglas(donde string, items []string) []textoUbicado {
	out := make([]textoUbicado, 0, len(items))
	for _, x := range items {
		out = append(out, textoUbicado{donde: donde, texto: x})
	}
	return out
}

func escenariosUbicados(prefijo string, escenarios []Escenario) []textoUbicado {
	out := make([]textoUbicado, 0, 4*len(escenarios))
	for _, e := range escenarios {
		for _, campo := range []string{e.Titulo, e.Dado, e.Cuando, e.Entonces} {
			out = append(out, textoUbicado{donde: prefijo + e.ID, texto: campo})
		}
	}
	return out
}

// tokensDe mide el índice y la receta (lo que 'apply' deja en la ficha y en sus escenarios) con las respuestas por
// defecto y con las que más traen.
func tokensDe(b Blueprint) TokensDeReceta {
	indice, _ := json.Marshal(IndiceDe(b))
	t := TokensDeReceta{Indice: tokens(string(indice)), PorDefecto: tokensDeReceta(b, respuestasPorDefecto(b)), Maxima: tokensDeReceta(b, respuestasMayores(b))}
	t.Maxima = max(t.Maxima, t.PorDefecto)
	defecto := respuestasPorDefecto(b)
	for _, e := range b.Conformidad {
		p, valor, ok := respuestaCitada(b, e.Titulo+" "+e.Dado+" "+e.Cuando)
		if !ok || defecto[p] == valor {
			continue
		}
		t.Ahorro += tokens(e.Titulo + e.Dado + e.Cuando + e.Entonces)
	}
	return t
}

func tokens(s string) int {
	return (utf8.RuneCountInString(s) + 3) / 4
}

func tokensDeReceta(b Blueprint, respuestas map[string]string) int {
	aplicadas, _, _ := b.Resolver(respuestas)
	gen, escenarios := b.componer(aplicadas)
	data, _ := yaml.Marshal(struct {
		Generado   Generado    `yaml:"generado"`
		Escenarios []Escenario `yaml:"escenarios"`
	}{gen, escenarios})
	return tokens(string(data))
}

func respuestasPorDefecto(b Blueprint) map[string]string {
	out := map[string]string{}
	for _, p := range b.Preguntas {
		out[p.ID] = primero(p.PorDefecto, primeraOpcion(p))
	}
	return out
}

func primeraOpcion(p Pregunta) string {
	if len(p.Opciones) == 0 {
		return ""
	}
	return p.Opciones[0].Valor
}

// respuestasMayores elige en cada pregunta la opción que más trae: la receta más grande que puede recibir un proyecto.
func respuestasMayores(b Blueprint) map[string]string {
	out := map[string]string{}
	for _, p := range b.Preguntas {
		mayor := -1
		for _, o := range p.Opciones {
			n := tokens(textoDeOpcion(o))
			if n <= mayor {
				continue
			}
			mayor, out[p.ID] = n, o.Valor
		}
	}
	return out
}

// Medir mide los blueprints pedidos: una carpeta con blueprint.yaml (p. ej. un aporte en curso), un id importado o,
// si no está importado, el publicado en la fuente de la organización. Sin pedidos, los importados; y si el proyecto
// no importó ninguno, todos los publicados en la fuente. Devuelve también de dónde salieron.
func Medir(root string, pedidos []string) ([]Calidad, string, error) {
	if len(pedidos) == 0 {
		return medirTodos(root)
	}
	var out []Calidad
	var faltan []string
	for _, p := range pedidos {
		b, err := blueprintPedido(root, p)
		if err != nil {
			faltan = append(faltan, p)
			continue
		}
		out = append(out, CalidadDe(b))
	}
	if len(faltan) == 0 {
		return out, "importados", nil
	}
	publicados, fuente, err := medirPublicados(root, faltan)
	return append(out, publicados...), fuente, err
}

func blueprintPedido(root, pedido string) (Blueprint, error) {
	if exists(filepath.Join(pedido, ManifestName)) {
		return ParseDir(pedido)
	}
	return Find(root, pedido)
}

func medirTodos(root string) ([]Calidad, string, error) {
	bps, errs := List(root)
	if len(errs) > 0 {
		return nil, "", errs[0]
	}
	if len(bps) == 0 {
		return medirPublicados(root, nil)
	}
	out := make([]Calidad, 0, len(bps))
	for _, b := range bps {
		out = append(out, CalidadDe(b))
	}
	return out, "importados", nil
}

// medirPublicados mide la última versión publicada en la fuente de la organización (todos, o los ids pedidos).
func medirPublicados(root string, ids []string) ([]Calidad, string, error) {
	reg, err := LoadRegistro(root)
	if err != nil {
		return nil, "", err
	}
	if reg.Origen == "" {
		return nil, "", fmt.Errorf("el proyecto no tiene blueprints importados ni un repositorio de la organización: impórtalos con 'qdd blueprint add'")
	}
	base, _, entry, cleanup, err := fetch(reg.Origen, "")
	defer cleanup()
	if err != nil {
		return nil, "", err
	}
	_, disponibles, err := catalogo(base)
	if err != nil {
		return nil, "", err
	}
	fuente := reg.Origen + " " + entry.Ref
	var out []Calidad
	for _, id := range sortedKeys(disponibles) {
		if len(ids) > 0 && !inList(ids, id) {
			continue
		}
		b, err := ParseDir(filepath.Join(base, filepath.FromSlash(disponibles[id])))
		if err != nil {
			return nil, fuente, err
		}
		out = append(out, CalidadDe(b))
	}
	for _, id := range ids {
		if _, ok := disponibles[id]; !ok {
			return out, fuente, fmt.Errorf("'%s' no está importado ni publicado en %s", id, fuente)
		}
	}
	return out, fuente, nil
}

// DefectosNuevos son los defectos de despues que antes no tenía: una versión nueva de un blueprint nunca empeora la
// receta (qdd blueprint propose la rechaza).
func DefectosNuevos(antes, despues Calidad) []HallazgoCalidad {
	previos := map[string]bool{}
	for _, h := range antes.Hallazgos {
		previos[h.Codigo+" "+h.Donde] = true
	}
	var out []HallazgoCalidad
	for _, h := range despues.Hallazgos {
		if h.Gravedad != GravedadDefecto || previos[h.Codigo+" "+h.Donde] {
			continue
		}
		out = append(out, h)
	}
	return out
}

func textosDeHallazgos(hallazgos []HallazgoCalidad) string {
	partes := make([]string, 0, len(hallazgos))
	for _, h := range hallazgos {
		partes = append(partes, h.Texto())
	}
	return strings.Join(partes, "; ")
}
