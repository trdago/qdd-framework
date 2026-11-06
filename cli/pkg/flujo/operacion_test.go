package flujo

import (
	"strings"
	"testing"
)

func TestLeerOperacion_AceptaCadaOperacionConSusCampos(t *testing.T) {
	buenas := []string{
		`{"op":"comentar","flujo":"pago","nodo":"a","texto":"¿y si falla?"}`,
		`{"op":"pedir-detalle","flujo":"pago","nodo":"a"}`,
		`{"op":"agregar","flujo":"pago","nodo":"a","nombre":"Revisa"}`,
		`{"op":"agregar","flujo":"pago","nodo":"a","nombre":"Revisa","descripcion":"d","tipo":"resultado","rama":true}`,
		`{"op":"modificar","flujo":"pago","nodo":"a","nombre":"Otro"}`,
		`{"op":"modificar","flujo":"pago","nodo":"a","descripcion":""}`,
		`{"op":"mover","flujo":"pago","nodo":"a","despues_de":"b"}`,
		`{"op":"quitar","flujo":"pago","nodo":"a"}`,
		`{"op":"conectar","flujo":"pago","nodo":"a","con":"c"}`,
		`{"op":"desconectar","flujo":"pago","nodo":"a","con":"c"}`,
		`{"op":"deshacer","flujo":"pago"}`,
		`{"op":"enviar","flujo":"pago"}`,
	}
	for _, texto := range buenas {
		if _, err := LeerOperacion(texto); err != nil {
			t.Errorf("%s: %v", texto, err)
		}
	}
}

func TestLeerOperacion_RechazaLoQueNoEsDelEsquema(t *testing.T) {
	malas := map[string]string{
		"no es JSON":             `comentar pago a`,
		"datos después":          `{"op":"deshacer","flujo":"pago"}{"op":"enviar","flujo":"pago"}`,
		"op desconocida":         `{"op":"borrar","flujo":"pago"}`,
		"detalle es de la IA":    `{"op":"detalle","flujo":"pago","nodo":"a","texto":"x"}`,
		"sin op":                 `{"flujo":"pago"}`,
		"campo desconocido":      `{"op":"deshacer","flujo":"pago","de":"ia"}`,
		"campo de otra op":       `{"op":"quitar","flujo":"pago","nodo":"a","texto":"x"}`,
		"rama fuera de agregar":  `{"op":"quitar","flujo":"pago","nodo":"a","rama":true}`,
		"falta el flujo":         `{"op":"deshacer"}`,
		"falta el nodo":          `{"op":"quitar","flujo":"pago"}`,
		"falta el texto":         `{"op":"comentar","flujo":"pago","nodo":"a"}`,
		"falta el nombre":        `{"op":"agregar","flujo":"pago","nodo":"a"}`,
		"falta despues_de":       `{"op":"mover","flujo":"pago","nodo":"a"}`,
		"falta con":              `{"op":"conectar","flujo":"pago","nodo":"a"}`,
		"modificar sin cambios":  `{"op":"modificar","flujo":"pago","nodo":"a"}`,
		"flujo con ruta":         `{"op":"deshacer","flujo":"../../etc/x"}`,
		"nodo con mayúsculas":    `{"op":"quitar","flujo":"pago","nodo":"A b"}`,
		"despues_de con espacio": `{"op":"mover","flujo":"pago","nodo":"a","despues_de":"b c"}`,
		"con que es una opción":  `{"op":"conectar","flujo":"pago","nodo":"a","con":"--de"}`,
		"tipo desconocido":       `{"op":"modificar","flujo":"pago","nodo":"a","tipo":"caja"}`,
		"control crudo":          "{\"op\":\"deshacer\",\n\"flujo\":\"pago\"}",
		"texto solo espacios":    `{"op":"comentar","flujo":"pago","nodo":"a","texto":"   "}`,
		"demasiado largo":        `{"op":"comentar","flujo":"pago","nodo":"a","texto":"` + strings.Repeat("x", MaxOperacion) + `"}`,
	}
	for motivo, texto := range malas {
		if _, err := LeerOperacion(texto); err == nil {
			t.Errorf("%s: se aceptó %s", motivo, texto)
		}
	}
}

func aplicar(t *testing.T, root, texto string) string {
	t.Helper()
	op, err := LeerOperacion(texto)
	if err != nil {
		t.Fatal(err)
	}
	linea, err := op.Aplicar(root, ahoraDePrueba)
	if err != nil {
		t.Fatal(err)
	}
	return linea
}

