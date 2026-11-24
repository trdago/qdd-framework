package cmd

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/gitenv"
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

func TestRelease_CadaProyectoCompilaConLoSuyo_FND310(t *testing.T) {
	nada, makefile, node := t.TempDir(), t.TempDir(), t.TempDir()
	mustWrite(t, filepath.Join(makefile, "Makefile"), "build:\n")
	mustWrite(t, filepath.Join(node, "Makefile"), "build:\n")
	mustWrite(t, filepath.Join(node, "package.json"), "{}")
	for dir, quiere := range map[string]string{nada: "", makefile: "make build", node: "npm run build --if-present"} {
		if got := strings.Join(comandoDeCompilar(dir), " "); got != quiere {
			t.Errorf("compila con %q, se esperaba %q", got, quiere)
		}
	}
	if err := compilarRelease(nada); err != nil {
		t.Errorf("sin qué compilar no es un error: %v", err)
	}
}

// Desde un hook, git exporta GIT_DIR: la release etiqueta el repositorio del proyecto, no el del hook.
func TestRelease_NoEtiquetaElRepositorioDelHook_FND310(t *testing.T) {
	dir, hook := repoDeRelease(t), repoDeRelease(t)
	t.Setenv("GIT_DIR", filepath.Join(hook, ".git"))
	if err := publicarRelease(dir, "v1.0.0", sinViolaciones, sinCompilar); err != nil {
		t.Fatal(err)
	}
	if tags := tagsDe(t, hook); tags != "" {
		t.Errorf("etiquetó el repositorio del hook: %s", tags)
	}
	if tags := tagsDe(t, dir); tags != "v1.0.0" {
		t.Errorf("el proyecto quedó con los tags %q", tags)
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
		cmd.Dir, cmd.Env = dir, gitenv.Limpio(os.Environ())
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return dir
}

func tagsDe(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "tag")
	cmd.Dir, cmd.Env = dir, gitenv.Limpio(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}
