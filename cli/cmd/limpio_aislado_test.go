package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// El ambiente limpio (e2e/limpio) solo vale si nada de esta máquina entra: certificar.sh lo arma con bubblewrap y
// dentro.sh lo comprueba como su primer paso. Esta prueba corre en cualquier PC, con o sin bwrap.

// contrato: ambiente-limpio#378ef7ec
func TestLimpio_NadaDeLaMaquinaEntraAlAmbiente(t *testing.T) {
	certificar, err := os.ReadFile(filepath.Join("..", "..", "e2e", "limpio", "certificar.sh"))
	if err != nil {
		t.Fatal(err)
	}
	_, bwrap, ok := strings.Cut(string(certificar), "\nbwrap ")
	if !ok {
		t.Fatal("certificar.sh no arma el ambiente con bwrap")
	}
	bwrap, _, _ = strings.Cut(bwrap, "/bin/bash /mnt/dentro.sh")
	for _, exigida := range []string{`--ro-bind "$raiz" /`, `--ro-bind "$entrada" /mnt`, "--tmpfs /home", "--clearenv",
		"--setenv HOME /home/persona", "--unshare-user"} {
		if !strings.Contains(bwrap, exigida) {
			t.Errorf("bwrap de certificar.sh sin %q", exigida)
		}
	}
	// Un montaje con escritura (--bind, --dev-bind) o del HOME de la máquina metería algo de afuera.
	if m := regexp.MustCompile(`--(dev-)?bind\s|\$HOME|~/`).FindString(bwrap); m != "" {
		t.Errorf("bwrap de certificar.sh monta algo con escritura o del HOME de la máquina: %q", m)
	}
	dentro, err := os.ReadFile(filepath.Join("..", "..", "e2e", "limpio", "dentro.sh"))
	if err != nil {
		t.Fatal(err)
	}
	pasos := regexp.MustCompile(`(?m)^paso "[^"]*" (\S+)`).FindAllStringSubmatch(string(dentro), -1)
	if len(pasos) == 0 || pasos[0][1] != "aislamiento" {
		t.Fatalf("el primer paso de dentro.sh comprueba el aislamiento antes de instalar nada: %v", pasos)
	}
	for _, revisa := range []string{"done < <(env)", `ls -A "$HOME"`, "ls -A /home", "ls -A /mnt", "for dir in / /mnt"} {
		if !strings.Contains(string(dentro), revisa) {
			t.Errorf("aislamiento de dentro.sh no revisa %q", revisa)
		}
	}
}
