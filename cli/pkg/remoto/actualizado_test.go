package remoto

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// FND-120: al actualizar qdd (npm i -g), el servicio seguía corriendo el qdd viejo en memoria: los comandos que el
// teléfono manda a la versión nueva (ejecutar, rc.25) quedaban cargando hasta reiniciar el servicio a mano.

func binarioFalso(t *testing.T) string {
	t.Helper()
	ruta := filepath.Join(t.TempDir(), "qdd")
	if err := os.WriteFile(ruta, []byte("v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	return ruta
}

func TestBucle_SiSuQDDSeActualizaTerminaParaQueElServicioLoReinicie(t *testing.T) {
	homeTemporal(t)
	bin := binarioFalso(t)
	_, _, _, fin := bucleDePrueba(t, func(b *Bucle) { b.Bin = bin })
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(bin, []byte("v2 más largo"), 0o755); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-fin:
		if !errors.Is(err, ErrActualizado) {
			t.Fatalf("termina con ErrActualizado: %v", err)
		}
		fin <- err
	case <-time.After(2 * time.Second):
		t.Fatal("el bucle sigue corriendo el qdd viejo")
	}
}

func TestBucle_MientrasElQDDNuevoNoTerminaDeInstalarseSigue(t *testing.T) {
	homeTemporal(t)
	bin := binarioFalso(t)
	_, _, _, fin := bucleDePrueba(t, func(b *Bucle) { b.Bin = bin })
	os.Remove(bin)
	select {
	case err := <-fin:
		t.Fatalf("sin el archivo (npm lo está reemplazando) no termina: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestBucle_ConElMismoQDDSigueCorriendo(t *testing.T) {
	homeTemporal(t)
	bin := binarioFalso(t)
	_, _, _, fin := bucleDePrueba(t, func(b *Bucle) { b.Bin = bin })
	select {
	case err := <-fin:
		t.Fatalf("nada cambió: no termina: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
}
