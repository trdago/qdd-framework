package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var sprintCmd = &cobra.Command{
	Use:     "sprint [número]",
	Aliases: []string{"SPRINT"},
	Short:   "Inicializa un nuevo Sprint de trabajo",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		sprintNum := args[0]
		fmt.Printf("🏃 Inicializando Sprint %s...\n", sprintNum)

		qddDir := filepath.Join(".", ".qdd")
		sprintsDir := filepath.Join(qddDir, "sprints")
		
		if err := os.MkdirAll(sprintsDir, 0755); err != nil {
			fmt.Printf("[!] Error creando directorio de sprints: %v\n", err)
			return
		}

		sprintFile := filepath.Join(sprintsDir, fmt.Sprintf("sprint-%s.md", sprintNum))
		
		if _, err := os.Stat(sprintFile); !os.IsNotExist(err) {
			fmt.Printf("[!] El sprint %s ya existe.\n", sprintNum)
			return
		}

		content := fmt.Sprintf(`# Sprint %s

## Objetivos (Sprint Goal)
- [ ] Definir objetivos aquí

## Tareas (Backlog)
- [ ] Tarea 1
- [ ] Tarea 2

## Métricas de Calidad Iniciales
- **QDD Score de Entrada:** (Ejecuta 'qdd score')
- **Deuda Técnica Inicial:** (Ejecuta 'qdd status')

---
*Gobernanza QDD: Todo código añadido en este sprint debe contar con evidencia (EV-FND) y pruebas unitarias.*
`, sprintNum)

		if err := os.WriteFile(sprintFile, []byte(content), 0644); err != nil {
			fmt.Printf("[!] Error escribiendo el sprint: %v\n", err)
			return
		}

		fmt.Printf("✅ Archivo de Sprint creado exitosamente en: %s\n", sprintFile)
		fmt.Println("[✔] Puedes comenzar a planificar tus tareas.")
	},
}

func init() {
	rootCmd.AddCommand(sprintCmd)
}
