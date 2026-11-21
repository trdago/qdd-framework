package remoto

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	// leerDelFinal es de a cuánto se lee hacia atrás el transcript para la cola inicial.
	leerDelFinal = 512 << 10
	// maxCola es lo más lejos del final que se busca la cola inicial (MEJ-078).
	maxCola = 8 << 20
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

// TranscriptDe es el transcript del proyecto (ruta = estado.ClaveLocal) con el turno más reciente, o "" si no hay.
// Manda la hora de su última línea con hora, no la del archivo: Claude Code agrega a sesiones viejas metadatos sin hora
// (cost-state, last-prompt) y con la fecha del archivo una vieja le ganaba a la que trabaja (FND-244). Con la misma
// hora, la del archivo. Se recorre del archivo más nuevo al más viejo y se para cuando uno es anterior a la mejor hora:
// ninguna línea es más nueva que su archivo, así que no se lee el final de cada transcript.
func TranscriptDe(ruta string) string {
	mejor, mejorT, mejorM := "", time.Time{}, time.Time{}
	for _, c := range transcriptsPorFecha(ruta) {
		if !mejorT.IsZero() && c.mod.Before(mejorT) {
			break
		}
		cuando := ultimaHora(c.archivo)
		if !masNuevo(cuando, c.mod, mejorT, mejorM) || !delProyecto(c.archivo, ruta) {
			continue
		}
		mejor, mejorT, mejorM = c.archivo, cuando, c.mod
	}
	return mejor
}

type transcript struct {
	archivo string
	mod     time.Time
}

// transcriptsPorFecha son los transcripts de las carpetas del proyecto (y de sus worktrees y subcarpetas), del archivo
// más nuevo al más viejo.
func transcriptsPorFecha(ruta string) []transcript {
	base := filepath.Join(consola.ClaudeDir(), "projects")
	cod := carpetaCodificada(ruta)
	dirs, _ := filepath.Glob(filepath.Join(base, cod+"*"))
	var out []transcript
	for _, d := range dirs {
		nombre := filepath.Base(d)
		if nombre != cod && !strings.HasPrefix(nombre, cod+"-") {
			continue
		}
		archivos, _ := filepath.Glob(filepath.Join(d, "*.jsonl"))
		for _, a := range archivos {
			if info, err := os.Stat(a); err == nil {
				out = append(out, transcript{archivo: a, mod: info.ModTime()})
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].mod.After(out[j].mod) })
	return out
}

func masNuevo(cuando, mod, mejorT, mejorM time.Time) bool {
	if !cuando.Equal(mejorT) {
		return cuando.After(mejorT)
	}
	return mod.After(mejorM)
}

// ultimaHora es la hora más nueva de las líneas con "timestamp" del final del transcript (los últimos leerDelFinal
// bytes); cero si no hay ninguna.
func ultimaHora(archivo string) time.Time {
	f, err := os.Open(archivo)
	if err != nil {
		return time.Time{}
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return time.Time{}
	}
	desde := max(info.Size()-leerDelFinal, 0)
	datos := make([]byte, info.Size()-desde)
	if _, err := f.ReadAt(datos, desde); err != nil && err != io.EOF {
		return time.Time{}
	}
	var mas time.Time
	for _, linea := range bytes.Split(datos, []byte("\n")) {
		if cuando := horaDeLinea(linea); cuando.After(mas) {
			mas = cuando
		}
	}
	return mas
}

func horaDeLinea(linea []byte) time.Time {
	if !bytes.Contains(linea, []byte(`"timestamp"`)) {
		return time.Time{}
	}
	var l struct {
		Timestamp string `json:"timestamp"`
	}
	if json.Unmarshal(linea, &l) != nil {
		return time.Time{}
	}
	cuando, _ := time.Parse(time.RFC3339Nano, l.Timestamp)
	return cuando
}

// sesionDeTranscript es la sesión de Claude Code de un transcript: Claude Code lo nombra <sesión>.jsonl.
func sesionDeTranscript(archivo string) string {
	if archivo == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(archivo), ".jsonl")
}

// sesionDeLaConsola es la sesión que el teléfono ve en la consola del proyecto, si sigue latiendo (MEJ-082): una
// instrucción del teléfono va a ella. Si la consola muestra una sesión que ya no late (o ninguna), no va dirigida.
func sesionDeLaConsola(ruta string, now time.Time) string {
	sesion := sesionDeTranscript(TranscriptDe(ruta))
	if sesion == "" || !consola.SesionLate(ruta, sesion, now) {
		return ""
	}
	return sesion
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
	// trabajando es desde cuándo Claude está trabajando en el turno en curso (hora del transcript); vacío si no
	// (FND-239).
	trabajando string
}

