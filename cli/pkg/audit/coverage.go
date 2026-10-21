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

func detectTestCommands(cwd string) []testCommand {
	var cmds []testCommand
	if hasNpmTestScript(cwd) {
		cmds = append(cmds, testCommand{Label: "npm run test", Dir: cwd, Args: []string{"npm", "run", "test"}})
	}
	if goDir, ok := goModuleDir(cwd); ok {
		cmds = append(cmds, testCommand{Label: "go test ./...", Dir: goDir, Args: []string{"go", "test", "./..."}})
	}
	if fileExists(filepath.Join(cwd, "pubspec.yaml")) && pathExists(filepath.Join(cwd, "test")) {
		cmds = append(cmds, testCommand{Label: "flutter test", Dir: cwd, Args: []string{"flutter", "test"}})
	}
	if hasPythonTests(cwd) {
		cmds = append(cmds, testCommand{Label: "python -m pytest", Dir: cwd, Args: []string{"python", "-m", "pytest", "-q"}})
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

func hasPythonTests(cwd string) bool {
	hasConfig := fileExists(filepath.Join(cwd, "pytest.ini")) || fileExists(filepath.Join(cwd, "pyproject.toml")) || fileExists(filepath.Join(cwd, "setup.cfg"))
	return hasConfig && pathExists(filepath.Join(cwd, "tests"))
}

// runTestCommand ejecuta un comando de pruebas. Un fallo de compilación es un
// fallo de pruebas: antes se trataba como "ignorable" y el gate aprobaba.
func runTestCommand(tc testCommand) error {
	cmd := exec.Command(tc.Args[0], tc.Args[1:]...)
	cmd.Dir = tc.Dir
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	fmt.Fprintf(os.Stderr, "TEST FAILURE LOG (%s):\n%s\n", tc.Label, tailLines(string(out), 60))
	return err
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
			Description: "No se detectó cómo ejecutar las pruebas del proyecto, así que el gate no puede verificar nada. Declara los comandos en .qdd/config.yaml (test_commands: [\"...\"]).",
		})
	}

	for _, tc := range cmds {
		fmt.Fprintf(os.Stderr, "[QA] Ejecutando pruebas: %s (en %s)\n", tc.Label, tc.Dir)
		if err := runTestCommand(tc); err != nil {
			fmt.Fprintf(os.Stderr, "[QA] FALLÓ: %s\n", tc.Label)
			violations = append(violations, Violation{
				Category:    "COVERAGE",
				RuleID:      "QA-01-ALL-TESTS-PASS",
				Description: fmt.Sprintf("Falló la suite de pruebas '%s': %v", tc.Label, err),
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
