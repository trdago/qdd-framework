package reportes

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/incidentes"
	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0024, fase 1. Ninguna prueba habla con el relay real ni toca el QDD_HOME real: relay falso y QDD_HOME temporal.

var ahora = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

var ctxPrueba = incidentes.Contexto{Raiz: "/home/ana/proyectos/tienda", Home: "/home/ana", Usuario: "ana"}

func entorno(t *testing.T) {
	t.Helper()
	t.Setenv("QDD_HOME", t.TempDir())
}

func nuevo(t *testing.T, titulo, detalle string) Reporte {
	t.Helper()
	r, err := Nuevo(Datos{Titulo: titulo, Detalle: detalle, Tipo: "bug", Version: "2.10.0-rc.41", SO: "linux/amd64"}, ctxPrueba, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestLimpiarQuitaSecretosRutasYNombre(t *testing.T) {
	texto := "Falla en /home/ana/proyectos/tienda/src/pago.go con token=abc123 y ghp_ABCDEFGHIJ1234567890. " +
		"Mi config está en /home/ana/.config/x. Soy ana, escríbeme a ana@correo.cl. Valor: cf-SECRETO-99. " +
		"El archivo «pago.go» falla."
	got := Limpiar(texto, ctxPrueba, []string{"cf-SECRETO-99"})
	for _, no := range []string{"/home/ana", "abc123", "ghp_ABCDEFGHIJ", "cf-SECRETO-99", "Soy ana,", "tienda/src"} {
		if strings.Contains(got, no) {
			t.Fatalf("quedó %q en %q", no, got)
		}
	}
	for _, si := range []string{"<proyecto>/src/pago.go", "~/.config/x", "ana@correo.cl", "«pago.go»", "<usuario>"} {
		if !strings.Contains(got, si) {
			t.Fatalf("falta %q en %q (un reporte lo escribe una persona: lo suyo se queda)", si, got)
		}
	}
}

func TestLimpiarNoTocaPalabrasQueContienenElNombre(t *testing.T) {
	got := Limpiar("la ventana de banana se cierra", ctxPrueba, nil)
	if got != "la ventana de banana se cierra" {
		t.Fatalf("cambió palabras que solo contienen el nombre: %q", got)
	}
}

func TestNuevoValidaTipoYTamanos(t *testing.T) {
	casos := []Datos{
		{Titulo: "", Tipo: "bug"},
		{Titulo: "x", Tipo: "queja"},
		{Titulo: strings.Repeat("é", MaxTitulo+1), Tipo: "bug"},
		{Titulo: "x", Detalle: strings.Repeat("é", MaxDetalle+1), Tipo: "bug"},
	}
	for _, d := range casos {
		if _, err := Nuevo(d, ctxPrueba, nil); err == nil {
			t.Fatalf("debió rechazar %+v", d)
		}
	}
	r, err := Nuevo(Datos{Titulo: "  hola  ", Version: "v", SO: "s"}, ctxPrueba, nil)
	if err != nil || r.Tipo != "bug" || r.Titulo != "hola" || r.Formato != 1 {
		t.Fatalf("sin tipo es un bug, con el título recortado: %+v %v", r, err)
	}
}

func TestVistaEsLoQueViaja(t *testing.T) {
	r := nuevo(t, "Se cae el panel", "al abrir en /home/ana/proyectos/tienda")
	v := r.Vista()
	for _, si := range []string{"Se cae el panel", "<proyecto>", "2.10.0-rc.41", "linux/amd64", "bug"} {
		if !strings.Contains(v, si) {
			t.Fatalf("la vista no muestra %q: %s", si, v)
		}
	}
	if strings.Contains(v, "/home/ana") {
		t.Fatalf("la vista muestra la ruta: %s", v)
	}
}

func TestLlaveSeCreaUnaVezPrivada(t *testing.T) {
	entorno(t)
	a, err := Llave()
	if err != nil {
		t.Fatal(err)
	}
	b, err := Llave()
	if err != nil || !a.Equal(b) {
		t.Fatalf("la segunda vez debe ser la misma llave: %v", err)
	}
	ruta, _ := rutaEn("llave")
	info, err := os.Stat(ruta)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("la llave debe quedar 0600: %v %v", info, err)
	}
}

// relayFalso valida como el relay de verdad (firma, campos, ventana de 10 min) y responde lo que se le pide.
func relayFalso(t *testing.T, estado int, cuerpo string, visto *Reporte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/functions/v1/reportar" || req.Header.Get("apikey") != "anon" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		datos, _ := io.ReadAll(req.Body)
		var r Reporte
		dec := json.NewDecoder(strings.NewReader(string(datos)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&r); err != nil {
			t.Errorf("campo de más o JSON malo: %v", err)
		}
		pub, _ := base64.StdEncoding.DecodeString(r.Instalacion)
		firma, _ := base64.StdEncoding.DecodeString(r.Firma)
		if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, []byte(r.mensaje()), firma) {
			t.Errorf("la firma no valida")
		}
		*visto = r
		w.WriteHeader(estado)
		_, _ = io.WriteString(w, cuerpo)
	}))
}

