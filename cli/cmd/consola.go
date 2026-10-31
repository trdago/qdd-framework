package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	Short: "Panel de tareas y barra de QDD en Claude Code (activar | desactivar | estado | actualizar)",
	Long: `Un solo interruptor para la consola de Claude Code:

  qdd consola activar [--sin-barra] [--recursos | --sin-recursos]
                                     instala el panel de tareas (mod qdd-tareas@qdd) y la barra en Go;
                                     --recursos suma la línea «D N% · G0 N% · R N% · C N%» (disco, GPU, RAM, CPU)
  qdd consola desactivar             lo quita y devuelve la barra que había
  qdd consola estado                 dice si está activo y al día (lo mismo que revisa 'qdd doctor')
  qdd consola actualizar             lo deja en la versión de esta CLI (--si-activo: solo si se activó)
  qdd consola statusline             la barra: lee el JSON de la consola por la entrada estándar
  qdd consola panel                  lo que dibuja el panel, en JSON: tareas y acciones
  qdd consola accion <id> [--confirmado]  ejecuta una acción que ofrece 'qdd consola panel'
  qdd consola pedir -m <motivo> <comando…>  deja un comando en el panel para que la persona lo ejecute
  qdd consola pedidos | descartar <id>      lista o quita los comandos que esperan en el panel

El panel se abre solo en los proyectos con .qdd/project/estado.yaml; /tareas lo abre y /tareas cerrar lo cierra.
Desde el panel se ejecutan los comandos que la IA dejó para la persona (a, b, d…), se aprueban las fichas en
borrador (1…9), se revisa el estado (c), se audita (u), se abre la carpeta (o) y se descarta una tarea con su ✕;
lo que escribe pide «s  Sí» antes de correr.
Si OpenCode está en la máquina, activar también deja el panel en su barra lateral, solo para mirar: lee cada 2 s
'qdd consola panel --json' y queda en la versión de esta CLI (actualizar y 'qdd doctor --fix' lo sincronizan).
Una barra propia (statusLine) no se toca.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
}

var consolaActivarCmd = &cobra.Command{Use: "activar", Short: "Instala el panel y la barra de QDD en Claude Code (y el panel en OpenCode, si está)", Run: runConsolaActivar}
var consolaDesactivarCmd = &cobra.Command{Use: "desactivar", Short: "Quita el panel y devuelve la barra anterior", Run: runConsolaDesactivar}
var consolaEstadoCmd = &cobra.Command{Use: "estado", Short: "Dice si la consola está activa y al día", Run: runConsolaEstado}
var consolaActualizarCmd = &cobra.Command{Use: "actualizar", Short: "Deja el panel en la versión de esta CLI", Run: runConsolaActualizar}
var consolaStatuslineCmd = &cobra.Command{Use: "statusline", Short: "Barra de estado de Claude Code (JSON por la entrada estándar); nunca falla", Run: runConsolaStatusline}
var consolaPanelCmd = &cobra.Command{Use: "panel", Short: "Lo que dibuja el panel, en JSON (tareas y acciones); nunca falla", Run: runConsolaPanel}
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
	datos := consola.Panel(estadoRaiz(), time.Now())
	datos.Version = Version
	out, err := json.Marshal(datos)
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
	// Un pedido se lanza aparte y el panel queda libre; el resultado llega por la cola (FND-089).
	return a.Lanzar(root, bin, salida)
}

// qdd consola correr <id> (oculto): lo lanza 'qdd consola accion' en su propia sesión para correr un pedido que ya
// marcó como ejecutando; no se llama a mano.
var consolaCorrerCmd = &cobra.Command{
	Use:    "correr <id>",
	Short:  "Corre un pedido lanzado desde el panel (interno)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := consolaRaiz
		if root == "" {
			root = estadoRaiz()
		}
		os.Exit(consola.CorrerLanzado(root, args[0], os.Stdout))
	},
}

var consolaRaiz string

// RFC 0006 §3.9: comandos que la IA deja para que la persona los ejecute desde el panel.
var consolaMotivo, consolaCarpeta string
var consolaDespues, consolaAntes []string

var consolaPedirCmd = &cobra.Command{
	Use:   "pedir -m <motivo> [--carpeta <dir>] [--despues <id>] [--antes <id>] <comando…>",
	Short: "Deja un comando en el panel para que la persona lo ejecute con un clic (no lo ejecuta)",
	Long: `Deja un comando en la cola del panel del proyecto. No lo ejecuta: corre solo cuando la persona lo confirma
en el panel («s  Sí»), que muestra el motivo, el comando exacto y la carpeta. Si termina bien queda hecho (en gris, no
se acepta dos veces); si falla queda para reintentarlo o descartarlo (x). Para comandos que la IA no puede o no debe ejecutar, como publicar.

