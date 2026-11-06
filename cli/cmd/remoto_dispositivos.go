package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.3): los teléfonos del dueño del PC. Revocar uno lo saca en el relay (su próxima acción se rechaza) y en
// este PC (revocados.json), con efecto inmediato aunque tenga la sesión abierta.

var remotoDispositivosCmd = &cobra.Command{
	Use:   "dispositivos",
	Short: "Lista los teléfonos de tu cuenta que ven este PC en QDD Móvil",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoDispositivos(cmd.Context(), os.Stdout, os.Stderr))
	},
}

var remotoRevocarCmd = &cobra.Command{
	Use:   "revocar <id>",
	Short: "Saca un teléfono al instante: su próxima acción se rechaza",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(remotoRevocar(cmd.Context(), args[0], os.Stdout, os.Stderr))
	},
}

func init() {
	remotoCmd.AddCommand(remotoDispositivosCmd, remotoRevocarCmd)
}

func relayYCredenciales() (*remoto.Relay, remoto.Credenciales, error) {
	c, err := remoto.LeerRelay()
	if err != nil {
		return nil, remoto.Credenciales{}, fmt.Errorf("falta el relay: usa 'qdd login'")
	}
	cred, err := remoto.LeerCredenciales()
	if err != nil {
		return nil, cred, err
	}
	return remoto.NuevoRelay(c), cred, nil
}

func remotoDispositivos(ctx context.Context, salida, errores io.Writer) int {
	r, cred, err := relayYCredenciales()
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	ds, err := r.Dispositivos(contexto(ctx), cred.Topico, cred.Firma)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	if len(ds) == 0 {
		fmt.Fprintln(salida, "Ningún teléfono registrado todavía: inicia sesión en QDD Móvil.")
		return 0
	}
	locales := remoto.LeerRevocados()
	for _, d := range ds {
		marca := "✓"
		if d.Revocado || locales[d.ID] {
			marca = "✕ revocado"
		}
		fmt.Fprintf(salida, "%s  %s (%s) · visto %s · %s\n", d.ID, d.Nombre, d.Plataforma, d.Visto.Local().Format("2006-01-02 15:04"), marca)
	}
	return 0
}

func remotoRevocar(ctx context.Context, id string, salida, errores io.Writer) int {
	if err := remoto.Revocar(id); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintf(salida, "[+] Este PC ya no acepta acciones de %s.\n", id)
	r, cred, err := relayYCredenciales()
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	ok, err := r.Revocar(contexto(ctx), cred.Topico, cred.Firma, id)
	if err != nil {
		fmt.Fprintf(errores, "[!] no se pudo revocar en el relay (%v): revócalo también desde QDD Móvil.\n", err)
		return 1
	}
	if !ok {
		fmt.Fprintf(errores, "[!] el relay no conoce ese teléfono en tu cuenta: revisa el id con 'qdd remoto dispositivos'.\n")
		return 1
	}
	fmt.Fprintln(salida, "[+] Revocado en el relay: su próxima acción se rechaza aunque tenga la sesión abierta.")
	return 0
}
