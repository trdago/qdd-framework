package mejoras

import (
	"fmt"
	"regexp"
	"slices"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-087 (REP-503, propuesta de la persona): quien reporta valida. Un reporte del relay no se cierra con la palabra de
// quien lo arregla: su tarea terminada (o Arreglar, si se arregló fuera del ciclo) lo deja «arreglada», y se cierra
// cuando llega, admitida por la persona, la verificación de quien lo reportó (qdd reportar --verifique). Si sigue
// fallando, vuelve a la cola el mismo reporte con la evidencia nueva: no se abre uno duplicado.

// Arreglada: el arreglo salió y espera que quien lo reportó lo verifique.
const Arreglada = "arreglada"

var reVerificacion = regexp.MustCompile(`^(Verificado|Sigue fallando) (REP-\d+) en (\S+)$`)

// VerificacionDe lee el título que arma qdd reportar --verifique (reportes.TituloVerificacion).
func VerificacionDe(titulo string) (rep string, sigueFallando bool, version string, ok bool) {
	m := reVerificacion.FindStringSubmatch(titulo)
	if m == nil {
		return "", false, "", false
	}
	return m[2], m[1] == "Sigue fallando", m[3], true
}

// Arreglar marca como arreglada en version una mejora admitida que se arregló fuera del ciclo.
func Arreglar(root, id, version, como string, now time.Time) error {
	return conCandado(root, func() error {
		cola, err := Lista(root)
		if err != nil {
			return err
		}
		i := slices.IndexFunc(cola, func(m Mejora) bool { return m.ID == id })
		if i < 0 {
			return fmt.Errorf("no hay una mejora %s en la cola", id)
		}
		m := cola[i]
		if m.Estado != EnCola && m.Estado != EnTarea {
			return fmt.Errorf("%s está %s: solo se marca arreglada una mejora admitida y abierta", id, m.Estado)
		}
		m.Estado, m.Motivo = Arreglada, recortar("arreglada en "+version+": "+como, maxDetalle)
		return escribir(m)
	})
}

// aplicarVerificacion cierra o reabre la mejora del reporte que v verifica, y deja v hecha.
func aplicarVerificacion(cola []Mejora, v Mejora, rep string, sigueFallando bool, version string) error {
	i := slices.IndexFunc(cola, func(m Mejora) bool { return m.Ref == "relay:"+rep })
	if i < 0 {
		return fmt.Errorf("%s verifica %s, que no está en la cola de mejoras", v.ID, rep)
	}
	o := cola[i]
	o.Estado, o.Motivo = Hecha, recortar("verificado por quien lo reportó en "+version+": "+v.Detalle, maxDetalle)
	if sigueFallando {
		o.Detalle, o.Motivo = recortar(o.Detalle+"\n\nSigue fallando en "+version+": "+v.Detalle, maxDetalle), ""
		o.Estado = reabierta(o)
	}
	if o.Estado == EnCola {
		o.Tarea = ""
	}
	if err := escribir(o); err != nil {
		return err
	}
	v.Estado, v.Motivo = Hecha, "verificación de "+o.ID+" ("+rep+")"
	return escribir(v)
}

// reabierta: con su tarea todavía abierta sigue en ella; si no, vuelve a la cola y el ciclo la entrega de nuevo.
func reabierta(o Mejora) string {
	if o.Estado == EnTarea {
		return EnTarea
	}
	return EnCola
}
