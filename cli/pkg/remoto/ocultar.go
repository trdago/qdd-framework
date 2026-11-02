package remoto

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Rule: Prohibido usar 'else'
// RFC 0013, inv. 16: nada que parezca un secreto sale del PC. Todo lo que se envía (la consola, el resultado o el error
// de una acción, el log) pasa por aquí: claves de API y tokens conocidos, JWT, cadenas de conexión con contraseña,
// llaves privadas, «Authorization: Bearer …», asignaciones estilo .env cuyo nombre dice que es un secreto, y los
// valores reales de los .env del proyecto (que se leen solo para ocultarlos; nunca se envían).

// Oculto es lo que queda en lugar de un secreto.
const Oculto = "•••"

// minValorEnv: un valor de .env más corto no se busca en el texto (taparía palabras comunes).
const minValorEnv = 8

var (
	llavePrivada = regexp.MustCompile(`-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----(?s:.*?)(?:-----END [A-Z0-9 ]*PRIVATE KEY-----|\z)`)
	conexion     = regexp.MustCompile(`(?i)\b((?:postgres(?:ql)?|mysql|mariadb|mongodb(?:\+srv)?|rediss?|amqps?|mssql|sqlserver)://)([^:@/\s]*):([^@\s]+)@`)
	jwt          = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{4,}\.eyJ[A-Za-z0-9_-]{4,}\.[A-Za-z0-9_-]*`)
	bearer       = regexp.MustCompile(`(?i)\b(bearer\s+)[A-Za-z0-9._~+/=-]{8,}`)
	tokens       = []*regexp.Regexp{
		regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{10,}`),
		regexp.MustCompile(`\bsk-(?:proj-|svcacct-|admin-)?[A-Za-z0-9_-]{16,}`),
		regexp.MustCompile(`\b[rs]k_(?:live|test)_[A-Za-z0-9]{10,}`),
		regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}`),
		regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}`),
		regexp.MustCompile(`\bglpat-[A-Za-z0-9_-]{20,}`),
		regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`),
		regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`),
		regexp.MustCompile(`\bsb_secret_[A-Za-z0-9_-]{10,}`),
		regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`),
		regexp.MustCompile(`\bnpm_[A-Za-z0-9]{36}`),
	}
	// asignacion es NOMBRE=valor o NOMBRE: valor (también con export y entre comillas); el nombre decide.
	asignacion = regexp.MustCompile(`(^|[\s;,{(])((?:export\s+)?["']?([A-Za-z_][A-Za-z0-9_.-]*)["']?\s*[=:]\s*)("[^"\n]*"|'[^'\n]*'|[^\s"',;}]+)`)
	// palabrasSecretas: un segmento del nombre (separado por _ - .) que dice que el valor es un secreto.
	palabrasSecretas = []string{"KEY", "TOKEN", "SECRET", "PASSWORD", "PASS", "PWD", "CREDENTIAL", "CREDENTIALS", "DSN", "APIKEY", "PASSWD", "PRIVATE"}
	// comunes: valores de .env que no son secretos y aparecen en cualquier texto.
	comunes = map[string]bool{"production": true, "development": true, "localhost": true, "staging": true}
)

// Ocultar reemplaza por «•••» lo que parece un secreto, con los patrones conocidos.
func Ocultar(s string) string {
	if s == "" {
		return s
	}
	s = llavePrivada.ReplaceAllString(s, Oculto)
	s = conexion.ReplaceAllString(s, "${1}${2}:"+Oculto+"@")
	s = jwt.ReplaceAllString(s, Oculto)
	s = bearer.ReplaceAllString(s, "${1}"+Oculto)
	for _, re := range tokens {
		s = re.ReplaceAllString(s, Oculto)
	}
	return asignacion.ReplaceAllStringFunc(s, ocultarAsignacion)
}

func ocultarAsignacion(m string) string {
	partes := asignacion.FindStringSubmatch(m)
	valor := strings.Trim(partes[4], `"'`)
	if !nombreSecreto(partes[3]) || valor == "" || valor == Oculto {
		return m
	}
	return partes[1] + partes[2] + Oculto
}

// nombreSecreto: DATABASE_URL, o algún segmento del nombre es una palabra secreta (API_KEY, GH_TOKEN, db.password),
// o el nombre termina en APIKEY, TOKEN, SECRET o PASSWORD (MYAPIKEY; MONKEY no).
func nombreSecreto(nombre string) bool {
	n := strings.ToUpper(nombre)
	if strings.Contains(n, "DATABASE_URL") || strings.HasSuffix(n, "_DSN") {
		return true
	}
	segmentos := strings.FieldsFunc(n, func(r rune) bool { return r == '_' || r == '-' || r == '.' })
	for _, seg := range segmentos {
		for _, p := range palabrasSecretas {
			if seg == p {
				return true
			}
		}
	}
	for _, p := range []string{"APIKEY", "TOKEN", "SECRET", "PASSWORD"} {
		if strings.HasSuffix(n, p) {
			return true
		}
	}
	return false
}

// Ocultador oculta también los valores reales de los .env del proyecto.
type Ocultador struct {
	valores []string
}

// OcultadorDe lee los .env* de root y de sus carpetas de primer nivel (sin node_modules, .git ni vendor).
func OcultadorDe(root string) *Ocultador {
	o := &Ocultador{}
	if root == "" {
		return o
	}
	vistos := map[string]bool{}
	for _, dir := range carpetasConEnv(root) {
		archivos, _ := filepath.Glob(filepath.Join(dir, ".env*"))
		for _, a := range archivos {
			for _, v := range valoresEnv(a) {
				vistos[v] = true
			}
		}
	}
	for v := range vistos {
		o.valores = append(o.valores, v)
	}
	// Los más largos primero: un valor que contiene a otro se tapa entero.
	sort.Slice(o.valores, func(i, j int) bool { return len(o.valores[i]) > len(o.valores[j]) })
	return o
}

var sinEnv = map[string]bool{"node_modules": true, ".git": true, "vendor": true, "dist": true, "build": true, "tmp": true}

func carpetasConEnv(root string) []string {
	dirs := []string{root}
	entradas, err := os.ReadDir(root)
	if err != nil {
		return dirs
	}
	for _, e := range entradas {
		if e.IsDir() && !sinEnv[e.Name()] {
			dirs = append(dirs, filepath.Join(root, e.Name()))
		}
	}
	return dirs
}

// valoresEnv son los valores de un archivo .env de al menos minValorEnv caracteres.
func valoresEnv(archivo string) []string {
	f, err := os.Open(archivo)
	if err != nil {
		return nil
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() || info.Size() > 1<<20 {
		return nil
	}
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		v, ok := valorDeLinea(sc.Text())
		if ok && utf8.RuneCountInString(v) >= minValorEnv && !comunes[strings.ToLower(v)] {
			out = append(out, v)
		}
	}
	return out
}

func valorDeLinea(linea string) (string, bool) {
	l := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(linea), "export "))
	if l == "" || strings.HasPrefix(l, "#") {
		return "", false
	}
	i := strings.IndexByte(l, '=')
	if i <= 0 {
		return "", false
	}
	v := strings.TrimSpace(l[i+1:])
	if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
		return v[1 : len(v)-1], true
	}
	if j := strings.Index(v, " #"); j >= 0 {
		v = strings.TrimSpace(v[:j])
	}
	return v, v != ""
}

// Ocultar tapa los valores de los .env del proyecto y después los patrones conocidos.
func (o *Ocultador) Ocultar(s string) string {
	if o == nil {
		return Ocultar(s)
	}
	for _, v := range o.valores {
		s = strings.ReplaceAll(s, v, Oculto)
	}
	return Ocultar(s)
}

// Recortar deja como mucho n caracteres de s (con «…» al final si cortó).
func Recortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
