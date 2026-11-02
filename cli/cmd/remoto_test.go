package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/qdd-framework/qdd/pkg/remoto"
	"github.com/qdd-framework/qdd/pkg/remoto/registro"
)

// Rule: Prohibido usar 'else'
// RFC 0013 (rc.1): qdd remoto estado lista los proyectos del PC con su panel, y doctor limpia las rutas muertas.

func proyectoRemoto(t *testing.T, nombre string) string {
	t.Helper()
	base, _ := filepath.EvalSymlinks(t.TempDir())
	root := filepath.Join(base, nombre)
	os.MkdirAll(filepath.Join(root, ".qdd"), 0o755)
	return root
}

func TestRemotoEstado_JSONYUnaLineaPorProyecto(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	a, b := proyectoRemoto(t, "alfa"), proyectoRemoto(t, "beta")
	anotarProyecto(a)
	registro.Anotar(b, time.Now().Add(time.Minute))
	os.RemoveAll(b)
	var out, errs bytes.Buffer
	if c := remotoEstado(&out, &errs, true, time.Now()); c != 0 {
		t.Fatalf("sale con 0: %s", errs.String())
	}
	var e remoto.Estado
	if err := json.Unmarshal(out.Bytes(), &e); err != nil {
		t.Fatalf("JSON: %v\n%s", err, out.String())
	}
	if e.Version != 1 || len(e.Proyectos) != 2 || e.Proyectos[0].Nombre != "beta" || e.Proyectos[0].Error == "" || e.Proyectos[1].Panel == nil {
		t.Fatalf("beta (más reciente) con error, alfa con panel: %+v", e.Proyectos)
	}
	out.Reset()
	remotoEstado(&out, &errs, false, time.Now())
	lineas := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lineas) != 2 || !strings.HasPrefix(lineas[0], "beta · ✕") || !strings.HasPrefix(lineas[1], "alfa · ") {
		t.Fatalf("una línea por proyecto: %q", out.String())
	}
}

func TestRemotoEstado_SinRegistroLoDice(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	var out, errs bytes.Buffer
	if c := remotoEstado(&out, &errs, false, time.Now()); c != 0 || !strings.Contains(out.String(), "Ningún proyecto") {
		t.Fatalf("sin registro: %d %q", c, out.String())
	}
}

func TestDoctor_REM01QuitaLasRutasQueYaNoExisten(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	if g := buildRemotoChecks(); len(g.Items) != 0 {
		t.Fatal("sin registro, el grupo no aparece")
	}
	viva, muerta := proyectoRemoto(t, "viva"), proyectoRemoto(t, "muerta")
	anotarProyecto(viva)
	anotarProyecto(muerta)
	os.RemoveAll(muerta)
	g := buildRemotoChecks()
	if !groupHasFailures(g) || !strings.Contains(g.Items[1].Error, muerta) {
		t.Fatalf("REM-01 la señala: %+v", g.Items[1])
	}
	repairRemoto(g)
	if g := buildRemotoChecks(); groupHasFailures(g) {
		t.Fatalf("--fix la quita: %+v", g.Items)
	}
	reg, _ := registro.Leer()
	if len(reg.Proyectos) != 1 || reg.Proyectos[0].Ruta != viva {
		t.Fatalf("queda la viva: %+v", reg.Proyectos)
	}
}

func TestDoctor_REM02ApartaUnRegistroIlegible(t *testing.T) {
	home := t.TempDir()
	t.Setenv("QDD_HOME", home)
	os.WriteFile(filepath.Join(home, "proyectos.json"), []byte("{roto"), 0o600)
	g := buildRemotoChecks()
	if len(g.Items) != 1 || g.Items[0].Success {
		t.Fatalf("REM-02 falla: %+v", g.Items)
	}
	repairRemoto(g)
	if _, err := os.Stat(filepath.Join(home, "proyectos.json.ilegible")); err != nil {
		t.Fatal("queda apartado, no borrado")
	}
	if g := buildRemotoChecks(); len(g.Items) != 0 {
		t.Fatalf("sin registro legible ni archivo, el grupo no aparece: %+v", g.Items)
	}
}
