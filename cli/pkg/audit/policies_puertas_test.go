package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FND-243: el dashboard guarda las políticas con SavePolicies, que reescribe policies.yaml desde QDDPolicies. Sin el
// campo puertas, cambiar cualquier política desde ahí apagaba las puertas sin decirlo.
func TestPolicies_GuardarConservaLasPuertas_FND243(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".qdd"), 0o755)
	os.WriteFile(filepath.Join(dir, ".qdd", "policies.yaml"), []byte("zero_else: true\npuertas:\n  publicar: ia\n"), 0o644)
	p := LoadPolicies(dir)
	p.OWASP = false
	if err := SavePolicies(dir, p); err != nil {
		t.Fatal(err)
	}
	datos, _ := os.ReadFile(filepath.Join(dir, ".qdd", "policies.yaml"))
	if !strings.Contains(string(datos), "puertas:") || !strings.Contains(string(datos), "publicar: ia") {
		t.Fatalf("las puertas siguen después de guardar:\n%s", datos)
	}
	os.WriteFile(filepath.Join(dir, ".qdd", "policies.yaml"), []byte("puertas: {}\n"), 0o644)
	SavePolicies(dir, LoadPolicies(dir))
	datos, _ = os.ReadFile(filepath.Join(dir, ".qdd", "policies.yaml"))
	if !strings.Contains(string(datos), "puertas: {}") {
		t.Fatalf("puertas: {} (activas, sin opciones) sigue igual:\n%s", datos)
	}
}
