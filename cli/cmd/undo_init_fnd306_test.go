package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/gitenv"
)

// Rule: Prohibido usar 'else'
// FND-306: 'qdd undo init' (y el rollback de 'qdd run init …' cuando falla un comando siguiente) borraba .qdd entero:
// fichas, findings y decisiones de un proyecto que ya existía.

func proyectoConQdd(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, ruta := range []string{".qdd/config.yaml", ".qdd/project/evidence/doctor/report_1.md", ".qdd/project/metrics/cognitive_history.json"} {
		dejarArchivo(t, filepath.Join(dir, ruta))
	}
	return dir
}

func dejarArchivo(t *testing.T, ruta string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestUndoInit_BorraLoQueAcabaDeCrearInit_FND306(t *testing.T) {
	dir := proyectoConQdd(t)
	if motivo, err := deshacerInit(dir); motivo != "" || err != nil {
		t.Fatalf("un init recién hecho se deshace: %q %v", motivo, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".qdd")); err == nil {
		t.Fatal(".qdd sigue ahí")
	}
}

func TestUndoInit_NoBorraConocimiento_FND306(t *testing.T) {
	for _, ruta := range []string{
		".qdd/project/findings/FND-001.yaml", ".qdd/project/modules/pagos.yaml", ".qdd/project/estado.yaml",
		".qdd/project/sprints/sprint-1.md", ".qdd/project/metrics/certificate_history.json", ".qdd/blueprints.lock.yaml", ".qdd/herramientas/x.yaml",
	} {
		dir := proyectoConQdd(t)
		dejarArchivo(t, filepath.Join(dir, ruta))
		motivo, _ := deshacerInit(dir)
		if !strings.Contains(motivo, filepath.FromSlash(ruta)) {
			t.Errorf("%s: el motivo nombra lo que se perdería, dice %q", ruta, motivo)
		}
		if _, err := os.Stat(filepath.Join(dir, ruta)); err != nil {
			t.Errorf("%s se borró", ruta)
		}
	}
}

func TestUndoInit_NoBorraLoQueGitYaSigue_FND306(t *testing.T) {
	dir := proyectoConQdd(t)
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		c.Env = gitenv.Limpio(os.Environ())
		if salida, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, salida)
		}
	}
	if motivo, _ := deshacerInit(dir); !strings.Contains(motivo, "git") {
		t.Fatalf("un .qdd que git sigue no es de este init: %q", motivo)
	}
	if _, err := os.Stat(filepath.Join(dir, ".qdd", "config.yaml")); err != nil {
		t.Fatal("se borró un .qdd versionado")
	}
}

func TestUndoInit_SinQddNoHaceNada_FND306(t *testing.T) {
	if motivo, err := deshacerInit(t.TempDir()); motivo != "" || err != nil {
		t.Fatalf("sin .qdd no hay qué deshacer: %q %v", motivo, err)
	}
}
