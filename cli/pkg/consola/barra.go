package consola

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/term"
)

// Rule: Prohibido usar 'else'
// Barra de estado de Claude Code en Go (blueprint consola-claude-code v7, RFC 0006 §3.6), sin bash, jq ni date, para
// que funcione igual en Linux, macOS y Windows. A la izquierda, el modelo y el modo de la IA; a la derecha, en lista y
// pegados al borde, el contexto, el uso de 5 h y el de la semana («contexto 38%», «5h 12% ↻ 18:30», «semana 40% ↻ sáb
// 09:00»). Sin carpeta ni segmento de tareas: eso lo muestra el panel (pedido del usuario, 2026-10-31).

// MargenBarra son las columnas que la barra deja libres a la derecha, para que una línea pegada al borde no salte de
// renglón dentro del margen que Claude Code pone a la barra.
const MargenBarra = 2

// nombresModo: el modo de permisos de la IA tal como lo lee la persona. Un modo desconocido no se muestra.
var nombresModo = map[string]string{"default": "normal", "acceptEdits": "acepta ediciones", "plan": "plan", "auto": "auto", "dontAsk": "no pregunta", "bypassPermissions": "sin permisos"}

// entrada son los campos del JSON que la consola entrega por la entrada estándar y que la barra usa.
type entrada struct {
	Cwd       string `json:"cwd"`
	Workspace struct {
		ProjectDir string `json:"project_dir"`
		CurrentDir string `json:"current_dir"`
	} `json:"workspace"`
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage *float64 `json:"used_percentage"`
	} `json:"context_window"`
	RateLimits struct {
		FiveHour tramoJSON `json:"five_hour"`
		SevenDay tramoJSON `json:"seven_day"`
	} `json:"rate_limits"`
}

type tramoJSON struct {
	UsedPercentage *float64        `json:"used_percentage"`
	ResetsAt       json.RawMessage `json:"resets_at"`
}

var dias = []string{"dom", "lun", "mar", "mié", "jue", "vie", "sáb"}

// Barra arma la barra. modo es el permission_mode de la IA (vacío si no se sabe) y ancho, las columnas de la terminal
// (0 si no se saben: la lista va sin relleno). Un campo que falta se omite (nunca un 0 % inventado); con JSON ilegible
// no muestra nada.
func Barra(datos []byte, now time.Time, modo string, ancho int) string {
	var in entrada
	if err := json.Unmarshal(datos, &in); err != nil {
		return ""
	}
	var lista []string
	if p := in.ContextWindow.UsedPercentage; p != nil {
		lista = append(lista, "contexto "+porcentaje(*p))
	}
	if t := tramo("5h", in.RateLimits.FiveHour, now, false); t != "" {
		lista = append(lista, t)
	}
	if t := tramo("semana", in.RateLimits.SevenDay, now, true); t != "" {
		lista = append(lista, t)
	}
	return componer(unir(" · ", in.Model.DisplayName, nombresModo[modo]), lista, ancho)
}

// componer pone la izquierda y el primero de la lista en la misma línea si caben; el resto, uno por línea a la derecha.
func componer(izquierda string, lista []string, ancho int) string {
	var lineas []string
	if len(lista) > 0 && cabe(izquierda, lista[0], ancho) {
		lineas = append(lineas, juntar(izquierda, lista[0], ancho))
		izquierda, lista = "", lista[1:]
	}
	if izquierda != "" {
		lineas = append(lineas, izquierda)
	}
	for _, l := range lista {
		lineas = append(lineas, aLaDerecha(l, ancho))
	}
	return strings.Join(lineas, "\n")
}

func util(ancho int) int {
	return ancho - MargenBarra
}

func largo(s string) int {
	return utf8.RuneCountInString(s)
}

func cabe(izquierda, derecha string, ancho int) bool {
	if ancho <= 0 || izquierda == "" {
		return true
	}
	return largo(izquierda)+2+largo(derecha) <= util(ancho)
}

func juntar(izquierda, derecha string, ancho int) string {
	if izquierda == "" {
		return aLaDerecha(derecha, ancho)
	}
	if ancho <= 0 {
		return izquierda + " · " + derecha
	}
	return izquierda + strings.Repeat(" ", util(ancho)-largo(izquierda)-largo(derecha)) + derecha
}

// aLaDerecha rellena la línea para que termine a MargenBarra del borde; sin ancho, o si no cabe, va tal cual.
func aLaDerecha(l string, ancho int) string {
	if ancho <= 0 || largo(l) >= util(ancho) {
		return l
	}
	return strings.Repeat(" ", util(ancho)-largo(l)) + l
}

func unir(sep string, partes ...string) string {
	var hay []string
	for _, p := range partes {
		if p != "" {
			hay = append(hay, p)
		}
	}
	return strings.Join(hay, sep)
}

// ModoDeLaBarra es el modo de la IA que qdd anotó en el último mensaje de la persona en este proyecto (qdd consola
// ciclo), o vacío si no se sabe: Claude Code no lo entrega a la barra.
func ModoDeLaBarra(datos []byte, now time.Time) string {
	var in entrada
	if json.Unmarshal(datos, &in) != nil {
		return ""
	}
	proyecto := primero(in.Workspace.ProjectDir, in.Workspace.CurrentDir, in.Cwd)
	if proyecto == "" {
		return ""
	}
	c, ok := ModoVigente(proyecto, now)
	if !ok {
		return ""
	}
	return c.Modo
}

// AnchoTerminal son las columnas de la terminal donde corre la consola: la barra corre sin terminal en su salida, así
// que pregunta a la terminal de la sesión (/dev/tty; CONOUT$ en Windows) y, si no hay, a COLUMNS. 0 si no se sabe.
func AnchoTerminal() int {
	ruta := "/dev/tty"
	if runtime.GOOS == "windows" {
		ruta = "CONOUT$"
	}
	f, err := os.OpenFile(ruta, os.O_RDWR, 0)
	if err != nil {
		return anchoDeEntorno()
	}
	defer f.Close()
	w, _, err := term.GetSize(int(f.Fd()))
	if err != nil || w <= 0 {
		return anchoDeEntorno()
	}
	return w
}

func anchoDeEntorno() int {
	n, err := strconv.Atoi(os.Getenv("COLUMNS"))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func tramo(rotulo string, t tramoJSON, now time.Time, conDia bool) string {
	if t.UsedPercentage == nil {
		return ""
	}
	s := rotulo + " " + porcentaje(*t.UsedPercentage)
	cuando, ok := renovacion(t.ResetsAt, now.Location())
	if !ok {
		return s
	}
	hora := cuando.Format("15:04")
	if conDia {
		hora = dias[cuando.Weekday()] + " " + hora
	}
	return s + " ↻ " + hora
}

// renovacion acepta segundos epoch (número o texto) o una fecha ISO 8601.
func renovacion(raw json.RawMessage, loc *time.Location) (time.Time, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return time.Time{}, false
	}
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return time.Unix(int64(n), 0).In(loc), true
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil || s == "" {
		return time.Time{}, false
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Unix(v, 0).In(loc), true
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t.In(loc), true
}

func porcentaje(p float64) string {
	return fmt.Sprintf("%.0f%%", p)
}

func nombreCarpeta(ruta string) string {
	limpia := strings.TrimRight(strings.ReplaceAll(filepath.ToSlash(ruta), `\`, "/"), "/")
	if i := strings.LastIndex(limpia, "/"); i >= 0 {
		return limpia[i+1:]
	}
	return limpia
}

func primero(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
