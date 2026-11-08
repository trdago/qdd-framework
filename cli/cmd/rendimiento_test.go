package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/rendimiento"
)

// Rule: Prohibido usar 'else'
// qdd rendimiento (pedido de la persona, 2026-11-07): doctor dice si la versión que corre se midió en este equipo y
// cómo salió; sin medir no es una falla, pero lo dice.

func TestDoctor_RendimientoSinMedirLoDiceSinFallar(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	it := checkRendimiento()
	if !it.Success || !strings.Contains(it.Error, "sin medir") || !strings.Contains(it.Error, "qdd rendimiento") {
		t.Fatalf("sin medir se dice, y cómo medir: %+v", it)
	}
}

func TestDoctor_RendimientoQueDejoProcesosEsUnaFalla(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	uno := 1.0
	inf := rendimiento.Informe{Version: Version, SO: "linux", Cuando: time.Now(), Medidas: []rendimiento.Medida{rendimiento.Evaluar(rendimiento.MedidaZombies, &uno, "linux")}}
	if err := rendimiento.Guardar(inf); err != nil {
		t.Fatal(err)
	}
	it := checkRendimiento()
	if it.Success || !strings.Contains(it.Error, "1 falla") {
		t.Fatalf("lo medido con una falla se ve como falla: %+v", it)
	}
}

func TestRendimiento_ElContextoDelMCPSeMideSinLanzarNada(t *testing.T) {
	m := medidaContexto()
	if m.Valor == nil || *m.Valor <= 0 || m.Estado != rendimiento.EstadoOK {
		t.Fatalf("tools/list se mide en el proceso y cabe en su presupuesto: %+v", m)
	}
}

func TestRendimiento_ElCodigoSeparaProcesosDeLentitud(t *testing.T) {
	uno, mucho := 1.0, 1e9
	con := func(ms ...rendimiento.Medida) rendimiento.Informe { return rendimiento.Informe{Medidas: ms} }
	lento := rendimiento.Evaluar(rendimiento.MedidaCPU, &mucho, "linux")
	zombie := rendimiento.Evaluar(rendimiento.MedidaZombies, &uno, "linux")
	if c := codigoRendimiento(con()); c != 0 {
		t.Fatalf("todo cupo: %d", c)
	}
	if c := codigoRendimiento(con(lento)); c != 2 {
		t.Fatalf("solo lento: %d", c)
	}
	if c := codigoRendimiento(con(lento, zombie)); c != 1 {
		t.Fatalf("dejó procesos: %d", c)
	}
}