// marca es lo que una línea del transcript dice del trabajo de Claude.
type marca struct {
	Type        string `json:"type"`
	Subtype     string `json:"subtype"`
	QuerySource string `json:"querySource"`
	IsMeta      bool   `json:"isMeta"`
	IsSidechain bool   `json:"isSidechain"`
	Timestamp   string `json:"timestamp"`
	Message     struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// trabajoTras sigue desde cuándo trabaja Claude a lo largo de unas líneas completas (FND-239): Claude Code escribe
// cada respuesta en el transcript solo al terminarla, así que sin esto el teléfono no ve nada mientras tanto. La
// primera api-request del hilo principal empieza el trabajo; el fin del turno (turn_duration), una compactación, una
// interrupción o un mensaje nuevo de la persona lo terminan.
func trabajoTras(desde string, datos []byte) string {
	for _, linea := range bytes.Split(datos, []byte("\n")) {
		desde = trabajoDeLinea(desde, linea)
	}
	return desde
}

func trabajoDeLinea(desde string, linea []byte) string {
	if !bytes.Contains(linea, []byte(`"api-request"`)) && !bytes.Contains(linea, []byte(`"system"`)) && !bytes.Contains(linea, []byte(`"user"`)) {
		return desde
	}
	var m marca
	if json.Unmarshal(linea, &m) != nil {
		return desde
	}
	if m.Type == "api-request" && (m.QuerySource == "" || m.QuerySource == "repl_main_thread") && desde == "" {
		return m.Timestamp
	}
	if m.Type == "system" && (m.Subtype == "turn_duration" || m.Subtype == "compact_boundary") {
		return ""
	}
	if m.IsMeta || m.IsSidechain {
		return desde
	}
	var texto string
	if m.Type == "user" && json.Unmarshal(m.Message.Content, &texto) == nil && !strings.HasPrefix(strings.TrimSpace(texto), "<") {
		return ""
	}
	if m.Type == "user" && bytes.Contains(m.Message.Content, []byte("[Request interrupted by user")) {
		return ""
	}
	return desde
}

// abrirLector empieza al final del transcript y devuelve las últimas colaInicial líneas visibles.
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
	l.pos = info.Size()
	lineas, resto, leido := colaDe(f, info.Size())
	l.resto, l.trabajando = resto, trabajoTras("", leido)
	return l, lineas
}

// colaDe lee el transcript hacia atrás de a leerDelFinal hasta juntar colaInicial líneas visibles o llegar a maxCola
// del final (MEJ-078): un resultado de herramienta pesa cientos de KB y no da ninguna línea. Devuelve también lo que
// sigue al último salto de línea, una línea a medio escribir que el lector completa después, y las líneas completas que
// leyó, en orden, para saber si Claude está trabajando (FND-239).
func colaDe(f io.ReaderAt, fin int64) ([]LineaConsola, []byte, []byte) {
	var lineas []LineaConsola
	var leido []byte
	var resto, pendiente []byte // pendiente: el final de una línea que empieza más atrás
	vioSalto := false
	desde := fin
	for desde > 0 && fin-desde < maxCola && len(lineas) < colaInicial {
		inicio := max(desde-leerDelFinal, 0)
		bloque := make([]byte, desde-inicio)
		n, _ := f.ReadAt(bloque, inicio)
		bloque = append(bloque[:n], pendiente...)
		desde = inicio
		if !vioSalto {
			j := bytes.LastIndexByte(bloque, '\n')
			if j < 0 {
				pendiente = bloque
				continue
			}
			resto = append([]byte{}, bloque[j+1:]...)
			bloque = bloque[:j]
			vioSalto = true
		}
		pendiente = nil
		if desde > 0 {
			i := bytes.IndexByte(bloque, '\n')
			if i < 0 {
				pendiente = bloque
				continue
			}
			pendiente = append([]byte{}, bloque[:i]...)
			bloque = bloque[i+1:]
		}
		lineas = append(visibles(bloque), lineas...)
		leido = append(append(append([]byte{}, bloque...), '\n'), leido...)
	}
	if !vioSalto {
		resto = pendiente
	}
	return lineas[max(len(lineas)-colaInicial, 0):], resto, leido
}

// visibles son las líneas de la consola de varias líneas completas del transcript.
func visibles(datos []byte) []LineaConsola {
	var out []LineaConsola
	for _, linea := range bytes.Split(datos, []byte("\n")) {
		out = append(out, LineasDe(linea)...)
	}
	return out
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
	l.trabajando = trabajoTras(l.trabajando, datos[:i])
	return visibles(datos[:i])
}
