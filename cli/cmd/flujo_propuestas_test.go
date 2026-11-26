package cmd

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
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

// aprobarComoIA corre «qdd flujo propuestas aprobar» con este binario como qdd, sin terminal (stdin vacío) y con las
// variables extra; sin QDD_PEDIDO salvo que se pase. Devuelve el código de salida y lo que dijo.
func aprobarComoIA(t *testing.T, root, id string, extra ...string) (int, string) {
	t.Helper()
	c := exec.Command(os.Args[0], "flujo", "propuestas", "aprobar", id)
	c.Dir = root
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, consola.VarPedido+"=") {
			c.Env = append(c.Env, v)
		}
	}
	c.Env = append(c.Env, extra...)
	salida, err := c.CombinedOutput()
	if err == nil {
		return 0, string(salida)
	}
	var salio *exec.ExitError
	if !errors.As(err, &salio) {
		t.Fatalf("no se pudo correr qdd: %v", err)
	}
	return salio.ExitCode(), string(salida)
}

// contrato: flujo#ac4c19a3
// Caja negra con el binario: una IA (sin terminal) que corre «qdd flujo propuestas aprobar» por su cuenta, o con un
// QDD_PEDIDO que la persona no confirmó, sale con error y el flujo no cambia. El relay de la máquina no responde
// (127.0.0.1:1): si la puerta faltara, la propuesta se integraría igual en local y la prueba lo vería.
func TestFlujo_AprobarSinElSiDeLaPersonaNoTocaElFlujo(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	root := proyectoConFlujo(t)
	relay := usarRelayFalso(t)
	correrFlujo(t, flujoCompartirCmd, map[string]string{"para": "Ana", "rol": "cliente"}, "pago")
	f, _ := flujo.Cargar(root, "pago")
	relay.propuestas = []invitaciones.Propuesta{{ID: "PRO-1", Invitacion: "7c1e0f2a-0000-4000-8000-000000000001", Flujo: "pago", Rol: "cliente", Autor: "Ana",
		Operacion: json.RawMessage(`{"op":"agregar","flujo":"pago","nodo":"` + f.Nodos[0].ID + `","nombre":"Revisa el RUT"}`), Estado: invitaciones.Pendiente}}
	if res, err := cicloDePropuestas(context.Background(), root, time.Now()); err != nil || res.Pedidas != 1 {
		t.Fatalf("una propuesta, un pedido: %+v %v", res, err)
	}
	relayMuerto := filepath.Join(home, "remoto", "relay.json")
	if err := os.MkdirAll(filepath.Dir(relayMuerto), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relayMuerto, []byte(`{"url":"http://127.0.0.1:1","clave":"sb_publishable_prueba"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	antes, err := flujo.Cargar(root, "pago")
	if err != nil {
		t.Fatal(err)
	}
	pedido := pedidoDe(t, root, invitaciones.ComandoAprobar("PRO-1"))
	for nombre, extra := range map[string][]string{
		"sin pedido":                  nil,
		"con un pedido sin confirmar": {consola.VarPedido + "=" + pedido.ID},
	} {
		codigo, salida := aprobarComoIA(t, root, "PRO-1", extra...)
		if codigo == 0 || !strings.Contains(salida, "la aprueba la persona") {
			t.Fatalf("%s: una IA no aprueba una propuesta: %d %q", nombre, codigo, salida)
		}
		despues, err := flujo.Cargar(root, "pago")
		if err != nil || despues.Revision != antes.Revision || len(despues.Nodos) != len(antes.Nodos) {
			t.Fatalf("%s: el flujo queda como estaba: rev %d → %d, nodos %d → %d (%v)", nombre, antes.Revision, despues.Revision, len(antes.Nodos), len(despues.Nodos), err)
		}
		if l, _ := invitaciones.Leer(root); l.Propuestas[0].Situacion == invitaciones.Aprobada {
			t.Fatalf("%s: la propuesta sigue esperando a la persona: %+v", nombre, l.Propuestas[0])
		}
	}
}

// contrato: flujo#565f2a08
// Con el modo goal activo, la propuesta de un invitado sigue siendo de la persona: el ciclo la deja como pedido 📝,
// una vuelta del modo goal no lo lanza ni lo registra como elección, y el flujo no cambia.
func TestFlujo_ElModoGoalNoApruebaUnaPropuesta(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := proyectoConFlujo(t)
	ahora := time.Now()
	if _, err := estado.Nueva(root, estado.CambioNueva{Nombre: "Pago con RUT"}, ahora); err != nil {
		t.Fatal(err)
	}
	if err := consola.LatirSesion(root, "s1", ahora); err != nil {
		t.Fatal(err)
	}
	if err := consola.ActivarGoal(root, "panel", ahora); err != nil {
		t.Fatal(err)
	}
	relay := usarRelayFalso(t)
	correrFlujo(t, flujoCompartirCmd, map[string]string{"para": "Ana", "rol": "cliente"}, "pago")
	antes, _ := flujo.Cargar(root, "pago")
	relay.propuestas = []invitaciones.Propuesta{{ID: "PRO-1", Invitacion: "7c1e0f2a-0000-4000-8000-000000000001", Flujo: "pago", Rol: "cliente", Autor: "Ana",
		Operacion: json.RawMessage(`{"op":"agregar","flujo":"pago","nodo":"` + antes.Nodos[0].ID + `","nombre":"Revisa el RUT"}`), Estado: invitaciones.Pendiente}}
	if res, err := cicloDePropuestas(context.Background(), root, ahora); err != nil || res.Pedidas != 1 {
		t.Fatalf("una propuesta, un pedido: %+v %v", res, err)
	}
	r, err := consola.CicloGoal(root, "true", consola.EntradaGoal{Sesion: "s1", Silencio: time.Hour}, ahora)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range r.Elegidas {
		if e.Tipo == "pedido" {
			t.Fatalf("el modo goal no elige el pedido de una propuesta: %+v", r.Elegidas)
		}
	}
	if g := consola.VistaGoal(root, ahora); !g.Activo || len(g.Elegidas) != 0 {
		t.Fatalf("el modo sigue activo y sin elecciones: %+v", g)
	}
	pedido := pedidoDe(t, root, invitaciones.ComandoAprobar("PRO-1"))
	if !pedido.DePersona || pedido.Estado != consola.EstadoPendiente || pedido.Confirmado != nil {
		t.Fatalf("el pedido 📝 sigue esperando el «s  Sí» de la persona: %+v", pedido)
	}
	despues, _ := flujo.Cargar(root, "pago")
	if despues.Revision != antes.Revision || len(despues.Nodos) != len(antes.Nodos) {
		t.Fatalf("el flujo no cambia: rev %d → %d", antes.Revision, despues.Revision)
	}
	if l, _ := invitaciones.Leer(root); l.Propuestas[0].Situacion == invitaciones.Aprobada || relay.resueltas["PRO-1"] != "" {
		t.Fatalf("la propuesta sigue sin resolver, también en el relay: %+v %v", l.Propuestas[0], relay.resueltas)
	}
}
