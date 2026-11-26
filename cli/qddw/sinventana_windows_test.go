//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// Rule: Prohibido usar 'else'
// Sprint 9 (FND-238), en Windows de verdad: el qdd que lanza qddw.exe no tiene consola (GetConsoleWindow = 0), y si
// Windows termina qddw.exe (schtasks /End), el qdd que lanzó termina con él: no queda un qdd remoto huérfano.

const ayudante = `package main

import (
	"os"
	"strconv"
	"syscall"
	"time"
)

func main() {
	ventana, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetConsoleWindow").Call()
	os.WriteFile(os.Args[1], []byte(strconv.Itoa(int(ventana))+" "+strconv.Itoa(os.Getpid())), 0o644)
	time.Sleep(2 * time.Minute)
}
`

func compilar(t *testing.T, salida string, args ...string) {
	t.Helper()
	c := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), append([]string{"build", "-buildvcs=false", "-o", salida}, args...)...)
	if out, err := c.CombinedOutput(); err != nil {
		t.Fatalf("compila %s: %v\n%s", salida, err, out)
	}
}

func TestSinVentana_ElQDDQueLanzaNoTieneConsolaYTerminaConEl_FND238(t *testing.T) {
	dir := t.TempDir()
	compilar(t, filepath.Join(dir, "qddw.exe"), "-ldflags", banderasRelease, ".")
	fuente := filepath.Join(dir, "ayudante", "main.go")
	os.MkdirAll(filepath.Dir(fuente), 0o755)
	os.WriteFile(fuente, []byte(ayudante), 0o644)
	compilar(t, filepath.Join(dir, "qdd.exe"), fuente)

	salida := filepath.Join(dir, "salida")
	qddw := exec.Command(filepath.Join(dir, "qddw.exe"), salida)
	if err := qddw.Start(); err != nil {
		t.Fatal(err)
	}
	var campos []string
	for limite := time.Now().Add(30 * time.Second); len(campos) != 2 && time.Now().Before(limite); time.Sleep(100 * time.Millisecond) {
		b, _ := os.ReadFile(salida)
		campos = strings.Fields(string(b))
	}
	if len(campos) != 2 {
		qddw.Process.Kill()
		t.Fatal("el qdd de prueba no arrancó")
	}
	if campos[0] != "0" {
		t.Fatalf("el qdd que lanza qddw.exe no tiene ventana de consola: GetConsoleWindow = %s", campos[0])
	}
	pid, _ := strconv.Atoi(campos[1])
	if err := qddw.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	qddw.Wait()
	if !termina(uint32(pid), 10*time.Second) {
		t.Fatal("al terminar qddw.exe termina el qdd que lanzó: no queda huérfano")
	}
}

func termina(pid uint32, espera time.Duration) bool {
	h, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err != nil {
		return true
	}
	defer windows.CloseHandle(h)
	r, _ := windows.WaitForSingleObject(h, uint32(espera.Milliseconds()))
	return r == windows.WAIT_OBJECT_0
}
