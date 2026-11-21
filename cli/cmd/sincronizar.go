package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd sincronizar: en el PC, la misma revisión que el botón «Sincronizar» del teléfono (pkg/remoto/sincronizar.go).
// Solo lee: dice lo que el tablero de cada proyecto con una consola abierta ya no dice bien, y no corrige nada.

var sincronizarJSON bool

var sincronizarCmd = &cobra.Command{
	Use:   "sincronizar",
	Short: "Revisa, sin escribir nada, el tablero y los flujos de cada proyecto con una consola abierta en esta máquina",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		reg, err := registro.Leer()
		if err != nil {
			fmt.Println("[!] No se pudo leer el registro de proyectos: qdd doctor --fix lo repara.")
			os.Exit(1)
		}
		now := time.Now()
		proyectos := remoto.Sincronizacion(remoto.Armar(Version, reg, now, 0).Proyectos, now)
		if sincronizarJSON {
			out, _ := json.MarshalIndent(map[string]any{"proyectos": proyectos}, "", "  ")
			fmt.Println(string(out))
			return
		}
		imprimirSincronizacion(os.Stdout, proyectos)
	},
}

func init() {
	sincronizarCmd.Flags().BoolVar(&sincronizarJSON, "json", false, "Salida en JSON, como el evento sincronizado")
	rootCmd.AddCommand(sincronizarCmd)
}

func imprimirSincronizacion(out io.Writer, proyectos []remoto.ProyectoSincronizado) {
	for _, p := range proyectos {
		if !p.Consola {
			fmt.Fprintf(out, "– %s: sin consola abierta, no se revisó\n", p.Nombre)
			continue
		}
		if len(p.Inconsistencias) == 0 {
			fmt.Fprintf(out, "✅ %s\n", p.Nombre)
			continue
		}
		fmt.Fprintf(out, "⚠ %s: %d\n", p.Nombre, len(p.Inconsistencias))
		for _, i := range p.Inconsistencias {
			fmt.Fprintf(out, "    %s\n", lineaInconsistencia(i))
		}
	}
	fmt.Fprintln(out, "[i] Solo se revisó: no se corrigió nada.")
}

func lineaInconsistencia(i remoto.Inconsistencia) string {
	donde := i.Tarea
	if i.Etapa != "" {
		donde += " · " + i.Etapa
	}
	if donde == "" {
		return i.Tipo + ": " + i.Detalle
	}
	if i.Detalle == "" {
		return i.Tipo + ": " + donde
	}
	return i.Tipo + ": " + donde + ": " + i.Detalle
}
