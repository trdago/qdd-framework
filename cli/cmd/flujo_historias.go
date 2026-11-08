package cmd

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/qdd-framework/qdd/pkg/listas"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// Historias de usuario de un flujo y visibilidad de sus notas (RFC 0025). Una historia aprobada se vuelve tarea del
// tablero con 'qdd flujo historia tarea', y lo que sigue es el ciclo de siempre. Una nota es interna hasta que la
// persona la comparte: solo entonces la ve el cliente en el archivo.

const maxNombreDeTareaDeHistoria = 80

var (
	historiaComo, historiaQuiero, historiaPara string
	historiaCriterios, historiaNodos           []string
	itemInterna                                bool
)

var flujoHistoriaCmd = &cobra.Command{
	Use:   "historia <flujo>",
	Short: "Escribe una historia de usuario: --como, --quiero, --para, un --criterio y un --nodo por cada uno",
	Long: `  qdd flujo historia venta --como vendedor --quiero "validar el RUT antes de crear el cliente" \
      --para "no duplicar clientes" --criterio "un RUT inválido no deja seguir" --nodo validar-rut
  qdd flujo historia tarea venta h1     la vuelve tarea del tablero
  qdd flujo historia quitar venta h1`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		h := flujo.Historia{Como: historiaComo, Quiero: historiaQuiero, Para: historiaPara, Criterios: historiaCriterios, Nodos: historiaNodos}
		editarFlujo(cmd, args[0], flujoDe, flujo.AgregarHistoria(h, flujoDe))
	},
}

var flujoHistoriaTareaCmd = &cobra.Command{
	Use:   "tarea <flujo> <historia>",
	Short: "Crea la tarea del tablero que implementa la historia y la deja anotada en ella",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		f, err := flujo.Cargar(root, args[0])
		if err != nil {
			salirCon("%v", err)
		}
		h, err := flujo.HistoriaDe(f, args[1])
		if err != nil {
			salirCon("%v", err)
		}
		if h.Tarea != "" {
			salirCon("la historia %s ya tiene la tarea %s", h.ID, h.Tarea)
		}
		id, err := estado.Nueva(root, estado.CambioNueva{Nombre: nombreDeTareaDeHistoria(h)}, time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		if _, _, err := flujo.Editar(root, f.ID, flujo.DeIA, time.Now(), flujo.TareaDeHistoria(h.ID, id)); err != nil {
			salirCon("la tarea %s quedó creada, pero no se anotó en la historia: %v", id, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[+] tarea %s creada para la historia %s de %s: %s\n", id, h.ID, f.ID, h.Frase())
		for _, c := range h.Criterios {
			fmt.Fprintf(cmd.OutOrStdout(), "  · criterio: %s\n", c)
		}
	},
}

var flujoHistoriaQuitarCmd = &cobra.Command{
	Use:   "quitar <flujo> <historia>",
	Short: "Quita una historia de usuario; su tarea, si tiene, sigue en el tablero",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		editarFlujo(cmd, args[0], flujoDe, flujo.QuitarHistoria(args[1]))
	},
}

var flujoHistoriasCmd = &cobra.Command{
	Use:   "historias <flujo>",
	Short: "Lista las historias de usuario del flujo, con su origen, sus criterios y su tarea",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := flujo.Cargar(estadoRaiz(), args[0])
		if err != nil {
			salirCon("%v", err)
		}
		if flujoJSON {
			lista := f.Historias
			if lista == nil {
				lista = []flujo.Historia{}
			}
			b, _ := listas.Marshal(lista)
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return
		}
		if len(f.Historias) == 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "El flujo %s no tiene historias de usuario: escribe una con 'qdd flujo historia %s --quiero …'\n", f.ID, f.ID)
			return
		}
		for _, h := range f.Historias {
			fmt.Fprintln(cmd.OutOrStdout(), lineaDeHistoria(h))
			for _, c := range h.Criterios {
				fmt.Fprintf(cmd.OutOrStdout(), "    · %s\n", c)
			}
		}
	},
}

var flujoItemCompartirCmd = &cobra.Command{
	Use:   "compartir <flujo> <nodo> <item>",
	Short: "Deja que el cliente vea la nota en el archivo; con --interna, vuelve a ser solo tuya",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		editarFlujo(cmd, args[0], flujoDe, flujo.CompartirItem(args[1], args[2], !itemInterna))
	},
}

func lineaDeHistoria(h flujo.Historia) string {
	partes := []string{h.ID + "  " + h.Frase()}
	if len(h.Nodos) > 0 {
		partes = append(partes, "en "+strings.Join(h.Nodos, ", "))
	}
	if h.Origen != "" {
		partes = append(partes, "de "+h.Origen)
	}
	if h.Tarea != "" {
		partes = append(partes, "tarea "+h.Tarea)
	}
	return strings.Join(partes, " · ")
}

// nombreDeTareaDeHistoria es lo que quiere la historia, con mayúscula y acotado: de ahí saca qdd el id de la tarea.
func nombreDeTareaDeHistoria(h flujo.Historia) string {
	nombre := []rune(h.Quiero)
	if len(nombre) > maxNombreDeTareaDeHistoria {
		nombre = append(nombre[:maxNombreDeTareaDeHistoria-1], '…')
	}
	primera, largo := utf8.DecodeRuneInString(string(nombre))
	return string(unicode.ToUpper(primera)) + string(nombre)[largo:]
}

func init() {
	flujoHistoriaCmd.Flags().StringVar(&historiaComo, "como", "", "Quién la necesita: vendedor, cliente, administrador…")
	flujoHistoriaCmd.Flags().StringVar(&historiaQuiero, "quiero", "", "Qué quiere hacer (obligatorio)")
	flujoHistoriaCmd.Flags().StringVar(&historiaPara, "para", "", "Para qué lo quiere")
	flujoHistoriaCmd.Flags().StringArrayVar(&historiaCriterios, "criterio", nil, "Un criterio de aceptación; repítelo por cada uno")
	flujoHistoriaCmd.Flags().StringArrayVar(&historiaNodos, "nodo", nil, "Un nodo que toca; repítelo por cada uno")
	for _, c := range []*cobra.Command{flujoHistoriaCmd, flujoHistoriaQuitarCmd, flujoItemCompartirCmd} {
		c.Flags().StringVar(&flujoDe, "de", flujo.DeIA, "Quién la hace: ia o persona, que la deja por enviar")
	}
	flujoHistoriasCmd.Flags().BoolVar(&flujoJSON, "json", false, "Salida JSON")
	flujoItemCompartirCmd.Flags().BoolVar(&itemInterna, "interna", false, "La vuelve interna: el cliente deja de verla")
	flujoHistoriaCmd.AddCommand(flujoHistoriaTareaCmd, flujoHistoriaQuitarCmd)
	flujoItemCmd.AddCommand(flujoItemCompartirCmd)
	flujoCmd.AddCommand(flujoHistoriaCmd, flujoHistoriasCmd)
}
