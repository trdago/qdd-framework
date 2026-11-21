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
	if len(d.URLs) != 3 {
		t.Fatalf("tres sitios, en este orden: %v", d.URLs)
	}
	for _, u := range d.URLs {
		if !strings.HasPrefix(u, "https://") {
			t.Fatalf("url: %s", u)
		}
	}
	if d.URLs[0] != "https://qdd-movile.3ldago.workers.dev/" {
		t.Fatalf("los PC de hoy siguen preguntando primero donde siempre: %v", d.URLs)
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
	return Directorio{URLs: []string{s.URL}, Publica: publica}, &veces
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

// FND-171: Propio protege a un PC emparejado. Sin emparejar no hay nada que perder, y un código de un relay retirado gasta
// los intentos de la persona.
func TestConfirmarRelay_ConElPCEmparejadoNoConsultaUnRelayPropio_FND171(t *testing.T) {
	homeTemporal(t)
	publica, privada := clavesDirectorio(t)
	d, veces := directorioDePrueba(t, publica, func() (int, []byte) {
		return 200, directorioFirmado(privada, relayPrueba, "sb_publishable_n", 2)
	})
	propio := ConfigRelay{URL: "https://mio.supabase.co", Clave: "sb_publishable_m", Propio: true}
	c, cambio, err := ConfirmarRelay(context.Background(), d.Consultar, propio, true)
	if err != nil || cambio || c != propio || *veces != 0 {
		t.Fatalf("no se pisa: %+v %v %v, %d consultas", c, cambio, err, *veces)
	}
}

func TestConfirmarRelay_SinEmparejarElDirectorioReemplazaUnPropio_FND171(t *testing.T) {
	homeTemporal(t)
	publica, privada := clavesDirectorio(t)
	d, _ := directorioDePrueba(t, publica, func() (int, []byte) {
		return 200, directorioFirmado(privada, relayPrueba, "sb_publishable_n", 2)
	})
	propio := ConfigRelay{URL: "https://antiguo.supabase.co", Clave: "sb_publishable_a", Propio: true}
	if err := GuardarRelay(propio); err != nil {
		t.Fatal(err)
	}
	c, cambio, err := ConfirmarRelay(context.Background(), d.Consultar, propio, false)
	if err != nil || !cambio || c.URL != relayPrueba || c.Propio || c.Serie != 2 {
		t.Fatalf("toma el firmado: %+v %v %v", c, cambio, err)
	}
	if guardado, err := LeerRelay(); err != nil || guardado != c {
		t.Fatalf("queda guardado: %+v %v", guardado, err)
	}
	if u := LeerConsultaDirectorio(); !u.OK {
		t.Fatalf("doctor ve que se confirmó: %+v", u)
	}
}

func TestConfirmarRelay_SinEmparejarUnPropioQueCoincideDejaDeSerlo_FND171(t *testing.T) {
	homeTemporal(t)
	publica, privada := clavesDirectorio(t)
	d, _ := directorioDePrueba(t, publica, func() (int, []byte) {
		return 200, directorioFirmado(privada, relayPrueba, "sb_publishable_n", 2)
	})
	c, cambio, err := ConfirmarRelay(context.Background(), d.Consultar, ConfigRelay{URL: relayPrueba, Clave: "sb_publishable_n", Propio: true}, false)
	if err != nil || cambio || c.Propio || c.Serie != 2 {
		t.Fatalf("es el mismo relay, ahora confirmado: %+v %v %v", c, cambio, err)
	}
}

func TestConfirmarRelay_SiLaConsultaFallaNoCambiaNadaYDejaElMotivo_FND171(t *testing.T) {
	homeTemporal(t)
	publica, _ := clavesDirectorio(t)
	d, _ := directorioDePrueba(t, publica, func() (int, []byte) { return 500, nil })
	propio := ConfigRelay{URL: "https://antiguo.supabase.co", Clave: "sb_publishable_a", Propio: true}
	if err := GuardarRelay(propio); err != nil {
		t.Fatal(err)
	}
	c, cambio, err := ConfirmarRelay(context.Background(), d.Consultar, propio, false)
	if err == nil || cambio || c != propio {
		t.Fatalf("sigue igual y dice por qué: %+v %v %v", c, cambio, err)
	}
	if guardado, err := LeerRelay(); err != nil || guardado != propio {
		t.Fatalf("el archivo no cambia: %+v %v", guardado, err)
	}
	if u := LeerConsultaDirectorio(); u.OK || u.Motivo == "" {
		t.Fatalf("doctor ve el motivo: %+v", u)
	}
}

func TestConsultar_NoSigueRedireccionesNiLeeDeMas(t *testing.T) {
	publica, privada := clavesDirectorio(t)
	buena := directorioFirmado(privada, relayPrueba, "sb_publishable_n", 2)
	falso := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(buena) }))
	defer falso.Close()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, falso.URL, http.StatusFound) }))
	defer s.Close()
	if _, err := (Directorio{URLs: []string{s.URL}, Publica: publica}).Consultar(context.Background(), 0); err == nil {
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

// FND-171: en una red que bloquea un sitio, un PC no tenía cómo enterarse de una mudanza y uno nuevo no tenía relay. La
// firma es lo que da confianza, no el sitio: la misma respuesta puede venir de cualquiera.
func sitioDePrueba(t *testing.T, codigo int, cuerpo []byte) (string, *int) {
	t.Helper()
	veces := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		veces++
		w.WriteHeader(codigo)
		_, _ = w.Write(cuerpo)
	}))
	t.Cleanup(s.Close)
	return s.URL, &veces
}

