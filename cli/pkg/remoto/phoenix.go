package remoto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): cliente de Supabase Realtime. Habla el protocolo de canales de Phoenix sobre WebSocket: v1 (un
// objeto JSON por mensaje, el que usa el PC) y v2 (un arreglo [join_ref, ref, topic, event, payload], que también se
// entiende al leer). Un solo canal, privado, con broadcast y presence; latido cada 25 s; si la conexión se cae, vuelve
// a conectar con espera exponencial y jitter, y al volver repite el join y su presence.

const (
	// CadaLatidoWS es cada cuánto se manda el heartbeat de Phoenix; sin respuesta antes del siguiente, la conexión
	// está muerta.
	CadaLatidoWS  = 25 * time.Second
	esperaMinima  = time.Second
	esperaMaxima  = 30 * time.Second
	limiteLectura = 4 << 20

	TipoBroadcast    = "broadcast"
	TipoPresencia    = "presencia"
	TipoConectado    = "conectado"
	TipoDesconectado = "desconectado"
)

// Evento es lo que el canal le entrega al bucle: un broadcast (Evento y Payload), un cambio de presence (Presencias
// trae el estado completo), o que se conectó o se cayó (Motivo).
type Evento struct {
	Tipo       string
	Evento     string
	Payload    json.RawMessage
	Presencias map[string][]json.RawMessage
	Motivo     string
}

// ConfigCanal es lo que hace falta para unirse: la URL del relay, la clave publicable (apikey y access_token: el PC es
// anon), el tópico sin el prefijo «realtime:» y la clave de presence.
type ConfigCanal struct {
	Relay       string
	Clave       string
	Topico      string
	PresenceKey string
	Vsn         string        // "1.0.0" (por defecto) o "2.0.0"
	Latido      time.Duration // 0: CadaLatidoWS
	EsperaMax   time.Duration // 0: esperaMaxima
}

// Canal es la conexión del PC a su canal. Correr la mantiene viva; Broadcast y Track se pueden llamar desde cualquier
// goroutine.
type Canal struct {
	cfg     ConfigCanal
	eventos chan Evento

	mu        sync.Mutex
	conn      *websocket.Conn
	ref       int
	joinRef   string
	unido     bool
	track     json.RawMessage
	presencia map[string]map[string]json.RawMessage
}

// NuevoCanal prepara el canal sin conectarlo.
func NuevoCanal(cfg ConfigCanal) *Canal {
	if cfg.Vsn == "" {
		cfg.Vsn = "1.0.0"
	}
	if cfg.Latido <= 0 {
		cfg.Latido = CadaLatidoWS
	}
	if cfg.EsperaMax <= 0 {
		cfg.EsperaMax = esperaMaxima
	}
	return &Canal{cfg: cfg, eventos: make(chan Evento, 256), presencia: map[string]map[string]json.RawMessage{}}
}

// Eventos es por donde llegan los broadcasts, los cambios de presence y las caídas.
func (c *Canal) Eventos() <-chan Evento { return c.eventos }

// URLSocket es wss://<ref>.supabase.co/realtime/v1/websocket?apikey=<clave>&vsn=<vsn> (ws:// si el relay es http://).
func URLSocket(relay, clave, vsn string) (string, error) {
	u, err := url.Parse(strings.TrimRight(relay, "/"))
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("la URL del relay no es válida: %q", relay)
	}
	u.Scheme = map[string]string{"https": "wss", "http": "ws", "wss": "wss", "ws": "ws"}[u.Scheme]
	if u.Scheme == "" {
		return "", fmt.Errorf("la URL del relay tiene que ser https:// (o http:// en pruebas): %q", relay)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/realtime/v1/websocket"
	q := url.Values{}
	q.Set("apikey", clave)
	q.Set("vsn", vsn)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// Correr conecta y se queda conectado hasta que ctx termine, reconectando con espera exponencial y jitter.
func (c *Canal) Correr(ctx context.Context) {
	espera := esperaMinima
	for ctx.Err() == nil {
		inicio := time.Now()
		err := c.sesion(ctx)
		c.emitir(Evento{Tipo: TipoDesconectado, Motivo: motivo(err)})
		if ctx.Err() != nil {
			return
		}
		if time.Since(inicio) > 2*c.cfg.EsperaMax {
			espera = esperaMinima
		}
		dormir(ctx, conJitter(espera))
		espera = min(espera*2, c.cfg.EsperaMax)
	}
}

func motivo(err error) string {
	if err == nil {
		return "cerrada"
	}
	return err.Error()
}

// conJitter reparte la espera entre la mitad y el total, para que muchos PCs no vuelvan todos a la vez.
func conJitter(d time.Duration) time.Duration {
	mitad := d / 2
	return mitad + time.Duration(rand.Int64N(int64(mitad)+1))
}

func dormir(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// sesion es una conexión: dial, join, presence y lectura hasta que se caiga.
func (c *Canal) sesion(ctx context.Context) error {
	dir, err := URLSocket(c.cfg.Relay, c.cfg.Clave, c.cfg.Vsn)
	if err != nil {
		return err
	}
	ctxDial, cancelar := context.WithTimeout(ctx, 15*time.Second)
	conn, _, err := websocket.Dial(ctxDial, dir, nil)
	cancelar()
	if err != nil {
		return fmt.Errorf("no se pudo conectar al relay: %w", err)
	}
	conn.SetReadLimit(limiteLectura)
	ctxSesion, terminar := context.WithCancel(ctx)
	defer terminar()
	defer conn.CloseNow()
	c.mu.Lock()
	c.conn, c.unido, c.presencia = conn, false, map[string]map[string]json.RawMessage{}
	c.mu.Unlock()
	defer c.soltar(conn)

	respuestas := make(chan Mensaje, 16)
	errLectura := make(chan error, 1)
	go func() { errLectura <- c.leer(ctxSesion, conn, respuestas) }()

	if err := c.unirse(ctxSesion, respuestas, errLectura); err != nil {
		return err
	}
	c.emitir(Evento{Tipo: TipoConectado})
	return c.latir(ctxSesion, respuestas, errLectura)
}

func (c *Canal) soltar(conn *websocket.Conn) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == conn {
		c.conn, c.unido = nil, false
	}
}

// unirse manda phx_join con la config de Supabase (broadcast sin eco ni ack, presence con su clave, canal privado) y
// el access_token, y espera la respuesta.
func (c *Canal) unirse(ctx context.Context, respuestas <-chan Mensaje, errLectura <-chan error) error {
	payload := map[string]any{
		"config": map[string]any{
			"broadcast":        map[string]bool{"self": false, "ack": false},
			"presence":         map[string]any{"key": c.cfg.PresenceKey, "enabled": true},
			"postgres_changes": []any{},
			"private":          true,
		},
		"access_token": c.cfg.Clave,
	}
	c.mu.Lock()
	c.ref++
	c.joinRef = strconv.Itoa(c.ref)
	ref := c.joinRef
	c.mu.Unlock()
	if err := c.escribir(ctx, Mensaje{JoinRef: ref, Ref: ref, Topic: "realtime:" + c.cfg.Topico, Event: "phx_join"}, payload); err != nil {
		return err
	}
	espera := time.NewTimer(15 * time.Second)
	defer espera.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errLectura:
			return err
		case <-espera.C:
			return errors.New("el relay no respondió al join")
		case m := <-respuestas:
			if m.Ref != ref {
				continue
			}
			if err := errDeRespuesta(m.Payload); err != nil {
				return fmt.Errorf("el relay rechazó el canal: %w", err)
			}
			return c.alUnirse(ctx)
		}
	}
}

