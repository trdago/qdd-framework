package flujo

import (
	"math/rand"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Flujos grandes, parte B (#186, decisión de la persona 2026-11-07: «Los tres, en orden»): el flujo se ve por niveles.
// Nivel 1 es el panorama (un grupo por caja, con sus hitos), nivel 2 el camino principal con los desvíos contados y nivel 3
// el dibujo completo. Un grupo del panorama se abre sin abrir los demás.

func enGrupo(n Nodo, g string) Nodo {
	n.Grupo = g
	return n
}

// analisis: buscar → filtrar (con un rechazo) → analizar → informe, en tres grupos.
func analisis() Flujo {
	return deNodos(
		enGrupo(paso("buscar", "Buscar licitaciones", "region"), "Descubrir"),
		enGrupo(paso("region", "Filtrar por región", "monto", "rechazo"), "Filtrar"),
		enGrupo(resultado("rechazo", "Descartada por región"), "Filtrar"),
		enGrupo(paso("monto", "Filtrar por monto", "jev"), "Filtrar"),
		enGrupo(paso("jev", "Evaluar con JEV", "informe"), "Analizar"),
		enGrupo(resultado("informe", "Informe"), "Analizar"),
	)
}

func TestNiveles_CalculadosDesdeElCaminoPrincipal(t *testing.T) {
	f := analisis()
	if got := strings.Join(CaminoPrincipal(f), ","); got != "buscar,region,monto,jev,informe" {
		t.Fatalf("el camino principal es el más largo desde el inicio: %s", got)
	}
	n := Niveles(f)
	if n["buscar"] != 1 || n["informe"] != 1 || n["region"] != 2 || n["rechazo"] != 3 {
		t.Fatalf("el inicio y el final son hitos, el camino es nivel 2 y lo de fuera nivel 3: %v", n)
	}
	f.Nodos[1].Nivel = 1
	f.Nodos[2].Nivel = 2
	if n := Niveles(f); n["region"] != 1 || n["rechazo"] != 2 {
		t.Fatalf("el nivel puesto en la tarjeta manda: %v", n)
	}
}

func idsDe(f Flujo) string {
	var ids []string
	for _, n := range f.Nodos {
		ids = append(ids, n.ID+">"+strings.Join(n.Siguen, "+"))
	}
	return strings.Join(ids, " ")
}

func TestPlegar_Nivel2QuitaLosDesviosYLosCuenta(t *testing.T) {
	p := Plegar(analisis(), Vista{Nivel: 2})
	if got := idsDe(p); got != "buscar>region region>monto monto>jev jev>informe informe>" {
		t.Fatalf("queda el camino, reconectado: %s", got)
	}
	if r := nodoDeFlujo(p, "region"); !strings.Contains(r.Descripcion, "1 desvío") {
		t.Fatalf("el nodo dice cuántos desvíos esconde: %q", r.Descripcion)
	}
}

func TestPlegar_Nivel1UnaCajaPorGrupoConSusHitos(t *testing.T) {
	f := analisis()
	f.Nodos[4].Nivel = 1 // jev: la persona lo marcó como hito
	p := Plegar(f, Vista{Nivel: 1})
	if got := idsDe(p); got != "grupo-descubrir>grupo-filtrar grupo-filtrar>grupo-analizar grupo-analizar>" {
		t.Fatalf("un nodo por grupo, en el orden del camino: %s", got)
	}
	g := nodoDeFlujo(p, "grupo-analizar")
	if g.Nombre != "Analizar" || !strings.Contains(g.Descripcion, "2 pasos") || !strings.Contains(g.Descripcion, "Evaluar con JEV") {
		t.Fatalf("la caja del grupo nombra sus hitos y cuenta sus pasos: %+v", g)
	}
	if strings.Contains(nodoDeFlujo(p, "grupo-filtrar").Descripcion, "Filtrar por monto") {
		t.Fatal("un paso que no es hito no se nombra en el panorama")
	}
}

func TestPlegar_AbrirUnGrupoMuestraSusPasos(t *testing.T) {
	p := Plegar(analisis(), Vista{Nivel: 1, Abiertos: []string{"Filtrar"}})
	if got := idsDe(p); got != "grupo-descubrir>region region>monto monto>grupo-analizar grupo-analizar>" {
		t.Fatalf("Filtrar abierto muestra su camino; los otros siguen plegados: %s", got)
	}
}

func TestPlegar_SinGruposElPanoramaSonLosHitos(t *testing.T) {
	f := deNodos(paso("a", "Llega", "b"), paso("b", "Valida", "c"), paso("c", "Cobra", "d"), resultado("d", "Fin"))
	if got := idsDe(Plegar(f, Vista{Nivel: 1})); got != "a>d d>" {
		t.Fatalf("sin grupos quedan los hitos, unidos: %s", got)
	}
	if r := nodoDeFlujo(Plegar(f, Vista{Nivel: 1}), "a"); !strings.Contains(r.Descripcion, "2 pasos") {
		t.Fatalf("y dice cuántos pasos hay entre ellos: %q", r.Descripcion)
	}
}

func TestPlegar_Nivel3EsElFlujoEntero(t *testing.T) {
	f := analisis()
	if idsDe(Plegar(f, Vista{Nivel: 3})) != idsDe(f) {
		t.Fatal("nivel 3 no cambia nada")
	}
}

func TestPlegar_ElPanoramaDeUnFlujoGrandeEsCorto(t *testing.T) {
	f := filtros(20)
	completo := len(Dibujar(f, 80, sinEstado))
	panorama := len(Dibujar(Plegar(f, Vista{Nivel: 1}), 80, sinEstado))
	if panorama*3 > completo {
		t.Fatalf("el panorama ocupa mucho menos: %d de %d líneas", panorama, completo)
	}
}

func nodoDeFlujo(f Flujo, id string) Nodo {
	for _, n := range f.Nodos {
		if n.ID == id {
			return n
		}
	}
	return Nodo{}
}

func TestPlegar_AlAzarSiempreEsUnFlujoValidoQueSeDibujaEntero(t *testing.T) {
	azar := rand.New(rand.NewSource(7))
	grupos := []string{"", "Uno", "Dos", "Tres"}
	for ronda := 0; ronda < 400; ronda++ {
		f := flujoAlAzar(azar)
		for i := range f.Nodos {
			f.Nodos[i].Grupo = grupos[azar.Intn(len(grupos))]
			f.Nodos[i].Nivel = azar.Intn(4)
		}
		for _, nivel := range []int{NivelHito, NivelPaso} {
			var abiertos []string
			if azar.Intn(2) == 0 {
				abiertos = []string{"Dos"}
			}
			p := Plegar(f, Vista{Nivel: nivel, Abiertos: abiertos})
			if problemas := Validar(p); len(problemas) > 0 {
				t.Fatalf("ronda %d nivel %d: %v\n%s", ronda, nivel, problemas, idsDe(p))
			}
			dibujo := strings.Join(Dibujar(p, 80, sinEstado), "\n")
			for _, n := range p.Nodos {
				if !strings.Contains(dibujo, n.Nombre) {
					t.Fatalf("ronda %d nivel %d: falta %q en\n%s", ronda, nivel, n.Nombre, dibujo)
				}
			}
		}
	}
}

func TestPlegar_ElInicioNuncaSeEsconde(t *testing.T) {
	f := analisis()
	f.Nodos[0].Nivel = NivelDesvio
	f.Nodos[0].Grupo = ""
	if p := Plegar(f, Vista{Nivel: NivelHito}); len(p.Nodos) == 0 || p.Nodos[0].ID != "buscar" {
		t.Fatalf("el flujo empieza donde empezaba: %s", idsDe(p))
	}
}

// Parte C (#186): la persona corrige el nivel y el grupo en la tarjeta del nodo.
func TestModificar_GrupoYNivelDesdeLaTarjeta(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	grupo, nivel := "Validar", 1
	f, linea := editar(t, root, DePersona, ModificarNodo("b", Campos{Grupo: &grupo, Nivel: &nivel}))
	if n := nodoDe(t, f, "b"); n.Grupo != "Validar" || n.Nivel != 1 || !strings.Contains(linea, "nivel") {
		t.Fatalf("%+v · %s", n, linea)
	}
	calculado := 0
	f, _ = editar(t, root, DePersona, ModificarNodo("b", Campos{Nivel: &calculado}))
	if Niveles(f)["b"] != NivelPaso {
		t.Fatal("nivel 0 vuelve al calculado")
	}
	malo := 4
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, ModificarNodo("b", Campos{Nivel: &malo})); err == nil {
		t.Fatal("el nivel es 0 (calculado), 1, 2 o 3")
	}
}

func TestOperacion_ModificarAceptaGrupoYNivel(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	aplicar(t, root, `{"op":"modificar","flujo":"pago","nodo":"b","grupo":"Validar","nivel":3}`)
	f, _ := Cargar(root, "pago")
	if n := nodoDe(t, f, "b"); n.Grupo != "Validar" || n.Nivel != 3 {
		t.Fatalf("%+v", n)
	}
	if _, err := LeerOperacion(`{"op":"comentar","flujo":"pago","nodo":"b","texto":"x","nivel":1}`); err == nil {
		t.Fatal("solo modificar lleva nivel")
	}
}

func TestPlegar_ElNodoElegidoSeVeAunqueSuNivelLoEsconda(t *testing.T) {
	p := Plegar(analisis(), Vista{Nivel: NivelHito, Mostrar: "rechazo"})
	if nodoDeFlujo(p, "rechazo").ID == "" {
		t.Fatalf("el nodo que la persona eligió se dibuja: %s", idsDe(p))
	}
	if problemas := Validar(p); len(problemas) > 0 {
		t.Fatal(problemas)
	}
}
