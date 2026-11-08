package flujo

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/materializar"
)

func paso(id, nombre string, siguen ...string) Nodo {
	return Nodo{ID: id, Tipo: TipoPaso, Nombre: nombre, Siguen: siguen}
}

func resultado(id, nombre string, siguen ...string) Nodo {
	return Nodo{ID: id, Tipo: TipoResultado, Nombre: nombre, Siguen: siguen}
}

func deNodos(nodos ...Nodo) Flujo {
	return Flujo{Formato: Formato, ID: "pago", Nombre: "Pago", Nodos: nodos}
}

func sinEstado(string) string { return "" }

func TestValidar_DiceCadaProblema(t *testing.T) {
	casos := map[string]struct {
		f    Flujo
		dice string
	}{
		"formato":     {Flujo{ID: "a", Nombre: "A", Nodos: []Nodo{paso("x", "X")}}, "formato"},
		"id ruta":     {Flujo{Formato: 1, ID: "../a", Nombre: "A", Nodos: []Nodo{paso("x", "X")}}, "id"},
		"sin nombre":  {Flujo{Formato: 1, ID: "a", Nodos: []Nodo{paso("x", "X")}}, "nombre"},
		"sin nodos":   {Flujo{Formato: 1, ID: "a", Nombre: "A"}, "nodo"},
		"repetido":    {deNodos(paso("x", "X"), paso("x", "Y")), "repetido"},
		"tipo":        {deNodos(Nodo{ID: "x", Tipo: "rombo", Nombre: "X"}), "rombo"},
		"nodo vacío":  {deNodos(paso("x", " ")), "sin nombre"},
		"destino":     {deNodos(paso("x", "X", "no")), "no"},
		"inalcanzado": {deNodos(paso("x", "X"), paso("y", "Y")), "no se llega"},
	}
	for nombre, c := range casos {
		t.Run(nombre, func(t *testing.T) {
			problemas := Validar(c.f)
			if !strings.Contains(strings.Join(problemas, "\n"), c.dice) {
				t.Fatalf("esperaba %q en %q", c.dice, problemas)
			}
		})
	}
	if p := Validar(deNodos(paso("x", "X", "y"), resultado("y", "Y", "x"))); len(p) != 0 {
		t.Fatalf("un ciclo es válido: %q", p)
	}
}

func TestGuardar_NoEscribeUnFlujoInvalidoYSeLeeIgual(t *testing.T) {
	root := t.TempDir()
	if err := Guardar(root, deNodos(paso("x", "X", "no"))); err == nil {
		t.Fatal("un flujo inválido no se guarda")
	}
	if _, err := os.Stat(filepath.Join(root, Carpeta, "pago.yaml")); err == nil {
		t.Fatal("no debe quedar el archivo")
	}
	f := deNodos(paso("x", "X", "y"), resultado("y", "Y"))
	if err := Guardar(root, f); err != nil {
		t.Fatal(err)
	}
	leido, err := Cargar(root, "pago")
	if err != nil {
		t.Fatal(err)
	}
	if leido.Nombre != "Pago" || len(leido.Nodos) != 2 || leido.Nodos[0].Siguen[0] != "y" {
		t.Fatalf("leído: %+v", leido)
	}
	if _, err := Cargar(root, "../pago"); err == nil {
		t.Fatal("un id con ruta no se lee")
	}
	_, err = Cargar(root, "nada")
	if err == nil || !strings.Contains(err.Error(), "no hay un flujo") || strings.Contains(err.Error(), root) {
		t.Fatalf("dice qué falta, sin la ruta: %v", err)
	}
}

