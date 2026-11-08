package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/qdd-framework/qdd/pkg/baul"
	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/incidentes"
	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/reportes"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// Rule: Prohibido usar 'else'
// qdd reportar (RFC 0024, fase 1): un camino directo a QDD para quien no tiene GitHub. El reporte se limpia, se muestra
// tal cual viaja y sale solo con el «sí» de la persona: en su terminal, o con un pedido 📮 del panel que solo ella
// confirma.

var (
	reportarTitulo        string
	reportarDetalle       string
	reportarDetalleDe     string
	reportarTipo          string
	preguntaEnviarReporte = "¿Enviar este reporte a QDD?"
)

// ofrecimiento es un reporte listo para ofrecer; Preguntar es nil sin la terminal de la persona.
type ofrecimiento struct {
	Reporte   reportes.Reporte
	Root      string
	Preguntar func(pregunta string) bool
	Enviar    func(reportes.Reporte) (string, error)
	Salida    io.Writer
	Ahora     time.Time
}

// ofrecerReporte muestra el texto exacto y lo envía con el «sí» de la persona; sin terminal deja el pedido 📮 (75).
func ofrecerReporte(o ofrecimiento) (int, error) {
	fmt.Fprintf(o.Salida, "Esto es exactamente lo que viaja a QDD:\n\n%s\n", o.Reporte.Vista())
	if o.Preguntar == nil {
		return dejarPedidoReporte(o)
	}
	if !o.Preguntar(preguntaEnviarReporte) {
		fmt.Fprintln(o.Salida, "No se envió.")
		return 0, nil
	}
	id, err := o.Enviar(o.Reporte)
	if err != nil {
		return 1, guardarSinEnviar(o, err)
	}
	fmt.Fprintf(o.Salida, "[✓] Enviado a QDD como %s. Gracias.\n", id)
	return 0, nil
}

// guardarSinEnviar: si el envío falla, el reporte no se pierde; queda pendiente para reintentarlo.
func guardarSinEnviar(o ofrecimiento, causa error) error {
	id, err := reportes.GuardarPendiente(o.Reporte, o.Ahora)
	if err != nil {
		return fmt.Errorf("%v (y no se pudo guardar: %v)", causa, err)
	}
	return fmt.Errorf("%v. Quedó guardado: reintenta con '%s'", causa, reportes.ComandoEnviar(id))
}

func dejarPedidoReporte(o ofrecimiento) (int, error) {
	if o.Root == "" || !existe(filepath.Join(o.Root, ".qdd")) {
		return 1, fmt.Errorf("el envío lo confirma la persona: corre 'qdd reportar' en tu terminal, o dentro de un proyecto con QDD para dejarlo en su panel")
	}
	id, err := reportes.GuardarPendiente(o.Reporte, o.Ahora)
	if err != nil {
		return 1, err
	}
	motivo := recortarMotivo("📮 Enviar reporte: " + o.Reporte.Titulo)
	if _, err := consola.PedirEnOrden(o.Root, reportes.ComandoEnviar(id), motivo, o.Root, consola.Orden{DePersona: true}, o.Ahora); err != nil {
		return 1, err
	}
	fmt.Fprintln(o.Salida, "[⏳] Quedó en el panel: se envía cuando la persona lo confirme con «s  Sí».")
	return reportes.CodigoTodaviaNo, nil
}

func recortarMotivo(s string) string {
	r := []rune(s)
	if len(r) <= consola.MaxMotivo {
		return s
	}
	return string(r[:consola.MaxMotivo-1]) + "…"
}

func existe(ruta string) bool {
	_, err := os.Stat(ruta)
	return err == nil
}

// autorizarEnvio: un pendiente sale solo desde el pedido 📮 que la persona confirmó, o con su «sí» en su terminal.
func autorizarEnvio(id, pedido string, preguntar func(string) bool, confirmado func(pedido, comando string) bool) error {
	if pedido != "" && confirmado(pedido, reportes.ComandoEnviar(id)) {
		return nil
	}
	if pedido == "" && preguntar != nil && preguntar(preguntaEnviarReporte) {
		return nil
	}
	return fmt.Errorf("el reporte lo envía la persona: con «s  Sí» en el pedido 📮 del panel, o en su terminal")
}

// relayDeReportes es el relay guardado o, sin él, el que dice el directorio firmado de fábrica.
func relayDeReportes(ctx context.Context) (remoto.ConfigRelay, error) {
	if c, err := remoto.LeerRelay(); err == nil {
		return c, nil
	}
	return remoto.DirectorioDeFabrica().Consultar(ctx, 0)
}

func enviarAlRelay(r reportes.Reporte) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cfg, err := relayDeReportes(ctx)
	if err != nil {
		return "", fmt.Errorf("no se encontró el relay de QDD: %w", err)
	}
	return reportes.Enviar(ctx, cfg, r, time.Now())
}

// preguntaDeTerminal es la pregunta en la terminal de la persona; nil desde una sesión de IA o sin terminal.
func preguntaDeTerminal() func(string) bool {
	if estado.SesionActual() != "" || !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil
	}
	return preguntarEnTerminal
}