// alUnirse marca el canal unido y repite el track que había antes de la caída.
func (c *Canal) alUnirse(ctx context.Context) error {
	c.mu.Lock()
	c.unido = true
	track := c.track
	c.mu.Unlock()
	if track == nil {
		return nil
	}
	return c.mandar(ctx, "presence", map[string]any{"type": "presence", "event": "track", "payload": track})
}

func errDeRespuesta(payload json.RawMessage) error {
	var r struct {
		Status   string          `json:"status"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal(payload, &r); err != nil {
		return err
	}
	if r.Status == "ok" {
		return nil
	}
	var razon struct {
		Reason string `json:"reason"`
	}
	_ = json.Unmarshal(r.Response, &razon)
	if razon.Reason == "" {
		razon.Reason = string(r.Response)
	}
	return errors.New(razon.Reason)
}

// latir manda el heartbeat; si el anterior no tuvo respuesta, la conexión está muerta.
func (c *Canal) latir(ctx context.Context, respuestas <-chan Mensaje, errLectura <-chan error) error {
	t := time.NewTicker(c.cfg.Latido)
	defer t.Stop()
	pendiente := ""
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errLectura:
			return err
		case m := <-respuestas:
			if m.Ref == pendiente {
				pendiente = ""
			}
		case <-t.C:
			if pendiente != "" {
				return errors.New("el relay dejó de responder el latido")
			}
			c.mu.Lock()
			c.ref++
			pendiente = strconv.Itoa(c.ref)
			c.mu.Unlock()
			if err := c.escribir(ctx, Mensaje{Ref: pendiente, Topic: "phoenix", Event: "heartbeat"}, map[string]any{}); err != nil {
				return err
			}
		}
	}
}

// leer reparte lo que llega: respuestas (phx_reply) al que espera, broadcasts y presence al bucle.
func (c *Canal) leer(ctx context.Context, conn *websocket.Conn, respuestas chan<- Mensaje) error {
	for {
		_, datos, err := conn.Read(ctx)
		if err != nil {
			return err
		}
		m, err := DecodificarMensaje(datos)
		if err != nil {
			continue
		}
		if err := c.recibir(m, respuestas); err != nil {
			return err
		}
	}
}

func (c *Canal) recibir(m Mensaje, respuestas chan<- Mensaje) error {
	if m.Event == "phx_reply" {
		select {
		case respuestas <- m:
		default:
		}
		return nil
	}
	if m.Topic != "realtime:"+c.cfg.Topico {
		return nil
	}
	if m.Event == "phx_error" || m.Event == "phx_close" {
		return fmt.Errorf("el relay cerró el canal (%s)", m.Event)
	}
	if m.Event == "broadcast" {
		var b struct {
			Event   string          `json:"event"`
			Payload json.RawMessage `json:"payload"`
		}
		if json.Unmarshal(m.Payload, &b) != nil {
			return nil
		}
		c.emitir(Evento{Tipo: TipoBroadcast, Evento: b.Event, Payload: b.Payload})
		return nil
	}
	if m.Event == "presence_state" {
		c.presenciaEstado(m.Payload)
		return nil
	}
	if m.Event == "presence_diff" {
		c.presenciaDiff(m.Payload)
	}
	return nil
}

type metas struct {
	Metas []json.RawMessage `json:"metas"`
}

func refDe(meta json.RawMessage) string {
	var r struct {
		Ref string `json:"phx_ref"`
	}
	_ = json.Unmarshal(meta, &r)
	return r.Ref
}

func (c *Canal) presenciaEstado(payload json.RawMessage) {
	var estado map[string]metas
	if json.Unmarshal(payload, &estado) != nil {
		return
	}
	c.mu.Lock()
	c.presencia = map[string]map[string]json.RawMessage{}
	for clave, ms := range estado {
		c.sumar(clave, ms.Metas)
	}
	foto := c.fotoPresencia()
	c.mu.Unlock()
	c.emitir(Evento{Tipo: TipoPresencia, Presencias: foto})
}

func (c *Canal) presenciaDiff(payload json.RawMessage) {
	var diff struct {
		Joins  map[string]metas `json:"joins"`
		Leaves map[string]metas `json:"leaves"`
	}
	if json.Unmarshal(payload, &diff) != nil {
		return
	}
	c.mu.Lock()
	for clave, ms := range diff.Leaves {
		for _, m := range ms.Metas {
			delete(c.presencia[clave], refDe(m))
		}
		if len(c.presencia[clave]) == 0 {
			delete(c.presencia, clave)
		}
	}
	for clave, ms := range diff.Joins {
		c.sumar(clave, ms.Metas)
	}
	foto := c.fotoPresencia()
	c.mu.Unlock()
	c.emitir(Evento{Tipo: TipoPresencia, Presencias: foto})
}

// sumar agrega metas a la presencia (con c.mu tomado).
func (c *Canal) sumar(clave string, ms []json.RawMessage) {
	if c.presencia[clave] == nil {
		c.presencia[clave] = map[string]json.RawMessage{}
	}
	for i, m := range ms {
		ref := refDe(m)
		if ref == "" {
			ref = "sin-ref-" + strconv.Itoa(i)
		}
		c.presencia[clave][ref] = m
	}
}

func (c *Canal) fotoPresencia() map[string][]json.RawMessage {
	foto := map[string][]json.RawMessage{}
	for clave, ms := range c.presencia {
		for _, m := range ms {
			foto[clave] = append(foto[clave], m)
		}
	}
	return foto
}

// Presencias es el estado de presence ahora.
func (c *Canal) Presencias() map[string][]json.RawMessage {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fotoPresencia()
}

func (c *Canal) emitir(e Evento) {
	select {
	case c.eventos <- e:
	default:
	}
}

// ErrSinCanal: el canal no está unido ahora (se está reconectando).
var ErrSinCanal = errors.New("el canal del relay no está conectado")

// Broadcast manda un evento al canal.
func (c *Canal) Broadcast(ctx context.Context, evento string, payload any) error {
	return c.mandar(ctx, "broadcast", map[string]any{"type": "broadcast", "event": evento, "payload": payload})
}

// Track publica la presence del PC; se repite sola al reconectar.
func (c *Canal) Track(ctx context.Context, meta any) error {
	b, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.track = b
	unido := c.unido
	c.mu.Unlock()
	if !unido {
		return nil
	}
	return c.mandar(ctx, "presence", map[string]any{"type": "presence", "event": "track", "payload": json.RawMessage(b)})
}

// Untrack deja de publicar la presence del PC.
func (c *Canal) Untrack(ctx context.Context) error {
	c.mu.Lock()
	c.track = nil
	c.mu.Unlock()
	return c.mandar(ctx, "presence", map[string]any{"type": "presence", "event": "untrack"})
}

func (c *Canal) mandar(ctx context.Context, evento string, payload any) error {
	c.mu.Lock()
	c.ref++
	unido, ref, joinRef := c.unido, c.ref, c.joinRef
	c.mu.Unlock()
	if !unido {
		return ErrSinCanal
	}
	return c.escribir(ctx, Mensaje{JoinRef: joinRef, Ref: strconv.Itoa(ref), Topic: "realtime:" + c.cfg.Topico, Event: evento}, payload)
}

func (c *Canal) escribir(ctx context.Context, m Mensaje, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	m.Payload = b
	datos, err := CodificarMensaje(m, c.cfg.Vsn)
	if err != nil {
		return err
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return ErrSinCanal
	}
	ctxEscribir, cancelar := context.WithTimeout(ctx, 10*time.Second)
	defer cancelar()
	return conn.Write(ctxEscribir, websocket.MessageText, datos)
}

// Mensaje es un mensaje de Phoenix.
type Mensaje struct {
	JoinRef string          `json:"join_ref,omitempty"`
	Ref     string          `json:"ref,omitempty"`
	Topic   string          `json:"topic"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

// CodificarMensaje escribe el mensaje en el formato de vsn: un objeto (1.0.0) o un arreglo (2.0.0).
func CodificarMensaje(m Mensaje, vsn string) ([]byte, error) {
	if m.Payload == nil {
		m.Payload = json.RawMessage("{}")
	}
	if !strings.HasPrefix(vsn, "2.") {
		return json.Marshal(m)
	}
	return json.Marshal([]any{nulo(m.JoinRef), nulo(m.Ref), m.Topic, m.Event, m.Payload})
}

func nulo(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// DecodificarMensaje lee un mensaje en cualquiera de los dos formatos.
func DecodificarMensaje(datos []byte) (Mensaje, error) {
	texto := strings.TrimSpace(string(datos))
	if !strings.HasPrefix(texto, "[") {
		var o struct {
			JoinRef any             `json:"join_ref"`
			Ref     any             `json:"ref"`
			Topic   string          `json:"topic"`
			Event   string          `json:"event"`
			Payload json.RawMessage `json:"payload"`
		}
		if err := json.Unmarshal(datos, &o); err != nil {
			return Mensaje{}, err
		}
		return Mensaje{JoinRef: textoRef(o.JoinRef), Ref: textoRef(o.Ref), Topic: o.Topic, Event: o.Event, Payload: o.Payload}, nil
	}
	var partes []json.RawMessage
	if err := json.Unmarshal(datos, &partes); err != nil || len(partes) != 5 {
		return Mensaje{}, fmt.Errorf("mensaje de Phoenix v2 inválido")
	}
	var joinRef, ref any
	var m Mensaje
	_ = json.Unmarshal(partes[0], &joinRef)
	_ = json.Unmarshal(partes[1], &ref)
	if json.Unmarshal(partes[2], &m.Topic) != nil || json.Unmarshal(partes[3], &m.Event) != nil {
		return Mensaje{}, fmt.Errorf("mensaje de Phoenix v2 inválido")
	}
	m.JoinRef, m.Ref, m.Payload = textoRef(joinRef), textoRef(ref), partes[4]
	return m, nil
}

// textoRef: Phoenix manda las refs como texto, pero un servidor puede mandarlas como número.
func textoRef(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if f, ok := v.(float64); ok {
		return strconv.FormatInt(int64(f), 10)
	}
	return ""
}
