//go:build !windows

package consola

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-129 (MEJ-003): Setsid saca al pedido de la sesión de la terminal, pero no del cgroup. Lanzado desde una unidad de
// systemd, muere cuando la unidad se detiene. En Linux corre en su propio scope; donde no se puede, como antes.

func TestFueraDeLaUnidad_ConSystemdCorreEnSuPropioScope_FND129(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("systemd solo existe en Linux")
	}
	antes := probarScope
	t.Cleanup(func() { probarScope = antes })
	probarScope = func() bool { return true }
	nombre, args := fueraDeLaUnidad("/opt/mi qdd", []string{"consola", "correr", "--raiz", "/p", "id"})
	got := nombre + " " + strings.Join(args, "|")
	if got != "systemd-run --user|--scope|--quiet|--collect|--expand-environment=no|/opt/mi qdd|consola|correr|--raiz|/p|id" {
		t.Fatalf("lanza con systemd-run, cada argumento aparte: %s", got)
	}
}

func TestFueraDeLaUnidad_SinSystemdLanzaComoAntes_FND129(t *testing.T) {
	antes := probarScope
	t.Cleanup(func() { probarScope = antes })
	probarScope = func() bool { return false }
	nombre, args := fueraDeLaUnidad("/opt/qdd", []string{"consola"})
	if nombre != "/opt/qdd" || len(args) != 1 {
		t.Fatalf("sin bus de usuario no cambia nada: %s %v", nombre, args)
	}
}

// Con systemd de verdad: el proceso conserva el PID que qdd anota, su carpeta y su entorno, y está en otro cgroup.
func TestFueraDeLaUnidad_DeVerdadConservaPIDCarpetaYEntorno_FND129(t *testing.T) {
	if runtime.GOOS != "linux" || !scopeDisponible() {
		t.Skip("sin systemd de usuario en esta máquina")
	}
	antes := probarScope
	t.Cleanup(func() { probarScope = antes })
	probarScope = func() bool { return true }
	dir := t.TempDir()
	salida := filepath.Join(dir, "salida")
	script := `pwd > ` + salida + `.tmp; echo "$MARCA_FND129" >> ` + salida + `.tmp; echo $$ >> ` + salida + `.tmp; cat /proc/self/cgroup >> ` + salida + `.tmp; mv ` + salida + `.tmp ` + salida
	nombre, args := fueraDeLaUnidad("sh", []string{"-c", script})
	c := exec.Command(nombre, args...)
	c.Dir = dir
	c.Env = append(os.Environ(), "MARCA_FND129=viaja")
	separar(c)
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	pid := c.Process.Pid
	if err := c.Wait(); err != nil {
		t.Fatal(err)
	}
	b := esperarArchivo(t, salida)
	lineas := strings.Split(strings.TrimSpace(string(b)), "\n")
	real, _ := filepath.EvalSymlinks(dir)
	if len(lineas) < 4 || lineas[0] != real || lineas[1] != "viaja" || lineas[2] != strconv.Itoa(pid) {
		t.Fatalf("carpeta, entorno y PID (%d): %q", pid, lineas)
	}
	propio, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(b, bytes.TrimSpace(propio)) {
		t.Fatalf("sigue en el cgroup de quien lo lanzó: %s", propio)
	}
}

func esperarArchivo(t *testing.T, ruta string) []byte {
	t.Helper()
	for i := 0; i < 100; i++ {
		b, err := os.ReadFile(ruta)
		if err == nil {
			return b
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("no apareció %s", ruta)
	return nil
}