func TestEnviarFirmaYGuardaEnviado(t *testing.T) {
	entorno(t)
	var visto Reporte
	srv := relayFalso(t, http.StatusCreated, `{"id":"REP-7"}`, &visto)
	defer srv.Close()
	r := nuevo(t, "Se cae el panel", "detalle")
	id, err := Enviar(context.Background(), remoto.ConfigRelay{URL: srv.URL, Clave: "anon"}, r, ahora)
	if err != nil || id != "REP-7" {
		t.Fatalf("debió enviarse: %q %v", id, err)
	}
	if visto.Titulo != "Se cae el panel" || visto.En != "2026-11-08T12:00:00Z" || visto.Version != "2.10.0-rc.41" {
		t.Fatalf("viajó otra cosa: %+v", visto)
	}
	enviados, err := Enviados()
	if err != nil || len(enviados) != 1 || enviados[0].ID != "REP-7" || enviados[0].Reporte.Titulo != "Se cae el panel" {
		t.Fatalf("lo enviado debe quedar guardado: %+v %v", enviados, err)
	}
}

func TestEnviarTopeYRelaySinFuncion(t *testing.T) {
	entorno(t)
	var visto Reporte
	tope := relayFalso(t, http.StatusTooManyRequests, `{"error":"tope"}`, &visto)
	defer tope.Close()
	_, err := Enviar(context.Background(), remoto.ConfigRelay{URL: tope.URL, Clave: "anon"}, nuevo(t, "a", ""), ahora)
	if !errors.Is(err, ErrTope) {
		t.Fatalf("429 es el tope: %v", err)
	}
	_, err = Enviar(context.Background(), remoto.ConfigRelay{URL: tope.URL, Clave: "otra"}, nuevo(t, "a", ""), ahora)
	if !errors.Is(err, ErrSinRelay) {
		t.Fatalf("404: el relay todavía no recibe reportes: %v", err)
	}
	if enviados, _ := Enviados(); len(enviados) != 0 {
		t.Fatalf("lo que no se envió no queda como enviado: %+v", enviados)
	}
}

func TestPendienteSeGuardaYSeEnviaUnaVez(t *testing.T) {
	entorno(t)
	r := nuevo(t, "Pendiente", "x")
	id, err := GuardarPendiente(r, ahora)
	if err != nil || id == "" {
		t.Fatal(err)
	}
	leido, err := LeerPendiente(id)
	if err != nil || leido.Titulo != "Pendiente" {
		t.Fatalf("debe leerse igual: %+v %v", leido, err)
	}
	if ComandoEnviar(id) != "qdd reportar enviar "+id {
		t.Fatalf("comando del pedido: %q", ComandoEnviar(id))
	}
	var visto Reporte
	srv := relayFalso(t, http.StatusCreated, `{"id":"REP-8"}`, &visto)
	defer srv.Close()
	if _, err := EnviarPendiente(context.Background(), remoto.ConfigRelay{URL: srv.URL, Clave: "anon"}, id, ahora); err != nil {
		t.Fatal(err)
	}
	if _, err := LeerPendiente(id); err == nil {
		t.Fatal("enviado, el pendiente se quita")
	}
}

