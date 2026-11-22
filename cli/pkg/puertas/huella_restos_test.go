package puertas

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-293: cuando el hook se rinde por tiempo (topePuertas, FND-277), qdd sale con la huella a medias: su defer no
// corre y el git abandonado puede seguir escribiendo el índice aparte. Quedaban .git/qdd-huella-* (y su .lock) para
// siempre. Cada huella limpia las que quedaron de antes; una reciente puede ser de otra huella que corre y no se toca.

func TestHuella_LimpiaLosRestosViejosYNoLosRecientes_FND293(t *testing.T) {
	dir := repoDePrueba(t)
	escribir(t, dir, "a.txt", "a")
	gitDir := filepath.Join(dir, ".git")
	viejo, viejoLock, reciente := filepath.Join(gitDir, "qdd-huella-111"), filepath.Join(gitDir, "qdd-huella-111.lock"), filepath.Join(gitDir, "qdd-huella-222")
	hace := time.Now().Add(-2 * RestoHuella)
	for _, f := range []string{viejo, viejoLock, reciente} {
		if err := os.WriteFile(f, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{viejo, viejoLock} {
		if err := os.Chtimes(f, hace, hace); err != nil {
			t.Fatal(err)
		}
	}
	mustHuella(t, dir)
	for _, f := range []string{viejo, viejoLock} {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Fatalf("un resto viejo se borra: %s (%v)", f, err)
		}
	}
	if _, err := os.Stat(reciente); err != nil {
		t.Fatalf("uno reciente puede ser de otra huella en curso: %v", err)
	}
	restos, _ := filepath.Glob(filepath.Join(gitDir, "qdd-huella-*"))
	if len(restos) != 1 {
		t.Fatalf("la huella no deja el suyo: %v", restos)
	}
}
