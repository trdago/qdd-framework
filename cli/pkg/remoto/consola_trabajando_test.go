package remoto

import (
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// FND-239 (pedido desde el teléfono, 2026-11-15: «la consola llega al teléfono hasta 40 s tarde»). Medido: qdd lee y
// manda en menos de 1 s, pero Claude Code escribe cada respuesta en el transcript solo al terminarla (p90 7-14 s, máximo
// 30-50 s). La consola dice desde cuándo Claude está trabajando: la primera api-request del hilo principal lo empieza;
// turn_duration, una interrupción o una compactación lo terminan.

func pedidoAPI(en string) map[string]any {
	return map[string]any{"type": "api-request", "querySource": "repl_main_thread", "timestamp": en}
}

func finDeTurno(en string) map[string]any {
	return map[string]any{"type": "system", "subtype": "turn_duration", "timestamp": en}
}

func TestTrabajando_DesdeLaPrimeraRespuestaHastaQueTerminaElTurno_FND239(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	archivo := escribirTranscript(t, root, "s", turno("user", "hola"), finDeTurno("2026-11-15T10:00:00Z"))
	l, _ := abrirLector(archivo)
	if l.trabajando != "" {
		t.Fatalf("con el turno terminado no trabaja: %q", l.trabajando)
	}
	escribirTranscript(t, root, "s", turno("user", "arregla esto"), pedidoAPI("2026-11-15T10:01:00.000Z"))
	l.leer()
	if l.trabajando != "2026-11-15T10:01:00.000Z" {
		t.Fatalf("empieza con la primera respuesta: %q", l.trabajando)
	}
	escribirTranscript(t, root, "s",
		map[string]any{"type": "user", "isMeta": true, "timestamp": "2026-11-15T10:01:10Z", "message": map[string]any{"content": "texto que inyecta Claude Code"}},
		turno("assistant", []map[string]any{{"type": "tool_use", "name": "Bash"}}),
		turno("user", []map[string]any{{"type": "tool_result", "content": "ok"}}),
		pedidoAPI("2026-11-15T10:01:30.000Z"))
	l.leer()
	if l.trabajando != "2026-11-15T10:01:00.000Z" {
		t.Fatalf("una herramienta y otra respuesta siguen siendo el mismo trabajo: %q", l.trabajando)
	}
	escribirTranscript(t, root, "s", turno("assistant", "Listo."), finDeTurno("2026-11-15T10:02:00Z"))
	l.leer()
	if l.trabajando != "" {
		t.Fatalf("turn_duration lo termina: %q", l.trabajando)
	}
}

func TestTrabajando_UnaInterrupcionOUnaCompactacionLoTerminan_FND239(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	archivo := escribirTranscript(t, root, "s", turno("user", "hola"))
	l, _ := abrirLector(archivo)
	escribirTranscript(t, root, "s", pedidoAPI("2026-11-15T10:01:00Z"), turno("user", "[Request interrupted by user]"))
	l.leer()
	if l.trabajando != "" {
		t.Fatalf("la persona lo interrumpió: %q", l.trabajando)
	}
	escribirTranscript(t, root, "s", pedidoAPI("2026-11-15T10:02:00Z"), map[string]any{"type": "system", "subtype": "compact_boundary", "timestamp": "2026-11-15T10:03:00Z"})
	l.leer()
	if l.trabajando != "" {
		t.Fatalf("una compactación cierra el turno: %q", l.trabajando)
	}
}

func TestTrabajando_UnSubagenteNoCuenta_FND239(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	archivo := escribirTranscript(t, root, "s", turno("user", "hola"))
	l, _ := abrirLector(archivo)
	escribirTranscript(t, root, "s", map[string]any{"type": "api-request", "querySource": "agent:custom", "timestamp": "2026-11-15T10:01:00Z"})
	l.leer()
	if l.trabajando != "" {
		t.Fatalf("solo el hilo principal: %q", l.trabajando)
	}
}

func TestTrabajando_QuienEmpiezaAMirarAMitadDeUnTurnoLoVe_FND239(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	archivo := escribirTranscript(t, root, "s", finDeTurno("2026-11-15T09:00:00Z"), turno("user", "arregla esto"),
		pedidoAPI("2026-11-15T10:01:00Z"), turno("assistant", []map[string]any{{"type": "tool_use", "name": "Bash"}}))
	l, _ := abrirLector(archivo)
	if l.trabajando != "2026-11-15T10:01:00Z" {
		t.Fatalf("la cola inicial también dice si trabaja: %q", l.trabajando)
	}
}

func TestTrabajando_SaleAunqueNoHayaLineasNuevas_FND239(t *testing.T) {
	homeTemporal(t)
	root := proyecto(t, "p", true)
	registro.Anotar(root, time.Now())
	clave, _ := registro.ClaveDe(root)
	escribirTranscript(t, root, "s", turno("user", "hola"), finDeTurno("2026-11-15T10:00:00Z"))
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- telefono("d1", clave)
	if m := consolaDe(t, tr.esperar(t, 2*time.Second, "consola")); m.Trabajando != "" {
		t.Fatalf("sin trabajo no lo dice: %+v", m)
	}
	escribirTranscript(t, root, "s", pedidoAPI("2026-11-15T10:01:00Z"))
	m := consolaDe(t, tr.esperar(t, 2*time.Second, "consola"))
	if m.Trabajando != "2026-11-15T10:01:00Z" || len(m.Lineas) != 0 {
		t.Fatalf("empezar a trabajar sale solo, sin esperar una línea: %+v", m)
	}
	tr.nada(t, 700*time.Millisecond)
	escribirTranscript(t, root, "s", turno("assistant", "Listo."), finDeTurno("2026-11-15T10:02:00Z"))
	m = consolaDe(t, tr.esperar(t, 2*time.Second, "consola"))
	if m.Trabajando != "" || textos(m) != "ia:Listo." {
		t.Fatalf("al terminar sale la respuesta y ya no trabaja: %+v", m)
	}
}
