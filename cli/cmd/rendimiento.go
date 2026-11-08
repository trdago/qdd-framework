package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/mark3labs/mcp-go/server"
	"github.com/qdd-framework/qdd/pkg/rendimiento"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// qdd rendimiento (pedido de la persona, 2026-11-07; RFC 0022): la misma prueba en cada equipo y sistema operativo, sin
// gastar tokens. Corre qdd en una caja temporal y dice cuánto tarda, cuánto consume y si deja procesos abiertos.

var (
	rendimientoSegundos int
	rendimientoJSON     bool
)

var rendimientoCmd = &cobra.Command{
	Use:   "rendimiento",
	Short: "Mide qdd en este equipo, sin tokens: latencias, CPU, memoria y procesos que deja abiertos",
	Long: `Corre qdd en una caja temporal (proyecto, HOME y QDD_HOME propios, los mismos en cada equipo), sin Claude ni
una IA: no gasta tokens ni toca tus proyectos. Mide:

  · latencia (p95) de lo que el panel llama a cada rato: la barra de estado, el panel, qdd estado y un flujo
  · CPU, memoria y procesos por hora de 'qdd consola seguir', el proceso largo de cada panel
  · zombies y procesos que siguen vivos al cerrarlo (una falla: la medición los cierra)
  · el contexto fijo que qdd suma a cada sesión (tools/list del MCP)

Cada medida tiene un presupuesto por sistema operativo. El informe queda por versión en la carpeta de qdd del usuario,
y 'qdd doctor' muestra el último. Sale con 1 si dejó procesos (zombies o vivos al cerrar) y con 2 si solo pasó
algún presupuesto: el CI falla con lo primero y avisa con lo segundo.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	Run: func(cmd *cobra.Command, args []string) {
		bin, err := os.Executable()
		if err != nil {
			salirCon("%v", err)
		}
		if !rendimientoJSON {
			fmt.Printf("[i] Midiendo qdd %s en una caja temporal (~%d s, sin tokens)…\n", Version, rendimientoSegundos+15)
		}
		inf, err := rendimiento.Medir(rendimiento.Opciones{Bin: bin, Version: Version, Segundos: rendimientoSegundos, Extra: []rendimiento.Medida{medidaContexto()}})
		if err != nil {
			salirCon("%v", err)
		}
		if err := rendimiento.Guardar(inf); err != nil {
			fmt.Fprintf(os.Stderr, "[!] no se pudo guardar el informe: %v\n", err)
		}
		imprimirRendimiento(os.Stdout, inf, rendimientoJSON)
		os.Exit(codigoRendimiento(inf))
	},
}

func init() {
	rendimientoCmd.Flags().IntVar(&rendimientoSegundos, "segundos", 30, "cuánto corre 'qdd consola seguir'")
	rendimientoCmd.Flags().BoolVar(&rendimientoJSON, "json", false, "el informe en JSON")
	rootCmd.AddCommand(rendimientoCmd)
}

// medidaContexto: lo que pesa tools/list, medido en el mismo proceso. Es lo que el MCP de qdd suma a cada sesión.
func medidaContexto() rendimiento.Medida {
	s := server.NewMCPServer("rendimiento", Version)
	registerMCPTools(s)
	total := 0
	for _, st := range s.ListTools() {
		b, err := json.Marshal(st.Tool)
		if err != nil {
			return rendimiento.Evaluar(rendimiento.MedidaContexto, nil, "")
		}
		total += utf8.RuneCount(b)
	}
	v := float64(total)
	return rendimiento.Evaluar(rendimiento.MedidaContexto, &v, "")
}

// codigoRendimiento: 1 si dejó procesos, 2 si solo pasó un presupuesto, 0 si todo cupo.
func codigoRendimiento(inf rendimiento.Informe) int {
	codigo := 0
	for _, m := range inf.Medidas {
		if m.Estado == rendimiento.EstadoFalla {
			return 1
		}
		if m.Estado == rendimiento.EstadoSobre {
			codigo = 2
		}
	}
	return codigo
}

var simboloRendimiento = map[string]string{rendimiento.EstadoOK: "✓", rendimiento.EstadoSobre: "▲", rendimiento.EstadoFalla: "✗", rendimiento.EstadoSinMedir: "·"}

func imprimirRendimiento(w io.Writer, inf rendimiento.Informe, comoJSON bool) {
	if comoJSON {
		b, _ := json.MarshalIndent(inf, "", "  ")
		fmt.Fprintln(w, string(b))
		return
	}
	fmt.Fprintf(w, "qdd %s · %s/%s · %d núcleos · consola seguir %d s\n\n", inf.Version, inf.SO, inf.Arquitectura, inf.Nucleos, inf.Segundos)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, m := range inf.Medidas {
		valor := "sin medir"
		if m.Valor != nil {
			valor = strings.TrimSuffix(strings.TrimSuffix(fmt.Sprintf("%.1f", *m.Valor), "0"), ".")
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t≤ %s %s\t%s\n", simboloRendimiento[m.Estado], m.Nombre, valor, strings.TrimSuffix(fmt.Sprintf("%g", m.Presupuesto), ".0"), m.Unidad, m.Que)
	}
	tw.Flush()
	for _, p := range inf.Restos {
		fmt.Fprintf(w, "\n[!] Quedó vivo y se cerró: %s (PID %d)", p.Nombre, p.PID)
	}
	fmt.Fprintf(w, "\n%s\n", inf.Resumen())
}

const nombreRendimiento = "Rendimiento medido en este equipo"

// checkRendimiento: lo último medido de la versión que corre. Sin medir no es una falla, pero dice cómo medirlo.
func checkRendimiento() *CheckItem {
	inf, err := rendimiento.Ultimo(Version)
	if errors.Is(err, rendimiento.ErrSinMedir) {
		return &CheckItem{Name: nombreRendimiento, Success: true, Error: strings.TrimPrefix(Version, "v") + ": sin medir (qdd rendimiento, ~45 s, sin tokens)"}
	}
	if err != nil {
		return &CheckItem{Name: nombreRendimiento, Error: err.Error()}
	}
	detalle := inf.Resumen() + " · " + inf.Cuando.Local().Format("2006-01-02 15:04")
	if inf.Falla() {
		return &CheckItem{Name: nombreRendimiento, Error: detalle + " · el detalle: qdd rendimiento"}
	}
	return &CheckItem{Name: nombreRendimiento, Success: true, Error: detalle}
}