func TestOperacion_LaEdicionQuedaDeLaPersonaPorEnviarYSeDeshace(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	linea := aplicar(t, root, `{"op":"comentar","flujo":"pago","nodo":"a","texto":"¿y si falla?"}`)
	if !strings.Contains(linea, "¿y si falla?") {
		t.Fatalf("la línea dice qué hizo: %q", linea)
	}
	f, err := Cargar(root, "pago")
	if err != nil {
		t.Fatal(err)
	}
	if got := f.Nodos[0].Comentarios; len(got) != 1 || got[0].De != DePersona {
		t.Fatalf("el comentario es de la persona: %+v", got)
	}
	if len(PorEnviar(root, "pago")) != 1 || !PuedeDeshacer(root, "pago") {
		t.Fatal("queda por enviar y se puede deshacer")
	}
	aplicar(t, root, `{"op":"deshacer","flujo":"pago"}`)
	f, err = Cargar(root, "pago")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Nodos[0].Comentarios) != 0 || len(PorEnviar(root, "pago")) != 0 {
		t.Fatalf("deshacer lo saca: %+v", f.Nodos[0])
	}
}

func TestOperacion_CadaEdicionCambiaElFlujo(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	aplicar(t, root, `{"op":"agregar","flujo":"pago","nodo":"a","nombre":"Revisa","descripcion":"con calma"}`)
	aplicar(t, root, `{"op":"modificar","flujo":"pago","nodo":"b","nombre":"Valida bien"}`)
	aplicar(t, root, `{"op":"conectar","flujo":"pago","nodo":"a","con":"c"}`)
	aplicar(t, root, `{"op":"desconectar","flujo":"pago","nodo":"a","con":"c"}`)
	aplicar(t, root, `{"op":"mover","flujo":"pago","nodo":"b","despues_de":"a"}`)
	aplicar(t, root, `{"op":"quitar","flujo":"pago","nodo":"revisa"}`)
	f, err := Cargar(root, "pago")
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Nodos) != 3 || f.Nodos[1].Nombre != "Valida bien" {
		t.Fatalf("flujo: %+v", f.Nodos)
	}
	if len(PorEnviar(root, "pago")) != 6 {
		t.Fatalf("las seis quedan por enviar: %v", PorEnviar(root, "pago"))
	}
}

func TestOperacion_PedirDetalleNoTocaElFlujo(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	aplicar(t, root, `{"op":"pedir-detalle","flujo":"pago","nodo":"a"}`)
	if got := PorEnviar(root, "pago"); len(got) != 1 || !strings.Contains(got[0], "🔎") {
		t.Fatalf("la pregunta queda por enviar: %v", got)
	}
}

func TestOperacion_LoQueNoSePuedeHacerNoEscribeNada(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	malas := []string{
		`{"op":"quitar","flujo":"pago","nodo":"no-existe"}`,
		`{"op":"quitar","flujo":"otro","nodo":"a"}`,
		`{"op":"desconectar","flujo":"pago","nodo":"a","con":"c"}`,
	}
	for _, texto := range malas {
		op, err := LeerOperacion(texto)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := op.Aplicar(root, ahoraDePrueba); err == nil {
			t.Errorf("%s: debió fallar", texto)
		}
	}
	if len(PorEnviar(root, "pago")) != 0 {
		t.Fatalf("no anotó nada: %v", PorEnviar(root, "pago"))
	}
}

func TestOperacion_EnviarNoSeAplicaComoEdicion(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	op, err := LeerOperacion(`{"op":"enviar","flujo":"pago"}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := op.Aplicar(root, ahoraDePrueba); err == nil {
		t.Fatal("enviar reclama lo pendiente y lo entrega quien la recibe")
	}
}

func TestListar_DiceQueEsperaEnvioYSiSePuedeDeshacer(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	if r := Listar(root, nil)[0]; r.PorEnviar != 0 || r.PuedeDeshacer {
		t.Fatalf("sin ediciones: %+v", r)
	}
	aplicar(t, root, `{"op":"comentar","flujo":"pago","nodo":"a","texto":"hola"}`)
	if r := Listar(root, nil)[0]; r.PorEnviar != 1 || !r.PuedeDeshacer {
		t.Fatalf("con un comentario: %+v", r)
	}
}

func TestEnviar_ElMensajeNoDiceDeDondeVino(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	aplicar(t, root, `{"op":"comentar","flujo":"pago","nodo":"a","texto":"hola"}`)
	mensaje, n := Enviar(root, "pago")
	if n != 1 || strings.Contains(mensaje, "desde el panel") {
		t.Fatalf("una edición puede venir del panel o del teléfono: %q", mensaje)
	}
}

func TestOperacion_SoloLoQuePideRespuestaLlamaALaIA(t *testing.T) {
	for texto, llama := range map[string]bool{
		`{"op":"comentar","flujo":"pago","nodo":"a","texto":"hola"}`: true,
		`{"op":"pedir-detalle","flujo":"pago","nodo":"a"}`:           true,
		`{"op":"quitar","flujo":"pago","nodo":"a"}`:                  false,
	} {
		op, err := LeerOperacion(texto)
		if err != nil {
			t.Fatal(err)
		}
		if op.LlamaALaIA() != llama {
			t.Errorf("%s: %v", texto, op.LlamaALaIA())
		}
	}
}
