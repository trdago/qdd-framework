package modules

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git no disponible")
	}
	root := t.TempDir()
	git(t, root, "init", "-q")
	git(t, root, "config", "user.email", "t@t")
	git(t, root, "config", "user.name", "t")
	return root
}

func git(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func put(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func validador() Module {
	return Module{
		ID:     "validador",
		Nombre: "Validador de ofertas",
		Rutas:  []string{"internal/validador"},
		ObjetivoOperativo: ObjetivoOperativo{
			Problema:        "Decidir si una oferta cumple los requisitos de la licitación",
			CriterioDeExito: "Coincide con el oro congelado en ≥95% de los casos",
		},
		ObjetivoTecnologico: ObjetivoTecnologico{
			Arquitectura: "Reglas deterministas en Go + LLM solo para extracción",
			Invariantes:  []string{"Si la oferta calla, el resultado es NO_CUMPLE"},
		},
		Pruebas: Pruebas{Comandos: []string{"go test ./internal/validador/..."}},
	}
}

func TestSave_ValidatesAndAlwaysWritesDraft(t *testing.T) {
	root := gitRepo(t)
	put(t, root, "internal/validador/v.go", "package validador\n")

	m := validador()
	m.Estado = EstadoAprobado // una IA no puede auto-aprobar
	path, problems := Save(root, m)
	if len(problems) > 0 {
		t.Fatalf("ficha válida rechazada: %v", problems)
	}
	mods, errs := Load(root)
	if len(errs) > 0 || len(mods) != 1 {
		t.Fatalf("Load = %v, %v", mods, errs)
	}
	if mods[0].Estado != EstadoBorrador || mods[0].File != path {
		t.Errorf("Save debe guardar en borrador: %+v", mods[0])
	}
}

func TestSave_RejectsIncompleteOrUnsafeRecords(t *testing.T) {
	root := gitRepo(t)
	m := Module{ID: "../escape", Rutas: []string{"../otro-repo", "no/existe"}}
	_, problems := Save(root, m)
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"id:", "nombre:", "objetivo_operativo.problema", "criterio_de_exito", "arquitectura", "pruebas:", "'../otro-repo' debe ser relativa", "'no/existe' no existe"} {
		if !strings.Contains(joined, want) {
			t.Errorf("faltó el problema %q en:\n%s", want, joined)
		}
	}
	if _, err := os.Stat(filepath.Join(root, Dir)); err == nil {
		t.Error("una ficha inválida no debe escribirse")
	}
}

func TestApproveAndDrift(t *testing.T) {
	root := gitRepo(t)
	put(t, root, "internal/validador/v.go", "package validador\n")
	put(t, root, "internal/otro/o.go", "package otro\n")
	git(t, root, "add", ".")
	git(t, root, "commit", "-qm", "init")

	if _, problems := Save(root, validador()); len(problems) > 0 {
		t.Fatal(problems)
	}
	m, problems := Approve(root, "validador")
	if len(problems) > 0 {
		t.Fatal(problems)
	}
	if m.Estado != EstadoAprobado || m.RevisadoEnCommit == "" || Drift(root, m) != 0 {
		t.Fatalf("aprobación incompleta: %+v drift=%d", m, Drift(root, m))
	}

	put(t, root, "internal/otro/o.go", "package otro\n// cambio ajeno\n")
	git(t, root, "commit", "-qam", "cambio en otro módulo")
	if d := Drift(root, m); d != 0 {
		t.Errorf("un cambio en otro módulo no es desfase del validador: drift=%d", d)
	}

	put(t, root, "internal/validador/v.go", "package validador\n// cambio\n")
	git(t, root, "commit", "-qam", "cambio en el validador")
	if d := Drift(root, m); d != 1 {
		t.Errorf("drift = %d, want 1", d)
	}
	if !strings.Contains(statusWarning(root, m), "cambió en 1 commit") {
		t.Errorf("el contexto debe advertir el desfase: %q", statusWarning(root, m))
	}
}

func TestUncoveredZones(t *testing.T) {
	root := gitRepo(t)
	for _, f := range []string{
		"internal/validador/v.go", "internal/validador/v_test.go", "internal/crm/c.go",
		"frontend/src/views/Home.vue", "frontend/src/main.ts", "scripts/tool.go",
		"docs/x.go", ".github/x.go", "main.go", "internal/api/handlers/h.go", "internal/api/server.go",
	} {
		put(t, root, f, "x")
	}
	git(t, root, "add", ".")

	got := UncoveredZones(root, []Module{validador()})
	want := []string{"frontend/src", "internal/api", "internal/crm"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("UncoveredZones = %v, want %v", got, want)
	}
}

func TestBuildContext_ShowsHeartOfTouchedModuleAndWarnsUncovered(t *testing.T) {
	root := gitRepo(t)
	put(t, root, "internal/validador/v.go", "package validador\n")
	put(t, root, "internal/crm/c.go", "package crm\n")
	put(t, root, ".qdd/understanding.json", `{"summary":"Notifica licitaciones","objectives":["Descubrir oportunidades"]}`)
	git(t, root, "add", ".")
	if _, problems := Save(root, validador()); len(problems) > 0 {
		t.Fatal(problems)
	}

	ctx := BuildContext(ContextInput{
		Root:         root,
		Paths:        []string{"internal/validador/v.go", filepath.Join(root, "internal/crm/c.go")},
		TestCommands: []string{"go test ./..."},
	})
	for _, want := range []string{
		"Notifica licitaciones", "Descubrir oportunidades", "Pruebas primero", "go test ./...",
		"Validador de ofertas", "Si la oferta calla, el resultado es NO_CUMPLE", "Ficha en borrador",
		"`internal/crm/c.go` no pertenece a ningún módulo con ficha", "Zonas de código sin ficha (1)",
	} {
		if !strings.Contains(ctx, want) {
			t.Errorf("el contexto no contiene %q:\n%s", want, ctx)
		}
	}
}