func TestListar_ResumeCadaFlujoYCambiaLaHuella(t *testing.T) {
	root := t.TempDir()
	if l := Listar(root, sinEstado); len(l) != 0 {
		t.Fatalf("sin carpeta no hay flujos: %+v", l)
	}
	if err := Guardar(root, deNodos(paso("x", "X"))); err != nil {
		t.Fatal(err)
	}
	antes := Listar(root, sinEstado)
	if len(antes) != 1 || antes[0].ID != "pago" || antes[0].Nombre != "Pago" || antes[0].Nodos != 1 || antes[0].Huella == "" {
		t.Fatalf("lista: %+v", antes)
	}
	if err := Guardar(root, deNodos(paso("x", "Otro"))); err != nil {
		t.Fatal(err)
	}
	if despues := Listar(root, sinEstado); despues[0].Huella == antes[0].Huella {
		t.Fatal("un cambio cambia la huella")
	}
	if err := os.WriteFile(filepath.Join(root, Carpeta, "roto.yaml"), []byte("nodos: ["), 0644); err != nil {
		t.Fatal(err)
	}
	l := Listar(root, sinEstado)
	if len(l) != 2 || l[1].ID != "roto" || l[1].Problema == "" {
		t.Fatalf("un archivo roto se ve, no desaparece: %+v", l)
	}
}

// Una caja muestra el estado de su tarea: si cambia, cambia lo que se dibuja, aunque el archivo sea el mismo.
func TestListar_LaHuellaCambiaSiCambiaElEstadoDeSuTarea(t *testing.T) {
	root := t.TempDir()
	if err := Guardar(root, deNodos(Nodo{ID: "a", Tipo: TipoPaso, Nombre: "A", Tarea: "rc"})); err != nil {
		t.Fatal(err)
	}
	uno := Listar(root, func(string) string { return "⏳" })[0].Huella
	dos := Listar(root, func(string) string { return "✅" })[0].Huella
	if uno == dos {
		t.Fatal("el estado de la tarea es parte de lo que se dibuja")
	}
}

func texto(f Flujo, ancho int) string {
	return strings.Join(Dibujar(f, ancho, sinEstado), "\n")
}

func TestDibujar_UnaCadenaHaciaAbajo(t *testing.T) {
	f := deNodos(resultado("a", "Llega"), Nodo{ID: "b", Tipo: TipoPaso, Nombre: "Valida", Descripcion: "rechaza si vence", Siguen: []string{"c"}}, resultado("c", "Fin"))
	f.Nodos[0].Siguen = []string{"b"}
	s := texto(f, 40)
	for _, parte := range []string{"( Llega )", "Valida", "rechaza si vence", "( Fin )", "┌", "└", "│"} {
		if !strings.Contains(s, parte) {
			t.Fatalf("falta %q:\n%s", parte, s)
		}
	}
	if strings.Index(s, "Llega") > strings.Index(s, "Valida") || strings.Index(s, "Valida") > strings.Index(s, "Fin") {
		t.Fatalf("va hacia abajo:\n%s", s)
	}
}

func TestDibujar_DosSalidasLadoALadoSiCaben(t *testing.T) {
	f := deNodos(paso("a", "Decide", "b", "c"), resultado("b", "Sí"), resultado("c", "No"))
	lineas := Dibujar(f, 40, sinEstado)
	var enLaMisma bool
	for _, l := range lineas {
		enLaMisma = enLaMisma || (strings.Contains(l, "Sí") && strings.Contains(l, "No"))
	}
	if !enLaMisma {
		t.Fatalf("caben lado a lado:\n%s", strings.Join(lineas, "\n"))
	}
	if !strings.Contains(strings.Join(lineas, "\n"), "┴") {
		t.Fatalf("la bifurcación se dibuja:\n%s", strings.Join(lineas, "\n"))
	}
}

func TestDibujar_SiNoCabenVanUnaDebajoDeOtraConSangria(t *testing.T) {
	f := deNodos(paso("a", "Decide", "b", "c", "d"), resultado("b", "Uno"), resultado("c", "Dos"), resultado("d", "Tres"))
	lineas := Dibujar(f, 24, sinEstado)
	s := strings.Join(lineas, "\n")
	for _, l := range lineas {
		if strings.Contains(l, "Uno") && strings.Contains(l, "Dos") {
			t.Fatalf("no caben lado a lado:\n%s", s)
		}
	}
	for _, parte := range []string{"├─", "└─", "Uno", "Dos", "Tres"} {
		if !strings.Contains(s, parte) {
			t.Fatalf("falta %q:\n%s", parte, s)
		}
	}
}

