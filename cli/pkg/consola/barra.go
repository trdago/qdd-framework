package consola

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Barra de estado de Claude Code en Go (blueprint consola-claude-code, RFC 0006 §3.6): la misma línea que
// referencias/statusline.sh («📁 proyecto · modelo · contexto N% · 5h N% (↻ HH:MM) · semana N% (↻ día HH:MM)») más el
// segmento de estado por frente, sin bash, jq ni date, para que funcione igual en Linux, macOS y Windows.

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

// Barra arma la línea de la barra. Un campo que falta se omite (nunca un 0 % inventado); con JSON ilegible no
// muestra nada. conEstado agrega el segmento de estado por frente del proyecto si tiene .qdd/project/estado.yaml.
func Barra(datos []byte, now time.Time, conEstado bool) string {
	var in entrada
	if err := json.Unmarshal(datos, &in); err != nil {
		return ""
	}
	proyecto := primero(in.Workspace.ProjectDir, in.Workspace.CurrentDir, in.Cwd)
	partes := []string{"📁 " + nombreCarpeta(proyecto)}
	if in.Model.DisplayName != "" {
		partes = append(partes, in.Model.DisplayName)
	}
	if p := in.ContextWindow.UsedPercentage; p != nil {
		partes = append(partes, "contexto "+porcentaje(*p))
	}
	if t := tramo("5h", in.RateLimits.FiveHour, now, false); t != "" {
		partes = append(partes, t)
	}
	if t := tramo("semana", in.RateLimits.SevenDay, now, true); t != "" {
		partes = append(partes, t)
	}
	if conEstado && proyecto != "" {
		if s := segmentoEstado(proyecto, now); s != "" {
			partes = append(partes, s)
		}
	}
	return strings.Join(partes, " · ")
}

// segmentoEstado lee solo el archivo local del proyecto y corta a los 50 ms: si tarda o falla, no hay segmento.
func segmentoEstado(proyecto string, now time.Time) string {
	linea := make(chan string, 1)
	go func() {
		defer func() { _ = recover() }()
		linea <- estado.StatuslineDir(proyecto, now)
	}()
	select {
	case l := <-linea:
		return l
	case <-time.After(50 * time.Millisecond):
		return ""
	}
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
	return s + " (↻ " + hora + ")"
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
