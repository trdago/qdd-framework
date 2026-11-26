package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
)

// La aprobación de una celda la da solo la persona: «aprobado» en qdd_estado (o --aprobado) no aprueba, deja un pedido
// 🔑 de la persona con «qdd estado aprobar», que el modo goal no lanza.
// contrato: estado-por-tareas#c53e6daf
// contrato: estado-por-tareas#8e8e07df
func TestEstadoTool_AprobadoDeLaIADejaUnPedidoDeLaPersona(t *testing.T) {
	root := chdirToTempRepo(t)
	if err := estado.Init(root, []string{"Código", "Validación real"}, time.Now()); err != nil {
		t.Fatal(err)
	}
	res, _ := handleEstadoTool(context.Background(), toolRequest(map[string]interface{}{
		"accion": "set", "frente": "deploy", "nombre": "Deploy", "etapa": "Validación real", "estado": "espera",
		"bloqueo": "aprobar el uso", "de_quien": "persona", "requiere_aprobacion": "persona", "aprobado": true,
	}))
	if res.IsError || !strings.Contains(resultText(t, res), "qdd estado aprobar") {
		t.Fatalf("dice que la aprobación queda como pedido de la persona: %q", resultText(t, res))
	}
	e, _ := estado.Load(root)
	for _, f := range e.Frentes {
		if f.Celdas["Validación real"].Aprobado {
			t.Fatal("la IA no aprueba")
		}
	}
	var pedido *consola.Pedido
	for _, p := range consola.Pedidos(root) {
		if strings.HasPrefix(p.Comando, "qdd estado aprobar ") {
			pedido = &p
		}
	}
	if pedido == nil || !pedido.DePersona || pedido.Comando != "qdd estado aprobar 'deploy' 'Validación real'" {
		t.Fatalf("un pedido de la persona con el comando citado: %+v", pedido)
	}
}

func TestEstadoAprobar_SoloConElSiDeLaPersona(t *testing.T) {
	comando := comandoAprobar("deploy", "Validación real")
	nunca := func(string, string) bool { return false }
	if err := autorizarAprobacion(comando, "", nil, nunca); err == nil {
		t.Fatal("sin pedido ni terminal (una IA) no aprueba")
	}
	if err := autorizarAprobacion(comando, "pedido:7", nil, nunca); err == nil {
		t.Fatal("un pedido que la persona no confirmó no aprueba")
	}
	si := func(p, c string) bool { return p == "pedido:7" && c == comando }
	if err := autorizarAprobacion(comando, "pedido:7", nil, si); err != nil {
		t.Fatalf("el pedido que la persona confirmó, con ese comando, aprueba: %v", err)
	}
	if err := autorizarAprobacion(comando, "", func(string) bool { return true }, nunca); err != nil {
		t.Fatalf("en su terminal, la persona aprueba: %v", err)
	}
}
