package materializar

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Origen del estilo: el de las ventanas del usuario (hypr-materialize) o el de por defecto, que vale en cualquier
// sistema (el acento del tema del terminal).
const (
	OrigenHypr       = "hypr-materialize"
	OrigenPorDefecto = "por defecto"
)

// Estilo es el acento y las duraciones del efecto.
type Estilo struct {
	Acento   string `json:"acento,omitempty"` // hex; vacío: el acento del tema del terminal
	AbrirMs  int    `json:"abrir_ms"`
	CerrarMs int    `json:"cerrar_ms"`
	Origen   string `json:"origen"`
}

var (
	reShader = regexp.MustCompile(`shader_(open|close)_default:(\S+?)@([0-9.]+)`)
	reAcento = regexp.MustCompile(`ACCENT\s*=\s*vec3\(\s*([0-9.]+)\s*,\s*([0-9.]+)\s*,\s*([0-9.]+)\s*\)`)
)

// EstiloDe lee el estilo de hypr-materialize (materialize.conf y su shader de abrir) en la configuración del usuario
// home; si no está, o no se entiende, es el de por defecto. Solo lee: nunca escribe en la configuración del escritorio.
func EstiloDe(home string) Estilo {
	porDefecto := Estilo{AbrirMs: AbrirMs, CerrarMs: CerrarMs, Origen: OrigenPorDefecto}
	conf, err := os.ReadFile(filepath.Join(configDe(home), "hypr", "materialize.conf"))
	if err != nil {
		return porDefecto
	}
	e := porDefecto
	abrir := ""
	for _, m := range reShader.FindAllStringSubmatch(string(conf), -1) {
		ms := segundosAMs(m[3])
		if m[1] == "open" {
			abrir = expandir(m[2], home)
			e.AbrirMs = elegir(ms, e.AbrirMs)
			continue
		}
		e.CerrarMs = elegir(ms, e.CerrarMs)
	}
	shader, err := os.ReadFile(abrir)
	if abrir == "" || err != nil {
		return porDefecto
	}
	acento, ok := acentoDe(string(shader))
	if !ok {
		return porDefecto
	}
	e.Acento = acento
	e.Origen = OrigenHypr
	return e
}

func configDe(home string) string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return x
	}
	return filepath.Join(home, ".config")
}

func expandir(ruta, home string) string {
	if strings.HasPrefix(ruta, "~/") {
		return filepath.Join(home, ruta[2:])
	}
	return ruta
}

func segundosAMs(s string) int {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || v <= 0 || v > 10 {
		return 0
	}
	return int(math.Round(v * 1000))
}

func elegir(ms, anterior int) int {
	if ms <= 0 {
		return anterior
	}
	return ms
}

func acentoDe(shader string) (string, bool) {
	m := reAcento.FindStringSubmatch(shader)
	if m == nil {
		return "", false
	}
	var c [3]float64
	for i := range c {
		v, err := strconv.ParseFloat(m[i+1], 64)
		if err != nil || v < 0 || v > 1 {
			return "", false
		}
		c[i] = v
	}
	return hex(c), true
}

// String describe el estilo para doctor: «acento #7daea3 · 0,8 s / 0,4 s (hypr-materialize)».
func (e Estilo) String() string {
	acento := e.Acento
	if acento == "" {
		acento = "del tema"
	}
	return fmt.Sprintf("acento %s · %s s / %s s (%s)", acento, coma(e.AbrirMs), coma(e.CerrarMs), e.Origen)
}

func coma(ms int) string {
	return strings.Replace(strconv.FormatFloat(float64(ms)/1000, 'f', -1, 64), ".", ",", 1)
}
