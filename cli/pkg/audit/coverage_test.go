package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func labels(cmds []testCommand) string {
	var out []string
	for _, c := range cmds {
		out = append(out, c.Label)
	}
	return strings.Join(out, " | ")
}

// Regresión: en proyectos sin package.json ni go.mod en la raíz (Flutter,
// firmware) el gate corría `go test`, recibía "cannot find main module", lo
// trataba como ignorable y aprobaba sin ejecutar ninguna prueba.
func TestRunCoverageCheck_NoTestCommandIsAViolation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "firmware", "main.ino"), "void setup(){}")

	violations := RunCoverageCheck(dir)
	if !hasRule(violations, "QA-00-NO-TEST-COMMAND") {
		t.Fatalf("se esperaba QA-00-NO-TEST-COMMAND, se obtuvo %v", violations)
	}
}

// Regresión: con package.json y go.mod a la vez solo se corría jest.
func TestDetectTestCommands_RunsEveryStackPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"test":"jest"}}`)
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "pubspec.yaml"), "name: app\n")
	writeFile(t, filepath.Join(dir, "test", "widget_test.dart"), "")

	got := labels(detectTestCommands(dir))
	want := "npm run test | go test ./... | flutter test"
	if got != want {
		t.Errorf("detectTestCommands = %q, want %q", got, want)
	}
}

func TestDetectTestCommands_IgnoresNpmInitPlaceholder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"test":"echo \"Error: no test specified\" && exit 1"}}`)

	if cmds := detectTestCommands(dir); len(cmds) != 0 {
		t.Errorf("el placeholder de npm init no es una suite de pruebas: %q", labels(cmds))
	}
}

func TestResolveTestCommands_DeclaredCommandsWin(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, ".qdd", "config.yaml"), "test_commands:\n  - make test\n  - npm run test:unit\n")

	got := labels(resolveTestCommands(dir))
	if got != "make test | npm run test:unit" {
		t.Errorf("resolveTestCommands = %q", got)
	}
}

func TestRunCoverageCheck_FailingDeclaredCommandIsAViolation(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "config.yaml"), "test_commands:\n  - \"true\"\n  - \"exit 3\"\n")

	violations := RunCoverageCheck(dir)
	count := 0
	for _, v := range violations {
		if v.RuleID == "QA-01-ALL-TESTS-PASS" {
			count++
			if !strings.Contains(v.Description, "exit 3") {
				t.Errorf("la violación debe nombrar el comando que falló: %q", v.Description)
			}
		}
	}
	if count != 1 {
		t.Errorf("se esperaba exactamente 1 QA-01 (solo falla 'exit 3'), hubo %d: %v", count, violations)
	}
}

func hasRule(violations []Violation, rule string) bool {
	for _, v := range violations {
		if v.RuleID == rule {
			return true
		}
	}
	return false
}

// El servidor MCP habla JSON-RPC por stdout: el motor de auditoría no debe
// escribir nada en stdout o corrompe el protocolo.
func TestRunCoverageCheck_WritesNothingToStdout(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "config.yaml"), "test_commands:\n  - \"true\"\n")

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stdout
	os.Stdout = w
	RunCoverageCheck(dir)
	os.Stdout = orig
	w.Close()

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	if n > 0 {
		t.Errorf("RunCoverageCheck escribió en stdout (rompe MCP): %q", string(buf[:n]))
	}
}
