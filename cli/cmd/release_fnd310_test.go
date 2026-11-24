package cmd

import (
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
)

// Rule: Prohibido usar 'else'

// contrato: reporte-y-release#d0fd5c69
// FND-310: qdd_release creaba el tag sin correr el gate y callaba si la compilación o el tag fallaban.
func TestRelease_NoEtiquetaLoQueElGateBloquea_FND310(t *testing.T) {
	dir := repoDeRelease(t)
	err := publicarRelease(dir, "v1.0.0", func(string) []audit.Violation {
		return []audit.Violation{{RuleID: "QA-01-TEST-FAIL", Description: "una prueba falla", File: "a_test.go"}}
	}, sinCompilar)
	if err == nil || !strings.Contains(err.Error(), "QA-01-TEST-FAIL") {
		t.Fatalf("el gate bloquea y la release siguió: %v", err)
	}
	if tags := tagsDe(t, dir); tags != "" {
		t.Errorf("no debía crear el tag: %s", tags)
	}
}

func TestRelease_SiNoCompilaNoEtiqueta_FND310(t *testing.T) {
	dir := repoDeRelease(t)
	err := publicarRelease(dir, "v1.0.0", sinViolaciones, func(string) error { return errors.New("no compila") })
	if err == nil || !strings.Contains(err.Error(), "no compila") {
		t.Fatalf("la compilación falló y no se dijo: %v", err)
	}
	if tags := tagsDe(t, dir); tags != "" {
		t.Errorf("no debía crear el tag: %s", tags)
	}
}

func TestRelease_UnTagQueYaExisteSeDice_FND310(t *testing.T) {
	dir := repoDeRelease(t)
	if err := publicarRelease(dir, "v1.0.0", sinViolaciones, sinCompilar); err != nil {
		t.Fatal(err)
	}
	if tags := tagsDe(t, dir); tags != "v1.0.0" {
		t.Fatalf("debía crear el tag v1.0.0, hay %q", tags)
	}
	if err := publicarRelease(dir, "v1.0.0", sinViolaciones, sinCompilar); err == nil {
		t.Error("el tag ya existía y la release dijo que terminó bien")
	}
}

func TestRelease_SinScriptDeBuildNoCompila_FND310(t *testing.T) {
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "package.json"), `{"name":"x"}`)
	if err := compilarRelease(dir); err != nil {
		t.Errorf("sin script build no hay qué compilar: %v", err)
	}
	mustWrite(t, filepath.Join(dir, "package.json"), `{"scripts":{"build":"exit 3"}}`)
	if err := compilarRelease(dir); err == nil {
		t.Error("npm run build falla y no se dijo")
	}
}

func sinViolaciones(string) []audit.Violation { return nil }

func sinCompilar(string) error { return nil }

func repoDeRelease(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, ".qdd"))
	mustWrite(t, filepath.Join(dir, ".qdd", "state.json"), `{"version":"v0.9.0"}`)
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}, {"-c", "user.email=a@b.c", "-c", "user.name=n", "commit", "-qm", "feat: x"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func tagsDe(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "tag")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
