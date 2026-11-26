package flujo

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Rule: Prohibido usar 'else'

// El asistente (RFC 0025, fase 4) es la sesión de Claude del moderador mientras se conversa con el cliente. Propone
// nodos, notas e historias como un invitado y la persona los aprueba uno por uno. No comenta: para responder ya tiene
// 'qdd flujo responder'.

const (
	// RolAsistente es el rol de lo que propone el asistente.
	RolAsistente = "asistente"
	// OrigenAsistente es el origen de lo que propone sin un minuto de la conversación que lo respalde.
	OrigenAsistente = "asistente"
	// prefijoTranscripcion lleva el minuto: «transcripcion@12:40».
	prefijoTranscripcion = "transcripcion@"
)

// opsDeAsistente son las que cambian el flujo, más la nota técnica y la historia de usuario.
var opsDeAsistente = []string{OpAgregar, OpModificar, OpMover, OpQuitar, OpConectar, OpDesconectar, OpNota, OpHistoria}

var minutoValido = regexp.MustCompile(`^[0-9]{1,3}:[0-5][0-9]$`)

// PermitidaAlAsistente dice si el asistente puede proponer la operación.
func (o Operacion) PermitidaAlAsistente() error {
	if !slices.Contains(opsDeAsistente, o.Op) {
		return fmt.Errorf("el asistente no propone «%s»: propone %s", o.Op, strings.Join(opsDeAsistente, ", "))
	}
	return nil
}

// OrigenDeAsistente es «asistente» o, con el minuto de la conversación (mm:ss), «transcripcion@mm:ss».
func OrigenDeAsistente(minuto string) (string, error) {
	if minuto == "" {
		return OrigenAsistente, nil
	}
	if !minutoValido.MatchString(minuto) {
		return "", fmt.Errorf("minuto %q: usa mm:ss, por ejemplo 12:40", minuto)
	}
	return prefijoTranscripcion + minuto, nil
}

func esOrigenDeAsistente(origen string) bool {
	if origen == OrigenAsistente {
		return true
	}
	minuto, ok := strings.CutPrefix(origen, prefijoTranscripcion)
	return ok && minutoValido.MatchString(minuto)
}

// AplicarDeAsistente aplica una propuesta del asistente que la persona aprobó. Queda en el historial con su origen y la
// IA se entera como de una edición de la persona.
func (o Operacion) AplicarDeAsistente(root, origen string, ahora time.Time) (string, error) {
	if err := o.PermitidaAlAsistente(); err != nil {
		return "", err
	}
	if !esOrigenDeAsistente(origen) {
		return "", fmt.Errorf("origen %q: es «asistente» o «transcripcion@mm:ss»", origen)
	}
	cambio := o.cambioDeAsistente(origen, ahora)
	firmado := func(f *Flujo) (hecho, error) {
		h, err := cambio(f)
		if err != nil {
			return hecho{}, err
		}
		h.linea = fmt.Sprintf("el asistente (%s) propuso y aprobaste: %s", strings.TrimPrefix(origen, prefijoTranscripcion), h.linea)
		return h, nil
	}
	_, linea, err := editarComo(root, o.Flujo, origen, true, ahora, firmado)
	return linea, err
}

func (o Operacion) cambioDeAsistente(origen string, ahora time.Time) Cambio {
	if o.Op == OpNota {
		return agregarItemDe(o.Nodo, ItemNota, o.Texto, OrigenAsistente, o.visible(), ahora)
	}
	if o.Op == OpHistoria {
		return AgregarHistoria(o.historia(), origen)
	}
	return o.cambio(ahora)
}
