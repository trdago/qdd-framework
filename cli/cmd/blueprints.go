package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/qdd-framework/qdd/pkg/blueprints"
	"github.com/spf13/cobra"
)

var blueprintCmd = &cobra.Command{
	Use:   "blueprint",
	Short: "Blueprints: arquetipos de módulo que tu organización mantiene en su propio repositorio",
	Long: `Un blueprint es el esqueleto conceptual de un módulo (p. ej. usuarios, roles y
permisos): vocabulario, preguntas de diseño, decisiones, invariantes y escenarios
de conformidad. Vive en el repositorio de tu organización (privado si quieres) y
se importa a cada proyecto; el core de QDD solo trae el mecanismo.

  qdd blueprint add git@github.com:org/qdd-blueprints.git//auth --ref v1.0.0
  qdd blueprint show auth
  qdd blueprint apply auth --ruta internal/auth
  qdd modules approve auth`,
	Run: runBlueprintList,
}

var blueprintAddCmd = &cobra.Command{
	Use:   "add <origen> [blueprint...]",
	Short: "Importa blueprints desde una carpeta o un repositorio git (uno, varios, un --paquete o --todos)",
	Long: `Importa blueprints con un solo clon. El origen es una carpeta local o un
repositorio git (privado si quieres; se usan tus credenciales git).

  qdd blueprint add git@github.com:org/qdd-blueprints.git --ref v1.2.0 auth auditoria
  qdd blueprint add git@github.com:org/qdd-blueprints.git --ref v1.2.0 --paquete saas
  qdd blueprint add git@github.com:org/qdd-blueprints.git --ref v1.2.0 --todos
  qdd blueprint add git@github.com:org/qdd-blueprints.git//auth --ref v1.2.0

Si alguno es inválido no se importa ninguno. Los paquetes y las carpetas se
declaran en catalogo.yaml en la raíz del repositorio.`,
	Args: cobra.MinimumNArgs(1),
	Run:  runBlueprintAdd,
}

var blueprintListCmd = &cobra.Command{
	Use:   "list",
	Short: "Lista los blueprints importados y los módulos generados con ellos",
	Run:   runBlueprintList,
}

var blueprintShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Muestra el vocabulario, las preguntas y las invariantes de un blueprint",
	Args:  cobra.ExactArgs(1),
	Run:   runBlueprintShow,
}

var blueprintApplyCmd = &cobra.Command{
	Use:   "apply <id> [id...]",
	Short: "Genera la ficha de cada módulo (en borrador) respondiendo las preguntas de sus blueprints",
	Args:  cobra.MinimumNArgs(1),
	Run:   runBlueprintApply,
}

var (
	blueprintPaquete    string
	blueprintTodos      bool
	blueprintRef        string
	blueprintModulo     string
	blueprintRutas      []string
	blueprintRespuestas []string
	blueprintPruebas    []string
)

func init() {
	blueprintAddCmd.Flags().StringVar(&blueprintRef, "ref", "", "tag, rama o commit del repositorio (recomendado: un tag de versión)")
	blueprintAddCmd.Flags().StringVar(&blueprintPaquete, "paquete", "", "importa un paquete declarado en el catalogo.yaml del repositorio")
	blueprintAddCmd.Flags().BoolVar(&blueprintTodos, "todos", false, "importa todos los blueprints del repositorio")
	blueprintApplyCmd.Flags().StringVar(&blueprintModulo, "modulo", "", "id de la ficha a generar (por defecto, el id del blueprint)")
	blueprintApplyCmd.Flags().StringArrayVar(&blueprintRutas, "ruta", nil, "carpeta del módulo en el proyecto (repetible)")
	blueprintApplyCmd.Flags().StringArrayVar(&blueprintRespuestas, "respuesta", nil, "respuesta a una pregunta, como pregunta=valor (repetible)")
	blueprintApplyCmd.Flags().StringArrayVar(&blueprintPruebas, "prueba", nil, "comando que prueba el módulo (repetible)")
	blueprintCmd.AddCommand(blueprintAddCmd, blueprintListCmd, blueprintShowCmd, blueprintApplyCmd)
	rootCmd.AddCommand(blueprintCmd)
}

