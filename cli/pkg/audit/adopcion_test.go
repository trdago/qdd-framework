package audit

import (
	"os"
	"path/filepath"
	"testing"
)

// FND-040: un proyecto nuevo y sano fallaba su primera auditoría por reglas de
// madurez (E2E, ADR, Conventional Commits). En adopción gradual son
// recomendaciones: se informan, bajan poco el puntaje y no bloquean.
func TestMarcarRecomendaciones_Gradual(t *testing.T) {
	v := []Violation{
		{RuleID: "QA-02-E2E-TESTS"},
		{RuleID: "TRC-01-CONVENTIONAL-COMMITS"},
		{RuleID: "CERT-030-MANDATORY-ADR"},
		{RuleID: "12F-01-CODEBASE"},
		{RuleID: "QA-01-ALL-TESTS-PASS"},
		{RuleID: "QA-00-NO-TEST-COMMAND"},
		{RuleID: "OWASP-01"},
	}
	marcarRecomendaciones(QDDPolicies{Adopcion: AdopcionGradual}, v)

	for _, x := range v[:4] {
		if !x.Recomendacion {
			t.Errorf("%s debería ser recomendación en adopción gradual", x.RuleID)
		}
	}
	for _, x := range v[4:] {
		if x.Recomendacion {
			t.Errorf("%s es un problema real: nunca debe dejar de bloquear", x.RuleID)
		}
	}
	if got := len(Bloqueantes(v)); got != 3 {
		t.Errorf("Bloqueantes = %d, want 3", got)
	}
}

// Los proyectos existentes no declaran adopción: siguen en modo estricto.
func TestMarcarRecomendaciones_EstrictoPorDefecto(t *testing.T) {
	v := []Violation{{RuleID: "QA-02-E2E-TESTS"}}
	marcarRecomendaciones(DefaultPolicies(), v)
	if v[0].Recomendacion {
		t.Error("sin 'adopcion: gradual' todas las reglas bloquean")
	}
	if len(Bloqueantes(v)) != 1 {
		t.Error("la violación debe bloquear en modo estricto")
	}
}

func TestLoadPolicies_LeeAdopcion(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, ".qdd"), 0755)
	os.WriteFile(filepath.Join(dir, ".qdd", "policies.yaml"), []byte("owasp: true\nadopcion: gradual\n"), 0644)
	if p := LoadPolicies(dir); p.Adopcion != AdopcionGradual {
		t.Errorf("Adopcion = %q, want gradual", p.Adopcion)
	}
}

// FND-041: cada comando calculaba un puntaje distinto (0, 94 o 100) para el
// mismo proyecto. Score es la única fórmula.
func TestScore(t *testing.T) {
	casos := []struct {
		nombre string
		v      []Violation
		want   int
	}{
		{"limpio", nil, 100},
		{"solo recomendaciones", []Violation{{Recomendacion: true}, {Recomendacion: true}, {Recomendacion: true}}, 94},
		{"una bloqueante", []Violation{{}}, 90},
		{"mixto", []Violation{{}, {Recomendacion: true}}, 88},
		{"piso en cero", make([]Violation, 20), 0},
	}
	for _, c := range casos {
		if got := Score(c.v); got != c.want {
			t.Errorf("%s: Score = %d, want %d", c.nombre, got, c.want)
		}
	}
}

// FND-057: el dashboard y el MCP usaban escalas distintas para el mismo
// puntaje (un 92 era World-Class en uno y B en el otro). Grado es la única.
func TestGrado(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{100, "World-Class"}, {90, "World-Class"}, {89, "A"}, {80, "A"},
		{79, "B"}, {70, "B"}, {69, "C"}, {50, "C"}, {49, "D (CRITICAL)"}, {0, "D (CRITICAL)"},
	}
	for _, c := range cases {
		if got := Grado(c.score); got != c.want {
			t.Errorf("Grado(%d) = %q; want %q", c.score, got, c.want)
		}
	}
}
