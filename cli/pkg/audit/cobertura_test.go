package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/qdd-framework/qdd/pkg/contexto"
)

func proyecto(t *testing.T, archivos map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, contenido := range archivos {
		p := filepath.Join(dir, rel)
		os.MkdirAll(filepath.Dir(p), 0755)
		os.WriteFile(p, []byte(contenido), 0644)
	}
	return dir
}

func coberturaDe(t *testing.T, c Cobertura, id string) CoberturaPrincipio {
	t.Helper()
	for _, cp := range c.Principios {
		if cp.Principio.ID == id {
			return cp
		}
	}
	t.Fatalf("la cobertura no incluye %s", id)
	return CoberturaPrincipio{}
}

// RFC 0005 §3.4: el detector de inyección solo lee Go. En un proyecto Node no
// revisa nada; antes eso se veía como "sin violaciones" y aprobaba en silencio.
func TestCobertura_SinDetectorParaElStackEsNoEvaluado(t *testing.T) {
	dir := proyecto(t, map[string]string{"package.json": `{"name":"web"}`})
	c := CalcularCobertura(dir, DefaultPolicies(), contexto.Leer(dir), nil)

	cp := coberturaDe(t, c, "P-NO-INJECTION")
	if cp.Estado != NoEvaluado {
		t.Fatalf("P-NO-INJECTION en Node = %s; want no_evaluado", cp.Estado)
	}
	if len(cp.NoEvaluadas) != 1 || cp.NoEvaluadas[0].Ruta != "." || !strings.Contains(cp.NoEvaluadas[0].Motivo, "node") {
		t.Errorf("NoEvaluadas = %+v; want la raíz, con el motivo 'sin detector para node'", cp.NoEvaluadas)
	}
}

func TestCobertura_MonorepoEsParcial(t *testing.T) {
	dir := proyecto(t, map[string]string{
		"backend/go.mod":        "module backend\n",
		"frontend/package.json": `{"name":"web"}`,
	})
	c := CalcularCobertura(dir, DefaultPolicies(), contexto.Leer(dir), nil)

	cp := coberturaDe(t, c, "P-NO-INJECTION")
	if cp.Estado != Parcial {
		t.Fatalf("P-NO-INJECTION = %s; want parcial", cp.Estado)
	}
	if len(cp.Evaluadas) != 1 || cp.Evaluadas[0] != "backend (go)" {
		t.Errorf("Evaluadas = %v; want [backend (go)]", cp.Evaluadas)
	}
	if len(cp.NoEvaluadas) != 1 || cp.NoEvaluadas[0].Ruta != "frontend" {
		t.Errorf("NoEvaluadas = %+v; want frontend", cp.NoEvaluadas)
	}
}

func TestCobertura_PoliticaApagadaEsDesactivado(t *testing.T) {
	dir := proyecto(t, map[string]string{"go.mod": "module x\n"})
	p := DefaultPolicies()
	p.OWASP = false
	c := CalcularCobertura(dir, p, contexto.Leer(dir), nil)

	if cp := coberturaDe(t, c, "P-NO-INJECTION"); cp.Estado != Desactivado {
		t.Errorf("P-NO-INJECTION con owasp: false = %s; want desactivado", cp.Estado)
	}
	if cp := coberturaDe(t, c, "P-READABLE-FLOW"); cp.Estado != Evaluado {
		t.Errorf("P-READABLE-FLOW en Go = %s; want evaluado", cp.Estado)
	}
}

// Las pruebas de una unidad se cuentan como evaluadas solo si el gate ejecuta
// un comando en esa unidad. Con test_commands que solo entran a cli/, las
// pruebas de cli/ui no corren en el gate.
func TestCobertura_PruebasPorUnidad(t *testing.T) {
	dir := proyecto(t, map[string]string{
		"cli/go.mod":          "module cli\n",
		"cli/ui/package.json": `{"name":"ui","scripts":{"test":"vitest run"}}`,
		".qdd/config.yaml":    "test_commands:\n  - cd cli && go test ./...\n",
	})
	c := CalcularCobertura(dir, DefaultPolicies(), contexto.Leer(dir), nil)

	cp := coberturaDe(t, c, "P-TESTS-PER-UNIT")
	if cp.Estado != Parcial {
		t.Fatalf("P-TESTS-PER-UNIT = %s; want parcial", cp.Estado)
	}
	if len(cp.Evaluadas) != 1 || cp.Evaluadas[0] != "cli" {
		t.Errorf("Evaluadas = %v; want [cli]", cp.Evaluadas)
	}
	if len(cp.NoEvaluadas) != 1 || cp.NoEvaluadas[0].Ruta != "cli/ui" || !strings.Contains(cp.NoEvaluadas[0].Motivo, "test_commands") {
		t.Errorf("NoEvaluadas = %+v; want cli/ui, con qué declarar", cp.NoEvaluadas)
	}
}

func TestCobertura_Resumen(t *testing.T) {
	dir := proyecto(t, map[string]string{
		"backend/go.mod":        "module backend\n",
		"frontend/package.json": `{"name":"web"}`,
	})
	p := DefaultPolicies()
	p.AIGovernance = false
	viol := []Violation{{RuleID: "CLEAN-01-NO-ELSE"}, {RuleID: "CLEAN-01-NO-ELSE"}}
	c := CalcularCobertura(dir, p, contexto.Leer(dir), viol)

	if c.Aplicables+c.Desactivados != len(Principios()) {
		t.Errorf("aplicables %d + desactivados %d != %d principios", c.Aplicables, c.Desactivados, len(Principios()))
	}
	if c.Evaluados+c.Parciales+c.NoEvaluados != c.Aplicables {
		t.Errorf("evaluados %d + parciales %d + no evaluados %d != aplicables %d", c.Evaluados, c.Parciales, c.NoEvaluados, c.Aplicables)
	}
	if c.Desactivados == 0 {
		t.Error("con ai_governance: false debería haber principios desactivados")
	}
	if cp := coberturaDe(t, c, "P-READABLE-FLOW"); cp.Violaciones != 2 {
		t.Errorf("P-READABLE-FLOW violaciones = %d; want 2", cp.Violaciones)
	}
}

// La cobertura solo informa: no cambia el puntaje ni lo que bloquea (2.2.0).
func TestEvaluar_LaCoberturaNoCambiaElPuntaje(t *testing.T) {
	dir := proyecto(t, map[string]string{
		"package.json":     `{"name":"web"}`,
		".qdd/config.yaml": "test_commands:\n  - \"true\"\n",
	})
	e := NewEngine(dir)
	r := e.Evaluar()
	if Score(r.Violaciones) != Score(e.RunAll()) {
		t.Errorf("Evaluar cambió el puntaje")
	}
	if r.Cobertura.NoEvaluados == 0 {
		t.Error("un proyecto Node debería tener principios no evaluados")
	}
}

// FND-150: sin esta clave, P-SEMVER-RELEASES saldría desactivado aunque la política esté activa.
func TestCobertura_LaPoliticaDeVersionesDecideSiSeRevisa_FND150(t *testing.T) {
	apagada := false
	if !politicaActiva(QDDPolicies{}, "versioning") {
		t.Error("sin declararla, la política de versiones está activa, igual que en el motor")
	}
	if politicaActiva(QDDPolicies{Versioning: &apagada}, "versioning") {
		t.Error("apagada, el principio de versiones sale como desactivado")
	}
}
