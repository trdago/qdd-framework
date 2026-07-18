package dashboard

import "testing"

// FND-021: the dashboard's score/audit_status are computed purely from open
// findings, unrelated to `qdd audit`'s static violations — these tests pin
// down that exact (currently undocumented-in-tests) behavior so a future
// change can't silently alter the formula without a test noticing.
func TestComputeFinalScore(t *testing.T) {
	cases := []struct {
		name         string
		openFindings int
		want         int
	}{
		{"no open findings scores perfectly", 0, 100},
		{"one open finding costs 30 points", 1, 70},
		{"three open findings costs 90 points", 3, 10},
		{"score never goes below zero", 10, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := computeFinalScore(c.openFindings)
			if got != c.want {
				t.Errorf("computeFinalScore(%d) = %d, want %d", c.openFindings, got, c.want)
			}
		})
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

func TestDetermineAuditStatus(t *testing.T) {
	if got := determineAuditStatus(0); got != "PASS" {
		t.Errorf("determineAuditStatus(0) = %q, want PASS", got)
	}
	if got := determineAuditStatus(1); got != "FAIL (Deuda Técnica Detectada)" {
		t.Errorf("determineAuditStatus(1) = %q, want FAIL", got)
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
