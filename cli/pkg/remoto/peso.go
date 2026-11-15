package remoto

import (
	"fmt"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-209: Realtime descarta sin aviso un broadcast que pasa su tope, y el estado completo (el panel de todos los
// proyectos) crece con cada proyecto. El servicio anota cuánto pesó el último estado completo que mandó; doctor lo
// compara con el tope sin usar la red (REM-21). TopeBroadcast es el del plan Free de Supabase, el más bajo: con otro
// plan el tope es mayor y el aviso llega antes de tiempo, nunca tarde.

const (
	TopeBroadcast = 256 * 1024
	// avisoPeso: desde qué fracción del tope se avisa.
	avisoPeso = 0.8
	// TopeSalidaTelefono: la salida de cada logro que va al teléfono (pedido de qdd-movile, 2026-11-15). Lo logrado era
	// lo que más pesaba del estado; el archivo local guarda hasta consola.TopeSalidaLogro, y la cantidad y los días de lo
	// logrado no cambian (eso cambia el contrato y lo decide la persona).
	TopeSalidaTelefono = 500
)

// salidaParaTelefono deja el final de la salida, que es lo que dice cómo terminó, con «…» delante si recortó.
func salidaParaTelefono(s string) string {
	r := []rune(s)
	if len(r) <= TopeSalidaTelefono {
		return s
	}
	return "…" + string(r[len(r)-(TopeSalidaTelefono-1):])
}

type pesoGuardado struct {
	Formato int       `json:"formato"`
	Bytes   int       `json:"bytes"`
	En      time.Time `json:"en"`
}

// PesoEstado es el último estado completo que mandó el servicio.
type PesoEstado struct {
	Bytes int
	En    time.Time
}

// GuardarPeso anota el peso del último estado completo (<QDD_HOME>/remoto/peso.json).
func GuardarPeso(bytes int, now time.Time) error {
	return guardarPrivado("peso.json", pesoGuardado{Formato: 1, Bytes: bytes, En: now.UTC()})
}

// LeerPeso es lo que dejó GuardarPeso.
func LeerPeso() (PesoEstado, error) {
	var g pesoGuardado
	if err := leerPrivado("peso.json", &g); err != nil {
		return PesoEstado{}, err
	}
	return PesoEstado{Bytes: g.Bytes, En: g.En}, nil
}

// AvisoDePeso dice, desde el 80 % del tope, cuánto pesa el estado y cuál es el tope; "" si está lejos.
func AvisoDePeso(bytes int) string {
	if float64(bytes) < avisoPeso*TopeBroadcast {
		return ""
	}
	return fmt.Sprintf("el estado para el teléfono pesa %d KB y Realtime descarta sin aviso lo que pasa de %d KB en el plan Free", bytes/1024, TopeBroadcast/1024)
}
