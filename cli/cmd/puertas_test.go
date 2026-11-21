package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/puertas"
)

func TestPuertasRevisar_CommitSoloConAuditAlDia(t *testing.T) {
	home, repo := proyectoConPuertas(t, "puertas: {}\n")
	escribirEnRepo(t, repo, "a.go", "package a\n")
	if codigo, msg := revisarPuertas(repo, "s-1", "git add -A && git commit -m 'feat: a'"); codigo != codigoRechazo || !strings.Contains(msg, "qdd audit") {
		t.Fatalf("sin audit aprobado el commit se rechaza: %d %q", codigo, msg)
	}
	huella, err := puertas.Huella(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := puertas.GuardarMarca(repo, huella); err != nil {
		t.Fatal(err)
	}
	if codigo, msg := revisarPuertas(repo, "s-1", "git add -A && git commit -m 'feat: a'"); codigo != 0 || msg != "" {
		t.Fatalf("con el audit al día pasa sin decir nada: %d %q", codigo, msg)
	}
	escribirEnRepo(t, repo, "b.go", "package b\n")
	if codigo, msg := revisarPuertas(repo, "s-1", "git commit -am x"); codigo != codigoRechazo || !strings.Contains(msg, "cambió") {
		t.Fatalf("un cambio después del audit se rechaza: %d %q", codigo, msg)
	}
	datos, err := os.ReadFile(filepath.Join(home, "puertas", filepath.Base(repo)+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(datos), "\n"); n != 3 {
		t.Fatalf("cada decisión queda anotada (3): %d\n%s", n, datos)
	}
	if strings.Contains(string(datos), repo) {
		t.Fatalf("el registro no lleva rutas: %s", datos)
	}
}

func TestPuertasRevisar_PublicarVaComoPedido(t *testing.T) {
	_, repo := proyectoConPuertas(t, "puertas: {}\n")
	escribirEnRepo(t, repo, "a.go", "package a\n")
	gitEn(t, repo, "add", "-A")
	gitEn(t, repo, "-c", "user.name=p", "-c", "user.email=p@p", "commit", "-qm", "x")
	gitEn(t, repo, "tag", "v1.0.0")
	for _, comando := range []string{"gh pr merge 3 --merge", "git push origin v1.0.0", "cd . && npm publish", "git commit --no-verify -m x"} {
		if codigo, msg := revisarPuertas(repo, "", comando); codigo != codigoRechazo || msg == "" {
			t.Errorf("%q se rechaza: %d %q", comando, codigo, msg)
		}
	}
	for _, comando := range []string{"git push -u origin fix/x", "gh pr view 3", "qdd consola pedir -m 'x' 'gh pr merge 3'"} {
		if codigo, msg := revisarPuertas(repo, "", comando); codigo != 0 {
			t.Errorf("%q pasa: %d %q", comando, codigo, msg)
		}
	}
}

func TestPuertasRevisar_SinPoliticaNoHayPuertas(t *testing.T) {
	home, repo := proyectoConPuertas(t, "zero_else: true\n")
	if codigo, msg := revisarPuertas(repo, "", "gh pr merge 3 && git commit -m x"); codigo != 0 || msg != "" {
		t.Fatalf("sin puertas en la política todo pasa: %d %q", codigo, msg)
	}
	if _, err := os.Stat(filepath.Join(home, "puertas")); !os.IsNotExist(err) {
		t.Fatalf("y no se anota nada: %v", err)
	}
}

func TestPuertasRevisar_UnaFallaPropiaDejaPasar(t *testing.T) {
	_, repo := proyectoConPuertas(t, "puertas: {}\n")
	if codigo, _ := revisarPuertas(repo, "", "git -C no-existe commit -m x"); codigo != 0 {
		t.Fatalf("si no puede decidir, la acción pasa: %d", codigo)
	}
}

func proyectoConPuertas(t *testing.T, politica string) (home, repo string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("QDD_HOME", home)
	repo = t.TempDir()
	gitEn(t, repo, "init", "-q")
	escribirEnRepo(t, repo, ".qdd/policies.yaml", politica)
	return home, repo
}

func escribirEnRepo(t *testing.T, dir, ruta, texto string) {
	t.Helper()
	p := filepath.Join(dir, ruta)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitEn(t *testing.T, dir string, args ...string) {
	t.Helper()
	if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestMarcaDelAudit_HuellaDeAntesDeAuditar(t *testing.T) {
	_, repo := proyectoConPuertas(t, "puertas: {}\n")
	escribirEnRepo(t, repo, "a.go", "package a\n")
	marcar := marcaDelAudit(repo)
	escribirEnRepo(t, repo, "generado/tipos.d.ts", "// lo dejó el audit\n")
	marcar()
	if codigo, _ := revisarPuertas(repo, "", "git commit -am x"); codigo != codigoRechazo {
		t.Fatal("lo que generó el audit no queda aprobado")
	}
	os.RemoveAll(filepath.Join(repo, "generado"))
	if codigo, msg := revisarPuertas(repo, "", "git commit -am x"); codigo != 0 {
		t.Fatalf("sin lo generado, el árbol es el auditado: %q", msg)
	}
}

func TestMarcaDelAudit_SinPuertasNoMarca(t *testing.T) {
	_, repo := proyectoConPuertas(t, "zero_else: true\n")
	marcaDelAudit(repo)()
	if m, _ := puertas.LeerMarca(repo); m != "" {
		t.Fatalf("sin puertas el audit no deja marca: %q", m)
	}
}
