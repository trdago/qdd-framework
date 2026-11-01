package cmd

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/incidentes"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// 'qdd incidentes': las fallas de QDD mismo (un pánico de la CLI o de una tool MCP, una autorreparación de doctor)
// quedan en un registro local anonimizado y se publican solas como issue del repositorio de QDD, sin duplicar
// (pedido del usuario, 2026-11-01). Las fallas del proyecto siguen como findings locales (qdd bug).

var incidentesSilencioso bool

var incidentesCmd = &cobra.Command{
	Use:   "incidentes",
	Short: "Fallas de QDD mismo que se publican solas como issue de QDD, anonimizadas (ver | publicar)",
	Long: `Un pánico de la CLI o de una tool MCP y cada autorreparación de qdd doctor quedan en <QDD_HOME>/incidentes,
un archivo por defecto (su firma), y se publican solos en segundo plano como issue del repositorio de QDD: la misma
firma suma una ocurrencia y comenta el issue que ya existe, con un tope de 5 publicaciones por día.

Solo viajan datos de QDD: versión, sistema, el comando con sus valores como <marcador> y la traza dentro del código
de QDD. Nunca rutas, nombres de archivo, valores ni datos del proyecto.

  qdd incidentes              lista el registro y dónde se publicó cada uno
  qdd incidentes ver <firma>  muestra el issue exactamente como se publica
  qdd incidentes publicar     publica ahora los pendientes

QDD_INCIDENTES=off los apaga.`,
	Run: func(cmd *cobra.Command, args []string) { os.Exit(listarIncidentes(os.Stdout)) },
}

var incidentesVerCmd = &cobra.Command{
	Use:   "ver <firma>",
	Short: "Muestra el issue de un incidente exactamente como se publica",
	Args:  cobra.ExactArgs(1),
	Run:   func(cmd *cobra.Command, args []string) { os.Exit(verIncidente(args[0], os.Stdout)) },
}

var incidentesPublicarCmd = &cobra.Command{
	Use:   "publicar",
	Short: "Publica ahora los incidentes pendientes (sin duplicar, con el tope del día)",
	Run: func(cmd *cobra.Command, args []string) {
		os.Exit(publicarIncidentes(incidentes.GH{Repo: incidentes.Repo()}, time.Now(), os.Stdout))
	},
}

func init() {
	incidentesPublicarCmd.Flags().BoolVar(&incidentesSilencioso, "silencioso", false, "sin salida (lo usa el lanzamiento en segundo plano)")
	incidentesCmd.AddCommand(incidentesVerCmd, incidentesPublicarCmd)
	rootCmd.AddCommand(incidentesCmd)
}

func listarIncidentes(out io.Writer) int {
	dir, err := incidentes.Dir()
	if err != nil {
		fmt.Fprintf(out, "[!] %v\n", err)
		return 1
	}
	todos, errLeer := incidentes.Leer(dir)
	if len(todos) == 0 && errLeer == nil {
		fmt.Fprintln(out, "No hay incidentes de QDD registrados en esta máquina.")
		return 0
	}
	for _, i := range todos {
		donde := i.Issue
		if i.Pendiente() {
			donde = strings.TrimSpace(fmt.Sprintf("pendiente (%d de %d publicadas) %s", i.Publicadas, i.Ocurrencias, i.Issue))
		}
		fmt.Fprintf(out, "%s  %s en %s ×%d · %s\n    %s\n", i.Firma, i.Tipo, i.Origen, i.Ocurrencias, donde, i.Mensaje)
	}
	if errLeer != nil {
		fmt.Fprintf(out, "[!] %v (qdd doctor --fix lo aparta)\n", errLeer)
	}
	return 0
}

func verIncidente(firma string, out io.Writer) int {
	dir, err := incidentes.Dir()
	if err != nil {
		fmt.Fprintf(out, "[!] %v\n", err)
		return 1
	}
	todos, _ := incidentes.Leer(dir)
	for _, i := range todos {
		if i.Firma != firma {
			continue
		}
		fmt.Fprintf(out, "%s\n\n%s", incidentes.Titulo(i), incidentes.Cuerpo(i))
		return 0
	}
	fmt.Fprintf(out, "[!] no hay un incidente con la firma %s: qdd incidentes los lista\n", firma)
	return 1
}

