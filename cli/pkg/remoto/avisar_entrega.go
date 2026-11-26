package remoto

import (
	"encoding/json"
	"fmt"
	"time"
)

// Rule: Prohibido usar 'else'
// Un push que el relay aceptó pero no llegó a ningún teléfono (MEJ-027, FND-217). avisar responde 200 con {enviados} y,
// en un relay nuevo, {destinos} (cuántos teléfonos tenían token) y {motivo} de lista cerrada solo con enviados 0
// (contrato con qdd-movile, 2026-11-15). El servicio lo anota para doctor (REM-22), que solo avisa: nunca se reintenta
// ni se bloquea nada. Lo anotado lleva el tipo, el motivo, los destinos y la hora: nada del proyecto.

const (
	MotivoSinDestinos = "sin_destinos" // ningún teléfono con push_token, o todos revocados
	MotivoSinFCM      = "sin_fcm"      // falta FCM_CUENTA_SERVICIO en el relay: lo arregla la persona
	MotivoFCMRechazo  = "fcm_rechazo"  // FCM rechazó todo: tokens vencidos o FCM caído

	// topeCuerpoAvisar: lo más que se lee de la respuesta de avisar; una más larga se trata como la de un relay anterior.
	topeCuerpoAvisar = 4 << 10

	archivoSinEntrega = "push.json"

	// entreRegistrosSinEntrega: el registro del servicio anota el mismo tipo y proyecto sin entrega una vez en este tiempo.
	entreRegistrosSinEntrega = time.Hour
)

// ErrSinEntrega: el relay aceptó el aviso (200) pero no llegó a ningún teléfono. Destinos es nil en un relay que no lo
// informa; Motivo, "" si no lo da.
type ErrSinEntrega struct {
	Tipo     string
	Clave    string
	Destinos *int
	Motivo   string
}

func (e ErrSinEntrega) Error() string {
	return fmt.Sprintf("push %s: %s", e.Tipo, TextoSinEntrega(e.Motivo, e.Destinos))
}

// respuestaAvisar es el cuerpo de la 200 de avisar; sin Enviados es la de un relay anterior.
type respuestaAvisar struct {
	Enviados *int   `json:"enviados"`
	Destinos *int   `json:"destinos"`
	Motivo   string `json:"motivo"`
}

// sinEntregaDe lee la respuesta de avisar: un ErrSinEntrega si dice que no llegó a nadie, nil si llegó o no lo dice.
func sinEntregaDe(datos []byte, tipo, clave string) error {
	if len(datos) > topeCuerpoAvisar {
		return nil
	}
	var r respuestaAvisar
	if json.Unmarshal(datos, &r) != nil || r.Enviados == nil || *r.Enviados > 0 {
		return nil
	}
	return ErrSinEntrega{Tipo: tipo, Clave: clave, Destinos: r.Destinos, Motivo: r.Motivo}
}

// TextoSinEntrega es la causa en palabras de la persona. Un motivo que no conoce (la lista solo crece) cae en lo que
// dicen los destinos.
func TextoSinEntrega(motivo string, destinos *int) string {
	if motivo == MotivoSinDestinos || (destinos != nil && *destinos == 0) {
		return "ningún teléfono tiene los avisos activados (sin token de push)"
	}
	if motivo == MotivoSinFCM {
		return "falta la configuración de FCM en el relay"
	}
	if motivo == MotivoFCMRechazo {
		return "FCM no entregó: tokens vencidos o FCM caído"
	}
	if destinos != nil {
		return "FCM no entregó o falta su configuración en el relay"
	}
	return "el último aviso no llegó a ningún teléfono"
}

// SinEntregaGuardada es <QDD_HOME>/remoto/push.json: el último aviso que no llegó a ningún teléfono.
type SinEntregaGuardada struct {
	Formato  int       `json:"formato"`
	Tipo     string    `json:"tipo"`
	Motivo   string    `json:"motivo,omitempty"`
	Destinos *int      `json:"destinos,omitempty"`
	En       time.Time `json:"en"`
}

// GuardarSinEntrega anota el último aviso sin entrega, sin el proyecto.
func GuardarSinEntrega(e ErrSinEntrega, en time.Time) error {
	return guardarPrivado(archivoSinEntrega, SinEntregaGuardada{Formato: 1, Tipo: e.Tipo, Motivo: e.Motivo, Destinos: e.Destinos, En: en.UTC()})
}

// LeerSinEntrega es lo que dejó GuardarSinEntrega; sin archivo, error.
func LeerSinEntrega() (SinEntregaGuardada, error) {
	var g SinEntregaGuardada
	err := leerPrivado(archivoSinEntrega, &g)
	return g, err
}

// LimpiarSinEntrega borra lo anotado: un aviso posterior sí llegó.
func LimpiarSinEntrega() error {
	return borrarPrivado(archivoSinEntrega)
}
