package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/invitaciones"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 4: en una conversación con el cliente, la IA no edita el flujo: propone con
// 'qdd flujo proponer <flujo> --de asistente [--minuto mm:ss] <operación>' y la persona aprueba cada una en «Para ti».

func proponer(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var salida bytes.Buffer
	flujoProponerCmd.SetOut(&salida)
	t.Cleanup(func() { proponerDe, proponerMinuto = "asistente", "" })
	err := ejecutarProponer(flujoProponerCmd, args)
	return salida.String(), err
}

func TestProponer_DejaLaPropuestaEnLaColaSinTocarElFlujo(t *testing.T) {
	root := proyectoConFlujo(t)
	t.Setenv("QDD_HOME", t.TempDir())
	proponerMinuto = "03:15"
	salida, err := proponer(t, "pago", `{"op":"historia","nodo":"a","quiero":"pagar con transferencia"}`)
	if err != nil || !strings.Contains(salida, "ASI-1") || !strings.Contains(salida, "Para ti") {
		t.Fatalf("dice su id y dónde se aprueba: %q %v", salida, err)
	}
	l, _ := invitaciones.Leer(root)
	if len(l.Propuestas) != 1 || l.Propuestas[0].Flujo != "pago" || l.Propuestas[0].Origen != "transcripcion@03:15" {
		t.Fatalf("en la cola, con el flujo del argumento y su minuto: %+v", l.Propuestas)
	}
	if _, err := proponer(t, "pago", `{"op":"quitar","flujo":"otro","nodo":"a"}`); err == nil {
		t.Fatal("una operación de otro flujo no entra")
	}
	proponerDe = "persona"
	if _, err := proponer(t, "pago", `{"op":"quitar","nodo":"a"}`); err == nil {
		t.Fatal("por ahora solo propone el asistente: la persona edita directo")
	}
	if _, err := proponer(t, "pago", `no es json`); err == nil {
		t.Fatal("lo que no es una operación no entra")
	}
}
