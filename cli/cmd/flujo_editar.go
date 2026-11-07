package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'

// Cada edición del flujo es un comando: el panel y la IA pasan por las mismas validaciones. --de persona lo pasa el
// panel, y es lo que deja la edición por enviar: lo que hace la IA no la llama a ella misma.

// flujoEnvio es lo que el panel le entrega a la IA, y cuántas líneas lleva por si no se pudo entregar.
type flujoEnvio struct {
	Mensaje string `json:"mensaje"`
	N       int    `json:"n"`
}

type textoOpcional struct {
	valor string
	flag  string
}

func (t *textoOpcional) puntero(cmd *cobra.Command) *string {
	if !cmd.Flags().Changed(t.flag) {
		return nil
	}
	return &t.valor
}

var (
	flujoDe        string
	flujoNombre    = textoOpcional{flag: "nombre"}
	flujoDescribe  = textoOpcional{flag: "descripcion"}
	flujoTipo      = textoOpcional{flag: "tipo"}
	flujoRama      bool
	flujoDespuesDe string
	flujoCon       string
	flujoN         int
)

// edicionDeFlujo arma un subcomando que aplica el cambio y dice en una línea qué hizo.
func edicionDeFlujo(uso, resumen string, cambio func(cmd *cobra.Command, args []string) flujo.Cambio) *cobra.Command {
	return &cobra.Command{
		Use:   uso,
		Short: resumen,
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			_, linea, err := flujo.Editar(estadoRaiz(), args[0], flujoDe, time.Now(), cambio(cmd, args))
			if err != nil {
				salirCon("%v", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), linea)
		},
	}
}

var flujoAgregarCmd = edicionDeFlujo("agregar <flujo> <nodo>", "Agrega un nodo después de otro; con --rama, como una salida más", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.Agregar(args[1], flujoNombre.valor, flujoTipo.valor, flujoDescribe.valor, flujoRama)
})

var flujoModificarCmd = edicionDeFlujo("modificar <flujo> <nodo>", "Cambia el nombre, la descripción o el tipo de un nodo", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.Modificar(args[1], flujoNombre.puntero(cmd), flujoDescribe.puntero(cmd), flujoTipo.puntero(cmd))
})

var flujoQuitarCmd = edicionDeFlujo("quitar <flujo> <nodo>", "Quita un nodo y une lo de antes con lo de después", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.Quitar(args[1])
})

var flujoMoverCmd = edicionDeFlujo("mover <flujo> <nodo>", "Cuelga un nodo después de otro", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.Mover(args[1], flujoDespuesDe)
})

var flujoConectarCmd = edicionDeFlujo("conectar <flujo> <nodo>", "Agrega una flecha hacia otro nodo", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.Conectar(args[1], flujoCon)
})

var flujoDesconectarCmd = edicionDeFlujo("desconectar <flujo> <nodo>", "Quita una flecha hacia otro nodo", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.Desconectar(args[1], flujoCon)
})

var flujoComentarCmd = edicionDeFlujo("comentar <flujo> <nodo> <texto>", "Comenta un nodo; la persona o la IA lo ven debajo de él", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.Comentar(args[1], textoDeArgs(args), flujoDe, time.Now())
})

var flujoDetalleCmd = edicionDeFlujo("detalle <flujo> <nodo> <texto>", "Explica un nodo: es la respuesta a un 🔎 de la persona", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.Detalle(args[1], textoDeArgs(args))
})

func textoDeArgs(args []string) string {
	if len(args) < 3 {
		salirCon("Falta el texto: el último argumento, entre comillas.")
	}
	return args[2]
}

// preguntaDeArgs: el nodo es opcional; sin él, la pregunta o la respuesta es sobre el flujo completo.
func preguntaDeArgs(args []string) (string, string) {
	if len(args) == 2 {
		return "", args[1]
	}
	return args[1], args[2]
}

var flujoPreguntarCmd = &cobra.Command{
	Use:   "preguntar <flujo> [<nodo>] <texto>",
	Short: "La persona pregunta por un nodo o por el flujo completo; queda por enviar a la IA",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		nodo, texto := preguntaDeArgs(args)
		_, linea, err := flujo.Editar(estadoRaiz(), args[0], flujo.DePersona, time.Now(), flujo.Preguntar(nodo, texto, time.Now()))
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), linea)
	},
}

