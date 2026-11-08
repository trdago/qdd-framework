package cmd

import (
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd flujo grupo (pedido de la persona, 2026-11-08): un grupo de nodos con nombre y clase (fase, cola, ambiente…) que se
// dibuja como un marco. Los nodos entran con 'qdd flujo modificar <flujo> <nodo> --grupo "<nombre>"'.

var (
	flujoClase       string
	flujoGrupoDetall string
)

var flujoGrupoCmd = edicionDeFlujo("grupo <flujo> <nombre>", "Le da clase (fase, cola, ambiente…) y descripción a un grupo de nodos", func(cmd *cobra.Command, args []string) flujo.Cambio {
	return flujo.DefinirGrupo(args[1], flujoClase, flujoGrupoDetall)
})

func init() {
	flujoGrupoCmd.Flags().StringVar(&flujoClase, "clase", "", "qué agrupa: fase, cola, ambiente… ('qdd flujo plantillas' sugiere las de cada plantilla)")
	flujoGrupoCmd.Flags().StringVar(&flujoGrupoDetall, "descripcion", "", "para qué está el grupo")
	flujoGrupoCmd.Flags().StringVar(&flujoDe, "de", flujo.DeIA, "Quién la hace: ia o persona, que la deja por enviar")
	flujoCmd.AddCommand(flujoGrupoCmd)
}
