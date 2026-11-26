package blueprints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-185 (MEJ-038): en Windows, git deja los archivos de texto con CRLF y la huella de los blueprints cambiaba. El
// lockfile escrito allí rompía qdd audit en Linux (BP-02-ALTERED) sin que nadie editara nada.

func carpetaCon(t *testing.T, archivos map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for nombre, contenido := range archivos {
		ruta := filepath.Join(dir, nombre)
		if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, []byte(contenido), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestHashDir_LaMismaRecetaConCRLFTieneLaMismaHuella_FND185(t *testing.T) {
	lf := map[string]string{"blueprint.yaml": "id: auth\nversion: 1.0.0\n", "docs/receta.md": "# Receta\n\nPaso uno.\n"}
	crlf := map[string]string{}
	for k, v := range lf {
		crlf[k] = strings.ReplaceAll(v, "\n", "\r\n")
	}
	a, err := HashDir(carpetaCon(t, lf))
	if err != nil {
		t.Fatal(err)
	}
	b, err := HashDir(carpetaCon(t, crlf))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("el fin de línea del sistema no cambia la huella: %s ≠ %s", a, b)
	}
	cambiado := map[string]string{"blueprint.yaml": "id: auth\nversion: 1.0.0\n", "docs/receta.md": "# Receta\n\nPaso dos.\n"}
	c, _ := HashDir(carpetaCon(t, cambiado))
	if c == a {
		t.Fatal("un cambio real sí cambia la huella")
	}
}

func TestHashDir_UnBinarioSeHasheaTalCual(t *testing.T) {
	a, _ := HashDir(carpetaCon(t, map[string]string{"logo.png": "\x89PNG\x00\r\n"}))
	b, _ := HashDir(carpetaCon(t, map[string]string{"logo.png": "\x89PNG\x00\n"}))
	if a == b {
		t.Fatal("un archivo binario (con NUL) no se normaliza")
	}
}

func TestHuellaDe_RecetaConCRLFIgual_FND185(t *testing.T) {
	lf := Blueprint{Dir: carpetaCon(t, map[string]string{ManifestName: "id: auth\nversion: 1.0.0\nnombre: x\n"})}
	crlf := Blueprint{Dir: carpetaCon(t, map[string]string{ManifestName: "id: auth\r\nversion: 1.0.0\r\nnombre: x\r\n"})}
	a, _ := huellaDe(lf)
	b, _ := huellaDe(crlf)
	if a == "" || a != b {
		t.Fatalf("la huella de certificación tampoco depende del sistema: %q %q", a, b)
	}
}

func TestAvisosDeOrigen_UnaCarpetaLocalSeAvisa_FND185(t *testing.T) {
	locales := []LockEntry{{ID: "auth", Origen: `C:\Users\p\qdd-blueprints-locales`}, {ID: "pagos", Origen: "/home/p/blueprints"}}
	avisos := AvisosDeOrigen(locales)
	if len(avisos) != 2 || !strings.Contains(avisos[0], "auth") || !strings.Contains(avisos[0], "otro PC") {
		t.Fatalf("una ruta local no existe en otro PC: %v", avisos)
	}
	remotos := []LockEntry{{ID: "auth", Origen: "git@github.com:org/qdd-blueprints.git"}, {ID: "b", Origen: "https://github.com/org/x.git"}}
	if avisos := AvisosDeOrigen(remotos); len(avisos) != 0 {
		t.Fatalf("un repositorio no se avisa: %v", avisos)
	}
}
