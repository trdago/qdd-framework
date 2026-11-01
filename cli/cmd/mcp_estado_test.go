package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

func TestEstadoTool_SinArchivoDiceComoCrearlo(t *testing.T) {
	chdirToTempRepo(t)
	res, _ := handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{}))
	if !res.IsError || !strings.Contains(resultText(t, res), "qdd estado init") {
		t.Fatalf("sin archivo debe decir cómo crearlo: %q", resultText(t, res))
	}
}

func TestEstadoTool_SetValidaYVerMuestraLaTabla(t *testing.T) {
	root := chdirToTempRepo(t)
	if err := estado.Init(root, []string{"Código", "Pruebas"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	res, _ := handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{
		"accion": "set", "frente": "login", "nombre": "Login", "etapa": "Código", "estado": "ok",
	}))
	if !res.IsError || !strings.Contains(resultText(t, res), "EPF-C02") {
		t.Fatalf("un ✅ sin evidencia no se escribe: %q", resultText(t, res))
	}
	res, _ = handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{
		"accion": "set", "frente": "login", "nombre": "Login", "etapa": "Código", "estado": "ok", "evidencia": "commit abc123",
	}))
	if res.IsError {
		t.Fatalf("set válido: %q", resultText(t, res))
	}
	res, _ = handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{"accion": "ver"}))
	if !strings.Contains(resultText(t, res), "| Login | ✅ commit abc123 |") {
		t.Fatalf("ver muestra la tabla: %q", resultText(t, res))
	}
	res, _ = handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{"accion": "check"}))
	if res.IsError {
		t.Fatalf("check sin violaciones: %q", resultText(t, res))
	}
}

func TestEstadoTool_CerrarExigeMotivo(t *testing.T) {
	root := chdirToTempRepo(t)
	if err := estado.Init(root, []string{"Código"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{
		"accion": "set", "frente": "x", "nombre": "X", "etapa": "Código", "estado": "falla", "falla": "rompe",
	}))
	res, _ := handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{"accion": "cerrar", "frente": "x"}))
	if !res.IsError {
		t.Fatalf("cerrar sin motivo debe rechazarse: %q", resultText(t, res))
	}
}

// §3.13 (blueprint estado-por-tareas): el agente declara la tarea en curso por MCP, la hace latir y la termina.
func TestEstadoTool_ActivarPulsoYTerminar_EPT_C01(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	root := chdirToTempRepo(t)
	if err := estado.Init(root, []string{"Código"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{
		"accion": "set", "frente": "x", "nombre": "X", "etapa": "Código", "estado": "falla", "falla": "rompe",
	}))
	res, _ := handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{"accion": "activar", "frente": "nada"}))
	if !res.IsError {
		t.Fatalf("no se activa una tarea que no existe: %q", resultText(t, res))
	}
	res, _ = handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{"accion": "activar", "frente": "x"}))
	if res.IsError || len(estado.EnCurso(estado.LeerActividad(root), time.Now())) != 1 {
		t.Fatalf("activar deja la tarea en curso: %q", resultText(t, res))
	}
	res, _ = handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{"accion": "pulso"}))
	if res.IsError || !strings.Contains(resultText(t, res), "x") {
		t.Fatalf("un pulso sin frente late la última activada: %q", resultText(t, res))
	}
	res, _ = handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{"accion": "terminar", "frente": "x"}))
	if res.IsError || len(estado.EnCurso(estado.LeerActividad(root), time.Now())) != 0 {
		t.Fatalf("terminar la apaga: %q", resultText(t, res))
	}
}

// Objetivos y dependencias por la tool (blueprint estado-por-tareas, pregunta «objetivos»).
func TestEstadoTool_ObjetivoYTarea_EPT_O01_EPT_D02(t *testing.T) {
	root := chdirToTempRepo(t)
	if err := estado.Init(root, []string{"Código"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	call := func(args map[string]interface{}) (bool, string) {
		res, _ := handleEstadoTool(context.Background(), toolRequest(args))
		return res.IsError, resultText(t, res)
	}
	if bad, txt := call(map[string]interface{}{"accion": "objetivo", "objetivo": "rel", "nombre": "Publicar"}); !bad {
		t.Fatalf("un objetivo sin criterio no se escribe: %q", txt)
	}
	if bad, txt := call(map[string]interface{}{"accion": "objetivo", "objetivo": "rel", "nombre": "Publicar", "criterio": "5 h de uso", "foco": true}); bad {
		t.Fatalf("con criterio sí: %q", txt)
	}
	call(map[string]interface{}{"accion": "tarea", "frente": "a", "objetivo": "rel"})
	if bad, txt := call(map[string]interface{}{"accion": "tarea", "frente": "b", "objetivo": "rel", "depende_de": "a"}); bad {
		t.Fatalf("una tarea que depende de otra: %q", txt)
	}
	if bad, txt := call(map[string]interface{}{"accion": "set", "frente": "b", "etapa": "Código", "estado": "ok", "evidencia": "abc1234"}); !bad || !strings.Contains(txt, "EPT-D02") {
		t.Fatalf("b no termina antes que a: %q", txt)
	}
	_, txt := call(map[string]interface{}{"accion": "ver"})
	if !strings.Contains(txt, "| 🎯 Publicar · foco |") || !strings.Contains(txt, "| b (espera a a) |") {
		t.Fatalf("ver agrupa por objetivo y dice a quién espera: %q", txt)
	}
	if bad, txt := call(map[string]interface{}{"accion": "objetivo", "objetivo": "rel", "evidencia": "corrida uso-5"}); bad {
		t.Fatalf("se cumple con evidencia: %q", txt)
	}
}
