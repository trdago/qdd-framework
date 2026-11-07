package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/flujo"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'

const anchoFlujoPorDefecto = 60

var (
	flujoJSON    bool
	flujoAncho   int
	flujoElegido string
)

var flujoCmd = &cobra.Command{
	Use:     "flujo",
	Aliases: []string{"flujos"},
	Short:   "Flujos del proyecto como nodos que el panel dibuja hacia abajo (RFC 0012)",
	Long: `Un flujo muestra cómo funciona algo complejo con cajas y círculos, para que la persona y la IA vean lo mismo en vez
de describirlo en prosa. Se guarda en ` + flujo.Carpeta + `/<id>.yaml, versionado. El panel lo abre en su propia pestaña.

  qdd flujo crear <archivo.yaml|->   valida y guarda (o reemplaza) el flujo; conserva sus comentarios
  qdd flujo nuevo "<nombre>"         un flujo con un solo nodo, para seguir editándolo
  qdd flujo ver <id> [--ancho N] [--elegido <nodo>] [--json]
  qdd flujo lista [--json]

La persona lo edita y lo comenta en el panel; a ti te llega una línea por cambio. Para responderle o cambiarlo:

  qdd flujo comentar <id> <nodo> "<texto>"    responde en el nodo; lo ve debajo de él
  qdd flujo detalle <id> <nodo> "<texto>"     explica el nodo: es la respuesta a un 🔎
  qdd flujo responder <id> [<nodo>] "<texto>" responde un ❓ de la persona, en el nodo o en el flujo completo
  qdd flujo item marcar <id> <nodo> <item> pasa|falla --evidencia E   marca una prueba del nodo (qdd-human)
  qdd flujo item agregar <id> <nodo> <prueba|nota|caracteristica|funcion> "<texto>"
  qdd flujo agregar <id> <nodo> --nombre N [--tipo paso|resultado] [--descripcion D] [--rama]
  qdd flujo modificar <id> <nodo> [--nombre N] [--descripcion D] [--tipo T]
  qdd flujo mover <id> <nodo> --despues-de <otro>
  qdd flujo quitar <id> <nodo>
  qdd flujo conectar|desconectar <id> <nodo> --con <otro>

Forma del archivo; empieza en el primer nodo y a todos tiene que poder llegarse desde ahí:

  formato: 1
  id: pago
  nombre: Cómo se cobra
  nodos:
    - {id: llega, tipo: resultado, nombre: Llega la solicitud, siguen: [firma]}
    - {id: firma, tipo: paso, nombre: Valida la firma, descripcion: rechaza si vence, siguen: [ok, rechazo]}
    - {id: ok, tipo: resultado, nombre: Cobrado}
    - {id: rechazo, tipo: resultado, nombre: Rechazado}

tipo es paso (caja) o resultado (círculo). tarea: <id> muestra el estado de esa tarea del tablero.`,
	// El panel lo llama cada vez que un flujo cambia: sin las comprobaciones de cada comando, como qdd estado.
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
}

var flujoCrearCmd = &cobra.Command{
	Use:   "crear <archivo.yaml|->",
	Short: "Valida y guarda un flujo; con - lo lee de la entrada",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		datos, err := leerArchivoOEntrada(args[0], cmd.InOrStdin())
		if err != nil {
			salirCon("%v", err)
		}
		f, err := flujo.Leer(datos)
		if err != nil {
			salirCon("El flujo no es válido:\n%v", err)
		}
		movidos, err := flujo.Reemplazar(estadoRaiz(), f)
		if err != nil {
			salirCon("%v", err)
		}
		if movidos > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "[!] %d comentario(s) de nodos que ya no están pasaron al primer nodo.\n", movidos)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "[+] Flujo '%s' guardado en %s/%s.yaml (%d nodos). El panel lo abre en su pestaña.\n", f.ID, flujo.Carpeta, f.ID, len(f.Nodos))
	},
}

func leerArchivoOEntrada(ruta string, entrada io.Reader) ([]byte, error) {
	if ruta == "-" {
		return io.ReadAll(entrada)
	}
	return os.ReadFile(ruta)
}

