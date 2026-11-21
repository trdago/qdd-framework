package flujo

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-07): mirando un flujo, preguntarle a un nodo qué hace o qué significa una palabra, y que
// la respuesta venga orientada a ese nodo; y preguntarle también al flujo completo.

func conDescripcion(n Nodo, d string) Nodo {
	n.Descripcion = d
	return n
}

func TestPreguntar_UnNodoLlevaSuContextoALaIA(t *testing.T) {
	root := flujoGuardado(t, paso("a", "Llega", "b"), conDescripcion(paso("b", "Valida la firma", "c", "d"), "con HMAC"), resultado("c", "Cobrado"), resultado("d", "Rechazado"))
	f, _ := editar(t, root, DePersona, Preguntar("b", "¿Qué es HMAC?", ahoraDePrueba))
	c := nodoDe(t, f, "b").Comentarios
	if len(c) != 1 || c[0].De != DePersona || !c[0].Pregunta || c[0].Texto != "¿Qué es HMAC?" {
		t.Fatalf("la pregunta queda bajo el nodo, marcada: %+v", c)
	}
	mensaje, n := Enviar(root, "pago")
	if n != 1 {
		t.Fatalf("una línea por enviar: %d", n)
	}
	for _, want := range []string{"❓", "Valida la firma", "¿Qué es HMAC?", "con HMAC", "Llega", "Cobrado", "Rechazado", "qdd flujo responder pago b"} {
		if !strings.Contains(mensaje, want) {
			t.Fatalf("falta %q: lleva la pregunta, el contexto del nodo (descripción, de dónde viene, qué sigue) y cómo responder: %q", want, mensaje)
		}
	}
}

func TestPreguntar_AlFlujoCompleto(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	f, _ := editar(t, root, DePersona, Preguntar("", "¿Para qué sirve este flujo?", ahoraDePrueba))
	if len(f.Comentarios) != 1 || !f.Comentarios[0].Pregunta {
		t.Fatalf("la pregunta queda en el flujo, no en un nodo: %+v", f.Comentarios)
	}
	mensaje, _ := Enviar(root, "pago")
	if !strings.Contains(mensaje, "¿Para qué sirve este flujo?") || !strings.Contains(mensaje, "qdd flujo responder pago \"") {
		t.Fatalf("se responde sin nodo: %q", mensaje)
	}
}

func TestResponder_LaIARespondeBajoElNodoOElFlujoSinLlamarseASiMisma(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Preguntar("b", "¿Qué valida?", ahoraDePrueba))
	editar(t, root, DePersona, Preguntar("", "¿Y el flujo?", ahoraDePrueba))
	Enviar(root, "pago")
	editar(t, root, DeIA, Responder("b", "La firma del banco.", ahoraDePrueba))
	f, _ := editar(t, root, DeIA, Responder("", "Cobra un pago.", ahoraDePrueba))
	if c := nodoDe(t, f, "b").Comentarios; len(c) != 2 || c[1].De != DeIA || c[1].Pregunta {
		t.Fatalf("la respuesta va bajo la pregunta del nodo: %+v", c)
	}
	if len(f.Comentarios) != 2 || f.Comentarios[1].De != DeIA {
		t.Fatalf("y la del flujo, en el flujo: %+v", f.Comentarios)
	}
	if m, n := Enviar(root, "pago"); n != 0 {
		t.Fatalf("la respuesta de la IA no vuelve a ella: %q", m)
	}
}

func TestPreguntar_VaciaONodoQueNoEstaNoEscribe(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Preguntar("b", "  ", ahoraDePrueba)); err == nil {
		t.Fatal("una pregunta vacía no se guarda")
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Preguntar("z", "¿y esto?", ahoraDePrueba)); err == nil {
		t.Fatal("un nodo que no está avisa")
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Preguntar("", strings.Repeat("x", maxComentario+1), ahoraDePrueba)); err == nil {
		t.Fatal("tiene el tope de un comentario")
	}
}

func TestReemplazar_ConservaLasPreguntasDelFlujo(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, Preguntar("", "¿Para qué sirve?", ahoraDePrueba))
	if _, err := Reemplazar(root, deNodos(cadena()...)); err != nil {
		t.Fatal(err)
	}
	f, err := Cargar(root, "pago")
	if err != nil || len(f.Comentarios) != 1 {
		t.Fatalf("reescribir el flujo no pierde sus preguntas: %+v %v", f.Comentarios, err)
	}
}

func TestValidar_UnComentarioRaroDelFlujoSeNombra(t *testing.T) {
	f := deNodos(cadena()...)
	f.Comentarios = []Comentario{{De: "otro", Texto: "x"}}
	if p := Validar(f); len(p) != 1 || !strings.Contains(p[0], "el flujo") {
		t.Fatalf("problemas: %v", p)
	}
}

func TestOperacion_PreguntarDesdeElTelefono(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	aplicar(t, root, `{"op":"preguntar","flujo":"pago","nodo":"b","texto":"¿Qué valida?"}`)
	aplicar(t, root, `{"op":"preguntar","flujo":"pago","texto":"¿Para qué sirve?"}`)
	if got := PorEnviar(root, "pago"); len(got) != 2 || !strings.Contains(got[0], "❓") {
		t.Fatalf("las dos quedan por enviar: %v", got)
	}
	op, err := LeerOperacion(`{"op":"preguntar","flujo":"pago","nodo":"b"}`)
	if err == nil {
		t.Fatalf("sin texto no es una pregunta: %+v", op)
	}
	op, _ = LeerOperacion(`{"op":"preguntar","flujo":"pago","texto":"¿y?"}`)
	if !op.LlamaALaIA() {
		t.Fatal("una pregunta llama a la IA al tiro")
	}
}
