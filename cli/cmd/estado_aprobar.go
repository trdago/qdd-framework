package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Lo que requiere aprobación lo aprueba la persona (estado-por-tareas#c53e6daf): «qdd estado aprobar» corre desde el
// pedido 🔑 que ella confirmó o en su terminal. La IA que marca «aprobado» solo deja ese pedido, que el modo goal no
// lanza.

var estadoAprobarCmd = &cobra.Command{
	Use:   "aprobar <tarea> <etapa>",
	Short: "Da la aprobación que una celda espera: corre desde el pedido 🔑 que la persona confirmó, o en su terminal",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		confirmado := func(pedido, comando string) bool { return consola.PedidoConfirmado(root, pedido, comando, time.Now()) }
		if err := autorizarAprobacion(comandoAprobar(args[0], args[1]), os.Getenv(consola.VarPedido), preguntaDeTerminal(), confirmado); err != nil {
			salirCon("%v", err)
		}
		if err := estado.Aprobar(root, args[0], args[1], time.Now()); err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[+] %s › %s aprobada\n", args[0], args[1])
	},
}

// comandoAprobar es el comando del pedido 🔑, con cada argumento citado (una etapa puede tener espacios).
func comandoAprobar(frente, etapa string) string {
	return "qdd estado aprobar " + citarArgumento(frente) + " " + citarArgumento(etapa)
}

func citarArgumento(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// autorizarAprobacion: el pedido de la persona con este comando y su «s  Sí», o su terminal con la pregunta respondida.
func autorizarAprobacion(comando, pedido string, preguntar func(string) bool, confirmado func(pedido, comando string) bool) error {
	if pedido != "" && confirmado(pedido, comando) {
		return nil
	}
	if pedido == "" && preguntar != nil && preguntar("¿Aprobar? ("+comando+")") {
		return nil
	}
	return errors.New("una aprobación la da la persona: con «s  Sí» en su pedido 🔑 del panel, o en su terminal")
}

// pedirAprobacion deja en el panel el pedido 🔑 de la persona que aprueba esa celda.
func pedirAprobacion(root, frente, etapa string, now time.Time) error {
	motivo := []rune("🔑 Aprobar " + frente + " › " + etapa)
	_, err := consola.PedirEnOrden(root, comandoAprobar(frente, etapa), string(motivo[:min(len(motivo), 60)]), "", consola.Orden{DePersona: true}, now)
	return err
}