func TestConsultar_SeQuedaConElPrimerSitioQueValidaYNoPreguntaMas_FND171(t *testing.T) {
	publica, privada := clavesDirectorio(t)
	primero, _ := sitioDePrueba(t, 200, directorioFirmado(privada, relayPrueba, "sb_publishable_1", 2))
	segundo, veces := sitioDePrueba(t, 200, directorioFirmado(privada, relayPrueba, "sb_publishable_2", 2))
	c, err := Directorio{URLs: []string{primero, segundo}, Publica: publica}.Consultar(context.Background(), 0)
	if err != nil || c.Clave != "sb_publishable_1" || *veces != 0 {
		t.Fatalf("%+v %v, el segundo se consultó %d veces", c, err, *veces)
	}
}

func TestConsultar_CualquierFalloDeUnSitioPasaAlSiguiente_FND171(t *testing.T) {
	publica, privada := clavesDirectorio(t)
	_, otraPrivada := clavesDirectorio(t)
	bueno := directorioFirmado(privada, relayPrueba, "sb_publishable_bueno", 3)
	casos := map[string]struct {
		codigo int
		cuerpo []byte
	}{
		"bloqueado":      {403, nil},
		"caído":          {500, nil},
		"no se entiende": {200, []byte("<html>")},
		"firma de otro":  {200, directorioFirmado(otraPrivada, relayPrueba, "sb_publishable_x", 3)},
		"serie vieja":    {200, directorioFirmado(privada, relayPrueba, "sb_publishable_viejo", 1)},
	}
	for nombre, caso := range casos {
		t.Run(nombre, func(t *testing.T) {
			malo, _ := sitioDePrueba(t, caso.codigo, caso.cuerpo)
			sano, _ := sitioDePrueba(t, 200, bueno)
			c, err := Directorio{URLs: []string{malo, sano}, Publica: publica}.Consultar(context.Background(), 2)
			if err != nil || c.Clave != "sb_publishable_bueno" {
				t.Fatalf("toma el del sitio sano: %+v %v", c, err)
			}
		})
	}
}

func TestConsultar_UnSitioQueRedirigePasaAlSiguienteSinSeguirla_FND171(t *testing.T) {
	publica, privada := clavesDirectorio(t)
	falso, vecesFalso := sitioDePrueba(t, 200, directorioFirmado(privada, relayPrueba, "sb_publishable_falso", 2))
	redirige := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, falso, http.StatusFound) }))
	t.Cleanup(redirige.Close)
	sano, _ := sitioDePrueba(t, 200, directorioFirmado(privada, relayPrueba, "sb_publishable_sano", 2))
	c, err := Directorio{URLs: []string{redirige.URL, sano}, Publica: publica}.Consultar(context.Background(), 0)
	if err != nil || c.Clave != "sb_publishable_sano" || *vecesFalso != 0 {
		t.Fatalf("%+v %v, la redirección se siguió %d veces", c, err, *vecesFalso)
	}
}

func TestConsultar_SiTodosFallanDiceElMotivoDeCadaUnoSinNombrarlos_FND171(t *testing.T) {
	publica, _ := clavesDirectorio(t)
	primero, _ := sitioDePrueba(t, 403, nil)
	segundo, _ := sitioDePrueba(t, 500, nil)
	_, err := Directorio{URLs: []string{primero, segundo}, Publica: publica}.Consultar(context.Background(), 0)
	if err == nil {
		t.Fatal("todos fallaron")
	}
	for _, parte := range []string{"sitio 1", "403", "sitio 2", "500"} {
		if !strings.Contains(err.Error(), parte) {
			t.Errorf("debe decir %q: %v", parte, err)
		}
	}
	if strings.Contains(err.Error(), "127.0.0.1") {
		t.Errorf("el motivo llega al reporte de doctor: sin hosts: %v", err)
	}
}

func TestConsultar_UnContextoCanceladoNoSigueConLosDemas_FND171(t *testing.T) {
	publica, _ := clavesDirectorio(t)
	primero, _ := sitioDePrueba(t, 500, nil)
	segundo, veces := sitioDePrueba(t, 500, nil)
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	if _, err := (Directorio{URLs: []string{primero, segundo}, Publica: publica}).Consultar(ctx, 0); err == nil {
		t.Fatal("no hay respuesta")
	}
	if *veces != 0 {
		t.Fatalf("se apagó: no pregunta más: %d", *veces)
	}
}
