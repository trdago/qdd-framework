package consola

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9: un pedido puede esperar a otros (--despues) o adelantarse a otros (--antes). El panel los muestra en
// ese orden y uno que espera no se puede ejecutar hasta que lo que espera quede hecho.

func mustPedirEnOrden(t *testing.T, root, comando string, orden Orden) Pedido {
	t.Helper()
	p, err := PedirEnOrden(root, comando, "m "+comando, "", orden, enCola)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func comandosDe(cola []Pedido) string {
	var out []string
	for _, p := range cola {
		out = append(out, p.Comando)
	}
	return strings.Join(out, " ")
}

func marcarHecho(t *testing.T, root, id string) {
	t.Helper()
	fin := enCola
	if err := conCola(root, enCola, func(cola []Pedido) ([]Pedido, error) {
		for i := range cola {
			if cola[i].ID == id {
				cola[i].Estado, cola[i].Fin = EstadoHecho, &fin
			}
		}
		return cola, nil
	}); err != nil {
		t.Fatal(err)
	}
}

// FND-080: el panel ofrecía los pedidos en el orden en que se pidieron, aunque unos dependían de otros: la persona
// pulsaba uno antes de tiempo y fallaba (los merges de #14, #15 y #13 antes del tag v2.7.0, 2026-10-30).
func TestPedidos_SeMuestranEnElOrdenDeSusDependencias_FND080(t *testing.T) {
	root := proyectoConCola(t)
	mergeA := mustPedirEnOrden(t, root, "exit 0 # merge-14", Orden{})
	mustPedirEnOrden(t, root, "exit 0 # merge-15", Orden{Despues: []string{mergeA.ID}})
	mustPedirEnOrden(t, root, "exit 0 # tag-261", Orden{Antes: []string{mergeA.ID}})
	mustPedirEnOrden(t, root, "exit 0 # tag-262", Orden{Antes: []string{mergeA.ID}, Despues: []string{"d"}})
	if got := comandosDe(Pedidos(root)); got != "exit 0 # tag-261 exit 0 # tag-262 exit 0 # merge-14 exit 0 # merge-15" {
		t.Fatalf("lo que va antes sube, justo antes de lo que lo espera: %s", got)
	}
	sinOrden := proyectoConCola(t)
	mustPedirEnOrden(t, sinOrden, "exit 0 # uno", Orden{})
	mustPedirEnOrden(t, sinOrden, "exit 0 # dos", Orden{})
	if got := comandosDe(Pedidos(sinOrden)); got != "exit 0 # uno exit 0 # dos" {
		t.Fatalf("sin dependencias, el orden es el de siempre: %s", got)
	}
}

func TestAcciones_UnPedidoQueEsperaNoTieneTeclaYDiceQueEspera_FND080(t *testing.T) {
	root := proyectoConCola(t)
	primero := mustPedirEnOrden(t, root, "exit 0 # primero", Orden{})
	segundo := mustPedirEnOrden(t, root, "exit 0 # segundo", Orden{Despues: []string{"a"}})
	if segundo.Tecla != "b" || len(segundo.Despues) != 1 || segundo.Despues[0] != primero.ID {
		t.Fatalf("la dependencia se guarda por id (se puede dar por tecla) y el pedido tiene su tecla: %+v", segundo)
	}
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+segundo.ID)
	if a.Estado != EstadoEspera || a.Tecla != "" || len(a.Espera) != 1 || a.Espera[0] != "a" {
		t.Fatalf("el panel lo muestra en espera de [a], sin tecla: %+v", a)
	}
	var salida bytes.Buffer
	if codigo := a.Ejecutar(root, "qdd", &salida); codigo != 1 || !strings.Contains(salida.String(), "espera a [a]") {
		t.Fatalf("no corre antes de lo que espera, y dice qué espera: %d %q", codigo, salida.String())
	}
	if Pedidos(root)[1].Estado != EstadoPendiente {
		t.Fatalf("rechazado, sigue pendiente: %+v", Pedidos(root))
	}
	marcarHecho(t, root, primero.ID)
	listo, _ := BuscarAccion(Acciones(root, false), "pedido:"+segundo.ID)
	if listo.Estado != EstadoPendiente || listo.Tecla != "b" || len(listo.Espera) != 0 {
		t.Fatalf("hecho lo que esperaba, se ofrece con su tecla: %+v", listo)
	}
}

func TestPedidos_UnaFallaDeLoQueEsperaLoSigueBloqueando_FND080(t *testing.T) {
	root := proyectoConCola(t)
	falla := mustPedirEnOrden(t, root, "exit 3", Orden{})
	espera := mustPedirEnOrden(t, root, "exit 0 # despues", Orden{Despues: []string{falla.ID}})
	a, _ := BuscarAccion(Acciones(root, false), "pedido:"+falla.ID)
	var salida bytes.Buffer
	a.Ejecutar(root, "qdd", &salida)
	b, _ := BuscarAccion(Acciones(root, false), "pedido:"+espera.ID)
	if b.Estado != EstadoEspera || b.Tecla != "" {
		t.Fatalf("si lo que espera falló, sigue esperando: %+v", b)
	}
}