// flujoDibujado es lo que recibe el panel: el dibujo ya hecho, para no repetir el trazado en cada superficie.
type flujoDibujado struct {
	ID            string       `json:"id"`
	Nombre        string       `json:"nombre"`
	Descripcion   string       `json:"descripcion,omitempty"`
	Lineas        []string     `json:"lineas"`
	Nodos         []flujo.Nodo `json:"nodos"`
	PorEnviar     []string     `json:"por_enviar"`
	PuedeDeshacer bool         `json:"puede_deshacer"`

	Comentarios []flujo.Comentario `json:"comentarios,omitempty"` // preguntas y respuestas sobre el flujo completo
}

var flujoVerCmd = &cobra.Command{
	Use:   "ver <id>",
	Short: "Dibuja un flujo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		f, err := flujo.Cargar(root, args[0])
		if err != nil {
			salirCon("%v", err)
		}
		lineas := flujo.DibujarElegido(f, flujoAncho, estadoDeTarea(root), flujoElegido)
		salida := cmd.OutOrStdout()
		if flujoJSON {
			b, err := json.Marshal(flujoDibujado{
				ID: f.ID, Nombre: f.Nombre, Descripcion: f.Descripcion, Lineas: lineas, Nodos: f.Nodos,
				PorEnviar: flujo.PorEnviar(root, f.ID), PuedeDeshacer: flujo.PuedeDeshacer(root, f.ID), Comentarios: f.Comentarios,
			})
			if err != nil {
				salirCon("%v", err)
			}
			fmt.Fprintln(salida, string(b))
			return
		}
		fmt.Fprintln(salida, f.Nombre)
		fmt.Fprintln(salida, strings.Join(lineas, "\n"))
	},
}

// estadoDeTarea da los símbolos de una tarea del tablero, o avisa que no está: un flujo puede nombrar una que ya no exista.
func estadoDeTarea(root string) func(string) string {
	cargar := sync.OnceValues(func() (estado.Estado, error) { return estado.Load(root) })
	return func(tarea string) string {
		e, err := cargar()
		if err != nil {
			return "(sin tablero)"
		}
		simbolos, ok := e.Simbolos(tarea)
		if !ok {
			return "(tarea " + tarea + " no está)"
		}
		return strings.Join(simbolos, " ")
	}
}

var flujoListaCmd = &cobra.Command{
	Use:   "lista",
	Short: "Los flujos del proyecto",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		root := estadoRaiz()
		lista := flujo.Listar(root, estadoDeTarea(root))
		salida := cmd.OutOrStdout()
		if flujoJSON {
			b, err := json.Marshal(lista)
			if err != nil {
				salirCon("%v", err)
			}
			fmt.Fprintln(salida, string(b))
			return
		}
		if len(lista) == 0 {
			fmt.Fprintln(salida, "Este proyecto no tiene flujos. Crea uno con 'qdd flujo crear <archivo.yaml>'.")
			return
		}
		for _, r := range lista {
			if r.Problema != "" {
				fmt.Fprintf(salida, "⚠ %s: %s\n", r.ID, r.Problema)
				continue
			}
			fmt.Fprintf(salida, "%s · %s · %d nodos\n", r.ID, r.Nombre, r.Nodos)
		}
	},
}

func init() {
	flujoVerCmd.Flags().BoolVar(&flujoJSON, "json", false, "Salida JSON, para el panel")
	flujoVerCmd.Flags().IntVar(&flujoAncho, "ancho", anchoFlujoPorDefecto, "Columnas disponibles")
	flujoVerCmd.Flags().StringVar(&flujoElegido, "elegido", "", "Nodo que se dibuja resaltado")
	flujoListaCmd.Flags().BoolVar(&flujoJSON, "json", false, "Salida JSON, para el panel")
	flujoCmd.AddCommand(flujoCrearCmd, flujoVerCmd, flujoListaCmd)
	rootCmd.AddCommand(flujoCmd)
}

func flujosDelPanel(root string) json.RawMessage {
	b, err := json.Marshal(flujo.Listar(root, estadoDeTarea(root)))
	if err != nil {
		return nil
	}
	return b
}
