package invitaciones

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'

// El ciclo lo pide el panel cada pocos segundos mientras hay invitaciones vivas. Cada propuesta válida es un pedido de
// la persona en el panel (aprobarla); descartar el pedido la rechaza. Lo que no cumple el esquema o el rol se rechaza
// solo, con su motivo. Nada entra al flujo sin el «sí» de la persona, ni con el modo goal: el pedido es DePersona.

// MaxAbiertas es cuántas propuestas esperan a la vez en el panel; las demás entran a medida que se resuelven.
const MaxAbiertas = 3

// Relay es lo que el ciclo necesita del relay (Cliente lo cumple).
type Relay interface {
	Propuestas(ctx context.Context, desde string, ahora time.Time) ([]Propuesta, error)
	Resolver(ctx context.Context, propuesta, estado, motivo string, ahora time.Time) error
	SubirFoto(ctx context.Context, flujo string, foto json.RawMessage, ahora time.Time) (int, error)
}

// Panel deja pedidos de la persona y dice en qué quedaron: "" si ya no está (la persona lo descartó).
type Panel interface {
	Pedir(comando, motivo string) (string, error)
	Estado(id, comando string) string
}

// Entorno es todo lo que el ciclo toca. Foto arma la foto limpia del flujo, la que ve el invitado.
type Entorno struct {
	Root  string
	Relay Relay
	Panel Panel
	Foto  func(flujo string) (json.RawMessage, error)
	Ahora time.Time
}

// Resumen es lo que hizo un ciclo.
type Resumen struct {
	Nuevas     int `json:"nuevas"`
	Pedidas    int `json:"pedidas"`
	Rechazadas int `json:"rechazadas"`
	Esperan    int `json:"esperan"`
}

// ComandoAprobar es el comando del pedido de una propuesta.
func ComandoAprobar(id string) string {
	return "qdd flujo propuestas aprobar " + id
}

// Ciclo recibe lo nuevo, sigue los pedidos, pide los que caben y le avisa al relay lo resuelto.
func Ciclo(ctx context.Context, e Entorno) (Resumen, error) {
	var res Resumen
	err := conLocal(e.Root, func(l *Local) error {
		ahora := e.Ahora
		l.Ultimo = &ahora
		var errs []error
		if l.Vivas("", e.Ahora) {
			errs = append(errs, recibir(ctx, e, l, &res))
		}
		seguirPedidos(e.Panel, l)
		errs = append(errs, pedir(e, l, &res))
		errs = append(errs, avisar(ctx, e, l))
		return errors.Join(errs...)
	})
	return res, err
}

// TocaCiclo dice si pasó la pausa desde el último ciclo y hay algo que hacer; así el panel puede llamar seguido.
func TocaCiclo(root string, ahora time.Time, pausa time.Duration) bool {
	l, err := Leer(root)
	if err != nil {
		return false
	}
	if !l.Vivas("", ahora) && !l.porAvisar() && !hayPorPedir(l) {
		return false
	}
	return l.Ultimo == nil || ahora.Sub(*l.Ultimo) >= pausa
}

func hayPorPedir(l Local) bool {
	for _, v := range l.Propuestas {
		if v.Situacion == PorPedir || v.Situacion == Pedida {
			return true
		}
	}
	return false
}

func recibir(ctx context.Context, e Entorno, l *Local, res *Resumen) error {
	nuevas, err := e.Relay.Propuestas(ctx, l.Desde, e.Ahora)
	for _, p := range nuevas {
		l.Desde = mayor(l.Desde, p.ID)
		inv, nuestra := l.invitacion(p.Invitacion)
		if !nuestra || l.vista(p.ID) != nil || p.Estado != Pendiente {
			continue
		}
		res.Nuevas++
		v := Vista{Propuesta: p, Situacion: PorPedir}
		if problema := revisar(p, inv); problema != nil {
			v.Situacion, v.Razon = Rechazada, recortar("no se pudo integrar: "+problema.Error(), MaxMotivo)
			res.Rechazadas++
		}
		l.Propuestas = append(l.Propuestas, v)
	}
	return err
}

// revisar aplica el esquema del PC, que es quien decide: la operación, su flujo, el rol de la invitación y el autor.
func revisar(p Propuesta, inv Invitacion) error {
	o, err := LeerOperacion(p.Operacion)
	if err != nil {
		return err
	}
	if o.Flujo != inv.Flujo {
		return fmt.Errorf("la invitación es del flujo %q", inv.Flujo)
	}
	if err := o.PermitidaA(inv.Rol); err != nil {
		return err
	}
	_, err = flujo.AutorDeInvitado(p.Autor)
	return err
}

// seguirPedidos mira en qué quedó cada pedido: descartado es rechazada; retirado vuelve a pedirse.
func seguirPedidos(panel Panel, l *Local) {
	for i := range l.Propuestas {
		v := &l.Propuestas[i]
		if v.Situacion != Pedida {
			continue
		}
		estado := panel.Estado(v.Pedido, ComandoAprobar(v.ID))
		if estado == "" {
			v.Situacion, v.Razon, v.Avisada = Rechazada, "", false
			continue
		}
		if estado == "retirado" {
			v.Situacion, v.Pedido = PorPedir, ""
		}
	}
}

func pedir(e Entorno, l *Local, res *Resumen) error {
	abiertas := 0
	for _, v := range l.Propuestas {
		if v.Situacion == Pedida {
			abiertas++
		}
	}
	for i := range l.Propuestas {
		v := &l.Propuestas[i]
		if v.Situacion != PorPedir {
			continue
		}
		if abiertas >= MaxAbiertas {
			res.Esperan++
			continue
		}
		id, err := e.Panel.Pedir(ComandoAprobar(v.ID), encabezado(*v)+describir(e.Root, l, *v))
		if err != nil {
			return fmt.Errorf("no se dejó en el panel %s: %w", v.ID, err)
		}
		v.Situacion, v.Pedido = Pedida, id
		abiertas++
		res.Pedidas++
	}
	return nil
}

