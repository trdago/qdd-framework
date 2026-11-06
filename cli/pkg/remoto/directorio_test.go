package remoto

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

const relayPrueba = "https://nuevo.supabase.co"

func clavesDirectorio(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	t.Helper()
	publica, privada, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return publica, privada
}

func directorioFirmado(privada ed25519.PrivateKey, url, clave string, serie int) []byte {
	firma := ed25519.Sign(privada, fmt.Appendf(nil, "%s|%s|%d", url, clave, serie))
	b, _ := json.Marshal(map[string]any{"url": url, "clave": clave, "serie": serie, "firma": base64.StdEncoding.EncodeToString(firma)})
	return b
}

func TestLeerDirectorio_AceptaLaFirmaDeLaClavePublica(t *testing.T) {
	publica, privada := clavesDirectorio(t)
	c, err := LeerDirectorio(directorioFirmado(privada, relayPrueba, "sb_publishable_x", 3), publica, 3)
	if err != nil {
		t.Fatal(err)
	}
	if c.URL != relayPrueba || c.Clave != "sb_publishable_x" || c.Serie != 3 || c.Propio {
		t.Fatalf("relay: %+v", c)
	}
}

func TestLeerDirectorio_RechazaLoQueNoEstaFirmadoNiCumple(t *testing.T) {
	publica, privada := clavesDirectorio(t)
	_, otra := clavesDirectorio(t)
	sinFirma, _ := json.Marshal(map[string]any{"url": relayPrueba, "clave": "sb_publishable_x", "serie": 3})
	firmada := directorioFirmado(privada, relayPrueba, "sb_publishable_x", 3)
	cambiada := strings.Replace(string(firmada), "nuevo.supabase.co", "falso.supabase.co", 1)
	casos := map[string][]byte{
		"otra clave":            directorioFirmado(otra, relayPrueba, "sb_publishable_x", 3),
		"sin firma":             sinFirma,
		"url cambiada":          []byte(cambiada),
		"ilegible":              []byte("{roto"),
		"campo desconocido":     []byte(strings.Replace(string(firmada), `{`, `{"extra":1,`, 1)),
		"datos después":         append(append([]byte{}, firmada...), []byte(`{}`)...),
		"serie más vieja":       directorioFirmado(privada, relayPrueba, "sb_publishable_x", 2),
		"serie cero":            directorioFirmado(privada, relayPrueba, "sb_publishable_x", 0),
		"con ruta":              directorioFirmado(privada, relayPrueba+"/rest/v1/", "sb_publishable_x", 3),
		"sin https":             directorioFirmado(privada, "http://nuevo.supabase.co", "sb_publishable_x", 3),
		"localhost sin https":   directorioFirmado(privada, "http://localhost:1", "sb_publishable_x", 3),
		"con usuario":           directorioFirmado(privada, "https://a@nuevo.supabase.co", "sb_publishable_x", 3),
		"con consulta":          directorioFirmado(privada, relayPrueba+"?a=1", "sb_publishable_x", 3),
		"clave secreta":         directorioFirmado(privada, relayPrueba, "sb_secret_x", 3),
		"separador en la url":   directorioFirmado(privada, relayPrueba+"|x", "sb_publishable_x", 3),
		"separador en la clave": directorioFirmado(privada, relayPrueba, "sb_publishable_x|4", 3),
	}
	for motivo, datos := range casos {
		if _, err := LeerDirectorio(datos, publica, 3); err == nil {
			t.Errorf("%s: se aceptó %s", motivo, datos)
		}
	}
}

func TestDirectorioDeFabrica_TraeUnaClavePublicaValida(t *testing.T) {
	d := DirectorioDeFabrica()
	if len(d.Publica) != ed25519.PublicKeySize {
		t.Fatalf("clave pública: %d bytes", len(d.Publica))
	}
	if !strings.HasPrefix(d.URL, "https://") {
		t.Fatalf("url: %s", d.URL)
	}
}

