//go:build linux

package consola

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-179: Process.Release() no recoge al hijo en Linux; cada proceso lanzado por 'qdd consola seguir' quedaba zombie
// mientras viviera la consola (919 en ~4 h, MEJ-042).

func TestSoltar_UnHijoQueTerminaNoQuedaZombie_FND179(t *testing.T) {
	c := exec.Command("true")
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	pid := c.Process.Pid
	soltar(c)
	if zombie(pid, 5*time.Second) {
		t.Fatalf("el hijo %d quedó zombie", pid)
	}
}

func TestSoltar_NoEsperaAQueElHijoTermine(t *testing.T) {
	c := exec.Command("sleep", "2")
	if err := c.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Process.Kill() }()
	antes := time.Now()
	soltar(c)
	if time.Since(antes) > 500*time.Millisecond {
		t.Fatal("soltar bloqueó hasta que el hijo terminó")
	}
}

func TestSoltar_NingunLanzamientoUsaRelease_FND179(t *testing.T) {
	for _, dir := range []string{".", "../incidentes"} {
		archivos, _ := filepath.Glob(filepath.Join(dir, "*.go"))
		for _, a := range archivos {
			if strings.HasSuffix(a, "_test.go") {
				continue
			}
			b, err := os.ReadFile(a)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "Process.Release()") {
				t.Errorf("%s usa Process.Release(): en Linux deja al hijo zombie; usa soltar", a)
			}
		}
	}
}

// zombie dice si el proceso sigue como zombie al cumplirse el tope. Si desaparece antes, alguien lo recogió.
func zombie(pid int, tope time.Duration) bool {
	limite := time.Now().Add(tope)
	for time.Now().Before(limite) {
		if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err != nil {
			return false
		}
		time.Sleep(20 * time.Millisecond)
	}
	return true
}
