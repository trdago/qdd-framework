package remoto

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.4): la consola remota, de solo lectura (inv. 7). Se transmite solo mientras un teléfono la mira
// (presence mirando: <clave>) y se corta al irse (inv. 6). Junta lo nuevo y manda como mucho un mensaje cada 500 ms
// (inv. 13). Pide permiso por el latido: si el relay dice que no (cupo, interruptor, presupuesto), manda una vez
// {pausada: motivo} y para; el estado y las aprobaciones siguen (inv. 14). Todo pasa por el ocultador (inv. 16).

const (
	// CadaConsola es cada cuánto sale un mensaje de consola como mucho: 2 por segundo.
	CadaConsola = 500 * time.Millisecond
	// maxLineasMensaje es el tope de líneas por mensaje; lo demás sale en el siguiente.
	maxLineasMensaje = 60
	// cadaBuscarTranscript es cada cuánto se mira si hay un transcript más nuevo (otra sesión de Claude Code).
	cadaBuscarTranscript = 5 * time.Second
)

// MensajeConsola es el evento consola.
type MensajeConsola struct {
	Version  int            `json:"version"`
	Proyecto string         `json:"proyecto"`
	Inicio   bool           `json:"inicio,omitempty"` // la primera tanda de quien empieza a mirar: reemplaza lo que había
	Lineas   []LineaConsola `json:"lineas,omitempty"`
	Pausada  string         `json:"pausada,omitempty"`
}

// mirada es la consola de un proyecto que alguien mira.
type mirada struct {
	clave     string
	ruta      string
	lector    *lector
	pendiente []LineaConsola
	inicio    bool
	pausada   bool
	quienes   map[string]bool // los teléfonos que la miran: uno nuevo recibe el inicio y la pausa (FND-110)
	pedidosA  time.Time
	buscado   time.Time
}

type estadoConsola struct {
	miradas    map[string]*mirada
	ticker     *time.Ticker
	planSabido bool
	permitida  bool
	turno      int
}

// alCambiarEspectadores empieza a seguir lo que alguien mira ahora y suelta lo que nadie mira.
func (b *Bucle) alCambiarEspectadores(ctx context.Context) {
	c := &b.extras.consola
	quiero := map[string]map[string]bool{}
	for _, t := range b.telefonos {
		if t.Mirando == "" {
			continue
		}
		if quiero[t.Mirando] == nil {
			quiero[t.Mirando] = map[string]bool{}
		}
		quiero[t.Mirando][t.Dispositivo] = true
	}
	for clave := range c.miradas {
		if quiero[clave] == nil {
			delete(c.miradas, clave)
		}
	}
	for clave, quienes := range quiero {
		if m := c.miradas[clave]; m != nil {
			m.alCambiarQuienes(quienes)
			continue
		}
		p, ok := b.proyectoPublicado(clave)
		if !ok {
			continue
		}
		c.miradas[clave] = &mirada{clave: clave, ruta: p.Ruta, inicio: true, pedidosA: b.ahora(), quienes: quienes}
	}
	b.ajustarTicker()
}

func (b *Bucle) ajustarTicker() {
	c := &b.extras.consola
	if len(c.miradas) == 0 && c.ticker != nil {
		c.ticker.Stop()
		c.ticker, b.extras.consolaTick = nil, nil
		return
	}
	if len(c.miradas) > 0 && c.ticker == nil {
		c.ticker = time.NewTicker(CadaConsola)
		b.extras.consolaTick = c.ticker.C
	}
}

// alCambiarPlan: si la consola vuelve a estar permitida, quien mira empieza de nuevo; si se pausó, el próximo tick lo
// dice.
func (b *Bucle) alCambiarPlan() {
	c := &b.extras.consola
	antes := c.planSabido && c.permitida
	c.planSabido, c.permitida = true, b.plan.ConsolaPermitida
	if antes || !c.permitida {
		return
	}
	for _, m := range c.miradas {
		m.pausada, m.inicio, m.lector, m.pendiente = false, true, nil, nil
	}
}

