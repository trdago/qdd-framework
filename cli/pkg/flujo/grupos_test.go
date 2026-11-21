package flujo

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Pedido de la persona (2026-11-08): «falta un elemento de agrupación con un nombre que se use como, por ejemplo, cola,
// ambiente, fase … lo veo como un [] que agrupa nodos y sirve para visualizar mejor». Un grupo tiene nombre y clase; sus
// nodos lo nombran en Nodo.Grupo, y se dibuja como un marco «[fase] Ingreso».

func TestDefinirGrupo_LeDaClaseYDescripcionYLoDice(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	editar(t, root, DePersona, ModificarNodo("a", Campos{Grupo: ptr("Ingreso")}))
	f, linea := editar(t, root, DePersona, DefinirGrupo("Ingreso", "fase", "lo que pasa al llegar"))
	g, ok := GrupoDe(f, "Ingreso")
	if !ok || g.Clase != "fase" || g.Descripcion != "lo que pasa al llegar" || !strings.Contains(linea, "[fase] Ingreso") {
		t.Fatalf("grupo: %+v %q", g, linea)
	}
	f, _ = editar(t, root, DePersona, DefinirGrupo("Ingreso", "cola", ""))
	if g, _ := GrupoDe(f, "Ingreso"); g.Clase != "cola" || len(f.Grupos) != 1 {
		t.Fatalf("definirlo otra vez lo cambia, no lo repite: %+v", f.Grupos)
	}
}

func TestValidar_GruposRepetidosOClaseLarga(t *testing.T) {
	f := deNodos(cadena()...)
	f.Grupos = []Grupo{{Nombre: "Ingreso", Clase: "fase"}, {Nombre: "Ingreso", Clase: "cola"}}
	if p := strings.Join(Validar(f), "\n"); !strings.Contains(p, "repetido") {
		t.Fatalf("un grupo repetido se dice: %s", p)
	}
	f.Grupos = []Grupo{{Nombre: "Ingreso", Clase: strings.Repeat("x", maxClase+1)}}
	if p := strings.Join(Validar(f), "\n"); !strings.Contains(p, "clase") {
		t.Fatalf("una clase larga se dice: %s", p)
	}
}

func TestPlantillas_SugierenClasesDeGrupo(t *testing.T) {
	for _, p := range Plantillas {
		if len(p.Clases) == 0 {
			t.Fatalf("%s: sugiere clases de grupo (fase, cola, ambiente…)", p.ID)
		}
	}
}

func TestFoto_LlevaLosGruposConSuClase(t *testing.T) {
	f := flujoConGrupos()
	f.Grupos = []Grupo{{Nombre: "Ingreso", Clase: "fase", Descripcion: "en /home/ana/proyecto"}}
	foto := ArmarFoto(f, limpiezaDePrueba, ahoraDePrueba)
	if len(foto.Grupos) != 1 || foto.Grupos[0].Clase != "fase" || foto.Grupos[0].Descripcion != "en <proyecto>" {
		t.Fatalf("grupos limpios con su clase: %+v", foto.Grupos)
	}
	for _, n := range foto.Vistas[0].Nodos {
		if n.Grupo && n.Clase != "fase" {
			t.Fatalf("la caja del panorama dice su clase: %+v", n)
		}
	}
}
