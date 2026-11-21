package consola

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Lo que Claude Code publica de cada proceso en <config>/sessions/<pid>.json (MEJ-084): su sesión y si está detenida en
// un diálogo (status «waiting», por ejemplo un permiso). Un mensaje que se le da a esa sesión queda en la cola de Claude
// Code detrás del diálogo. Si el archivo no está, no se lee o su proceso ya no corre, no cuenta: se entrega como antes.

const esperaUnDialogo = "waiting"

type procesoDeClaude struct {
	PID       json.RawMessage `json:"pid"`
	SessionID string          `json:"sessionId"`
	Status    string          `json:"status"`
}

// SesionEnDialogo dice si algún proceso vivo de esa sesión de Claude Code espera un diálogo en la terminal.
func SesionEnDialogo(sesion string) bool {
	archivos, _ := filepath.Glob(filepath.Join(claudeDir(), "sessions", "*.json"))
	for _, a := range archivos {
		if procesoEnDialogo(a, sesion) {
			return true
		}
	}
	return false
}

func procesoEnDialogo(archivo, sesion string) bool {
	b, err := os.ReadFile(archivo)
	if err != nil {
		return false
	}
	var p procesoDeClaude
	if json.Unmarshal(b, &p) != nil || p.SessionID != sesion || p.Status != esperaUnDialogo {
		return false
	}
	return procesoVivo(pidDe(p.PID))
}

// pidDe acepta el pid como número o como texto.
func pidDe(crudo json.RawMessage) int {
	var texto string
	if json.Unmarshal(crudo, &texto) == nil {
		n, _ := strconv.Atoi(texto)
		return n
	}
	var n int
	_ = json.Unmarshal(crudo, &n)
	return n
}

// puedeRecibir: la sesión late y Claude Code no la tiene detenida en un diálogo.
func puedeRecibir(root, sesion string, now time.Time) bool {
	return SesionLate(root, sesion, now) && !SesionEnDialogo(sesion)
}

// motivoSinRecibir dice por qué una sesión abierta no toma instrucciones, con su id corto (el del transcript).
func motivoSinRecibir(sesion string) string {
	corto, _, _ := strings.Cut(sesion, "-")
	return fmt.Sprintf("la sesión %s espera un diálogo en la terminal (un permiso o una pregunta): se entrega cuando lo respondas o en otra sesión abierta", corto)
}
