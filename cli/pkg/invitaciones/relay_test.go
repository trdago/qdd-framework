package invitaciones

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 3: el PC habla con el relay de qdd-movile firmando cada llamada con la llave de la instalación. Las
// pruebas usan un relay falso que revisa la firma como el de verdad: nunca tocan el servicio real.

var ahoraDePrueba = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

type llamada struct {
	metodo, funcion, consulta string
	cuerpo                    []byte
}

// relayFalso revisa la firma de cada llamada y responde con lo que diga responder.
func relayFalso(t *testing.T, publica ed25519.PublicKey, responder func(l llamada) (int, string)) (*Cliente, *[]llamada) {
	t.Helper()
	var llamadas []llamada
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cuerpo, _ := io.ReadAll(r.Body)
		l := llamada{metodo: r.Method, funcion: strings.TrimPrefix(r.URL.Path, "/functions/v1/"), consulta: r.URL.RawQuery, cuerpo: cuerpo}
		if r.Header.Get("apikey") != "clave-publica" {
			t.Errorf("falta la apikey del relay")
		}
		suma := sha256.Sum256(cuerpo)
		texto := strings.Join([]string{"flujo", r.Method, l.funcion, l.consulta, r.Header.Get("x-qdd-en"), hex.EncodeToString(suma[:])}, "|")
		firma, _ := base64.StdEncoding.DecodeString(r.Header.Get("x-qdd-firma"))
		instalacion, _ := base64.StdEncoding.DecodeString(r.Header.Get("x-qdd-instalacion"))
		if !ed25519.Verify(publica, []byte(texto), firma) || !ed25519.PublicKey(instalacion).Equal(publica) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"firma"}`))
			return
		}
		llamadas = append(llamadas, l)
		codigo, respuesta := responder(l)
		w.WriteHeader(codigo)
		_, _ = w.Write([]byte(respuesta))
	}))
	t.Cleanup(srv.Close)
	return &Cliente{Relay: remoto.ConfigRelay{URL: srv.URL, Clave: "clave-publica"}, Llave: llaveDePrueba(t)}, &llamadas
}

func llaveDePrueba(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	return ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
}

func publicaDePrueba(t *testing.T) ed25519.PublicKey {
	return llaveDePrueba(t).Public().(ed25519.PublicKey)
}

func TestInvitar_FirmaYDevuelveElToken(t *testing.T) {
	c, llamadas := relayFalso(t, publicaDePrueba(t), func(llamada) (int, string) {
		return 201, `{"invitacion":"7c1e0f2a-0000-4000-8000-000000000001","token":"tok"}`
	})
	creada, err := c.Invitar(context.Background(), Nueva{Flujo: "pago", Rol: "cliente", Foto: json.RawMessage(`{"nombre":"Pago"}`), Vence: ahoraDePrueba.Add(72 * time.Hour), Tope: 50}, ahoraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	if creada.Invitacion == "" || creada.Token != "tok" {
		t.Fatalf("la invitación y su token: %+v", creada)
	}
	l := (*llamadas)[0]
	if l.metodo != http.MethodPost || l.funcion != "flujo-invitar" {
		t.Fatalf("POST a flujo-invitar: %+v", l)
	}
	var cuerpo map[string]any
	_ = json.Unmarshal(l.cuerpo, &cuerpo)
	if len(cuerpo) != 5 || cuerpo["vence"] != "2026-11-11T12:00:00Z" || cuerpo["tope"] != float64(50) {
		t.Fatalf("los cinco campos exactos del contrato: %s", l.cuerpo)
	}
}

func TestInvitar_ValidaAntesDeLlamar(t *testing.T) {
	c, llamadas := relayFalso(t, publicaDePrueba(t), func(llamada) (int, string) { return 201, `{}` })
	casos := map[string]Nueva{
		"rol":   {Flujo: "pago", Rol: "dueño", Foto: json.RawMessage(`{}`), Vence: ahoraDePrueba.Add(time.Hour), Tope: 1},
		"vence": {Flujo: "pago", Rol: "cliente", Foto: json.RawMessage(`{}`), Vence: ahoraDePrueba.Add(-time.Hour), Tope: 1},
		"lejos": {Flujo: "pago", Rol: "cliente", Foto: json.RawMessage(`{}`), Vence: ahoraDePrueba.Add(91 * 24 * time.Hour), Tope: 1},
		"tope":  {Flujo: "pago", Rol: "cliente", Foto: json.RawMessage(`{}`), Vence: ahoraDePrueba.Add(time.Hour), Tope: 201},
		"foto":  {Flujo: "pago", Rol: "cliente", Foto: json.RawMessage(`"` + strings.Repeat("x", MaxFoto) + `"`), Vence: ahoraDePrueba.Add(time.Hour), Tope: 1},
	}
	for nombre, n := range casos {
		if _, err := c.Invitar(context.Background(), n, ahoraDePrueba); err == nil {
			t.Fatalf("%s: se rechaza en el PC, sin gastar el tope del relay", nombre)
		}
	}
	if len(*llamadas) != 0 {
		t.Fatalf("no llamó al relay: %d", len(*llamadas))
	}
}

func TestRelay_FirmaMalaYTopeSeDicenClaro(t *testing.T) {
	otra := ed25519.NewKeyFromSeed([]byte(strings.Repeat("o", ed25519.SeedSize))).Public().(ed25519.PublicKey)
	c, _ := relayFalso(t, otra, func(llamada) (int, string) { return 200, `{}` })
	if _, err := c.Propuestas(context.Background(), "", ahoraDePrueba); err == nil || !strings.Contains(err.Error(), "firma") {
		t.Fatalf("una firma que el relay no reconoce lo dice: %v", err)
	}
	c, _ = relayFalso(t, publicaDePrueba(t), func(llamada) (int, string) { return 429, `{"error":"tope"}` })
	_, err := c.Invitar(context.Background(), Nueva{Flujo: "pago", Rol: "cliente", Foto: json.RawMessage(`{}`), Vence: ahoraDePrueba.Add(time.Hour), Tope: 1}, ahoraDePrueba)
	if !errors.Is(err, ErrTope) {
		t.Fatalf("el tope del relay es ErrTope: %v", err)
	}
}

func TestPropuestas_PaginaDesdeLaUltima(t *testing.T) {
	c, llamadas := relayFalso(t, publicaDePrueba(t), func(l llamada) (int, string) {
		if l.consulta == "desde=PRO-7" {
			return 200, paginaDe(8, porPagina)
		}
		return 200, paginaDe(8+porPagina, 3)
	})
	todas, err := c.Propuestas(context.Background(), "PRO-7", ahoraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	if len(todas) != porPagina+3 || todas[0].ID != "PRO-8" {
		t.Fatalf("todas las páginas, en orden: %d, %+v", len(todas), todas[0])
	}
	if (*llamadas)[1].consulta != "desde=PRO-107" {
		t.Fatalf("la segunda página sigue desde la última vista: %q", (*llamadas)[1].consulta)
	}
}

func paginaDe(desde, cuantas int) string {
	var lista []Propuesta
	for i := range cuantas {
		lista = append(lista, Propuesta{ID: "PRO-" + itoa(desde+i), Invitacion: "inv", Flujo: "pago", Rol: "cliente", Autor: "Ana", Operacion: json.RawMessage(`{"op":"quitar","flujo":"pago","nodo":"a"}`), Estado: "pendiente"})
	}
	datos, _ := json.Marshal(map[string]any{"propuestas": lista})
	return string(datos)
}

func itoa(n int) string {
	datos, _ := json.Marshal(n)
	return string(datos)
}

func TestResolver_RepetirEsIgualYLaContrariaNo(t *testing.T) {
	c, llamadas := relayFalso(t, publicaDePrueba(t), func(l llamada) (int, string) {
		if strings.Contains(string(l.cuerpo), "rechazada") {
			return 409, `{"error":"resuelta"}`
		}
		return 200, `{"ok":true}`
	})
	if err := c.Resolver(context.Background(), "PRO-8", Aprobada, "", ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	if string((*llamadas)[0].cuerpo) != `{"propuesta":"PRO-8","estado":"aprobada"}` {
		t.Fatalf("sin motivo, el campo no va: %s", (*llamadas)[0].cuerpo)
	}
	if err := c.Resolver(context.Background(), "PRO-8", Rechazada, "no", ahoraDePrueba); !errors.Is(err, ErrYaResuelta) {
		t.Fatalf("la contraria es ErrYaResuelta: %v", err)
	}
}

func TestSubirFotoYRevocar(t *testing.T) {
	c, llamadas := relayFalso(t, publicaDePrueba(t), func(l llamada) (int, string) {
		if l.metodo == http.MethodPut {
			return 200, `{"invitaciones":2}`
		}
		return 200, `{"ok":true}`
	})
	n, err := c.SubirFoto(context.Background(), "pago", json.RawMessage(`{"nombre":"Pago"}`), ahoraDePrueba)
	if err != nil || n != 2 {
		t.Fatalf("la foto nueva llega a las invitaciones vivas: %d, %v", n, err)
	}
	if err := c.Revocar(context.Background(), "7c1e0f2a-0000-4000-8000-000000000001", ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	if l := (*llamadas)[1]; l.metodo != http.MethodDelete || l.consulta != "invitacion=7c1e0f2a-0000-4000-8000-000000000001" {
		t.Fatalf("DELETE con la invitación en la consulta, firmada: %+v", l)
	}
	if err := c.Revocar(context.Background(), "no-es-uuid", ahoraDePrueba); err == nil {
		t.Fatal("una invitación que no es un uuid no se manda")
	}
}
