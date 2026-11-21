package mejoras

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/consola"
	"github.com/qdd-framework/qdd/pkg/estado"
	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'

// fotoDeArchivos lee cada archivo bajo las carpetas, menos la hora del último ciclo (que el ciclo anota siempre).
func fotoDeArchivos(t *testing.T, carpetas ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, dir := range carpetas {
		filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || strings.Contains(p, "mejoras-ciclo") {
				return nil
			}
			b, _ := os.ReadFile(p)
			out[p] = string(b)
			return nil
		})
	}
	return out
}

// Un reporte y un issue entran una vez aunque vuelvan a llegar, y un ciclo sin cambios no escribe nada: ni la cola, ni
// el tablero, ni la entrada.
// contrato: mejoras#94bd81a0
func TestCiclo_CadaReporteEIssueEntraUnaVezYUnCicloSinCambiosNoEscribeNada(t *testing.T) {
	root := proyecto(t)
	issues := conIssues(
		Issue{Numero: 4, Titulo: "Mejora del dueño", Asociacion: "OWNER", Creado: t0},
		Issue{Numero: 5, Titulo: "Otra del dueño", Asociacion: "MEMBER", Creado: t0.Add(time.Second)},
		Issue{Numero: 6, Titulo: "Una tercera", Asociacion: "OWNER", Creado: t0.Add(2 * time.Second)},
	)
	reportar(t, root, "Las consultas no salen en el tablero", t0)
	ciclo(t, root, issues, t0)
	antes := fotoDeArchivos(t, root, os.Getenv("QDD_HOME"))
	r := ciclo(t, root, issues, t0.Add(time.Minute))
	if len(r.Nuevas)+len(r.Tareas)+len(r.Hechas) != 0 || r.Mensaje != "" {
		t.Fatalf("un ciclo sin cambios no hace nada: %+v", r)
	}
	despues := fotoDeArchivos(t, root, os.Getenv("QDD_HOME"))
	for ruta, contenido := range antes {
		if despues[ruta] != contenido {
			t.Fatalf("el segundo ciclo cambió %s", ruta)
		}
	}
	if len(despues) != len(antes) {
		t.Fatalf("el segundo ciclo creó o borró archivos: %d → %d", len(antes), len(despues))
	}

	reportar(t, root, "Las consultas no salen en el tablero", t0.Add(2*time.Minute))
	ciclo(t, root, issues, t0.Add(3*time.Minute))
	porRef := map[string]int{}
	for _, m := range mustLista(t, root) {
		porRef[m.Ref]++
	}
	if len(mustLista(t, root)) != 4 {
		t.Fatalf("un reporte y tres issues, cada uno una vez: %+v", mustLista(t, root))
	}
	for ref, n := range porRef {
		if n != 1 {
			t.Fatalf("%s entró %d veces", ref, n)
		}
	}
}

// Una fuente que falla (GitHub sin red, el relay caído) se salta y lo dice; el reporte local entra igual y lo que ya
// estaba en la cola desde GitHub no se descarta por no haber podido leerlo.
// contrato: mejoras#b214e579
func TestCiclo_UnaFuenteQueFallaSeSaltaLoDiceYNoSePierdeNada(t *testing.T) {
	root := proyecto(t)
	ciclo(t, root, conIssues(
		Issue{Numero: 1, Titulo: "Uno", Asociacion: "OWNER", Creado: t0},
		Issue{Numero: 2, Titulo: "Dos", Asociacion: "OWNER", Creado: t0.Add(time.Second)},
		Issue{Numero: 3, Titulo: "Tres", Asociacion: "OWNER", Creado: t0.Add(2 * time.Second)},
	), t0)
	reportar(t, root, "Un reporte de una sesión", t0.Add(time.Minute))
	caidas := Fuentes{
		GitHub: func(string) (string, []Issue, error) { return "", nil, errors.New("gh: sin red") },
		Relay:  func(string, string) ([]ReporteRelay, error) { return nil, errors.New("relay: sin red") },
	}
	r := ciclo(t, root, caidas, t0.Add(2*time.Minute))
	if len(r.Avisos) != 2 || !strings.Contains(strings.Join(r.Avisos, "|"), "gh: sin red") || !strings.Contains(strings.Join(r.Avisos, "|"), "relay: sin red") {
		t.Fatalf("cada fuente que falla dice por qué: %+v", r.Avisos)
	}
	if len(r.Nuevas) != 1 {
		t.Fatalf("el reporte local entra igual: %+v", r)
	}
	for _, m := range mustLista(t, root) {
		if m.Estado == Descartada || m.Estado == Hecha {
			t.Fatalf("sin leer GitHub nada se descarta ni se da por hecho: %+v", m)
		}
	}
	if _, _, err := issuesDeGitHubSinGh(t, root); err == nil || !strings.Contains(err.Error(), "falta gh") {
		t.Fatalf("sin gh dice qué falta: %v", err)
	}
}

