package flujo

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 4: el asistente es la sesión de Claude del moderador. Mientras se conversa con el cliente propone
// nodos, notas e historias como cualquier invitado, y nada entra sin el «sí» de la persona. Aprobado, queda a nombre de
// «asistente» o del minuto de la conversación que lo respalda («transcripcion@12:40»).

func TestAsistente_ProponeNodosNotasEHistoriasPeroNoComenta(t *testing.T) {
	for _, texto := range []string{
		`{"op":"agregar","flujo":"pago","nodo":"b","nombre":"Cobra"}`,
		`{"op":"nota","flujo":"pago","nodo":"b","texto":"usar la API del SII"}`,
		`{"op":"historia","flujo":"pago","quiero":"validar el RUT"}`,
		`{"op":"conectar","flujo":"pago","nodo":"a","con":"c"}`,
	} {
		if err := operacionDe(t, texto).PermitidaAlAsistente(); err != nil {
			t.Fatalf("el asistente propone %s: %v", texto, err)
		}
	}
	for _, texto := range []string{
		`{"op":"comentar","flujo":"pago","nodo":"b","texto":"¿y el RUT?"}`,
		`{"op":"deshacer","flujo":"pago"}`,
		`{"op":"item-marcar","flujo":"pago","nodo":"b","item":"i1","estado":"pasa"}`,
	} {
		if err := operacionDe(t, texto).PermitidaAlAsistente(); err == nil {
			t.Fatalf("el asistente no propone %s", texto)
		}
	}
}

func TestAsistente_OrigenEsElMinutoDeLaConversacion(t *testing.T) {
	casos := map[string]string{"": OrigenAsistente, "12:40": "transcripcion@12:40", "125:03": "transcripcion@125:03"}
	for minuto, quiere := range casos {
		origen, err := OrigenDeAsistente(minuto)
		if err != nil || origen != quiere {
			t.Fatalf("%q: %q %v", minuto, origen, err)
		}
	}
	for _, malo := range []string{"12", "12:60", "a:00", "12:40 ", "1:2"} {
		if _, err := OrigenDeAsistente(malo); err == nil {
			t.Fatalf("%q no es un minuto", malo)
		}
	}
}

func TestAsistente_AprobadaQuedaConSuOrigen(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	historia := operacionDe(t, `{"op":"historia","flujo":"pago","nodo":"b","como":"vendedor","quiero":"validar el RUT"}`)
	linea, err := historia.AplicarDeAsistente(root, "transcripcion@12:40", ahoraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(linea, "asistente") || !strings.Contains(linea, "12:40") {
		t.Fatalf("la línea dice de dónde salió: %q", linea)
	}
	nota := operacionDe(t, `{"op":"nota","flujo":"pago","nodo":"b","texto":"usar la API del SII"}`)
	if _, err := nota.AplicarDeAsistente(root, OrigenAsistente, ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	f, _ := Cargar(root, "pago")
	if len(f.Historias) != 1 || f.Historias[0].Origen != "transcripcion@12:40" {
		t.Fatalf("la historia guarda su minuto: %+v", f.Historias)
	}
	b := nodoDe(t, f, "b")
	if len(b.Items) != 1 || b.Items[0].Autor != OrigenAsistente || b.Items[0].Visible != "" {
		t.Fatalf("la nota del asistente es interna y a su nombre: %+v", b.Items)
	}
	h := Historial(root, "pago")
	if len(h) != 2 || h[0].Autor != "transcripcion@12:40" || h[1].Autor != OrigenAsistente {
		t.Fatalf("el historial anota el origen: %+v", h)
	}
	if _, err := operacionDe(t, `{"op":"quitar","flujo":"pago","nodo":"b"}`).AplicarDeAsistente(root, "persona", ahoraDePrueba); err == nil {
		t.Fatal("un origen que no es del asistente no se aplica")
	}
	if _, err := operacionDe(t, `{"op":"comentar","flujo":"pago","nodo":"b","texto":"x"}`).AplicarDeAsistente(root, OrigenAsistente, ahoraDePrueba); err == nil {
		t.Fatal("lo que el asistente no propone tampoco se aplica")
	}
}