// deReportar: reportar a QDD no exige un proyecto completo; quien no pudo terminar 'qdd init' también tiene que poder
// avisar.
func deReportar(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c == reportarCmd {
			return true
		}
	}
	return false
}

func raizActual() string {
	cwd, _ := os.Getwd()
	return estado.ProyectoDe(cwd)
}

var reportarCmd = &cobra.Command{
	Use:   "reportar --titulo \"…\" [--detalle \"…\" | --detalle-de <archivo>] [--tipo bug|mejora|pregunta]",
	Short: "Envía un problema o una idea a QDD sin GitHub: limpio, a la vista y solo con tu «sí»",
	Long: `Envía un reporte al relay de QDD, sin cuenta de GitHub. Antes de enviarlo quita secretos, valores del baúl,
la ruta del proyecto, tu carpeta y tu nombre de usuario, y te muestra el texto exacto. Sale solo con tu «sí»:
en tu terminal se pregunta; desde una IA queda un pedido 📮 en el panel que solo tú confirmas.`,
	Run: func(cmd *cobra.Command, args []string) {
		detalle := reportarDetalle
		if reportarDetalleDe != "" {
			datos, err := os.ReadFile(reportarDetalleDe)
			if err != nil {
				salirCon("%v", err)
			}
			detalle = string(datos)
		}
		root := raizActual()
		r, err := reportes.Nuevo(reportes.Datos{
			Titulo: reportarTitulo, Detalle: detalle, Tipo: reportarTipo,
			Version: strings.TrimSpace(Version), SO: runtime.GOOS + "/" + runtime.GOARCH,
		}, incidentes.ContextoDe(root), baul.ValoresDelLlavero())
		if err != nil {
			salirCon("%v", err)
		}
		codigo, err := ofrecerReporte(ofrecimiento{Reporte: r, Root: root, Preguntar: preguntaDeTerminal(), Enviar: enviarAlRelay, Salida: os.Stdout, Ahora: time.Now()})
		if err != nil {
			salirCon("%v", err)
		}
		os.Exit(codigo)
	},
}

var reportarEnviarCmd = &cobra.Command{
	Use:    "enviar <id>",
	Short:  "Envía un reporte pendiente: corre desde el pedido 📮 que la persona confirmó, o en su terminal",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r, err := reportes.LeerPendiente(args[0])
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("Esto es exactamente lo que viaja a QDD:\n\n%s\n", r.Vista())
		root := raizActual()
		confirmado := func(pedido, comando string) bool { return consola.PedidoConfirmado(root, pedido, comando, time.Now()) }
		if err := autorizarEnvio(args[0], os.Getenv(consola.VarPedido), preguntaDeTerminal(), confirmado); err != nil {
			salirCon("%v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cfg, err := relayDeReportes(ctx)
		if err != nil {
			salirCon("no se encontró el relay de QDD: %v", err)
		}
		id, err := reportes.EnviarPendiente(ctx, cfg, args[0], time.Now())
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[✓] Enviado a QDD como %s. Gracias.\n", id)
	},
}

var reportarLlaveCmd = &cobra.Command{
	Use:   "llave",
	Short: "Muestra la llave pública de esta instalación (la privada nunca sale de la máquina)",
	Run: func(cmd *cobra.Command, args []string) {
		pub, err := reportes.Publica()
		if err != nil {
			salirCon("%v", err)
		}
		fmt.Println(pub)
	},
}

var reportarEnviadosCmd = &cobra.Command{
	Use:   "enviados",
	Short: "Los reportes que esta instalación envió a QDD, con el id que les dio el relay",
	Run: func(cmd *cobra.Command, args []string) {
		lista, err := reportes.Enviados()
		if err != nil {
			salirCon("%v", err)
		}
		if len(lista) == 0 {
			fmt.Println("Esta instalación no ha enviado reportes.")
			return
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
		fmt.Fprintln(tw, "CUÁNDO\tID\tTIPO\tTÍTULO")
		for _, e := range lista {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.En.Local().Format("2006-01-02 15:04"), e.ID, e.Reporte.Tipo, e.Reporte.Titulo)
		}
		tw.Flush()
	},
}

func init() {
	reportarCmd.Flags().StringVar(&reportarTitulo, "titulo", "", "qué pasó, en una línea (máx. 120)")
	reportarCmd.Flags().StringVar(&reportarDetalle, "detalle", "", "pasos, qué esperabas y qué viste (máx. 6000)")
	reportarCmd.Flags().StringVar(&reportarDetalleDe, "detalle-de", "", "lee el detalle de un archivo")
	reportarCmd.Flags().StringVar(&reportarTipo, "tipo", "bug", "bug, mejora o pregunta")
	reportarCmd.MarkFlagsMutuallyExclusive("detalle", "detalle-de")
	reportarCmd.AddCommand(reportarEnviarCmd, reportarLlaveCmd, reportarEnviadosCmd)
	rootCmd.AddCommand(reportarCmd)
}
