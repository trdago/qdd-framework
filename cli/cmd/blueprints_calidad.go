package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'

var blueprintQualityCmd = &cobra.Command{
	Use:     "quality [id|carpeta...]",
	Aliases: []string{"calidad"},
	Short:   "Mide cada blueprint como receta: defectos (⚠), mejoras que lo acortan sin quitar nada (✂) y tokens",
	Run:     runBlueprintQuality,
}

func init() {
	blueprintQualityCmd.Flags().BoolVar(&blueprintJSON, "json", false, "la medición en JSON")
	blueprintCmd.AddCommand(blueprintQualityCmd)
}

func runBlueprintQuality(cmd *cobra.Command, args []string) {
	medidas, fuente, err := blueprints.Medir(modulesRoot(), args)
	if err != nil {
		salirCon("%v", err)
	}
	if blueprintJSON {
		imprimirJSON(medidas)
		return
	}
	fmt.Print(informeDeCalidad(medidas, fuente, "qdd blueprint contribute <id>", "qdd blueprint propose"))
	for _, c := range medidas {
		if c.Defectos > 0 {
			os.Exit(1)
		}
	}
}

// informeDeCalidad es la medición en estilo chat: una línea por blueprint y sus hallazgos debajo.
func informeDeCalidad(medidas []blueprints.Calidad, fuente, aportar, proponer string) string {
	var s strings.Builder
	fmt.Fprintf(&s, "Calidad de la receta (%s) · ⚠ defecto −%d · ✂ mejora −%d · tokens ≈ caracteres/4\n", fuente, 10, 3)
	pendientes := 0
	for _, c := range medidas {
		s.WriteString("- " + blueprints.LineaCalidad(c) + "\n")
		for _, h := range c.Hallazgos {
			s.WriteString("  " + h.Texto() + "\n")
		}
		s.WriteString(textoDeEntrevista(c))
		pendientes += len(c.Hallazgos)
	}
	if pendientes == 0 {
		s.WriteString("✅ Recetas completas: nada que arreglar ni acortar.\n")
		return s.String()
	}
	fmt.Fprintf(&s, "Arreglo: %s, corrige (ningún arreglo quita una regla, opción ni escenario) y, con la persona, %s; propose rechaza un aporte que agrega defectos.\n", aportar, proponer)
	return s.String()
}