var flujoResponderCmd = &cobra.Command{
	Use:   "responder <flujo> [<nodo>] <texto>",
	Short: "La IA responde un ❓ de la persona, bajo el nodo o en el flujo completo",
	Args:  cobra.RangeArgs(2, 3),
	Run: func(cmd *cobra.Command, args []string) {
		nodo, texto := preguntaDeArgs(args)
		_, linea, err := flujo.Editar(estadoRaiz(), args[0], flujo.DeIA, time.Now(), flujo.Responder(nodo, texto, time.Now()))
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), linea)
	},
}

var flujoPedirDetalleCmd = &cobra.Command{
	Use:   "pedir-detalle <flujo> <nodo>",
	Short: "Deja por enviar la pregunta de la persona por un nodo",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if err := flujo.PedirDetalle(estadoRaiz(), args[0], args[1], time.Now()); err != nil {
			salirCon("%v", err)
		}
	},
}

var flujoDeshacerCmd = &cobra.Command{
	Use:   "deshacer <flujo>",
	Short: "Deshace la última edición de la persona, si la IA todavía no la recibe",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := flujo.Deshacer(estadoRaiz(), args[0]); err != nil {
			salirCon("%v", err)
		}
	},
}

var flujoEnviarCmd = &cobra.Command{
	Use:   "enviar <flujo>",
	Short: "Reclama lo pendiente y escribe el mensaje para la IA; lo reclamado no sale dos veces",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		mensaje, n := flujo.Enviar(estadoRaiz(), args[0])
		if !flujoJSON {
			fmt.Fprintln(cmd.OutOrStdout(), mensaje)
			return
		}
		b, err := json.Marshal(flujoEnvio{Mensaje: mensaje, N: n})
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
	},
}

var flujoDevolverCmd = &cobra.Command{
	Use:   "devolver <flujo>",
	Short: "Deja como pendientes las últimas --n líneas: el panel no pudo entregarlas",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := flujo.Devolver(estadoRaiz(), args[0], flujoN); err != nil {
			salirCon("%v", err)
		}
	},
}

var flujoNuevoCmd = &cobra.Command{
	Use:   "nuevo <nombre>",
	Short: "Crea un flujo con un solo nodo, para seguir editándolo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		f, err := flujo.Nuevo(estadoRaiz(), args[0])
		if err != nil {
			salirCon("%v", err)
		}
		if !flujoJSON {
			fmt.Fprintf(cmd.OutOrStdout(), "[+] Flujo '%s' creado.\n", f.ID)
			return
		}
		b, err := json.Marshal(struct {
			ID     string `json:"id"`
			Nombre string `json:"nombre"`
		}{f.ID, f.Nombre})
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(b))
	},
}

func init() {
	ediciones := []*cobra.Command{flujoAgregarCmd, flujoModificarCmd, flujoQuitarCmd, flujoMoverCmd, flujoConectarCmd, flujoDesconectarCmd, flujoComentarCmd, flujoDetalleCmd}
	for _, c := range ediciones {
		c.Flags().StringVar(&flujoDe, "de", flujo.DeIA, "Quién la hace: ia o persona, que la deja por enviar")
	}
	for _, c := range []*cobra.Command{flujoAgregarCmd, flujoModificarCmd} {
		c.Flags().StringVar(&flujoNombre.valor, flujoNombre.flag, "", "Nombre del nodo")
		c.Flags().StringVar(&flujoDescribe.valor, flujoDescribe.flag, "", "Descripción del nodo")
		c.Flags().StringVar(&flujoTipo.valor, flujoTipo.flag, "", "paso o resultado")
	}
	flujoAgregarCmd.Flags().BoolVar(&flujoRama, "rama", false, "Como una salida más, sin tocar lo que ya seguía")
	flujoMoverCmd.Flags().StringVar(&flujoDespuesDe, "despues-de", "", "Nodo después del que va")
	for _, c := range []*cobra.Command{flujoConectarCmd, flujoDesconectarCmd} {
		c.Flags().StringVar(&flujoCon, "con", "", "El otro nodo")
	}
	flujoEnviarCmd.Flags().BoolVar(&flujoJSON, "json", false, "Salida JSON, para el panel")
	flujoNuevoCmd.Flags().BoolVar(&flujoJSON, "json", false, "Salida JSON, para el panel")
	flujoDevolverCmd.Flags().IntVar(&flujoN, "n", 0, "Cuántas líneas")
	flujoCmd.AddCommand(append(ediciones, flujoPreguntarCmd, flujoResponderCmd, flujoPedirDetalleCmd, flujoDeshacerCmd, flujoEnviarCmd, flujoDevolverCmd, flujoNuevoCmd)...)
}
