package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd remoto sonda: la misma prueba del canal con el teléfono que corre el servicio (REM-20), en el momento y solo para
// mirar: no repara ni anota nada. Lo que se imprime es el tramo y su motivo fijo, nunca la URL, la clave ni el tópico.

var remotoSondaJSON bool

var remotoSondaCmd = &cobra.Command{
	Use:   "sonda",
	Short: "Prueba ahora el canal con QDD Móvil (ida y vuelta si hay un teléfono mirando); no repara nada",
	Long: `Abre una conexión propia al canal de este PC en el relay, como lo haría el teléfono, y dice en qué tramo falla:
relay (sin relay o sin emparejar), conexion (red), canal (el relay no deja entrar), realtime (no responde) o eco (lo que
el PC publica no vuelve). Con un teléfono mirando manda un broadcast «sonda» con solo una marca y espera que vuelva; sin
teléfono no publica nada. Para reparar, qdd doctor --fix.`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoSonda(os.Stdout, remotoSondaJSON))
	},
}

func init() {
	remotoSondaCmd.Flags().BoolVar(&remotoSondaJSON, "json", false, "el resultado en JSON")
	remotoCmd.AddCommand(remotoSondaCmd)
}

// remotoSonda sale con 0 si el canal funciona y con 1 si falla en algún tramo.
func remotoSonda(salida io.Writer, comoJSON bool) int {
	c, _ := remoto.LeerRelay()
	cred, _ := remoto.LeerCredenciales()
	r := remotoSondear(context.Background(), c, cred, remoto.TopeSonda)
	if comoJSON {
		b, _ := json.Marshal(r)
		fmt.Fprintln(salida, string(b))
		return codigoSonda(r)
	}
	if r.OK {
		fmt.Fprintf(salida, "[+] %s\n", textoSondaBuena(r))
		return 0
	}
	fmt.Fprintf(salida, "[!] Falla en el tramo %s: %s (%d ms). qdd doctor --fix lo repara si se puede.\n", r.Tramo, r.Motivo, r.MS)
	return 1
}

func codigoSonda(r remoto.ResultadoSonda) int {
	if r.OK {
		return 0
	}
	return 1
}
