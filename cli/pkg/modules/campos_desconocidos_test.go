package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Regresión FND-099: aprobar con un qdd que no conocía «implementa» reescribió la ficha sin ese campo. Reescribir una
// ficha conserva lo que este qdd no conoce: un qdd más viejo nunca borra lo que escribió uno más nuevo.
func TestAprobar_ConservaLosCamposQueEsteQDDNoConoce(t *testing.T) {
	root := gitRepo(t)
	m := validador()
	for _, r := range m.Rutas {
		put(t, root, filepath.Join(r, "x.go"), "package x\n")
	}
	if _, problems := Save(root, m); len(problems) > 0 {
		t.Fatal(problems)
	}
	path := filepath.Join(root, Dir, m.ID+".yaml")
	data, _ := os.ReadFile(path)
	put(t, root, filepath.Join(Dir, m.ID+".yaml"), string(data)+"de_una_version_futura:\n    - blueprint: auth\n      fuera: {AUTH-01: lo hace otro}\n")
	git(t, root, "add", "-A")
	git(t, root, "commit", "-qm", "x")
	if _, problems := Approve(root, m.ID); len(problems) > 0 {
		t.Fatal(problems)
	}
	data, _ = os.ReadFile(path)
	if !strings.Contains(string(data), "de_una_version_futura:") || !strings.Contains(string(data), "AUTH-01: lo hace otro") || !strings.Contains(string(data), "estado: aprobado") {
		t.Fatalf("aprobar no borra lo que no conoce:\n%s", data)
	}
}
