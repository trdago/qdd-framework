package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd consola fin-de-turno (MEJ-026, issue #150): el mod de Claude Code lo llama al terminar cada turno de la
// conversación principal con el último mensaje de la IA y las herramientas que llamó. Si la IA dejó a la persona una
// pregunta o un comando solo en el chat, responde qué le falta dejar en el panel y el mod le devuelve el turno una vez.

var consolaFinDeTurnoCmd = &cobra.Command{
	Use:    "fin-de-turno",
	Short:  "Lee el turno que terminó (JSON por la entrada estándar) y dice qué le falta dejar a la IA en el panel",
	Hidden: true,
	Args:   cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(finDeTurno(estadoRaiz(), os.Stdin, os.Stdout))
	},
}

func init() {
	consolaCmd.AddCommand(consolaFinDeTurnoCmd)
}

// finDeTurno escribe {"devolver": "<qué falta>"}, vacío si nada o si el proyecto no usa QDD (no hay panel donde dejarlo).
func finDeTurno(root string, entrada io.Reader, salida io.Writer) int {
	var t consola.TurnoTerminado
	if err := json.NewDecoder(entrada).Decode(&t); err != nil {
		fmt.Fprintf(os.Stderr, "[!] el turno no se pudo leer: %v\n", err)
		return 1
	}
	devolver := ""
	if _, err := os.Stat(filepath.Join(root, ".qdd")); err == nil {
		devolver = consola.QueFaltaAlTerminar(t)
	}
	b, _ := json.Marshal(map[string]string{"devolver": devolver})
	fmt.Fprintln(salida, string(b))
	return 0
}
