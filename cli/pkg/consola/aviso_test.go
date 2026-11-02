package consola

import (
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Aviso al chat (pedido del usuario, 2026-11-02: «el chat principal tiene que enterarse… cuando acepto la tarea en el
// panel»): la sesión de Claude Code que confirmó un pedido recibe su resultado una sola vez, aunque haya varias abiertas
// o el mod se recargue. qdd guarda qué sesión lo confirmó y marca «avisado» con la cola bloqueada.

func pedidoEnPanel(t *testing.T, root, id string) Accion {
	t.Helper()
	for _, a := range accionesDePedidos(root) {
		if a.pedido == id {
			return a
		}
	}
	t.Fatalf("el pedido %s no está en el panel: %+v", id, Pedidos(root))
	return Accion{}
}

func aceptadoPor(t *testing.T, root, comando, motivo, sesion string) Pedido {
	t.Helper()
	p := mustPedir(t, root, comando, motivo, "")
	if _, err := aceptar(root, p.ID, sesion, time.Now()); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestAviso_HechoLoRecibeSoloLaSesionQueLoConfirmoYUnaVez(t *testing.T) {
	root := proyectoConCola(t)
	p := aceptadoPor(t, root, "echo publicado", "Publicar rc.16", "sesion-1")
	if a := pedidoEnPanel(t, root, p.ID); a.Avisar || a.Sesion != "sesion-1" {
		t.Fatalf("mientras corre no se avisa, pero se sabe quién lo confirmó: %+v", a)
	}
	terminar(root, p.ID, 0, "publicado", true)
	a := pedidoEnPanel(t, root, p.ID)
	if !a.Avisar || a.Sesion != "sesion-1" {
		t.Fatalf("al terminar, la sesión que lo confirmó tiene un aviso: %+v", a)
	}
	for _, parte := range []string{"✓", "[" + p.Tecla + "]", "«Publicar rc.16»", "exit 0", "publicado", "Sigue"} {
		if !strings.Contains(a.Mensaje, parte) {
			t.Fatalf("el mensaje para la IA dice %q: %q", parte, a.Mensaje)
		}
	}
	if err := Avisado(root, p.ID, "sesion-2"); err == nil {
		t.Fatal("otra sesión no lo reclama: no lo confirmó ella")
	}
	if err := Avisado(root, prefijoPedido+p.ID, "sesion-1"); err != nil {
		t.Fatalf("la que lo confirmó lo reclama (con o sin prefijo): %v", err)
	}
	if err := Avisado(root, p.ID, "sesion-1"); err == nil {
		t.Fatal("una segunda vez no: el aviso llega una sola vez (un reload del mod no lo repite)")
	}
	if a := pedidoEnPanel(t, root, p.ID); a.Avisar || a.Mensaje != "" {
		t.Fatalf("ya avisado, no se ofrece de nuevo: %+v", a)
	}
}

func TestAviso_UnaFallaDiceElCodigoYLaSalida(t *testing.T) {
	root := proyectoConCola(t)
	p := aceptadoPor(t, root, "exit 3", "Mergear #64", "s")
	terminar(root, p.ID, 3, "conflicto en main", true)
	a := pedidoEnPanel(t, root, p.ID)
	for _, parte := range []string{"✗", "«Mergear #64»", "falló (exit 3)", "conflicto en main"} {
		if !a.Avisar || !strings.Contains(a.Mensaje, parte) {
			t.Fatalf("una falla avisa con %q: %+v", parte, a)
		}
	}
}

func TestAviso_SinSesionNoHayAviso(t *testing.T) {
	root := proyectoConCola(t)
	p := aceptadoPor(t, root, "exit 0", "m", "")
	terminar(root, p.ID, 0, "", true)
	if a := pedidoEnPanel(t, root, p.ID); a.Avisar || a.Mensaje != "" {
		t.Fatalf("un panel que no dice su sesión (OpenCode, uno anterior) no recibe aviso: %+v", a)
	}
}

func TestAviso_TodaviaNoArmadoNoAvisa(t *testing.T) {
	root := proyectoConCola(t)
	p := aceptadoPor(t, root, "exit 75", "m", "s")
	terminar(root, p.ID, CodigoTodaviaNo, "falta el tag", true)
	if a := pedidoEnPanel(t, root, p.ID); a.Estado != EstadoArmado || a.Avisar {
		t.Fatalf("armado no terminó: avisa cuando corra de verdad: %+v", a)
	}
}

func TestAviso_UnHechoSinAvisarNoSaleConElCicloHastaAvisarse(t *testing.T) {
	root := proyectoConCola(t)
	p := aceptadoPor(t, root, "exit 0", "m", "s")
	terminar(root, p.ID, 0, "ok", true)
	if err := MarcarCiclo(root, "auto", time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if a := pedidoEnPanel(t, root, p.ID); !a.Avisar {
		t.Fatalf("un mensaje de la persona no se lleva el aviso pendiente: %+v", a)
	}
	if err := Avisado(root, p.ID, "s"); err != nil {
		t.Fatal(err)
	}
	if cola := Pedidos(root); len(cola) != 0 {
		t.Fatalf("avisado, sale del tablero como cualquier hecho ya visto: %+v", cola)
	}
}

func TestAviso_ReintentarloVuelveAAvisarALaQueLoReintenta(t *testing.T) {
	root := proyectoConCola(t)
	p := aceptadoPor(t, root, "exit 3", "m", "s1")
	terminar(root, p.ID, 3, "falló", true)
	if err := Avisado(root, p.ID, "s1"); err != nil {
		t.Fatal(err)
	}
	if _, err := aceptar(root, p.ID, "s2", time.Now()); err != nil {
		t.Fatal(err)
	}
	terminar(root, p.ID, 0, "ok", true)
	if a := pedidoEnPanel(t, root, p.ID); !a.Avisar || a.Sesion != "s2" {
		t.Fatalf("el reintento es otra confirmación: avisa a quien lo reintentó: %+v", a)
	}
}

func TestAviso_ElMensajeEsCortoAunqueLaSalidaNoLoSea(t *testing.T) {
	root := proyectoConCola(t)
	p := aceptadoPor(t, root, "exit 3", "m", "s")
	larga := strings.Repeat("x", 3000) + "\nel final importa"
	terminar(root, p.ID, 3, larga, true)
	a := pedidoEnPanel(t, root, p.ID)
	if n := len([]rune(a.Mensaje)); n > MaxMensajeAviso || !strings.Contains(a.Mensaje, "el final importa") {
		t.Fatalf("cabe en %d y conserva el final de la salida: %d %q", MaxMensajeAviso, n, a.Mensaje)
	}
}
