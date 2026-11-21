package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/puertas"
)

func TestPuertasHook_PreCommitYCommitMsg(t *testing.T) {
	home, repo := proyectoConPuertas(t, "puertas: {}\n")
	t.Setenv("CLAUDECODE", "")
	escribirEnRepo(t, repo, ".qdd/project/findings/FND-001.yaml", "id: FND-001\ntitle: x\nstatus: resolved\ntest_result: TestX\n")
	if codigo, msg := hookPuertas(repo, "pre-commit", ""); codigo != codigoRechazo || !strings.Contains(msg, "qdd audit") {
		t.Fatalf("pre-commit sin audit rechaza con 2: %d %q", codigo, msg)
	}
	marcarAuditado(t, repo)
	if codigo, _ := hookPuertas(repo, "pre-commit", ""); codigo != 0 {
		t.Fatal("pre-commit con el audit al día pasa")
	}
	escribirEnRepo(t, repo, "tmp-msg", "fix: algo\n# Please enter the commit message for your changes. FND-001\n")
	if codigo, msg := hookPuertas(repo, "commit-msg", filepath.Join(repo, "tmp-msg")); codigo != codigoRechazo || !strings.Contains(msg, "finding") {
		t.Fatalf("commit-msg: un FND en un comentario del editor no cuenta: %d %q", codigo, msg)
	}
	escribirEnRepo(t, repo, "tmp-msg", "fix: algo (FND-001)\n")
	if codigo, _ := hookPuertas(repo, "commit-msg", filepath.Join(repo, "tmp-msg")); codigo != 0 {
		t.Fatal("commit-msg con su finding pasa")
	}
	regs, _, _ := puertas.LeerRegistro(home, filepath.Base(repo))
	if len(regs) != 4 {
		t.Fatalf("fuera de Claude Code el hook anota cada decisión: %d", len(regs))
	}
	t.Setenv("CLAUDECODE", "1")
	hookPuertas(repo, "pre-commit", "")
	if regs, _, _ := puertas.LeerRegistro(home, filepath.Base(repo)); len(regs) != 4 {
		t.Fatalf("en Claude Code ya la anotó el panel: no se cuenta dos veces: %d", len(regs))
	}
}

func TestDoctorPuertas_ActivaEInstala(t *testing.T) {
	_, repo := proyectoConPuertas(t, "zero_else: true\nowasp: false\n")
	g := buildPuertasChecks(repo)
	if !groupHasFailures(g) || !strings.Contains(g.Items[0].Error, "doctor --fix") {
		t.Fatalf("sin puertas, doctor lo dice y cómo activarlas: %+v", g.Items[0])
	}
	repairPuertas(repo, g)
	if g := buildPuertasChecks(repo); groupHasFailures(g) {
		for _, it := range g.Items {
			t.Errorf("después de --fix: %+v", it)
		}
	}
	p := audit.LoadPolicies(repo)
	if p.Puertas == nil || !p.ZeroElse || p.OWASP {
		t.Fatalf("activar las puertas no cambia las demás políticas: %+v", p)
	}
}

func TestDoctorPuertas_SinPoliticasNoLasInventa(t *testing.T) {
	repo := t.TempDir()
	gitEn(t, repo, "init", "-q")
	g := buildPuertasChecks(repo)
	repairPuertas(repo, g)
	if _, err := os.Stat(filepath.Join(repo, ".qdd", "policies.yaml")); !os.IsNotExist(err) {
		t.Fatal("sin policies.yaml no se crea uno solo con puertas: las demás quedarían apagadas")
	}
}

func TestInit_ProyectoNuevoConPuertas(t *testing.T) {
	repo := t.TempDir()
	gitEn(t, repo, "init", "-q")
	ensureAdopcion(repo, true)
	if p := audit.LoadPolicies(repo); p.Puertas == nil || p.Adopcion != audit.AdopcionGradual {
		t.Fatalf("un proyecto nuevo nace con puertas: %+v", p)
	}
	for _, e := range puertas.EstadoHooks(repo) {
		if e.Estado != puertas.HookAlDia {
			t.Errorf("y con sus hooks: %+v", e)
		}
	}
}

func TestInit_EnUnaSubcarpetaNoTocaLosHooks(t *testing.T) {
	repo := t.TempDir()
	gitEn(t, repo, "init", "-q")
	sub := filepath.Join(repo, "sub")
	os.MkdirAll(sub, 0o755)
	ensureAdopcion(sub, true)
	if _, err := os.Stat(filepath.Join(repo, ".git", "hooks", "pre-commit")); !os.IsNotExist(err) {
		t.Fatal("un proyecto en una subcarpeta no instala los hooks de todo el repositorio")
	}
}

// En el CI (sin husky) TestPipelineExecution corre init y doctor en cli/cmd: los hooks son de la raíz y desde ahí no se
// pueden instalar, así que doctor no los revisa en una subcarpeta (si no, quedaba en rojo para siempre).
func TestDoctorPuertas_EnUnaSubcarpetaNoRevisaLosHooks(t *testing.T) {
	_, repo := proyectoConPuertas(t, "puertas: {}\n")
	sub := filepath.Join(repo, "sub")
	escribirEnRepo(t, sub, ".qdd/policies.yaml", "puertas: {}\n")
	if g := buildPuertasChecks(sub); groupHasFailures(g) || len(g.Items) != 1 {
		t.Fatalf("en una subcarpeta solo revisa la política: %+v", g.Items)
	}
	if g := buildPuertasChecks(repo); !groupHasFailures(g) {
		t.Fatal("en la raíz sí revisa los hooks (aquí faltan)")
	}
}
