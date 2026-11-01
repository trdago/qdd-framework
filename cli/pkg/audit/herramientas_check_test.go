package audit

import (
	"path/filepath"
	"strings"
	"testing"
)

// HER-01: una definición inválida o con un secreto no pasa el gate; el ejecutable no se exige (el CI no lo compila).
func TestHerramientasCheck_UnaDefinicionInvalidaNoPasaElGate_HER_C02(t *testing.T) {
	dir := t.TempDir()
	if v := RunHerramientasCheck(dir); len(v) != 0 {
		t.Fatalf("sin herramientas no reporta nada: %v", v)
	}
	valida := "id: sim\nnombre: Sim\ncomando: [./bin/sim]\nacciones:\n  - {id: ver, nombre: Ver, args: [ver]}\nprueba: {args: [--help]}\n"
	writeFile(t, filepath.Join(dir, ".qdd", "herramientas", "sim.yaml"), valida)
	if v := RunHerramientasCheck(dir); len(v) != 0 {
		t.Fatalf("válida, aunque su ejecutable no esté compilado en esta máquina: %v", v)
	}
	writeFile(t, filepath.Join(dir, ".qdd", "herramientas", "sim.yaml"), strings.Replace(valida, "args: [ver]", "args: [ver, --password, hunter2]", 1))
	writeFile(t, filepath.Join(dir, ".qdd", "herramientas", "otra.yaml"), "id: otra\nvalores: {token: x}\n")
	v := RunHerramientasCheck(dir)
	if len(v) != 2 || !hasRule(v, "HER-01-DEFINICION") || len(Bloqueantes(v)) != 2 {
		t.Fatalf("un secreto y un campo desconocido bloquean: %v", v)
	}
	for _, x := range v {
		if strings.Contains(x.Description, "hunter2") {
			t.Fatalf("el gate nunca repite el secreto: %q", x.Description)
		}
	}
}