func TestDibujar_UnaBifurcacionQueSeJuntaDibujaElEncuentroUnaVez(t *testing.T) {
	f := deNodos(paso("a", "Decide", "b", "c"), paso("b", "Rama uno", "d"), paso("c", "Rama dos", "d"), resultado("d", "Junta"))
	s := texto(f, 60)
	if strings.Count(s, "Junta") != 1 || strings.Contains(s, "sigue en") {
		t.Fatalf("el encuentro va una vez, debajo de las dos ramas:\n%s", s)
	}
	if strings.Index(s, "Rama dos") > strings.Index(s, "Junta") {
		t.Fatalf("el encuentro va debajo:\n%s", s)
	}
	if !strings.Contains(s, "└") || !strings.Contains(s, "┬") {
		t.Fatalf("las ramas se unen:\n%s", s)
	}
}

func TestDibujar_UnaRamaDirectaAlEncuentro(t *testing.T) {
	f := deNodos(paso("a", "Decide", "b", "c"), paso("b", "Rama", "c"), resultado("c", "Junta"))
	s := texto(f, 40)
	if strings.Count(s, "( Junta )") != 1 || strings.Index(s, "Rama") > strings.Index(s, "( Junta )") {
		t.Fatalf("el salto directo no repite el encuentro:\n%s", s)
	}
}

func TestDibujar_UnCicloDiceAQueVuelve(t *testing.T) {
	f := deNodos(paso("a", "Intenta", "b"), paso("b", "Revisa", "a"))
	s := texto(f, 40)
	if strings.Count(s, "Intenta") != 2 || !strings.Contains(s, "↺ vuelve a Intenta") {
		t.Fatalf("un ciclo se dice:\n%s", s)
	}
}

func TestDibujar_UnaTareaMuestraSuEstado(t *testing.T) {
	f := deNodos(Nodo{ID: "a", Tipo: TipoPaso, Nombre: "Publicar", Tarea: "rc"})
	lineas := Dibujar(f, 40, func(tarea string) string {
		if tarea != "rc" {
			t.Fatalf("pide otra tarea: %q", tarea)
		}
		return "✅ ⏳"
	})
	if !strings.Contains(strings.Join(lineas, "\n"), "✅ ⏳") {
		t.Fatalf("la caja muestra el estado:\n%s", strings.Join(lineas, "\n"))
	}
}

func TestDibujar_UnTextoLargoSeParteDentroDeLaCaja(t *testing.T) {
	f := deNodos(Nodo{ID: "a", Tipo: TipoPaso, Nombre: "Un nombre bastante largo que no cabe en una línea", Descripcion: strings.Repeat("x", 60)})
	for _, l := range Dibujar(f, 30, sinEstado) {
		if materializar.Ancho(l) > 30 {
			t.Fatalf("se pasa del ancho (%d): %q", materializar.Ancho(l), l)
		}
	}
}

// Con cualquier forma: cada nodo aparece, y ninguna línea pasa del ancho salvo en lo más angosto.
func TestDibujar_CualquierFormaMuestraCadaNodo(t *testing.T) {
	azar := rand.New(rand.NewSource(1))
	for ronda := 0; ronda < 400; ronda++ {
		f := flujoAlAzar(azar)
		ancho := 40 + azar.Intn(41)
		s := strings.Join(Dibujar(f, ancho, sinEstado), "\n")
		for _, n := range f.Nodos {
			if !strings.Contains(s, n.Nombre) {
				t.Fatalf("ronda %d, ancho %d: falta %s\n%+v\n%s", ronda, ancho, n.Nombre, f.Nodos, s)
			}
		}
	}
}

// Lo que se dibuja cabe en el ancho del panel: una rama no empuja el dibujo fuera de la pantalla.
func TestDibujar_NoSePasaDelAncho(t *testing.T) {
	azar := rand.New(rand.NewSource(2))
	for ronda := 0; ronda < 400; ronda++ {
		f := flujoAlAzar(azar)
		for _, ancho := range []int{40, 60, 80} {
			for _, l := range Dibujar(f, ancho, sinEstado) {
				if materializar.Ancho(l) > ancho {
					t.Fatalf("ronda %d: %d columnas en %d:\n%s", ronda, materializar.Ancho(l), ancho, strings.Join(Dibujar(f, ancho, sinEstado), "\n"))
				}
			}
		}
	}
}

