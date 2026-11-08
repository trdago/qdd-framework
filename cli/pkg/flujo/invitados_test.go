package flujo

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 3: lo que propone un invitado (cliente o experto) llega como una operación del mismo esquema que el
// teléfono. Solo entra cuando la persona la aprueba, y entonces queda a nombre del invitado en el historial, en el
// comentario o en la nota, y la IA se entera como de una edición de la persona.

func operacionDe(t *testing.T, texto string) Operacion {
	t.Helper()
	o, err := LeerOperacion(texto)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestInvitado_CadaRolProponeSoloLoSuyo(t *testing.T) {
	nota := operacionDe(t, `{"op":"nota","flujo":"pago","nodo":"b","texto":"usar la API del SII"}`)
	if err := nota.PermitidaA(RolCliente); err == nil {
		t.Fatal("una nota técnica es del experto")
	}
	if err := nota.PermitidaA(RolExperto); err != nil {
		t.Fatalf("el experto anota: %v", err)
	}
	for _, texto := range []string{
		`{"op":"item-marcar","flujo":"pago","nodo":"b","item":"i1","estado":"pasa"}`,
		`{"op":"deshacer","flujo":"pago"}`,
		`{"op":"enviar","flujo":"pago"}`,
		`{"op":"pedir-detalle","flujo":"pago","nodo":"b"}`,
	} {
		if err := operacionDe(t, texto).PermitidaA(RolExperto); err == nil {
			t.Fatalf("un invitado no hace %s", texto)
		}
	}
	if err := operacionDe(t, `{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`).PermitidaA("dueño"); err == nil {
		t.Fatal("un rol que no existe no propone nada")
	}
}

func TestInvitado_AprobadaQuedaASuNombreYLaIASeEntera(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	o := operacionDe(t, `{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`)
	linea, err := o.AplicarDeInvitado(root, "pago", "Ana", RolCliente, ahoraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(linea, "Ana") || !strings.Contains(linea, "Cobra") {
		t.Fatalf("la línea dice quién lo propuso: %q", linea)
	}
	h := Historial(root, "pago")
	if len(h) != 1 || h[0].Autor != "cliente:Ana" {
		t.Fatalf("el historial la anota a nombre del invitado: %+v", h)
	}
	if p := PorEnviar(root, "pago"); len(p) != 1 || !strings.Contains(p[0], "Ana") {
		t.Fatalf("la IA se entera como de una edición de la persona: %v", p)
	}
}

func TestInvitado_ComentarioYNotaConSuAutor(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	if _, err := operacionDe(t, `{"op":"comentar","flujo":"pago","nodo":"b","texto":"aquí falta el RUT"}`).AplicarDeInvitado(root, "pago", "Ana", RolCliente, ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	if _, err := operacionDe(t, `{"op":"nota","flujo":"pago","nodo":"b","texto":"usar la API del SII"}`).AplicarDeInvitado(root, "pago", "Luis", RolExperto, ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	f, _ := Cargar(root, "pago")
	b := nodoDe(t, f, "b")
	if len(b.Comentarios) != 1 || b.Comentarios[0].De != DeInvitado || b.Comentarios[0].Autor != "Ana" {
		t.Fatalf("el comentario es del invitado, con su nombre: %+v", b.Comentarios)
	}
	if len(b.Items) != 1 || b.Items[0].Tipo != "nota" || b.Items[0].Autor != "Luis" {
		t.Fatalf("la nota del experto es un ítem nota con su autor: %+v", b.Items)
	}
}

func TestInvitado_OtraOperacionUOtroFlujoNoSeAplican(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	o := operacionDe(t, `{"op":"quitar","flujo":"otro","nodo":"b"}`)
	if _, err := o.AplicarDeInvitado(root, "pago", "Ana", RolCliente, ahoraDePrueba); err == nil {
		t.Fatal("una propuesta para otro flujo no se aplica en este")
	}
	nota := operacionDe(t, `{"op":"nota","flujo":"pago","nodo":"b","texto":"x"}`)
	if _, err := nota.AplicarDeInvitado(root, "pago", "Ana", RolCliente, ahoraDePrueba); err == nil {
		t.Fatal("el rol se comprueba también al aplicar")
	}
}

func TestDescribir_EnPalabrasConLosNombres(t *testing.T) {
	f := deNodos(cadena()...)
	casos := map[string]string{
		`{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`:           "agregar «Cobra» después de «Valida»",
		`{"op":"quitar","flujo":"pago","nodo":"a"}`:                             "quitar «Llega»",
		`{"op":"modificar","flujo":"pago","nodo":"b","nombre":"Revisa"}`:        "cambiar «Valida»: nombre «Revisa»",
		`{"op":"comentar","flujo":"pago","nodo":"b","texto":"falta el RUT"}`:    "comenta «Valida»: falta el RUT",
		`{"op":"conectar","flujo":"pago","nodo":"a","con":"c"}`:                 "flecha de «Llega» a «Fin»",
		`{"op":"mover","flujo":"pago","nodo":"a","despues_de":"b"}`:             "mover «Llega» después de «Valida»",
		`{"op":"nota","flujo":"pago","nodo":"c","texto":"usar la API del SII"}`: "nota técnica interna en «Fin»: usar la API del SII",
	}
	for texto, quiere := range casos {
		if d := operacionDe(t, texto).Describir(f); d != quiere {
			t.Fatalf("%s: %q, no %q", texto, d, quiere)
		}
	}
}
