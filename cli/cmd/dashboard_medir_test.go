package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

// FND-051: al abrir el dashboard sin certificación vigente se mide una vez,
// para mostrar el puntaje real en vez de "Sin medir"; si ya hay, no se repite.
func TestMedirSiFalta(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	defer os.Chdir(orig)
	os.Chdir(dir)

	llamadas := 0
	certificar := func() { llamadas++ }

	if !medirSiFalta(dir, certificar) || llamadas != 1 {
		t.Fatalf("sin certificación debe medir una vez (llamadas = %d)", llamadas)
	}

	historial := filepath.Join(dir, ".qdd", "project", "metrics", "certificate_history.json")
	os.MkdirAll(filepath.Dir(historial), 0755)
	os.WriteFile(historial, []byte(`[{"score":94,"escala":2,"tendency":"Estable"}]`), 0644)

	if medirSiFalta(dir, certificar) || llamadas != 1 {
		t.Errorf("con certificación vigente no debe volver a medir (llamadas = %d)", llamadas)
	}
}
