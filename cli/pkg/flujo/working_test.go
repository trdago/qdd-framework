package flujo

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-182 (MEJ-036, MEJ-037): qdd dejaba en .qdd/working un archivo con el nombre del último comando, y los flujos usan
// .qdd/working/flujos como carpeta. Con ese archivo, crear, editar y comentar fallaban con «not a directory» y el
// comentario de la persona se perdía.
func TestEditar_ConElMarcadorViejoEnWorkingSigueFuncionando_FND182(t *testing.T) {
	root := flujoGuardado(t, cadena()...)
	marcador := filepath.Join(root, ".qdd", "working")
	if err := os.RemoveAll(marcador); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(marcador, []byte("ciclo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Editar(root, "pago", DePersona, ahoraDePrueba, Comentar("b", "¿y esto?", DePersona, ahoraDePrueba)); err != nil {
		t.Fatalf("el comentario se guarda aunque quede el marcador viejo: %v", err)
	}
	if info, err := os.Stat(marcador); err != nil || !info.IsDir() {
		t.Fatalf(".qdd/working vuelve a ser una carpeta: %v %v", info, err)
	}
}
