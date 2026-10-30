package estado

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// RFC 0006 §3.9: lo local de un proyecto (cola de pedidos, actividad) es uno por repositorio. Un worktree usa el del
// checkout principal, así todas las sesiones de un mismo repositorio ven el mismo panel.

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	c := exec.Command("git", args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// repoConWorktree crea un repositorio con un commit y un worktree; devuelve las dos raíces (rutas reales).
func repoConWorktree(t *testing.T) (string, string) {
	t.Helper()
	base, _ := filepath.EvalSymlinks(t.TempDir())
	main := filepath.Join(base, "repo")
	if err := os.MkdirAll(filepath.Join(main, "sub", ".qdd"), 0755); err != nil {
		t.Fatal(err)
	}
	git(t, main, "init", "-q")
	git(t, main, "commit", "-q", "--allow-empty", "-m", "x")
	wt := filepath.Join(base, "wt")
	git(t, main, "worktree", "add", "-q", wt)
	return main, wt
}

func TestClaveLocal_UnWorktreeUsaLaDelRepositorio(t *testing.T) {
	main, wt := repoConWorktree(t)
	if got := ClaveLocal(wt); got != main {
		t.Fatalf("un worktree se guarda como su checkout principal: %s", got)
	}
	if got := ClaveLocal(main); got != main {
		t.Fatalf("el checkout principal es él mismo: %s", got)
	}
	if got := ClaveLocal(filepath.Join(wt, "sub")); got != filepath.Join(main, "sub") {
		t.Fatalf("un proyecto dentro del repositorio conserva su carpeta: %s", got)
	}
	suelto, _ := filepath.EvalSymlinks(t.TempDir())
	if got := ClaveLocal(suelto); got != suelto {
		t.Fatalf("fuera de git, la carpeta es ella misma: %s", got)
	}
	r1, _ := RutaLocal(wt, "pedidos")
	r2, _ := RutaLocal(main, "pedidos")
	if r1 != r2 {
		t.Fatalf("la cola de un worktree es la del repositorio: %s %s", r1, r2)
	}
}

func TestProyectoDe_EsLaCarpetaConQddOElRepositorio(t *testing.T) {
	main, wt := repoConWorktree(t)
	if got := ProyectoDe(filepath.Join(main, "sub", "x")); got != filepath.Join(main, "sub") {
		t.Fatalf("una carpeta pertenece al proyecto con .qdd más cercano: %s", got)
	}
	if got := ProyectoDe(filepath.Join(wt, "otra")); got != wt {
		t.Fatalf("sin .qdd, al repositorio: %s", got)
	}
	suelto, _ := filepath.EvalSymlinks(t.TempDir())
	if got := ProyectoDe(suelto); got != "" {
		t.Fatalf("fuera de un proyecto y de git, ninguno: %s", got)
	}
}
