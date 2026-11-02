package remoto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): el bucle de qdd remoto correr. Publica la presence del PC; sin teléfonos no manda nada más (inv.
// 12). Con teléfonos responde sync con el estado completo y, cuando algo cambia, manda un delta con los paneles que
// cambiaron, juntando lo que pase en 300 ms (inv. 13). Da un latido por RPC cada minuto (aunque no haya teléfonos),
// con las tareas en curso y el consumo, y usa lo que responde (plan, límites, consola permitida).

const (
	Agrupar    = 300 * time.Millisecond
	CadaLatido = time.Minute
	CadaSondeo = 5 * time.Second
	// maxMensajesLatido y maxConsolaLatido son lo que el relay suma como mucho por latido.
	maxMensajesLatido = 10000
	maxConsolaLatido  = 2
)

// Transporte es el canal del relay (Canal, o uno falso en pruebas).
type Transporte interface {
	Broadcast(ctx context.Context, evento string, payload any) error
	Track(ctx context.Context, meta any) error
	Eventos() <-chan Evento
}

// RelayBucle es lo que el bucle usa del relay por RPC.
type RelayBucle interface {
	Latido(ctx context.Context, topico, firma string, enCurso, consolaMin, mensajes int) (Plan, error)
	Avisar(ctx context.Context, topico, firma, tipo, proyecto, clave string) error
}

// Telefono es un teléfono en el canal, según su presence.
type Telefono struct {
	Rol         string `json:"rol"`
	Dispositivo string `json:"dispositivo"`
	Mirando     string `json:"mirando"`
}

// Bucle es qdd remoto correr. Sus campos de estado los toca solo la goroutine de Correr.
type Bucle struct {
	QDD   string
	Cred  Credenciales
	T     Transporte
	R     RelayBucle
	Bin   string    // el qdd que ejecuta las acciones
	Log   io.Writer // una línea por suceso
	Ahora func() time.Time

	Agrupar    time.Duration
	CadaLatido time.Duration
	CadaSondeo time.Duration

	desde       time.Time
	plan        Plan
	telefonos   []Telefono
	seq         uint64
	base        bool              // los teléfonos de ahora ya tienen un estado completo
	enviados    map[string]string // clave → JSON del panel que tienen los teléfonos
	estructura  map[string]string
	vistos      map[string]map[string]bool // clave → pedidos y fichas que esperan (para el push)
	mensajes    int
	consolaSeg  float64
	sinPush     bool
	errRegistro string
	vigia       *vigia
	agrupando   *time.Timer
	latidos     chan resultadoLatido
	avisos      chan error
	extras      extrasBucle
}

type resultadoLatido struct {
	plan       Plan
	err        error
	mensajes   int
	consolaMin int
}

// PresenciaPC es la presence del PC: rol, versión de qdd y desde cuándo está encendido.
type PresenciaPC struct {
	Rol        string `json:"rol"`
	VersionQDD string `json:"version_qdd"`
	Desde      string `json:"desde"`
}

func (b *Bucle) ahora() time.Time {
	if b.Ahora == nil {
		return time.Now()
	}
	return b.Ahora()
}

func (b *Bucle) logf(formato string, args ...any) {
	if b.Log == nil {
		return
	}
	fmt.Fprintf(b.Log, "%s %s\n", time.Now().Format(time.RFC3339), Ocultar(fmt.Sprintf(formato, args...)))
}

func (b *Bucle) iniciar() {
	b.desde = b.ahora().UTC()
	b.plan = PlanGratis()
	b.enviados, b.vistos = map[string]string{}, map[string]map[string]bool{}
	b.latidos, b.avisos = make(chan resultadoLatido, 1), make(chan error, 8)
	if b.Agrupar <= 0 {
		b.Agrupar = Agrupar
	}
	if b.CadaLatido <= 0 {
		b.CadaLatido = CadaLatido
	}
	if b.CadaSondeo <= 0 {
		b.CadaSondeo = CadaSondeo
	}
	b.vigia = nuevaVigia()
	b.iniciarExtras()
}

// Correr es el bucle; vuelve con ErrCredenciales si el relay ya no reconoce la máquina (se quitó desde el teléfono).
func (b *Bucle) Correr(ctx context.Context) error {
	b.iniciar()
	go b.vigia.correr(ctx)
	b.track(ctx)
	b.latir(ctx)
	b.recalcular(ctx)
	latido := time.NewTicker(b.CadaLatido)
	defer latido.Stop()
	sondeo := time.NewTicker(b.CadaSondeo)
	defer sondeo.Stop()
	for {
		var agrupado <-chan time.Time
		if b.agrupando != nil {
			agrupado = b.agrupando.C
		}
		select {
		case <-ctx.Done():
			return nil
		case e := <-b.T.Eventos():
			b.alEvento(ctx, e)
		case r := <-b.latidos:
			if err := b.alLatido(r); err != nil {
				return err
			}
		case err := <-b.avisos:
			b.alAviso(err)
		case <-latido.C:
			b.latir(ctx)
		case <-sondeo.C:
			b.marcar()
		case <-b.vigia.Cambios():
			b.marcar()
		case <-agrupado:
			b.agrupando = nil
			b.recalcular(ctx)
		case r := <-b.extras.resultados:
			b.alResultado(ctx, r)
		case <-b.extras.consolaTick:
			b.tickConsola(ctx)
		}
	}
}

func (b *Bucle) track(ctx context.Context) {
	p := PresenciaPC{Rol: "pc", VersionQDD: b.QDD, Desde: b.desde.Format(time.RFC3339)}
	if err := b.T.Track(ctx, p); err != nil {
		b.logf("presence: %v", err)
		return
	}
	b.mensajes++
}

// marcar abre la ventana de 300 ms: lo que cambie dentro sale junto.
func (b *Bucle) marcar() {
	if b.agrupando != nil {
		return
	}
	b.agrupando = time.NewTimer(b.Agrupar)
}

func (b *Bucle) alEvento(ctx context.Context, e Evento) {
	if e.Tipo == TipoConectado {
		b.logf("conectado al relay")
		b.track(ctx)
		return
	}
	if e.Tipo == TipoDesconectado {
		b.logf("desconectado del relay: %s", e.Motivo)
		b.base = false
		return
	}
	if e.Tipo == TipoPresencia {
		b.alPresencia(ctx, e.Presencias)
		return
	}
	if e.Tipo != TipoBroadcast {
		return
	}
	if e.Evento == "sync" {
		b.mandarEstado(ctx, b.armar())
		b.reenviarSolicitudes(ctx)
		return
	}
	if e.Evento == "accion" {
		b.alAccion(ctx, e.Payload)
	}
}

// Telefonos son los teléfonos de una presence (rol movil).
func Telefonos(presencias map[string][]json.RawMessage) []Telefono {
	var out []Telefono
	claves := make([]string, 0, len(presencias))
	for k := range presencias {
		claves = append(claves, k)
	}
	sort.Strings(claves)
	for _, k := range claves {
		for _, m := range presencias[k] {
			var t Telefono
			if json.Unmarshal(m, &t) == nil && t.Rol == "movil" {
				out = append(out, t)
			}
		}
	}
	return out
}

func (b *Bucle) alPresencia(ctx context.Context, presencias map[string][]json.RawMessage) {
	antes := len(b.telefonos)
	b.telefonos = Telefonos(presencias)
	b.alCambiarEspectadores(ctx)
	if len(b.telefonos) == 0 {
		b.base, b.enviados, b.estructura = false, map[string]string{}, nil
		if antes > 0 {
			b.logf("sin teléfonos: solo presence")
		}
		return
	}
	if antes == 0 {
		b.logf("%d teléfono(s) en el canal", len(b.telefonos))
	}
	if !b.base {
		b.mandarEstado(ctx, b.armar())
	}
}

func (b *Bucle) armar() Estado {
	reg, err := registro.Leer()
	if err != nil {
		if err.Error() != b.errRegistro {
			b.logf("registro de proyectos: %v", err)
		}
		b.errRegistro = err.Error()
		return Estado{Version: VersionProtocolo, QDD: b.QDD, Maquina: EstaMaquina(), En: b.ahora().UTC(), Proyectos: []ProyectoEstado{}}
	}
	b.errRegistro = ""
	b.vigia.vigilar(carpetasDe(reg))
	return Armar(b.QDD, reg, b.ahora(), b.plan.LimiteProyectos())
}

// recalcular arma el estado, avisa por push lo nuevo y, si hay teléfonos con un estado base, les manda lo que cambió.
func (b *Bucle) recalcular(ctx context.Context) {
	e := b.armar()
	b.avisarNuevos(ctx, e)
	b.seguirSolicitudes(ctx, e)
	if len(b.telefonos) == 0 || !b.base {
		return
	}
	b.mandarCambios(ctx, e)
}

func (b *Bucle) broadcast(ctx context.Context, evento string, payload any) bool {
	if err := b.T.Broadcast(ctx, evento, payload); err != nil {
		b.logf("no se pudo mandar %s: %v", evento, err)
		return false
	}
	b.mensajes++
	return true
}

// mandarEstado manda el estado completo con el siguiente seq: es la base de los deltas.
func (b *Bucle) mandarEstado(ctx context.Context, e Estado) {
	if len(b.telefonos) == 0 {
		return
	}
	b.seq++
	e.Seq = b.seq
	if !b.broadcast(ctx, "estado", e) {
		b.base = false
		return
	}
	b.base = true
	b.enviados, b.estructura = paneles(e), estructuraDe(e)
}

// paneles es el JSON de cada panel, para saber cuál cambió.
func paneles(e Estado) map[string]string {
	out := map[string]string{}
	for _, p := range e.Proyectos {
		b, _ := json.Marshal(p.Panel)
		out[p.Clave] = string(b)
	}
	return out
}

// estructuraDe es lo que un delta no puede decir de cada proyecto (nombre, ruta y error) y cuántos quedaron fuera
// del plan (clave "").
func estructuraDe(e Estado) map[string]string {
	out := map[string]string{"": fmt.Sprint(e.FueraDelPlan)}
	for _, p := range e.Proyectos {
		out[p.Clave] = p.Nombre + "\x00" + p.Ruta + "\x00" + p.Error
	}
	return out
}

// pideEstado: cambió algo que un delta no dice (un proyecto nuevo, un nombre, un error, los de fuera del plan). Que
// un proyecto salga sí lo dice (null).
func pideEstado(antes, ahora map[string]string) bool {
	for clave, v := range ahora {
		if antes[clave] != v {
			return true
		}
	}
	return false
}

// Delta es el evento delta: los paneles que cambiaron (null: el proyecto salió).
type Delta struct {
	Version   int                              `json:"version"`
	Seq       uint64                           `json:"seq"`
	Proyectos map[string]*consola.DatosConsola `json:"proyectos"`
}

// mandarCambios manda un delta con los paneles que cambiaron, o el estado completo si cambió lo que un delta no dice.
func (b *Bucle) mandarCambios(ctx context.Context, e Estado) {
	if pideEstado(b.estructura, estructuraDe(e)) {
		b.mandarEstado(ctx, e)
		return
	}
	ahora := paneles(e)
	d := Delta{Version: VersionProtocolo, Proyectos: map[string]*consola.DatosConsola{}}
	for clave := range b.enviados {
		if _, ok := ahora[clave]; !ok {
			d.Proyectos[clave] = nil
		}
	}
	for _, p := range e.Proyectos {
		if b.enviados[p.Clave] != ahora[p.Clave] {
			d.Proyectos[p.Clave] = p.Panel
		}
	}
	if len(d.Proyectos) == 0 {
		return
	}
	b.seq++
	d.Seq = b.seq
	if !b.broadcast(ctx, "delta", d) {
		b.base = false
		return
	}
	b.enviados, b.estructura = ahora, estructuraDe(e)
}

// esperan son los pedidos pendientes y las fichas en borrador de un panel: lo que espera a la persona.
func esperan(p ProyectoEstado) map[string]string {
	out := map[string]string{}
	if p.Panel == nil {
		return out
	}
	for _, a := range p.Panel.Acciones {
		if a.Tipo == consola.TipoPedido && a.Estado == consola.EstadoPendiente {
			out[a.ID] = AvisoPedido
		}
		if a.Tipo == consola.TipoAprobar {
			out[a.ID] = AvisoFicha
		}
	}
	return out
}

// avisarNuevos pide un push cuando aparece un pedido o una ficha nueva en un panel. La primera vez que se ve un
// proyecto es la base: lo que ya esperaba no se avisa.
func (b *Bucle) avisarNuevos(ctx context.Context, e Estado) {
	for _, p := range e.Proyectos {
		ahora := esperan(p)
		antes, conocido := b.vistos[p.Clave]
		b.vistos[p.Clave] = map[string]bool{}
		for id := range ahora {
			b.vistos[p.Clave][id] = true
		}
		if !conocido {
			continue
		}
		tipos := map[string]bool{}
		for id, tipo := range ahora {
			if !antes[id] {
				tipos[tipo] = true
			}
		}
		for _, tipo := range []string{AvisoPedido, AvisoFicha} {
			if tipos[tipo] {
				b.avisar(ctx, tipo, p.Nombre, p.Clave)
			}
		}
	}
}

// avisar pide el push sin bloquear el bucle: es de mejor esfuerzo (inv. 17: solo tipo y proyecto).
func (b *Bucle) avisar(ctx context.Context, tipo, nombre, clave string) {
	go func() {
		c, cancelar := context.WithTimeout(ctx, 15*time.Second)
		defer cancelar()
		err := b.R.Avisar(c, b.Cred.Topico, b.Cred.Firma, tipo, nombre, clave)
		select {
		case b.avisos <- err:
		default:
		}
	}()
}

func (b *Bucle) alAviso(err error) {
	if err == nil {
		b.sinPush = false
		return
	}
	if errors.Is(err, ErrNoDisponible) && b.sinPush {
		return
	}
	if errors.Is(err, ErrNoDisponible) {
		b.sinPush = true
		b.logf("push no disponible en el relay (la Edge Function avisar no está desplegada)")
		return
	}
	b.logf("push: %v", err)
}

// latir da el latido en segundo plano con lo consumido desde el anterior.
func (b *Bucle) latir(ctx context.Context) {
	mensajes := min(b.mensajes, maxMensajesLatido)
	consolaMin := min(int(b.consolaSeg/60), maxConsolaLatido)
	b.mensajes -= mensajes
	b.consolaSeg -= float64(consolaMin) * 60
	enCurso := EnCurso(b.ahora())
	go func() {
		c, cancelar := context.WithTimeout(ctx, 20*time.Second)
		defer cancelar()
		p, err := b.R.Latido(c, b.Cred.Topico, b.Cred.Firma, enCurso, consolaMin, mensajes)
		select {
		case b.latidos <- resultadoLatido{plan: p, err: err, mensajes: mensajes, consolaMin: consolaMin}:
		case <-ctx.Done():
		}
	}()
}

// alLatido usa lo que respondió el relay; si no lo recibió, devuelve lo consumido para el próximo.
func (b *Bucle) alLatido(r resultadoLatido) error {
	if errors.Is(r.err, ErrCredenciales) {
		b.logf("%v", r.err)
		return ErrCredenciales
	}
	if r.err != nil {
		b.mensajes += r.mensajes
		b.consolaSeg += float64(r.consolaMin) * 60
		b.logf("latido: %v", r.err)
		return nil
	}
	limiteAntes := b.plan.LimiteProyectos()
	b.plan = r.plan
	b.alCambiarPlan()
	if b.plan.LimiteProyectos() != limiteAntes {
		b.marcar()
	}
	return nil
}

// EnCurso cuenta lo que corre ahora en todos los proyectos del registro: tareas «● running» y pedidos ejecutando. Con
// eso el relay avisa si el PC deja de latir con trabajo en curso.
func EnCurso(now time.Time) int {
	reg, err := registro.Leer()
	if err != nil {
		return 0
	}
	n := 0
	for _, p := range reg.Proyectos {
		n += len(estado.EnCurso(estado.LeerActividad(p.Ruta), now))
		for _, pd := range consola.PedidosA(p.Ruta, now) {
			if pd.Estado == consola.EstadoEjecutando {
				n++
			}
		}
	}
	return n
}
