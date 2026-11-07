package audit

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// RFC 0020: una suite que declara sus rutas no se repite si ya pasó con esos mismos archivos. Es lo mismo que hace la caché
// de Go, para cualquier suite y cualquier lenguaje. Lo que no se puede asegurar, corre.

type repoDeSuites struct {
	raiz   string
	cuenta string
}

// repoConSuite crea un repo con una suite de la interfaz que anota cada corrida fuera del repo.
func repoConSuite(t *testing.T, salida string) repoDeSuites {
	t.Helper()
	t.Setenv("CI", "")
	t.Setenv(envPruebasCompletas, "")
	raiz := t.TempDir()
	cuenta := filepath.Join(t.TempDir(), "corridas")
	gitDePrueba(t, raiz, "init", "-q")
	writeFile(t, filepath.Join(raiz, "ui", "boton.css"), "color: red\n")
	writeFile(t, filepath.Join(raiz, "reglas", "negocio.go"), "package reglas\n")
	writeFile(t, filepath.Join(raiz, ".qdd", "config.yaml"), `test_commands:
  - comando: 'echo x >> "`+cuenta+`"; `+salida+`'
    rutas: [ui]
`)
	gitDePrueba(t, raiz, "add", ".")
	return repoDeSuites{raiz: raiz, cuenta: cuenta}
}

func gitDePrueba(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=p", "-c", "user.email=p@p.p"}, args...)...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func (r repoDeSuites) corridas(t *testing.T) int {
	t.Helper()
	b, err := os.ReadFile(r.cuenta)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(b), "x")
}

func (r repoDeSuites) auditar(t *testing.T) []Violation {
	t.Helper()
	var bloquean []Violation
	for _, v := range RunCoverageCheck(r.raiz) {
		if v.RuleID == "QA-01-ALL-TESTS-PASS" {
			bloquean = append(bloquean, v)
		}
	}
	return bloquean
}

func TestSuite_ConLosMismosArchivosNoSeRepite(t *testing.T) {
	r := repoConSuite(t, "true")
	r.auditar(t)
	r.auditar(t)
	if n := r.corridas(t); n != 1 {
		t.Fatalf("corrió %d veces", n)
	}
}

// El caso de la persona, al revés: cambiar las reglas de negocio no repite las pruebas de la interfaz.
func TestSuite_UnCambioFueraDeSusRutasNoLaRepite(t *testing.T) {
	r := repoConSuite(t, "true")
	r.auditar(t)
	writeFile(t, filepath.Join(r.raiz, "reglas", "negocio.go"), "package reglas\n// cambio\n")
	r.auditar(t)
	if n := r.corridas(t); n != 1 {
		t.Fatalf("corrió %d veces", n)
	}
}

func TestSuite_UnCambioEnSusRutasLaRepite(t *testing.T) {
	casos := map[string]func(t *testing.T, raiz string){
		"sin commitear": func(t *testing.T, raiz string) {
			writeFile(t, filepath.Join(raiz, "ui", "boton.css"), "color: blue\n")
		},
		"archivo nuevo": func(t *testing.T, raiz string) {
			writeFile(t, filepath.Join(raiz, "ui", "nuevo.css"), "x\n")
		},
		"archivo borrado": func(t *testing.T, raiz string) {
			writeFile(t, filepath.Join(raiz, "ui", "otro.css"), "x\n")
			gitDePrueba(t, raiz, "add", ".")
			if err := os.Remove(filepath.Join(raiz, "ui", "boton.css")); err != nil {
				t.Fatal(err)
			}
		},
	}
	for nombre, cambiar := range casos {
		t.Run(nombre, func(t *testing.T) {
			r := repoConSuite(t, "true")
			r.auditar(t)
			cambiar(t, r.raiz)
			r.auditar(t)
			if n := r.corridas(t); n != 2 {
				t.Fatalf("corrió %d veces", n)
			}
		})
	}
}

