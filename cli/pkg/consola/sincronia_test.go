package consola

import (
	"strings"
	"testing"
	"time"
)

// Sincronía panel–chat (pedido del usuario, 2026-11-01): si la persona acepta, rechaza, cambia o deja sin efecto en el
// chat algo que el panel espera, la IA tiene que saberlo para reflejarlo. qdd le dice, con cada mensaje y solo cuando
// cambió, qué espera el panel de la persona.

func TestEsperas_PedidosFichasYEtapasQueEsperanALaPersona(t *testing.T) {
	root := proyectoConTareas(t)
	p := mustPedir(t, root, "git push origin v1", "Publicar v1 → npm", "")
	got := strings.Join(Esperas(root, enCola), "\n")
	for _, quiere := range []string{"pedido " + p.Tecla + " «Publicar v1 → npm»", "⏳ Barra · PR: merge"} {
		if !strings.Contains(got, quiere) {
			t.Errorf("Esperas no trae %q:\n%s", quiere, got)
		}
	}
	if strings.Contains(got, "Otra") {
		t.Errorf("un ⏳ de otro (no de la persona) no es del panel:\n%s", got)
	}
}

func TestContextoCiclo_SoloCuandoCambiaLoQueEspera(t *testing.T) {
	root := proyectoConTareas(t)
	primero := ContextoCiclo(root, "", enCola)
	if !strings.Contains(primero, "Espera a la persona") || !strings.Contains(primero, "--hecho") {
		t.Fatalf("el primer mensaje lo dice, con cómo reflejarlo:\n%s", primero)
	}
	if again := ContextoCiclo(root, "", enCola.Add(time.Minute)); again != "" {
		t.Fatalf("sin cambios no gasta tokens: %q", again)
	}
	mustPedir(t, root, "gh pr merge 7", "Merge #7", "")
	if nuevo := ContextoCiclo(root, "", enCola.Add(2*time.Minute)); !strings.Contains(nuevo, "«Merge #7»") {
		t.Fatalf("un pedido nuevo vuelve a decirlo:\n%s", nuevo)
	}
}

func TestContextoCiclo_SinNadaQueEsperarNoDiceNada(t *testing.T) {
	root := proyectoConCola(t)
	if got := ContextoCiclo(root, "", enCola); got != "" {
		t.Fatalf("sin esperas: %q", got)
	}
}

func TestResolverEnElChat_HechoPorLaPersonaLiberaAlQueEspera_CCC_C32(t *testing.T) {
	root := proyectoConCola(t)
	primero := mustPedirEnOrden(t, root, "exit 0 # merge", Orden{})
	segundo := mustPedirEnOrden(t, root, "exit 0 # tag", Orden{Despues: []string{primero.ID}})
	if err := HechoPorLaPersona(root, primero.Tecla, "lo mergeó en GitHub", enCola); err != nil {
		t.Fatal(err)
	}
	p := pedidoDe(t, root, primero.ID, enCola)
	if p.Estado != EstadoHecho || !strings.Contains(p.Salida, "lo mergeó en GitHub") {
		t.Fatalf("queda hecho, con lo que dijo: %+v", p)
	}
	if a, _ := BuscarAccion(Acciones(root, false), "pedido:"+segundo.ID); a.Tecla == "" {
		t.Fatal("el que esperaba ya se puede ejecutar")
	}
	if err := HechoPorLaPersona(root, "z", "", enCola); err == nil {
		t.Fatal("un pedido que no existe es error")
	}
}

func TestDescartar_AceptaLaTeclaQueVeLaPersona(t *testing.T) {
	root := proyectoConCola(t)
	p := mustPedir(t, root, "exit 0", "No va", "")
	if err := Descartar(root, p.Tecla); err != nil {
		t.Fatal(err)
	}
	if len(Pedidos(root)) != 0 {
		t.Fatal("descartado por su tecla")
	}
}
