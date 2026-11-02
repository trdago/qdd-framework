package remoto

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0013: prueba contra el relay de desarrollo de verdad. Corre solo con QDD_REMOTO_RELAY_PRUEBA=1 y con la URL y la
// clave publicable en QDD_REMOTO_RELAY_URL y QDD_REMOTO_RELAY_CLAVE (nunca en el repositorio); el CI no la corre.
// Sin un teléfono no se puede aceptar un código: prueba lo que el PC hace solo.

func relayVivo(t *testing.T) ConfigRelay {
	t.Helper()
	if os.Getenv("QDD_REMOTO_RELAY_PRUEBA") != "1" {
		t.Skip("prueba contra el relay real: QDD_REMOTO_RELAY_PRUEBA=1")
	}
	c := ConfigRelay{URL: strings.TrimRight(os.Getenv("QDD_REMOTO_RELAY_URL"), "/"), Clave: os.Getenv("QDD_REMOTO_RELAY_CLAVE")}
	if err := ValidarClave(c.Clave); err != nil {
		t.Fatalf("la clave de la prueba tiene que ser la publicable: %v", err)
	}
	if err := ValidarURL(c.URL); err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRelayVivo_LoQueElPCHaceSolo(t *testing.T) {
	c := relayVivo(t)
	r := NuevoRelay(c)
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	o, err := NuevaOferta("qdd-prueba-" + time.Now().Format("150405"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := r.Latido(ctx, o.Cred.Topico, o.Cred.Firma, 0, 0, 0); !errors.Is(err, ErrCredenciales) {
		t.Fatalf("una máquina que no existe no late (28000): %v", err)
	}
	if err := r.Ofrecer(ctx, HashCodigo(o.Codigo), o.Cred.Nombre, o.Cred.Topico, o.Cred.Firma); err != nil {
		t.Fatalf("ofrecer con el formato del relay: %v", err)
	}
	id, err := r.Emparejada(ctx, HashCodigo(o.Codigo))
	if err != nil || id != "" {
		t.Fatalf("sin teléfono, todavía no está emparejada: %q %v", id, err)
	}
	if _, err := r.Dispositivos(ctx, o.Cred.Topico, o.Cred.Firma); !errors.Is(err, ErrCredenciales) {
		t.Fatalf("dispositivos exige canal y firma: %v", err)
	}
	err = r.Avisar(ctx, o.Cred.Topico, o.Cred.Firma, AvisoPedido, "qdd", "qdd-0123")
	if err == nil {
		t.Fatal("avisar con una máquina que no existe no se acepta")
	}
	t.Logf("avisar con una máquina inexistente: %v", err)

	canal := NuevoCanal(ConfigCanal{Relay: c.URL, Clave: c.Clave, Topico: o.Cred.Topico, PresenceKey: "pc"})
	go canal.Correr(ctx)
	e := esperarEvento(t, canal.Eventos(), 30*time.Second, func(e Evento) bool { return e.Tipo == TipoDesconectado || e.Tipo == TipoConectado })
	if e.Tipo != TipoDesconectado || !strings.Contains(e.Motivo, "Unauthorized") {
		t.Fatalf("el canal de una máquina que no existe no admite al PC: %+v", e)
	}
}
