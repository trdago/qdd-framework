package remoto

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Autodiagnóstico del canal con QDD Móvil (pedido de la persona desde el teléfono, 2026-11-15): la sonda entra al canal
// del PC como lo haría el teléfono y, si hay un teléfono mirando, manda un broadcast que tiene que volver. Cada falla
// nombra su tramo. Sin teléfono no publica nada (inv. 12).

func relayDe(s *servidorFalso) ConfigRelay { return ConfigRelay{URL: s.URL(), Clave: "publicable"} }

func credPrueba() Credenciales { return Credenciales{Topico: topicoPrueba, Firma: "f"} }

// contrato: remoto#b9dc9a33
func TestSonda_SinTelefonoSoloEntraAlCanalYNoPublica(t *testing.T) {
	s := nuevoServidorFalso(t, "publicable", topicoPrueba)
	r := Sondear(context.Background(), relayDe(s), credPrueba(), 3*time.Second)
	if !r.OK || r.Eco || r.Telefonos != 0 || r.Mensajes != 0 || r.Tramo != "" {
		t.Fatalf("sin teléfono, entrar al canal basta: %+v", r)
	}
	if n := s.Broadcasts(EventoSonda); n != 0 {
		t.Fatalf("sin teléfono no se publica nada (inv. 12): %d broadcasts", n)
	}
	joins := s.Joins()
	if len(joins) != 1 {
		t.Fatalf("un solo join: %v", joins)
	}
	config := joins[0]["config"].(map[string]any)
	if config["private"] != true {
		t.Fatalf("el canal es privado, como el del servicio: %v", config)
	}
}

func TestSonda_ConTelefonoElBroadcastVaYVuelve(t *testing.T) {
	s := nuevoServidorFalso(t, "publicable", topicoPrueba)
	s.presencias["d1"] = []map[string]any{{"rol": "movil", "dispositivo": "d1", "phx_ref": "r1"}}
	r := Sondear(context.Background(), relayDe(s), credPrueba(), 3*time.Second)
	if !r.OK || !r.Eco || r.Telefonos != 1 || r.Mensajes != 2 {
		t.Fatalf("con un teléfono, ida y vuelta: %+v", r)
	}
	m := s.Esperar(t, time.Second, func(m Mensaje) bool { return m.Event == "broadcast" })
	var b struct {
		Event   string         `json:"event"`
		Payload map[string]any `json:"payload"`
	}
	json.Unmarshal(m.Payload, &b)
	if b.Event != EventoSonda || len(b.Payload) != 2 || b.Payload["version"] != float64(1) || b.Payload["marca"] == "" {
		t.Fatalf("la sonda lleva solo su versión y una marca, nada del proyecto: %s", m.Payload)
	}
}

func TestSonda_UnBroadcastQueNoVuelveEsElTramoEco(t *testing.T) {
	s := nuevoServidorFalso(t, "publicable", topicoPrueba)
	s.presencias["d1"] = []map[string]any{{"rol": "movil", "dispositivo": "d1", "phx_ref": "r1"}}
	s.PerderEco()
	inicio := time.Now()
	r := Sondear(context.Background(), relayDe(s), credPrueba(), 1500*time.Millisecond)
	if r.OK || r.Tramo != TramoEco {
		t.Fatalf("el relay aceptó el canal pero no entregó: %+v", r)
	}
	if time.Since(inicio) > 3*time.Second {
		t.Fatalf("la sonda respeta su tope: %s", time.Since(inicio))
	}
}

func TestSonda_UnCanalRechazadoEsElTramoCanal(t *testing.T) {
	s := nuevoServidorFalso(t, "publicable")
	r := Sondear(context.Background(), relayDe(s), credPrueba(), 3*time.Second)
	if r.OK || r.Tramo != TramoCanal {
		t.Fatalf("el relay no deja entrar al canal: %+v", r)
	}
}

func TestSonda_SinConexionEsElTramoConexionYNoDiceLaURLNiLaClave(t *testing.T) {
	s := nuevoServidorFalso(t, "publicable", topicoPrueba)
	c := relayDe(s)
	s.srv.Close()
	r := Sondear(context.Background(), c, credPrueba(), 3*time.Second)
	if r.OK || r.Tramo != TramoConexion {
		t.Fatalf("no llega al relay: %+v", r)
	}
	host := strings.TrimPrefix(c.URL, "http://")
	if strings.Contains(r.Motivo, host) || strings.Contains(r.Motivo, "publicable") || strings.Contains(r.Motivo, topicoPrueba) {
		t.Fatalf("el motivo viaja al teléfono y a doctor: nunca la URL, la clave ni el tópico: %q", r.Motivo)
	}
}

func TestSonda_SinRelayOSinEmparejarEsElTramoRelay(t *testing.T) {
	for _, caso := range []struct {
		c    ConfigRelay
		cred Credenciales
	}{{ConfigRelay{}, credPrueba()}, {ConfigRelay{URL: "http://127.0.0.1:9", Clave: "k"}, Credenciales{}}} {
		r := Sondear(context.Background(), caso.c, caso.cred, time.Second)
		if r.OK || r.Tramo != TramoRelay {
			t.Fatalf("sin relay guardado o sin emparejar no hay canal que probar: %+v", r)
		}
	}
}

func TestSonda_LaReparacionDependeDelTramo(t *testing.T) {
	casos := map[string]string{
		"":            "",
		TramoRelay:    "",
		TramoConexion: "",
		TramoCanal:    ReparaDirectorio,
		TramoRealtime: ReparaReiniciar,
		TramoEco:      ReparaReiniciar,
	}
	for tramo, want := range casos {
		r := ResultadoSonda{OK: tramo == "", Tramo: tramo}
		if got := ReparacionDe(r); got != want {
			t.Errorf("tramo %q: got %q, want %q", tramo, got, want)
		}
	}
}

func TestSonda_ReiniciaComoMuchoUnaVezCadaSeisHoras(t *testing.T) {
	ahora := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	if !PuedeReiniciarPorSonda(time.Time{}, ahora) {
		t.Fatal("nunca reinició: puede")
	}
	if PuedeReiniciarPorSonda(ahora.Add(-time.Hour), ahora) {
		t.Fatal("reinició hace una hora: no, para no quedar en un bucle de reinicios")
	}
	if !PuedeReiniciarPorSonda(ahora.Add(-7*time.Hour), ahora) {
		t.Fatal("hace 7 h: puede otra vez")
	}
}

func TestSonda_SeGuardaYSeLee(t *testing.T) {
	homeTemporal(t)
	en := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	r := ResultadoSonda{OK: false, Tramo: TramoEco, Motivo: MotivoDeTramo(TramoEco), En: en, Reparado: ReparaReiniciar}
	if err := GuardarSonda(r, en); err != nil {
		t.Fatal(err)
	}
	g, err := LeerSonda()
	if err != nil || g.Ultima.Tramo != TramoEco || !g.ReinicioEn.Equal(en) {
		t.Fatalf("queda la última y cuándo reinició: %+v %v", g, err)
	}
	ok := ResultadoSonda{OK: true, En: en.Add(time.Hour)}
	if err := GuardarSonda(ok, time.Time{}); err != nil {
		t.Fatal(err)
	}
	g, _ = LeerSonda()
	if !g.Ultima.OK || !g.ReinicioEn.Equal(en) {
		t.Fatalf("una sonda sin reinicio conserva el último reinicio: %+v", g)
	}
}