// describir cuenta la propuesta con los nombres del flujo de hoy y dice si se hizo sobre una revisión anterior.
func describir(root string, l *Local, v Vista) string {
	o, err := LeerOperacion(v.Operacion)
	if err != nil {
		return v.ID
	}
	f, _ := flujo.Cargar(root, o.Flujo)
	texto := o.Describir(f)
	if v.Base != "" && v.Base != "r"+strconv.Itoa(f.Revision) {
		texto += " (sobre " + v.Base + ")"
	}
	return texto
}

// avisar le cuenta al relay cada resolución pendiente y le sube las fotos nuevas. Una que el relay ya tenía resuelta
// al revés, o que ya no existe, no se reintenta.
func avisar(ctx context.Context, e Entorno, l *Local) error {
	for i := range l.Propuestas {
		v := &l.Propuestas[i]
		if !resuelta(*v) || v.Avisada {
			continue
		}
		if delAsistente(*v) {
			v.Avisada = true
			continue
		}
		err := e.Relay.Resolver(ctx, v.ID, v.Situacion, v.Razon, e.Ahora)
		if err != nil && !errors.Is(err, ErrYaResuelta) && !errors.Is(err, ErrAjena) {
			return err
		}
		v.Avisada = true
	}
	for len(l.FotosPorSubir) > 0 {
		id := l.FotosPorSubir[0]
		foto, err := e.Foto(id)
		if err != nil {
			return err
		}
		if _, err := e.Relay.SubirFoto(ctx, id, foto, e.Ahora); err != nil {
			return err
		}
		l.FotosPorSubir = l.FotosPorSubir[1:]
	}
	return nil
}

// Aprobar aplica la propuesta a nombre de quien la hizo. La aplicación no depende de la red: si el relay no responde,
// se entera en el próximo ciclo. Solo sube la foto nueva si alguien mira ese flujo con una invitación viva.
func Aprobar(ctx context.Context, e Entorno, id string) (string, error) {
	var linea string
	err := conLocal(e.Root, func(l *Local) error {
		v, err := porResolver(l, id)
		if err != nil {
			return err
		}
		o, err := LeerOperacion(v.Operacion)
		if err != nil {
			return err
		}
		linea, err = aplicar(e, l, v, o)
		if err != nil {
			return fmt.Errorf("%w: si ya no se puede integrar, recházala con 'qdd flujo propuestas rechazar %s'", err, id)
		}
		v.Situacion, v.Razon, v.Avisada = Aprobada, "", false
		if l.Vivas(o.Flujo, e.Ahora) {
			l.fotoPorSubir(o.Flujo)
		}
		if err := avisar(ctx, e, l); err != nil {
			linea += " (el invitado se entera cuando vuelva la conexión con el relay)"
		}
		return nil
	})
	return linea, err
}

// aplicar la integra: la del asistente con su origen, la de un invitado con el rol que este PC anotó al invitarlo.
func aplicar(e Entorno, l *Local, v *Vista, o flujo.Operacion) (string, error) {
	if delAsistente(*v) {
		return o.AplicarDeAsistente(e.Root, v.Origen, e.Ahora)
	}
	inv, ok := l.invitacion(v.Invitacion)
	if !ok {
		return "", fmt.Errorf("la invitación de %s ya no está en este proyecto", v.ID)
	}
	return o.AplicarDeInvitado(e.Root, inv.Flujo, v.Autor, inv.Rol, e.Ahora)
}

// Rechazar la deja fuera sin tocar el flujo; el invitado ve el motivo, si lo hay.
func Rechazar(ctx context.Context, e Entorno, id, motivo string) error {
	motivo = strings.TrimSpace(motivo)
	if len([]rune(motivo)) > MaxMotivo {
		return fmt.Errorf("el motivo tiene más de %d caracteres", MaxMotivo)
	}
	return conLocal(e.Root, func(l *Local) error {
		v, err := porResolver(l, id)
		if err != nil {
			return err
		}
		v.Situacion, v.Razon, v.Avisada = Rechazada, motivo, false
		_ = avisar(ctx, e, l)
		return nil
	})
}

func porResolver(l *Local, id string) (*Vista, error) {
	v := l.vista(id)
	if v == nil {
		return nil, fmt.Errorf("no hay una propuesta %s en este proyecto: 'qdd flujo propuestas' dice cuáles hay", id)
	}
	if resuelta(*v) {
		return nil, fmt.Errorf("la propuesta %s ya está %s", id, v.Situacion)
	}
	return v, nil
}

// numeroDe es n en «PRO-n»; 0 si no lo es.
func numeroDe(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "PRO-"))
	if err != nil || !strings.HasPrefix(id, "PRO-") {
		return 0
	}
	return n
}

func mayor(a, b string) string {
	if numeroDe(b) > numeroDe(a) {
		return b
	}
	return a
}

func recortar(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// LeerOperacion compacta el JSON antes de leerlo: el relay o el archivo local pueden traerlo con sangría.
func LeerOperacion(raw json.RawMessage) (flujo.Operacion, error) {
	var compacto bytes.Buffer
	if err := json.Compact(&compacto, raw); err != nil {
		return flujo.Operacion{}, fmt.Errorf("la operación no es JSON: %w", err)
	}
	return flujo.LeerOperacion(compacto.String())
}
