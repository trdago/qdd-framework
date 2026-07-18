// Package evolution implements `qdd evolution`: a read-only, deterministic
// analysis that studies the project's own accumulated QDD knowledge
// (findings, certifications, audit violations, score history) to recommend
// what the single next improvement should be. It never creates or modifies
// certifications itself — per the framework's Modo Consultivo principle,
// adopting a new standard always requires explicit human authorization.
package evolution

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Finding mirrors the real shape written to .qdd/project/findings/*.yaml.
// Note: some historical findings use "impact" (free text like "HIGH - ...")
// rather than a dedicated "severity" enum, so both are read tolerantly.
type Finding struct {
	ID       string `yaml:"id"`
	Title    string `yaml:"title"`
	Status   string `yaml:"status"`
	Impact   string `yaml:"impact"`
	Severity string `yaml:"severity"`
}

func (f Finding) isOpen() bool {
	s := strings.ToUpper(strings.TrimSpace(f.Status))
	return s != "RESOLVED" && s != "CLOSED" && s != ""
}

func (f Finding) priorityWeight() int {
	text := strings.ToUpper(f.Impact + " " + f.Severity)
	if strings.Contains(text, "CRITICAL") {
		return 3
	}
	if strings.Contains(text, "HIGH") {
		return 2
	}
	if strings.Contains(text, "MEDIUM") {
		return 1
	}
	return 0
}

// ProjectCert mirrors .qdd/project/certification/*.yaml.
type ProjectCert struct {
	ID     string `yaml:"id"`
	Title  string `yaml:"title"`
	Status string `yaml:"status"`
}

func (c ProjectCert) isPending() bool {
	return strings.EqualFold(strings.TrimSpace(c.Status), "pending")
}

// CoreCert mirrors .qdd/core/certification/*.yaml — the certifications
// available for adoption, shipped to every QDD-governed project.
type CoreCert struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type certHistoryEntry struct {
	Timestamp       string `json:"timestamp"`
	Score           int    `json:"score"`
	TotalViolations int    `json:"total_violations"`
	Tendency        string `json:"tendency"`
}

// Report is the outcome of studying the project's accumulated QDD knowledge.
type Report struct {
	Score            int
	Tendency         string
	Violations       int
	OpenFindings     []Finding
	PendingCerts     []ProjectCert
	AvailableCoreCerts []CoreCert
	Priority         string
	Recommendation   string
}

// Analyze studies findings, certifications, audit violations and score
// history under cwd and produces a single ranked recommendation for the
// next improvement. violationsCount comes from the caller (audit.RunAll),
// keeping this package independent of the audit engine's internals.
func Analyze(cwd string, violationsCount int) (*Report, error) {
	findings, err := loadFindings(filepath.Join(cwd, ".qdd", "project", "findings"))
	if err != nil {
		return nil, err
	}

	projectCerts, err := loadProjectCerts(filepath.Join(cwd, ".qdd", "project", "certification"))
	if err != nil {
		return nil, err
	}

	coreCerts, err := loadCoreCerts(filepath.Join(cwd, ".qdd", "core", "certification"))
	if err != nil {
		return nil, err
	}

	score, tendency := latestScoreAndTendency(filepath.Join(cwd, ".qdd", "project", "metrics", "certificate_history.json"))

	report := &Report{
		Score:              score,
		Tendency:           tendency,
		Violations:         violationsCount,
		OpenFindings:       openFindingsSortedByPriority(findings),
		PendingCerts:       pendingCerts(projectCerts),
		AvailableCoreCerts: coreCerts,
	}
	report.Priority, report.Recommendation = recommend(report)
	return report, nil
}

func recommend(r *Report) (priority, recommendation string) {
	if r.Tendency == "Empeorando" {
		return "CRITICAL", "La tendencia de calidad está empeorando (certificate_history.json). Antes de sumar cualquier función nueva, investiga qué regresó y por qué."
	}
	if r.Violations > 0 {
		return "HIGH", "Hay violaciones activas de `qdd audit`. Repáralas antes de certificar o de adoptar un nuevo estándar — un nuevo certificado sobre una base inestable no aporta valor real."
	}
	if len(r.OpenFindings) > 0 {
		top := r.OpenFindings[0]
		return "HIGH", "Resuelve primero el finding abierto de mayor impacto: " + top.ID + " - " + top.Title
	}
	if len(r.PendingCerts) > 0 {
		top := r.PendingCerts[0]
		return "MEDIUM", "Hay una certificación de proyecto pendiente de validar: " + top.ID + " - " + top.Title + ". Ciérrala antes de abrir un nuevo estándar."
	}
	return "LOW", "No se detectaron brechas, findings abiertos ni violaciones. El proyecto está estable: en Modo Consultivo, evalúa si conviene adoptar un estándar de la industria aún no cubierto por las certificaciones actuales del proyecto."
}

// loadYAMLEntries reads every *.yaml file in dir and unmarshals each into a T,
// skipping unreadable or invalid entries. A missing dir yields an empty,
// error-free result since a brand-new project may not have created it yet.
func loadYAMLEntries[T any](dir string) ([]T, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var items []T
	for _, entry := range entries {
		if item, ok := parseYAMLEntry[T](dir, entry); ok {
			items = append(items, item)
		}
	}
	return items, nil
}

func parseYAMLEntry[T any](dir string, entry os.DirEntry) (T, bool) {
	var zero T
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
		return zero, false
	}
	content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
	if err != nil {
		return zero, false
	}
	var item T
	if err := yaml.Unmarshal(content, &item); err != nil {
		return zero, false
	}
	return item, true
}

func loadFindings(dir string) ([]Finding, error) {
	return loadYAMLEntries[Finding](dir)
}

func loadProjectCerts(dir string) ([]ProjectCert, error) {
	return loadYAMLEntries[ProjectCert](dir)
}

func loadCoreCerts(dir string) ([]CoreCert, error) {
	certs, err := loadYAMLEntries[CoreCert](dir)
	if err != nil {
		return nil, err
	}
	return filterCoreCertsWithID(certs), nil
}

func filterCoreCertsWithID(certs []CoreCert) []CoreCert {
	var withID []CoreCert
	for _, c := range certs {
		if c.ID != "" {
			withID = append(withID, c)
		}
	}
	return withID
}

func openFindingsSortedByPriority(findings []Finding) []Finding {
	var open []Finding
	for _, f := range findings {
		if f.isOpen() {
			open = append(open, f)
		}
	}
	sort.SliceStable(open, func(i, j int) bool {
		return open[i].priorityWeight() > open[j].priorityWeight()
	})
	return open
}

func pendingCerts(certs []ProjectCert) []ProjectCert {
	var pending []ProjectCert
	for _, c := range certs {
		if c.isPending() {
			pending = append(pending, c)
		}
	}
	return pending
}

func latestScoreAndTendency(historyPath string) (score int, tendency string) {
	data, err := os.ReadFile(historyPath)
	if err != nil {
		return 0, "Sin historial"
	}

	var history []certHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil || len(history) == 0 {
		return 0, "Sin historial"
	}

	last := history[len(history)-1]
	return last.Score, last.Tendency
}
