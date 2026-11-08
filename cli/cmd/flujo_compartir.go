package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/incidentes"
	"github.com/qdd-framework/qdd/pkg/reportes"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd flujo compartir (RFC 0025, fase 1): el archivo HTML de solo lectura para conversar un flujo con alguien sin QDD.
// El texto pasa por la limpieza de qdd reportar (valores del baúl, credenciales, la ruta del proyecto, la carpeta y el
// nombre del usuario). La invitación por el relay, para que proponga en vivo, llega en la fase 3.

var (
	flujoPara   string
	flujoSalida string
)

var flujoCompartirCmd = &cobra.Command{
	Use:   "compartir <flujo> [--para <nombre>] [--salida <archivo.html>]",
	Short: "Escribe un archivo HTML del flujo para conversarlo con alguien sin QDD (N1 · N2 · N3, leyenda, solo lectura)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		f, err := flujo.Cargar(root, args[0])
		if err != nil {
			salirCon("%v", err)
		}
		contexto, valores := incidentes.ContextoDe(root), baul.ValoresDelLlavero()
		limpiar := func(s string) string { return reportes.Limpiar(s, contexto, valores) }
		html, err := flujo.ArchivoHTML(flujo.ArmarFoto(f, limpiar, time.Now()))
		if err != nil {
			salirCon("%v", err)
		}
		destino := archivoDeCompartir(root, f.ID)
		if err := os.MkdirAll(filepath.Dir(destino), 0o755); err != nil {
			salirCon("%v", err)
		}
		if err := os.WriteFile(destino, html, 0o644); err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[+] %s\n    Archivo de solo lectura (r%d): se abre en cualquier navegador; adjúntalo en un correo o un chat.\n", destino, f.Revision)
	},
}

// archivoDeCompartir es --salida o ./tmp/flujo-<id>[-<para>].html en el proyecto.
func archivoDeCompartir(root, id string) string {
	if flujoSalida != "" {
		return flujoSalida
	}
	nombre := "flujo-" + id
	if para := flujo.IDGrupo(flujoPara); flujoPara != "" && para != "grupo-" {
		nombre += "-" + para[len("grupo-"):]
	}
	return filepath.Join(root, "tmp", nombre+".html")
}

func init() {
	flujoCompartirCmd.Flags().StringVar(&flujoPara, "para", "", "con quién se conversa (va en el nombre del archivo)")
	flujoCompartirCmd.Flags().StringVar(&flujoSalida, "salida", "", "dónde escribir el archivo (por defecto ./tmp/flujo-<id>-<para>.html)")
	flujoCmd.AddCommand(flujoCompartirCmd)
}
