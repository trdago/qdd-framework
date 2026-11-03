package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0013 §«Preguntas desde el teléfono» (pkg/consola/preguntas.go): lo usa el mod de Claude Code para que lo que la
// consola le pregunta a la persona también se pueda responder desde QDD Móvil. Gana quien responde primero.

var (
	preguntaSegundos int
	preguntaVencida  bool
	preguntaMotivo   string
	preguntaSesion   string
	preguntaJSON     bool
)

// cadaMiradaPregunta es cada cuánto mira la cola «qdd consola pregunta esperar».
const cadaMiradaPregunta = 200 * time.Millisecond

var consolaPreguntaCmd = &cobra.Command{
	Use:   "pregunta",
	Short: "Las preguntas de la consola que también se responden desde QDD Móvil (registrar | esperar | cerrar | vencer | lista)",
	Long: `Lo usa el panel de Claude Code (mod qdd-tareas): cada pregunta de la IA (AskUserQuestion) o permiso de herramienta
que se le hace a la persona queda en una cola local del repositorio y qdd remoto la publica en el teléfono. Gana quien
responde primero, el terminal o un teléfono; un permiso desde el teléfono solo se permite una vez o se niega.

  qdd consola pregunta registrar            lee {sesion, herramienta, entrada} por la entrada estándar; devuelve {id, accion}
                                            (sin QDD Móvil activado no registra: id vacío y el motivo)
  qdd consola pregunta esperar <id>         espera hasta --segundos la respuesta de un teléfono; imprime cómo está
  qdd consola pregunta cerrar <id>          la respondió el terminal (o --vencida --motivo …)
  qdd consola pregunta vencer --sesion <id> vence las pendientes de una sesión que se cerró
  qdd consola pregunta lista [--json]       la cola de este proyecto`,
}

var consolaPreguntaRegistrarCmd = &cobra.Command{
	Use: "registrar", Short: "Registra una pregunta de la consola (JSON por la entrada estándar)", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(preguntaRegistrar(estadoRaiz(), os.Stdin, os.Stdout, os.Stderr))
	},
}

var consolaPreguntaEsperarCmd = &cobra.Command{
	Use: "esperar <id>", Short: "Espera la respuesta de un teléfono a una pregunta (hasta --segundos)", Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(preguntaEsperar(estadoRaiz(), args[0], time.Duration(preguntaSegundos)*time.Second, os.Stdout, os.Stderr))
	},
}

var consolaPreguntaCerrarCmd = &cobra.Command{
	Use: "cerrar <id>", Short: "Cierra una pregunta desde el PC: la respondió el terminal (o venció)", Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(preguntaCerrar(estadoRaiz(), args[0], preguntaVencida, preguntaMotivo, os.Stdout, os.Stderr))
	},
}

var consolaPreguntaVencerCmd = &cobra.Command{
	Use: "vencer --sesion <id>", Short: "Vence las preguntas pendientes de una sesión de Claude Code que se cerró", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(preguntaVencer(estadoRaiz(), preguntaSesion, preguntaMotivo, os.Stdout, os.Stderr))
	},
}

var consolaPreguntaListaCmd = &cobra.Command{
	Use: "lista", Short: "Las preguntas de la consola de este proyecto y cómo están", Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(preguntaLista(estadoRaiz(), preguntaJSON, os.Stdout))
	},
}

func init() {
	consolaPreguntaEsperarCmd.Flags().IntVar(&preguntaSegundos, "segundos", 10, "cuánto esperar como mucho (después dice «pendiente» y se vuelve a llamar)")
	consolaPreguntaCerrarCmd.Flags().BoolVar(&preguntaVencida, "vencida", false, "no la respondió nadie: venció (con --motivo)")
	consolaPreguntaCerrarCmd.Flags().StringVar(&preguntaMotivo, "motivo", "", "por qué venció (lo lee la persona en el teléfono)")
	consolaPreguntaVencerCmd.Flags().StringVar(&preguntaSesion, "sesion", "", "la sesión de Claude Code que se cerró")
	consolaPreguntaVencerCmd.Flags().StringVar(&preguntaMotivo, "motivo", "", "por qué venció (por defecto, que se cerró la sesión)")
	consolaPreguntaListaCmd.Flags().BoolVar(&preguntaJSON, "json", false, "salida JSON")
	consolaPreguntaCmd.AddCommand(consolaPreguntaRegistrarCmd, consolaPreguntaEsperarCmd, consolaPreguntaCerrarCmd, consolaPreguntaVencerCmd, consolaPreguntaListaCmd)
	consolaCmd.AddCommand(consolaPreguntaCmd)
}

