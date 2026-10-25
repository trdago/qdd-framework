package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// testCommand es un comando de pruebas que QDD ejecuta como parte del gate.
type testCommand struct {
	Label string // lo que se muestra al usuario, p. ej. "go test ./..."
	Dir   string // carpeta donde se ejecuta
	Args  []string
}

type qddTestConfig struct {
	TestCommands []string `yaml:"test_commands"`
}

// resolveTestCommands decide qué pruebas corre el gate. Si el proyecto declara
// test_commands en .qdd/config.yaml, esos mandan (es la forma de soportar
// cualquier stack). Si no, se detectan TODOS los stacks presentes: un proyecto
// con package.json y go.mod corre ambas suites, no solo una.
func resolveTestCommands(cwd string) []testCommand {
	declared := declaredTestCommands(cwd)
	if len(declared) > 0 {
		return declared
	}
	return detectTestCommands(cwd)
}

func declaredTestCommands(cwd string) []testCommand {
	data, err := os.ReadFile(filepath.Join(cwd, ".qdd", "config.yaml"))
	if err != nil {
		return nil
	}
	var cfg qddTestConfig
	if yaml.Unmarshal(data, &cfg) != nil {
		return nil
	}
	var cmds []testCommand
	for _, c := range cfg.TestCommands {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		cmds = append(cmds, testCommand{Label: c, Dir: cwd, Args: []string{"sh", "-c", c}})
	}
	return cmds
}

// detectTestCommands busca las suites en la raíz y, si ahí no hay ninguna, en
// cada subcarpeta de primer nivel (monorepos: backend/, frontend/...).
func detectTestCommands(cwd string) []testCommand {
	cmds := detectTestCommandsIn(cwd)
	if len(cmds) > 0 {
		return cmds
	}
	for _, sub := range subcarpetasDeCodigo(cwd) {
		for _, tc := range detectTestCommandsIn(filepath.Join(cwd, sub)) {
			tc.Label += " (" + sub + ")"
			cmds = append(cmds, tc)
		}
	}
	return cmds
}

var carpetasSinPruebasPropias = map[string]bool{
	"node_modules": true, "vendor": true, "dist": true, "build": true,
	"target": true, "venv": true, "__pycache__": true,
}

func subcarpetasDeCodigo(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var subs []string
	for _, e := range entries {
		if e.IsDir() && !strings.HasPrefix(e.Name(), ".") && !carpetasSinPruebasPropias[e.Name()] {
			subs = append(subs, e.Name())
		}
	}
	return subs
}

func detectTestCommandsIn(cwd string) []testCommand {
	var cmds []testCommand
	if hasNpmTestScript(cwd) {
		cmds = append(cmds, testCommand{Label: "npm run test", Dir: cwd, Args: []string{"npm", "run", "test"}})
	}
	if goDir, ok := goModuleDir(cwd); ok {
		cmds = append(cmds, testCommand{Label: "go test ./...", Dir: goDir, Args: []string{"go", "test", "./..."}})
	}
	if fileExists(filepath.Join(cwd, "pubspec.yaml")) && hayPruebasDart(filepath.Join(cwd, "test")) {
		cmds = append(cmds, testCommand{Label: "flutter test", Dir: cwd, Args: []string{"flutter", "test"}})
	}
	if hasPythonTests(cwd) {
		py := pythonBin(cwd)
		cmds = append(cmds, testCommand{Label: py + " -m pytest", Dir: cwd, Args: []string{py, "-m", "pytest", "-q"}})
	}
	return cmds
}

// hasNpmTestScript ignora el script por defecto de `npm init`, que siempre falla.
func hasNpmTestScript(cwd string) bool {
	data, err := os.ReadFile(filepath.Join(cwd, "package.json"))
	if err != nil {
		return false
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return false
	}
	script := strings.TrimSpace(pkg.Scripts["test"])
	return script != "" && !strings.Contains(script, "no test specified")
}

// goModuleDir soporta go.mod en la raíz o en cli/ (layout del propio QDD).
func goModuleDir(cwd string) (string, bool) {
	for _, dir := range []string{cwd, filepath.Join(cwd, "cli")} {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir, true
		}
	}
	return "", false
}

// hasPythonTests reconoce una suite de pytest por su configuración, por pytest
// declarado en requirements*.txt, o por archivos test_*.py en tests/ (FND-047).
func hasPythonTests(cwd string) bool {
	if !pathExists(filepath.Join(cwd, "tests")) {
		return false
	}
	hasConfig := fileExists(filepath.Join(cwd, "pytest.ini")) || fileExists(filepath.Join(cwd, "pyproject.toml")) || fileExists(filepath.Join(cwd, "setup.cfg"))
	return hasConfig || requirementsMencionanPytest(cwd) || hayArchivosDePytest(filepath.Join(cwd, "tests"))
}

func requirementsMencionanPytest(cwd string) bool {
	archivos, _ := filepath.Glob(filepath.Join(cwd, "requirements*.txt"))
	for _, a := range archivos {
		data, err := os.ReadFile(a)
		if err == nil && strings.Contains(strings.ToLower(string(data)), "pytest") {
			return true
		}
	}
	return false
}

func hayArchivosDePytest(testsDir string) bool {
	prefijo, _ := filepath.Glob(filepath.Join(testsDir, "test_*.py"))
	sufijo, _ := filepath.Glob(filepath.Join(testsDir, "*_test.py"))
	return len(prefijo)+len(sufijo) > 0
}

