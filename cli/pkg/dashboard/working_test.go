package dashboard

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'
// FND-182: el comando que corre se lee de .qdd/working/comando, y del archivo .qdd/working de las versiones anteriores.
func TestLoadWorkingOn_LeeLaMarcaNuevaYLaVieja_FND182(t *testing.T) {
	qdd := t.TempDir()
	if err := os.MkdirAll(filepath.Join(qdd, "working"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(qdd, "working", "comando"), []byte("audit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var r QDDState
	loadWorkingOn(&r, qdd)
	if r.WorkingOn != "audit" {
		t.Fatalf("marca nueva: %q", r.WorkingOn)
	}
	vieja := t.TempDir()
	if err := os.WriteFile(filepath.Join(vieja, "working"), []byte("ciclo"), 0o644); err != nil {
		t.Fatal(err)
	}
	var v QDDState
	loadWorkingOn(&v, vieja)
	if v.WorkingOn != "ciclo" {
		t.Fatalf("marca vieja: %q", v.WorkingOn)
	}
}
