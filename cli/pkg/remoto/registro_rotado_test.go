package remoto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Rule: Prohibido usar 'else'
// Sprint 9 (FND-238): sin consola, qdd remoto escribe su registro en un archivo, que no puede crecer sin tope.

func leer(t *testing.T, ruta string) string {
	t.Helper()
	b, _ := os.ReadFile(ruta)
	return string(b)
}

func TestRegistroRotado_AlPasarElTopeGuardaUnaCopiaYEmpiezaDeNuevo_FND238(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "remoto", "correr.log")
	r, err := AbrirRegistroRotado(ruta, 100)
	if err != nil {
		t.Fatal(err)
	}
	uno, dos, tres := strings.Repeat("a", 59)+"\n", strings.Repeat("b", 59)+"\n", strings.Repeat("c", 59)+"\n"
	r.Write([]byte(uno))
	r.Write([]byte(dos))
	if leer(t, ruta) != dos || leer(t, ruta+".1") != uno {
		t.Fatalf("al pasar el tope, lo anterior queda en .1: %q / %q", leer(t, ruta), leer(t, ruta+".1"))
	}
	r.Write([]byte(tres))
	if leer(t, ruta) != tres || leer(t, ruta+".1") != dos {
		t.Fatal("una sola copia: la más vieja se pierde")
	}
	r.Close()
	r, err = AbrirRegistroRotado(ruta, 100)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	r.Write([]byte(uno))
	if leer(t, ruta) != uno || leer(t, ruta+".1") != tres {
		t.Fatal("al volver a abrir cuenta lo que ya tenía el archivo")
	}
}

func TestRegistroRotado_UnaLineaMasGrandeQueElTopeIgualSeEscribe_FND238(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "correr.log")
	r, err := AbrirRegistroRotado(ruta, 10)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	larga := strings.Repeat("x", 30) + "\n"
	if n, err := r.Write([]byte(larga)); err != nil || n != len(larga) || leer(t, ruta) != larga {
		t.Fatalf("no se pierde: %d %v %q", n, err, leer(t, ruta))
	}
}