func runBlueprintAdd(cmd *cobra.Command, args []string) {
	entries, problems := blueprints.Add(blueprints.AddInput{
		Root: modulesRoot(), Origen: args[0], Ref: blueprintRef,
		Blueprints: args[1:], Paquete: blueprintPaquete, Todos: blueprintTodos,
	})
	if len(problems) > 0 {
		fmt.Printf("[!] No se importó nada:\n  - %s\n", strings.Join(problems, "\n  - "))
		os.Exit(1)
	}
	var ids []string
	for _, e := range entries {
		fmt.Printf("[+] Blueprint '%s@%s' importado en %s/%s", e.ID, e.Version, blueprints.Dir, e.ID)
		if e.Commit != "" {
			fmt.Printf(" (commit %.12s)", e.Commit)
		}
		fmt.Println(".")
		ids = append(ids, e.ID)
	}
	fmt.Printf("    Siguiente paso: qdd blueprint apply %s\n", strings.Join(ids, " "))
}

func runBlueprintList(cmd *cobra.Command, args []string) {
	root := modulesRoot()
	bps, errs := blueprints.List(root)
	for _, err := range errs {
		fmt.Printf("[!] Blueprint ilegible: %v\n", err)
	}
	if len(bps) == 0 {
		fmt.Println("Sin blueprints importados. Usa: qdd blueprint add <carpeta | repo//subcarpeta> [--ref vX.Y.Z]")
		return
	}
	lock, _ := blueprints.LoadLock(root)
	for _, b := range bps {
		origen := "sin lockfile"
		if e, ok := lock.Entry(b.ID); ok {
			origen = e.Origen
		}
		fmt.Printf("- %-24s %-10s %s\n", b.ID, b.Version, origen)
	}
	apps, _ := blueprints.LoadApplications(root)
	if len(apps) > 0 {
		fmt.Println("\nMódulos generados:")
	}
	for _, a := range apps {
		fmt.Printf("  %-22s ← %s@%s (%s)\n", a.Modulo, a.Blueprint, a.Version, a.AplicadoEl)
	}
	for _, p := range blueprints.Check(root) {
		fmt.Printf("[!] %s: %s\n", p.Regla, p.Descripcion)
	}
}

func runBlueprintShow(cmd *cobra.Command, args []string) {
	b, err := blueprints.Find(modulesRoot(), args[0])
	if err != nil {
		fmt.Printf("[!] %v\n", err)
		os.Exit(1)
	}
	fmt.Print(blueprints.Describe(b))
}

func runBlueprintApply(cmd *cobra.Command, args []string) {
	respuestas, err := parseRespuestas(blueprintRespuestas)
	if err != nil {
		fmt.Printf("[!] %v\n", err)
		os.Exit(1)
	}
	if len(args) > 1 && (blueprintModulo != "" || len(blueprintRutas) > 0 || len(blueprintPruebas) > 0) {
		fmt.Println("[!] --modulo, --ruta y --prueba son de un solo módulo: aplica ese blueprint por separado.")
		os.Exit(1)
	}
	if problems := huerfanas(args, respuestas); len(problems) > 0 {
		fmt.Printf("[!] %s\n", strings.Join(problems, "\n[!] "))
		os.Exit(1)
	}
	root := modulesRoot()
	interactive := isTerminal(os.Stdin)
	reader := bufio.NewReader(os.Stdin)
	for _, id := range args {
		in := blueprints.ApplyInput{Root: root, Blueprint: id, Modulo: blueprintModulo, Rutas: blueprintRutas, Respuestas: respuestasDe(id, respuestas, len(args) > 1), Pruebas: blueprintPruebas}
		res, problems := applyAsking(in, reader, os.Stdout, interactive)
		if len(problems) > 0 {
			fmt.Printf("[!] No se pudo aplicar el blueprint '%s':\n  - %s\n", id, strings.Join(problems, "\n  - "))
			os.Exit(1)
		}
		printApplyResult(root, res)
	}
}

// huerfanas rechaza, al aplicar varios blueprints, las respuestas que no dicen
// a cuál pertenecen o que nombran uno que no se está aplicando.
func huerfanas(ids []string, respuestas map[string]string) []string {
	if len(ids) < 2 {
		return nil
	}
	var problems []string
	for k := range respuestas {
		bp, _, ok := strings.Cut(k, ".")
		if ok && inArgs(ids, bp) {
			continue
		}
		problems = append(problems, fmt.Sprintf("--respuesta %s: con varios blueprints usa blueprint.pregunta=valor (p. ej. %s.%s)", k, ids[0], strings.TrimPrefix(k, bp+".")))
	}
	return problems
}

