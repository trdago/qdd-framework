package remoto

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// FND-209 (acordado con qdd-movile, 2026-11-22): el estado completo de 12 proyectos pesaba 184 KB de los 256 KB que
// Realtime acepta, y Realtime descarta sin aviso lo que pasa. A un teléfono que declara estado_por_partes, un estado
// completo que pasa TopeParte le llega en partes: cada una con la cabecera completa, el mismo seq, parte/de y solo sus
// proyectos. La app lo arma con todas las partes del mismo seq y nunca dibuja uno a medias. Un teléfono sin la
// capacidad lo recibe entero, como antes.

func estadoDePrueba(proyectos int, relleno int) Estado {
	e := Estado{Version: VersionProtocolo, QDD: "v", Maquina: Maquina{Nombre: "pc", SO: "linux"}, En: time.Unix(0, 0).UTC(),
		Seq: 7, FueraDelPlan: 2, Limites: &consola.Limites{}, Proyectos: []ProyectoEstado{}}
	for i := range proyectos {
		e.Proyectos = append(e.Proyectos, ProyectoEstado{Clave: string(rune('a' + i)), Nombre: strings.Repeat("x", relleno)})
	}
	return e
}

func pesoDe(t *testing.T, e Estado) int {
	t.Helper()
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	return len(b)
}

func TestPartes_CadaParteCabeYLlevaLaCabeceraCompleta_FND209(t *testing.T) {
	e := estadoDePrueba(5, 1000)
	partes := partirEstado(e, 2500)
	if len(partes) != 3 {
		t.Fatalf("5 proyectos de ~1 KB en partes de 2,5 KB son 3 partes: %d", len(partes))
	}
	var claves []string
	for i, p := range partes {
		if p.Parte != i+1 || p.De != 3 || p.Seq != 7 || p.QDD != "v" || p.FueraDelPlan != 2 || p.Limites == nil || p.Maquina.Nombre != "pc" {
			t.Fatalf("la parte %d lleva la cabecera completa y su número: %+v", i+1, p)
		}
		if pesoDe(t, p) > 2500 {
			t.Fatalf("la parte %d cabe en el tope: %d", i+1, pesoDe(t, p))
		}
		for _, pr := range p.Proyectos {
			claves = append(claves, pr.Clave)
		}
	}
	if strings.Join(claves, "") != "abcde" {
		t.Fatalf("todos los proyectos, una vez y en orden: %v", claves)
	}
}

func TestPartes_UnEstadoQueCabeVaEnteroSinParte_FND209(t *testing.T) {
	e := estadoDePrueba(2, 100)
	partes := partirEstado(e, 100_000)
	if len(partes) != 1 || partes[0].Parte != 0 || partes[0].De != 0 {
		t.Fatalf("cabe: un solo mensaje, sin parte ni de: %+v", partes)
	}
}

func TestPartes_UnProyectoMasGrandeQueElTopeVaSolo_FND209(t *testing.T) {
	e := estadoDePrueba(3, 100)
	e.Proyectos[1].Nombre = strings.Repeat("y", 5000)
	partes := partirEstado(e, 2000)
	if len(partes) != 3 || len(partes[1].Proyectos) != 1 || partes[1].Proyectos[0].Clave != "b" {
		t.Fatalf("el grande va solo en su parte, sin perderse: %+v", partes)
	}
}

func partesEnviadas(t *testing.T, tr *transporteFalso, n int) []Estado {
	t.Helper()
	var out []Estado
	for len(out) < n {
		out = append(out, estadoEnviado(t, tr.esperar(t, 10*time.Second, "estado")))
	}
	return out
}

func TestPartes_ConTodosLosTelefonosConLaCapacidadElEstadoVaEnPartes_FND209(t *testing.T) {
	homeTemporal(t)
	for _, n := range []string{"p1", "p2", "p3"} {
		proyectoConLogrado(t, n, 1, "ok", time.Now())
	}
	viejo := TopeParte
	TopeParte = 1
	t.Cleanup(func() { TopeParte = viejo })
	var pesos []int
	_, tr, _, _ := bucleDePrueba(t, func(b *Bucle) { b.AlPesar = func(n int) { pesos = append(pesos, n) } })
	tr.eventos <- telefonoCon("d1", CapacidadEstadoPorPartes)
	partes := partesEnviadas(t, tr, 3)
	for i, p := range partes {
		if p.Parte != i+1 || p.De != 3 || p.Seq != partes[0].Seq || len(p.Proyectos) != 1 {
			t.Fatalf("tres partes del mismo seq, una por proyecto: %+v", p)
		}
	}
	if len(pesos) == 0 || pesos[0] >= pesoDe(t, Estado{Proyectos: append(append(partes[0].Proyectos, partes[1].Proyectos...), partes[2].Proyectos...)}) {
		t.Fatalf("lo que se pesa es la parte más grande, que es lo que Realtime mide: %v", pesos)
	}
}

func TestPartes_UnTelefonoSinLaCapacidadRecibeElEstadoEntero_FND209(t *testing.T) {
	homeTemporal(t)
	for _, n := range []string{"p1", "p2"} {
		proyectoConLogrado(t, n, 1, "ok", time.Now())
	}
	viejo := TopeParte
	TopeParte = 1
	t.Cleanup(func() { TopeParte = viejo })
	_, tr, _, _ := bucleDePrueba(t)
	tr.eventos <- juntarTelefonos(telefonoCon("d1", CapacidadEstadoPorPartes), telefonoCon("d2"))
	e := estadoEnviado(t, tr.esperar(t, 10*time.Second, "estado"))
	if e.Parte != 0 || e.De != 0 || len(e.Proyectos) != 2 {
		t.Fatalf("uno no sabe armar partes: el estado va entero: parte=%d de=%d proyectos=%d", e.Parte, e.De, len(e.Proyectos))
	}
}

// El ejemplo compartido con qdd-movile (testdata/estado_partes.json, como esperas.json): tres partes de un estado de
// ejemplo, tal como las manda el PC. Si cambia la forma, esta prueba falla y el fixture de la app también debe cambiar.
// Se regenera con QDD_ACTUALIZAR_FIXTURE=1.
func TestPartes_ElEjemploCompartidoEsLoQueMandaElPC_FND209(t *testing.T) {
	e := estadoDePrueba(5, 40)
	for i := range e.Proyectos {
		e.Proyectos[i].Nombre = "Proyecto " + e.Proyectos[i].Clave
		e.Proyectos[i].Ruta = "/home/persona/proyectos/" + e.Proyectos[i].Clave
	}
	partes := partirEstado(e, 350)
	got, err := json.MarshalIndent(partes, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	ruta := "testdata/estado_partes.json"
	if os.Getenv("QDD_ACTUALIZAR_FIXTURE") == "1" {
		if err := os.WriteFile(ruta, append(got, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(want)) != string(got) {
		t.Fatalf("el ejemplo compartido no es lo que manda el PC:\n%s", got)
	}
	if len(partes) != 3 {
		t.Fatalf("el ejemplo muestra tres partes: %d", len(partes))
	}
}
