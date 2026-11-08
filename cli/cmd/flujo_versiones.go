package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Versiones y plantillas de un flujo (RFC 0025, decisiones de la persona 2026-11-08: «revisiones + versiones con
// nombre» y «un flujo, tipos por plantilla»).

// numeroDeVersion lee «v2» o «2».
func numeroDeVersion(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(s)), "v"))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("versión %q: usa v1, v2… ('qdd flujo versiones <flujo>' dice cuáles hay)", s)
	}
	return n, nil
}

// versionPedida es la versión pedida, o el flujo de hoy con «actual».
func versionPedida(root, id, cual string) (flujo.Flujo, error) {
	if cual == "actual" {
		return flujo.Cargar(root, id)
	}
	n, err := numeroDeVersion(cual)
	if err != nil {
		return flujo.Flujo{}, err
	}
	return flujo.CargarVersion(root, id, n)
}

func jsonDeFlujo(w io.Writer, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		salirCon("%v", err)
	}
	fmt.Fprintln(w, string(b))
}

var flujoVersionCmd = &cobra.Command{
	Use:   "version <flujo> <nombre>",
	Short: "Guarda la foto actual del flujo como una versión con nombre («acordada con Ana»)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		v, err := flujo.CerrarVersion(estadoRaiz(), args[0], args[1], time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[+] %s v%d «%s» sobre la revisión r%d.\n", args[0], v.N, v.Nombre, v.Revision)
	},
}

var flujoVersionesCmd = &cobra.Command{
	Use:   "versiones <flujo>",
	Short: "Lista las versiones con nombre del flujo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := flujo.Cargar(estadoRaiz(), args[0])
		if err != nil {
			salirCon("%v", err)
		}
		salida := cmd.OutOrStdout()
		if flujoJSON {
			jsonDeFlujo(salida, append([]flujo.Version{}, f.Versiones...))
			return
		}
		fmt.Fprintf(salida, "%s está en la revisión r%d.\n", f.ID, f.Revision)
		for _, v := range f.Versiones {
			fmt.Fprintf(salida, "  v%d  r%-4d %s  %s\n", v.N, v.Revision, v.En, v.Nombre)
		}
	},
}

var flujoCompararCmd = &cobra.Command{
	Use:   "comparar <flujo> <v1> <v2|actual>",
	Short: "Dice qué nodos se agregaron, se quitaron o cambiaron entre dos versiones",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		a, err := versionPedida(root, args[0], args[1])
		if err != nil {
			salirCon("%v", err)
		}
		b, err := versionPedida(root, args[0], args[2])
		if err != nil {
			salirCon("%v", err)
		}
		d := flujo.Comparar(a, b)
		salida := cmd.OutOrStdout()
		if flujoJSON {
			jsonDeFlujo(salida, d)
			return
		}
		if len(d.Agregados)+len(d.Quitados)+len(d.Cambiados) == 0 {
			fmt.Fprintf(salida, "%s y %s tienen los mismos nodos.\n", args[1], args[2])
			return
		}
		for _, n := range d.Agregados {
			fmt.Fprintf(salida, "+ %s «%s»\n", n.ID, n.Nombre)
		}
		for _, n := range d.Quitados {
			fmt.Fprintf(salida, "- %s «%s»\n", n.ID, n.Nombre)
		}
		for _, c := range d.Cambiados {
			fmt.Fprintf(salida, "~ %s «%s»: %s\n", c.ID, c.Nombre, strings.Join(c.Campos, ", "))
		}
	},
}

var flujoVolverCmd = &cobra.Command{
	Use:   "volver <flujo> <v1>",
	Short: "Deja el flujo como en esa versión, en una revisión nueva (la historia no se reescribe)",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		n, err := numeroDeVersion(args[1])
		if err != nil {
			salirCon("%v", err)
		}
		f, err := flujo.Volver(estadoRaiz(), args[0], n, flujoDe, time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[+] %s volvió a v%d en la revisión r%d.\n", f.ID, n, f.Revision)
	},
}

var flujoHistorialCmd = &cobra.Command{
	Use:   "historial <flujo>",
	Short: "Quién cambió qué en el flujo, revisión por revisión",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		h := flujo.Historial(estadoRaiz(), args[0])
		salida := cmd.OutOrStdout()
		if flujoJSON {
			jsonDeFlujo(salida, append([]flujo.Revision{}, h...))
			return
		}
		if len(h) == 0 {
			fmt.Fprintf(salida, "%s no tiene revisiones anotadas.\n", args[0])
			return
		}
		for _, r := range h {
			fmt.Fprintf(salida, "r%-4d %s  %-8s %s\n", r.Rev, r.En, r.Autor, r.Resumen)
		}
	},
}

var flujoPlantillasCmd = &cobra.Command{
	Use:   "plantillas",
	Short: "Las plantillas de flujo: sus tipos de nodo y el nombre de sus niveles N1 · N2 · N3",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		salida := cmd.OutOrStdout()
		if flujoJSON {
			jsonDeFlujo(salida, flujo.Plantillas)
			return
		}
		for _, p := range flujo.Plantillas {
			fmt.Fprintf(salida, "%s (%s) · N1 %s · N2 %s · N3 %s\n", p.ID, p.Nombre, p.Niveles[0], p.Niveles[1], p.Niveles[2])
			for _, t := range p.Tipos {
				fmt.Fprintf(salida, "  %s %-15s %s\n", t.Icono, t.ID, t.Explica)
			}
		}
	},
}

func init() {
	for _, c := range []*cobra.Command{flujoVersionesCmd, flujoCompararCmd, flujoHistorialCmd, flujoPlantillasCmd} {
		c.Flags().BoolVar(&flujoJSON, "json", false, "Salida JSON")
	}
	flujoVolverCmd.Flags().StringVar(&flujoDe, "de", flujo.DeIA, "quién vuelve: persona deja la edición por enviar a la IA")
	flujoCmd.AddCommand(flujoVersionCmd, flujoVersionesCmd, flujoCompararCmd, flujoVolverCmd, flujoHistorialCmd, flujoPlantillasCmd)
}
