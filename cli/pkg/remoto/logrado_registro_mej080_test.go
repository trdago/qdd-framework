package remoto

import (
	"bytes"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-080 (FND-229): con qdd rc.85, QDD Móvil pedía lo logrado y no volvía nada, y el registro del servicio no decía
// nada. Cada pedir_logrado deja una línea con su desenlace (respondido o por qué se ignoró), un teléfono con un campo
// opcional de un tipo inesperado sigue contando como teléfono, y un evento que el canal no pudo entregar se cuenta.

// logSeguro es el registro del servicio en una prueba: lo escribe la goroutine del bucle y lo lee la prueba.
type logSeguro struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *logSeguro) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *logSeguro) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func (l *logSeguro) esperar(t *testing.T, texto string) {
	t.Helper()
	limite := time.Now().Add(5 * time.Second)
	for time.Now().Before(limite) {
		if strings.Contains(l.String(), texto) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("el registro no dice %q:\n%s", texto, l.String())
}

func TestLogrado_CadaPedidoDejaEnElRegistroSiSeRespondio_MEJ080(t *testing.T) {
	homeTemporal(t)
	_, clave := proyectoConLogrado(t, "p", 2, "ok", time.Now())
	log := &logSeguro{}
	_, tr, _, _ := bucleDePrueba(t, func(b *Bucle) { b.Log = log })
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	tr.esperar(t, 10*time.Second, "estado")
	tr.eventos <- pedirLogrado(clave, "s1")
	tr.esperar(t, 10*time.Second, EventoLogrado)
	log.esperar(t, "pedir_logrado "+clave+" (s1): respondido con 2 entradas")
	tr.eventos <- pedirLogrado("no-existe-abc", "s2")
	log.esperar(t, "pedir_logrado no-existe-abc (s2): respondido con 0 entradas, error proyecto")
}

func TestLogrado_UnPedidoIgnoradoDiceElPorQue_MEJ080(t *testing.T) {
	homeTemporal(t)
	_, clave := proyectoConLogrado(t, "p", 1, "ok", time.Now())
	log := &logSeguro{}
	_, tr, _, _ := bucleDePrueba(t, func(b *Bucle) { b.Log = log })
	tr.eventos <- pedirLogrado(clave, "s0")
	log.esperar(t, "pedir_logrado "+clave+" (s0): ignorado, no hay teléfonos en el canal")
	tr.eventos <- telefonoCon("d1", CapacidadLogradoBajoPedido)
	tr.esperar(t, 10*time.Second, "estado")
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: EventoPedirLogrado, Payload: json.RawMessage(`{"proyecto": 3}`)}
	log.esperar(t, "pedir_logrado ignorado: ilegible")
	tr.eventos <- pedirLogrado("Con Mayúsculas", "s1")
	log.esperar(t, `pedir_logrado ignorado: clave de proyecto inválida "Con Mayúsculas"`)
	tr.eventos <- pedirLogrado(clave, strings.Repeat("a", 65))
	log.esperar(t, "pedir_logrado "+clave+": ignorado, solicitud inválida")
}

func TestTelefonos_UnCampoOpcionalDeOtroTipoNoBorraAlTelefono_MEJ080(t *testing.T) {
	m := json.RawMessage(`{"rol":"movil","dispositivo":"d1","mirando":"p-1","version_app":130,"capacidades":["logrado_bajo_pedido",7],"phx_ref":"r"}`)
	got := Telefonos(map[string][]json.RawMessage{"d1": {m}})
	if len(got) != 1 || got[0].Dispositivo != "d1" || got[0].Mirando != "p-1" {
		t.Fatalf("sigue siendo un teléfono mirando: %+v", got)
	}
	if got[0].VersionApp != "130" || len(got[0].Capacidades) != 1 || got[0].Capacidades[0] != CapacidadLogradoBajoPedido {
		t.Fatalf("lo que se entiende de los opcionales se conserva: %+v", got[0])
	}
	pc := json.RawMessage(`{"rol":"pc","version_app":2}`)
	if got := Telefonos(map[string][]json.RawMessage{"pc": {pc}}); len(got) != 0 {
		t.Fatalf("un PC no es un teléfono: %+v", got)
	}
}

func TestCanal_UnEventoQueNoCupoSeCuentaEnElSiguiente_MEJ080(t *testing.T) {
	c := &Canal{eventos: make(chan Evento, 1)}
	c.emitir(Evento{Tipo: TipoBroadcast, Evento: "uno"})
	c.emitir(Evento{Tipo: TipoBroadcast, Evento: "perdido"})
	c.emitir(Evento{Tipo: TipoBroadcast, Evento: "perdido"})
	<-c.eventos
	c.emitir(Evento{Tipo: TipoBroadcast, Evento: "tres"})
	if e := <-c.eventos; e.Evento != "tres" || e.Perdidos != 2 {
		t.Fatalf("el siguiente evento dice cuántos se perdieron antes: %+v", e)
	}
	c.emitir(Evento{Tipo: TipoBroadcast, Evento: "cuatro"})
	if e := <-c.eventos; e.Perdidos != 0 {
		t.Fatalf("y se cuentan una sola vez: %+v", e)
	}
}

func TestBucle_EventosPerdidosQuedanEnElRegistro_MEJ080(t *testing.T) {
	homeTemporal(t)
	log := &logSeguro{}
	_, tr, _, _ := bucleDePrueba(t, func(b *Bucle) { b.Log = log })
	tr.eventos <- Evento{Tipo: TipoBroadcast, Evento: "otro", Perdidos: 3}
	log.esperar(t, "se perdieron 3 eventos del relay")
}
