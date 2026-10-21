package audit

import (
	"path/filepath"
	"testing"
)

func TestModuleHeart_InvalidRecordAlwaysReported(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".qdd", "project", "modules", "crm.yaml"), "id: crm\nnombre: CRM\nestado: borrador\nrutas: [internal/crm]\n")

	v := RunModuleHeartCheck(dir, false)
	if !hasRule(v, "HEART-01-INVALID-RECORD") {
		t.Fatalf("una ficha incompleta debe reportarse aunque module_heart esté apagada: %v", v)
	}
}

func TestModuleHeart_ZonesOnlyWhenEnforced(t *testing.T) {
	dir := t.TempDir()
	if v := RunModuleHeartCheck(dir, false); len(v) != 0 {
		t.Fatalf("sin fichas y sin module_heart no debe haber violaciones: %v", v)
	}
}
