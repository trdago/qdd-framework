package audit

import (
	"os"
	"path/filepath"
	"reflect"
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
// contrato: cobertura-auditoria#b3ca6f36
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

// FND-225: con una regla que ya bloquea, qdd audit no corre las pruebas (QA-04-PRUEBAS-PENDIENTES, RFC 0020). La
// cobertura no puede decir entonces que las pruebas de cada unidad se revisaron.
// contrato: gate-pruebas#9bdb6c50
func TestCobertura_PruebasPendientesNoSonEvaluadas_FND225(t *testing.T) {
	dir := proyecto(t, map[string]string{
		"go.mod":           "module x\n",
		".qdd/config.yaml": "test_commands:\n  - go test ./...\n",
	})
	pendientes := []Violation{{Category: "COVERAGE", RuleID: ReglaPruebasPendientes, Recomendacion: true}}
	c := CalcularCobertura(dir, DefaultPolicies(), contexto.Leer(dir), pendientes)

	cp := coberturaDe(t, c, "P-TESTS-PER-UNIT")
	if cp.Estado != NoEvaluado {
		t.Fatalf("P-TESTS-PER-UNIT con las pruebas pendientes = %s; want no_evaluado", cp.Estado)
	}
	if len(cp.NoEvaluadas) != 1 || !strings.Contains(cp.NoEvaluadas[0].Motivo, "--completo") {
		t.Errorf("NoEvaluadas = %+v; want la raíz, diciendo que no se corrieron y cómo verlas (--completo)", cp.NoEvaluadas)
	}
	if cp.Violaciones != 0 {
		t.Errorf("Violaciones = %d; una recomendación de pruebas pendientes no es una violación del principio", cp.Violaciones)
	}

	if cp := coberturaDe(t, CalcularCobertura(dir, DefaultPolicies(), contexto.Leer(dir), nil), "P-TESTS-PER-UNIT"); cp.Estado != Evaluado {
		t.Errorf("sin pruebas pendientes P-TESTS-PER-UNIT = %s; want evaluado", cp.Estado)
	}
}

// contrato: cobertura-auditoria#619be7c1
// Propiedad sobre todos los principios: en un proyecto Rust sin git, ninguno que no pueda leer Rust, ni ninguno cuyo
// detector diga que no puede revisar este repositorio, aparece como evaluado; y el resumen no los cuenta como evaluados.
func TestCobertura_NingunPrincipioSinRevisarApareceAprobado(t *testing.T) {
	dir := proyecto(t, map[string]string{"Cargo.toml": "[package]\nname = \"x\"\n"})
	c := CalcularCobertura(dir, DefaultPolicies(), contexto.Leer(dir), nil)
	evaluados := 0
	for _, cp := range c.Principios {
		id, d := cp.Principio.ID, detectores[cp.Principio.ID]
		if cp.Estado == Evaluado {
			evaluados++
		}
		if cp.Estado == Desactivado {
			continue
		}
		leeRust := false
		for _, s := range d.stacks {
			leeRust = leeRust || s == contexto.Rust
		}
		if cp.Principio.Alcance == AlcanceUnidad && !leeRust && (cp.Estado == Evaluado || len(cp.NoEvaluadas) == 0) {
			t.Errorf("%s no lee Rust y aparece %s, sin unidades no evaluadas: %+v", id, cp.Estado, cp.NoEvaluadas)
		}
		if d.evaluable == nil {
			continue
		}
		if puede, _ := d.evaluable(dir); !puede && (cp.Estado == Evaluado || len(cp.NoEvaluadas) == 0) {
			t.Errorf("%s no puede revisar este repositorio y aparece %s: %+v", id, cp.Estado, cp.NoEvaluadas)
		}
	}
	if cp := coberturaDe(t, c, "P-TRACEABLE-COMMITS"); cp.Estado != NoEvaluado {
		t.Errorf("sin git, P-TRACEABLE-COMMITS = %s; want no_evaluado", cp.Estado)
	}
	if c.Evaluados != evaluados || c.Evaluados+c.Parciales+c.NoEvaluados != c.Aplicables {
		t.Errorf("el resumen cuenta solo lo evaluado: %+v (evaluados de verdad: %d)", c, evaluados)
	}
}

// Calcular la cobertura sobre una auditoría con violaciones deja la lista
// igual, campo por campo, y Evaluar devuelve las mismas que RunAll.
// contrato: cobertura-auditoria#b3ca6f36
func TestCobertura_NoCambiaLasViolacionesNiElPuntaje(t *testing.T) {
	dir := proyecto(t, map[string]string{
		"go.mod":           "module x\n",
		"main.go":          "package main\n\nfunc f(a bool) int {\n\tif a {\n\t\treturn 1\n\t} else {\n\t\treturn 2\n\t}\n}\n",
		".qdd/config.yaml": "test_commands:\n  - \"true\"\n",
	})
	e := NewEngine(dir)
	vs := e.RunAll()
	if len(vs) == 0 {
		t.Fatal("la prueba necesita violaciones")
	}
	copia := append([]Violation{}, vs...)
	puntaje := Score(vs)

	CalcularCobertura(dir, LoadPolicies(dir), contexto.Leer(dir), vs)

	if !reflect.DeepEqual(vs, copia) {
		t.Errorf("CalcularCobertura cambió las violaciones:\nantes %+v\ndespués %+v", copia, vs)
	}
	if Score(vs) != puntaje {
		t.Errorf("CalcularCobertura cambió el puntaje: %d -> %d", puntaje, Score(vs))
	}
	r := e.Evaluar()
	if !reflect.DeepEqual(r.Violaciones, copia) {
		t.Errorf("Evaluar devolvió otras violaciones que RunAll:\nRunAll %+v\nEvaluar %+v", copia, r.Violaciones)
	}
}
