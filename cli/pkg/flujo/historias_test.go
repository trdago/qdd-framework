package flujo

import (
	"encoding/json"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0025: un flujo guarda sus historias de usuario junto a los nodos («como … quiero … para …», con criterios), y
// las notas técnicas del experto son internas o compartidas. La foto del cliente lleva las historias y solo las notas
// compartidas; nunca el origen de una historia ni su tarea.

func historiaDePrueba() Historia {
	return Historia{Como: "vendedor", Quiero: "validar el RUT antes de crear el cliente", Para: "no duplicar clientes",
		Criterios: []string{"un RUT inválido no deja seguir"}, Nodos: []string{"b"}}
}

func TestHistoria_SeAgregaConSuIDYSuOrigen(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	f, linea := editar(t, root, DePersona, AgregarHistoria(historiaDePrueba(), "persona"))
	if len(f.Historias) != 1 || f.Historias[0].ID != "h1" || f.Historias[0].Origen != "persona" {
		t.Fatalf("la historia queda con id h1 y su origen: %+v", f.Historias)
	}
	if !strings.Contains(linea, "validar el RUT") || !strings.Contains(linea, "h1") {
		t.Fatalf("la IA lee la historia: %q", linea)
	}
	f, _ = editar(t, root, DePersona, AgregarHistoria(historiaDePrueba(), "persona"))
	if f.Historias[1].ID != "h2" {
		t.Fatalf("la siguiente es h2: %+v", f.Historias)
	}
}

func TestHistoria_SeRevisaAntesDeGuardar(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	casos := map[string]Historia{
		"sin quiero":      {Como: "vendedor"},
		"nodo que no hay": {Quiero: "x", Nodos: []string{"zz"}},
		"criterio vacío":  {Quiero: "x", Criterios: []string{"  "}},
		"texto largo":     {Quiero: strings.Repeat("x", maxHistoria+1)},
	}
	for nombre, h := range casos {
		if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, AgregarHistoria(h, "persona")); err == nil {
			t.Fatalf("%s: no se guarda", nombre)
		}
	}
	f := deNodos(cadena()...)
	f.Historias = []Historia{{ID: "h1", Quiero: "x"}, {ID: "h1", Quiero: "y"}}
	if len(Validar(f)) == 0 {
		t.Fatal("un id repetido es un problema del flujo")
	}
}

func TestHistoria_QuitarElNodoLaDejaSinEseNodo(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, AgregarHistoria(historiaDePrueba(), "persona"))
	f, _ := editar(t, root, DePersona, Quitar("b"))
	if len(f.Historias) != 1 || len(f.Historias[0].Nodos) != 0 {
		t.Fatalf("la historia sigue, sin el nodo que ya no está: %+v", f.Historias)
	}
}

func TestHistoria_ReescribirElFlujoLasConserva(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, AgregarHistoria(historiaDePrueba(), "persona"))
	if _, err := Reemplazar(root, deNodos(cadena()...)); err != nil {
		t.Fatal(err)
	}
	f, err := Cargar(root, "pago")
	if err != nil || len(f.Historias) != 1 {
		t.Fatalf("la IA reescribe los nodos sin perder las historias: %+v %v", f.Historias, err)
	}
}

func TestHistoria_TareaYQuitar(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, AgregarHistoria(historiaDePrueba(), "persona"))
	f, _ := editar(t, root, DePersona, TareaDeHistoria("h1", "validar-el-rut"))
	if f.Historias[0].Tarea != "validar-el-rut" {
		t.Fatalf("la historia recuerda su tarea: %+v", f.Historias[0])
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, TareaDeHistoria("h9", "x")); err == nil {
		t.Fatal("una historia que no hay")
	}
	f, linea := editar(t, root, DePersona, QuitarHistoria("h1"))
	if len(f.Historias) != 0 || !strings.Contains(linea, "h1") {
		t.Fatalf("se quita: %+v %q", f.Historias, linea)
	}
}

