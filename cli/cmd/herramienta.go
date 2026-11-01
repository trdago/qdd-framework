package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/herramientas"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'

var (
	herramientaJSON bool
	herramientaSi   bool
)

var herramientaCmd = &cobra.Command{
	Use:     "herramienta",
	Aliases: []string{"herramientas", "tool"},
	Short:   "Los ejecutables del proyecto como pestañas del panel (RFC 0011)",
	Long: `Cada herramienta es un ejecutable del proyecto con sus acciones (botones) y los campos que piden.
La IA arma la definición una vez; QDD la valida, corre su prueba de humo y la guarda en .qdd/herramientas/<id>.yaml.
Desde ahí el panel la dibuja y la corre sin IA. Los secretos nunca van en la definición: solo el nombre de su variable.

  qdd herramienta candidatos [--json]          los ejecutables del proyecto (bin/, scripts/, cmd/*, npm, make…)
  qdd herramienta ayuda <comando…>             su --help (o -h, help), sin entrada y con tiempo límite
  qdd herramienta guardar <archivo.yaml>       valida, prueba y guarda la definición
  qdd herramienta listar [--json]              las guardadas y lo que les falta en esta máquina
  qdd herramienta correr <id> <acción> [campo=valor…] [--si]
  qdd herramienta quitar <id> --si`,
}

var herramientaCandidatosCmd = &cobra.Command{
	Use:   "candidatos",
	Short: "Los ejecutables del proyecto que se pueden agregar",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cs := herramientas.Candidatos(modulesRoot())
		if herramientaJSON {
			imprimirJSON(cs)
			return
		}
		if len(cs) == 0 {
			fmt.Println("No encontré ejecutables (bin/, scripts/, cmd/*, target/release, package.json, Makefile). Agrega uno con su ruta.")
			return
		}
		for _, c := range cs {
			marca := ""
			if c.Agregada {
				marca = " (ya agregada)"
			}
			fmt.Printf("- %s · %s%s\n", strings.Join(c.Comando, " "), c.Origen, marca)
		}
	},
}

var herramientaAyudaCmd = &cobra.Command{
	Use:   "ayuda <comando> [args…]",
	Short: "Lee la ayuda del ejecutable para armar su definición",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		texto, err := herramientas.Ayuda(context.Background(), modulesRoot(), args)
		fmt.Print(texto)
		if err != nil {
			salirCon("%v", err)
		}
	},
}

var herramientaGuardarCmd = &cobra.Command{
	Use:   "guardar <archivo.yaml>",
	Short: "Valida la definición, corre su prueba de humo y la guarda",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		data, err := os.ReadFile(args[0])
		if err != nil {
			salirCon("%v", err)
		}
		rel, problemas := herramientas.Guardar(modulesRoot(), data)
		if len(problemas) > 0 {
			fmt.Fprintln(os.Stderr, "[!] No se guardó:")
			for _, p := range problemas {
				fmt.Fprintf(os.Stderr, "  - %s\n", p)
			}
			os.Exit(1)
		}
		fmt.Printf("[+] Guardada en %s, con su prueba de humo en 0. Ya es una pestaña del panel; versiónala con el proyecto.\n", filepath.ToSlash(rel))
	},
}

// vistaDeHerramienta es lo que el panel dibuja: la definición, su archivo y lo que le falta en esta máquina.
type vistaDeHerramienta struct {
	herramientas.Herramienta
	Archivo string `json:"archivo"`
	Aviso   string `json:"aviso,omitempty"`
}

func listadoDeHerramientas(root string) ([]vistaDeHerramienta, []error) {
	hs, errs := herramientas.Cargar(root)
	var out []vistaDeHerramienta
	for _, h := range hs {
		out = append(out, vistaDeHerramienta{h, filepath.ToSlash(herramientas.Archivo(h.ID)), herramientas.Aviso(root, h)})
	}
	return out, errs
}

var herramientaListarCmd = &cobra.Command{
	Use:   "listar",
	Short: "Las herramientas guardadas y lo que les falta en esta máquina",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		vs, errs := listadoDeHerramientas(modulesRoot())
		if herramientaJSON {
			var problemas []string
			for _, e := range errs {
				problemas = append(problemas, e.Error())
			}
			imprimirJSON(map[string]interface{}{"herramientas": vs, "problemas": problemas})
			return
		}
		for _, v := range vs {
			estado := "✅"
			if v.Aviso != "" {
				estado = "⏳ " + v.Aviso
			}
			fmt.Printf("- %s · %s · %d acción(es) · %s\n", v.ID, v.Nombre, len(v.Acciones), estado)
		}
		for _, e := range errs {
			fmt.Printf("[!] %v\n", e)
		}
		if len(vs)+len(errs) == 0 {
			fmt.Println("No hay herramientas. En el panel: ⚙ Config → + Agregar herramienta.")
		}
	},
}

// valoresDeArgs lee «campo=valor»: todo lo que sigue al primer «=» es el valor, entero.
func valoresDeArgs(args []string) (map[string]string, error) {
	out := map[string]string{}
	for _, a := range args {
		campo, valor, ok := strings.Cut(a, "=")
		if !ok || campo == "" {
			return nil, fmt.Errorf("«%s»: cada valor va como campo=valor", a)
		}
		out[campo] = valor
	}
	return out, nil
}

var herramientaCorrerCmd = &cobra.Command{
	Use:   "correr <id> <acción> [campo=valor…]",
	Short: "Corre una acción sin shell, en la raíz del proyecto (lo que usa el panel)",
	Args:  cobra.MinimumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		root := modulesRoot()
		h, err := herramientas.Buscar(root, args[0])
		if err != nil {
			salirCon("%v", err)
		}
		valores, err := valoresDeArgs(args[2:])
		if err != nil {
			salirCon("%v", err)
		}
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		code, err := herramientas.Correr(ctx, root, h, args[1], valores, herramientas.Opciones{Confirmado: herramientaSi, Salida: os.Stdout, Errores: os.Stderr})
		if err != nil {
			salirCon("%v", err)
		}
		os.Exit(code)
	},
}

var herramientaQuitarCmd = &cobra.Command{
	Use:   "quitar <id> --si",
	Short: "Borra la definición (su pestaña se cierra)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if !herramientaSi {
			salirCon("quitar borra %s: confírmalo con --si", filepath.ToSlash(herramientas.Archivo(args[0])))
		}
		if err := herramientas.Quitar(modulesRoot(), args[0]); err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[+] Quitada: se borró %s.\n", filepath.ToSlash(herramientas.Archivo(args[0])))
	},
}

func init() {
	for _, c := range []*cobra.Command{herramientaCandidatosCmd, herramientaListarCmd} {
		c.Flags().BoolVar(&herramientaJSON, "json", false, "en JSON (lo que lee el panel)")
	}
	herramientaCorrerCmd.Flags().BoolVar(&herramientaSi, "si", false, "la persona confirmó una acción que lo pide")
	herramientaQuitarCmd.Flags().BoolVar(&herramientaSi, "si", false, "la persona confirmó que la quita")
	herramientaCmd.AddCommand(herramientaCandidatosCmd, herramientaAyudaCmd, herramientaGuardarCmd, herramientaListarCmd, herramientaCorrerCmd, herramientaQuitarCmd)
	rootCmd.AddCommand(herramientaCmd)
}
