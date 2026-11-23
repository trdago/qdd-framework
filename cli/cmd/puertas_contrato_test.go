package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/puertas"
)

// Rule: Prohibido usar 'else'
// Contrato de la ficha puertas: una puerta que no puede decidir deja pasar y anota el error, y doctor las revisa y
// las repara sin tocar lo ajeno.

// ultimoRegistro es la última decisión anotada del repositorio.
func ultimoRegistro(t *testing.T, home, repo string) puertas.Registro {
	t.Helper()
	regs, _, err := puertas.LeerRegistro(home, filepath.Base(repo))
	if err != nil || len(regs) == 0 {
		t.Fatalf("no hay decisiones anotadas: %v", err)
	}
	return regs[len(regs)-1]
}

// Un error propio (la marca del audit no se puede leer) deja pasar el commit, por el panel y por el hook, y se anota.
// contrato: puertas#91d6f84a
func TestPuertas_UnErrorPropioDejaPasarYSeAnota(t *testing.T) {
	home, repo := proyectoConPuertas(t, "puertas: {}\n")
	t.Setenv("CLAUDECODE", "")
	marca := filepath.Join(repo, ".git", "qdd-audit-marca")
	if err := os.MkdirAll(marca, 0o755); err != nil {
		t.Fatal(err)
	}
	if codigo, msg := revisarPuertas(repo, "s-1", "git commit -m x"); codigo != 0 || msg != "" {
		t.Fatalf("el panel deja pasar: %d %q", codigo, msg)
	}
	if r := ultimoRegistro(t, home, repo); r.Resultado != puertas.Error || r.Puerta != puertas.PuertaAudit {
		t.Fatalf("y anota el error: %+v", r)
	}
	if codigo, msg := hookPuertas(repo, "pre-commit", ""); codigo != 0 || msg != "" {
		t.Fatalf("el hook deja pasar: %d %q", codigo, msg)
	}
	if regs, _, _ := puertas.LeerRegistro(home, filepath.Base(repo)); len(regs) != 2 || regs[1].Resultado != puertas.Error {
		t.Fatalf("y también lo anota: %+v", regs)
	}
}

// Una puerta que tarda más que su tope (aquí, un git lento al armar la huella) deja pasar y anota el error, por el
// panel y por el hook de git, que no tiene otro tope: sin esto el commit esperaba lo que tardara git (FND-277).
// contrato: puertas#91d6f84a
func TestPuertas_UnaPuertaQueNoDecideATiempoDejaPasarYLoAnota_FND277(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("el git lento es un script de sh")
	}
	gitReal, err := exec.LookPath("git")
	if err != nil {
		t.Skip("sin git")
	}
	home, repo := proyectoConPuertas(t, "puertas: {}\n")
	t.Setenv("CLAUDECODE", "")
	if err := puertas.GuardarMarca(repo, "abc"); err != nil {
		t.Fatal(err)
	}
	lento := t.TempDir()
	// Cada «add -A» abandonado deja una marca al terminar: la limpieza la espera en vez de adivinar cuánto tarda.
	script := "#!/bin/sh\ncase \" $* \" in *\" add -A \"*) sleep 3; " + gitReal + " \"$@\"; r=$?; touch " + lento + "/listo.$$; exit $r ;; esac\nexec " + gitReal + " \"$@\"\n"
	if err := os.WriteFile(filepath.Join(lento, "git"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", lento+string(os.PathListSeparator)+os.Getenv("PATH"))
	viejo := topePuertas
	topePuertas = 200 * time.Millisecond
	t.Cleanup(func() { topePuertas = viejo })
	// Las dos huellas abandonadas (panel y hook) siguen en segundo plano: se esperan antes de borrar el repositorio
	// (con tope, por si una no llegó a git).
	t.Cleanup(func() {
		for limite := time.Now().Add(15 * time.Second); time.Now().Before(limite); time.Sleep(20 * time.Millisecond) {
			if listos, _ := filepath.Glob(filepath.Join(lento, "listo.*")); len(listos) >= 2 {
				return
			}
		}
	})

	for nombre, revisar := range map[string]func() (int, string){
		"panel": func() (int, string) { return revisarPuertas(repo, "s-1", "git commit -m x") },
		"hook":  func() (int, string) { return hookPuertas(repo, "pre-commit", "") },
	} {
		inicio := time.Now()
		codigo, msg := revisar()
		// Git tarda 3 s y el tope es de 200 ms: 2 s de margen para una máquina cargada (FND-298) siguen probando que no
		// lo espera.
		if d := time.Since(inicio); d > 2*time.Second {
			t.Errorf("%s: no espera a git más que su tope: %s", nombre, d)
		}
		if codigo != 0 || msg != "" {
			t.Errorf("%s: sin decidir a tiempo, deja pasar: %d %q", nombre, codigo, msg)
		}
		if r := ultimoRegistro(t, home, repo); r.Resultado != puertas.Error {
			t.Errorf("%s: y anota el error: %+v", nombre, r)
		}
	}
}

// doctor ve la política y cada hook (falta o viejo); --fix los deja al día sin tocar lo ajeno del hook.
// contrato: puertas#77eb9692
func TestDoctorPuertas_DiagnosticaYReparaSinTocarHooksAjenos(t *testing.T) {
	_, repo := proyectoConPuertas(t, "zero_else: true\n")
	ajeno := "#!/bin/sh\nnpx lint-staged\n"
	escribirEnRepo(t, repo, ".git/hooks/pre-commit", ajeno)
	escribirEnRepo(t, repo, ".git/hooks/commit-msg", "#!/bin/sh\n# >>> qdd puertas >>> (lo instala qdd puertas instalar; qdd doctor --fix lo repara)\nqdd puertas hook viejo\n# <<< qdd puertas <<<\necho mío\n")
	g := buildPuertasChecks(repo)
	if !groupHasFailures(g) || len(g.Items) != 1 || !strings.Contains(g.Items[0].Error, "desactivadas") {
		t.Fatalf("sin la política, doctor lo dice: %+v", g.Items)
	}
	escribirEnRepo(t, repo, ".qdd/policies.yaml", "zero_else: true\npuertas: {}\n")
	g = buildPuertasChecks(repo)
	fallas := map[string]string{}
	for _, it := range g.Items {
		if !it.Success {
			fallas[it.Name] = it.Error
		}
	}
	if !strings.HasPrefix(fallas["Hook pre-commit de las puertas"], puertas.HookFalta) || !strings.HasPrefix(fallas["Hook commit-msg de las puertas"], puertas.HookViejo) {
		t.Fatalf("diagnostica el hook que falta y el viejo: %v", fallas)
	}
	repairPuertas(repo, g)
	if g := buildPuertasChecks(repo); groupHasFailures(g) {
		t.Fatalf("después de --fix queda al día: %+v", g.Items)
	}
	pre, _ := os.ReadFile(filepath.Join(repo, ".git", "hooks", "pre-commit"))
	if !strings.HasSuffix(string(pre), "npx lint-staged\n") || !strings.Contains(string(pre), "qdd puertas hook pre-commit") {
		t.Fatalf("el hook ajeno sigue entero y suma el bloque:\n%s", pre)
	}
	msg, _ := os.ReadFile(filepath.Join(repo, ".git", "hooks", "commit-msg"))
	if strings.Contains(string(msg), "hook viejo") || !strings.HasSuffix(string(msg), "echo mío\n") {
		t.Fatalf("el bloque viejo se reemplaza y lo ajeno sigue:\n%s", msg)
	}
}
