package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/puertas"
)

// contrato: puertas#4ad3e5ef
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

// FND-325 (MEJ-102): un proyecto de QDD sin policies.yaml quedaba en un bucle: doctor --fix mandaba a qdd init e init
// a sí mismo. --fix lo crea con las políticas que ya regían (las por defecto, sin adopción gradual) y las puertas.
func TestDoctorPuertas_SinPoliticasLasCreaConLasQueYaRegian_FND325(t *testing.T) {
	t.Setenv("QDD_HOME", t.TempDir())
	repo := t.TempDir()
	gitEn(t, repo, "init", "-q")
	escribirEnRepo(t, repo, ".qdd/config.yaml", "languages: [Go]\n")
	antes := audit.LoadPolicies(repo)
	repairPuertas(repo, buildPuertasChecks(repo))
	if g := buildPuertasChecks(repo); groupHasFailures(g) {
		t.Fatalf("después de --fix las puertas quedan activas y con sus hooks: %+v", g.Items[0])
	}
	despues := audit.LoadPolicies(repo)
	if despues.Puertas == nil {
		t.Fatal("--fix activa las puertas")
	}
	despues.Puertas = nil
	if !reflect.DeepEqual(antes, despues) {
		t.Fatalf("las demás políticas quedan como regían sin el archivo:\nantes   %+v\ndespués %+v", antes, despues)
	}
}

// FND-325: una carpeta sin QDD no recibe un .qdd inventado por doctor: ahí sí corresponde qdd init.
func TestDoctorPuertas_SinQDDNoInventaPoliticas(t *testing.T) {
	repo := t.TempDir()
	gitEn(t, repo, "init", "-q")
	repairPuertas(repo, buildPuertasChecks(repo))
	if _, err := os.Stat(filepath.Join(repo, ".qdd")); !os.IsNotExist(err) {
		t.Fatal("sin .qdd/config.yaml doctor no crea .qdd")
	}
}

func TestInit_ProyectoNuevoConPuertas(t *testing.T) {
	repo := t.TempDir()
	gitEn(t, repo, "init", "-q")
	ensureAdopcion(repo, true)
	if p := audit.LoadPolicies(repo); p.Puertas == nil || p.Adopcion != audit.AdopcionGradual || p.GoSinPruebas != audit.GoSinPruebasBloquea {
		t.Fatalf("un proyecto nuevo nace con puertas y exige pruebas en Go (FND-309): %+v", p)
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
