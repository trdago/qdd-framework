package remoto

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.2): el cliente de Realtime contra un Phoenix falso: join privado con la clave publicable, latido,
// broadcast en los dos sentidos, presence y reconexión cuando la red se cae.

const topicoPrueba = "qdd:pc:0b6b1c1e-6a2f-4f39-9f4e-4a1c2d3e4f50:AAAAAAAAAAAAAAAAAAAAAA"

func canalDePrueba(t *testing.T, s *servidorFalso, clave string) (*Canal, context.CancelFunc) {
	t.Helper()
	c := NuevoCanal(ConfigCanal{Relay: s.URL(), Clave: clave, Topico: topicoPrueba, PresenceKey: "pc", Latido: 60 * time.Millisecond, EsperaMax: 80 * time.Millisecond})
	ctx, cancelar := context.WithCancel(context.Background())
	go c.Correr(ctx)
	t.Cleanup(cancelar)
	return c, cancelar
}

func conectado(e Evento) bool { return e.Tipo == TipoConectado }

func TestCanal_SeUneAlCanalPrivadoConLaClavePublicable(t *testing.T) {
	s := nuevoServidorFalso(t, "publicable", topicoPrueba)
	c, _ := canalDePrueba(t, s, "publicable")
	esperarEvento(t, c.Eventos(), 2*time.Second, conectado)
	j := s.Joins()[0]
	cfg := j["config"].(map[string]any)
	bc := cfg["broadcast"].(map[string]any)
	if cfg["private"] != true || bc["self"] != false || bc["ack"] != false || cfg["presence"].(map[string]any)["key"] != "pc" {
		t.Fatalf("config del join: %+v", cfg)
	}
	if cfg["presence"].(map[string]any)["enabled"] != true {
		t.Fatalf("sin enabled, el canal puede no avisar quién entra (FND-167): %+v", cfg)
	}
	if j["access_token"] != "publicable" {
		t.Fatalf("el PC es anon: su access_token es la clave publicable: %+v", j)
	}
}

func TestCanal_UnTopicoQueNoExisteSeRechazaYReintenta(t *testing.T) {
	s := nuevoServidorFalso(t, "publicable")
	c, _ := canalDePrueba(t, s, "publicable")
	e := esperarEvento(t, c.Eventos(), 2*time.Second, func(e Evento) bool { return e.Tipo == TipoDesconectado })
	if !strings.Contains(e.Motivo, "Unauthorized") {
		t.Fatalf("dice por qué: %q", e.Motivo)
	}
	esperarEvento(t, c.Eventos(), 2*time.Second, func(e Evento) bool { return e.Tipo == TipoDesconectado })
	if len(s.Joins()) < 2 {
		t.Fatal("vuelve a intentarlo")
	}
}

func TestCanal_LatidoYSinRespuestaReconecta(t *testing.T) {
	s := nuevoServidorFalso(t, "k", topicoPrueba)
	c, _ := canalDePrueba(t, s, "k")
	esperarEvento(t, c.Eventos(), 2*time.Second, conectado)
	time.Sleep(200 * time.Millisecond)
	if s.Latidos() < 2 {
		t.Fatalf("late: %d", s.Latidos())
	}
	s.mu.Lock()
	s.sinLatido = true
	s.mu.Unlock()
	e := esperarEvento(t, c.Eventos(), 2*time.Second, func(e Evento) bool { return e.Tipo == TipoDesconectado })
	if !strings.Contains(e.Motivo, "latido") {
		t.Fatalf("sin respuesta al latido, la conexión está muerta: %q", e.Motivo)
	}
	s.mu.Lock()
	s.sinLatido = false
	s.mu.Unlock()
	esperarEvento(t, c.Eventos(), 2*time.Second, conectado)
}

func TestCanal_BroadcastEnLosDosSentidos(t *testing.T) {
	s := nuevoServidorFalso(t, "k", topicoPrueba)
	c, _ := canalDePrueba(t, s, "k")
	esperarEvento(t, c.Eventos(), 2*time.Second, conectado)
	if err := c.Broadcast(context.Background(), "estado", map[string]any{"version": 1, "seq": 1}); err != nil {
		t.Fatal(err)
	}
	m := s.Esperar(t, time.Second, func(m Mensaje) bool { return m.Event == "broadcast" })
	var b struct {
		Type, Event string
		Payload     map[string]any
	}
	json.Unmarshal(m.Payload, &b)
	if b.Type != "broadcast" || b.Event != "estado" || b.Payload["seq"] != float64(1) || m.Topic != "realtime:"+topicoPrueba {
		t.Fatalf("broadcast del PC: %+v %s", b, m.Topic)
	}
	s.Difundir(topicoPrueba, "sync", map[string]any{"version": 1})
	e := esperarEvento(t, c.Eventos(), time.Second, func(e Evento) bool { return e.Tipo == TipoBroadcast })
	if e.Evento != "sync" || !strings.Contains(string(e.Payload), `"version":1`) {
		t.Fatalf("broadcast del teléfono: %+v", e)
	}
}

