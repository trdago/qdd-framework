package incidentes

import (
	"os"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// Rule: Prohibido usar 'else'
// Nada del proyecto sale de la máquina: Sanitizar deja solo el texto que escribió QDD y reemplaza todo lo demás por
// un marcador. Ante la duda, se reemplaza: perder un detalle cuesta menos que filtrar uno.

// Contexto es lo de esta máquina que nunca viaja: la carpeta del proyecto, la del usuario y su nombre.
type Contexto struct {
	Raiz    string
	Home    string
	Usuario string
}

// ContextoDe arma el contexto de la carpeta actual.
func ContextoDe(raiz string) Contexto {
	c := Contexto{Raiz: raiz}
	c.Home, _ = os.UserHomeDir()
	if u, err := user.Current(); err == nil {
		c.Usuario = u.Username
	}
	return c
}

var (
	reClaveValor = regexp.MustCompile(`(?i)\b(token|clave|password|passwd|secret|api[_-]?key|bearer)(\s*[:=]\s*|\s+)\S+`)
	reCorreo     = regexp.MustCompile(`[\w.+\-]+@[\w\-]+(\.[\w\-]+)+`)
	reURL        = regexp.MustCompile(`\b[a-zA-Z][a-zA-Z0-9+.\-]*://\S+`)
	rePrefijo    = regexp.MustCompile(`\b(ghp|gho|ghu|ghs|ghr|github_pat|sk|npm|xox[abpr]|AKIA)[A-Za-z0-9_\-]{8,}`)
	reLargo      = regexp.MustCompile(`\b[A-Za-z0-9+_\-]{32,}\b`)
	reRuta       = regexp.MustCompile(`(?:~|\.{1,2})?/?[\w.\-]+(?:/[\w.\-]+)+/?|(?:~|\.{1,2})?/[\w.\-]+/?`)
	reArchivo    = regexp.MustCompile(`\b[\w\-]+\.(?:go|py|js|mjs|ts|tsx|jsx|json|ya?ml|toml|ini|csv|tsv|txt|md|sql|env|sh|rb|java|kt|rs|c|h|cpp|cs|php|html|css|xml|lock|log|pdf|docx?|xlsx?|pptx?|png|jpe?g|zip)\b`)
	reComillas   = regexp.MustCompile(`'[^']*'|"[^"]*"|«[^»]*»|“[^”]*”|` + "`[^`]*`")
)

// archivosDeQDD son los nombres que QDD escribe y que sirven para reproducir: se dejan.
var archivosDeQDD = map[string]bool{
	"estado.yaml": true, "config.yaml": true, "blueprints.lock.yaml": true, "state.json": true,
	"cognitive_history.json": true, "blueprint.yaml": true, "certificacion.yaml": true, "construccion.yaml": true,
}

// Sanitizar reemplaza en un mensaje todo lo que puede ser del proyecto o de la persona: credenciales, correos, URLs,
// rutas, nombres de archivo y valores entre comillas; luego el nombre de la carpeta del proyecto y el del usuario.
// Lo de .qdd/ y los archivos que escribe QDD se quedan: son de QDD y sirven para reproducir.
func Sanitizar(s string, c Contexto) string {
	s = reClaveValor.ReplaceAllString(s, "$1=<secreto>")
	s = reCorreo.ReplaceAllString(s, "<correo>")
	s = reURL.ReplaceAllString(s, "<url>")
	s = rePrefijo.ReplaceAllString(s, "<secreto>")
	s = reRuta.ReplaceAllStringFunc(s, ruta)
	s = reArchivo.ReplaceAllStringFunc(s, archivo)
	s = reLargo.ReplaceAllString(s, "<secreto>")
	s = reComillas.ReplaceAllStringFunc(s, comillas)
	s = reemplazarPalabra(s, filepath.Base(c.Raiz), "<proyecto>")
	return reemplazarPalabra(s, c.Usuario, "<usuario>")
}

// ruta deja las carpetas de .qdd/ (son de QDD) y reemplaza todo lo demás; el último tramo de una ruta de .qdd/ puede
// ser del proyecto (modules/pagos.yaml) y se queda solo si es un archivo que QDD escribe.
func ruta(r string) string {
	if !strings.HasPrefix(r, ".qdd/") {
		return "<ruta>"
	}
	dir, ultimo := path.Split(strings.TrimSuffix(r, "/"))
	if archivosDeQDD[ultimo] || !strings.Contains(ultimo, ".") {
		return r
	}
	return dir + "<archivo>"
}

func archivo(a string) string {
	if archivosDeQDD[a] {
		return a
	}
	return "<archivo>"
}

func comillas(q string) string {
	abre, cierra := []rune(q)[0], []rune(q)[len([]rune(q))-1]
	return string(abre) + "<valor>" + string(cierra)
}

// reemplazarPalabra cambia una palabra entera (sin importar mayúsculas); las de menos de 3 letras no, para no
// romper el mensaje («a», «io»).
func reemplazarPalabra(s, palabra, marcador string) string {
	if len(palabra) < 3 || palabra == "." || palabra == string(filepath.Separator) {
		return s
	}
	re := regexp.MustCompile(`(?i)(^|[^\pL\pN_\-])` + regexp.QuoteMeta(palabra) + `($|[^\pL\pN_\-])`)
	return re.ReplaceAllString(s, "${1}"+marcador+"${2}")
}

// Comando es lo que se corrió, sin datos: el camino del comando de QDD, el nombre de cada bandera y un marcador por
// cada valor o argumento.
func Comando(camino string, args []string) string {
	partes := []string{camino}
	valorSigue := false
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			partes = append(partes, marcadorDe(valorSigue))
			valorSigue = false
			continue
		}
		nombre, _, conValor := strings.Cut(a, "=")
		if conValor {
			partes = append(partes, nombre+"=<valor>")
			valorSigue = false
			continue
		}
		partes = append(partes, nombre)
		valorSigue = strings.HasPrefix(a, "--")
	}
	return strings.Join(partes, " ")
}

func marcadorDe(valor bool) string {
	if valor {
		return "<valor>"
	}
	return "<arg>"
}
