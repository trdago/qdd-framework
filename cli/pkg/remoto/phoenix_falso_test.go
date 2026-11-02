package remoto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// Rule: Prohibido usar 'else'
// Un servidor de Phoenix falso, con lo que hace Supabase Realtime para un canal privado: join con su config y
// access_token, heartbeat, broadcast, presence (state y diff) y caídas. Lo usan las pruebas del canal y del bucle.

type servidorFalso struct {
	t       *testing.T
	srv     *httptest.Server
	clave   string
	topicos map[string]bool

	mu         sync.Mutex
	conns      []*websocket.Conn
	joins      []map[string]any
	latidos    int
	sinLatido  bool
	recibidos  chan Mensaje
	presencias map[string][]map[string]any
	ref        int
}

func nuevoServidorFalso(t *testing.T, clave string, topicos ...string) *servidorFalso {
	t.Helper()
	s := &servidorFalso{t: t, clave: clave, topicos: map[string]bool{}, recibidos: make(chan Mensaje, 512), presencias: map[string][]map[string]any{}}
	for _, tp := range topicos {
		s.topicos[tp] = true
	}
	s.srv = httptest.NewServer(http.HandlerFunc(s.atender))
	t.Cleanup(func() {
		s.Cortar()
		s.srv.Close()
	})
	return s
}

func (s *servidorFalso) URL() string { return s.srv.URL }

func (s *servidorFalso) atender(w http.ResponseWriter, r *http.Request) {
	q, _ := url.ParseQuery(r.URL.RawQuery)
	if r.URL.Path != "/realtime/v1/websocket" || q.Get("apikey") != s.clave || q.Get("vsn") == "" {
		http.Error(w, "no", http.StatusUnauthorized)
		return
	}
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.conns = append(s.conns, conn)
	s.mu.Unlock()
	ctx := context.Background()
	for {
		_, datos, err := conn.Read(ctx)
		if err != nil {
			return
		}
		m, err := DecodificarMensaje(datos)
		if err != nil {
			continue
		}
		s.manejar(ctx, conn, m)
	}
}

func (s *servidorFalso) manejar(ctx context.Context, conn *websocket.Conn, m Mensaje) {
	if m.Event == "heartbeat" {
		s.mu.Lock()
		s.latidos++
		sin := s.sinLatido
		s.mu.Unlock()
		if sin {
			return
		}
		s.responder(ctx, conn, m, "ok", map[string]any{})
		return
	}
	if m.Event == "phx_join" {
		var p map[string]any
		json.Unmarshal(m.Payload, &p)
		s.mu.Lock()
		s.joins = append(s.joins, p)
		s.mu.Unlock()
		topico := strings.TrimPrefix(m.Topic, "realtime:")
		if p["access_token"] != s.clave || !s.topicos[topico] {
			s.responder(ctx, conn, m, "error", map[string]any{"reason": "Unauthorized: You do not have permissions to read from this Channel topic: " + topico})
			return
		}
		s.responder(ctx, conn, m, "ok", map[string]any{"postgres_changes": []any{}})
		s.enviarA(ctx, conn, m.Topic, "presence_state", s.estadoPresencia())
		return
	}
	s.recibidos <- m
}

func (s *servidorFalso) responder(ctx context.Context, conn *websocket.Conn, m Mensaje, status string, resp any) {
	b, _ := json.Marshal(map[string]any{"status": status, "response": resp})
	out, _ := CodificarMensaje(Mensaje{JoinRef: m.JoinRef, Ref: m.Ref, Topic: m.Topic, Event: "phx_reply", Payload: b}, "1.0.0")
	conn.Write(ctx, websocket.MessageText, out)
}

func (s *servidorFalso) enviarA(ctx context.Context, conn *websocket.Conn, topic, evento string, payload any) {
	b, _ := json.Marshal(payload)
	out, _ := CodificarMensaje(Mensaje{Topic: topic, Event: evento, Payload: b}, "1.0.0")
	conn.Write(ctx, websocket.MessageText, out)
}

func (s *servidorFalso) todas(topico, evento string, payload any) {
	s.mu.Lock()
	conns := append([]*websocket.Conn{}, s.conns...)
	s.mu.Unlock()
	for _, c := range conns {
		s.enviarA(context.Background(), c, "realtime:"+topico, evento, payload)
	}
}

// Difundir manda un broadcast al canal (lo que haría un teléfono o realtime.send).
func (s *servidorFalso) Difundir(topico, evento string, payload any) {
	s.todas(topico, "broadcast", map[string]any{"type": "broadcast", "event": evento, "payload": payload})
}

func (s *servidorFalso) estadoPresencia() map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]any{}
	for k, ms := range s.presencias {
		out[k] = map[string]any{"metas": ms}
	}
	return out
}

// Entra suma un teléfono a la presence y avisa con presence_diff.
func (s *servidorFalso) Entra(topico, clave string, meta map[string]any) {
	s.mu.Lock()
	s.ref++
	meta["phx_ref"] = "ref-" + clave + "-" + strings.Repeat("x", s.ref)
	s.presencias[clave] = append(s.presencias[clave], meta)
	s.mu.Unlock()
	s.todas(topico, "presence_diff", map[string]any{"joins": map[string]any{clave: map[string]any{"metas": []any{meta}}}, "leaves": map[string]any{}})
}

// Sale quita un teléfono de la presence.
func (s *servidorFalso) Sale(topico, clave string) {
	s.mu.Lock()
	ms := s.presencias[clave]
	delete(s.presencias, clave)
	s.mu.Unlock()
	s.todas(topico, "presence_diff", map[string]any{"joins": map[string]any{}, "leaves": map[string]any{clave: map[string]any{"metas": ms}}})
}

// Cortar cierra todas las conexiones, como una caída de la red.
func (s *servidorFalso) Cortar() {
	s.mu.Lock()
	conns := s.conns
	s.conns = nil
	s.mu.Unlock()
	for _, c := range conns {
		c.CloseNow()
	}
}

func (s *servidorFalso) Joins() []map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]map[string]any{}, s.joins...)
}

func (s *servidorFalso) Latidos() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.latidos
}

// Esperar devuelve el siguiente mensaje del cliente que cumple f, o falla pasado el plazo.
func (s *servidorFalso) Esperar(t *testing.T, plazo time.Duration, f func(Mensaje) bool) Mensaje {
	t.Helper()
	limite := time.After(plazo)
	for {
		select {
		case m := <-s.recibidos:
			if f(m) {
				return m
			}
		case <-limite:
			t.Fatal("el cliente no mandó lo esperado")
			return Mensaje{}
		}
	}
}

// esperarEvento devuelve el siguiente evento del canal que cumple f.
func esperarEvento(t *testing.T, eventos <-chan Evento, plazo time.Duration, f func(Evento) bool) Evento {
	t.Helper()
	limite := time.After(plazo)
	for {
		select {
		case e := <-eventos:
			if f(e) {
				return e
			}
		case <-limite:
			t.Fatal("el canal no entregó lo esperado")
			return Evento{}
		}
	}
}
