package mejoras

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRevisar_LegibleTopeYTareas(t *testing.T) {
	root := proyecto(t)
	if r := Revisar(root); r.Hay {
		t.Fatalf("sin cola no hay nada que revisar: %+v", r)
	}
	reportar(t, root, "Primera mejora", t0)
	reportar(t, root, "Segunda mejora", t0.Add(time.Second))
	ciclo(t, root, sinGitHub(), t0.Add(time.Minute))
	r := Revisar(root)
	if !r.Hay || r.Ilegible != nil || r.EnTarea != 2 || len(r.SinTarea) != 0 {
		t.Fatalf("cola sana: %+v", r)
	}
	if err := borrarTarea(root, mejora(t, root, "MEJ-001").Tarea); err != nil {
		t.Fatal(err)
	}
	if r = Revisar(root); strings.Join(r.SinTarea, ",") != "MEJ-001" {
		t.Fatalf("una mejora en tarea sin su tarea en el tablero: %+v", r)
	}
	if err := os.WriteFile(filepath.Join(root, Dir, "MEJ-009-rota.yaml"), []byte("id: [rota"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r = Revisar(root); r.Ilegible == nil || !strings.Contains(r.Ilegible.Error(), "MEJ-009") {
		t.Fatalf("un archivo ilegible dice cuál: %+v", r)
	}
}
