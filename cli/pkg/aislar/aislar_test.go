package aislar

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Rule: Prohibido usar 'else'

type pruebasQueDejanUnProceso struct{ dir *string }

// Run simula un pedido que la prueba lanzó y que sigue escribiendo en QDD_HOME un momento después de que las pruebas
// terminaron.
func (p pruebasQueDejanUnProceso) Run() int {
	*p.dir = os.Getenv(Variable)
	go func() {
		time.Sleep(150 * time.Millisecond)
		os.MkdirAll(filepath.Join(*p.dir, "claude", "pedidos"), 0o700)
	}()
	return 0
}

// Lo que escribe un proceso rezagado justo después de las pruebas no queda en el TMPDIR de la máquina (FND-288).
func TestCorrer_BorraSuCarpetaAunqueUnProcesoRezagadoLaVuelvaACrear(t *testing.T) {
	t.Setenv(Variable, "")
	t.Setenv("TMPDIR", t.TempDir())
	var dir string
	if c := Correr(pruebasQueDejanUnProceso{&dir}); c != 0 {
		t.Fatalf("devuelve el código de las pruebas: %d", c)
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("la carpeta de las pruebas quedó en la máquina: %s (%v)", dir, err)
	}
}

type pruebasQueMiran struct{ vio *string }

func (p pruebasQueMiran) Run() int {
	*p.vio = os.Getenv(Variable)
	return 0
}

// Un ayudante (el mismo binario con -test.run=…) usa el QDD_HOME aislado de su padre y no crea otro: si lo matan, no
// deja carpeta (FND-288). Uno ajeno a aislar sí recibe el suyo.
func TestCorrer_UnAyudanteUsaElQDDHomeDeSuPadre(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	padre := filepath.Join(t.TempDir(), prefijo+"123")
	os.Mkdir(padre, 0o700)
	t.Setenv(Variable, padre)
	var vio string
	Correr(pruebasQueMiran{&vio})
	if vio != padre {
		t.Fatalf("el ayudante usa el QDD_HOME de su padre: %q", vio)
	}
	if _, err := os.Stat(padre); err != nil {
		t.Fatal("y no lo borra: es del padre")
	}
	t.Setenv(Variable, t.TempDir())
	Correr(pruebasQueMiran{&vio})
	if !strings.HasPrefix(filepath.Base(vio), prefijo) {
		t.Fatalf("con un QDD_HOME que no es de aislar, crea el suyo: %q", vio)
	}
}
