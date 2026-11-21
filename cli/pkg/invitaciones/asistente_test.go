package invitaciones

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/flujo"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 4: lo que propone el asistente entra a la misma cola que lo de los invitados. Es un pedido de la
// persona en el panel, de a tres, y nada llega al relay: el asistente está en el PC.

func TestAsistente_SuPropuestaEsUnPedidoDeLaPersonaConSuMinuto(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	id, err := ProponerDeAsistente(e.Root, json.RawMessage(`{"op":"historia","flujo":"pago","nodo":"b","quiero":"validar el RUT"}`), "12:40", ahoraDePrueba)
	if err != nil || !strings.HasPrefix(id, PrefijoAsistente) {
		t.Fatalf("queda en la cola con su id: %q %v", id, err)
	}
	if !TocaCiclo(e.Root, ahoraDePrueba, 0) {
		t.Fatal("el panel tiene algo que pedir aunque no haya invitaciones")
	}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(p.pedidos) != 1 || p.pedidos["p1"] != ComandoAprobar(id) {
		t.Fatalf("un pedido para aprobarla: %v", p.pedidos)
	}
	if !strings.HasPrefix(p.motivos[0], "🎙 asistente (12:40): ") || !strings.Contains(p.motivos[0], "validar el RUT") {
		t.Fatalf("el motivo dice de qué minuto salió: %q", p.motivos[0])
	}
	otra, err := ProponerDeAsistente(e.Root, json.RawMessage(`{"op":"historia","flujo":"pago","nodo":"b","quiero":"validar el RUT"}`), "12:41", ahoraDePrueba)
	if err != nil || otra != id {
		t.Fatalf("la misma propuesta sin resolver no se repite: %q %q %v", otra, id, err)
	}
	if len(r.resueltas) != 0 {
		t.Fatalf("el relay no sabe nada del asistente: %v", r.resueltas)
	}
}

func TestAsistente_AprobadaSeAplicaSinRelayYSinFotoSiNadieMira(t *testing.T) {
	e, r, _ := entornoDePrueba(t)
	if err := MarcarRevocada(e.Root, invPago); err != nil {
		t.Fatal(err)
	}
	id, err := ProponerDeAsistente(e.Root, json.RawMessage(`{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`), "", ahoraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	linea, err := Aprobar(context.Background(), e, id)
	if err != nil || !strings.Contains(linea, "Cobra") {
		t.Fatalf("aplicada: %q %v", linea, err)
	}
	f, _ := flujo.Cargar(e.Root, "pago")
	if len(f.Nodos) != 4 || flujo.Historial(e.Root, "pago")[0].Autor != flujo.OrigenAsistente {
		t.Fatalf("el nodo, a nombre del asistente: %+v", f.Nodos)
	}
	if len(r.resueltas) != 0 || r.fotos != 0 {
		t.Fatalf("sin invitación viva no sale nada al relay: %v %d", r.resueltas, r.fotos)
	}
	if TocaCiclo(e.Root, ahoraDePrueba, 0) {
		t.Fatal("resuelta, ya no queda nada por hacer")
	}
}

func TestAsistente_ConUnInvitadoMirandoSubeLaFotoNueva(t *testing.T) {
	e, r, _ := entornoDePrueba(t)
	id, _ := ProponerDeAsistente(e.Root, json.RawMessage(`{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`), "", ahoraDePrueba)
	if _, err := Aprobar(context.Background(), e, id); err != nil {
		t.Fatal(err)
	}
	if r.fotos != 1 || len(r.resueltas) != 0 {
		t.Fatalf("el cliente ve la foto nueva; el relay no resuelve nada: %d %v", r.fotos, r.resueltas)
	}
}

func TestAsistente_DescartadaNoTocaElFlujoNiElRelay(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	id, _ := ProponerDeAsistente(e.Root, json.RawMessage(`{"op":"quitar","flujo":"pago","nodo":"b"}`), "", ahoraDePrueba)
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	p.descartar("p1")
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	l, _ := Leer(e.Root)
	if v := l.vista(id); v == nil || v.Situacion != Rechazada || len(r.resueltas) != 0 {
		t.Fatalf("rechazada y sin avisarle al relay: %+v %v", v, r.resueltas)
	}
	f, _ := flujo.Cargar(e.Root, "pago")
	if len(f.Nodos) != 3 {
		t.Fatal("y no toca el flujo")
	}
}

func TestAsistente_LoQueNoPuedeProponerNoEntra(t *testing.T) {
	e, _, _ := entornoDePrueba(t)
	for _, caso := range []struct{ op, minuto string }{
		{`{"op":"comentar","flujo":"pago","nodo":"b","texto":"x"}`, ""},
		{`{"op":"quitar","flujo":"otro","nodo":"b"}`, ""},
		{`{"op":"quitar","flujo":"pago","nodo":"b"}`, "12"},
		{`{"op":"quitar","flujo":"pago","nodo":"b","extra":1}`, ""},
	} {
		if _, err := ProponerDeAsistente(e.Root, json.RawMessage(caso.op), caso.minuto, ahoraDePrueba); err == nil {
			t.Fatalf("no entra: %s %q", caso.op, caso.minuto)
		}
	}
	l, _ := Leer(e.Root)
	if len(l.Propuestas) != 0 {
		t.Fatalf("la cola sigue vacía: %+v", l.Propuestas)
	}
}

func TestAsistente_TieneTopeDeSinResolver(t *testing.T) {
	e, _, _ := entornoDePrueba(t)
	for i := range MaxDeAsistente {
		op := `{"op":"historia","flujo":"pago","quiero":"cosa ` + itoa(i) + `"}`
		if _, err := ProponerDeAsistente(e.Root, json.RawMessage(op), "", ahoraDePrueba); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ProponerDeAsistente(e.Root, json.RawMessage(`{"op":"historia","flujo":"pago","quiero":"una más"}`), "", ahoraDePrueba); err == nil {
		t.Fatal("con el tope lleno, espera a que la persona resuelva")
	}
}

func TestAsistente_UnaDelRelayNuncaPasaPorAsistente(t *testing.T) {
	e, r, p := entornoDePrueba(t)
	falsa := propuestaDe(1, "asistente", `{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`)
	falsa.Rol = flujo.RolAsistente
	r.propuestas = []Propuesta{falsa}
	if _, err := Ciclo(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if len(p.motivos) != 1 || strings.HasPrefix(p.motivos[0], "🎙") {
		t.Fatalf("llega como de un invitado, no del asistente: %v", p.motivos)
	}
	if _, err := Aprobar(context.Background(), e, "PRO-1"); err != nil {
		t.Fatal(err)
	}
	if a := flujo.Historial(e.Root, "pago")[0].Autor; a != "cliente:asistente" {
		t.Fatalf("se aplica con el rol de la invitación: %q", a)
	}
}
