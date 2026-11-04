package cmd

import (
	"fmt"

	"github.com/qdd-framework/qdd/pkg/consola"
)

// Rule: Prohibido usar 'else'
// Modo goal (pkg/consola/goal.go): doctor lo revisa si está activo o su archivo no se lee. GOAL-01: el archivo se lee;
// --fix lo aparta como .ilegible y el modo queda apagado, porque activarlo es una decisión de la persona.

const grupoGoal = "11. Modo goal (qdd goal)"

const itemGoalLegible = "Archivo del modo goal legible (GOAL-01)"

func buildGoalChecks(projectPath string) *CheckGroup {
	g := &CheckGroup{Name: grupoGoal}
	activo, err := consola.RevisarGoal(projectPath)
	if err != nil {
		g.Items = append(g.Items, &CheckItem{Name: itemGoalLegible, Error: err.Error() + " (qdd doctor --fix lo aparta y deja el modo apagado)"})
		return g
	}
	if activo {
		g.Items = append(g.Items, &CheckItem{Name: itemGoalLegible, Success: true})
	}
	return g
}

func repairGoal(projectPath string, g *CheckGroup) {
	for _, item := range g.Items {
		if item.Success {
			continue
		}
		fmt.Println("  [~] Apartando el archivo del modo goal (queda apagado)...")
		if err := consola.ApartarGoal(projectPath); err != nil {
			fmt.Printf("  [!] %v\n", err)
		}
	}
}
