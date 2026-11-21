package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/invitaciones"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0025, fase 4: el asistente es la sesión de Claude del moderador mientras se conversa con un cliente. No edita el
// flujo: deja propuestas de nodos, notas e historias, cada una con el minuto que la respalda, y la persona las aprueba
// una por una en «Para ti» (el panel las pide en su ciclo de propuestas).

var proponerDe, proponerMinuto string

var flujoProponerCmd = &cobra.Command{
	Use:   "proponer <flujo> '<operación JSON>' [--de asistente] [--minuto mm:ss]",
	Short: "Propone un cambio al flujo como asistente: la persona lo aprueba en «Para ti»; no edita el flujo",
	Long: `  qdd flujo proponer venta '{"op":"historia","nodo":"ingreso","como":"vendedor","quiero":"validar el RUT"}' --minuto 12:40
  qdd flujo proponer venta '{"op":"agregar","nodo":"ingreso","nombre":"Validar RUT"}'
  qdd flujo proponer venta '{"op":"nota","nodo":"ingreso","texto":"usar la API del SII"}'

La operación es la del teléfono y el archivo del cliente (agregar, modificar, mover, quitar, conectar, desconectar,
nota, historia); "flujo" se toma del argumento. --minuto es el de la conversación que la respalda.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := ejecutarProponer(cmd, args); err != nil {
			salirCon("%v", err)
		}
	},
}

func ejecutarProponer(cmd *cobra.Command, args []string) error {
	if proponerDe != flujo.RolAsistente {
		return fmt.Errorf("--de %q: por ahora propone solo el asistente; la persona edita el flujo directo", proponerDe)
	}
	operacion, err := conFlujo(args[1], args[0])
	if err != nil {
		return err
	}
	id, err := invitaciones.ProponerDeAsistente(estadoRaiz(), operacion, proponerMinuto, time.Now())
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "[+] %s en la cola: la persona la aprueba o la descarta en «Para ti»; el flujo no cambia hasta entonces.\n", id)
	return nil
}

// conFlujo pone el flujo del argumento en la operación; si ya trae otro, no es para este flujo.
func conFlujo(texto, id string) (json.RawMessage, error) {
	var campos map[string]json.RawMessage
	if err := json.Unmarshal([]byte(texto), &campos); err != nil || campos == nil {
		return nil, errors.New("la operación no es un objeto JSON: 'qdd flujo proponer --help' muestra ejemplos")
	}
	propio, _ := json.Marshal(id)
	previo, trae := campos["flujo"]
	if trae && string(previo) != string(propio) {
		return nil, fmt.Errorf("la operación es para el flujo %s, no para %q", previo, id)
	}
	campos["flujo"] = propio
	return json.Marshal(campos)
}

func init() {
	flujoProponerCmd.Flags().StringVar(&proponerDe, "de", flujo.RolAsistente, "quién propone (asistente)")
	flujoProponerCmd.Flags().StringVar(&proponerMinuto, "minuto", "", "minuto de la conversación que la respalda (mm:ss)")
	flujoCmd.AddCommand(flujoProponerCmd)
}
