package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Tendency enum for the trend
type Tendency string

const (
	TendencyImproving Tendency = "Mejorando"
	TendencyWorsening Tendency = "Empeorando"
	TendencyStable    Tendency = "Estable"
)

// Certificate represents a snapshot of the framework's quality
type Certificate struct {
	Timestamp      string     `json:"timestamp"`
	Score          int        `json:"score"`
	TotalViolations int        `json:"total_violations"`
	Tendency       Tendency   `json:"tendency"`
}

// GenerateCertificate creates a certificate based on violations and saves it to history.
func GenerateCertificate(cwd string, violations []Violation) (*Certificate, error) {
	score := calculateScore(len(violations))
	cert := createInitialCert(score, len(violations))

	historyPath := filepath.Join(cwd, ".qdd", "project", "metrics", "certificate_history.json")
	if err := ensureDirExists(historyPath); err != nil {
		return nil, err
	}

	history, err := readCertHistory(historyPath)
	if err != nil {
		return nil, err
	}

	updateCertTendency(cert, history)
	history = append(history, *cert)
	
	if err := saveCertHistory(historyPath, history); err != nil {
		return nil, err
	}

	return cert, nil
}

func calculateScore(violationsCount int) int {
	score := 100 - (violationsCount * 2)
	if score < 0 {
		return 0
	}
	return score
}

func createInitialCert(score, violationsCount int) *Certificate {
	return &Certificate{
		Timestamp:      time.Now().Format(time.RFC3339),
		Score:          score,
		TotalViolations: violationsCount,
		Tendency:       TendencyStable,
	}
}

func ensureDirExists(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("error creating metrics directory: %v", err)
	}
	return nil
}

func readCertHistory(path string) ([]Certificate, error) {
	var history []Certificate
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error reading certificate history: %v", err)
		}
		return history, nil
	}
	
	if parseErr := json.Unmarshal(data, &history); parseErr != nil {
		return nil, fmt.Errorf("error parsing certificate history: %v", parseErr)
	}
	return history, nil
}

func updateCertTendency(cert *Certificate, history []Certificate) {
	if len(history) == 0 {
		return
	}
	
	lastCert := history[len(history)-1]
	if cert.Score > lastCert.Score {
		cert.Tendency = TendencyImproving
	}
	if cert.Score < lastCert.Score {
		cert.Tendency = TendencyWorsening
	}
}

func saveCertHistory(path string, history []Certificate) error {
	outData, err := json.MarshalIndent(history, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling certificate history: %v", err)
	}

	if err := os.WriteFile(path, outData, 0644); err != nil {
		return fmt.Errorf("error writing certificate history: %v", err)
	}
	return nil
}
