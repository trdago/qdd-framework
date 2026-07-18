package evolution

import (
	"os"
	"path/filepath"
	"testing"
)

func setupProject(t *testing.T) string {
	t.Helper()
	cwd := t.TempDir()
	mustMkdirAll(t, filepath.Join(cwd, ".qdd", "project", "findings"))
	mustMkdirAll(t, filepath.Join(cwd, ".qdd", "project", "certification"))
	mustMkdirAll(t, filepath.Join(cwd, ".qdd", "core", "certification"))
	mustMkdirAll(t, filepath.Join(cwd, ".qdd", "project", "metrics"))
	return cwd
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("failed to create dir %q: %v", dir, err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %q: %v", path, err)
	}
}

func TestAnalyze_NoSignals_RecommendsLowPriority(t *testing.T) {
	cwd := setupProject(t)

	report, err := Analyze(cwd, 0)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if report.Priority != "LOW" {
		t.Errorf("Priority = %q, want LOW when there are no findings/certs/violations", report.Priority)
	}
	if len(report.OpenFindings) != 0 {
		t.Errorf("expected 0 open findings, got %d", len(report.OpenFindings))
	}
}

func TestAnalyze_WorseningTendency_IsTopPriority(t *testing.T) {
	cwd := setupProject(t)
	writeFile(t, filepath.Join(cwd, ".qdd", "project", "metrics", "certificate_history.json"),
		`[{"timestamp":"2026-01-01T00:00:00Z","score":40,"total_violations":10,"tendency":"Empeorando"}]`)

	report, err := Analyze(cwd, 0)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if report.Priority != "CRITICAL" {
		t.Errorf("Priority = %q, want CRITICAL when tendency is Empeorando", report.Priority)
	}
	if report.Tendency != "Empeorando" {
		t.Errorf("Tendency = %q, want Empeorando", report.Tendency)
	}
}

func TestAnalyze_ActiveViolations_OutrankFindingsAndCerts(t *testing.T) {
	cwd := setupProject(t)
	writeFile(t, filepath.Join(cwd, ".qdd", "project", "findings", "FND-001.yaml"),
		"id: FND-001\ntitle: Some open bug\nstatus: OPEN\nimpact: HIGH\n")

	report, err := Analyze(cwd, 3)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if report.Priority != "HIGH" {
		t.Errorf("Priority = %q, want HIGH when there are active audit violations", report.Priority)
	}
	if report.Violations != 3 {
		t.Errorf("Violations = %d, want 3", report.Violations)
	}
}

func TestAnalyze_OpenFindings_SortedByImpactDescending(t *testing.T) {
	cwd := setupProject(t)
	writeFile(t, filepath.Join(cwd, ".qdd", "project", "findings", "FND-001.yaml"),
		"id: FND-001\ntitle: Low impact issue\nstatus: OPEN\nimpact: LOW\n")
	writeFile(t, filepath.Join(cwd, ".qdd", "project", "findings", "FND-002.yaml"),
		"id: FND-002\ntitle: Critical issue\nstatus: OPEN\nimpact: CRITICAL - breaks prod\n")
	writeFile(t, filepath.Join(cwd, ".qdd", "project", "findings", "FND-003.yaml"),
		"id: FND-003\ntitle: Already resolved\nstatus: RESOLVED\nimpact: HIGH\n")

	report, err := Analyze(cwd, 0)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(report.OpenFindings) != 2 {
		t.Fatalf("expected 2 open findings (RESOLVED excluded), got %d", len(report.OpenFindings))
	}
	if report.OpenFindings[0].ID != "FND-002" {
		t.Errorf("expected FND-002 (CRITICAL) to be ranked first, got %s", report.OpenFindings[0].ID)
	}
	if report.Priority != "HIGH" {
		t.Errorf("Priority = %q, want HIGH when there are open findings", report.Priority)
	}
}

func TestAnalyze_PendingCert_RecommendedWhenNoOtherSignal(t *testing.T) {
	cwd := setupProject(t)
	writeFile(t, filepath.Join(cwd, ".qdd", "project", "certification", "CERT-050.yaml"),
		"id: CERT-050\ntitle: New standard being adopted\nstatus: pending\n")

	report, err := Analyze(cwd, 0)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if report.Priority != "MEDIUM" {
		t.Errorf("Priority = %q, want MEDIUM when only a pending cert exists", report.Priority)
	}
	if len(report.PendingCerts) != 1 || report.PendingCerts[0].ID != "CERT-050" {
		t.Errorf("expected CERT-050 to be listed as pending, got %+v", report.PendingCerts)
	}
}

func TestAnalyze_MissingDirectories_DoNotError(t *testing.T) {
	cwd := t.TempDir() // no .qdd structure at all

	report, err := Analyze(cwd, 0)
	if err != nil {
		t.Fatalf("Analyze should tolerate a project with no .qdd yet, got error: %v", err)
	}
	if report.Priority != "LOW" {
		t.Errorf("Priority = %q, want LOW as a safe default", report.Priority)
	}
}
