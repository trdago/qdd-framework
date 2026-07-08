package benchmark

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/qdd-framework/qdd/pkg/audit"
	_ "modernc.org/sqlite"
)

// RunBenchmark evalúa el estado actual de la capa cognitiva (QCL) y devuelve los puntajes FAANG-Grade.
func RunBenchmark(cwd string) CognitiveScores {
	return CognitiveScores{
		ContextScore:       evaluateContext(cwd),
		UnderstandingScore: evaluateUnderstanding(cwd),
		ImprovementScore:   evaluateImprovements(cwd),
		ComplianceScore:    evaluateCompliance(cwd),
	}
}

func evaluateContext(cwd string) int {
	score := 0

	configPath := filepath.Join(cwd, ".qdd", "config.yaml")
	if _, err := os.Stat(configPath); err == nil {
		score += 10
	}

	db, err := openKnowledgeDB(cwd)
	if err != nil {
		return score
	}
	defer db.Close()

	totalNodes := getTotalNodes(db)
	if totalNodes == 0 {
		return score
	}

	score += 45

	return score + calculateOrphanScore(db, totalNodes)
}

func openKnowledgeDB(cwd string) (*sql.DB, error) {
	knowledgePath := filepath.Join(cwd, ".qdd", "knowledge.db")
	return sql.Open("sqlite", knowledgePath)
}

func getTotalNodes(db *sql.DB) int {
	var totalNodes int
	if err := db.QueryRowContext(context.Background(), "SELECT count(1) FROM nodes").Scan(&totalNodes); err != nil {
		return 0
	}
	return totalNodes
}

func calculateOrphanScore(db *sql.DB, totalNodes int) int {
	var orphanNodes int
	query := `
		SELECT count(1) FROM nodes 
		WHERE id NOT IN (SELECT source_id FROM edges) 
		AND id NOT IN (SELECT target_id FROM edges)
	`
	if err := db.QueryRowContext(context.Background(), query).Scan(&orphanNodes); err != nil {
		return 0
	}

	orphanRate := (orphanNodes * 100) / totalNodes

	if orphanRate <= 10 {
		return 45
	}

	if orphanRate <= 30 {
		return 20
	}

	return 0
}

func evaluateUnderstanding(cwd string) int {
	score := 0
	understandingPath := filepath.Join(cwd, ".qdd", "understanding.json")

	data, err := os.ReadFile(understandingPath)
	if err != nil {
		return score
	}

	var understanding struct {
		Summary    string   `json:"summary"`
		Components []string `json:"components"`
		Objectives []string `json:"objectives"`
		Guidelines []string `json:"guidelines"`
	}

	if parseErr := json.Unmarshal(data, &understanding); parseErr != nil {
		return score
	}

	score += evaluateUnderstandingBase(understanding.Summary, understanding.Components)
	score += evaluateConstitutionalAlignment(understanding.Objectives)
	score += evaluateGuidelinesAlignment(understanding.Guidelines)

	return score
}

func evaluateUnderstandingBase(summary string, components []string) int {
	score := 0
	if len(summary) > 100 {
		score += 20
	}
	if len(components) >= 3 {
		score += 20
	}
	return score
}

func evaluateConstitutionalAlignment(objectives []string) int {
	for _, obj := range objectives {
		if isConstitutional(strings.ToLower(obj)) {
			return 30
		}
	}
	return 0
}

func isConstitutional(lowerObj string) bool {
	return isSecurityTerm(lowerObj) || isScalabilityTerm(lowerObj) || isPerformanceTerm(lowerObj)
}

func isSecurityTerm(lowerObj string) bool {
	return strings.Contains(lowerObj, "security") || strings.Contains(lowerObj, "seguridad")
}

func isScalabilityTerm(lowerObj string) bool {
	return strings.Contains(lowerObj, "scalability") || strings.Contains(lowerObj, "escalabilidad")
}

func isPerformanceTerm(lowerObj string) bool {
	return strings.Contains(lowerObj, "performance") || strings.Contains(lowerObj, "rendimiento")
}

func evaluateGuidelinesAlignment(guidelines []string) int {
	for _, g := range guidelines {
		if checkGuidelineMatch(strings.ToLower(g)) {
			return 30
		}
	}
	return 0
}

func checkGuidelineMatch(lowerG string) bool {
	return strings.Contains(lowerG, "rule") || strings.Contains(lowerG, "framework") || strings.Contains(lowerG, "qdd")
}

func evaluateImprovements(cwd string) int {
	return evaluateFindings(cwd) + evaluateMetrics(cwd)
}

func evaluateFindings(cwd string) int {
	findingsDir := filepath.Join(cwd, ".qdd", "project", "findings")
	entries, err := os.ReadDir(findingsDir)
	if err != nil {
		return 0
	}

	validFindings := 0
	resolvedFindings := 0

	for _, e := range entries {
		processFindingEntry(findingsDir, e, &validFindings, &resolvedFindings)
	}

	return calculateFindingScore(validFindings, resolvedFindings)
}

func calculateFindingScore(validFindings, resolvedFindings int) int {
	if validFindings == 0 {
		return 0
	}

	score := 20
	resolutionRate := (resolvedFindings * 100) / validFindings

	if resolutionRate >= 50 {
		return score + 30
	}

	if resolutionRate > 0 && resolutionRate < 50 {
		return score + 15
	}

	return score
}

func processFindingEntry(findingsDir string, e os.DirEntry, validFindings, resolvedFindings *int) {
	if !isValidFindingFile(e) {
		return
	}

	data, err := os.ReadFile(filepath.Join(findingsDir, e.Name()))
	if err != nil {
		return
	}

	analyzeFindingContent(string(data), validFindings, resolvedFindings)
}

func isValidFindingFile(e os.DirEntry) bool {
	return !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml")
}

func analyzeFindingContent(content string, validFindings, resolvedFindings *int) {
	if !strings.Contains(content, "severity:") || !strings.Contains(content, "description:") {
		return
	}

	*validFindings++
	lowerContent := strings.ToLower(content)

	if strings.Contains(lowerContent, "status: resolved") || strings.Contains(lowerContent, "status: closed") {
		*resolvedFindings++
	}
}

func evaluateMetrics(cwd string) int {
	metricsDir := filepath.Join(cwd, ".qdd", "project", "metrics")
	metrics, err := os.ReadDir(metricsDir)
	if err != nil {
		return 0
	}

	validMetrics := countValidMetrics(metrics)

	if validMetrics >= 2 {
		return 50
	}
	if validMetrics == 1 {
		return 25
	}

	return 0
}

func countValidMetrics(metrics []os.DirEntry) int {
	valid := 0
	for _, e := range metrics {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			valid++
		}
	}
	return valid
}

func evaluateCompliance(cwd string) int {
	engine := audit.NewEngine(cwd)
	violations := engine.RunAll()

	score := 100 - (len(violations) * 10)

	certDir := filepath.Join(cwd, ".qdd", "core", "certification")
	certRules, err := os.ReadDir(certDir)
	if err != nil {
		score -= 50
	}

	validRules := countValidCertRules(certRules)

	if validRules < 5 {
		score -= 50
	}

	if score < 0 {
		return 0
	}
	return score
}

func countValidCertRules(certRules []os.DirEntry) int {
	validRules := 0
	for _, f := range certRules {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".yaml") {
			validRules++
		}
	}
	return validRules
}
