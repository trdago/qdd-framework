package cmd

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/invitaciones"
	"github.com/qdd-framework/qdd/pkg/remoto"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 3: compartir con --rol crea la invitación en el relay y deja su token solo dentro del archivo; cada
// propuesta llega al panel como un pedido de la persona, y solo ella la aprueba. Las pruebas usan un relay falso (TLS
// local): nunca el servicio real.

type relayFalsoDeFlujo struct {
	mu         sync.Mutex
	invitadas  []map[string]any
	propuestas []invitaciones.Propuesta
	resueltas  map[string]string
}

func usarRelayFalso(t *testing.T) *relayFalsoDeFlujo {
	t.Helper()
	r := &relayFalsoDeFlujo{resueltas: map[string]string{}}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		defer r.mu.Unlock()
		cuerpo, _ := io.ReadAll(req.Body)
		funcion := strings.TrimPrefix(req.URL.Path, "/functions/v1/")
		if funcion == "flujo-invitar" && req.Method == http.MethodPost {
			var m map[string]any
			_ = json.Unmarshal(cuerpo, &m)
			r.invitadas = append(r.invitadas, m)
			w.WriteHeader(201)
			_, _ = w.Write([]byte(`{"invitacion":"7c1e0f2a-0000-4000-8000-000000000001","token":"token-de-prueba-123"}`))
			return
		}
		if funcion == "flujo-propuestas" && req.Method == http.MethodGet {
			datos, _ := json.Marshal(map[string]any{"propuestas": r.propuestas})
			_, _ = w.Write(datos)
			return
		}
		if funcion == "flujo-propuestas" && req.Method == http.MethodPost {
			var m map[string]string
			_ = json.Unmarshal(cuerpo, &m)
			r.resueltas[m["propuesta"]] = m["estado"]
		}
		_, _ = w.Write([]byte(`{"ok":true,"invitaciones":1}`))
	}))
	t.Cleanup(srv.Close)
	anterior := clienteDeInvitaciones
	clienteDeInvitaciones = func(context.Context) (*invitaciones.Cliente, error) {
		return &invitaciones.Cliente{Relay: remoto.ConfigRelay{URL: srv.URL, Clave: "publica"}, Llave: ed25519.NewKeyFromSeed(make([]byte, 32)), HTTP: srv.Client()}, nil
	}
	t.Cleanup(func() { clienteDeInvitaciones = anterior })
	return r
}

func TestFlujo_CompartirConRolCreaLaInvitacionYElTokenSoloVaEnElArchivo(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := proyectoConFlujo(t)
	relay := usarRelayFalso(t)
	salida := correrFlujo(t, flujoCompartirCmd, map[string]string{"para": "Ana", "rol": "cliente", "dias": "7"}, "pago")
	datos, err := os.ReadFile(filepath.Join(root, "tmp", "flujo-pago-ana.html"))
	if err != nil {
		t.Fatalf("el archivo: %v (%s)", err, salida)
	}
	if !strings.Contains(string(datos), "token-de-prueba-123") || !strings.Contains(string(datos), "connect-src https://127.0.0.1") {
		t.Fatal("el archivo lleva el token y solo se conecta con el relay")
	}
	if strings.Contains(salida, "token-de-prueba-123") {
		t.Fatal("el token no se imprime: solo vive en el archivo")
	}
	if len(relay.invitadas) != 1 || relay.invitadas[0]["rol"] != "cliente" {
		t.Fatalf("una invitación con su rol: %+v", relay.invitadas)
	}
	l, _ := invitaciones.Leer(root)
	if len(l.Invitaciones) != 1 || l.Invitaciones[0].Para != "Ana" {
		t.Fatalf("el PC anota la invitación, sin el token: %+v", l)
	}
	local, _ := json.Marshal(l)
	if strings.Contains(string(local), "token-de-prueba-123") {
		t.Fatal("el token no queda en el PC")
	}
}

func TestFlujo_AprobarLaAutorizaSoloLaPersona(t *testing.T) {
	confirmado := func(pedido, comando string) bool {
		return pedido == "#9" && comando == invitaciones.ComandoAprobar("PRO-1")
	}
	si := func(string) bool { return true }
	if err := autorizarPropuesta("PRO-1", "#9", nil, confirmado); err != nil {
		t.Fatalf("con el pedido que la persona confirmó: %v", err)
	}
	if err := autorizarPropuesta("PRO-1", "", si, confirmado); err != nil {
		t.Fatalf("con su «sí» en la terminal: %v", err)
	}
	if err := autorizarPropuesta("PRO-1", "#8", si, confirmado); err == nil {
		t.Fatal("un pedido que no es el suyo no autoriza, aunque haya terminal")
	}
	if err := autorizarPropuesta("PRO-1", "", nil, confirmado); err == nil {
		t.Fatal("una IA sin pedido ni terminal no aprueba")
	}
}

func TestFlujo_CicloDejaLaPropuestaEnElPanelYAprobarLaAplica(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := proyectoConFlujo(t)
	relay := usarRelayFalso(t)
	correrFlujo(t, flujoCompartirCmd, map[string]string{"para": "Ana", "rol": "cliente"}, "pago")
	f, _ := flujo.Cargar(root, "pago")
	relay.propuestas = []invitaciones.Propuesta{{ID: "PRO-1", Invitacion: "7c1e0f2a-0000-4000-8000-000000000001", Flujo: "pago", Rol: "cliente", Autor: "Ana",
		Operacion: json.RawMessage(`{"op":"agregar","flujo":"pago","nodo":"` + f.Nodos[0].ID + `","nombre":"Revisa el RUT"}`), Estado: invitaciones.Pendiente}}
	res, err := cicloDePropuestas(context.Background(), root, time.Now())
	if err != nil || res.Pedidas != 1 {
		t.Fatalf("una propuesta, un pedido: %+v %v", res, err)
	}
	l, _ := invitaciones.Leer(root)
	if d := describirPropuesta(root, l.Propuestas[0]); !strings.Contains(d, "Revisa el RUT") {
		t.Fatalf("la lista dice qué propone, aunque el archivo local guarde la operación con sangría: %q", d)
	}
	pedido := pedidoDe(t, root, invitaciones.ComandoAprobar("PRO-1"))
	if !pedido.DePersona || !strings.Contains(pedido.Motivo, "Ana") {
		t.Fatalf("un pedido que solo confirma la persona, y dice quién propone: %+v", pedido)
	}
	linea, err := aprobarPropuesta(context.Background(), root, "PRO-1", time.Now())
	if err != nil || !strings.Contains(linea, "Revisa el RUT") {
		t.Fatalf("aprobada: %q %v", linea, err)
	}
	if relay.resueltas["PRO-1"] != invitaciones.Aprobada {
		t.Fatalf("el relay se entera: %v", relay.resueltas)
	}
}

func pedidoDe(t *testing.T, root, comando string) consola.Pedido {
	t.Helper()
	for _, p := range consola.PedidosA(root, time.Now()) {
		if p.Comando == comando {
			return p
		}
	}
	t.Fatalf("no hay un pedido «%s» en el panel", comando)
	return consola.Pedido{}
}
