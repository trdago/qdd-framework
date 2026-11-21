package invitaciones

import (
	"context"
	"time"
)

// Rule: Prohibido usar 'else'
// Lo que qdd doctor revisa de los flujos compartidos (RFC 0025, fase 6), sin red: lee solo el archivo local. FLC-01
// una invitación viva cuyo flujo ya no existe, FLC-02 propuestas sin resolver con más de PropuestaVieja, FLC-03 fotos
// que el relay todavía no tiene. FLC-04 es el historial del flujo (FLU-02, pkg/flujo); FLC-05 y FLC-06 llegan con el
// audio (fase 5). Reparar nunca toca una propuesta.

// PropuestaVieja: una propuesta sin resolver que lleva más que esto se nombra en doctor.
const PropuestaVieja = 7 * 24 * time.Hour

// Revision es lo que encontró doctor. Compartido dice si el proyecto compartió algo alguna vez: sin eso no hay grupo.
type Revision struct {
	Compartido    bool
	Huerfanas     []Invitacion
	Viejas        []Vista
	FotosPorSubir []string
}

// Revocador es la parte del relay que revoca una invitación.
type Revocador interface {
	Revocar(ctx context.Context, invitacion string, ahora time.Time) error
}

// Revisar lee el archivo local; uno ilegible o ausente es nada compartido.
func Revisar(root string, existe func(flujo string) bool, ahora time.Time) Revision {
	l, err := Leer(root)
	if err != nil {
		return Revision{}
	}
	r := Revision{
		Compartido:    len(l.Invitaciones) > 0 || len(l.Propuestas) > 0 || len(l.FotosPorSubir) > 0,
		Huerfanas:     huerfanas(l, existe, ahora),
		FotosPorSubir: l.FotosPorSubir,
	}
	for _, v := range l.Propuestas {
		llego, err := time.Parse(time.RFC3339, v.Llego)
		if resuelta(v) || err != nil || ahora.Sub(llego) <= PropuestaVieja {
			continue
		}
		r.Viejas = append(r.Viejas, v)
	}
	return r
}

func huerfanas(l Local, existe func(string) bool, ahora time.Time) []Invitacion {
	var out []Invitacion
	for _, i := range l.Invitaciones {
		if !i.Viva(ahora) || existe(i.Flujo) {
			continue
		}
		out = append(out, i)
	}
	return out
}

// RevocarHuerfanas revoca en el relay cada invitación viva de un flujo que ya no existe y la anota revocada; sin red,
// se detiene y la deja como estaba (sigue viva allá). Devuelve las que revocó.
func RevocarHuerfanas(ctx context.Context, root string, relay Revocador, existe func(string) bool, ahora time.Time) ([]string, error) {
	l, err := Leer(root)
	if err != nil {
		return nil, err
	}
	var hechas []string
	for _, i := range huerfanas(l, existe, ahora) {
		if err := relay.Revocar(ctx, i.ID, ahora); err != nil {
			return hechas, err
		}
		if err := MarcarRevocada(root, i.ID); err != nil {
			return hechas, err
		}
		hechas = append(hechas, i.ID)
	}
	return hechas, nil
}

// SubirFotosPendientes sube las fotos que el relay todavía no tiene, sin tocar las propuestas.
func SubirFotosPendientes(ctx context.Context, e Entorno) error {
	return conLocal(e.Root, func(l *Local) error { return subirFotos(ctx, e, l) })
}
