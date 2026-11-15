package remoto

import (
	"context"
	"encoding/json"
	"regexp"
	"slices"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// Logrado bajo pedido (contrato con qdd-movile-83, 2026-11-15; opción elegida por el modo goal de la persona con la
// recomendada): lo logrado era lo que más pesaba del estado (FND-209). Un teléfono que declara la capacidad
// logrado_bajo_pedido en su presence lo pide por proyecto al abrirlo, y el estado deja de llevarlo solo cuando todos los
// teléfonos del canal la declaran. Es opcional por capacidad, como goal: no sube la versión del protocolo.

const (
	CapacidadLogradoBajoPedido = "logrado_bajo_pedido"
	EventoPedirLogrado         = "pedir_logrado"
	EventoLogrado              = "logrado"

	ErrorLogradoProyecto = "proyecto"
	ErrorLogradoPlan     = "plan"

	// TopeRespuestaLogrado: una respuesta más grande quita lo más viejo hasta caber y dice recortado.
	TopeRespuestaLogrado = 64 * 1024
	// EntreLogrados: como mucho una respuesta calculada por proyecto en este tiempo; dentro, la misma.
	EntreLogrados = 5 * time.Second
)

var (
	reClaveProyecto = regexp.MustCompile(`^[a-z0-9-]{1,80}$`)
	reSolicitud     = regexp.MustCompile(`^[A-Za-z0-9-]{1,64}$`)
)

// PedidoLogrado es lo que manda el teléfono.
type PedidoLogrado struct {
	Version   int    `json:"version"`
	Proyecto  string `json:"proyecto"`
	Solicitud string `json:"solicitud"`
}

// RespuestaLogrado es lo que contesta el PC: las entradas con el contrato de siempre (≤ 60, 14 días, la más nueva
// primero, salida ≤ 2000 y sin secretos), o [] con el error.
type RespuestaLogrado struct {
	Version   int             `json:"version"`
	Proyecto  string          `json:"proyecto"`
	Solicitud string          `json:"solicitud"`
	Entradas  []consola.Logro `json:"entradas"`
	Recortado bool            `json:"recortado,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type respuestaGuardada struct {
	en        time.Time
	respuesta RespuestaLogrado
}

// todosPidenLogrado: hay teléfonos y todos declaran la capacidad.
func (b *Bucle) todosPidenLogrado() bool {
	if len(b.telefonos) == 0 {
		return false
	}
	for _, t := range b.telefonos {
		if !slices.Contains(t.Capacidades, CapacidadLogradoBajoPedido) {
			return false
		}
	}
	return true
}

// aplicarLogradoBajoPedido anuncia en cada panel que lo logrado se pide, y lo saca si todos los teléfonos lo piden.
func (b *Bucle) aplicarLogradoBajoPedido(e *Estado) {
	b.sinLogrado = b.todosPidenLogrado()
	for i := range e.Proyectos {
		panel := e.Proyectos[i].Panel
		if panel == nil {
			continue
		}
		panel.LogradoBajoPedido = true
		if b.sinLogrado {
			panel.Logrado = nil
		}
	}
}

// alPedirLogrado contesta solo con un teléfono mirando (inv. 12) y a un pedido válido; lo inválido se ignora. Cada
// pedido deja una línea en el registro con su desenlace (MEJ-080): ignorarlo en silencio no dejaba saber qué pasaba.
func (b *Bucle) alPedirLogrado(ctx context.Context, payload json.RawMessage) {
	var p PedidoLogrado
	if err := json.Unmarshal(payload, &p); err != nil {
		b.logf("pedir_logrado ignorado: ilegible (%v)", err)
		return
	}
	if !reClaveProyecto.MatchString(p.Proyecto) {
		b.logf("pedir_logrado ignorado: clave de proyecto inválida %q", Recortar(p.Proyecto, 80))
		return
	}
	if !reSolicitud.MatchString(p.Solicitud) {
		b.logf("pedir_logrado %s: ignorado, solicitud inválida", p.Proyecto)
		return
	}
	if len(b.telefonos) == 0 {
		b.logf("pedir_logrado %s (%s): ignorado, no hay teléfonos en el canal", p.Proyecto, p.Solicitud)
		return
	}
	now := b.ahora()
	guardada, ok := b.logrados[p.Proyecto]
	if !ok || now.Sub(guardada.en) >= EntreLogrados {
		guardada = respuestaGuardada{en: now, respuesta: b.logradoDe(p.Proyecto, now)}
		if b.logrados == nil {
			b.logrados = map[string]respuestaGuardada{}
		}
		b.logrados[p.Proyecto] = guardada
	}
	r := guardada.respuesta
	r.Solicitud = p.Solicitud
	if !b.broadcast(ctx, EventoLogrado, r) {
		return
	}
	b.logf("pedir_logrado %s (%s): respondido con %d entradas%s", p.Proyecto, p.Solicitud, len(r.Entradas), desenlaceLogrado(r))
}

// desenlaceLogrado es lo que agrega la línea del registro cuando la respuesta no es la completa.
func desenlaceLogrado(r RespuestaLogrado) string {
	if r.Error != "" {
		return ", error " + r.Error
	}
	if r.Recortado {
		return ", recortado"
	}
	return ""
}

// logradoDe arma la respuesta de un proyecto: desconocido o fuera del plan, [] con su error.
func (b *Bucle) logradoDe(clave string, now time.Time) RespuestaLogrado {
	r := RespuestaLogrado{Version: VersionProtocolo, Proyecto: clave, Entradas: []consola.Logro{}}
	reg, err := registro.Leer()
	if err != nil {
		r.Error = ErrorLogradoProyecto
		return r
	}
	i := slices.IndexFunc(reg.Proyectos, func(p registro.Proyecto) bool { return p.Clave == clave })
	if i < 0 {
		r.Error = ErrorLogradoProyecto
		return r
	}
	if !slices.ContainsFunc(Publicables(reg, b.plan.LimiteProyectos()), func(p registro.Proyecto) bool { return p.Clave == clave }) {
		r.Error = ErrorLogradoPlan
		return r
	}
	root := reg.Proyectos[i].Ruta
	r.Entradas = ocultarLogrado(OcultadorDe(root), consola.Logrado(root, now), func(s string) string { return s })
	return caberEnTope(r)
}

// caberEnTope quita lo más viejo (lo último) hasta que la respuesta quepa en TopeRespuestaLogrado.
func caberEnTope(r RespuestaLogrado) RespuestaLogrado {
	for len(r.Entradas) > 0 {
		datos, err := json.Marshal(r)
		if err != nil || len(datos)+solicitudMaxima <= TopeRespuestaLogrado {
			return r
		}
		r.Entradas, r.Recortado = r.Entradas[:len(r.Entradas)-1], true
	}
	return r
}

// solicitudMaxima: la respuesta se mide sin solicitud (se cachea para varias), así que se le reserva su largo.
const solicitudMaxima = 64
