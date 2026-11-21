package cmd

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'

// VarHuellasPrivadas apunta a un archivo fuera del repositorio con el sha256 (hex, una por línea) de cada nombre real
// que no puede aparecer en la documentación pública: la lista de nombres sería ella misma la filtración.
const VarHuellasPrivadas = "QDD_HUELLAS_PRIVADAS"

var palabraDeDoc = regexp.MustCompile(`[a-z0-9][a-z0-9_-]*`)

// Ningún ejemplo de docs/ ni de rfcs/ nombra una organización, un cliente o un proyecto real. Sin el archivo de huellas
// (en CI) no hay con qué comparar y se salta; en el PC de la persona lo pasa «make test».
// contrato: documentacion#d7c8baeb
func TestDocs_NoNombranProyectosReales(t *testing.T) {
	ruta := os.Getenv(VarHuellasPrivadas)
	if ruta == "" {
		t.Skip(VarHuellasPrivadas + " vacía: sin las huellas de los nombres reales no hay con qué comparar")
	}
	huellas := leerHuellas(t, ruta)
	for _, carpeta := range []string{"../../docs", "../../rfcs"} {
		filepath.WalkDir(carpeta, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".md") {
				return nil
			}
			for n, linea := range lineasDe(t, p) {
				for _, w := range palabraDeDoc.FindAllString(strings.ToLower(linea), -1) {
					if huellas[huella(w)] {
						t.Errorf("%s:%d nombra un proyecto real (huella %s)", p, n+1, huella(w)[:8])
					}
				}
			}
			return nil
		})
	}
}

func TestDocs_LasHuellasSeComparanPorPalabra(t *testing.T) {
	dir := t.TempDir()
	h := filepath.Join(dir, "huellas")
	os.WriteFile(h, []byte(huella("mi-cliente")+"\n\n# comentario\n"), 0o600)
	huellas := leerHuellas(t, h)
	if !huellas[huella("mi-cliente")] || len(huellas) != 1 {
		t.Fatalf("una huella por línea, sin vacías ni comentarios: %v", huellas)
	}
	palabras := palabraDeDoc.FindAllString(strings.ToLower("Probado en Mi-Cliente (rama mi-cliente-2)."), -1)
	if !contienePalabra(palabras, "mi-cliente") || contienePalabra(palabras, "mi") {
		t.Fatalf("separa por palabra completa: %v", palabras)
	}
}

func contienePalabra(ws []string, w string) bool {
	for _, x := range ws {
		if x == w {
			return true
		}
	}
	return false
}

func huella(w string) string {
	s := sha256.Sum256([]byte(w))
	return hex.EncodeToString(s[:])
}

func leerHuellas(t *testing.T, ruta string) map[string]bool {
	t.Helper()
	f, err := os.Open(ruta)
	if err != nil {
		t.Fatalf("%s: %v", VarHuellasPrivadas, err)
	}
	defer f.Close()
	huellas := map[string]bool{}
	s := bufio.NewScanner(f)
	for s.Scan() {
		l := strings.TrimSpace(s.Text())
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		huellas[strings.ToLower(l)] = true
	}
	return huellas
}

func lineasDe(t *testing.T, p string) []string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(string(b), "\n")
}
