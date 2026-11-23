package foto

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'
// Fotos de QDD Móvil (ADR-0003 de qdd-movile, acordado el 2026-11-23): la foto que la persona adjunta a un mensaje
// para la IA queda en tmp/fotos/movil-<id>.<ext> del proyecto, 0600, con la extensión del tipo que dicen sus bytes.
// La persona eligió que se borren a los 7 días (el original sigue en su teléfono): al guardar una se borran las movil-*
// más viejas, nunca las de qdd foto.

var pngMinimo = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func TestGuardarDelTelefono_QuedaEnTmpFotosConSuTipo(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	ruta, err := GuardarDelTelefono(root, "f1a2b3c4", pngMinimo, now)
	if err != nil {
		t.Fatal(err)
	}
	if ruta != filepath.Join(root, "tmp", "fotos", "movil-f1a2b3c4.png") {
		t.Fatalf("la extensión sale de los bytes, no de lo declarado: %s", ruta)
	}
	info, err := os.Stat(ruta)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("solo la persona la lee: %v", info.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(root, "tmp", "fotos", ".gitignore")); err != nil {
		t.Fatal("la carpeta lleva su .gitignore: una foto nunca entra al repositorio")
	}
	if _, err := GuardarDelTelefono(root, "otra", []byte("no soy una imagen"), now); err == nil {
		t.Fatal("lo que no es una imagen no se guarda")
	}
	for _, mal := range []string{"", "../fuera", "a/b", "con espacio", string(make([]byte, 65))} {
		if _, err := GuardarDelTelefono(root, mal, pngMinimo, now); err == nil {
			t.Fatalf("id %q: solo [A-Za-z0-9_-] de 1 a 64", mal)
		}
	}
}

func TestGuardarDelTelefono_BorraLasDelTelefonoDeMasDe7Dias(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)
	vieja, _ := GuardarDelTelefono(root, "vieja", pngMinimo, now)
	reciente, _ := GuardarDelTelefono(root, "reciente", pngMinimo, now)
	delPortapapeles := filepath.Join(root, "tmp", "fotos", "2026-11-01_10-00-00.png")
	os.WriteFile(delPortapapeles, pngMinimo, 0o644)
	hace8 := now.Add(-8 * 24 * time.Hour)
	os.Chtimes(vieja, hace8, hace8)
	os.Chtimes(delPortapapeles, hace8, hace8)
	os.Chtimes(reciente, now.Add(-6*24*time.Hour), now.Add(-6*24*time.Hour))

	if _, err := GuardarDelTelefono(root, "nueva", pngMinimo, now); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vieja); !os.IsNotExist(err) {
		t.Fatal("la del teléfono de hace 8 días se borra")
	}
	if _, err := os.Stat(reciente); err != nil {
		t.Fatal("la de hace 6 días sigue")
	}
	if _, err := os.Stat(delPortapapeles); err != nil {
		t.Fatal("una de qdd foto no es del teléfono: no se toca")
	}
}
