package consola

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'
// Aviso al chat (pedido del usuario, 2026-11-02: «el chat principal tiene que enterarse… cuando acepto la tarea en el
// panel»): el panel dice qué sesión de Claude Code confirma un pedido (qdd consola accion --sesion) y, cuando termina
// (hecho o con falla), qdd le entrega a esa sesión un mensaje para la IA. La sesión lo reclama con qdd consola avisado,
// que lo marca con la cola bloqueada: llega una sola vez aunque haya varias sesiones abiertas o el mod se recargue.

// MaxMensajeAviso es el tope del mensaje que la IA recibe al terminar un pedido: un resumen estilo chat con el final
// de la salida, que es lo que dice cómo terminó.
const MaxMensajeAviso = 600

// maxSesion es el tope del id de sesión que se guarda (lo da el panel; uno más largo no es un id).
const maxSesion = 128

var errSinAviso = errors.New("no hay un aviso pendiente para esta sesión")

// porAvisar: terminó (hecho o con falla), lo confirmó una sesión que dijo cuál es y todavía no se le avisó.
func (p Pedido) porAvisar() bool {
	return (p.Estado == EstadoHecho || p.Estado == EstadoFallo) && p.Sesion != "" && !p.Avisado
}

// mensajeAviso es lo que la IA de la sesión que confirmó lee al terminar el pedido: qué pedido, cómo terminó, el final
// de su salida y qué hacer ahora.
func mensajeAviso(p Pedido) string {
	cabeza := fmt.Sprintf("✓ Pedido [%s] «%s» terminó bien (exit 0).", p.Tecla, p.Motivo)
	cola := " Sigue con lo que venía después."
	if p.Estado == EstadoFallo {
		cabeza = fmt.Sprintf("✗ Pedido [%s] «%s» falló (exit %d).", p.Tecla, p.Motivo, p.Codigo)
		cola = " Busca la causa y arréglala; si hay que reintentarlo, déjalo de nuevo en el panel."
	}
	salida := strings.TrimSpace(p.Salida)
	if salida == "" {
		return cabeza + cola
	}
	cabe := MaxMensajeAviso - len([]rune(cabeza+cola)) - len([]rune(" Salida: …"))
	return cabeza + " Salida: " + finalDe(salida, cabe) + cola
}

// finalDe deja las últimas n letras de s (con «…» delante si cortó): el final de una salida dice cómo terminó.
func finalDe(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n < 0 {
		n = 0
	}
	return "…" + string(r[len(r)-n:])
}

// sesionValida recorta el id que da el panel; uno vacío o demasiado largo no sirve para avisar.
func sesionValida(sesion string) string {
	s := strings.TrimSpace(sesion)
	if len(s) > maxSesion {
		return ""
	}
	return s
}

// tomarRetirados devuelve, una sola vez, el aviso de cada pedido retirado que pidió esta sesión (RFC 0017). Lee antes de
// bloquear la cola: casi nunca hay nada, y esto corre con cada mensaje de la persona.
func tomarRetirados(root, sesion string, now time.Time) []string {
	sesion = sesionValida(sesion)
	if sesion == "" {
		return nil
	}
	hay := false
	for _, p := range PedidosA(root, now) {
		hay = hay || retiradoPorAvisar(p, sesion)
	}
	if !hay {
		return nil
	}
	var avisos []string
	_ = conCola(root, now, func(cola []Pedido) ([]Pedido, error) {
		for i, p := range cola {
			if !retiradoPorAvisar(p, sesion) {
				continue
			}
			cola[i].Avisado = true
			avisos = append(avisos, fmt.Sprintf("[qdd · panel] Se retiró tu pedido «%s»: su tarea «%s» siguió cambiando. Si todavía va, pídelo de nuevo.", p.Motivo, p.Tarea))
		}
		return cola, nil
	})
	return avisos
}

func retiradoPorAvisar(p Pedido, sesion string) bool {
	return p.Estado == EstadoRetirado && p.Pidio == sesion && !p.Avisado
}

// Avisado marca que la sesión ya recibió el aviso del pedido (id con o sin el prefijo «pedido:»). Solo la sesión que
// lo confirmó lo reclama, y una sola vez: si ya se avisó, si es de otra sesión o si no terminó, es un error.
func Avisado(root, id, sesion string) error {
	_, err := ReclamarAviso(root, id, sesion)
	return err
}

// ReclamarAviso es Avisado y además devuelve el mensaje del pedido tal como quedó al reclamarlo: el panel lo arma con
// una lectura anterior, que pudo cambiar entremedio (FND-187).
func ReclamarAviso(root, id, sesion string) (string, error) {
	id = strings.TrimPrefix(id, prefijoPedido)
	sesion = sesionValida(sesion)
	var mensaje string
	err := conCola(root, time.Now(), func(cola []Pedido) ([]Pedido, error) {
		i := indiceID(cola, id)
		if i < 0 {
			return nil, fmt.Errorf("no hay un pedido '%s' en la cola", id)
		}
		if sesion == "" || cola[i].Sesion != sesion || !cola[i].porAvisar() {
			return nil, errSinAviso
		}
		cola[i].Avisado = true
		mensaje = mensajeAviso(cola[i])
		return cola, nil
	})
	return mensaje, err
}