func publicarIncidentes(p incidentes.Publicador, now time.Time, out io.Writer) int {
	if incidentesSilencioso {
		out = io.Discard
	}
	if !incidentes.Activo() {
		fmt.Fprintln(out, "Los incidentes están apagados (QDD_INCIDENTES=off).")
		return 0
	}
	dir, err := incidentes.Dir()
	if err != nil {
		fmt.Fprintf(out, "[!] %v\n", err)
		return 1
	}
	r := incidentes.Publicar(dir, p, now, incidentes.TopeDiario)
	fmt.Fprintf(out, "[+] %d issue(s) nuevo(s), %d comentario(s); %d pendiente(s)\n", r.Creados, r.Comentados, r.Pendientes)
	if r.Err != nil {
		fmt.Fprintf(out, "[!] %v\n", r.Err)
		return 1
	}
	return 0
}

// registrarIncidente guarda la falla anonimizada y lanza la publicación aparte; nunca falla ni demora a quien la
// llama: si algo no resulta, la falla original sigue su curso.
func registrarIncidente(tipo, origen, comando, mensaje string, traza []string, reparacion string) {
	defer func() { _ = recover() }()
	if !incidentes.Activo() {
		return
	}
	dir, err := incidentes.Dir()
	if err != nil {
		return
	}
	cwd, _ := os.Getwd()
	ctx := incidentes.ContextoDe(cwd)
	i := incidentes.Nuevo(tipo, origen, incidentes.Sanitizar(mensaje, ctx), traza, Version)
	i.Comando, i.Reparacion = comando, incidentes.Sanitizar(reparacion, ctx)
	if _, err := incidentes.Registrar(dir, i, time.Now()); err != nil {
		return
	}
	bin, _ := os.Executable()
	incidentes.Lanzar(dir, bin, time.Now())
}

// registrarPanico: un pánico atrapado (CLI o tool MCP) es un defecto de QDD aunque no haya tumbado el proceso. origen
// es el comando o la tool (entra en la firma); comando, cómo se llamó, con sus valores como marcadores.
func registrarPanico(origen, comando string, r any) {
	registrarIncidente(incidentes.TipoPanico, origen, comando, fmt.Sprint(r), incidentes.Traza(debug.Stack()), "")
}

// registrarAutorreparacion: doctor reparó algo que QDD mismo escribió y quedó mal (un registro ilegible, una cola
// rota). La reparación funciona, pero el estado que la pidió no debió darse.
func registrarAutorreparacion(origen, encontrado string, hecho []string) {
	if len(hecho) == 0 {
		return
	}
	registrarIncidente(incidentes.TipoAutorreparacion, origen, "", encontrado, nil, strings.Join(hecho, "; "))
}

// origenCLI es el comando de QDD que corría («qdd estado set») y cómo se llamó, con sus valores como marcadores.
func origenCLI(args []string) (string, string) {
	c, resto, err := rootCmd.Find(args)
	if err != nil || c == nil {
		return "qdd", "qdd"
	}
	return c.CommandPath(), incidentes.Comando(c.CommandPath(), resto)
}

// grupoIncidentes revisa el registro de incidentes de esta máquina, si existe: un archivo ilegible es falla (--fix lo
// aparta, sin borrarlo) y lo pendiente de publicar se informa con el motivo.
const grupoIncidentes = "8. Incidentes de QDD (qdd incidentes)"

const nombreIncidentes = "Registro de incidentes de QDD"

func buildIncidentesChecks() *CheckGroup {
	g := &CheckGroup{Name: grupoIncidentes}
	dir, err := incidentes.Dir()
	if err != nil {
		return g
	}
	if _, err := os.Stat(dir); err != nil {
		return g
	}
	todos, errLeer := incidentes.Leer(dir)
	if errLeer != nil {
		g.Items = append(g.Items, &CheckItem{Name: nombreIncidentes, Error: errLeer.Error() + ": qdd doctor --fix lo aparta (queda como .ilegible)"})
		return g
	}
	pendientes := 0
	for _, i := range todos {
		if i.Pendiente() {
			pendientes++
		}
	}
	detalle := fmt.Sprintf("%d incidente(s), %d pendiente(s) de publicar", len(todos), pendientes)
	if !incidentes.Activo() {
		detalle += " (apagados con QDD_INCIDENTES=off)"
	}
	g.Items = append(g.Items, &CheckItem{Name: nombreIncidentes, Success: true, Error: detalle})
	return g
}

func repairIncidentes() {
	dir, err := incidentes.Dir()
	if err != nil {
		return
	}
	fmt.Println("  [~] Apartando los incidentes ilegibles...")
	imprimirReparado(incidentes.ApartarIlegibles(dir))
}
