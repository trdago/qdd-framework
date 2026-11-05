package audit

import (
	"os"
	"path/filepath"
	"testing"
)

// Rule: Prohibido usar 'else'

func conFinding(t *testing.T, cwd, nombre, contenido string) {
	t.Helper()
	dir := filepath.Join(cwd, ".qdd", "project", "findings")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, nombre), []byte(contenido), 0o644); err != nil {
		t.Fatal(err)
	}
}

func reglasDe(vs []Violation) map[string]string {
	out := map[string]string{}
	for _, v := range vs {
		out[v.RuleID] = v.File
	}
	return out
}

// MEJ-002: un finding sin estado no se cuenta como abierto ni como resuelto, y nadie se enteraba.
func TestRunFindingsCheck_FindingSinEstado(t *testing.T) {
	cwd := t.TempDir()
	conFinding(t, cwd, "FND-001.yaml", "id: FND-001\ntitle: x\ndate: \"2020-01-01\"\n")
	vs := RunFindingsCheck(cwd)
	if reglasDe(vs)["FND-01-SIN-ESTADO"] == "" {
		t.Fatalf("un finding sin status bloquea: %+v", vs)
	}
	if len(Bloqueantes(vs)) != 1 {
		t.Fatalf("bloquea: %+v", vs)
	}
}

func TestRunFindingsCheck_EstadoNoReconocido(t *testing.T) {
	cwd := t.TempDir()
	conFinding(t, cwd, "FND-001.yaml", "id: FND-001\nstatus: quizas\n")
	if reglasDe(RunFindingsCheck(cwd))["FND-01-SIN-ESTADO"] == "" {
		t.Fatal("un status que no se reconoce es como no tenerlo")
	}
}

func TestRunFindingsCheck_ResueltoNuevoSinPrueba(t *testing.T) {
	cwd := t.TempDir()
	conFinding(t, cwd, "FND-001.yaml", "id: FND-001\nstatus: resolved\ndate: \"2026-11-06\"\n")
	if reglasDe(RunFindingsCheck(cwd))["FND-02-SIN-PRUEBA"] == "" {
		t.Fatal("un resuelto nuevo dice qué prueba lo cubre")
	}
}

// El historial no se castiga: en los proyectos de la máquina hay más de 200 resueltos antiguos sin test_result.
func TestRunFindingsCheck_ResueltoAntiguoSinPruebaNoBloquea(t *testing.T) {
	cwd := t.TempDir()
	conFinding(t, cwd, "FND-001.yaml", "id: FND-001\nstatus: resolved\ndate: \"2026-11-04\"\n")
	conFinding(t, cwd, "FND-002.yaml", "id: FND-002\nstatus: resolved\n")
	conFinding(t, cwd, "FND-003.yaml", "id: FND-003\nstatus: resolved\ndate: 2026-10-24\n")
	if vs := RunFindingsCheck(cwd); len(vs) != 0 {
		t.Fatalf("antes de la regla, o sin fecha, no se exige: %+v", vs)
	}
}

func TestRunFindingsCheck_LoSanoNoReportaNada(t *testing.T) {
	cwd := t.TempDir()
	conFinding(t, cwd, "FND-001.yaml", "id: FND-001\nstatus: resolved\ndate: \"2026-12-01\"\ntest_result: \"pkg/x_test.go\"\n")
	conFinding(t, cwd, "FND-002.yaml", "id: FND-002\nstatus: open\ndate: \"2026-12-01\"\n")
	conFinding(t, cwd, "FND-003.yaml", "id: FND-003\nstatus: wontfix\ndate: \"2026-12-01\"\n")
	if vs := RunFindingsCheck(cwd); len(vs) != 0 {
		t.Fatalf("abierto o descartado no piden prueba: %+v", vs)
	}
}

func TestRunFindingsCheck_TestResultVacioNoCuenta(t *testing.T) {
	cwd := t.TempDir()
	conFinding(t, cwd, "FND-001.yaml", "id: FND-001\nstatus: resolved\ndate: \"2026-12-01\"\ntest_result: \"  \"\n")
	if reglasDe(RunFindingsCheck(cwd))["FND-02-SIN-PRUEBA"] == "" {
		t.Fatal("vacío es no tenerlo")
	}
}

func TestRunFindingsCheck_SinCarpetaNoReportaNada(t *testing.T) {
	if vs := RunFindingsCheck(t.TempDir()); len(vs) != 0 {
		t.Fatalf("sin findings no hay nada que revisar: %+v", vs)
	}
}
