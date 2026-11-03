package gitenv

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-117: el pre-push de husky corre «make test» desde un worktree con GIT_DIR exportado; las pruebas que corren git
// en carpetas temporales lo heredaban y escribían ~42 commits y dos tags en el repositorio real.

func init() {
	LimpiarProceso()
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestLimpiarProceso_GitEnUnaCarpetaTemporalNoTocaElRepoDelHook(t *testing.T) {
	real := t.TempDir()
	git(t, real, "init", "-q", "-b", "main")
	git(t, real, "-c", "user.name=a", "-c", "user.email=a@a", "commit", "-q", "--allow-empty", "-m", "real")
	t.Setenv("GIT_DIR", filepath.Join(real, ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(real, ".git", "index"))

	LimpiarProceso()

	tmp := t.TempDir()
	git(t, tmp, "init", "-q", "-b", "main")
	git(t, tmp, "-c", "user.name=a", "-c", "user.email=a@a", "commit", "-q", "--allow-empty", "-m", "fixture")
	git(t, tmp, "tag", "v0.4.2")
	os.Unsetenv("GIT_DIR")
	if log := git(t, real, "log", "--format=%s"); log != "real" {
		t.Fatalf("el repositorio del hook quedó intacto: %q", log)
	}
	if tags := git(t, real, "tag"); tags != "" {
		t.Fatalf("sin tags ajenos: %q", tags)
	}
}

func TestLimpio_QuitaLasVariablesQueApuntanARepositorio(t *testing.T) {
	env := []string{"PATH=/bin", "GIT_DIR=/x/.git", "GIT_WORK_TREE=/x", "GIT_INDEX_FILE=/x/i", "GIT_TERMINAL_PROMPT=0", "GIT_COMMON_DIR=/x"}
	got := strings.Join(Limpio(env), " ")
	if got != "PATH=/bin GIT_TERMINAL_PROMPT=0" {
		t.Fatalf("quedan solo las que no eligen repositorio: %q", got)
	}
}

// Toda carpeta de pruebas que corre git limpia el entorno antes (un paquete nuevo no puede olvidarlo).
func TestArquitectura_LasPruebasQueCorrenGitLimpianElEntorno(t *testing.T) {
	corre := regexp.MustCompile(`exec\.Command(Context)?\((ctx, )?"git"`)
	paquetes := map[string]bool{}
	limpian := map[string]bool{}
	filepath.Walk("../..", func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, _ := os.ReadFile(p)
		dir := filepath.Dir(p)
		if corre.Match(b) {
			paquetes[dir] = true
		}
		if strings.Contains(string(b), "gitenv.LimpiarProceso()") || strings.Contains(string(b), "\tLimpiarProceso()") {
			limpian[dir] = true
		}
		return nil
	})
	for dir := range paquetes {
		if !limpian[dir] {
			t.Errorf("%s corre git en sus pruebas sin gitenv.LimpiarProceso() (FND-117)", dir)
		}
	}
}