// directorioDePrueba sirve la respuesta y cuenta las consultas.
func directorioDePrueba(t *testing.T, publica ed25519.PublicKey, responder func() (int, []byte)) (Directorio, *int) {
	t.Helper()
	veces := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		veces++
		codigo, cuerpo := responder()
		w.WriteHeader(codigo)
		_, _ = w.Write(cuerpo)
	}))
	t.Cleanup(s.Close)
	return Directorio{URL: s.URL, Publica: publica}, &veces
}

func TestActualizarRelay_ElRelayQueCambioSeGuardaConSuSerie(t *testing.T) {
	homeTemporal(t)
	publica, privada := clavesDirectorio(t)
	d, _ := directorioDePrueba(t, publica, func() (int, []byte) {
		return 200, directorioFirmado(privada, relayPrueba, "sb_publishable_n", 2)
	})
	antes := ConfigRelay{URL: "https://viejo.supabase.co", Clave: "sb_publishable_v", Serie: 1}
	c, cambio, err := ActualizarRelay(context.Background(), d.Consultar, antes)
	if err != nil || !cambio || c.URL != relayPrueba || c.Serie != 2 {
		t.Fatalf("relay: %+v %v %v", c, cambio, err)
	}
	guardado, err := LeerRelay()
	if err != nil || guardado.URL != relayPrueba || guardado.Serie != 2 {
		t.Fatalf("guardado: %+v %v", guardado, err)
	}
	if u := LeerConsultaDirectorio(); !u.OK || u.Serie != 2 {
		t.Fatalf("queda anotado para doctor: %+v", u)
	}
}

func TestActualizarRelay_ElMismoRelayNoEsUnCambio(t *testing.T) {
	homeTemporal(t)
	publica, privada := clavesDirectorio(t)
	d, _ := directorioDePrueba(t, publica, func() (int, []byte) {
		return 200, directorioFirmado(privada, relayPrueba, "sb_publishable_n", 5)
	})
	antes := ConfigRelay{URL: relayPrueba, Clave: "sb_publishable_n", Serie: 4}
	c, cambio, err := ActualizarRelay(context.Background(), d.Consultar, antes)
	if err != nil || cambio || c.Serie != 5 {
		t.Fatalf("recuerda la serie, sin reiniciar nada: %+v %v %v", c, cambio, err)
	}
	if guardado, _ := LeerRelay(); guardado.Serie != 5 {
		t.Fatalf("la serie queda guardada: %+v", guardado)
	}
}

func TestActualizarRelay_ConUnaRespuestaMalaSigueConElRecordado(t *testing.T) {
	publica, privada := clavesDirectorio(t)
	_, otra := clavesDirectorio(t)
	casos := map[string]func() (int, []byte){
		"caído": func() (int, []byte) { return 500, nil },
		"otra firma": func() (int, []byte) {
			return 200, directorioFirmado(otra, "https://falso.supabase.co", "sb_publishable_f", 9)
		},
		"vieja": func() (int, []byte) {
			return 200, directorioFirmado(privada, "https://abandonado.supabase.co", "sb_publishable_a", 1)
		},
	}
	for motivo, responder := range casos {
		homeTemporal(t)
		d, _ := directorioDePrueba(t, publica, responder)
		antes := ConfigRelay{URL: relayPrueba, Clave: "sb_publishable_n", Serie: 4}
		if err := GuardarRelay(antes); err != nil {
			t.Fatal(err)
		}
		c, cambio, err := ActualizarRelay(context.Background(), d.Consultar, antes)
		if err == nil || cambio || c != antes {
			t.Fatalf("%s: sigue con el recordado y dice por qué: %+v %v %v", motivo, c, cambio, err)
		}
		if guardado, _ := LeerRelay(); guardado != antes {
			t.Fatalf("%s: no toca lo guardado: %+v", motivo, guardado)
		}
		if u := LeerConsultaDirectorio(); u.OK || u.Motivo == "" {
			t.Fatalf("%s: doctor ve el motivo: %+v", motivo, u)
		}
	}
}

