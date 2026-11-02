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
// RFC 0013: qdd remoto es el lado PC de QDD Móvil. Sus hooks reemplazan a los de la raíz (como qdd consola): no exige
// un proyecto, no escribe .qdd/working y su tráfico no cuenta como uso en qdd uso (inv. 10, RFC 0010).

var remotoJSON bool

var remotoCmd = &cobra.Command{
	Use:   "remoto",
	Short: "Consola remota: el panel de todos los proyectos de este PC en QDD Móvil (estado | activar | emparejar | …)",
	Long: `El lado PC de QDD Móvil (RFC 0013). Un solo proceso por máquina publica el panel de todos sus proyectos por un
canal de Supabase Realtime y ejecuta solo las acciones que el panel ofrece, firmadas por el relay y confirmadas en el
teléfono. El PC encendido es la fuente de verdad: en la nube no se guarda ni el estado ni las acciones.

  qdd remoto estado [--json]   lo que este PC publicaría, sin red: un panel por proyecto del registro
                               (<QDD_HOME>/proyectos.json, que anotan qdd init y cada mensaje con el panel)`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
}

var remotoEstadoCmd = &cobra.Command{
	Use:   "estado",
	Short: "Lo que este PC publicaría en QDD Móvil, sin red (un panel por proyecto del registro)",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoEstado(os.Stdout, os.Stderr, remotoJSON, time.Now()))
	},
}

func init() {
	remotoEstadoCmd.Flags().BoolVar(&remotoJSON, "json", false, "el JSON del evento estado (lo que dibuja la app)")
	remotoCmd.AddCommand(remotoEstadoCmd)
	rootCmd.AddCommand(remotoCmd)
}

func remotoEstado(salida, errores io.Writer, comoJSON bool, now time.Time) int {
	reg, err := registro.Leer()
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	e := remoto.Armar(Version, reg, now, 0)
	if comoJSON {
		b, err := json.MarshalIndent(e, "", "  ")
		if err != nil {
			fmt.Fprintf(errores, "[!] %v\n", err)
			return 1
		}
		fmt.Fprintln(salida, string(b))
		return 0
	}
	if len(e.Proyectos) == 0 {
		fmt.Fprintln(salida, "Ningún proyecto en el registro todavía: se anotan con qdd init y con cada mensaje en una sesión con el panel.")
		return 0
	}
	for _, p := range e.Proyectos {
		fmt.Fprintln(salida, remoto.Linea(p))
	}
	return 0
}

// anotarProyecto deja el proyecto en el registro de la máquina (qdd init); nunca corta lo que lo llama.
func anotarProyecto(root string) {
	_ = registro.Anotar(root, time.Now())
}
