package cmd

import (
	"os"
	"path/filepath"
)

// Rule: Prohibido usar 'else'
// .qdd/working es la carpeta de trabajo local (flujos y más). El nombre del comando que corre va en working/comando;
// antes iba en working como archivo y tapaba la carpeta cuando un comando salía con os.Exit (FND-182).

func rutaComando(root string) string {
	return filepath.Join(root, ".qdd", "working", "comando")
}

func marcarComando(root, nombre string) {
	if _, err := os.Stat(filepath.Join(root, ".qdd")); err != nil {
		return
	}
	carpeta := filepath.Dir(rutaComando(root))
	if info, err := os.Stat(carpeta); err == nil && !info.IsDir() {
		_ = os.Remove(carpeta)
	}
	if err := os.MkdirAll(carpeta, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(rutaComando(root), []byte(nombre), 0o644)
}

func desmarcarComando(root string) {
	_ = os.Remove(rutaComando(root))
}
