package remoto

import (
	"context"
	"encoding/json"
	"testing"
)

// Rule: Prohibido usar 'else'
// MEJ-028 (qdd-movile, 2026-11-06; acordado el 2026-11-23): Supabase factura cada broadcast como 1 envío más 1 por cada
// suscriptor que lo recibe, y lo mismo cada evento de presencia que reparte. El PC informaba solo sus envíos: con un
// teléfono, la mitad de lo facturado. Ahora suma lo que se factura de todo lo que pasa por el canal, mandado o
// recibido; la app no informa nada.

func bucleContador(telefonos int) *Bucle {
	b := &Bucle{QDD: "v-prueba", Cred: Credenciales{Topico: topicoPrueba, Firma: "f"}, T: nuevoTransporte(), R: nuevoRelayBucle()}
	for i := 0; i < telefonos; i++ {
		b.telefonos = append(b.telefonos, Telefono{Rol: "movil"})
	}
	return b
}

func TestMensajes_UnBroadcastDelPCCuentaAlQueLoMandaYACadaTelefono_MEJ028(t *testing.T) {
	b := bucleContador(2)
	if !b.broadcast(context.Background(), "prueba", map[string]string{}) {
		t.Fatal("la prueba necesita que el envío salga")
	}
	if b.mensajes != 3 {
		t.Fatalf("1 envío + 2 teléfonos que lo reciben = 3; contó %d", b.mensajes)
	}
}

func TestMensajes_LoQueMandaUnTelefonoCuentaAlQueLoMandaYALosDemas_MEJ028(t *testing.T) {
	b := bucleContador(2)
	payload, _ := json.Marshal(map[string]string{})
	b.alEvento(context.Background(), Evento{Tipo: TipoBroadcast, Evento: "evento-que-el-pc-no-usa", Payload: payload})
	if b.mensajes != 3 {
		t.Fatalf("1 envío del teléfono + el PC + el otro teléfono = 3; contó %d", b.mensajes)
	}
}

func TestMensajes_ElTrackDelPCCuentaSoloSuEnvio_MEJ028(t *testing.T) {
	b := bucleContador(2)
	b.track(context.Background())
	if b.mensajes != 1 {
		t.Fatalf("el track es 1 envío; el reparto llega como evento de presencia y se cuenta al recibirlo: %d", b.mensajes)
	}
}

func TestMensajes_UnEventoDePresenciaCuentaACadaSuscrito_MEJ028(t *testing.T) {
	b := bucleContador(1)
	b.alEvento(context.Background(), sinTelefonos())
	if b.mensajes != 1 {
		t.Fatalf("se fue el teléfono: el reparto llega solo al PC = 1; contó %d", b.mensajes)
	}
}