func TestDibujar_LasRamasConSangriaCuelganDelNodoSinEmpujarlo(t *testing.T) {
	f := deNodos(Nodo{ID: "a", Tipo: TipoPaso, Nombre: "Un nodo ancho que decide por dónde sigue todo", Siguen: []string{"b", "c", "d"}},
		paso("b", "Una rama con nombre largo", "e"), paso("c", "Otra rama con nombre largo", "e"), paso("d", "La tercera rama larga", "e"),
		paso("e", "Un encuentro con un nombre muy largo también"))
	lineas := Dibujar(f, 34, sinEstado)
	s := strings.Join(lineas, "\n")
	for _, l := range lineas {
		if materializar.Ancho(l) > 34 {
			t.Fatalf("%d columnas:\n%s", materializar.Ancho(l), s)
		}
	}
	if strings.Count(s, "encuentro") != 1 || !strings.Contains(s, "├─") {
		t.Fatalf("forma:\n%s", s)
	}
}

func flujoAlAzar(azar *rand.Rand) Flujo {
	total := 2 + azar.Intn(11)
	nodos := make([]Nodo, total)
	for i := range nodos {
		nodos[i] = paso(fmt.Sprintf("n%d", i), fmt.Sprintf("N%02d", i))
	}
	for i := 1; i < total; i++ {
		padre := azar.Intn(i)
		nodos[padre].Siguen = append(nodos[padre].Siguen, nodos[i].ID)
	}
	for extra := azar.Intn(5); extra > 0; extra-- {
		a, b := azar.Intn(total), azar.Intn(total)
		if contiene(nodos[a].Siguen, nodos[b].ID) {
			continue
		}
		nodos[a].Siguen = append(nodos[a].Siguen, nodos[b].ID)
	}
	return deNodos(nodos...)
}

func TestDibujar_CincuentaNodosEsRapido(t *testing.T) {
	nodos := make([]Nodo, 50)
	for i := range nodos {
		nodos[i] = paso(fmt.Sprintf("n%d", i), fmt.Sprintf("Paso %d", i))
		if i < 49 {
			nodos[i].Siguen = []string{fmt.Sprintf("n%d", i+1)}
		}
	}
	if len(Dibujar(deNodos(nodos...), 60, sinEstado)) < 50 {
		t.Fatal("dibuja los 50")
	}
}

// Un flujo viaja entero en un mensaje al teléfono, y Supabase descarta en silencio lo que pasa de 256 KB en el plan Free.
func TestFlujo_ElQuePesaDeMasNoSeGuardaYSeVeConSuProblema(t *testing.T) {
	root := t.TempDir()
	pesado := Flujo{Formato: 1, ID: "pesado", Nombre: "Pesado", Nodos: []Nodo{
		{ID: "a", Tipo: TipoResultado, Nombre: "A", Descripcion: strings.Repeat("x", MaxBytes)},
	}}
	if err := Guardar(root, pesado); err == nil || !strings.Contains(err.Error(), "tope") {
		t.Fatalf("no se guarda y dice el tope: %v", err)
	}
	if _, err := os.Stat(ruta(root, "pesado")); err == nil {
		t.Fatal("un flujo que pesa de más no se escribe")
	}
	os.MkdirAll(filepath.Join(root, Carpeta), 0755)
	datos := "formato: 1\nid: pesado\nnombre: Pesado\nnodos:\n  - {id: a, tipo: resultado, nombre: A, descripcion: " + strings.Repeat("x", MaxBytes) + "}\n"
	os.WriteFile(ruta(root, "pesado"), []byte(datos), 0644)
	lista := Listar(root, func(string) string { return "" })
	if len(lista) != 1 || !strings.Contains(lista[0].Problema, "tope") {
		t.Fatalf("un archivo que pesa de más se ve con su problema: %+v", lista)
	}
}
