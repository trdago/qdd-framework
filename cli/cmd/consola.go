package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
)

var consolaCmd = &cobra.Command{
	Use:   "consola",
	Short: "Panel de estado y barra de QDD en Claude Code (activar | desactivar | estado | actualizar)",
	Long: `Un solo interruptor para la consola de Claude Code:

  qdd consola activar [--sin-barra]  instala el panel de estado por frente (mod qdd-estado@qdd) y la barra en Go
  qdd consola desactivar             lo quita y devuelve la barra que había
  qdd consola estado                 dice si está activo y al día (lo mismo que revisa 'qdd doctor')
  qdd consola actualizar             lo deja en la versión de esta CLI (--si-activo: solo si se activó)
  qdd consola statusline             la barra: lee el JSON de la consola por la entrada estándar
  qdd consola panel                  lo que dibuja el panel, en JSON: estado por frente y acciones
  qdd consola accion <id> [--confirmado]  ejecuta una acción que ofrece 'qdd consola panel'

El panel se abre solo en los proyectos con .qdd/project/estado.yaml; /estado lo abre y /estado cerrar lo cierra.
Desde el panel se aprueban las fichas en borrador (1…9), se revisa el estado (c) y se audita (u); lo que escribe
pide «s  Sí» antes de correr.
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
	imprimirHecho(consolaActual().Activar(!consolaSinBarra))
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
	if l := consola.Barra(datos, time.Now(), true); l != "" {
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
