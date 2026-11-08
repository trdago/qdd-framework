package consola

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0023: el permiso de usar un token del baúl lo da la persona con su «Sí». Ese pedido no lo lanza el modo goal, y
// el comando sabe qué pedido lo corre para comprobar que la persona lo confirmó.

func TestPedidoDePersona_ElModoGoalNoLoLanza(t *testing.T) {
	root := conGoal(t)
	if _, err := PedirEnOrden(root, "echo permiso", "🔑 Permitir cloudflare", root, Orden{DePersona: true}, enCola); err != nil {
		t.Fatal(err)
	}
	if elegidas := correrPedidos(root, "/no/existe", "s1", enCola); len(elegidas) != 0 {
		t.Fatalf("un permiso del baúl lo da la persona, también con el modo goal: %+v", elegidas)
	}
	a, _ := BuscarAccion(Acciones(root, false), "pedido:1")
	if !a.DePersona {
		t.Fatalf("el panel sabe que es de la persona: %+v", a)
	}
}

func TestPedido_ElComandoRecibeElIDDeSuPedido(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("usa sh")
	}
	root := proyectoConCola(t)
	marca := filepath.Join(root, "id.txt")
	p := mustPedir(t, root, "printf %s \"$"+VarPedido+"\" > "+marca, "Probar", root)
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+p.ID)
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 0 {
		t.Fatalf("código %d: %s", codigo, salida.String())
	}
	b, _ := os.ReadFile(marca)
	if strings.TrimSpace(string(b)) != p.ID {
		t.Fatalf("el comando ve %s=%q, se esperaba %q", VarPedido, b, p.ID)
	}
}

func TestPedidoConfirmado_SoloElDeLaPersonaAceptadoConSuComando(t *testing.T) {
	root := proyectoConCola(t)
	p, err := PedirEnOrden(root, "qdd baul permitir 1", "🔑 Permitir", root, Orden{DePersona: true}, enCola)
	if err != nil {
		t.Fatal(err)
	}
	if PedidoConfirmado(root, p.ID, "qdd baul permitir 1", enCola) {
		t.Fatal("sin el «Sí» no está confirmado")
	}
	if _, err := aceptar(root, p.ID, "s1", enCola); err != nil {
		t.Fatal(err)
	}
	if !PedidoConfirmado(root, p.ID, "qdd baul permitir 1", enCola) {
		t.Fatal("aceptado por la persona, está confirmado")
	}
	if PedidoConfirmado(root, p.ID, "qdd baul permitir 2", enCola) {
		t.Fatal("con otro comando no vale")
	}
	comun := mustPedir(t, root, "qdd baul permitir 3", "Otro", root)
	if _, err := aceptar(root, comun.ID, "s1", enCola); err != nil {
		t.Fatal(err)
	}
	if PedidoConfirmado(root, comun.ID, "qdd baul permitir 3", enCola) {
		t.Fatal("un pedido común de la IA no vale como permiso: el modo goal los acepta solo")
	}
}

func TestSesionLate_LaQueLatioYNoLaCerrada(t *testing.T) {
	root := proyectoConCola(t)
	if SesionLate(root, "s1", enCola) {
		t.Fatal("sin latido no está viva")
	}
	if err := LatirSesion(root, "s1", enCola); err != nil {
		t.Fatal(err)
	}
	if !SesionLate(root, "s1", enCola) || SesionLate(root, "s2", enCola) {
		t.Fatal("viva solo la que latió")
	}
	if SesionLate(root, "s1", enCola.Add(VidaSesion*2)) {
		t.Fatal("deja de estar viva si no late")
	}
	if err := CerrarSesion(root, "s1", enCola); err != nil {
		t.Fatal(err)
	}
	if SesionLate(root, "s1", enCola) {
		t.Fatal("cerrada no está viva")
	}
}