// gitDelSistema se busca antes de que una prueba cambie el PATH.
var gitDelSistema, _ = exec.LookPath("git")

// repoDeGitHub deja el proyecto con un remote de GitHub y un PATH con solo git y los espías que se le pasen.
func repoDeGitHub(t *testing.T, root string, espias map[string]string) (bin string) {
	t.Helper()
	if gitDelSistema == "" {
		t.Skip("sin git")
	}
	for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", "https://github.com/trdago/demo.git"}} {
		c := exec.Command(gitDelSistema, args...)
		c.Dir = root
		c.Env = append(gitenv.Limpio(os.Environ()), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	bin = t.TempDir()
	if err := os.Symlink(gitDelSistema, filepath.Join(bin, "git")); err != nil {
		t.Fatal(err)
	}
	for nombre, script := range espias {
		if err := os.WriteFile(filepath.Join(bin, nombre), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	return bin
}

func issuesDeGitHubSinGh(t *testing.T, root string) (string, []Issue, error) {
	t.Helper()
	repoDeGitHub(t, root, nil)
	return issuesDeGitHub(root)
}

// Con las fuentes de verdad, el ciclo solo lee GitHub (gh api con GET) y no corre ningún modelo; cerrar el issue de una
// mejora hecha queda como pedido de la persona, sin correrlo.
// contrato: mejoras#65b59eb8
func TestCiclo_SoloLeeGitHubYNoLlamaANingunModelo(t *testing.T) {
	root := proyecto(t)
	log := filepath.Join(t.TempDir(), "llamadas.log")
	issues := filepath.Join(t.TempDir(), "issues.json")
	os.WriteFile(issues, []byte(`[{"number":12,"title":"Mejora del dueño","body":"x","html_url":"u","created_at":"2026-11-03T15:00:00Z","author_association":"OWNER","user":{"login":"trdago"},"labels":[]}]`+"\n"), 0o644)
	espia := func(nombre string) string { return "echo \"" + nombre + " $*\" >> '" + log + "'\n" }
	espias := map[string]string{
		"gh": espia("gh") + "case \"$*\" in\n*'issues?state=open'*) while IFS= read -r l; do echo \"$l\"; done < '" + issues + "';;\n*) echo '{\"state\":\"open\"}';;\nesac\n",
	}
	for _, modelo := range []string{"claude", "ollama", "codex", "gemini", "llm", "curl", "wget", "lms"} {
		espias[modelo] = espia(modelo) + "exit 1\n"
	}
	repoDeGitHub(t, root, espias)

	r, err := Ciclo(root, FuentesReales(), t0)
	if err != nil || len(r.Tareas) != 1 {
		t.Fatalf("el issue entra y es tarea: %+v %v", r, err)
	}
	if err := estado.Cerrar(root, mejora(t, root, "MEJ-001").Tarea, "commit abc123", false, t0.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := Ciclo(root, FuentesReales(), t0.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	pedido := false
	for _, p := range consola.PedidosA(root, t0.Add(2*time.Minute)) {
		pedido = pedido || strings.Contains(p.Comando, "gh issue close 12")
	}
	if !pedido {
		t.Fatal("cerrar el issue queda como pedido de la persona")
	}
	b, _ := os.ReadFile(log)
	llamadas := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(llamadas) == 0 || llamadas[0] == "" {
		t.Fatal("leyó GitHub con gh")
	}
	for _, l := range llamadas {
		if !strings.HasPrefix(l, "gh api repos/trdago/demo/issues") {
			t.Fatalf("solo gh api para leer issues: %q", l)
		}
		for _, escribe := range []string{" -X", "--method", " -f ", " -F ", "--field", "--input", "close", "comment", "POST", "PATCH"} {
			if strings.Contains(l, escribe) {
				t.Fatalf("el ciclo no publica en GitHub: %q", l)
			}
		}
	}
}
