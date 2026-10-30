package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/integration"
	"github.com/qdd-framework/qdd/pkg/modules"
	"github.com/spf13/cobra"
)

// Rule: Prohibido usar 'else'
// RFC 0006: 'qdd estado' lee, escribe y comprueba .qdd/project/estado.yaml y lo muestra en la consola (barra de estado y
// panel en vivo). Sus hooks reemplazan a los de la raíz: no escribe .qdd/working, no migra ni exige contexto, porque la
// barra de estado lo ejecuta a cada rato y no puede ensuciarla ni demorarla.

var (
	estadoFormato   string
	estadoAlcance   string
	estadoEtapas    string
	estadoDir       string
	estadoJSON      bool
	estadoResp      string
	estadoIntervalo time.Duration
	estadoCambio    estado.Cambio
	estadoMotivo    string
	estadoDescarta  bool
	estadoFin       bool
	estadoQuien     string
	estadoFuente    string
)

var estadoCmd = &cobra.Command{
	Use:   "estado",
	Short: "Estado por frente: la tabla de frentes × etapas del proyecto (RFC 0006)",
	Long: `Lee y escribe .qdd/project/estado.yaml (blueprint estado-por-tareas, antes estado-por-frente; destino «archivo»).

  qdd estado [--json]                imprime la tabla (markdown o mensajería, según la ficha; --json para un panel)
  qdd estado init                    crea el archivo con las etapas de la ficha
  qdd estado set <frente> <etapa> <ok|espera|falla|na> [--evidencia …] [--bloqueo … --de-quien …] [--falla …]
  qdd estado cerrar <frente> --motivo "…" [--descartado]
  qdd estado activa <tarea> [--fin]  la tarea en curso: el panel la muestra «● running» (blueprint estado-por-tareas)
  qdd estado pulso [<tarea>] [--fuente agente|subagente|comando]  una señal de vida (sin tarea, la última activada)
  qdd estado check [--respuesta archivo.md] [--json]
  qdd estado statusline [--dir ruta] una línea para la barra de estado de la consola
  qdd estado watch                   panel en vivo (ponlo en un panel dividido de la terminal)`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { return nil },
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	Run:               runEstadoVer,
}

var estadoVerCmd = &cobra.Command{Use: "ver", Short: "Imprime la tabla", Run: runEstadoVer}

var estadoInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Crea .qdd/project/estado.yaml con las etapas de la ficha (o --etapas)",
	Run:   runEstadoInit,
}

var estadoSetCmd = &cobra.Command{
	Use:   "set <frente> <etapa> <estado>",
	Short: "Crea o actualiza una celda; valida antes de escribir",
	Args:  cobra.ExactArgs(3),
	Run:   runEstadoSet,
}

var estadoCerrarCmd = &cobra.Command{
	Use:   "cerrar <frente>",
	Short: "Cierra (o descarta) un frente con su motivo: un frente nunca desaparece en silencio",
	Args:  cobra.ExactArgs(1),
	Run:   runEstadoCerrar,
}

var estadoActivaCmd = &cobra.Command{
	Use:   "activa <tarea>",
	Short: "Declara la tarea en curso (con --fin la apaga); se apaga sola a los 120 s sin pulsos",
	Args:  cobra.ExactArgs(1),
	Run:   runEstadoActiva,
}
var estadoPulsoCmd = &cobra.Command{
	Use:   "pulso [<tarea>]",
	Short: "Señal de vida de una tarea en curso (sin tarea, la última activada); para scripts, tests, deploys y workers",
	Args:  cobra.MaximumNArgs(1),
	Run:   runEstadoPulso,
}
var estadoCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Lector automático: escenarios EPF-C01…C07 (y EPF-A01 con --respuesta); sale con 1 si hay violaciones",
	Run:   runEstadoCheck,
}

var estadoStatuslineCmd = &cobra.Command{
	Use:   "statusline",
	Short: "Resumen de una línea para la barra de estado; nunca falla (sin archivo no imprime nada)",
	Run:   runEstadoStatusline,
}

var estadoWatchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Panel en vivo con la tabla, lo que espera cada ⏳ y las fallas (q o Ctrl-C para salir)",
	Run:   runEstadoWatch,
}

