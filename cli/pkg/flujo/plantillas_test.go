package flujo

import (
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0025 (decisión de la persona, 2026-11-08): «un flujo, tipos por plantilla». Una página web, un flujo de datos o un
// sistema de agentes son el mismo grafo; la plantilla trae los tipos de nodo, su forma, su leyenda y el nombre de cada
// nivel N1 · N2 · N3.

func TestPlantillas_CadaUnaTraeSusNivelesYTiposConResultado(t *testing.T) {
	for _, id := range []string{"proceso", "web", "datos", "agentes"} {
		p, ok := PlantillaPorID(id)
		if !ok {
			t.Fatalf("falta la plantilla %q", id)
		}
		for i, nivel := range p.Niveles {
			if strings.TrimSpace(nivel) == "" {
				t.Fatalf("%s: el nivel N%d no tiene nombre", id, i+1)
			}
		}
		if !p.Tiene(TipoResultado) {
			t.Fatalf("%s: todo flujo empieza y termina en un resultado (círculo)", id)
		}
		for _, tn := range p.Tipos {
			if tn.Nombre == "" || tn.Icono == "" || tn.Explica == "" {
				t.Fatalf("%s: el tipo %q necesita nombre, ícono y leyenda: %+v", id, tn.ID, tn)
			}
			if !idValido.MatchString(tn.ID) {
				t.Fatalf("%s: el id de tipo %q tiene que servir en una operación", id, tn.ID)
			}
		}
	}
}

func TestPlantilla_VaciaEsProcesoYLosFlujosDeAntesSiguenValidos(t *testing.T) {
	f := deNodos(cadena()...)
	if p := PlantillaDe(f); p.ID != PlantillaProceso || !p.Tiene(TipoPaso) {
		t.Fatalf("sin plantilla es proceso: %+v", p)
	}
	if problemas := Validar(f); len(problemas) > 0 {
		t.Fatalf("un flujo de antes sigue válido: %v", problemas)
	}
}

func TestValidar_UnTipoQueLaPlantillaNoConoceSeRechaza(t *testing.T) {
	f := deNodos(paso("a", "Inicio", "b"), Nodo{ID: "b", Tipo: "pagina", Nombre: "Home"})
	if problemas := strings.Join(Validar(f), "\n"); !strings.Contains(problemas, `"pagina"`) || !strings.Contains(problemas, "paso o resultado") {
		t.Fatalf("en un proceso una página no va: %s", problemas)
	}
	f.Plantilla = "web"
	f.Nodos[0].Tipo = "pagina"
	if problemas := Validar(f); len(problemas) > 0 {
		t.Fatalf("en la plantilla web, página sí: %v", problemas)
	}
	f.Plantilla = "revista"
	if problemas := strings.Join(Validar(f), "\n"); !strings.Contains(problemas, "plantilla") {
		t.Fatalf("una plantilla que no existe se dice: %s", problemas)
	}
}

func TestOperacion_AceptaLosTiposDeCualquierPlantilla(t *testing.T) {
	if _, err := LeerOperacion(`{"op":"modificar","flujo":"sitio","nodo":"a","tipo":"pagina"}`); err != nil {
		t.Fatalf("el tipo lo decide la plantilla del flujo al guardar: %v", err)
	}
	if _, err := LeerOperacion(`{"op":"modificar","flujo":"sitio","nodo":"a","tipo":"cohete"}`); err == nil {
		t.Fatal("un tipo que ninguna plantilla conoce se rechaza al leer")
	}
}

func TestNombresDeNivel_DeLaPlantilla(t *testing.T) {
	f := deNodos(cadena()...)
	f.Plantilla = "web"
	if n := NombresDeNivel(f); n != [3]string{"Páginas", "Secciones", "Elementos"} {
		t.Fatalf("N1 · N2 · N3 con el nombre de la plantilla: %v", n)
	}
}
