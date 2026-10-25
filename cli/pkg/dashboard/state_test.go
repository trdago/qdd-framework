package dashboard

import (
	"os"
	"path/filepath"
	"testing"
)

func escribirHistorial(t *testing.T, dir, contenido string) {
	t.Helper()
	path := filepath.Join(dir, ".qdd", "project", "metrics", "certificate_history.json")
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(contenido), 0644); err != nil {
		t.Fatal(err)
	}
}

// FND-051 (reemplaza la decisión de FND-021): el dashboard mostraba 100
// "World-Class" con pruebas que fallaban, porque su puntaje solo contaba
// findings abiertos. Ahora muestra el puntaje único de la última certificación.
func TestPuntajeDelDashboard(t *testing.T) {
	dir := t.TempDir()

	score, grade, medido := puntajeDelDashboard(dir)
	if medido || score != 0 || grade != gradoSinMedir {
		t.Errorf("sin certificar: %d %q %v; want 0, %q, false (nunca un 100 inventado)", score, grade, medido, gradoSinMedir)
	}

	escribirHistorial(t, dir, `[{"score":100,"tendency":"Estable"}]`)
	if _, _, medido := puntajeDelDashboard(dir); medido {
		t.Error("un puntaje de la escala anterior no se muestra como actual")
	}

	escribirHistorial(t, dir, `[{"score":84,"escala":2,"tendency":"Estable"}]`)
	score, grade, medido = puntajeDelDashboard(dir)
	if !medido || score != 84 || grade != "A" {
		t.Errorf("certificado: %d %q %v; want 84, A, true", score, grade, medido)
	}
}

// FND-051: sin historial el gráfico mostraba una evolución inventada (40 → 75 → 100).
func TestLoadHistoricalTrends_NoInventaDatos(t *testing.T) {
	var s QDDState
	loadHistoricalTrends(&s, t.TempDir())
	if len(s.Historical) != 0 {
		t.Errorf("historial inventado: %v", s.Historical)
	}
}

func TestDetermineDashboardGrade(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{100, "World-Class"},
		{90, "World-Class"},
		{85, "A"},
		{75, "B"},
		{69, "C"},
		{50, "C"},
		{49, "D (CRITICAL)"},
		{0, "D (CRITICAL)"},
	}

	for _, c := range cases {
		got := determineDashboardGrade(c.score)
		if got != c.want {
			t.Errorf("determineDashboardGrade(%d) = %q, want %q", c.score, got, c.want)
		}
	}
}

func TestDetermineQualityStatus(t *testing.T) {
	if got := determineQualityStatus(0); got != "PASS" {
		t.Errorf("determineQualityStatus(0) = %q, want PASS", got)
	}
	if got := determineQualityStatus(1); got != "FAIL (Deuda Técnica Detectada)" {
		t.Errorf("determineQualityStatus(1) = %q, want FAIL", got)
	}
}

func TestContainsAny(t *testing.T) {
	if !containsAny("hay un timeout aquí", []string{"timeout", "flaky"}) {
		t.Error("expected a match on 'timeout'")
	}
	if containsAny("todo tranquilo", []string{"timeout", "flaky"}) {
		t.Error("expected no match")
	}
}

func TestCheckPillars(t *testing.T) {
	cases := []struct {
		name string
		text string
		want string
	}{
		{"certification keyword", "falta el CERT-030", "Certificación"},
		{"stability keyword", "test flaky detectado", "Estabilidad"},
		{"security keyword", "sql injection posible", "Seguridad"},
		{"structural keyword", "alta complejidad ciclomática", "Estructural"},
		{"unknown keyword defaults to certification", "algo random sin match", "Certificación"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := checkPillars(c.text)
			if got != c.want {
				t.Errorf("checkPillars(%q) = %q, want %q", c.text, got, c.want)
			}
		})
	}
}

func TestDetermineFindingPillar(t *testing.T) {
	got := determineFindingPillar("Timeout en API", nil)
	if got != "Estabilidad" {
		t.Errorf("determineFindingPillar with nil rawData = %q, want Estabilidad", got)
	}

	got = determineFindingPillar("Bug genérico", map[string]interface{}{"description": "Uso de SQL sin sanitizar"})
	if got != "Seguridad" {
		t.Errorf("determineFindingPillar with description = %q, want Seguridad", got)
	}
}

func TestGetLowerDescription(t *testing.T) {
	if got := getLowerDescription(nil); got != "" {
		t.Errorf("getLowerDescription(nil) = %q, want empty string", got)
	}
	if got := getLowerDescription(map[string]interface{}{"description": "HOLA"}); got != "hola" {
		t.Errorf("getLowerDescription = %q, want lowercased 'hola'", got)
	}
}

func TestExtractStatusFromContent(t *testing.T) {
	// FND: determineFinding behavior found by testing, not assumed — a
	// checklist with unchecked items (but nothing checked yet) counts as
	// IN-PROGRESS, not BACKLOG. Only the total absence of any checklist
	// marker (no "- [x]" and no "- [ ]") yields BACKLOG.
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"all checked", "- [x] paso uno\n- [x] paso dos", "COMPLETED"},
		{"only unchecked items counts as in-progress", "- [ ] paso uno\n- [ ] paso dos", "IN-PROGRESS"},
		{"no checklist at all", "solo texto libre", "BACKLOG"},
		{"mixed", "- [x] paso uno\n- [ ] paso dos", "IN-PROGRESS"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractStatusFromContent(c.content)
			if got != c.want {
				t.Errorf("extractStatusFromContent(%q) = %q, want %q", c.content, got, c.want)
			}
		})
	}
}

func TestDetermineSprintStatus(t *testing.T) {
	got := determineSprintStatus("- [x] a\n- [ ] b", map[string]interface{}{"status": "CUSTOM"})
	if got != "CUSTOM" {
		t.Errorf("expected explicit rawData status to win, got %q", got)
	}

	got = determineSprintStatus("- [x] a\n- [x] b", nil)
	if got != "COMPLETED" {
		t.Errorf("expected fallback to content-derived status, got %q", got)
	}
}