// tickConsola junta lo nuevo de cada consola mirada y manda un mensaje (de un proyecto por vez, por turnos).
func (b *Bucle) tickConsola(ctx context.Context) {
	c := &b.extras.consola
	if len(c.miradas) == 0 || len(b.telefonos) == 0 || !c.planSabido {
		return
	}
	claves := make([]string, 0, len(c.miradas))
	for clave := range c.miradas {
		claves = append(claves, clave)
	}
	sort.Strings(claves)
	if !c.permitida {
		b.pausar(ctx, claves)
		return
	}
	b.consolaSeg += CadaConsola.Seconds()
	for _, clave := range claves {
		b.juntar(c.miradas[clave])
	}
	for i := range claves {
		m := c.miradas[claves[(c.turno+i)%len(claves)]]
		if len(m.pendiente) == 0 && !m.inicio {
			continue
		}
		c.turno = (c.turno + i + 1) % len(claves)
		b.mandarConsola(ctx, m)
		return
	}
}

// pausar manda {pausada: motivo} una sola vez por consola y no transmite más.
func (b *Bucle) pausar(ctx context.Context, claves []string) {
	for _, clave := range claves {
		m := b.extras.consola.miradas[clave]
		if m.pausada {
			continue
		}
		m.pausada = true
		b.broadcast(ctx, "consola", MensajeConsola{Version: VersionProtocolo, Proyecto: clave, Pausada: b.plan.MotivoPausa()})
		return
	}
}

// juntar suma lo nuevo de la conversación y de los pedidos que terminaron, ya sin secretos. Si apareció un transcript
// más nuevo (otra sesión), empieza de nuevo con su cola.
func (b *Bucle) juntar(m *mirada) {
	b.cambiarTranscript(m)
	nuevas := append(m.lector.leer(), b.pedidosTerminados(m)...)
	o := b.ocultador(m.ruta)
	for _, l := range nuevas {
		l.Texto = Recortar(o.Ocultar(l.Texto), maxTexto)
		m.pendiente = append(m.pendiente, l)
	}
}

func (b *Bucle) cambiarTranscript(m *mirada) {
	if m.lector != nil && b.ahora().Sub(m.buscado) < cadaBuscarTranscript {
		return
	}
	m.buscado = b.ahora()
	archivo := TranscriptDe(m.ruta)
	if m.lector != nil && m.lector.archivo == archivo {
		return
	}
	var cola []LineaConsola
	m.lector, cola = abrirLector(archivo)
	o := b.ocultador(m.ruta)
	m.pendiente, m.inicio = nil, true
	for _, l := range cola {
		l.Texto = Recortar(o.Ocultar(l.Texto), maxTexto)
		m.pendiente = append(m.pendiente, l)
	}
}

// pedidosTerminados son los pedidos del proyecto que terminaron desde la última vez, con el final de su salida.
func (b *Bucle) pedidosTerminados(m *mirada) []LineaConsola {
	var out []LineaConsola
	desde := m.pedidosA
	for _, p := range consola.Pedidos(m.ruta) {
		if p.Fin == nil || !p.Fin.After(desde) || (p.Estado != consola.EstadoHecho && p.Estado != consola.EstadoFallo) {
			continue
		}
		como := "terminó bien"
		if p.Estado == consola.EstadoFallo {
			como = fmt.Sprintf("falló (exit %d)", p.Codigo)
		}
		texto := fmt.Sprintf("[%s] %s · %s", p.Tecla, p.Motivo, como)
		if p.Salida != "" {
			texto += "\n" + p.Salida
		}
		out = append(out, LineaConsola{Fuente: "pedido", Rol: "pedido", Texto: texto, En: p.Fin.UTC().Format(time.RFC3339)})
		if p.Fin.After(m.pedidosA) {
			m.pedidosA = *p.Fin
		}
	}
	return out
}

func (b *Bucle) mandarConsola(ctx context.Context, m *mirada) {
	tanda := m.pendiente
	if len(tanda) > maxLineasMensaje {
		tanda = tanda[:maxLineasMensaje]
	}
	msg := MensajeConsola{Version: VersionProtocolo, Proyecto: m.clave, Inicio: m.inicio, Lineas: tanda}
	if !b.broadcast(ctx, "consola", msg) {
		return
	}
	m.pendiente, m.inicio = m.pendiente[len(tanda):], false
}

// alCambiarQuienes: si empezó a mirar un teléfono que no estaba, la consola vuelve a mandar su tanda de inicio (las
// últimas líneas, que reemplazan lo que había en todos) y, si está pausada, el motivo (FND-110).
func (m *mirada) alCambiarQuienes(quienes map[string]bool) {
	nuevo := false
	for d := range quienes {
		if !m.quienes[d] {
			nuevo = true
		}
	}
	m.quienes = quienes
	if !nuevo {
		return
	}
	m.pausada, m.inicio, m.lector, m.pendiente = false, true, nil, nil
}