// pythonBin prefiere el entorno virtual del proyecto y luego python3: muchas
// distribuciones ya no instalan un ejecutable llamado "python".
func pythonBin(cwd string) string {
	for _, venv := range []string{".venv", "venv"} {
		rel := filepath.Join(venv, "bin", "python")
		if fileExists(filepath.Join(cwd, rel)) {
			return "./" + rel
		}
	}
	for _, c := range []string{"python3", "python"} {
		if _, err := exec.LookPath(c); err == nil {
			return c
		}
	}
	return "python3"
}

// runTestCommand ejecuta un comando de pruebas. Un fallo de compilación es un
// fallo de pruebas: antes se trataba como "ignorable" y el gate aprobaba.
// Devuelve también la salida, para explicar el fallo (pistaDeFallo).
func runTestCommand(tc testCommand) (string, error) {
	cmd := exec.Command(tc.Args[0], tc.Args[1:]...)
	cmd.Dir = tc.Dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return "", nil
	}
	fmt.Fprintf(os.Stderr, "TEST FAILURE LOG (%s):\n%s\n", tc.Label, tailLines(string(out), 60))
	return string(out) + "\n" + err.Error(), err
}

// señalesDeDependencias delatan que las pruebas no llegaron a correr porque
// falta instalar algo (lo típico en un clon recién hecho).
var señalesDeDependencias = []string{
	"ModuleNotFoundError", "No module named", "Cannot find module", "ERR_MODULE_NOT_FOUND",
	": not found", "command not found", "executable file not found", "cannot find package",
}

// pistaDeFallo agrega qué hacer cuando la suite falla por dependencias
// faltantes y no por una prueba (FND-049).
func pistaDeFallo(salida string) string {
	for _, s := range señalesDeDependencias {
		if strings.Contains(salida, s) {
			return " Parece que faltan dependencias o herramientas por instalar (p. ej. npm install, pip install -r requirements.txt, flutter pub get): instálalas y vuelve a ejecutar qdd audit."
		}
	}
	return ""
}

// motivoSinPruebas explica, según el stack presente, qué le falta al proyecto
// para que el gate pueda correr sus pruebas (FND-049).
func motivoSinPruebas(cwd string) string {
	if fileExists(filepath.Join(cwd, "package.json")) {
		return "package.json no tiene un script \"test\" (p. ej. \"test\": \"vitest run\")."
	}
	if fileExists(filepath.Join(cwd, "pubspec.yaml")) {
		return "no hay archivos *_test.dart en test/."
	}
	if detectaPython(cwd) {
		return "no hay pruebas de pytest (tests/test_*.py)."
	}
	return "no se reconoció ninguna suite (npm, go, flutter, pytest)."
}

func detectaPython(cwd string) bool {
	for _, n := range []string{"requirements.txt", "pyproject.toml", "setup.py", "Pipfile"} {
		if fileExists(filepath.Join(cwd, n)) {
			return true
		}
	}
	return false
}

// hayPruebasDart busca archivos *_test.dart en cualquier nivel de test/.
func hayPruebasDart(testDir string) bool {
	encontrado := false
	filepath.WalkDir(testDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(d.Name(), "_test.dart") {
			encontrado = true
			return filepath.SkipAll
		}
		return nil
	})
	return encontrado
}

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}

// RunCoverageCheck evalúa que las pruebas del proyecto se ejecuten y pasen
// (Anti-Regresión). Si no hay forma de correr pruebas, el gate NO aprueba en
// silencio: lo informa como violación.
func RunCoverageCheck(cwd string) []Violation {
	var violations []Violation

	cmds := resolveTestCommands(cwd)
	if len(cmds) == 0 {
		violations = append(violations, Violation{
			Category:    "COVERAGE",
			RuleID:      "QA-00-NO-TEST-COMMAND",
			Description: "El gate no puede verificar nada porque " + motivoSinPruebas(cwd) + " Agrega pruebas o declara cómo correrlas en .qdd/config.yaml (test_commands: [\"...\"]).",
		})
	}

	for _, tc := range cmds {
		fmt.Fprintf(os.Stderr, "[QA] Ejecutando pruebas: %s (en %s)\n", tc.Label, tc.Dir)
		if salida, err := runTestCommand(tc); err != nil {
			fmt.Fprintf(os.Stderr, "[QA] FALLÓ: %s\n", tc.Label)
			violations = append(violations, Violation{
				Category:    "COVERAGE",
				RuleID:      "QA-01-ALL-TESTS-PASS",
				Description: fmt.Sprintf("Falló la suite de pruebas '%s': %v.%s", tc.Label, err, pistaDeFallo(salida)),
			})
			continue
		}
		fmt.Fprintf(os.Stderr, "[QA] OK: %s\n", tc.Label)
	}

	// QA-02-E2E-TESTS: Exigir entorno de pruebas End-to-End
	hasE2E := false
	e2ePaths := []string{"e2e", "tests/e2e", "cypress", "playwright"}
	for _, p := range e2ePaths {
		if _, err := os.Stat(cwd + "/" + p); err == nil {
			hasE2E = true
			break
		}
	}

	if !hasE2E {
		violations = append(violations, Violation{
			Category:    "COVERAGE",
			RuleID:      "QA-02-E2E-TESTS",
			Description: "No se detectó un entorno de pruebas E2E (carpetas: e2e, tests/e2e, cypress, playwright). QDD exige validación de extremo a extremo.",
		})
	}

	return violations
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// TestCommandLabels devuelve los comandos de prueba que usaría el gate, para
// mostrarlos a una IA (p. ej. en qdd_project_context) sin ejecutarlos.
func TestCommandLabels(cwd string) []string {
	var labels []string
	for _, tc := range resolveTestCommands(cwd) {
		labels = append(labels, tc.Label)
	}
	return labels
}