func init() {
	estadoCmd.PersistentFlags().StringVar(&estadoDir, "dir", "", "carpeta del proyecto (por defecto, la actual)")
	for _, c := range []*cobra.Command{estadoCmd, estadoVerCmd} {
		c.Flags().StringVar(&estadoFormato, "formato", "", "markdown | mensajeria (por defecto, el de la ficha)")
		c.Flags().StringVar(&estadoAlcance, "alcance", "", "abiertos | todos (por defecto, el de la ficha)")
		c.Flags().BoolVar(&estadoJSON, "json", false, "salida JSON para quien dibuja el estado (p. ej. el mod de Claude Code)")
	}
	estadoInitCmd.Flags().StringVar(&estadoEtapas, "etapas", "", "etapas separadas por coma (si la ficha dice «propias»)")
	f := estadoSetCmd.Flags()
	f.StringVar(&estadoCambio.Nombre, "nombre", "", "nombre del frente (al crearlo o para renombrarlo)")
	f.StringVar(&estadoCambio.Evidencia, "evidencia", "", "evidencia del ✅: commit, versión, n/n pruebas o id de corrida")
	f.StringVar(&estadoCambio.Bloqueo, "bloqueo", "", "qué espera el ⏳")
	f.StringVar(&estadoCambio.DeQuien, "de-quien", "", "de quién lo espera el ⏳")
	f.StringVar(&estadoCambio.Falla, "falla", "", "la falla del ❌, nombrada")
	f.StringVar(&estadoCambio.RequiereAprobacion, "requiere-aprobacion", "", "quién debe aprobar antes de pasar a ✅")
	f.BoolVar(&estadoCambio.Aprobado, "aprobado", false, "marca la aprobación como dada")
	estadoCerrarCmd.Flags().StringVar(&estadoMotivo, "motivo", "", "por qué se cierra (obligatorio)")
	estadoCerrarCmd.Flags().BoolVar(&estadoDescarta, "descartado", false, "se descarta en vez de cerrarse")
	estadoActivaCmd.Flags().BoolVar(&estadoFin, "fin", false, "apaga la tarea en vez de declararla en curso")
	estadoActivaCmd.Flags().StringVar(&estadoQuien, "quien", "", "quién la trabaja (opcional, p. ej. claude-code)")
	estadoPulsoCmd.Flags().StringVar(&estadoFuente, "fuente", estado.FuenteComando, "agente | subagente | comando")
	estadoCheckCmd.Flags().StringVar(&estadoResp, "respuesta", "", "archivo con la respuesta o el reporte (EPF-A01)")
	estadoCheckCmd.Flags().BoolVar(&estadoJSON, "json", false, "salida JSON")
	estadoWatchCmd.Flags().DurationVar(&estadoIntervalo, "intervalo", time.Second, "cada cuánto revisa si el archivo cambió")

	estadoCmd.AddCommand(estadoVerCmd, estadoInitCmd, estadoSetCmd, estadoCerrarCmd, estadoActivaCmd, estadoPulsoCmd, estadoCheckCmd, estadoStatuslineCmd, estadoWatchCmd)
	rootCmd.AddCommand(estadoCmd)
}

func estadoRaiz() string {
	dir := estadoDir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "."
		}
		dir = cwd
	}
	if raiz := estado.BuscarRaiz(dir); raiz != "" {
		return raiz
	}
	return integration.FindProjectRoot(dir)
}

func estadoOpciones(root string) map[string]string {
	m, ok := fichaDelEstado(root)
	if !ok {
		return map[string]string{}
	}
	return estado.Opciones(m.ObjetivoTecnologico.Decisiones)
}

// fichaDelEstado busca la ficha por el blueprint que la generó (un proyecto migrado conserva el id de su módulo) y,
// si no, por los ids de esos blueprints.
func fichaDelEstado(root string) (modules.Module, bool) {
	if m, ok := blueprints.FichaDeBlueprint(root, estado.Blueprints...); ok {
		return m, true
	}
	for _, id := range estado.Blueprints {
		if m, ok := modules.Find(root, id); ok {
			return m, true
		}
	}
	return modules.Module{}, false
}

func salirCon(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[!] "+format+"\n", a...)
	os.Exit(1)
}

func cargarEstado(root string) estado.Estado {
	e, err := estado.Load(root)
	if os.IsNotExist(err) {
		salirCon("%v", estado.ErrSinArchivo)
	}
	if err != nil {
		salirCon("%v", err)
	}
	return e
}

func runEstadoVer(cmd *cobra.Command, args []string) {
	root := estadoRaiz()
	e := cargarEstado(root)
	if estadoJSON {
		out, _ := json.Marshal(estado.Datos(e, time.Now()))
		fmt.Println(string(out))
		return
	}
	op := estadoOpciones(root)
	formato := primero(estadoFormato, op["formato"], "markdown")
	alcance := estado.Alcance(primero(estadoAlcance, op["alcance"], string(estado.AlcanceAbiertos)))
	if formato == "mensajeria" {
		fmt.Print(estado.Mensajeria(e, time.Now(), alcance))
		return
	}
	fmt.Print(estado.Markdown(e, time.Now(), alcance))
}