func TestPendienteRechazaIdConRuta(t *testing.T) {
	entorno(t)
	for _, id := range []string{"../llave", "a/b", "", "."} {
		if _, err := LeerPendiente(id); err == nil {
			t.Fatalf("un id %q no puede salir de pendientes/", id)
		}
	}
}

func TestLlavePublicaNoEsLaPrivada(t *testing.T) {
	entorno(t)
	pub, err := Publica()
	if err != nil {
		t.Fatal(err)
	}
	ruta, _ := rutaEn("llave")
	datos, _ := os.ReadFile(ruta)
	if strings.TrimSpace(string(datos)) == pub {
		t.Fatal("el archivo de la llave guarda la semilla privada, no la pública")
	}
	decod, err := base64.StdEncoding.DecodeString(pub)
	if err != nil || len(decod) != ed25519.PublicKeySize {
		t.Fatalf("la pública es ed25519 en base64: %v", err)
	}
}

// FND-186: una ruta del proyecto con un tramo de 32+ caracteres (como las de Claude Code, «-home-ana-proyectos-…»)
// perdía ese tramo por la regla de cadenas largas antes de reemplazar la ruta, y el resto de la ruta viajaba.
func TestLimpiarRutaConTramoLargo_FND186(t *testing.T) {
	c := incidentes.Contexto{Raiz: "/tmp/claude-1000/-home-ana-Documents-proyectos-tienda/abc/p", Home: "/home/ana", Usuario: "ana"}
	got := Limpiar("falla en /tmp/claude-1000/-home-ana-Documents-proyectos-tienda/abc/p/x.go", c, nil)
	if got != "falla en <proyecto>/x.go" {
		t.Fatalf("la ruta del proyecto se reemplaza entera: %q", got)
	}
}

// RFC 0024, fase 3: la instalación lectora pide los reportes firmando «reportes|desde|en» con su llave.
func TestLeerFirmaLaConsultaYPagina(t *testing.T) {
	entorno(t)
	pedidos := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/functions/v1/reportes" || req.Method != http.MethodGet || req.Header.Get("apikey") != "anon" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		desde, en := req.URL.Query().Get("desde"), req.Header.Get("x-qdd-en")
		pub, _ := base64.StdEncoding.DecodeString(req.Header.Get("x-qdd-instalacion"))
		firma, _ := base64.StdEncoding.DecodeString(req.Header.Get("x-qdd-firma"))
		if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, []byte("reportes|"+desde+"|"+en), firma) || en != "2026-11-08T12:00:00Z" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":"firma"}`)
			return
		}
		pedidos++
		if desde == "" {
			lote := make([]string, 0, 100)
			for i := 1; i <= 100; i++ {
				lote = append(lote, fmt.Sprintf(`{"id":"REP-%d","tipo":"bug","titulo":"t%d","detalle":"","version":"v","so":"linux/amd64","llego":"2026-11-08T11:00:00Z"}`, i, i))
			}
			_, _ = io.WriteString(w, `{"reportes":[`+strings.Join(lote, ",")+`]}`)
			return
		}
		if desde == "REP-100" {
			_, _ = io.WriteString(w, `{"reportes":[{"id":"REP-101","tipo":"mejora","titulo":"último","detalle":"d","version":"v","so":"s","llego":"2026-11-08T11:30:00Z"}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"reportes":[]}`)
	}))
	defer srv.Close()
	cfg := remoto.ConfigRelay{URL: srv.URL, Clave: "anon"}
	lista, err := Leer(context.Background(), cfg, "", ahora)
	if err != nil || len(lista) != 101 || lista[100].ID != "REP-101" || lista[100].Tipo != "mejora" || pedidos != 2 {
		t.Fatalf("debe seguir de a 100 hasta el final: %d reportes, %d pedidos, %v", len(lista), pedidos, err)
	}
}

func TestLeerSinSerLectora(t *testing.T) {
	entorno(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"error":"no-lector"}`)
	}))
	defer srv.Close()
	_, err := Leer(context.Background(), remoto.ConfigRelay{URL: srv.URL, Clave: "anon"}, "", ahora)
	if !errors.Is(err, ErrNoLector) || !strings.Contains(err.Error(), "qdd reportar llave") {
		t.Fatalf("403: esta instalación no es lectora, y dice cómo serlo: %v", err)
	}
}
