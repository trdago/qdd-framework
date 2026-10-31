package consola

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9 (pedidos del usuario, 2026-10-30): el mod de Claude Code marca un ciclo con cada mensaje de la persona
// (qdd consola ciclo --modo <permission_mode>). Un pedido hecho antes del último ciclo ya se vio y sale del tablero; el
// modo de permisos de la IA principal le dice a qdd_project_context cuándo pedir autorización.

// VidaModo: un modo anotado hace más que esto ya no se sabe (la sesión pudo cerrarse o cambiar).
const VidaModo = 12 * time.Hour

// modosDeClaudeCode son los modos de permisos que entrega Claude Code (permission_mode); otro valor no se anota.
var modosDeClaudeCode = map[string]bool{"default": true, "acceptEdits": true, "plan": true, "auto": true, "dontAsk": true, "bypassPermissions": true}

// Ciclo es el último mensaje de la persona: cuándo y en qué modo estaba la IA.
type Ciclo struct {
	Formato int       `json:"formato"`
	Modo    string    `json:"modo,omitempty"`
	En      time.Time `json:"en"`
}

func cicloPath(root string) (string, error) {
	return RutaLocal(root, "ciclo")
}

// MarcarCiclo anota un mensaje de la persona en el proyecto. Un modo desconocido queda sin medir, no inventado.
func MarcarCiclo(root, modo string, now time.Time) error {
	ruta, err := cicloPath(root)
	if err != nil {
		return err
	}
	c := Ciclo{Formato: 1, En: now}
	if modosDeClaudeCode[modo] {
		c.Modo = modo
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o700); err != nil {
		return err
	}
	tmp := ruta + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, ruta); err != nil {
		return err
	}
	return AnotarHuellas(root, now)
}

// UltimoCiclo devuelve el último ciclo del proyecto; ok falso si nunca hubo uno.
func UltimoCiclo(root string) (Ciclo, bool) {
	ruta, err := cicloPath(root)
	if err != nil {
		return Ciclo{}, false
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return Ciclo{}, false
	}
	var c Ciclo
	if json.Unmarshal(b, &c) != nil || c.En.IsZero() {
		return Ciclo{}, false
	}
	return c, true
}

// ModoVigente es el modo del último ciclo si se conoce y es de hace menos de VidaModo.
func ModoVigente(root string, now time.Time) (Ciclo, bool) {
	c, ok := UltimoCiclo(root)
	if !ok || c.Modo == "" || now.Sub(c.En) > VidaModo {
		return Ciclo{}, false
	}
	return c, true
}
