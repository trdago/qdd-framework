package cmd

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Rule: Prohibido usar 'else'
// Blueprint estado-por-tareas: la ayuda de 'qdd estado' y la tool qdd_estado dicen «tarea»; «frente» solo queda como
// el nombre del blueprint anterior (estado-por-frente).
func TestEstado_LaAyudaDiceTarea(t *testing.T) {
	var textos []string
	var juntar func(c *cobra.Command)
	juntar = func(c *cobra.Command) {
		textos = append(textos, c.Use, c.Short, c.Long)
		c.Flags().VisitAll(func(f *pflag.Flag) { textos = append(textos, f.Usage) })
		for _, h := range c.Commands() {
			juntar(h)
		}
	}
	juntar(estadoCmd)
	textos = append(textos, descripcionEstadoMCP)
	for _, s := range textos {
		if sinFrente(s) {
			continue
		}
		t.Fatalf("dice «frente»: %q", s)
	}
}

func sinFrente(s string) bool {
	return !strings.Contains(strings.ToLower(strings.ReplaceAll(s, "estado-por-frente", "")), "frente")
}
