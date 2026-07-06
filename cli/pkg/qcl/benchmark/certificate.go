package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/qdd-framework/qdd/pkg/audit"
)

// CognitiveScores representa las 4 dimensiones de calidad del framework cognitivo (QCL)
type CognitiveScores struct {
	ContextScore      int `json:"context_score"`
	UnderstandingScore int `json:"understanding_score"`
	ImprovementScore   int `json:"improvement_score"`
	ComplianceScore    int `json:"compliance_score"`
}

// CognitiveCertificate representa el certificado generado tras un benchmark QCL
type CognitiveCertificate struct {
	Timestamp string          `json:"timestamp"`
	Scores    CognitiveScores `json:"scores"`
	Total     int             `json:"total"`
	Tendency  audit.Tendency  `json:"tendency"`
}

// GenerateCognitiveCertificate toma los puntajes brutos, calcula totales,
// actualiza el historial y determina la tendencia (Mejorando, Empeorando, Estable).
func GenerateCognitiveCertificate(cwd string, scores CognitiveScores) (*CognitiveCertificate, error) {
	total := (scores.ContextScore + scores.UnderstandingScore + scores.ImprovementScore + scores.ComplianceScore) / 4

	cert := &CognitiveCertificate{
		Timestamp: time.Now().Format(time.RFC3339),
		Scores:    scores,
		Total:     total,
		Tendency:  audit.TendencyStable,
	}

	historyPath := filepath.Join(cwd, ".qdd", "project", "metrics", "cognitive_history.json")
	
	dir := filepath.Dir(historyPath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("error creando directorio de metricas: %v", err)
	}

	var history []CognitiveCertificate
	
	data, err := os.ReadFile(historyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error leyendo historial cognitivo: %v", err)
		}
	}

	// Si no hay error (err == nil) significa que el archivo existía y pudimos leerlo
	if err == nil {
		parseErr := json.Unmarshal(data, &history)
		if parseErr != nil {
			return nil, fmt.Errorf("error parseando historial cognitivo: %v", parseErr)
		}
	}

	if len(history) > 0 {
		lastCert := history[len(history)-1]
		if cert.Total > lastCert.Total {
			cert.Tendency = audit.TendencyImproving
		}
		if cert.Total < lastCert.Total {
			cert.Tendency = audit.TendencyWorsening
		}
	}

	history = append(history, *cert)
	
	outData, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error marshaling historial cognitivo: %v", err)
	}

	err = os.WriteFile(historyPath, outData, 0644)
	if err != nil {
		return nil, fmt.Errorf("error escribiendo historial cognitivo: %v", err)
	}

	return cert, nil
}
