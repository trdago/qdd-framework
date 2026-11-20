package modules

import (
	"fmt"
	"strings"
	"testing"
)

// MEJ-073 (REP-490): qdd_project_context con 3 rutas devolvió 68.841 caracteres; Claude Code lo rechazó por pasar el
// tope de una herramienta. Las decisiones largas de las fichas eran casi todo. Por defecto salen resumidas a una línea;
// las invariantes, completas; el resto, a pedido con Completo.

// PresupuestoContexto es lo más que puede pesar el contexto de 3 fichas grandes: ~3.000 tokens.
const presupuestoContexto = 12000

func fichaGrande(id, ruta string) Module {
	m := Module{
		ID:     id,
		Nombre: "Módulo " + id,
		Estado: EstadoBorrador,
		Rutas:  []string{ruta},
		ObjetivoOperativo: ObjetivoOperativo{
			Problema:        "Problema de " + id,
			CriterioDeExito: "Criterio de " + id,
			CasoReal:        strings.Repeat("caso real largo ", 40),
		},
		ObjetivoTecnologico: ObjetivoTecnologico{
			Arquitectura: "Arquitectura de " + id,
			Invariantes:  []string{"INVARIANTE-" + id + " " + strings.Repeat("nunca se rompe ", 20)},
		},
		Pruebas: Pruebas{Comandos: []string{"go test ./" + ruta}},
	}
	for i := 0; i < 30; i++ {
		inicio := fmt.Sprintf("DECISION-%s-%02d se decidió así.", id, i)
		m.ObjetivoTecnologico.Decisiones = append(m.ObjetivoTecnologico.Decisiones, inicio+" "+strings.Repeat("con un detalle muy largo ", 24))
	}
	return m
}

func contextoDeTresFichas(t *testing.T, completo bool) (string, []Module) {
	t.Helper()
	root := gitRepo(t)
	var mods []Module
	var rutas []string
	for _, id := range []string{"pantallas", "relay", "panel"} {
		m := fichaGrande(id, "internal/"+id)
		put(t, root, "internal/"+id+"/a.go", "package x\n")
		if _, problemas := Save(root, m); len(problemas) > 0 {
			t.Fatal(problemas)
		}
		mods = append(mods, m)
		rutas = append(rutas, "internal/"+id+"/a.go")
	}
	for i := 0; i < 40; i++ {
		otro := fichaGrande(fmt.Sprintf("otro%02d", i), fmt.Sprintf("otros/o%02d", i))
		put(t, root, otro.Rutas[0]+"/a.go", "package x\n")
		if _, problemas := Save(root, otro); len(problemas) > 0 {
			t.Fatal(problemas)
		}
	}
	return BuildContext(ContextInput{Root: root, Paths: rutas, TestCommands: []string{"go test ./..."}, Completo: completo}), mods
}

func TestContexto_TresFichasGrandesCabenEnElPresupuesto_MEJ073(t *testing.T) {
	ctx, mods := contextoDeTresFichas(t, false)
	t.Logf("contexto liviano de 3 fichas grandes: %d caracteres", len(ctx))
	if len(ctx) > presupuestoContexto {
		t.Fatalf("el contexto de 3 fichas grandes pesa %d caracteres; el presupuesto es %d", len(ctx), presupuestoContexto)
	}
	for _, m := range mods {
		if !strings.Contains(ctx, m.ObjetivoTecnologico.Invariantes[0]) {
			t.Errorf("la invariante de %s sale completa", m.ID)
		}
		for _, d := range m.ObjetivoTecnologico.Decisiones {
			if !strings.Contains(ctx, strings.SplitN(d, ".", 2)[0]) {
				t.Errorf("cada decisión de %s se nombra: falta %q", m.ID, strings.SplitN(d, ".", 2)[0])
			}
		}
	}
	for _, debe := range []string{"Pruebas primero", "go test ./internal/relay", "detalle=completo", "30 decisiones", "dividirla"} {
		if !strings.Contains(ctx, debe) {
			t.Errorf("el contexto liviano no dice %q", debe)
		}
	}
	if strings.Contains(ctx, "otro07") {
		t.Error("con rutas, el mapa de los otros módulos sale a pedido, no en cada llamada")
	}
}

func TestContexto_CompletoTraeLasDecisionesEnteras_MEJ073(t *testing.T) {
	ctx, mods := contextoDeTresFichas(t, true)
	for _, m := range mods {
		if !strings.Contains(ctx, m.ObjetivoTecnologico.Decisiones[29]) || !strings.Contains(ctx, m.ObjetivoOperativo.CasoReal) {
			t.Errorf("completo trae la ficha entera de %s", m.ID)
		}
	}
	if !strings.Contains(ctx, "otro07") {
		t.Error("completo trae el mapa de módulos")
	}
}

func TestContexto_SinRutasTraeElMapa_MEJ073(t *testing.T) {
	root := gitRepo(t)
	put(t, root, "internal/relay/a.go", "package x\n")
	if _, problemas := Save(root, fichaGrande("relay", "internal/relay")); len(problemas) > 0 {
		t.Fatal(problemas)
	}
	ctx := BuildContext(ContextInput{Root: root})
	if !strings.Contains(ctx, "## Mapa de módulos") || !strings.Contains(ctx, "`relay`") {
		t.Fatalf("sin rutas, el mapa general:\n%s", ctx)
	}
}
