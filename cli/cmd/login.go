package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Issue #102: emparejar con QDD Móvil es el login. Es qdd remoto activar con otro nombre, para quien solo tiene la app.

var loginCmd = &cobra.Command{
	Use:   "login [--relay <url> --clave <publicable>]",
	Short: "Empareja este PC con QDD Móvil: muestra el QR y deja el servicio que lo publica",
	Long: `Empareja este PC con QDD Móvil: muestra un QR, lo escaneas con la app y el PC queda publicando.

Sin banderas consulta el directorio firmado del relay; si no responde, sigue con el guardado. --relay y --clave son
para quien opera su propio relay, y ese relay no se consulta. Es lo mismo que qdd remoto activar.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoActivar(cmd.Context(), os.Stdout, os.Stderr))
	},
}

func init() {
	loginCmd.Flags().StringVar(&remotoRelayURL, "relay", "", "URL de tu propio relay (https://<ref>.supabase.co)")
	loginCmd.Flags().StringVar(&remotoClave, "clave", "", "clave publicable de tu propio relay (sb_publishable_… o el JWT anon); una secreta se rechaza")
	loginCmd.Flags().BoolVar(&remotoSinServ, "sin-servicio", false, "no deja el servicio: córrelo tú con 'qdd remoto correr'")
	rootCmd.AddCommand(loginCmd)
}
