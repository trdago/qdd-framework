package consola

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/estado"
)

// Rule: Prohibido usar 'else'
// Contrato de la ficha goal: el modo nunca abre una sesión y solo cuenta como avance lo que hizo alguien.

// Sin una sesión abierta el modo se pausa y no la abre: un claude falso en el PATH anota si alguien lo lanzó. Las vueltas
// las pide el panel de una sesión abierta; sin ella no hay vueltas, y lo que se publica dice pausado.
// contrato: goal#2bf53bbd
func TestGoal_NuncaAbreUnaSesionYVuelveConElla(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("el claude falso es un script de sh")
	}
	root := conGoal(t)
	bin := t.TempDir()
	marca := filepath.Join(t.TempDir(), "lanzado")
	escribirArchivo(t, filepath.Join(bin, "claude"), "#!/bin/sh\ntouch "+marca+"\n")
	if err := os.Chmod(filepath.Join(bin, "claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	// La última vuelta del panel que se cerraba llega cuando la sesión ya no late.
	sinSesion := enCola.Add(VidaSesion + time.Second)
	cicloGoal(t, root, EntradaGoal{Silencio: time.Hour}, sinSesion)
	if _, err := os.Stat(marca); err == nil {
		t.Fatal("el modo goal nunca lanza claude para abrir una sesión")
	}
	if HaySesion(root, sinSesion) {
		t.Fatal("tampoco se anota una sesión que nadie abrió")
	}
	if g := VistaGoal(root, sinSesion); g.Estado != GoalPausado || g.Motivo != MotivoSinConsola {
		t.Fatalf("sin una sesión abierta se pausa: %+v", g)
	}
	if err := LatirSesion(root, "s1", sinSesion); err != nil {
		t.Fatal(err)
	}
	if g := VistaGoal(root, sinSesion); !g.Activo || g.Estado == GoalPausado && g.Motivo == MotivoSinConsola {
		t.Fatalf("vuelve cuando la sesión vuelve: %+v", g)
	}
}

// Avance real es el tablero (sin su fecha), el último commit y los cambios de git; lo que qdd escribe solo no cuenta.
// contrato: goal#8bc038cf
func TestGoal_SoloElTableroElCommitYLosCambiosSonAvance(t *testing.T) {
	root := conGoal(t)
	git := func(args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-c", "user.email=a@b", "-c", "user.name=a"}, args...)...)
		c.Dir = root
		if b, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, b)
		}
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-qm", "base")
	base := huellaGoal(root)

	e, err := estado.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	e.Actualizado = "2030-01-01T00:00:00Z"
	if err := estado.Save(root, e); err != nil {
		t.Fatal(err)
	}
	for _, ruta := range []string{".qdd/project/metrics/m.json", ".qdd/project/evidence/e.json", ".qdd/project/cola_de_mejoras/c.yaml", ".qdd/working"} {
		escribirArchivo(t, filepath.Join(root, ruta), "qdd")
	}
	if huellaGoal(root) != base {
		t.Fatal("la fecha del tablero y lo que qdd escribe solo no son avance")
	}

	if err := estado.Set(root, estado.Cambio{Frente: "barra", Etapa: "PR", Estado: "✅", Evidencia: "PR #9"}, enCola); err != nil {
		t.Fatal(err)
	}
	tablero := huellaGoal(root)
	if tablero == base {
		t.Fatal("una celda nueva del tablero es avance")
	}
	git("commit", "-q", "--allow-empty", "-m", "vacío")
	commit := huellaGoal(root)
	if commit == tablero {
		t.Fatal("un commit nuevo es avance")
	}
	escribirArchivo(t, filepath.Join(root, "codigo.go"), "package x\n")
	if huellaGoal(root) == commit {
		t.Fatal("un cambio del proyecto sin commitear es avance")
	}
}
