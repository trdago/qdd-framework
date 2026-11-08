package reportes

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/qdd-framework/qdd/pkg/incidentes"
)

// Rule: Prohibido usar 'else'
// RFC 0024: un reporte a QDD lo escribe una persona y lo envía con su «sí». La limpieza es más suave que la de
// incidentes: quedan sus comillas, nombres de archivo y correos (puede dejar un contacto); salen credenciales, valores
// del baúl, la ruta del proyecto, la carpeta del usuario y su nombre. Lo que viaja es exactamente Vista.

const (
	MaxTitulo  = 120
	MaxDetalle = 6000
	// CodigoTodaviaNo es el «todavía no» de los pedidos: el reporte espera el «sí» de la persona en el panel.
	CodigoTodaviaNo = 75
)

// Tipos son los que acepta el relay.
var Tipos = []string{"bug", "mejora", "pregunta"}

// Datos es lo que escribe la persona, más la versión de qdd y el sistema.
type Datos struct {
	Titulo  string
	Detalle string
	Tipo    string
	Version string
	SO      string
}

// Reporte es el cuerpo exacto que recibe el relay; Instalacion, En y Firma se ponen al enviar.
type Reporte struct {
	Formato     int    `json:"formato"`
	Instalacion string `json:"instalacion"`
	En          string `json:"en"`
	Tipo        string `json:"tipo"`
	Titulo      string `json:"titulo"`
	Detalle     string `json:"detalle"`
	Version     string `json:"version"`
	SO          string `json:"so"`
	Firma       string `json:"firma"`
}

var (
	reClaveValor = regexp.MustCompile(`(?i)\b(token|clave|password|passwd|secret|api[_-]?key)(\s*[:=]\s*)\S+`)
	reBearer     = regexp.MustCompile(`(?i)\b(bearer)\s+\S+`)
	rePrefijo    = regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr|github_pat|sk|npm|xox[abpr]|AKIA|sb_secret|sb_publishable)[A-Za-z0-9_\-]{8,}`)
	reLargo      = regexp.MustCompile(`\b[A-Za-z0-9+_\-]{32,}\b`)
	reCorreo     = regexp.MustCompile(`[\w.+\-]+@[\w\-]+(\.[\w\-]+)+`)
)

// Limpiar quita del texto lo que no debe salir de la máquina y deja lo demás como lo escribió la persona.
func Limpiar(s string, c incidentes.Contexto, valores []string) string {
	for _, v := range valores {
		if strings.TrimSpace(v) == "" {
			continue
		}
		s = strings.ReplaceAll(s, v, baul.Tapado)
	}
	// Las rutas van antes que las cadenas largas: un tramo largo de la ruta del proyecto (FND-186) se cortaba primero y
	// el resto de la ruta viajaba.
	s = reemplazarRuta(s, c.Raiz, "<proyecto>")
	s = reemplazarRuta(s, c.Home, "~")
	s = reClaveValor.ReplaceAllString(s, "$1$2<secreto>")
	s = reBearer.ReplaceAllString(s, "$1 <secreto>")
	s = rePrefijo.ReplaceAllString(s, "<secreto>")
	s = reLargo.ReplaceAllString(s, "<secreto>")
	return fueraDeCorreos(s, func(tramo string) string { return palabra(tramo, c.Usuario, "<usuario>") })
}

// reemplazarRuta cambia una carpeta (no la raíz «/») por su marcador.
func reemplazarRuta(s, ruta, marcador string) string {
	ruta = strings.TrimRight(ruta, "/")
	if ruta == "" {
		return s
	}
	return strings.ReplaceAll(s, ruta, marcador)
}

// fueraDeCorreos aplica f solo a lo que no es un correo: el contacto que la persona deja se envía tal cual.
func fueraDeCorreos(s string, f func(string) string) string {
	var b strings.Builder
	desde := 0
	for _, m := range reCorreo.FindAllStringIndex(s, -1) {
		b.WriteString(f(s[desde:m[0]]))
		b.WriteString(s[m[0]:m[1]])
		desde = m[1]
	}
	b.WriteString(f(s[desde:]))
	return b.String()
}

// palabra reemplaza p solo como palabra entera: «ana» sí, «banana» no.
func palabra(s, p, marcador string) string {
	if strings.TrimSpace(p) == "" {
		return s
	}
	re := regexp.MustCompile(`(^|[^\p{L}\p{N}_])` + regexp.QuoteMeta(p) + `($|[^\p{L}\p{N}_])`)
	for re.MatchString(s) {
		s = re.ReplaceAllString(s, "${1}"+marcador+"${2}")
	}
	return s
}

// Nuevo arma el reporte limpio y lo valida con los topes del relay.
func Nuevo(d Datos, c incidentes.Contexto, valores []string) (Reporte, error) {
	tipo := strings.TrimSpace(d.Tipo)
	if tipo == "" {
		tipo = "bug"
	}
	if !slices.Contains(Tipos, tipo) {
		return Reporte{}, fmt.Errorf("tipo %q: usa %s", tipo, strings.Join(Tipos, ", "))
	}
	r := Reporte{
		Formato: 1,
		Tipo:    tipo,
		Titulo:  Limpiar(strings.TrimSpace(d.Titulo), c, valores),
		Detalle: Limpiar(strings.TrimSpace(d.Detalle), c, valores),
		Version: d.Version,
		SO:      d.SO,
	}
	if r.Titulo == "" {
		return Reporte{}, fmt.Errorf("falta el título: --titulo \"qué pasó, en una línea\"")
	}
	if n := utf8.RuneCountInString(r.Titulo); n > MaxTitulo {
		return Reporte{}, fmt.Errorf("el título tiene %d caracteres; el máximo es %d", n, MaxTitulo)
	}
	if n := utf8.RuneCountInString(r.Detalle); n > MaxDetalle {
		return Reporte{}, fmt.Errorf("el detalle tiene %d caracteres; el máximo es %d", n, MaxDetalle)
	}
	return r, nil
}

// Vista es el texto exacto que viaja, para que la persona lo lea antes de decir «sí».
func (r Reporte) Vista() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Tipo:    %s\n", r.Tipo)
	fmt.Fprintf(&b, "Título:  %s\n", r.Titulo)
	fmt.Fprintf(&b, "Versión: %s · %s\n", r.Version, r.SO)
	if r.Detalle == "" {
		return b.String()
	}
	fmt.Fprintf(&b, "Detalle:\n%s\n", r.Detalle)
	return b.String()
}

// mensaje es lo que se firma, en el orden del contrato.
func (r Reporte) mensaje() string {
	return fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s|%s", r.Formato, r.Instalacion, r.En, r.Tipo, r.Titulo, r.Detalle, r.Version, r.SO)
}
