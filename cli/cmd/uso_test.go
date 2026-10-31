package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/uso"
)

// Rule: Prohibido usar 'else'
// Uso medido (pedido del usuario, 2026-10-30): qdd anota su propio uso real en los proyectos QDD y 'qdd uso' lo muestra.

func TestRegistrarUso_SoloEnProyectosQDD(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	proyecto, suelto := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(proyecto, ".qdd"), 0o755)
	t0 := time.Date(2026, 10, 30, 18, 0, 0, 0, time.UTC)
	for _, m := range []int{0, 2} {
		registrarUsoEn("v2.8.0-rc.1", proyecto, t0.Add(time.Duration(m)*time.Minute))
		registrarUsoEn("v2.8.0-rc.1", suelto, t0.Add(time.Duration(m)*time.Minute+time.Second))
	}
	u, err := uso.Leer("2.8.0-rc.1")
	if err != nil || len(u.Proyectos) != 1 || u.Proyectos[filepath.Base(proyecto)] == 0 {
		t.Fatalf("solo cuenta el uso en un proyecto QDD: %+v %v", u, err)
	}
	var out bytes.Buffer
	imprimirUso(&out, "v2.8.0-rc.1", 5)
	if !strings.Contains(out.String(), "2.8.0-rc.1") || !strings.Contains(out.String(), "de 5 h") {
		t.Fatalf("qdd uso muestra lo medido y la meta para pasar a latest: %s", out.String())
	}
}