func TestSuite_UnaQueFallaNuncaSeRecuerda(t *testing.T) {
	r := repoConSuite(t, "exit 3")
	if v := r.auditar(t); len(v) != 1 {
		t.Fatalf("bloquea: %+v", v)
	}
	if v := r.auditar(t); len(v) != 1 {
		t.Fatalf("sigue bloqueando: %+v", v)
	}
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

func TestSuite_EnElCIyConCompletoSiempreCorre(t *testing.T) {
	for _, variable := range []string{"CI", envPruebasCompletas} {
		t.Run(variable, func(t *testing.T) {
			r := repoConSuite(t, "true")
			r.auditar(t)
			t.Setenv(variable, "1")
			r.auditar(t)
			if n := r.corridas(t); n != 2 {
				t.Fatalf("corrió %d veces", n)
			}
		})
	}
}

func TestSuite_SinRutasSiempreCorre(t *testing.T) {
	r := repoConSuite(t, "true")
	writeFile(t, filepath.Join(r.raiz, ".qdd", "config.yaml"), `test_commands:
  - 'echo x >> "`+r.cuenta+`"'
`)
	r.auditar(t)
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

// Una ruta mal escrita haría que la suite no corriera nunca más sin que nadie lo note.
func TestSuite_UnaRutaSinArchivosSiempreCorre(t *testing.T) {
	r := repoConSuite(t, "true")
	writeFile(t, filepath.Join(r.raiz, ".qdd", "config.yaml"), `test_commands:
  - comando: 'echo x >> "`+r.cuenta+`"'
    rutas: [ui, no-existe]
`)
	r.auditar(t)
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

func TestSuite_FueraDeUnRepoGitSiempreCorre(t *testing.T) {
	t.Setenv("CI", "")
	t.Setenv(envPruebasCompletas, "")
	raiz := t.TempDir()
	cuenta := filepath.Join(t.TempDir(), "corridas")
	writeFile(t, filepath.Join(raiz, "ui", "a.css"), "x\n")
	writeFile(t, filepath.Join(raiz, ".qdd", "config.yaml"), `test_commands:
  - comando: 'echo x >> "`+cuenta+`"'
    rutas: [ui]
`)
	r := repoDeSuites{raiz: raiz, cuenta: cuenta}
	r.auditar(t)
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

func TestSuite_CambiarElComandoLaRepite(t *testing.T) {
	r := repoConSuite(t, "true")
	r.auditar(t)
	writeFile(t, filepath.Join(r.raiz, ".qdd", "config.yaml"), `test_commands:
  - comando: 'echo x >> "`+r.cuenta+`"; true; true'
    rutas: [ui]
`)
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

func TestSuite_LoRecordadoIlegibleVuelveACorrer(t *testing.T) {
	r := repoConSuite(t, "true")
	r.auditar(t)
	if err := os.WriteFile(filepath.Join(r.raiz, ".git", archivoSuites), []byte("{roto"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

// Un programa puede cambiar sin que cambie ningún archivo: lo recordado vence.
func TestSuite_LoRecordadoVenceALas24Horas(t *testing.T) {
	r := repoConSuite(t, "true")
	r.auditar(t)
	ruta := filepath.Join(r.raiz, ".git", archivoSuites)
	b, err := os.ReadFile(ruta)
	if err != nil {
		t.Fatal(err)
	}
	var recordadas map[string]time.Time
	if err := json.Unmarshal(b, &recordadas); err != nil {
		t.Fatal(err)
	}
	for huella := range recordadas {
		recordadas[huella] = time.Now().Add(-vigenciaSuite - time.Minute)
	}
	viejas, _ := json.Marshal(recordadas)
	if err := os.WriteFile(ruta, viejas, 0o600); err != nil {
		t.Fatal(err)
	}
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}

// Los worktrees comparten lo recordado: con los mismos archivos, una rama nueva no repite la suite.
func TestSuite_OtroWorktreeConLosMismosArchivosNoLaRepite(t *testing.T) {
	r := repoConSuite(t, "true")
	gitDePrueba(t, r.raiz, "commit", "-qm", "base")
	r.auditar(t)
	otro := filepath.Join(t.TempDir(), "otro")
	gitDePrueba(t, r.raiz, "worktree", "add", "-q", otro)
	(repoDeSuites{raiz: otro, cuenta: r.cuenta}).auditar(t)
	if n := r.corridas(t); n != 1 {
		t.Fatalf("corrió %d veces", n)
	}
}

func TestSuite_SinSuHerramientaSigueSinMedirAunqueTengaRutas(t *testing.T) {
	r := repoConSuite(t, "true")
	writeFile(t, filepath.Join(r.raiz, ".qdd", "config.yaml"), `test_commands:
  - comando: 'echo x >> "`+r.cuenta+`"'
    rutas: [ui]
    requiere: [qdd-programa-que-no-existe]
`)
	violaciones := RunCoverageCheck(r.raiz)
	encontrada := false
	for _, v := range violaciones {
		encontrada = encontrada || v.RuleID == "QA-03-SIN-MEDIR"
	}
	if !encontrada || r.corridas(t) != 0 {
		t.Fatalf("queda sin medir: %+v, %d corridas", violaciones, r.corridas(t))
	}
}

func TestRunAllCompleto_RepiteLaSuiteYDejaElEntornoComoEstaba(t *testing.T) {
	r := repoConSuite(t, "true")
	r.auditar(t)
	os.Unsetenv(envPruebasCompletas)
	NewEngine(r.raiz).RunAllCompleto()
	if n := r.corridas(t); n != 2 {
		t.Fatalf("un certificado no usa lo recordado: corrió %d veces", n)
	}
	if _, estaba := os.LookupEnv(envPruebasCompletas); estaba {
		t.Fatal("queda como estaba: las auditorías siguientes vuelven a usar lo recordado")
	}
	r.auditar(t)
	if n := r.corridas(t); n != 2 {
		t.Fatalf("corrió %d veces", n)
	}
}