// preguntaRegistrar registra la pregunta solo si este PC tiene QDD Móvil activado: sin teléfono no hay a quién más
// preguntarle, y el terminal pregunta como siempre.
func preguntaRegistrar(root string, entrada io.Reader, salida, errores io.Writer) int {
	if _, err := remoto.LeerCredenciales(); err != nil {
		return jsonPregunta(salida, map[string]string{"id": "", "motivo": "QDD Móvil no está activado en este PC (qdd remoto activar)"})
	}
	datos, err := io.ReadAll(io.LimitReader(entrada, 1<<20))
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	var n consola.NuevaPregunta
	if err := json.Unmarshal(datos, &n); err != nil {
		fmt.Fprintf(errores, "[!] la pregunta no se entiende: %v\n", err)
		return 1
	}
	p, err := consola.RegistrarPregunta(root, n, time.Now())
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	return jsonPregunta(salida, map[string]string{"id": p.ID, "accion": "pregunta:" + p.ID})
}

// preguntaEsperar anota que la sesión la sigue esperando y mira la cola hasta que cambie o pase el plazo; imprime cómo
// quedó (con la respuesta para Claude Code si la respondió un teléfono).
func preguntaEsperar(root, id string, plazo time.Duration, salida, errores io.Writer) int {
	p, err := consola.LatirPregunta(root, id, time.Now())
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	limite := time.Now().Add(plazo)
	for p.Estado == consola.PreguntaPendiente && time.Now().Before(limite) {
		time.Sleep(min(cadaMiradaPregunta, time.Until(limite)))
		actual, ok := consola.PreguntaDe(root, id, time.Now())
		if !ok {
			fmt.Fprintf(errores, "[!] %v\n", consola.ErrPreguntaDesconocida)
			return 1
		}
		p = actual
	}
	return jsonPregunta(salida, consola.EntregaDe(p))
}

func preguntaCerrar(root, id string, vencida bool, motivo string, salida, errores io.Writer) int {
	como := consola.EnPC
	if vencida {
		como = consola.PreguntaVencida
	}
	p, err := consola.CerrarPregunta(root, id, como, motivo, time.Now())
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	return jsonPregunta(salida, consola.EntregaDe(p))
}

func preguntaVencer(root, sesion, motivo string, salida, errores io.Writer) int {
	if motivo == "" {
		motivo = "se cerró la sesión de Claude Code que la hizo"
	}
	n, err := consola.VencerSesion(root, sesion, motivo, time.Now())
	if err != nil {
		fmt.Fprintf(errores, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintf(salida, "[+] %d pregunta(s) vencida(s)\n", n)
	return 0
}

func preguntaLista(root string, comoJSON bool, salida io.Writer) int {
	cola := consola.PreguntasA(root, time.Now())
	if comoJSON {
		return jsonPregunta(salida, cola)
	}
	if len(cola) == 0 {
		fmt.Fprintln(salida, "[i] No hay preguntas de la consola en este proyecto.")
		return 0
	}
	for _, p := range cola {
		fmt.Fprintf(salida, "%s · %s · %s\n", p.ID, estadoDePregunta(p), p.Etiqueta)
	}
	return 0
}

func estadoDePregunta(p consola.Pregunta) string {
	if p.Estado == consola.PreguntaRespondida {
		return "respondida en " + consola.LugarDe(p)
	}
	if p.Estado == consola.PreguntaVencida {
		return "vencida: " + p.Motivo
	}
	return p.Estado
}

func jsonPregunta(salida io.Writer, v any) int {
	b, err := json.Marshal(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[!] %v\n", err)
		return 1
	}
	fmt.Fprintln(salida, string(b))
	return 0
}
