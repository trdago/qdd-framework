package benchmark

import (
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
	
	knowledgePath := filepath.Join(cwd, ".qdd", "knowledge.db")
	db, err := sql.Open("sqlite", knowledgePath)
	if err != nil {
		return score
	}
	defer db.Close()

	var totalNodes int
	if err := db.QueryRow("SELECT COUNT(*) FROM nodes").Scan(&totalNodes); err != nil || totalNodes == 0 {
		return score
	}
	
	score += 45 // Puntuación base por tener nodos poblados

	var orphanNodes int
	query := `
		SELECT COUNT(*) FROM nodes 
		WHERE id NOT IN (SELECT source_id FROM edges) 
		AND id NOT IN (SELECT target_id FROM edges)
	`
	if err := db.QueryRow(query).Scan(&orphanNodes); err == nil {
		// Calcular porcentaje de orfandad (Google-Grade Graph Density)
		orphanRate := (orphanNodes * 100) / totalNodes
		
		if orphanRate <= 10 {
			score += 45 // Perfecto, menos del 10% de orfandad
			return score
		}
		
		if orphanRate <= 30 {
			score += 20 // Regular
			return score
		}
	}

	return score
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

	if len(understanding.Summary) > 100 {
		score += 20
	}
	if len(understanding.Components) >= 3 {
		score += 20
	}

	// Anthropic-Grade: Constitutional Alignment
	hasConstitutionalAlignment := false
	for _, obj := range understanding.Objectives {
		lowerObj := strings.ToLower(obj)
		if strings.Contains(lowerObj, "security") || strings.Contains(lowerObj, "seguridad") ||
			strings.Contains(lowerObj, "scalability") || strings.Contains(lowerObj, "escalabilidad") ||
			strings.Contains(lowerObj, "performance") || strings.Contains(lowerObj, "rendimiento") {
			hasConstitutionalAlignment = true
			break
		}
	}
	
	if hasConstitutionalAlignment {
		score += 30
	}

	if len(understanding.Guidelines) > 0 {
		for _, g := range understanding.Guidelines {
			lowerG := strings.ToLower(g)
			if strings.Contains(lowerG, "rule") || strings.Contains(lowerG, "framework") || strings.Contains(lowerG, "qdd") {
				score += 30
				break
			}
		}
	}

	return score
}

func evaluateImprovements(cwd string) int {
	score := 0
	
	findingsDir := filepath.Join(cwd, ".qdd", "project", "findings")
	entries, err := os.ReadDir(findingsDir)
	
	validFindings := 0
	resolvedFindings := 0
	
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(findingsDir, e.Name()))
			if err != nil {
				continue
			}
			
			content := string(data)
			if strings.Contains(content, "severity:") && strings.Contains(content, "description:") {
				validFindings++
				
				// Meta-Grade: Incident Resolution check
				if strings.Contains(strings.ToLower(content), "status: resolved") || strings.Contains(strings.ToLower(content), "status: closed") {
					resolvedFindings++
				}
			}
		}
	}

	// Meta-Grade Resolution Rate
	if validFindings > 0 {
		score += 20 // Por tener hallazgos válidos
		
		resolutionRate := (resolvedFindings * 100) / validFindings
		if resolutionRate >= 50 {
			score += 30 // Puntuación completa por alta tasa de resolución
		}
		if resolutionRate > 0 && resolutionRate < 50 {
			score += 15
		}
	}

	metricsDir := filepath.Join(cwd, ".qdd", "project", "metrics")
	metrics, err := os.ReadDir(metricsDir)
	if err == nil {
		validMetrics := 0
		for _, e := range metrics {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
				validMetrics++
			}
		}
		
		if validMetrics >= 2 {
			score += 50
			return score
		}
		if validMetrics == 1 {
			score += 25
			return score
		}
	}

	return score
}

func evaluateCompliance(cwd string) int {
	engine := audit.NewEngine(cwd)
	violations := engine.RunAll()

	score := 100 - (len(violations) * 10) // Penalización más estricta por violación

	// Exigencia estricta de certificaciones (reglas as-code) activas
	certDir := filepath.Join(cwd, ".qdd", "core", "certification")
	certRules, err := os.ReadDir(certDir)
	if err != nil {
		score -= 50
	}
	
	validRules := 0
	for _, f := range certRules {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".yaml") {
			validRules++
		}
	}

	if validRules < 5 {
		score -= 50 // Severa penalización por no tener al menos 5 reglas (Beyond Limits)
	}

	if score < 0 {
		return 0
	}
	return score
}