func TestHistoria_LaProponeUnInvitadoYQuedaConSuOrigen(t *testing.T) {
	o := operacionDe(t, `{"op":"historia","flujo":"pago","nodo":"b","como":"vendedor","quiero":"validar el RUT","para":"no duplicar","criterios":["un RUT inválido no deja seguir"]}`)
	for _, rol := range RolesDeInvitado {
		if err := o.PermitidaA(rol); err != nil {
			t.Fatalf("%s escribe historias: %v", rol, err)
		}
	}
	root := flujoGuardado(t, cadena()...)
	if _, err := o.AplicarDeInvitado(root, "pago", "Ana", RolCliente, ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	f, _ := Cargar(root, "pago")
	h := f.Historias[0]
	if h.Origen != "cliente:Ana" || h.Quiero != "validar el RUT" || len(h.Criterios) != 1 || h.Nodos[0] != "b" {
		t.Fatalf("la historia aprobada: %+v", h)
	}
	if d := o.Describir(f); !strings.Contains(d, "historia") || !strings.Contains(d, "«Valida»") || !strings.Contains(d, "validar el RUT") {
		t.Fatalf("en palabras: %q", d)
	}
	for _, texto := range []string{
		`{"op":"historia","flujo":"pago","como":"vendedor"}`,
		`{"op":"comentar","flujo":"pago","nodo":"b","texto":"x","quiero":"y"}`,
		`{"op":"historia","flujo":"pago","quiero":"x","criterios":["a","b","c","d","e","f","g","h","i","j","k"]}`,
	} {
		if _, err := LeerOperacion(texto); err == nil {
			t.Fatalf("no cumple el esquema: %s", texto)
		}
	}
}

func TestNota_ElExpertoPideCompartirlaYLaPersonaLaApruebaAsi(t *testing.T) {
	o := operacionDe(t, `{"op":"nota","flujo":"pago","nodo":"b","texto":"usar la API del SII","compartida":true}`)
	if d := o.Describir(deNodos(cadena()...)); !strings.Contains(d, "compartida") {
		t.Fatalf("la persona ve que se compartirá con el cliente: %q", d)
	}
	root := flujoGuardado(t, cadena()...)
	if _, err := o.AplicarDeInvitado(root, "pago", "Luis", RolExperto, ahoraDePrueba); err != nil {
		t.Fatal(err)
	}
	f, _ := Cargar(root, "pago")
	if it := nodoDe(t, f, "b").Items[0]; it.Visible != VisibleCompartida || it.Autor != "Luis" {
		t.Fatalf("la nota queda compartida y a su nombre: %+v", it)
	}
	if _, err := LeerOperacion(`{"op":"agregar","flujo":"pago","nodo":"b","nombre":"x","compartida":true}`); err == nil {
		t.Fatal("solo una nota se comparte")
	}
}

func TestNota_LaPersonaCambiaSuVisibilidad(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, AgregarItem("b", ItemNota, "usar la API del SII", ahoraDePrueba))
	f, linea := editar(t, root, DePersona, CompartirItem("b", "i1", true))
	if nodoDe(t, f, "b").Items[0].Visible != VisibleCompartida || !strings.Contains(linea, "compartida") {
		t.Fatalf("queda compartida: %+v %q", nodoDe(t, f, "b").Items, linea)
	}
	f, _ = editar(t, root, DePersona, CompartirItem("b", "i1", false))
	if nodoDe(t, f, "b").Items[0].Visible != "" {
		t.Fatalf("vuelve a interna: %+v", nodoDe(t, f, "b").Items)
	}
	editar(t, root, DePersona, AgregarItem("b", ItemPrueba, "valida un RUT", ahoraDePrueba))
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, CompartirItem("b", "i2", true)); err == nil {
		t.Fatal("una prueba no se comparte")
	}
	g := deNodos(cadena()...)
	g.Nodos[1].Items = []Item{{ID: "i1", Tipo: ItemNota, Texto: "x", Visible: "publica"}}
	if len(Validar(g)) == 0 {
		t.Fatal("una visibilidad que no existe es un problema del flujo")
	}
}

func TestFoto_LlevaLasHistoriasYSoloLasNotasCompartidas(t *testing.T) {
	f := flujoConGrupos()
	f.Nodos[1].Items = append(f.Nodos[1].Items,
		Item{ID: "i2", Tipo: ItemNota, Texto: "API del SII en /home/ana/proyecto", Visible: VisibleCompartida, Autor: "Luis"},
		Item{ID: "i3", Tipo: ItemCaracteristica, Texto: "caracteristica secreta"})
	f.Historias = []Historia{{ID: "h1", Como: "vendedor", Quiero: "validar el RUT", Para: "no duplicar",
		Criterios: []string{"inválido no sigue"}, Nodos: []string{"a"}, Origen: "transcripcion@12:40", Tarea: "t-9"}}
	foto := ArmarFoto(f, limpiezaDePrueba, ahoraDePrueba)
	datos, _ := json.Marshal(foto)
	texto := string(datos)
	for _, prohibido := range []string{"interna", "caracteristica secreta", "transcripcion@", "t-9", "/home/ana"} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("la foto lleva %q: %s", prohibido, texto)
		}
	}
	notas := foto.Nodos[1].Notas
	if len(notas) != 1 || notas[0].Texto != "API del SII en <proyecto>" {
		t.Fatalf("la nota compartida, limpia: %+v", notas)
	}
	if len(foto.Historias) != 1 || foto.Historias[0].Quiero != "validar el RUT" || foto.Historias[0].Nodos[0] != "a" {
		t.Fatalf("las historias van en la foto: %+v", foto.Historias)
	}
}
