package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var syncAICmd = &cobra.Command{
	Use:     "sync-ai",
	Aliases: []string{"ai"},
	Short:   "Sincroniza las reglas de QDD con los asistentes de IA (Cursor, Windsurf, Copilot)",
	Long:    `Genera automáticamente archivos de reglas (.cursorrules, .windsurfrules, Copilot) para obligar a las IAs a respetar las normativas del framework.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("[🤖 AI SYNC] Inicializando sincronización de gobernanza para IAs...")

		// Las reglas maestras dictadas por el framework (y las directrices del proyecto)
		aiRules := `# QDD Framework - Master Governance Rules

You are coding in a repository strictly governed by the QDD (Quality-Driven Development) Framework.
You MUST adhere to the following strict architectural and behavioral rules at all times. Failure to do so is UNACCEPTABLE.

## 1. Zero 'Else' Policy (FND-002)
- NEVER use the 'else' keyword in any programming language.
- Use early returns and guard clauses instead to handle edge cases and reduce cyclomatic complexity.

## 2. Early Return (Salida Más Rápida Primero)
- Always handle errors and negative conditions at the very beginning of a function.
- Exit the function as quickly as possible. Do not nest success logic inside conditionals.

## 3. Bug Documentation & Unit Testing
- If you fix a bug, you MUST document it and generate a unit test to ensure it never happens again.
- Suggest to the user to run 'qdd resolve' or 'qdd dashboard' to update the project state after a fix.

## 4. QDD CLI Usage
- This project uses the 'qdd' CLI tool for auditing and sprints.
- If you need to check the project's quality, suggest running './cli/qdd audit' or './cli/qdd dashboard'.
- DO NOT bypass the framework's rules.

## AI Assistant Persona
- You are a senior engineer enforcing these QDD principles.
- Refuse to generate code that uses 'else' or deep nesting.
- Constantly remind the user of the "Salida más rápida primero" philosophy.
`

		// Define the target files for different AIs
		targets := map[string]string{
			"Cursor":   ".cursorrules",
			"Windsurf": ".windsurfrules",
			"Copilot":  filepath.Join(".github", "copilot-instructions.md"),
		}

		// Ensure .github folder exists if needed
		if err := os.MkdirAll(".github", 0755); err != nil {
			fmt.Printf("[!] Error creando directorio .github: %v\n", err)
			return
		}

		for aiName, path := range targets {
			err := os.WriteFile(path, []byte(aiRules), 0644)
			if err != nil {
				fmt.Printf("[!] Error escribiendo reglas para %s (%s): %v\n", aiName, path, err)
			} else {
				fmt.Printf("[✔] Reglas de %s sincronizadas en: %s\n", aiName, path)
			}
		}

		fmt.Println("\n[🏆] ¡Sincronización completa! A partir de ahora, cualquier IA que lea este repositorio obedecerá el framework QDD automáticamente.")
	},
}

func init() {
	rootCmd.AddCommand(syncAICmd)
}