func TestActualizarRelay_UnRelayPropioNoConsultaNunca(t *testing.T) {
	homeTemporal(t)
	publica, privada := clavesDirectorio(t)
	d, veces := directorioDePrueba(t, publica, func() (int, []byte) {
		return 200, directorioFirmado(privada, relayPrueba, "sb_publishable_n", 2)
	})
	propio := ConfigRelay{URL: "https://mio.supabase.co", Clave: "sb_publishable_m", Propio: true}
	c, cambio, err := ActualizarRelay(context.Background(), d.Consultar, propio)
	if err != nil || cambio || c != propio || *veces != 0 {
		t.Fatalf("no se pisa un relay propio: %+v %v %v, %d consultas", c, cambio, err, *veces)
	}
}

func TestConsultar_NoSigueRedireccionesNiLeeDeMas(t *testing.T) {
	publica, privada := clavesDirectorio(t)
	buena := directorioFirmado(privada, relayPrueba, "sb_publishable_n", 2)
	falso := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(buena) }))
	defer falso.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, falso.URL, http.StatusFound) }))
	defer s.Close()
	if _, err := (Directorio{URL: s.URL, Publica: publica}).Consultar(context.Background(), 0); err == nil {
		t.Fatal("una redirección no se sigue")
	}
	grande := append(append([]byte{}, buena...), []byte(strings.Repeat(" ", maxDirectorio))...)
	d, _ := directorioDePrueba(t, publica, func() (int, []byte) { return 200, grande })
	if _, err := d.Consultar(context.Background(), 0); err == nil {
		t.Fatal("una respuesta demasiado grande se rechaza")
	}
}

func TestBucle_SinConexionConsultaConTope(t *testing.T) {
	inicio := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	en := func(d time.Duration) time.Time { return inicio.Add(d) }
	b := &Bucle{CadaConsulta: 10 * time.Minute, EsperaConsulta: time.Minute}
	if b.tocaConsultar(en(time.Hour)) {
		t.Fatal("conectado no consulta")
	}
	b.anotarCaida(inicio)
	if b.tocaConsultar(en(30 * time.Second)) {
		t.Fatal("una caída corta no consulta")
	}
	if !b.tocaConsultar(en(2 * time.Minute)) {
		t.Fatal("una caída larga consulta")
	}
	b.consultadoEn = en(2 * time.Minute)
	if b.tocaConsultar(en(5 * time.Minute)) {
		t.Fatal("no consulta en cada reintento")
	}
	if !b.tocaConsultar(en(13 * time.Minute)) {
		t.Fatal("pasado el tope, vuelve a consultar")
	}
	b.anotarCaida(en(14 * time.Minute))
	if b.caidoDesde != inicio {
		t.Fatal("otra caída seguida no reinicia la cuenta")
	}
	b.anotarConexion()
	if b.tocaConsultar(en(time.Hour)) {
		t.Fatal("al volver la conexión deja de consultar")
	}
}

func TestBucle_UnRelayQueCambioTerminaElBucleParaReiniciar(t *testing.T) {
	homeTemporal(t)
	tr, rl := nuevoTransporte(), nuevoRelayBucle()
	b := &Bucle{QDD: "v-prueba", Cred: Credenciales{Topico: topicoPrueba, Firma: "f"}, T: tr, R: rl,
		Agrupar: 30 * time.Millisecond, CadaLatido: time.Hour, CadaSondeo: 5 * time.Millisecond,
		EsperaConsulta: time.Millisecond, CadaConsulta: time.Hour,
		Consultar: func(context.Context) (bool, error) { return true, nil }}
	ctx, cancelar := context.WithCancel(context.Background())
	defer cancelar()
	fin := make(chan error, 1)
	go func() { fin <- b.Correr(ctx) }()
	tr.eventos <- Evento{Tipo: TipoDesconectado, Motivo: "sin red"}
	select {
	case err := <-fin:
		if !errors.Is(err, ErrRelayCambio) {
			t.Fatalf("termina con ErrRelayCambio: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("el bucle sigue con el relay viejo")
	}
}

func TestErrRelayCambio_NoEsUnErrorDeCredenciales(t *testing.T) {
	if errors.Is(ErrRelayCambio, ErrCredenciales) {
		t.Fatal("un cambio de relay no significa que quitaron el PC")
	}
}