func primero(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}

func runEstadoInit(cmd *cobra.Command, args []string) {
	root := estadoRaiz()
	etapas := estado.EtapasDe(estadoOpciones(root)["etapas"])
	if estadoEtapas != "" {
		etapas = nil
		for _, s := range strings.Split(estadoEtapas, ",") {
			if t := strings.TrimSpace(s); t != "" {
				etapas = append(etapas, t)
			}
		}
	}
	if len(etapas) == 0 {
		salirCon("la ficha del estado (%s) no define etapas conocidas: pásalas con --etapas \"A,B,C\"", strings.Join(estado.Blueprints, " o "))
	}
	if err := estado.Init(root, etapas, time.Now()); err != nil {
		salirCon("%v", err)
	}
	fmt.Printf("[+] %s creado con las etapas: %s\n", estado.Ruta, strings.Join(etapas, " · "))
}

func runEstadoSet(cmd *cobra.Command, args []string) {
	c := estadoCambio
	c.Frente, c.Etapa, c.Estado = args[0], args[1], args[2]
	if err := estado.Set(estadoRaiz(), c, time.Now()); err != nil {
		salirCon("%v", err)
	}
	fmt.Printf("[+] %s › %s = %s\n", c.Frente, c.Etapa, estado.NormalizarEstado(c.Estado))
}

func runEstadoCerrar(cmd *cobra.Command, args []string) {
	if err := estado.Cerrar(estadoRaiz(), args[0], estadoMotivo, estadoDescarta, time.Now()); err != nil {
		salirCon("%v", err)
	}
	fmt.Printf("[+] %s cerrado: %s\n", args[0], estadoMotivo)
}

func runEstadoActiva(cmd *cobra.Command, args []string) {
	root := estadoRaiz()
	if estadoFin {
		if err := estado.Terminar(root, args[0]); err != nil {
			salirCon("%v", err)
		}
		fmt.Printf("[+] %s ya no está en curso\n", args[0])
		return
	}
	if err := estado.Activar(root, args[0], estadoQuien, time.Now()); err != nil {
		salirCon("%v", err)
	}
	fmt.Printf("[+] %s en curso: el panel la muestra «● running»; se apaga sola a los %d s sin pulsos\n", args[0], int(estado.VidaActividad.Seconds()))
}

// runEstadoPulso no imprime nada si no hay tarea en curso: lo llaman el mod en cada herramienta y los scripts.
func runEstadoPulso(cmd *cobra.Command, args []string) {
	id := ""
	if len(args) == 1 {
		id = args[0]
	}
	if _, err := estado.Pulsar(estadoRaiz(), id, estadoFuente, time.Now()); err != nil {
		salirCon("%v", err)
	}
}

func runEstadoCheck(cmd *cobra.Command, args []string) {
	root := estadoRaiz()
	e := cargarEstado(root)
	vs := estado.Check(e, estado.Anterior(root))
	if estadoResp != "" {
		data, err := os.ReadFile(estadoResp)
		if err != nil {
			salirCon("no se pudo leer la respuesta: %v", err)
		}
		vs = append(vs, estado.CheckRespuesta(e, string(data), time.Now())...)
	}
	if estadoJSON {
		out, _ := json.MarshalIndent(map[string]any{"violaciones": vs, "revision_humana": []string{"EPF-C08"}}, "", "  ")
		fmt.Println(string(out))
		salirSiHay(vs)
		return
	}
	for _, v := range vs {
		fmt.Println(v.String())
	}
	if len(vs) == 0 {
		fmt.Printf("[✓] %s cumple EPF-C01…C07", estado.Ruta)
		if estadoResp != "" {
			fmt.Print(" y EPF-A01")
		}
		fmt.Println(". EPF-C08 (texto y tabla no se contradicen) queda para revisión humana.")
	}
	salirSiHay(vs)
}

func salirSiHay(vs []estado.Violacion) {
	if len(vs) > 0 {
		os.Exit(1)
	}
}

// runEstadoStatusline nunca falla ni escribe en stderr, y no tarda más de 50 ms: si algo sale mal, la barra no
// muestra el segmento.
func runEstadoStatusline(cmd *cobra.Command, args []string) {
	dir := estadoDir
	if dir == "" {
		dir, _ = os.Getwd()
	}
	linea := make(chan string, 1)
	go func() {
		defer func() { _ = recover() }()
		linea <- estado.StatuslineDir(dir, time.Now())
	}()
	select {
	case l := <-linea:
		if l != "" {
			fmt.Print(l)
		}
	case <-time.After(50 * time.Millisecond):
	}
}
