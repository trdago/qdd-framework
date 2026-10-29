package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/qdd-framework/qdd/pkg/audit"
	"github.com/qdd-framework/qdd/pkg/certificaciones"
	"github.com/qdd-framework/qdd/pkg/dashboard"
	"github.com/qdd-framework/qdd/pkg/evolution"
	"github.com/qdd-framework/qdd/pkg/findings"
)

func proyectoConFindings(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	base := filepath.Join(dir, ".qdd", "project", "findings")
	os.MkdirAll(base, 0755)
	archivos := map[string]string{
		"FND-001.yaml": "title: a\nstatus: open\n",
		"FND-002.yaml": "title: b\nstatus: RESOLVED\n",
		"FND-003.yaml": "title: c\nstatus: \"resolved\"\n",
		"FND-004.yaml": "title: d\naction_taken: se corrigió\n",
	}
	for nombre, contenido := range archivos {
		os.WriteFile(filepath.Join(base, nombre), []byte(contenido), 0644)
	}
	old, _ := os.Getwd()
	os.Chdir(dir)
	t.Cleanup(func() { os.Chdir(old) })
	return dir
}

// FND-056: la misma carpeta de findings daba conteos distintos en el
// dashboard, qdd_score, report y evolution. Ahora todos usan pkg/findings.
func TestIndicadores_MismoConteoDeFindingsEnTodosLosLectores(t *testing.T) {
	dir := proyectoConFindings(t)
	want := findings.Conteo{Abiertos: 1, Resueltos: 2, SinEstado: 1}
	if got := findings.ContarEn(dir); got != want {
		t.Fatalf("pkg/findings = %+v; want %+v", got, want)
	}

	if open, resolved := getFindingsStats(); open != want.Abiertos || resolved != want.Resueltos {
		t.Errorf("qdd_score: abiertos %d, resueltos %d; want %d, %d", open, resolved, want.Abiertos, want.Resueltos)
	}
	if open, resolved := countProjectFindings(dir); open != want.Abiertos || resolved != want.Resueltos {
		t.Errorf("report: abiertos %d, resueltos %d; want %d, %d", open, resolved, want.Abiertos, want.Resueltos)
	}
	if evo, err := evolution.Analyze(dir, evolution.Medicion{}); err != nil || len(evo.OpenFindings) != want.Abiertos {
		t.Errorf("evolution: %d abiertos (%v); want %d", len(evo.OpenFindings), err, want.Abiertos)
	}
	s := dashboard.BuildState()
	if s.FindingsConteo != want {
		t.Errorf("dashboard: %+v; want %+v", s.FindingsConteo, want)
	}
	if s.QualityStatus == "PASS" {
		t.Error("dashboard: hay un finding abierto y QualityStatus dice PASS")
	}
}

// FND-056: un finding sin status no es deuda abierta. El dashboard decía
// "FAIL (Deuda Técnica Detectada)" en un proyecto sin deuda.
func TestIndicadores_SinEstadoNoEsDeudaAbierta(t *testing.T) {
	dir := proyectoConFindings(t)
	os.Remove(filepath.Join(dir, ".qdd", "project", "findings", "FND-001.yaml"))
	if s := dashboard.BuildState(); s.QualityStatus != "PASS" {
		t.Errorf("QualityStatus = %q con 0 findings abiertos; want PASS", s.QualityStatus)
	}
}

// FND-057: qdd_score y report usan la misma escala que el dashboard.
func TestIndicadores_UnaSolaEscalaDeGrado(t *testing.T) {
	for _, score := range []int{100, 92, 85, 72, 55, 30} {
		_, grade, _ := getScoreMetadata(&evolution.Report{Score: score}, nil)
		if grade != audit.Grado(score) {
			t.Errorf("report: grado de %d = %q; want %q", score, grade, audit.Grado(score))
		}
	}
}

// FND-055: sin informe de evolución, report devolvía 100 "A+ (Excelente)"
// aunque hubiera problemas que bloquean. El puntaje sale de la auditoría.
func TestIndicadores_ReportNoInventaPuntaje(t *testing.T) {
	violations := []audit.Violation{{RuleID: "QA-01-ALL-TESTS-PASS"}}
	score, grade, _ := getScoreMetadata(nil, violations)
	if score != audit.Score(violations) || grade != audit.Grado(score) {
		t.Errorf("report sin evolución = %d %q; want %d %q", score, grade, audit.Score(violations), audit.Grado(audit.Score(violations)))
	}
}

// FND-060: report buscaba el texto 'status: certified' y qdd_score comparaba
// con "certified" exacto, así que 'status: "PASSED"' no contaba como cumplida;
// el dashboard además contaba el catálogo. Ahora todos usan pkg/certificaciones.
func TestIndicadores_MismoConteoDeCertificacionesEnTodosLosLectores(t *testing.T) {
	dir := proyectoConFindings(t)
	archivos := map[string]string{
		"core/certification/CERT-CAT.yaml":    "id: CERT-CAT\nstatus: certified\n",
		"project/certification/CERT-A.yaml":   "id: CERT-A\nstatus: certified\n",
		"project/certification/CERT-B.yaml":   "id: CERT-B\nstatus: \"PASSED\"\n",
		"project/certification/CERT-C.yaml":   "id: CERT-C\nstatus: pending\n",
		"project/certification/CERT-SIN.yaml": "id: CERT-SIN\n",
	}
	for ruta, contenido := range archivos {
		path := filepath.Join(dir, ".qdd", ruta)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(contenido), 0644)
	}
	want := certificaciones.Conteo{Cumplen: 2, Pendientes: 1, SinEstado: 1, Disponibles: 1}
	if got := dashboard.BuildState().CertificacionesConteo; got != want {
		t.Errorf("dashboard: %+v; want %+v", got, want)
	}
	if total, cumplen := getCertificationStats(); total != 4 || cumplen != 2 {
		t.Errorf("qdd_score: %d/%d; want 2/4 (solo las del proyecto)", cumplen, total)
	}
	if total, cumplen, pendientes := countProjectCertifications(dir); total != 4 || cumplen != 2 || pendientes != 2 {
		t.Errorf("report: total %d, cumplen %d, pendientes %d; want 4, 2, 2 (sin estado cuenta como pendiente)", total, cumplen, pendientes)
	}
}
