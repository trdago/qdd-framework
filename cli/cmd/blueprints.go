package cmd

import (
	"bufio"
	"encoding/json"
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
de conformidad. Vive en el repositorio de tu organización (privado si quieres).

QDD administra todo su ciclo de vida (RFC 0004):
  - El repositorio se clona una vez por máquina en la carpeta central de QDD
    (~/.local/share/qdd/blueprints); el proyecto solo versiona su lockfile.
  - Los proyectos de la misma organización (mismo dueño en su remote git) ven
    sus blueprints sin configurar nada.
  - Solo se importan versiones publicadas (tags vX.Y.Z); la versión de cada
    cambio la calcula QDD, nunca una persona.

  qdd blueprint add git@github.com:org/qdd-blueprints.git auth   # la primera vez
  qdd blueprint add auditoria                 # después, basta con nombrarlos
  qdd blueprint apply auth --ruta internal/auth
  qdd blueprint sync                          # trae lo publicado y dice qué actualizar
  qdd blueprint contribute auth               # prepara un aporte en la carpeta central
  qdd blueprint propose auth -m "…"           # QDD versiona, publica la rama y abre el PR
  qdd blueprint release                       # tras el merge, etiqueta la nueva versión

Un proyecto puede usar además blueprints de otras fuentes (p. ej. tu propio
repositorio público): impórtalos con su URL y el lockfile recuerda de dónde
salió cada uno. Un id pertenece a una sola fuente; sync, add <id> y contribute
usan siempre la del lockfile.

  qdd blueprint add git@github.com:yo/mis-blueprints.git mi-estandar
  qdd blueprint contribute nuevo --origen git@github.com:yo/mis-blueprints.git
  qdd blueprint release --origen git@github.com:yo/mis-blueprints.git`,
	Run: runBlueprintList,
}

var blueprintAddCmd = &cobra.Command{
	Use:   "add [origen] [blueprint...]",
	Short: "Importa blueprints del repositorio de tu organización (uno, varios, un --paquete o --todos)",
	Long: `Importa blueprints de una versión publicada del repositorio de tu organización.
La primera vez indica el repositorio (privado si quieres; se usan tus
credenciales git): QDD lo recuerda en .qdd/config.yaml y en el registro central,
para este y los demás proyectos de la organización.

  qdd blueprint add git@github.com:org/qdd-blueprints.git auth
  qdd blueprint add auditoria notificaciones      # usa el repositorio registrado
  qdd blueprint add --paquete base
  qdd blueprint add --todos                        # todo, en su última versión

Sin origen, cada blueprint ya importado se actualiza desde su propia fuente (la
que dice el lockfile) y los nuevos salen del repositorio de la organización.
Un id ya importado desde otra fuente no se reemplaza salvo con --reemplazar.

Sin --ref se importa la última versión publicada (tag vX.Y.Z). Si alguno es
inválido no se importa ninguno. Los paquetes y las carpetas se declaran en
catalogo.yaml en la raíz del repositorio.`,
	Args: cobra.ArbitraryArgs,
	Run:  runBlueprintAdd,
}

var blueprintSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Trae lo publicado en cada fuente del proyecto y dice qué blueprints actualizar",
	Args:  cobra.NoArgs,
	Run:   runBlueprintSync,
}

var blueprintContributeCmd = &cobra.Command{
	Use:   "contribute <id>",
	Short: "Prepara un aporte a un blueprint (nuevo o existente) en la carpeta central de QDD",
	Args:  cobra.ExactArgs(1),
	Run:   runBlueprintContribute,
}

var blueprintProposeCmd = &cobra.Command{
	Use:   "propose <id> -m <descripción>",
	Short: "Valida el aporte, fija su versión semver, publica la rama y abre el pull request",
	Args:  cobra.ExactArgs(1),
	Run:   runBlueprintPropose,
}

var blueprintReleaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Publica (etiqueta vX.Y.Z) lo integrado en el repositorio de la organización",
	Args:  cobra.NoArgs,
	Run:   runBlueprintRelease,
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
	blueprintMensaje    string
	blueprintOrigen     string
	blueprintReemplazar bool
	blueprintModulo     string
	blueprintRutas      []string
	blueprintRespuestas []string
	blueprintPruebas    []string
)

func init() {
	blueprintAddCmd.Flags().StringVar(&blueprintRef, "ref", "", "versión publicada (tag vX.Y.Z); por defecto, la última")
	blueprintProposeCmd.Flags().StringVarP(&blueprintMensaje, "mensaje", "m", "", "qué cambia y por qué (va al commit y al pull request)")
	blueprintAddCmd.Flags().StringVar(&blueprintPaquete, "paquete", "", "importa un paquete declarado en el catalogo.yaml del repositorio")
	blueprintAddCmd.Flags().BoolVar(&blueprintTodos, "todos", false, "importa todos los blueprints del repositorio")
	blueprintAddCmd.Flags().BoolVar(&blueprintReemplazar, "reemplazar", false, "reemplaza un blueprint importado desde otra fuente por el de este origen")
	for _, c := range []*cobra.Command{blueprintContributeCmd, blueprintProposeCmd, blueprintReleaseCmd} {
		c.Flags().StringVar(&blueprintOrigen, "origen", "", "fuente (repositorio) de un blueprint nuevo; por defecto, la de la organización")
	}
	blueprintApplyCmd.Flags().StringVar(&blueprintModulo, "modulo", "", "id de la ficha a generar (por defecto, el id del blueprint)")
	blueprintApplyCmd.Flags().StringArrayVar(&blueprintRutas, "ruta", nil, "carpeta del módulo en el proyecto (repetible)")
	blueprintApplyCmd.Flags().StringArrayVar(&blueprintRespuestas, "respuesta", nil, "respuesta a una pregunta, como pregunta=valor (repetible)")
	blueprintApplyCmd.Flags().StringArrayVar(&blueprintPruebas, "prueba", nil, "comando que prueba el módulo (repetible)")
	blueprintCmd.AddCommand(blueprintAddCmd, blueprintListCmd, blueprintShowCmd, blueprintApplyCmd,
		blueprintSyncCmd, blueprintContributeCmd, blueprintProposeCmd, blueprintReleaseCmd)
	rootCmd.AddCommand(blueprintCmd)
}

func runBlueprintAdd(cmd *cobra.Command, args []string) {
	origen, ids := splitAddArgs(args)
	res, problems := blueprints.AddManaged(blueprints.AddInput{
		Root: modulesRoot(), Origen: origen, Ref: blueprintRef,
		Blueprints: ids, Paquete: blueprintPaquete, Todos: blueprintTodos,
		Reemplazar: blueprintReemplazar,
	})
	if len(problems) > 0 && len(res.Entries) == 0 {
		fmt.Printf("[!] No se importó nada:\n  - %s\n", strings.Join(problems, "\n  - "))
		os.Exit(1)
	}
	printAddResult(res, problems)
}

// splitAddArgs separa el origen (si el primer argumento lo es) de los ids:
// 'qdd blueprint add auth auditoria' usa el repositorio registrado.
func splitAddArgs(args []string) (string, []string) {
	if len(args) > 0 && blueprints.EsOrigen(args[0]) {
		return args[0], args[1:]
	}
	return "", args
}

func printAddResult(res blueprints.AddResult, warnings []string) {
	var ids []string
	for _, e := range res.Entries {
		fmt.Printf("[+] Blueprint '%s@%s' importado desde %s (commit %.12s).\n", e.ID, e.Version, e.Ref, e.Commit)
		ids = append(ids, e.ID)
	}
	if res.Guardado {
		fmt.Printf("[+] Repositorio de la organización registrado en .qdd/config.yaml y en el registro central: %s. Desde ahora basta con: qdd blueprint add <id>...\n", res.Registro.Origen)
	}
	for _, w := range warnings {
		fmt.Printf("[!] %s\n", w)
	}
	fmt.Printf("    Siguiente paso: qdd blueprint apply %s\n", strings.Join(ids, " "))
}

func runBlueprintList(cmd *cobra.Command, args []string) {
	root := modulesRoot()
	if reg, _ := blueprints.LoadRegistro(root); reg.Origen != "" {
		fmt.Printf("Repositorio de la organización: %s%s\n", reg.Origen, origenDelRegistro(reg))
	}
	if home, err := blueprints.Home(); err == nil {
		fmt.Printf("Carpeta central de QDD: %s\n\n", filepath.Join(home, "blueprints"))
	}
	bps, errs := blueprints.List(root)
	for _, err := range errs {
		fmt.Printf("[!] Blueprint ilegible: %v\n", err)
	}
	if len(bps) == 0 {
		fmt.Println("Sin blueprints importados. Usa: qdd blueprint add [repositorio] <id>... (o --paquete / --todos)")
		return
	}
	lock, _ := blueprints.LoadLock(root)
	for _, b := range bps {
		e, _ := lock.Entry(b.ID)
		fmt.Printf("- %-24s %-10s %s %s\n", b.ID, b.Version, e.Ref, e.Origen)
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

func runBlueprintSync(cmd *cobra.Command, args []string) {
	res, err := blueprints.Sincronizar(modulesRoot())
	if err != nil && len(res) == 0 {
		fmt.Printf("[!] %v\n", err)
		os.Exit(1)
	}
	var ids []string
	fallo := err != nil
	for _, s := range res {
		if s.Problema != "" {
			fmt.Printf("[!] %s: %s\n", s.Origen, s.Problema)
			fallo = true
			continue
		}
		fmt.Printf("[+] %s sincronizado: última versión publicada %s.\n", s.Origen, s.Ref)
		for _, n := range s.Novedades {
			fmt.Printf("[!] %s: %s → %s\n", n.ID, n.Actual, n.Disponible)
			ids = append(ids, n.ID)
		}
	}
	if err != nil {
		fmt.Printf("[!] %v\n", err)
	}
	if len(ids) > 0 {
		fmt.Printf("    Actualiza: qdd blueprint add %s (después, qdd blueprint apply para los módulos que los usan).\n", strings.Join(ids, " "))
	}
	if len(ids) == 0 && !fallo {
		fmt.Println("    Los blueprints del proyecto están al día.")
	}
	if fallo {
		os.Exit(1)
	}
}

func runBlueprintContribute(cmd *cobra.Command, args []string) {
	ap, err := blueprints.Aportar(modulesRoot(), args[0], blueprintOrigen)
	if err != nil {
		fmt.Printf("[!] %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[+] Aporte listo en la rama %s de %s.\n", ap.Rama, ap.Repo)
	fmt.Printf("    Edita el blueprint aquí: %s\n", ap.Carpeta)
	fmt.Printf("    Después: qdd blueprint propose %s%s -m \"qué cambia y por qué\" (QDD fija la versión y abre el PR).\n", args[0], conOrigen())
}

func runBlueprintPropose(cmd *cobra.Command, args []string) {
	p, err := blueprints.Proponer(modulesRoot(), args[0], blueprintMensaje, blueprintOrigen)
	if err != nil {
		fmt.Printf("[!] %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[+] %s v%s (cambio %s) publicado en la rama %s (commit %.12s).\n", args[0], p.Version, p.Cambio, p.Rama, p.Commit)
	for _, r := range p.Razones {
		fmt.Printf("    - %s\n", r)
	}
	if p.Aviso != "" {
		fmt.Printf("[!] %s\n", p.Aviso)
		return
	}
	fmt.Printf("[+] Pull request: %s\n    Tras el merge: qdd blueprint release%s\n", p.PR, conOrigen())
}

func runBlueprintRelease(cmd *cobra.Command, args []string) {
	pub, err := blueprints.Publicar(modulesRoot(), blueprintOrigen)
	if err != nil {
		fmt.Printf("[!] %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[+] Publicada la versión %s (cambio %s).\n", pub.Tag, pub.Cambio)
	for _, c := range pub.Cambios {
		fmt.Printf("    - %s: %s → %s\n", c.ID, c.Actual, c.Disponible)
	}
	fmt.Println("    Cada proyecto la recibe con: qdd blueprint sync")
}

// migrationNotice migra, una sola vez, los blueprints del esquema anterior
// (copias en .qdd/blueprints/) a la carpeta central y devuelve qué pasó y qué
// queda por hacer, escrito para que una IA lo termine. Vacío si no hubo nada.
func migrationNotice(root string) string {
	m, err := blueprints.Migrar(root)
	if err != nil {
		return fmt.Sprintf("⚠️ QDD no pudo migrar los blueprints de este proyecto a la carpeta central: %v\n", err)
	}
	if !m.Hizo() && len(m.Pendientes) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("## Migración de blueprints (RFC 0004)\n")
	if m.Hizo() {
		home, _ := blueprints.Home()
		fmt.Fprintf(&b, "QDD movió %s a su carpeta central (%s): el proyecto ya solo versiona %s. Incluye en el próximo commit la eliminación de %s/, el lockfile actualizado y .qdd/state.json.\n", strings.Join(m.Migrados, ", "), filepath.Join(home, "blueprints"), blueprints.LockFile, blueprints.Dir)
	}
	if err := sellarVersion(root, Version); m.Hizo() && err != nil {
		fmt.Fprintf(&b, "⚠️ No se pudo fijar la versión mínima de QDD en .qdd/state.json: %v\n", err)
	}
	for _, p := range m.Pendientes {
		fmt.Fprintf(&b, "- BP-05-LEGACY (%s), termínalo tú: %s\n", p.ID, p.Motivo)
	}
	return b.String()
}

// conOrigen repite --origen en los siguientes pasos que sugiere la CLI.
func conOrigen() string {
	if blueprintOrigen == "" {
		return ""
	}
	return " --origen " + blueprintOrigen
}

func origenDelRegistro(reg blueprints.Registro) string {
	if reg.Central {
		return " (registro central de la organización)"
	}
	return " (.qdd/config.yaml)"
}

// sellarVersion exige en .qdd/state.json al menos esta versión de QDD: tras
// migrar los blueprints a la carpeta central, un QDD anterior (otra máquina,
// un CI fijado) se detiene con un error claro en vez de trabajar sin ver los
// blueprints. Nunca baja la versión exigida; un binario de desarrollo no sella.
func sellarVersion(root, version string) error {
	if _, ok := blueprintsVersion(version); !ok {
		return nil
	}
	path := filepath.Join(root, ".qdd", "state.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	state := map[string]interface{}{}
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("%s: JSON inválido: %w", path, err)
	}
	actual, _ := state["version"].(string)
	if actual != "" && !isOlder(actual, version) {
		return nil
	}
	state["version"] = version
	out, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(out, '\n'), 0644)
}

// blueprintsVersion acepta solo versiones publicadas (vX.Y.Z).
func blueprintsVersion(v string) (string, bool) {
	var a, b, c int
	n, err := fmt.Sscanf(v, "v%d.%d.%d", &a, &b, &c)
	return v, err == nil && n == 3 && v == fmt.Sprintf("v%d.%d.%d", a, b, c)
}
