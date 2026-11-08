package flujo

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-07): cada nodo tiene ítems. Al probar con qdd-human son las pruebas, y dicen si pasaron;
// desde el mismo nodo se agrega algo específico que probar. En un flujo cualquiera sirven para notas, características y
// funciones.

func TestItem_UnaPruebaNacePendienteYSeMarcaConSuEvidencia(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	f, linea := editar(t, root, DePersona, AgregarItem("b", ItemPrueba, "rechaza una firma vencida", ahoraDePrueba))
	items := nodoDe(t, f, "b").Items
	if len(items) != 1 || items[0].ID != "i1" || items[0].Tipo != ItemPrueba || items[0].Estado != PruebaPendiente || items[0].Texto != "rechaza una firma vencida" {
		t.Fatalf("una prueba nueva queda pendiente, con su id: %+v", items)
	}
	if !strings.Contains(linea, "rechaza una firma vencida") || !strings.Contains(linea, "b") {
		t.Fatalf("la IA ve qué probar y dónde: %q", linea)
	}
	mensaje, _ := Enviar(root, "pago")
	if !strings.Contains(mensaje, "qdd flujo item marcar pago b i1") {
		t.Fatalf("y cómo marcarla: %q", mensaje)
	}
	f, _ = editar(t, root, DeIA, MarcarItem("b", "i1", PruebaFalla, ".qdd/evidence/run_1", ahoraDePrueba))
	got := nodoDe(t, f, "b").Items[0]
	if got.Estado != PruebaFalla || got.Evidencia != ".qdd/evidence/run_1" || got.Cuando == "" {
		t.Fatalf("queda marcada, con evidencia y cuándo: %+v", got)
	}
	f, _ = editar(t, root, DePersona, MarcarItem("b", "i1", PruebaPasa, "", ahoraDePrueba))
	if got := nodoDe(t, f, "b").Items[0]; got.Estado != PruebaPasa || got.Evidencia != "" {
		t.Fatalf("una marca nueva reemplaza la evidencia anterior: %+v", got)
	}
}

func TestItem_NotasCaracteristicasYFuncionesNoTienenEstado(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	for _, tipo := range []string{ItemNota, ItemCaracteristica, ItemFuncion} {
		editar(t, root, DePersona, AgregarItem("a", tipo, "algo de "+tipo, ahoraDePrueba))
	}
	f, err := Cargar(root, "pago")
	if err != nil {
		t.Fatal(err)
	}
	items := nodoDe(t, f, "a").Items
	if len(items) != 3 || items[2].ID != "i3" {
		t.Fatalf("tres ítems con ids seguidos: %+v", items)
	}
	for _, it := range items {
		if it.Estado != "" {
			t.Fatalf("solo una prueba tiene estado: %+v", it)
		}
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, MarcarItem("a", "i1", PruebaPasa, "", ahoraDePrueba)); err == nil {
		t.Fatal("una nota no se marca como pasada")
	}
}

func TestItem_LoQueNoSePuedeNoEscribe(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	malos := []Cambio{
		AgregarItem("b", "tarea", "x", ahoraDePrueba),
		AgregarItem("b", ItemNota, "  ", ahoraDePrueba),
		AgregarItem("z", ItemNota, "x", ahoraDePrueba),
		AgregarItem("b", ItemNota, strings.Repeat("x", maxItem+1), ahoraDePrueba),
		MarcarItem("b", "i9", PruebaPasa, "", ahoraDePrueba),
		QuitarItem("b", "i9"),
	}
	for i, c := range malos {
		if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, c); err == nil {
			t.Errorf("cambio %d: debió fallar", i)
		}
	}
	editar(t, root, DePersona, AgregarItem("b", ItemPrueba, "x", ahoraDePrueba))
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, MarcarItem("b", "i1", "quizas", "", ahoraDePrueba)); err == nil {
		t.Error("un estado que no existe se rechaza")
	}
}