Si un pedido depende de otro, dilo: --despues <id|tecla> lo deja esperando a ese (el panel lo muestra «espera a [a]»,
sin botón, hasta que quede hecho) y --antes <id|tecla> hace que ese otro espere a este. El panel los ordena solo.
Pedir de nuevo un comando que ya está, con --despues o --antes, lo reordena sin duplicarlo.
Si depende de algo fuera de la cola (un tag, un CI, el merge de otra sesión), el comando lo revisa primero y sale con
'exit 75' si todavía no está: el panel lo muestra en gris, «todavía no», no como falla. Un comando que corre en otro
repositorio (--carpeta) queda en el panel de ese repositorio; los worktrees comparten el del checkout principal.

El comando corre aparte del panel (FND-089): recargar el panel o cerrar la terminal no lo corta. Igual escríbelo
para que se pueda repetir (si algo lo corta a medias, la persona lo reintenta): revisa lo ya hecho antes de hacerlo
(un tag que ya existe se reusa, un PR ya mergeado no se vuelve a mergear). Y no esperes adentro: si falta algo, sal
con 'exit 75' y la persona lo vuelve a apretar.

  qdd consola pedir -m "Publicar v2.5.1 (dispara npm y el binario)" -- git push origin v2.5.1
  qdd consola pedir -m "…" "git tag -a v2.5.1 -m v2.5.1 && git push origin v2.5.1"
  qdd consola pedir -m "Mergear #14, después del tag" --despues a -- gh pr merge 14 --merge`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		orden := consola.Orden{Despues: consolaDespues, Antes: consolaAntes}
		os.Exit(consolaPedir(estadoRaiz(), consolaMotivo, consolaCarpeta, orden, args, os.Stdout, os.Stderr))
	},
}

var consolaPedidosCmd = &cobra.Command{
	Use:   "pedidos",
	Short: "Lista los comandos que esperan a la persona en el panel",
	Run: func(cmd *cobra.Command, args []string) {
		listarPedidos(estadoRaiz(), os.Stdout)
	},
}

// listarPedidos muestra cada pedido con su tecla (✓ si ya está hecho, sin tecla si se está ejecutando) y su estado.
func listarPedidos(root string, salida io.Writer) {
	cola := consola.Pedidos(root)
	if len(cola) == 0 {
		fmt.Fprintln(salida, "No hay comandos esperando en el panel.")
		return
	}
	for _, p := range cola {
		fmt.Fprintf(salida, "[%s] %s · %s\n    %s\n    en %s · %s\n", marcaPedido(p), p.Motivo, estadoPedido(p, consola.EsperaDe(root, p.ID)), p.Comando, p.Carpeta, p.Creado.Local().Format("2006-01-02 15:04"))
	}
}

func marcaPedido(p consola.Pedido) string {
	if p.Estado == consola.EstadoHecho {
		return "✓"
	}
	if p.Estado == consola.EstadoEjecutando {
		return "⟳"
	}
	return p.Tecla
}

func estadoPedido(p consola.Pedido, espera []string) string {
	if len(espera) > 0 {
		return esperaTexto(espera)
	}
	cuando := ""
	if p.Fin != nil {
		cuando = " " + p.Fin.Local().Format("15:04")
	}
	if p.Estado == consola.EstadoFallo {
		return fmt.Sprintf("falló (código %d)%s", p.Codigo, cuando)
	}
	return p.Estado + cuando
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
	consolaPedirCmd.Flags().StringSliceVar(&consolaDespues, "despues", nil, "pedidos que tienen que quedar hechos antes (id o tecla, separados por coma)")
	consolaPedirCmd.Flags().StringSliceVar(&consolaAntes, "antes", nil, "pedidos que tienen que esperar a este (id o tecla, separados por coma)")
	consolaCorrerCmd.Flags().StringVar(&consolaRaiz, "raiz", "", "raíz del proyecto del pedido")
	consolaCmd.AddCommand(consolaPedirCmd, consolaPedidosCmd, consolaDescartarCmd, consolaCorrerCmd)
}

func consolaPedir(root, motivo, carpeta string, orden consola.Orden, args []string, salida, errores io.Writer) int {
	p, err := consola.PedirEnOrden(root, strings.Join(args, " "), motivo, carpeta, orden, time.Now())
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	cola := consola.ColaDe(root, p.Carpeta)
	fmt.Fprintf(salida, "[+] Quedó en %s con la tecla %s: %s%s\n    %s (en %s)\n    Corre solo cuando la persona lo confirma en el panel («s  Sí»).\n", panelDe(root, cola), p.Tecla, p.Motivo, conEspera(consola.EsperaDe(cola, p.ID)), p.Comando, p.Carpeta)
	return 0
}

// panelDe nombra el panel donde quedó un pedido: el de esta sesión o el del repositorio donde corre (FND-081).
func panelDe(root, cola string) string {
	if cola == root {
		return "el panel de estado"
	}
	return "el panel de " + filepath.Base(cola) + " (el repositorio donde corre, no el de esta sesión)"
}

// esperaTexto dice a qué pedidos espera uno: «espera a [a] [b]».
func esperaTexto(espera []string) string {
	return "espera a [" + strings.Join(espera, "] [") + "]"
}

func conEspera(espera []string) string {
	if len(espera) == 0 {
		return ""
	}
	return " · " + esperaTexto(espera)
}