func TestCanal_PresenceDeTelefonosYTrackDelPC(t *testing.T) {
	s := nuevoServidorFalso(t, "k", topicoPrueba)
	s.presencias["tel1"] = []map[string]any{{"phx_ref": "a", "rol": "movil", "dispositivo": "d1"}}
	c, _ := canalDePrueba(t, s, "k")
	e := esperarEvento(t, c.Eventos(), 2*time.Second, func(e Evento) bool { return e.Tipo == TipoPresencia })
	if len(e.Presencias["tel1"]) != 1 {
		t.Fatalf("presence_state: %+v", e.Presencias)
	}
	if err := c.Track(context.Background(), map[string]any{"rol": "pc"}); err != nil {
		t.Fatal(err)
	}
	m := s.Esperar(t, time.Second, func(m Mensaje) bool { return m.Event == "presence" })
	if !strings.Contains(string(m.Payload), `"event":"track"`) || !strings.Contains(string(m.Payload), `"rol":"pc"`) {
		t.Fatalf("track: %s", m.Payload)
	}
	s.Entra(topicoPrueba, "tel2", map[string]any{"rol": "movil", "dispositivo": "d2"})
	e = esperarEvento(t, c.Eventos(), time.Second, func(e Evento) bool { return e.Tipo == TipoPresencia && len(e.Presencias) == 2 })
	s.Sale(topicoPrueba, "tel1")
	e = esperarEvento(t, c.Eventos(), time.Second, func(e Evento) bool { return e.Tipo == TipoPresencia && len(e.Presencias) == 1 })
	if _, ok := e.Presencias["tel2"]; !ok {
		t.Fatalf("queda tel2: %+v", e.Presencias)
	}
}

func TestCanal_TrasUnaCaidaVuelveYRepiteSuPresence(t *testing.T) {
	s := nuevoServidorFalso(t, "k", topicoPrueba)
	c, _ := canalDePrueba(t, s, "k")
	esperarEvento(t, c.Eventos(), 2*time.Second, conectado)
	c.Track(context.Background(), map[string]any{"rol": "pc"})
	s.Esperar(t, time.Second, func(m Mensaje) bool { return m.Event == "presence" })
	s.Cortar()
	esperarEvento(t, c.Eventos(), 2*time.Second, func(e Evento) bool { return e.Tipo == TipoDesconectado })
	if err := c.Broadcast(context.Background(), "x", nil); err == nil {
		t.Fatal("sin canal, broadcast falla en vez de perderse en silencio")
	}
	esperarEvento(t, c.Eventos(), 2*time.Second, conectado)
	m := s.Esperar(t, time.Second, func(m Mensaje) bool { return m.Event == "presence" })
	if !strings.Contains(string(m.Payload), `"rol":"pc"`) {
		t.Fatalf("al volver repite el track: %s", m.Payload)
	}
	if len(s.Joins()) != 2 {
		t.Fatalf("un join por conexión: %d", len(s.Joins()))
	}
}

func TestMensaje_V1YV2(t *testing.T) {
	m := Mensaje{JoinRef: "1", Ref: "2", Topic: "realtime:x", Event: "broadcast", Payload: json.RawMessage(`{"a":1}`)}
	for _, vsn := range []string{"1.0.0", "2.0.0"} {
		b, err := CodificarMensaje(m, vsn)
		if err != nil {
			t.Fatal(err)
		}
		if vsn == "2.0.0" && !strings.HasPrefix(string(b), `["1","2","realtime:x","broadcast",`) {
			t.Fatalf("v2 es un arreglo: %s", b)
		}
		got, err := DecodificarMensaje(b)
		if err != nil || got.JoinRef != "1" || got.Ref != "2" || got.Topic != "realtime:x" || got.Event != "broadcast" || string(got.Payload) != `{"a":1}` {
			t.Fatalf("%s ida y vuelta: %+v %v", vsn, got, err)
		}
	}
	got, err := DecodificarMensaje([]byte(`{"ref":7,"topic":"phoenix","event":"phx_reply","payload":{}}`))
	if err != nil || got.Ref != "7" {
		t.Fatalf("una ref numérica también se lee: %+v %v", got, err)
	}
	if conJitter(time.Second) < 500*time.Millisecond || conJitter(time.Second) > time.Second {
		t.Fatal("el jitter queda entre la mitad y el total")
	}
	if u, _ := URLSocket("https://abc.supabase.co", "sb_publishable_x", "1.0.0"); u != "wss://abc.supabase.co/realtime/v1/websocket?apikey=sb_publishable_x&vsn=1.0.0" {
		t.Fatalf("URL del socket: %s", u)
	}
}
