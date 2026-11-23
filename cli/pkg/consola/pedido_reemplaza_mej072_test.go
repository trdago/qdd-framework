package consola

import (
	"io"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// MEJ-072 (qdd-movile, 2026-11-10): [d] esperaba a [a]; [a] falló y la IA lo pidió de nuevo corregido. El nuevo salió
// con otra tecla y [d] siguió esperando al que falló, para siempre; reencadenarlo con antes=d se rechazaba porque [d]
// estaba armado. La persona tuvo que descartar y confirmar todo otra vez, 4 veces en una hora.

func pedidoQueFalla(t *testing.T, root, comando string) Pedido {
	t.Helper()
	p := mustPedirEnOrden(t, root, comando, Orden{})
	if err := marcarEjecutando(root, p.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if codigo := CorrerLanzado(root, p.ID, io.Discard); codigo == 0 {
		t.Fatalf("la prueba necesita un pedido que falle: %q", comando)
	}
	return p
}

func pedidoEnCola(t *testing.T, root, id string) (Pedido, bool) {
	t.Helper()
	for _, p := range leerCola(root) {
		if p.ID == id {
			return p, true
		}
	}
	return Pedido{}, false
}

func TestPedirDeNuevo_ReemplazaAlQueFalloYLaCadenaLoEspera_MEJ072(t *testing.T) {
	root := proyectoConCola(t)
	push := pedidoQueFalla(t, root, "exit 3 # push")
	deploy := mustPedirEnOrden(t, root, "exit 0 # deploy", Orden{Despues: []string{push.ID}})
	if estado, err := aceptar(root, deploy.ID, "", enCola); err != nil || estado != EstadoArmado {
		t.Fatalf("la persona confirmó el deploy mientras espera: %q %v", estado, err)
	}
	nuevo := mustPedirEnOrden(t, root, "exit 0 # push corregido", Orden{DeNuevo: true, Reemplaza: push.Tecla})
	if nuevo.Tecla != push.Tecla {
		t.Fatalf("el nuevo conserva el lugar del que falló: tecla %q, era %q", nuevo.Tecla, push.Tecla)
	}
	if _, sigue := pedidoEnCola(t, root, push.ID); sigue {
		t.Fatal("el que falló sale de la cola: lo reemplazó el nuevo")
	}
	d, _ := pedidoEnCola(t, root, deploy.ID)
	if d.Estado != EstadoArmado {
		t.Fatalf("lo que la persona confirmó sigue confirmado: %+v", d)
	}
	if strings.Join(d.Despues, ",") != nuevo.ID {
		t.Fatalf("el deploy espera al nuevo, no al que falló: %v", d.Despues)
	}
}

func TestPedirDeNuevo_ReemplazaConservaLoQueElViejoEsperaba_MEJ072(t *testing.T) {
	root := proyectoConCola(t)
	tag := mustPedirEnOrden(t, root, "exit 0 # tag", Orden{})
	push := mustPedirEnOrden(t, root, "exit 3 # push", Orden{Despues: []string{tag.ID}})
	nuevo := mustPedirEnOrden(t, root, "exit 0 # push corregido", Orden{DeNuevo: true, Reemplaza: push.ID})
	n, _ := pedidoEnCola(t, root, nuevo.ID)
	if strings.Join(n.Despues, ",") != tag.ID {
		t.Fatalf("el nuevo espera lo mismo que esperaba el reemplazado: %v", n.Despues)
	}
}

func TestPedirDeNuevo_SoloReemplazaLoQueNoCorreNiEspera_MEJ072(t *testing.T) {
	root := proyectoConCola(t)
	tag := mustPedirEnOrden(t, root, "exit 0 # tag", Orden{})
	push := mustPedirEnOrden(t, root, "exit 0 # push", Orden{Despues: []string{tag.ID}})
	if _, err := aceptar(root, push.ID, "", enCola); err != nil {
		t.Fatal(err)
	}
	if _, err := PedirEnOrden(root, "exit 0 # otro push", "m", "", Orden{Reemplaza: push.Tecla}, enCola); err == nil || !strings.Contains(err.Error(), "armado") {
		t.Fatalf("un pedido armado tiene la confirmación de la persona: no se reemplaza: %v", err)
	}
	if _, err := PedirEnOrden(root, "exit 0 # otro push", "m", "", Orden{Reemplaza: "z"}, enCola); err == nil {
		t.Fatal("reemplazar algo que no está en la cola es un error")
	}
	p, _ := pedidoEnCola(t, root, push.ID)
	if p.Estado != EstadoArmado {
		t.Fatalf("lo rechazado no toca la cola: %+v", p)
	}
}

func TestPedir_AntesDeUnArmado_LoDejaArmadoYEsperando_MEJ072(t *testing.T) {
	root := proyectoConCola(t)
	tag := mustPedirEnOrden(t, root, "exit 0 # tag", Orden{})
	deploy := mustPedirEnOrden(t, root, "exit 0 # deploy", Orden{Despues: []string{tag.ID}})
	if _, err := aceptar(root, deploy.ID, "", enCola); err != nil {
		t.Fatal(err)
	}
	migrar, err := PedirEnOrden(root, "exit 0 # migrar", "m", "", Orden{Antes: []string{deploy.Tecla}}, enCola)
	if err != nil {
		t.Fatalf("encadenar antes de un armado se acepta: agrega una espera, no cambia lo que la persona confirmó: %v", err)
	}
	d, _ := pedidoEnCola(t, root, deploy.ID)
	if d.Estado != EstadoArmado || !strings.Contains(strings.Join(d.Despues, ","), migrar.ID) {
		t.Fatalf("sigue armado y ahora espera también al nuevo: %+v", d)
	}
}
