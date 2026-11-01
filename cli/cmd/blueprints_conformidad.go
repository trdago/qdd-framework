package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'

var blueprintConformanceCmd = &cobra.Command{
	Use:     "conformance",
	Aliases: []string{"conformidad"},
	Short:   "Cada escenario que un módulo debe cumplir de su blueprint tiene una prueba con su id",
	Long: `Mide cada módulo que aplicó un blueprint (con sus respuestas) o que lo implementa como origen (su ficha dice
implementa; cubre todas las opciones): cada escenario debe tener una prueba que lleve su id (con «-» o «_»)
en los archivos de prueba de las rutas de su ficha. Que esas pruebas pasen lo dice el gate (qdd audit).
Sale con 1 si falta alguno.`,
	Run: runBlueprintConformance,
}

func init() {
	blueprintConformanceCmd.Flags().BoolVar(&blueprintJSON, "json", false, "la conformidad en JSON")
	blueprintCmd.AddCommand(blueprintConformanceCmd)
}

func runBlueprintConformance(cmd *cobra.Command, args []string) {
	cs, err := blueprints.Conformar(modulesRoot())
	if err != nil {
		salirCon("%v", err)
	}
	if blueprintJSON {
		imprimirJSON(cs)
		return
	}
	fmt.Print(informeDeConformidad(cs))
	for _, c := range cs {
		if len(c.Faltan) > 0 {
			os.Exit(1)
		}
	}
}

func informeDeConformidad(cs []blueprints.Conformidad) string {
	if len(cs) == 0 {
		return "Ningún módulo aplica ni implementa un blueprint (registro de aplicación o «implementa» en su ficha).\n"
	}
	var s strings.Builder
	faltan := 0
	for _, c := range cs {
		s.WriteString(blueprints.LineaConformidad(c) + "\n")
		faltan += len(c.Faltan)
	}
	if faltan > 0 {
		fmt.Fprintf(&s, "Faltan %d prueba(s): escribe una por escenario, con su id en el nombre, en las rutas de la ficha; pruebas primero.\n", faltan)
	}
	return s.String()
}