func TestItem_QuitarLoSacaDelNodo(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, AgregarItem("b", ItemNota, "uno", ahoraDePrueba))
	editar(t, root, DePersona, AgregarItem("b", ItemNota, "dos", ahoraDePrueba))
	editar(t, root, DePersona, QuitarItem("b", "i2"))
	f, _ := editar(t, root, DePersona, QuitarItem("b", "i1"))
	if len(nodoDe(t, f, "b").Items) != 0 {
		t.Fatalf("quitados: %+v", nodoDe(t, f, "b").Items)
	}
	f, _ = editar(t, root, DePersona, AgregarItem("b", ItemNota, "tres", ahoraDePrueba))
	if id := nodoDe(t, f, "b").Items[0].ID; id == "" {
		t.Fatalf("tiene id: %q", id)
	}
}

func TestValidar_UnItemRaroSeNombra(t *testing.T) {
	n := paso("a", "A")
	n.Items = []Item{{ID: "i1", Tipo: ItemNota, Texto: "x", Estado: PruebaPasa}, {ID: "i1", Tipo: "otro", Texto: "y"}}
	p := Validar(deNodos(n))
	if len(p) != 3 {
		t.Fatalf("estado en una nota, id repetido y tipo desconocido: %v", p)
	}
}

func TestReemplazar_ConservaLosItems(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, AgregarItem("b", ItemPrueba, "x", ahoraDePrueba))
	if _, err := Reemplazar(root, deNodos(cadena()...)); err != nil {
		t.Fatal(err)
	}
	f, _ := Cargar(root, "pago")
	if len(nodoDe(t, f, "b").Items) != 1 {
		t.Fatalf("reescribir el flujo no pierde los ítems: %+v", f.Nodos)
	}
}

func TestOperacion_ItemsDesdeElTelefono(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	aplicar(t, root, `{"op":"item-agregar","flujo":"pago","nodo":"b","tipo":"prueba","texto":"rechaza vencidas"}`)
	aplicar(t, root, `{"op":"item-marcar","flujo":"pago","nodo":"b","item":"i1","estado":"falla","evidencia":"lo probé a mano"}`)
	aplicar(t, root, `{"op":"item-agregar","flujo":"pago","nodo":"b","tipo":"nota","texto":"ojo con la hora"}`)
	aplicar(t, root, `{"op":"item-quitar","flujo":"pago","nodo":"b","item":"i2"}`)
	f, _ := Cargar(root, "pago")
	items := nodoDe(t, f, "b").Items
	if len(items) != 1 || items[0].Estado != PruebaFalla || items[0].Evidencia != "lo probé a mano" {
		t.Fatalf("ítems: %+v", items)
	}
	for _, malo := range []string{
		`{"op":"item-agregar","flujo":"pago","nodo":"b","tipo":"paso","texto":"x"}`,
		`{"op":"item-marcar","flujo":"pago","nodo":"b","item":"i1"}`,
		`{"op":"agregar","flujo":"pago","nodo":"b","nombre":"x","tipo":"prueba"}`,
		`{"op":"item-quitar","flujo":"pago","nodo":"b","item":"I 1"}`,
	} {
		if _, err := LeerOperacion(malo); err == nil {
			t.Errorf("%s: debió rechazarse", malo)
		}
	}
}

func TestListar_CuentaLasPruebasDeCadaFlujo(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	if p := Listar(root, sinEstado)[0].Pruebas; p != nil {
		t.Fatalf("sin pruebas no hay conteo: %+v", p)
	}
	editar(t, root, DePersona, AgregarItem("a", ItemPrueba, "uno", ahoraDePrueba))
	editar(t, root, DePersona, AgregarItem("b", ItemPrueba, "dos", ahoraDePrueba))
	editar(t, root, DePersona, AgregarItem("b", ItemPrueba, "tres", ahoraDePrueba))
	editar(t, root, DePersona, AgregarItem("b", ItemNota, "no cuenta", ahoraDePrueba))
	editar(t, root, DeIA, MarcarItem("a", "i1", PruebaPasa, "", ahoraDePrueba))
	editar(t, root, DeIA, MarcarItem("b", "i1", PruebaFalla, "", ahoraDePrueba))
	p := Listar(root, sinEstado)[0].Pruebas
	if p == nil || p.Pasa != 1 || p.Falla != 1 || p.Pendiente != 1 {
		t.Fatalf("el mapa de flujos muestra cuántas pasan, fallan y faltan: %+v", p)
	}
}
