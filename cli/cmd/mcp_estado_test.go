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
