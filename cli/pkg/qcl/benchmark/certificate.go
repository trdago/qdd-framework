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
	if err := os.MkdirAll(filepath.Dir(historyPath), 0755); err != nil {
		return nil, fmt.Errorf("error creando directorio de metricas: %v", err)
	}

	history, err := loadCognitiveHistory(historyPath)
	if err != nil {
		return nil, err
	}

	cert.Tendency = calculateTendency(history, cert.Total)

	history = append(history, *cert)
	
	if err := saveCognitiveHistory(historyPath, history); err != nil {
		return nil, err
	}

	return cert, nil
}

func loadCognitiveHistory(historyPath string) ([]CognitiveCertificate, error) {
	var history []CognitiveCertificate
	
	data, err := os.ReadFile(historyPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error leyendo historial cognitivo: %v", err)
		}
		return history, nil
	}

	if parseErr := json.Unmarshal(data, &history); parseErr != nil {
		return nil, fmt.Errorf("error parseando historial cognitivo: %v", parseErr)
	}

	return history, nil
}

func calculateTendency(history []CognitiveCertificate, currentTotal int) audit.Tendency {
	if len(history) == 0 {
		return audit.TendencyStable
	}
	
	lastCert := history[len(history)-1]
	if currentTotal > lastCert.Total {
		return audit.TendencyImproving
	}
	if currentTotal < lastCert.Total {
		return audit.TendencyWorsening
	}
	
	return audit.TendencyStable
}

func saveCognitiveHistory(historyPath string, history []CognitiveCertificate) error {
	outData, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling historial cognitivo: %v", err)
	}

	if err := os.WriteFile(historyPath, outData, 0644); err != nil {
		return fmt.Errorf("error escribiendo historial cognitivo: %v", err)
	}
	
	return nil
}