func inArgs(ids []string, id string) bool {
	for _, it := range ids {
		if it == id {
			return true
		}
	}
	return false
}

// respuestasDe separa las respuestas de cada blueprint cuando se aplican
// varios a la vez: '--respuesta auth.identificador=rut' es solo para 'auth'.
// Con un solo blueprint, también vale la forma corta 'identificador=rut'.
func respuestasDe(id string, respuestas map[string]string, varios bool) map[string]string {
	out := map[string]string{}
	for k, v := range respuestas {
		bp, pregunta, ok := strings.Cut(k, ".")
		if ok && bp == id {
			out[pregunta] = v
			continue
		}
		if !ok && !varios {
			out[k] = v
		}
	}
	return out
}

// applyAsking aplica el blueprint y, mientras falten decisiones, se las
// pregunta al humano (si hay terminal). Sin terminal, las lista y falla:
// las decisiones de diseño no se toman por defecto.
func applyAsking(in blueprints.ApplyInput, reader *bufio.Reader, out io.Writer, interactive bool) (blueprints.ApplyResult, []string) {
	for {
		res, problems := blueprints.Apply(in)
		if len(problems) > 0 || len(res.Pendientes) == 0 {
			return res, problems
		}
		if !interactive {
			return res, pendingProblems(res.Pendientes)
		}
		p := res.Pendientes[0]
		valor, err := ask(reader, out, p)
		if err != nil {
			return res, pendingProblems(res.Pendientes)
		}
		in.Respuestas = setAnswer(in.Respuestas, p.ID, valor)
	}
}

func pendingProblems(pendientes []blueprints.Pregunta) []string {
	var out []string
	for _, p := range pendientes {
		out = append(out, fmt.Sprintf("falta --respuesta %s=<%s>: %s", p.ID, strings.Join(p.Valores(), "|"), p.Pregunta))
	}
	return out
}

func ask(reader *bufio.Reader, out io.Writer, p blueprints.Pregunta) (string, error) {
	fmt.Fprint(out, blueprints.DescribePregunta(p))
	for {
		fmt.Fprintf(out, "Elige 1-%d o escribe el valor", len(p.Opciones))
		if p.PorDefecto != "" {
			fmt.Fprintf(out, " [Enter = %s]", p.PorDefecto)
		}
		fmt.Fprint(out, ": ")
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" && p.PorDefecto != "" && err == nil {
			return p.PorDefecto, nil
		}
		if valor, ok := pickOption(p, line); ok {
			return valor, nil
		}
		if err != nil {
			return "", fmt.Errorf("sin respuesta para '%s'", p.ID)
		}
		fmt.Fprintf(out, "'%s' no es una opción.\n", line)
	}
}

func pickOption(p blueprints.Pregunta, input string) (string, bool) {
	valores := p.Valores()
	if n, err := strconv.Atoi(input); err == nil && n >= 1 && n <= len(valores) {
		return valores[n-1], true
	}
	for _, v := range valores {
		if v == input {
			return v, true
		}
	}
	return "", false
}

func setAnswer(m map[string]string, k, v string) map[string]string {
	if m == nil {
		m = map[string]string{}
	}
	m[k] = v
	return m
}

func parseRespuestas(items []string) (map[string]string, error) {
	out := map[string]string{}
	for _, it := range items {
		k, v, ok := strings.Cut(it, "=")
		if !ok || strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			return nil, fmt.Errorf("respuesta '%s': usa el formato pregunta=valor", it)
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out, nil
}

// isTerminal detecta una terminal sin dependencias extra: un dispositivo de
// caracteres que no sea /dev/null (que también lo es).
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(info, null)
}

func printApplyResult(root string, res blueprints.ApplyResult) {
	fmt.Printf("[+] Ficha generada en %s como BORRADOR.\n", relPathOrSelf(root, res.Ficha))
	fmt.Printf("    Escenarios de conformidad: %s\n", res.Conformidad)
	fmt.Printf("    Decisiones del proyecto:   %s\n", relPathOrSelf(root, res.Aplicacion))
	for _, e := range res.Esqueletos {
		fmt.Printf("    Esqueleto creado:          %s/README.md\n", e)
	}
	fmt.Println("    Revisa la ficha y apruébala con: qdd modules approve <id>")
}

func relPathOrSelf(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return filepath.ToSlash(rel)
}
