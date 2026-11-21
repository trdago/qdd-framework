package incidentes

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// comoBinarioDeQDD hace que Lanzar se comporte como en el producto: solo se apaga en un binario de go test.
func comoBinarioDeQDD(t *testing.T) {
	t.Helper()
	antes := os.Args[0]
	os.Args[0] = "/usr/bin/qdd"
	t.Cleanup(func() { os.Args[0] = antes })
}

// La publicación es un proceso aparte: Lanzar vuelve en el acto aunque publicar tarde segundos.
// contrato: incidentes#e2a4dba8
func TestLanzar_VuelveEnElActoYPublicaAparte(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("el publicador falso es un script de sh")
	}
	home := t.TempDir()
	t.Setenv(HomeEnv, home)
	t.Setenv(ApagadoEnv, "")
	comoBinarioDeQDD(t)
	marca := filepath.Join(t.TempDir(), "publicar")
	bin := filepath.Join(t.TempDir(), "qdd")
	script := "#!/bin/sh\necho \"$@\" > " + marca + ".tmp\nmv " + marca + ".tmp " + marca + "\nsleep 3\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, "incidentes")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}

	inicio := time.Now()
	lanzado := Lanzar(dir, bin, time.Now())
	demora := time.Since(inicio)

	if !lanzado {
		t.Fatal("Lanzar no lanzó la publicación")
	}
	if demora > time.Second {
		t.Errorf("Lanzar demoró %v a quien falló; la publicación tiene que correr aparte", demora)
	}
	limite := time.Now().Add(5 * time.Second)
	for time.Now().Before(limite) {
		data, err := os.ReadFile(marca)
		if err == nil {
			if got := strings.TrimSpace(string(data)); got != "incidentes publicar --silencioso" {
				t.Errorf("se lanzó %q; want «incidentes publicar --silencioso»", got)
			}
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("la publicación aparte nunca corrió")
}
