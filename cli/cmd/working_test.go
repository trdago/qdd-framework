package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-182: el nombre del comando que corre va dentro de .qdd/working, no en su lugar: esa ruta es la carpeta de
// trabajo local (flujos y más). Un archivo que dejó una versión anterior se reemplaza por la carpeta.
func TestMarcarComando_VaDentroDeWorkingYReemplazaElArchivoViejo_FND182(t *testing.T) {
	root := t.TempDir()
	viejo := filepath.Join(root, ".qdd", "working")
	if err := os.MkdirAll(filepath.Dir(viejo), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(viejo, []byte("ciclo"), 0o644); err != nil {
		t.Fatal(err)
	}
	marcarComando(root, "estado")
	b, err := os.ReadFile(filepath.Join(viejo, "comando"))
	if err != nil || string(b) != "estado" {
		t.Fatalf("el comando queda en .qdd/working/comando: %q %v", b, err)
	}
	desmarcarComando(root)
	if _, err := os.Stat(filepath.Join(viejo, "comando")); !os.IsNotExist(err) {
		t.Fatalf("al terminar se borra: %v", err)
	}
	if info, err := os.Stat(viejo); err != nil || !info.IsDir() {
		t.Fatal("y la carpeta queda")
	}
}

func TestMarcarComando_FueraDeUnProyectoNoCreaNada(t *testing.T) {
	root := t.TempDir()
	marcarComando(root, "estado")
	if _, err := os.Stat(filepath.Join(root, ".qdd")); !os.IsNotExist(err) {
		t.Fatalf("sin .qdd no se crea: %v", err)
	}
}
