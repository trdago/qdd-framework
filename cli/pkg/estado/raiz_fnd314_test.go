package estado

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'

// FND-314: un repositorio con su propio .qdd, sin tablero todavía, dentro de otro proyecto escribía en el tablero de
// afuera. Un worktree (.git es un archivo) sí usa el de su checkout principal.
func TestBuscarRaiz_UnRepositorioPropioNoUsaElTableroDeAfuera_FND314(t *testing.T) {
	afuera := proyectoConTablero(t)
	casos := []struct {
		nombre  string
		prepara func(dir string)
		quiere  string
	}{
		{"repositorio propio", func(d string) { carpetas(t, filepath.Join(d, ".git"), filepath.Join(d, ".qdd")) }, ""},
		{"worktree", func(d string) {
			carpetas(t, filepath.Join(d, ".qdd"))
			archivo(t, filepath.Join(d, ".git"))
		}, afuera},
		{"subcarpeta", func(string) {}, afuera},
		{"repositorio sin QDD", func(d string) { carpetas(t, filepath.Join(d, ".git")) }, afuera},
	}
	for _, c := range casos {
		t.Run(c.nombre, func(t *testing.T) {
			dir := filepath.Join(afuera, c.nombre)
			hondo := filepath.Join(dir, "src")
			carpetas(t, hondo)
			c.prepara(dir)
			if got := BuscarRaiz(hondo); got != c.quiere {
				t.Errorf("BuscarRaiz = %q, se esperaba %q", got, c.quiere)
			}
		})
	}
}

func TestBuscarRaiz_ElRepositorioConTableroEsSuRaiz_FND314(t *testing.T) {
	raiz := proyectoConTablero(t)
	carpetas(t, filepath.Join(raiz, ".git"), filepath.Join(raiz, "src"))
	if got := BuscarRaiz(filepath.Join(raiz, "src")); got != raiz {
		t.Errorf("BuscarRaiz = %q, se esperaba %q", got, raiz)
	}
}

func proyectoConTablero(t *testing.T) string {
	t.Helper()
	raiz := t.TempDir()
	carpetas(t, filepath.Dir(filepath.Join(raiz, Ruta)))
	archivo(t, filepath.Join(raiz, Ruta))
	return raiz
}

func carpetas(t *testing.T, rutas ...string) {
	t.Helper()
	for _, r := range rutas {
		if err := os.MkdirAll(r, 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

func archivo(t *testing.T, ruta string) {
	t.Helper()
	if err := os.WriteFile(ruta, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}
