package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Ejecutar desde el teléfono» (pkg/consola/telefono.go): lo usa el mod de Claude Code para que una
// instrucción mandada desde QDD Móvil llegue a la sesión abierta del proyecto, como un mensaje de la persona. Cada
// sesión interactiva late; el latido le entrega, una sola vez, lo que espera.

var (
	telefonoSesion   string
	telefonoFin      bool
	telefonoConfirma bool
	telefonoMotivo   string
	telefonoJSON     bool
)

var consolaTelefonoCmd = &cobra.Command{
	Use:   "telefono",
	Short: "Lo mandado desde QDD Móvil a este proyecto (latir | fallo | lista)",
	Long: `Lo usa el panel de Claude Code (mod qdd-tareas). Con Pro, el teléfono manda a ejecutar un comando qdd (lo corre
qdd remoto) o una instrucción para la IA, que espera a una sesión de Claude Code abierta en el proyecto.

  qdd consola telefono latir --sesion <id> [--fin]   la sesión sigue abierta (o se cerró); devuelve {activo, entregar:
                                                     [{solicitud, nombre, mensaje}]}: lo que esta sesión tiene que
                                                     darle a la IA, una sola vez (sin QDD Móvil activado: activo false)
  qdd consola telefono fallo <solicitud> --sesion <id> --motivo …   no se pudo entregar
  qdd consola telefono lista [--json]                la cola de este proyecto`,
}

var consolaTelefonoLatirCmd = &cobra.Command{
	Use: "latir --sesion <id>", Short: "La sesión sigue abierta: devuelve las instrucciones del teléfono que le tocan", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(telefonoLatir(estadoRaiz(), telefonoSesion, telefonoFin, telefonoConfirma, os.Stdout, os.Stderr))
	},
}

var consolaTelefonoEntregadaCmd = &cobra.Command{
	Use: "entregada <solicitud> --sesion <id>", Short: "Claude Code aceptó la instrucción tomada con --confirma", Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(telefonoEntregada(estadoRaiz(), args[0], telefonoSesion, os.Stdout, os.Stderr))
	},
}

var consolaTelefonoFalloCmd = &cobra.Command{
	Use: "fallo <solicitud> --sesion <id>", Short: "La instrucción no se pudo entregar a la IA", Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(telefonoFallo(estadoRaiz(), args[0], telefonoSesion, telefonoMotivo, os.Stdout, os.Stderr))
	},
}

var consolaTelefonoListaCmd = &cobra.Command{
	Use: "lista", Short: "Lo mandado desde el teléfono a este proyecto y cómo está", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(telefonoLista(estadoRaiz(), telefonoJSON, os.Stdout))
	},
}

func init() {
	consolaTelefonoLatirCmd.Flags().StringVar(&telefonoSesion, "sesion", "", "la sesión de Claude Code")
	consolaTelefonoLatirCmd.Flags().BoolVar(&telefonoFin, "fin", false, "la sesión se cerró: deja de contar en el acto")
	consolaTelefonoLatirCmd.Flags().BoolVar(&telefonoConfirma, "confirma", false, "la sesión confirmará cada entrega con 'entregada' o 'fallo'")
	consolaTelefonoEntregadaCmd.Flags().StringVar(&telefonoSesion, "sesion", "", "la sesión que la tomó")
	consolaTelefonoFalloCmd.Flags().StringVar(&telefonoSesion, "sesion", "", "la sesión que la tomó")
	consolaTelefonoFalloCmd.Flags().StringVar(&telefonoMotivo, "motivo", "", "por qué no se entregó")
	consolaTelefonoListaCmd.Flags().BoolVar(&telefonoJSON, "json", false, "salida JSON")
	consolaTelefonoCmd.AddCommand(consolaTelefonoLatirCmd, consolaTelefonoEntregadaCmd, consolaTelefonoFalloCmd, consolaTelefonoListaCmd)
	consolaCmd.AddCommand(consolaTelefonoCmd)
}

// entrega es una instrucción que la sesión le da a la IA.
type entrega struct {
	Solicitud string `json:"solicitud"`
	Nombre    string `json:"nombre"`
	Mensaje   string `json:"mensaje"`
}

// telefonoLatir anota la sesión siempre (el baúl pide su permiso a una sesión que late, FND-191) y, solo con QDD Móvil
// activado, le entrega lo que espera: sin teléfono emparejado nadie manda instrucciones.
func telefonoLatir(root, sesion string, fin, confirma bool, salida, errores io.Writer) int {
	now := time.Now()
	if fin {
		return terminarLatido(consola.CerrarSesion(root, sesion, now), nil, salida, errores)
	}
	if err := consola.LatirSesion(root, sesion, now); err != nil {
		return terminarLatido(err, nil, salida, errores)
	}
	if _, err := remoto.LeerCredenciales(); err != nil {
		return jsonPregunta(salida, map[string]any{"activo": false, "entregar": []entrega{}})
	}
	if confirma {
		tomadas, err := consola.TomarParaEntregar(root, sesion, now)
		return terminarLatido(err, tomadas, salida, errores)
	}
	tomadas, err := consola.TomarInstrucciones(root, sesion, now)
	return terminarLatido(err, tomadas, salida, errores)
}

func terminarLatido(err error, tomadas []consola.Ejecucion, salida, errores io.Writer) int {
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	out := []entrega{}
	for _, e := range tomadas {
		out = append(out, entrega{Solicitud: e.Solicitud, Nombre: e.Nombre, Mensaje: consola.MensajeInstruccion(e)})
	}
	return jsonPregunta(salida, map[string]any{"activo": true, "entregar": out})
}

func telefonoEntregada(root, solicitud, sesion string, salida, errores io.Writer) int {
	if err := consola.ConfirmarEntrega(root, solicitud, sesion, time.Now()); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintf(salida, "[+] La instrucción %s quedó entregada.\n", solicitud)
	return 0
}

func telefonoFallo(root, solicitud, sesion, motivo string, salida, errores io.Writer) int {
	if err := consola.FallarEntrega(root, solicitud, sesion, motivo, time.Now()); err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintf(salida, "[+] La instrucción %s quedó sin entregar: %s\n", solicitud, motivo)
	return 0
}

func telefonoLista(root string, comoJSON bool, salida io.Writer) int {
	cola := consola.EjecucionesA(root, time.Now())
	if comoJSON {
		return jsonPregunta(salida, cola)
	}
	if len(cola) == 0 {
		fmt.Fprintln(salida, "[i] Nada mandado desde el teléfono a este proyecto.")
		return 0
	}
	for _, e := range cola {
		fmt.Fprintf(salida, "%s · %s · %s · %s\n", e.Solicitud, e.Tipo, e.Estado, e.Texto)
	}
	return 0
}
