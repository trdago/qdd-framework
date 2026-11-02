package remoto

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.4): la conversación que se ve en la consola remota sale del transcript de Claude Code del proyecto
// (<CLAUDE_CONFIG_DIR o ~/.claude>/projects/<carpeta codificada>/<sesión>.jsonl, la más reciente). Claude Code codifica
// la carpeta cambiando todo lo que no es letra o número por «-»; un worktree o una subcarpeta del proyecto tiene su
// propia carpeta, que empieza igual. Como «qdd» y «qdd-movil» también empiezan igual, se confirma con el cwd que
// anota el transcript. De cada línea salen los turnos de la persona y de la IA y el nombre de cada herramienta; nunca
// los resultados de las herramientas ni el razonamiento.

const (
	// maxTexto es lo más largo que va un turno a la consola.
	maxTexto = 2000
	// colaInicial es cuántas líneas ve quien empieza a mirar.
	colaInicial = 40
	// leerDelFinal es cuánto se lee del final del transcript para la cola inicial.
	leerDelFinal = 512 << 10
)

var noAlfanumerico = regexp.MustCompile(`[^A-Za-z0-9]`)

// LineaConsola es una línea de la consola remota.
type LineaConsola struct {
	Fuente string `json:"fuente"` // conversacion o pedido
	Rol    string `json:"rol"`    // persona, ia, herramienta o pedido
	Texto  string `json:"texto"`
	En     string `json:"en,omitempty"`
}

// carpetaCodificada es como Claude Code nombra la carpeta de un proyecto.
func carpetaCodificada(ruta string) string {
	return noAlfanumerico.ReplaceAllString(ruta, "-")
}

// TranscriptDe es el transcript más reciente del proyecto (ruta = estado.ClaveLocal), o "" si no hay.
func TranscriptDe(ruta string) string {
	base := filepath.Join(consola.ClaudeDir(), "projects")
	cod := carpetaCodificada(ruta)
	dirs, _ := filepath.Glob(filepath.Join(base, cod+"*"))
	mejor, mejorT := "", time.Time{}
	for _, d := range dirs {
		nombre := filepath.Base(d)
		if nombre != cod && !strings.HasPrefix(nombre, cod+"-") {
			continue
		}
		archivos, _ := filepath.Glob(filepath.Join(d, "*.jsonl"))
		for _, a := range archivos {
			info, err := os.Stat(a)
			if err != nil || !info.ModTime().After(mejorT) || !delProyecto(a, ruta) {
				continue
			}
			mejor, mejorT = a, info.ModTime()
		}
	}
	return mejor
}

// delProyecto confirma con el cwd del transcript que es de este proyecto (o de un worktree o subcarpeta suya).
func delProyecto(archivo, ruta string) bool {
	f, err := os.Open(archivo)
	if err != nil {
		return false
	}
	defer f.Close()
	sc := bufio.NewScanner(io.LimitReader(f, 4<<20))
	sc.Buffer(make([]byte, 64<<10), 4<<20)
	for sc.Scan() {
		var l struct {
			Cwd string `json:"cwd"`
		}
		if json.Unmarshal(sc.Bytes(), &l) != nil || l.Cwd == "" {
			continue
		}
		c := estado.ClaveLocal(l.Cwd)
		return c == ruta || strings.HasPrefix(c, ruta+string(filepath.Separator))
	}
	return false
}

// entrada es lo que se lee de una línea del transcript.
type entrada struct {
	Type        string `json:"type"`
	IsSidechain bool   `json:"isSidechain"`
	IsMeta      bool   `json:"isMeta"`
	Timestamp   string `json:"timestamp"`
	Message     struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type bloque struct {
	Type string `json:"type"`
	Text string `json:"text"`
	Name string `json:"name"`
}

// LineasDe convierte una línea del transcript en líneas de la consola (ninguna si no es un turno que se muestra).
func LineasDe(datos []byte) []LineaConsola {
	var e entrada
	if json.Unmarshal(datos, &e) != nil || e.IsSidechain || e.IsMeta || (e.Type != "user" && e.Type != "assistant") {
		return nil
	}
	rol := map[string]string{"user": "persona", "assistant": "ia"}[e.Type]
	var texto string
	if json.Unmarshal(e.Message.Content, &texto) == nil {
		return lineaDeTexto(rol, texto, e.Timestamp)
	}
	var bloques []bloque
	if json.Unmarshal(e.Message.Content, &bloques) != nil {
		return nil
	}
	var out []LineaConsola
	for _, b := range bloques {
		if b.Type == "text" {
			out = append(out, lineaDeTexto(rol, b.Text, e.Timestamp)...)
		}
		if b.Type == "tool_use" && b.Name != "" {
			out = append(out, LineaConsola{Fuente: "conversacion", Rol: "herramienta", Texto: b.Name, En: e.Timestamp})
		}
	}
	return out
}

// lineaDeTexto deja fuera lo que Claude Code inyecta (recordatorios del sistema, salidas de comandos locales).
func lineaDeTexto(rol, texto, en string) []LineaConsola {
	t := strings.TrimSpace(texto)
	if t == "" || strings.HasPrefix(t, "<") {
		return nil
	}
	return []LineaConsola{{Fuente: "conversacion", Rol: rol, Texto: Recortar(t, maxTexto), En: en}}
}

// lector sigue un transcript desde donde quedó.
type lector struct {
	archivo string
	pos     int64
	resto   []byte
}

// abrirLector empieza al final del transcript y devuelve las últimas colaInicial líneas.
func abrirLector(archivo string) (*lector, []LineaConsola) {
	l := &lector{archivo: archivo}
	if archivo == "" {
		return l, nil
	}
	f, err := os.Open(archivo)
	if err != nil {
		return l, nil
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return l, nil
	}
	desde := max(info.Size()-leerDelFinal, 0)
	datos := make([]byte, info.Size()-desde)
	n, _ := f.ReadAt(datos, desde)
	datos = datos[:n]
	l.pos = desde + int64(n)
	if desde > 0 {
		if i := bytes.IndexByte(datos, '\n'); i >= 0 {
			datos = datos[i+1:]
		}
	}
	lineas := l.partir(datos)
	if len(lineas) > colaInicial {
		lineas = lineas[len(lineas)-colaInicial:]
	}
	return l, lineas
}

// leer devuelve lo nuevo desde la última vez (solo líneas completas).
func (l *lector) leer() []LineaConsola {
	if l.archivo == "" {
		return nil
	}
	f, err := os.Open(l.archivo)
	if err != nil {
		return nil
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.Size() <= l.pos {
		return nil
	}
	datos := make([]byte, min(info.Size()-l.pos, 4<<20))
	n, _ := f.ReadAt(datos, l.pos)
	l.pos += int64(n)
	return l.partir(datos[:n])
}

func (l *lector) partir(datos []byte) []LineaConsola {
	datos = append(l.resto, datos...)
	i := bytes.LastIndexByte(datos, '\n')
	if i < 0 {
		l.resto = datos
		return nil
	}
	l.resto = append([]byte{}, datos[i+1:]...)
	var out []LineaConsola
	for _, linea := range bytes.Split(datos[:i], []byte("\n")) {
		out = append(out, LineasDe(linea)...)
	}
	return out
}
