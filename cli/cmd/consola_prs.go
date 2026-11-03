package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Los PRs abiertos del repositorio a un clic en el panel, siempre opcionales (pkg/consola/prs.go).

// qdd consola prs (oculto): lo lanza el panel cada minuto, aparte; trae los PRs abiertos con gh y los ofrece.
var consolaPRsCmd = &cobra.Command{
	Use:    "prs",
	Short:  "Trae los PRs abiertos del repositorio y los ofrece en el panel (interno)",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		root := consolaRaiz
		if root == "" {
			root = estadoRaiz()
		}
		if err := consola.TraerPRs(root, time.Now()); err != nil {
			fmt.Fprintf(os.Stderr, "[!] %v\n", err)
			os.Exit(1)
		}
	},
}

// qdd consola mergear-pr <N>: el comando de un pedido de PR. Mergea solo con el CI verde; con el CI corriendo sale con
// «todavía no» (75) y el pedido queda armado.
var consolaMergearPRCmd = &cobra.Command{
	Use:   "mergear-pr <número>",
	Short: "Mergea un PR del repositorio si su CI está verde (el pedido «🔀 Mergear #N» del panel)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		n, err := strconv.Atoi(strings.TrimPrefix(args[0], "#"))
		if err != nil || n <= 0 {
			salirCon("'%s' no es un número de PR", args[0])
		}
		os.Exit(consola.MergearPR(estadoRaiz(), n, os.Stdout))
	},
}

// prsAlDia pide los PRs abiertos aparte, a lo más cada minuto, sin demorar al panel. Desde las pruebas (un binario
// .test) no lanza nada.
func prsAlDia(root string) {
	bin, err := os.Executable()
	if err != nil || strings.HasSuffix(bin, ".test") {
		return
	}
	consola.PRsAlDia(root, bin, time.Now())
}

func init() {
	consolaPRsCmd.Flags().StringVar(&consolaRaiz, "raiz", "", "raíz del proyecto")
	consolaCmd.AddCommand(consolaPRsCmd, consolaMergearPRCmd)
}
