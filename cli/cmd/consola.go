package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.6: 'qdd consola' deja el panel de estado y la barra de QDD en Claude Code con un solo interruptor. Sus
// hooks reemplazan a los de la raíz (como 'qdd estado'): la barra lo ejecuta cada 2 s y no puede escribir .qdd/working
// ni exigir un proyecto.

var (
	consolaSinBarra   bool
	consolaSiActivo   bool
	consolaConfirmado bool
	consolaRecursos   bool
	consolaSinRecurso bool
)

var consolaCmd = &cobra.Command{
	Use:   "consola",
	Short: "Panel de estado y barra de QDD en Claude Code (activar | desactivar | estado | actualizar)",
	Long: `Un solo interruptor para la consola de Claude Code:

  qdd consola activar [--sin-barra] [--recursos | --sin-recursos]
                                     instala el panel de estado por frente (mod qdd-estado@qdd) y la barra en Go;
                                     --recursos suma la línea «D N% · G0 N% · R N% · C N%» (disco, GPU, RAM, CPU)
  qdd consola desactivar             lo quita y devuelve la barra que había
  qdd consola estado                 dice si está activo y al día (lo mismo que revisa 'qdd doctor')
  qdd consola actualizar             lo deja en la versión de esta CLI (--si-activo: solo si se activó)
  qdd consola statusline             la barra: lee el JSON de la consola por la entrada estándar
  qdd consola panel                  lo que dibuja el panel, en JSON: estado por frente y acciones
  qdd consola accion <id> [--confirmado]  ejecuta una acción que ofrece 'qdd consola panel'
  qdd consola pedir -m <motivo> <comando…>  deja un comando en el panel para que la persona lo ejecute
  qdd consola pedidos | descartar <id>      lista o quita los comandos que esperan en el panel

El panel se abre solo en los proyectos con .qdd/project/estado.yaml; /estado lo abre y /estado cerrar lo cierra.
Desde el panel se ejecutan los comandos que la IA dejó para la persona (a, b, d…), se aprueban las fichas en
borrador (1…9), se revisa el estado (c) y se audita (u); lo que escribe pide «s  Sí» antes de correr.
Una barra propia (statusLine) no se toca.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
}

var consolaActivarCmd = &cobra.Command{Use: "activar", Short: "Instala el panel y la barra de QDD en Claude Code", Run: runConsolaActivar}
var consolaDesactivarCmd = &cobra.Command{Use: "desactivar", Short: "Quita el panel y devuelve la barra anterior", Run: runConsolaDesactivar}
var consolaEstadoCmd = &cobra.Command{Use: "estado", Short: "Dice si la consola está activa y al día", Run: runConsolaEstado}
var consolaActualizarCmd = &cobra.Command{Use: "actualizar", Short: "Deja el panel en la versión de esta CLI", Run: runConsolaActualizar}
var consolaStatuslineCmd = &cobra.Command{Use: "statusline", Short: "Barra de estado de Claude Code (JSON por la entrada estándar); nunca falla", Run: runConsolaStatusline}
var consolaPanelCmd = &cobra.Command{Use: "panel", Short: "Lo que dibuja el panel, en JSON (estado por frente y acciones); nunca falla", Run: runConsolaPanel}
var consolaAccionCmd = &cobra.Command{
	Use:   "accion <id>",
	Short: "Ejecuta una acción que ofrece 'qdd consola panel' (lo que escribe, solo con --confirmado)",
	Args:  cobra.ExactArgs(1),
	Run:   runConsolaAccion,
}

func init() {
	consolaActivarCmd.Flags().BoolVar(&consolaSinBarra, "sin-barra", false, "instala solo el panel; no cambia la barra (statusLine)")
	consolaActivarCmd.Flags().BoolVar(&consolaRecursos, "recursos", false, "la barra suma una línea con disco del proyecto, GPU, RAM y CPU (D · G0 · R · C)")
	consolaActivarCmd.Flags().BoolVar(&consolaSinRecurso, "sin-recursos", false, "quita la línea de recursos de la barra")
	consolaActivarCmd.MarkFlagsMutuallyExclusive("recursos", "sin-recursos")
	consolaActualizarCmd.Flags().BoolVar(&consolaSiActivo, "si-activo", false, "no hace nada si la consola no se activó (para instaladores)")
	consolaPanelCmd.Flags().Bool("json", true, "salida JSON (la única que hay)")
	consolaAccionCmd.Flags().BoolVar(&consolaConfirmado, "confirmado", false, "la persona ya confirmó (el segundo paso del panel)")
	consolaCmd.AddCommand(consolaActivarCmd, consolaDesactivarCmd, consolaEstadoCmd, consolaActualizarCmd, consolaStatuslineCmd,
		consolaPanelCmd, consolaAccionCmd)
	rootCmd.AddCommand(consolaCmd)
}

func consolaActual() consola.Consola {
	return consola.Consola{Version: Version}
}

func imprimirHecho(hecho []string, err error) {
	for _, h := range hecho {
		fmt.Println("[+] " + h)
	}
	if err != nil {
		salirCon("%v", err)
	}
}

func runConsolaActivar(cmd *cobra.Command, args []string) {
	c := consolaActual()
	if consolaRecursos || consolaSinRecurso {
		v := consolaRecursos
		c.Recursos = &v
	}
	imprimirHecho(c.Activar(!consolaSinBarra))
}

func runConsolaDesactivar(cmd *cobra.Command, args []string) {
	imprimirHecho(consolaActual().Desactivar())
}

func runConsolaActualizar(cmd *cobra.Command, args []string) {
	c := consolaActual()
	if _, ok := c.LeerRegistro(); !ok {
		if consolaSiActivo {
			return
		}
		salirCon("la consola no está activada: usa 'qdd consola activar'")
	}
	imprimirHecho(c.Reparar())
}

func runConsolaEstado(cmd *cobra.Command, args []string) {
	diags := consolaActual().Diagnosticar()
	if len(diags) == 0 {
		fmt.Println("[i] La consola de QDD no está activada en esta máquina. Actívala con 'qdd consola activar'.")
		return
	}
	malos := 0
	for _, d := range diags {
		if d.OK {
			fmt.Println("[✓] " + d.Nombre)
			continue
		}
		malos++
		fmt.Printf("[✗] %s: %s\n", d.Nombre, d.Detalle)
	}
	if malos > 0 {
		os.Exit(1)
	}
}

// runConsolaStatusline nunca falla ni escribe en stderr: con una entrada ilegible no imprime nada.
func runConsolaStatusline(cmd *cobra.Command, args []string) {
	defer func() { _ = recover() }()
	datos, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	if err != nil {
		return
	}
	c := consolaActual()
	reg, _ := c.LeerRegistro()
	l := consola.Barra(datos, time.Now(), true)
	if reg.Recursos {
		l = consola.BarraConRecursos(datos, time.Now(), true, c.DirRecursos())
	}
	if l != "" {
		fmt.Print(l)
	}
}

// runConsolaPanel siempre imprime un JSON: sin estado.yaml, el estado es null y el motivo dice cómo crearlo.
func runConsolaPanel(cmd *cobra.Command, args []string) {
	out, err := json.Marshal(consola.Panel(estadoRaiz(), time.Now()))
	if err != nil {
		salirCon("%v", err)
	}
	fmt.Println(string(out))
}

// runConsolaAccion vuelve a calcular las acciones: el panel solo pasa el id y qdd decide si existe y si está confirmada.
func runConsolaAccion(cmd *cobra.Command, args []string) {
	os.Exit(consolaAccion(estadoRaiz(), args[0], consolaConfirmado, os.Stdout, os.Stderr))
}

func consolaAccion(root, id string, confirmado bool, salida, errores io.Writer) int {
	a, err := consola.BuscarAccion(consola.Acciones(root, consola.HayEstado(root)), id)
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	if err := consola.Autorizar(a, confirmado); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	bin, err := os.Executable()
	if err != nil {
		bin = "qdd"
	}
	return a.Ejecutar(root, bin, salida)
}

// RFC 0006 §3.9: comandos que la IA deja para que la persona los ejecute desde el panel.
var consolaMotivo, consolaCarpeta string

var consolaPedirCmd = &cobra.Command{
	Use:   "pedir -m <motivo> [--carpeta <dir>] <comando…>",
	Short: "Deja un comando en el panel para que la persona lo ejecute con un clic (no lo ejecuta)",
	Long: `Deja un comando en la cola del panel del proyecto. No lo ejecuta: corre solo cuando la persona lo confirma
en el panel («s  Sí»), que muestra el motivo, el comando exacto y la carpeta. Si termina bien sale de la cola; si
falla queda para reintentarlo o descartarlo (x). Para comandos que la IA no puede o no debe ejecutar, como publicar.

  qdd consola pedir -m "Publicar v2.5.1 (dispara npm y el binario)" -- git push origin v2.5.1
  qdd consola pedir -m "…" "git tag -a v2.5.1 -m v2.5.1 && git push origin v2.5.1"`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(consolaPedir(estadoRaiz(), consolaMotivo, consolaCarpeta, args, os.Stdout, os.Stderr))
	},
}

var consolaPedidosCmd = &cobra.Command{
	Use:   "pedidos",
	Short: "Lista los comandos que esperan a la persona en el panel",
	Run: func(cmd *cobra.Command, args []string) {
		cola := consola.Pedidos(estadoRaiz())
		if len(cola) == 0 {
			fmt.Println("No hay comandos esperando en el panel.")
			return
		}
		for _, p := range cola {
			fmt.Printf("[%s] %s\n    %s\n    en %s · %s\n", consola.TeclaDe(estadoRaiz(), p.ID), p.Motivo, p.Comando, p.Carpeta, p.Creado.Local().Format("2006-01-02 15:04"))
		}
	},
}

var consolaDescartarCmd = &cobra.Command{
	Use:   "descartar <id>",
	Short: "Quita un comando de la cola del panel sin ejecutarlo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := consola.Descartar(estadoRaiz(), args[0]); err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[+] Pedido %s descartado.\n", args[0])
	},
}

func init() {
	consolaPedirCmd.Flags().StringVarP(&consolaMotivo, "motivo", "m", "", "por qué lo tiene que ejecutar la persona (se lee en el panel)")
	consolaPedirCmd.Flags().StringVar(&consolaCarpeta, "carpeta", "", "dónde corre (por defecto, la raíz del proyecto)")
	consolaCmd.AddCommand(consolaPedirCmd, consolaPedidosCmd, consolaDescartarCmd)
}

func consolaPedir(root, motivo, carpeta string, args []string, salida, errores io.Writer) int {
	p, err := consola.Pedir(root, strings.Join(args, " "), motivo, carpeta, time.Now())
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintf(salida, "[+] Quedó en el panel de estado con la tecla %s: %s\n    %s (en %s)\n    Corre solo cuando la persona lo confirma en el panel («s  Sí»).\n", consola.TeclaDe(root, p.ID), p.Motivo, p.Comando, p.Carpeta)
	return 0
}