func TestPedidos_UnHechoQueOtroEsperaNoSeVa_FND080(t *testing.T) {
	root := proyectoConCola(t)
	hecho := mustPedirEnOrden(t, root, "exit 0 # hecho", Orden{})
	mustPedirEnOrden(t, root, "exit 0 # despues", Orden{Despues: []string{hecho.ID}})
	marcarHecho(t, root, hecho.ID)
	if cola := PedidosA(root, enCola.Add(VidaHecho+time.Minute)); len(cola) != 2 {
		t.Fatalf("mientras otro abierto lo espera, el hecho sigue en la cola: %+v", cola)
	}
}

func TestDescartar_LoQueOtroEsperabaLoLibera_FND080(t *testing.T) {
	root := proyectoConCola(t)
	primero := mustPedirEnOrden(t, root, "exit 0 # primero", Orden{})
	segundo := mustPedirEnOrden(t, root, "exit 0 # segundo", Orden{Despues: []string{primero.ID}})
	if err := Descartar(root, primero.ID); err != nil {
		t.Fatal(err)
	}
	cola := Pedidos(root)
	if len(cola) != 1 || len(cola[0].Despues) != 0 {
		t.Fatalf("descartar lo que se esperaba quita la dependencia: %+v", cola)
	}
	if a, _ := BuscarAccion(Acciones(root, false), "pedido:"+segundo.ID); a.Tecla != "b" {
		t.Fatalf("y el que esperaba se puede ejecutar: %+v", a)
	}
}

func TestPedirEnOrden_RechazaCiclosYReferenciasQueNoSirven(t *testing.T) {
	root := proyectoConCola(t)
	uno := mustPedirEnOrden(t, root, "exit 0 # uno", Orden{})
	dos := mustPedirEnOrden(t, root, "exit 0 # dos", Orden{Despues: []string{uno.ID}})
	antes := comandosDe(Pedidos(root))
	casos := map[string]Orden{
		"ciclo":     {Despues: []string{dos.ID}, Antes: []string{uno.ID}},
		"no existe": {Despues: []string{"z"}},
	}
	for nombre, orden := range casos {
		if _, err := PedirEnOrden(root, "exit 0 # tres", "m", "", orden, enCola); err == nil {
			t.Fatalf("%s: se rechaza", nombre)
		}
	}
	marcarHecho(t, root, uno.ID)
	if _, err := PedirEnOrden(root, "exit 0 # tres", "m", "", Orden{Antes: []string{uno.ID}}, enCola); err == nil || !strings.Contains(err.Error(), "ya se ejecutó") {
		t.Fatalf("no se puede ir antes de algo que ya corrió: %v", err)
	}
	if got := comandosDe(Pedidos(root)); got != antes {
		t.Fatalf("un pedido rechazado no cambia la cola: %s", got)
	}
}

func TestPedirEnOrden_PedirloDeNuevoConOrdenReordenaElQueYaEstaba(t *testing.T) {
	root := proyectoConCola(t)
	merge := mustPedirEnOrden(t, root, "exit 0 # merge", Orden{})
	tag := mustPedirEnOrden(t, root, "exit 0 # tag", Orden{})
	otra := mustPedirEnOrden(t, root, "exit 0 # merge", Orden{Despues: []string{tag.ID}})
	if otra.ID != merge.ID || len(otra.Despues) != 1 {
		t.Fatalf("el mismo comando abierto no se duplica, pero suma el orden: %+v", otra)
	}
	if got := comandosDe(Pedidos(root)); got != "exit 0 # tag exit 0 # merge" {
		t.Fatalf("y la cola queda en el orden nuevo: %s", got)
	}
}

func TestPedidos_UnaColaConUnCicloSeLeeSinPerderPedidos(t *testing.T) {
	root := proyectoConCola(t)
	ruta, _ := colaPath(root)
	if err := os.MkdirAll(filepath.Dir(ruta), 0700); err != nil {
		t.Fatal(err)
	}
	cola := `[{"id":"1","comando":"uno","motivo":"m","carpeta":"/","creado":"2026-10-30T12:00:00Z","tecla":"a","estado":"pendiente","despues":["2"]},
{"id":"2","comando":"dos","motivo":"m","carpeta":"/","creado":"2026-10-30T12:00:00Z","tecla":"b","estado":"pendiente","despues":["1","9"]}]`
	if err := os.WriteFile(ruta, []byte(cola), 0600); err != nil {
		t.Fatal(err)
	}
	if got := comandosDe(Pedidos(root)); got != "dos uno" && got != "uno dos" {
		t.Fatalf("una cola editada a mano con un ciclo se lee entera: %s", got)
	}
}
