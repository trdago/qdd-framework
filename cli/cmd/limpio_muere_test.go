package cmd

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

// bwrapDeCertificar devuelve el comando bwrap de certificar.sh tal como está escrito, hasta dentro.sh.
func bwrapDeCertificar(t *testing.T) (string, string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "e2e", "limpio", "certificar.sh"))
	if err != nil {
		t.Fatal(err)
	}
	_, resto, ok := strings.Cut(string(data), "\nbwrap ")
	if !ok {
		t.Fatal("certificar.sh no arma el ambiente con bwrap")
	}
	comando, _, ok := strings.Cut(resto, "/bin/bash /mnt/dentro.sh")
	if !ok {
		t.Fatal("el bwrap de certificar.sh no corre /mnt/dentro.sh")
	}
	return string(data), "bwrap " + comando + "/bin/bash /mnt/dentro.sh"
}

// Corre el bwrap de certificar.sh con la raíz de esta máquina en lugar de la de Debian: lo que dentro.sh escribe en
// su HOME y en /tmp existe mientras corre y no queda afuera; la raíz y la entrada no se pueden escribir.
// contrato: ambiente-limpio#c686e2bf
func TestLimpio_LoQueEscribeElAmbienteMuereConEl(t *testing.T) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("sin bubblewrap")
	}
	if err := exec.Command("bwrap", "--ro-bind", "/", "/", "--unshare-user", "--unshare-net", "true").Run(); err != nil {
		t.Skipf("bubblewrap no puede crear un ambiente aquí: %v", err)
	}
	certificar, bwrap := bwrapDeCertificar(t)
	if !strings.Contains(certificar, `trap 'rm -rf "$entrada"' EXIT`) {
		t.Error("certificar.sh no borra la entrada del ambiente al salir")
	}
	entrada := t.TempDir()
	afuera := t.TempDir()
	marcaTmp := fmt.Sprintf("/tmp/qdd-limpio-marca-%d", time.Now().UnixNano())
	dentro := "echo hola > \"$HOME/marca\" && cat \"$HOME/marca\" > /dev/null && echo home-escribe\n" +
		"echo hola > " + marcaTmp + " && echo tmp-escribe\n" +
		"( echo x > /mnt/escrito ) 2>/dev/null && echo mnt-escribe\n" +
		"( echo x > " + afuera + "/escrito ) 2>/dev/null && echo raiz-escribe\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(entrada, "dentro.sh"), []byte(dentro), 0o644); err != nil {
		t.Fatal(err)
	}
	script := "set -euo pipefail\nraiz=/\nentrada=" + entrada + "\nred=(--unshare-net)\nversion=\"\"\n" + bwrap + "\n"
	out, err := exec.Command("bash", "-c", script).CombinedOutput()
	if err != nil {
		t.Fatalf("el ambiente no corrió: %v\n%s", err, out)
	}
	salida := string(out)
	if !strings.Contains(salida, "home-escribe") || !strings.Contains(salida, "tmp-escribe") {
		t.Fatalf("dentro del ambiente se escribe en HOME y /tmp: %q", salida)
	}
	for _, prohibido := range []string{"mnt-escribe", "raiz-escribe"} {
		if strings.Contains(salida, prohibido) {
			t.Errorf("el ambiente pudo escribir fuera de su tmpfs: %q", salida)
		}
	}
	for _, ruta := range []string{marcaTmp, filepath.Join(entrada, "escrito"), filepath.Join(afuera, "escrito")} {
		if _, err := os.Stat(ruta); err == nil {
			t.Errorf("lo que escribió el ambiente quedó en la máquina: %s", ruta)
			os.Remove(ruta)
		}
	}
	quedan, _ := os.ReadDir(entrada)
	if len(quedan) != 1 {
		t.Errorf("la entrada quedó con %d archivos; want solo dentro.sh", len(quedan))
	}
}
